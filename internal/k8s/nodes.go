package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps the Kubernetes client and configuration
type Client struct {
	Clientset *kubernetes.Clientset
}

// NewClient creates a new Kubernetes client
func NewClient() (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := clientcmd.RecommendedHomeFile
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("error building kubeconfig: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating Kubernetes client: %v", err)
	}

	return &Client{Clientset: clientset}, nil
}

// GetNodes returns the list of node names
func (c *Client) GetNodes() ([]string, error) {
	nodeList, err := c.Clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	nodes := make([]string, 0, len(nodeList.Items))
	for _, node := range nodeList.Items {
		nodes = append(nodes, node.Name)
	}
	return nodes, nil
}

// GetSummary fetches and decodes the stats summary from a node
func (c *Client) GetSummary(node string) (*Summary, error) {
	path := fmt.Sprintf("/api/v1/nodes/%s/proxy/stats/summary", node)
	res := c.Clientset.RESTClient().Get().AbsPath(path).Do(context.TODO())
	raw, err := res.Raw()
	if err != nil {
		return nil, err
	}
	var s Summary
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// FindPodUsingPVC finds a pod that uses the specified PVC or any pod in the namespace if no direct match
func (c *Client) FindPodUsingPVC(namespace, pvcName string) (string, error) {
	// Get all pods in the namespace
	pods, err := c.Clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("error listing pods: %v", err)
	}

	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no pods found in namespace '%s'", namespace)
	}

	fmt.Printf("Searching for pod using PVC '%s' in namespace '%s'...\n", pvcName, namespace)

	// First look for a pod that directly uses the PVC
	for _, pod := range pods.Items {
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == pvcName {
				fmt.Printf("Found pod '%s' directly using the PVC\n", pod.Name)
				return pod.Name, nil
			}
		}
	}

	fmt.Println("No pod found directly using the PVC. Trying to infer pod based on naming patterns...")

	// Try to infer the pod from the PVC name for StatefulSets/Strimzi/Kafka

	// Pattern 1: data-0-kafka-kafka-pool-0
	// This looks like Strimzi Kafka PVC pattern
	if strings.HasPrefix(pvcName, "data-") {
		parts := strings.Split(pvcName, "-")

		// Try different patterns to match pods
		var patterns []string

		// If format is data-X-cluster-something
		if len(parts) >= 4 {
			ordinal := parts[1] // e.g., "0", "1", etc.
			cluster := parts[2] // e.g., "kafkasomething"

			// Common Strimzi Kafka pod naming patterns
			patterns = append(patterns, []string{
				fmt.Sprintf("%s-%s", cluster, ordinal),
				fmt.Sprintf("%s-%s", parts[2], ordinal),
				fmt.Sprintf("%s-%s-%s", parts[2], parts[3], ordinal),
				fmt.Sprintf("%s-kafka-%s", cluster, ordinal),
				fmt.Sprintf("%s-zookeeper-%s", cluster, ordinal),
			}...)
		}

		fmt.Printf("Looking for pods with patterns: %v\n", patterns)

		// Check if any pod matches our patterns
		for _, pattern := range patterns {
			for _, pod := range pods.Items {
				if strings.HasPrefix(pod.Name, pattern) {
					fmt.Printf("Found pod '%s' matching pattern '%s'\n", pod.Name, pattern)
					return pod.Name, nil
				}
			}
		}
	}

	// Look for pods that contain parts of the PVC name
	for _, pod := range pods.Items {
		// Split PVC name and pod name by '-'
		pvcParts := strings.Split(pvcName, "-")
		podParts := strings.Split(pod.Name, "-")

		// If any significant part matches, use that pod
		for _, pvcPart := range pvcParts {
			if len(pvcPart) > 3 { // Only consider significant parts (longer than 3 chars)
				for _, podPart := range podParts {
					if pvcPart == podPart && len(podPart) > 3 {
						fmt.Printf("Found pod '%s' sharing common part '%s' with PVC\n", pod.Name, pvcPart)
						return pod.Name, nil
					}
				}
			}
		}
	}

	// If all else fails, just use any running pod (preferring pods with "kafka" in the name)
	for _, pod := range pods.Items {
		if strings.Contains(strings.ToLower(pod.Name), "kafka") &&
			pod.Status.Phase == corev1.PodRunning {
			fmt.Printf("Using pod '%s' as it contains 'kafka' in the name\n", pod.Name)
			return pod.Name, nil
		}
	}

	// Last resort: use any running pod
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			fmt.Printf("Using pod '%s' as fallback (first running pod found)\n", pod.Name)
			return pod.Name, nil
		}
	}

	// Absolute last resort: use any pod
	fmt.Printf("Using pod '%s' as last resort\n", pods.Items[0].Name)
	return pods.Items[0].Name, nil
}

// int64Ptr returns a pointer to an int64
func int64Ptr(i int64) *int64 {
	return &i
}

// boolPtr returns a pointer to a bool
func boolPtr(b bool) *bool {
	return &b
}

