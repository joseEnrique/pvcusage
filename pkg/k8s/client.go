package k8s

import (
	"context"
	"encoding/json"
	"fmt"

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
