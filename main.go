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
)

func main() {
	// Define flags.
	watchFlag := flag.Bool("watch", false, "Enable watch mode (refresh every s seconds)")
	interval := flag.Int("s", 5, "Interval in seconds for watch mode")
	filter := flag.String("filter", "", "Filter PVCs by usage percentage (e.g. '>50', '<=80', '=90')")
	topN := flag.Int("top", 0, "Show only top N PVCs by usage percentage")
	flag.Parse()

	// Create Kubernetes client
	client, err := k8s.NewClient()
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}

	if *watchFlag {
		// Setup signal handling for graceful termination.
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		// Show first update immediately
		display.UpdateTable(client, *filter, *topN)

		// Then start the ticker for subsequent updates
		ticker := time.NewTicker(time.Duration(*interval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				display.ClearScreen()
				display.UpdateTable(client, *filter, *topN)
			case <-sigs:
				fmt.Println("\nTerminating watch mode...")
				return
			}
		}
	} else {
		display.UpdateTable(client, *filter, *topN)
	}
}