// CreatePerformancePod creates a sidecar pod to monitor performance of a PVC
func (c *Client) CreatePerformancePod(namespace, targetPod, pvcName string) (string, error) {
	// Get target pod details
	pod, err := c.Clientset.CoreV1().Pods(namespace).Get(context.TODO(), targetPod, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error getting target pod: %v", err)
	}

	// Create a name for the performance pod - truncate if longer than 50 chars to leave room for prefix
	safePVCName := pvcName
	if len(pvcName) > 50 {
		safePVCName = pvcName[:50]
	}
	perfPodName := fmt.Sprintf("pvc-perf-monitor-%s", safePVCName)

	// Ensure pod name is not longer than 63 characters (Kubernetes limit)
	if len(perfPodName) > 63 {
		perfPodName = perfPodName[:63]
		// Ensure it doesn't end with a dash
		for strings.HasSuffix(perfPodName, "-") {
			perfPodName = perfPodName[:len(perfPodName)-1]
		}
	}

	// Check if performance pod already exists and delete it if it does
	_, err = c.Clientset.CoreV1().Pods(namespace).Get(context.TODO(), perfPodName, metav1.GetOptions{})
	if err == nil {
		// Pod exists, delete it
		err = c.Clientset.CoreV1().Pods(namespace).Delete(context.TODO(), perfPodName, metav1.DeleteOptions{})
		if err != nil {
			return "", fmt.Errorf("error deleting existing performance pod: %v", err)
		}

		// Wait for pod to be deleted
		fmt.Printf("Waiting for previous performance pod to be deleted...")
		for i := 0; i < 30; i++ {
			_, err = c.Clientset.CoreV1().Pods(namespace).Get(context.TODO(), perfPodName, metav1.GetOptions{})
			if err != nil {
				break
			}
			time.Sleep(1 * time.Second)
			fmt.Printf(".")
		}
		fmt.Println()
	}

	// Create safe label values (limited to 63 chars)
	safeTargetPod := targetPod
	if len(targetPod) > 63 {
		safeTargetPod = targetPod[:63]
		// Ensure it doesn't end with a dash
		for strings.HasSuffix(safeTargetPod, "-") {
			safeTargetPod = safeTargetPod[:len(safeTargetPod)-1]
		}
	}

	safePVCNameLabel := pvcName
	if len(pvcName) > 63 {
		safePVCNameLabel = pvcName[:63]
		// Ensure it doesn't end with a dash
		for strings.HasSuffix(safePVCNameLabel, "-") {
			safePVCNameLabel = safePVCNameLabel[:len(safePVCNameLabel)-1]
		}
	}

	// Create the performance pod using the proper Kubernetes API types
	perfPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      perfPodName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":        "pvc-perf-monitor",
				"target-pod": safeTargetPod,
				"pvc-name":   safePVCNameLabel,
			},
		},
		Spec: corev1.PodSpec{
			NodeName: pod.Spec.NodeName, // Ensure we run on the same node to access the PVC
			Containers: []corev1.Container{
				{
					Name:  "perf-monitor",
					Image: "nicolaka/netshoot", // Container with network and IO diagnostic tools
					Command: []string{
						"sh",
						"-c",
						`echo "Starting PVC Performance Monitor (Read-Only mode)"
						if [ -d "/mnt/pvc" ]; then
							echo "PVC mounted successfully at /mnt/pvc in read-only mode"
							# Show basic info about the mount
							df -h /mnt/pvc
							mount | grep /mnt/pvc
							
							# In loop, monitor the PVC
							while true; do
								echo "------- PVC Monitor $(date) -------"
								# Show disk usage in a parseable format
								echo "DISK_USAGE_BEGIN"
								df -h /mnt/pvc | grep -v "Filesystem"
								echo "DISK_USAGE_END"
								
								# System stats
								echo "SYSTEM_STATS_BEGIN"
								top -bn1 | grep "load average:" | head -1
								echo "SYSTEM_STATS_END"
								
								# IO stats
								echo "IO_STATS_BEGIN"
								iostat -dxh 1 1 | grep -v "loop\|ram"
								echo "IO_STATS_END"
								
								# Disk throughput estimation
								echo "THROUGHPUT_BEGIN"
								dd if=/dev/zero of=/dev/null bs=1M count=1000 2>&1 | grep -i "bytes"
								echo "THROUGHPUT_END"
								
								sleep 1
							done
						else
							echo "Error: PVC directory /mnt/pvc not found"
							exit 1
						fi`,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "pvc-volume",
							MountPath: "/mnt/pvc",
							ReadOnly:  true, // Mount as read-only to prevent any writes
						},
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: boolPtr(false),
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("50m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "pvc-volume",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
							ReadOnly:  true, // Mount as read-only to prevent any writes
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	// Create the pod
	createdPod, err := c.Clientset.CoreV1().Pods(namespace).Create(context.TODO(), perfPod, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("error creating performance pod: %v", err)
	}

	return createdPod.Name, nil
}
