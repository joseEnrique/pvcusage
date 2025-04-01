package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joseEnrique/pvcusage/internal/display"
	"github.com/joseEnrique/pvcusage/internal/k8s"
	"github.com/joseEnrique/pvcusage/internal/perf"
	"github.com/joseEnrique/pvcusage/internal/pvc"
)

func main() {
	// Define flags.
	watchFlag := flag.Bool("watch", false, "Enable watch mode (refresh every s seconds)")
	interval := flag.Int("s", 5, "Interval in seconds for watch mode")
	filter := flag.String("filter", "", "Filter PVCs by usage percentage (e.g. '>50', '<=80', '=90')")
	topN := flag.Int("top", 0, "Show only top N PVCs by usage percentage")

	// New flags for PVC performance analysis
	pvcNameFlag := flag.String("pvc", "", "Name of a specific PVC to analyze")
	namespaceFlag := flag.String("namespace", "", "Namespace of the PVC to analyze or filter PVCs by namespace")
	perfFlag := flag.Bool("perf", false, "Enable performance monitoring for the specified PVC")

	flag.Parse()

	// Create Kubernetes client
	client, err := k8s.NewClient()
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}

	// If a specific PVC is provided with the perf flag, analyze its performance
	if *pvcNameFlag != "" && *perfFlag {
		if *namespaceFlag == "" {
			log.Fatalf("Error: -namespace flag is required when using -pvc with -perf")
		}

		log.Printf("Starting performance analysis for PVC '%s' in namespace '%s'...", *pvcNameFlag, *namespaceFlag)

		// Find pod that uses this PVC
		pod, err := client.FindPodUsingPVC(*namespaceFlag, *pvcNameFlag)
		if err != nil {
			log.Fatalf("Error finding pod using PVC: %v", err)
		}

		if pod == "" {
			log.Fatalf("No pod found using PVC '%s' in namespace '%s'", *pvcNameFlag, *namespaceFlag)
		}

		log.Printf("Found pod '%s' using the PVC", pod)

		// Start performance monitoring
		perfMonitor, err := perf.StartMonitoring(client, *namespaceFlag, pod, *pvcNameFlag)
		if err != nil {
			log.Fatalf("Error starting performance monitoring: %v", err)
		}

		// Setup signal handling for graceful termination
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		// Show performance metrics in real-time
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				display.ClearScreen()
				display.ShowPerfMetrics(perfMonitor.GetLatestMetrics())
			case <-sigs:
				fmt.Println("\nStopping performance monitoring...")
				perfMonitor.Stop()
				return
			}
		}
	} else if *watchFlag {
		// Regular watch mode for PVC usage
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		// Show first update immediately
		updateTableWithNamespaceFilter(client, *filter, *namespaceFlag, *topN)

		// Then start the ticker for subsequent updates
		ticker := time.NewTicker(time.Duration(*interval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				display.ClearScreen()
				updateTableWithNamespaceFilter(client, *filter, *namespaceFlag, *topN)
			case <-sigs:
				fmt.Println("\nTerminating watch mode...")
				return
			}
		}
	} else {
		// One-time display of PVC usage
		updateTableWithNamespaceFilter(client, *filter, *namespaceFlag, *topN)
	}
}

// updateTableWithNamespaceFilter gets PVC usage data, filters by namespace if provided, then by other criteria
func updateTableWithNamespaceFilter(client *k8s.Client, filterExpression, namespace string, topN int) {
	usages, err := pvc.GetUsages(client)
	if err != nil {
		log.Printf("Error getting PVC usages: %v", err)
		return
	}

	// First filter by namespace if provided
	if namespace != "" {
		var namespaceFiltered []pvc.Usage
		for _, usage := range usages {
			if usage.Namespace == namespace {
				namespaceFiltered = append(namespaceFiltered, usage)
			}
		}
		usages = namespaceFiltered
		fmt.Printf("Filtered to show only PVCs in namespace: %s\n", namespace)
	}

	// Then apply any additional filtering expression
	filteredUsages, err := pvc.FilterUsages(usages, filterExpression)
	if err != nil {
		log.Printf("Error filtering usages: %v", err)
		return
	}

	// Limit to top N if specified
	limitedUsages := pvc.LimitTopN(filteredUsages, topN)

	// Display results
	table := display.NewTable()
	table.Show(limitedUsages)
}
