package perf

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/joseEnrique/pvcusage/internal/k8s"
)

// FileInfo represents information about a file
type FileInfo struct {
	Name string
	Size string
}

// Metrics represents performance metrics for a PVC
type Metrics struct {
	Timestamp         time.Time
	IOPS              int64
	Throughput        int64 // bytes per second
	Latency           int64 // milliseconds
	DiskUtilPct       float64
	DiskSpace         int64
	DiskSpaceUsed     int64
	DiskSpacePct      float64
	ReadOnly          bool
	FileCount         int64
	LogFileCount      int64
	DatabaseFileCount int64
	JSONFileCount     int64
	LargestFiles      []FileInfo
	SystemLoad        float64
	CPUWaitPercentage float64
	LastLogLines      []string
}

// Monitor watches the performance of a PVC
type Monitor struct {
	client    *k8s.Client
	namespace string
	podName   string
	pvcName   string
	perfPod   string
	stopChan  chan struct{}
	metrics   *Metrics
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// StartMonitoring creates a performance pod and starts collecting metrics
func StartMonitoring(client *k8s.Client, namespace, podName, pvcName string) (*Monitor, error) {
	// Create a monitoring pod that accesses the same PVC
	perfPod, err := client.CreatePerformancePod(namespace, podName, pvcName)
	if err != nil {
		return nil, fmt.Errorf("error creating performance pod: %v", err)
	}

	// Wait for the pod to be ready
	log.Printf("Waiting for performance pod %s to be ready...", perfPod)
	err = waitForPodReady(client, namespace, perfPod)
	if err != nil {
		return nil, fmt.Errorf("error waiting for performance pod: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	m := &Monitor{
		client:    client,
		namespace: namespace,
		podName:   podName,
		pvcName:   pvcName,
		perfPod:   perfPod,
		stopChan:  make(chan struct{}),
		metrics:   &Metrics{},
		ctx:       ctx,
		cancel:    cancel,
	}

	// Start collecting metrics
	go m.collectMetrics()

	return m, nil
}

// waitForPodReady waits until the pod is in the Running state
func waitForPodReady(client *k8s.Client, namespace, podName string) error {
	for {
		pod, err := client.Clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if pod.Status.Phase == "Running" {
			return nil
		}

		log.Printf("Pod %s is in %s state, waiting...", podName, pod.Status.Phase)
		time.Sleep(1 * time.Second)
	}
}

// collectMetrics continuously collects performance metrics
func (m *Monitor) collectMetrics() {
	ticker := time.NewTicker(1 * time.Second) // Collect every 1 second
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			metrics, err := m.getMetricsFromPod()
			if err != nil {
				log.Printf("Error collecting metrics: %v", err)
				continue
			}

			m.mu.Lock()
			m.metrics = metrics
			m.mu.Unlock()

		case <-m.ctx.Done():
			return
		}
	}
}

// getMetricsFromPod gets metrics from the pod by parsing logs and status
func (m *Monitor) getMetricsFromPod() (*Metrics, error) {
	// Get PVC details to get real disk space stats
	pvc, err := m.client.Clientset.CoreV1().PersistentVolumeClaims(m.namespace).Get(
		context.TODO(), m.pvcName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting PVC: %v", err)
	}

	// Get logs from the performance pod
	logs, err := m.client.Clientset.CoreV1().Pods(m.namespace).GetLogs(m.perfPod, &corev1.PodLogOptions{
		TailLines: int64Ptr(50), // Get the last 50 lines
	}).Do(context.TODO()).Raw()
	if err != nil {
		log.Printf("Warning: couldn't get logs from performance pod: %v", err)
	}

	// Get storage capacity
	storageQuantity := pvc.Spec.Resources.Requests["storage"]
	storageBytes := storageQuantity.Value()

	// Simulate used storage (in a real scenario, this would be from df command in the pod)
	usedBytes := storageBytes / 2 // just a placeholder
	usedPct := float64(usedBytes) / float64(storageBytes) * 100

	// Extract the latest lines that are non-empty
	logLines := extractNonEmptyLines(string(logs), 5)

	// Check if the PVC is mounted read-only
	isReadOnly := false
	for _, volume := range pvc.Spec.AccessModes {
		if volume == corev1.ReadOnlyMany {
			isReadOnly = true
			break
		}
	}
	// Also check if we specifically mounted it read-only
	if !isReadOnly {
		isReadOnly = true // We always mount read-only now
	}

	// Create sample metrics with additional fields
	return &Metrics{
		Timestamp:         time.Now(),
		IOPS:              int64(100 + time.Now().Second()),                  // simulate varying IOPS
		Throughput:        int64(1024 * 1024 * (5 + time.Now().Second()%10)), // 5-15 MB/s
		Latency:           int64(1 + time.Now().Second()%5),                  // 1-5ms
		DiskUtilPct:       float64(30 + time.Now().Second()%40),              // 30-70%
		DiskSpace:         storageBytes,
		DiskSpaceUsed:     usedBytes,
		DiskSpacePct:      usedPct,
		ReadOnly:          isReadOnly,
		FileCount:         int64(1000 + time.Now().Second()%2000), // 1000-3000 files
		LogFileCount:      int64(200 + time.Now().Second()%300),   // 200-500 log files
		DatabaseFileCount: int64(10 + time.Now().Second()%20),     // 10-30 db files
		JSONFileCount:     int64(50 + time.Now().Second()%100),    // 50-150 JSON files
		LargestFiles:      generateSampleLargestFiles(),
		SystemLoad:        1.0 + float64(time.Now().Second()%100)/50.0, // 1.0 - 3.0
		CPUWaitPercentage: float64(1 + time.Now().Second()%30),         // 1-30%
		LastLogLines:      logLines,
	}, nil
}

// Helper function to generate sample largest files
func generateSampleLargestFiles() []FileInfo {
	files := []FileInfo{
		{Name: "/mnt/pvc/logs/server.log", Size: "245MB"},
		{Name: "/mnt/pvc/data/db.sqlite", Size: "122MB"},
		{Name: "/mnt/pvc/kafka/segment.dat", Size: "98MB"},
		{Name: "/mnt/pvc/config/settings.json", Size: "45MB"},
		{Name: "/mnt/pvc/temp/cache.bin", Size: "28MB"},
	}
	return files
}

// Helper function to convert int to pointer for TailLines
func int64Ptr(i int64) *int64 {
	return &i
}

// Helper function to extract non-empty lines from log output
func extractNonEmptyLines(logs string, maxLines int) []string {
	if logs == "" {
		return []string{"No logs available from monitor pod"}
	}

	// Split logs by newline and extract non-empty lines
	var lines []string
	for _, line := range strings.Split(logs, "\n") {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine != "" {
			lines = append(lines, trimmedLine)
			if len(lines) >= maxLines {
				break
			}
		}
	}

	// If no lines were found, return a default message
	if len(lines) == 0 {
		return []string{"No meaningful data in logs"}
	}

	return lines
}

// GetLatestMetrics returns the most recent metrics
func (m *Monitor) GetLatestMetrics() Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m.metrics
}

// Stop stops monitoring and cleans up resources
func (m *Monitor) Stop() error {
	m.cancel()

	// Delete the performance pod
	err := m.client.Clientset.CoreV1().Pods(m.namespace).Delete(
		context.TODO(), m.perfPod, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("error deleting performance pod: %v", err)
	}

	log.Printf("Performance pod %s deleted", m.perfPod)
	return nil
}
