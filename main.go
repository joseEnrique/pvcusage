package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"syscall"
	"text/tabwriter"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Summary represents the node stats summary structure.
type Summary struct {
	Pods []Pod `json:"pods"`
}

// Pod contains volume usage stats.
type Pod struct {
	Volumes []Volume `json:"volume"`
}

// Volume represents a volume in the stats summary.
// We extract data only for volumes that have a PVC reference.
type Volume struct {
	PVCRef *struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	} `json:"pvcRef"`
	CapacityBytes  int64 `json:"capacityBytes"`
	UsedBytes      int64 `json:"usedBytes"`
	AvailableBytes int64 `json:"availableBytes"`
}

// PVCUsage holds the PVC-related usage information for output.
type PVCUsage struct {
	Namespace      string
	PVC            string
	CapacityBytes  int64
	UsedBytes      int64
	AvailableBytes int64
	PercentageUsed float64
}

// getNodes returns the list of node names using the Kubernetes SDK.
func getNodes(clientset *kubernetes.Clientset) ([]string, error) {
	nodeList, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	nodes := make([]string, 0, len(nodeList.Items))
	for _, node := range nodeList.Items {
		nodes = append(nodes, node.Name)
	}
	return nodes, nil
}

// getSummary fetches and decodes the stats summary from a node via its proxy endpoint.
func getSummary(clientset *kubernetes.Clientset, node string) (*Summary, error) {
	// Build the API path equivalent to: /api/v1/nodes/<node>/proxy/stats/summary
	path := fmt.Sprintf("/api/v1/nodes/%s/proxy/stats/summary", node)
	res := clientset.RESTClient().Get().AbsPath(path).Do(context.TODO())
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

// humanizeBytes converts a byte count into a human-readable IEC string.
func humanizeBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// parseFilter parses a filter string like ">50", "<=80", "=90" and returns the operator and value
func parseFilter(filter string) (string, float64, error) {
	if filter == "" {
		return "", 0, nil
	}

	// Extract operator and value
	var operator string
	var value float64
	var err error

	switch filter[0] {
	case '>', '<', '=':
		operator = string(filter[0])
		if len(filter) > 1 && filter[1] == '=' {
			operator += "="
			value, err = strconv.ParseFloat(filter[2:], 64)
		} else {
			value, err = strconv.ParseFloat(filter[1:], 64)
		}
	default:
		// If no operator specified, assume ">"
		operator = ">"
		value, err = strconv.ParseFloat(filter, 64)
	}

	if err != nil {
		return "", 0, fmt.Errorf("invalid filter value: %v", err)
	}

	return operator, value, nil
}

// updateTable queries all nodes, extracts PVC usage data, and prints a formatted table.
func updateTable(clientset *kubernetes.Clientset, filter string, topN int) {
	nodes, err := getNodes(clientset)
	if err != nil {
		log.Printf("Error getting nodes: %v", err)
		return
	}
	var usages []PVCUsage
	for _, node := range nodes {
		summary, err := getSummary(clientset, node)
		if err != nil {
			log.Printf("Error getting summary for node %s: %v", node, err)
			continue
		}
		for _, pod := range summary.Pods {
			for _, vol := range pod.Volumes {
				if vol.PVCRef != nil {
					// Avoid division by zero.
					if vol.CapacityBytes == 0 {
						continue
					}
					percentage := float64(vol.UsedBytes) / float64(vol.CapacityBytes) * 100
					usages = append(usages, PVCUsage{
						Namespace:      vol.PVCRef.Namespace,
						PVC:            vol.PVCRef.Name,
						CapacityBytes:  vol.CapacityBytes,
						UsedBytes:      vol.UsedBytes,
						AvailableBytes: vol.AvailableBytes,
						PercentageUsed: percentage,
					})
				}
			}
		}
	}

	// Sort by percentage used (descending)
	sort.Slice(usages, func(i, j int) bool {
		return usages[i].PercentageUsed > usages[j].PercentageUsed
	})

	// Parse and apply filter
	operator, value, err := parseFilter(filter)
	if err != nil {
		log.Printf("Error parsing filter: %v", err)
		return
	}

	var filteredUsages []PVCUsage
	for _, u := range usages {
		if operator != "" {
			include := false
			switch operator {
			case ">":
				include = u.PercentageUsed > value
			case ">=":
				include = u.PercentageUsed >= value
			case "<":
				include = u.PercentageUsed < value
			case "<=":
				include = u.PercentageUsed <= value
			case "=":
				include = u.PercentageUsed == value
			}
			if !include {
				continue
			}
		}
		filteredUsages = append(filteredUsages, u)
	}

	// Apply top N limit
	if topN > 0 && len(filteredUsages) > topN {
		filteredUsages = filteredUsages[:topN]
	}

	// Set up a tabwriter for column alignment.
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Namespace\tPVC\tSize\tUsed\tAvail\tUse%")
	for _, u := range filteredUsages {
		capStr := humanizeBytes(u.CapacityBytes)
		usedStr := humanizeBytes(u.UsedBytes)
		availStr := humanizeBytes(u.AvailableBytes)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%.0f%%\n",
			u.Namespace, u.PVC, capStr, usedStr, availStr, u.PercentageUsed)
	}
	w.Flush()
}

// clearScreen clears the terminal (works on most ANSI terminals).
func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func main() {
	// Define flags.
	watchFlag := flag.Bool("watch", false, "Enable watch mode (refresh every s seconds)")
	interval := flag.Int("s", 5, "Interval in seconds for watch mode")
	filter := flag.String("filter", "", "Filter PVCs by usage percentage (e.g. '>50', '<=80', '=90')")
	topN := flag.Int("top", 0, "Show only top N PVCs by usage percentage")
	flag.Parse()

	// Create Kubernetes client configuration.
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := clientcmd.RecommendedHomeFile
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			log.Fatalf("Error building kubeconfig: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}

	if *watchFlag {
		// Setup signal handling for graceful termination.
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		// Show first update immediately
		updateTable(clientset, *filter, *topN)

		// Then start the ticker for subsequent updates
		ticker := time.NewTicker(time.Duration(*interval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				clearScreen()
				updateTable(clientset, *filter, *topN)
			case <-sigs:
				fmt.Println("\nTerminating watch mode...")
				return
			}
		}
	} else {
		updateTable(clientset, *filter, *topN)
	}
}
