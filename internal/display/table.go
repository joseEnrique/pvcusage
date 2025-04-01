package display

import (
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/joseEnrique/pvcusage/internal/k8s"
	"github.com/joseEnrique/pvcusage/internal/perf"
	"github.com/joseEnrique/pvcusage/internal/pvc"
)

// Table displays PVC usage information in a formatted table
type Table struct {
	writer *tabwriter.Writer
}

// NewTable creates a new table display
func NewTable() *Table {
	return &Table{
		writer: tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0),
	}
}

// Show displays the PVC usages in a formatted table
func (t *Table) Show(usages []pvc.Usage) {
	fmt.Fprintln(t.writer, "Namespace\tPVC\tSize\tUsed\tAvail\tUse%")
	for _, u := range usages {
		capStr := HumanizeBytes(u.CapacityBytes)
		usedStr := HumanizeBytes(u.UsedBytes)
		availStr := HumanizeBytes(u.AvailableBytes)
		fmt.Fprintf(t.writer, "%s\t%s\t%s\t%s\t%s\t%.0f%%\n",
			u.Namespace, u.PVC, capStr, usedStr, availStr, u.PercentageUsed)
	}
	t.writer.Flush()
}

// ClearScreen clears the terminal (works on most ANSI terminals)
func ClearScreen() {
	fmt.Print("\033[H\033[2J")
}

// UpdateTable queries all nodes, extracts PVC usage data, and prints a formatted table.
func UpdateTable(client *k8s.Client, filter string, topN int) {
	usages, err := pvc.GetUsages(client)
	if err != nil {
		log.Printf("Error getting PVC usages: %v", err)
		return
	}

	filteredUsages, err := pvc.FilterUsages(usages, filter)
	if err != nil {
		log.Printf("Error filtering usages: %v", err)
		return
	}

	limitedUsages := pvc.LimitTopN(filteredUsages, topN)

	table := NewTable()
	table.Show(limitedUsages)
}

// ShowPerfMetrics displays performance metrics for a PVC
func ShowPerfMetrics(metrics perf.Metrics) {
	// ANSI colors for better visualization
	green := "\033[32m"
	yellow := "\033[33m"
	red := "\033[31m"
	blue := "\033[34m"
	cyan := "\033[36m"
	bold := "\033[1m"
	reset := "\033[0m"

	// Header
	fmt.Printf("%s%s==================================================%s\n", bold, blue, reset)
	fmt.Printf("%s%sPVC Performance Monitor - %s%s\n", bold, blue, metrics.Timestamp.Format(time.RFC822), reset)
	fmt.Printf("%s%s==================================================%s\n", bold, blue, reset)
	fmt.Println()

	// Storage metrics
	fmt.Printf("%s%sStorage Metrics:%s\n", bold, cyan, reset)
	fmt.Printf("  Capacity:      %s\n", HumanizeBytes(metrics.DiskSpace))

	// Change color based on usage percentage
	usageColor := green
	if metrics.DiskSpacePct > 70 {
		usageColor = yellow
	}
	if metrics.DiskSpacePct > 90 {
		usageColor = red
	}

	fmt.Printf("  Used:          %s%s (%.1f%%)%s\n",
		usageColor,
		HumanizeBytes(metrics.DiskSpaceUsed),
		metrics.DiskSpacePct,
		reset)
	fmt.Printf("  Available:     %s\n", HumanizeBytes(metrics.DiskSpace-metrics.DiskSpaceUsed))

	// Use if/else instead of ternary operator (which doesn't exist in Go)
	modeStr := "Read-Write"
	if metrics.ReadOnly {
		modeStr = "Read-Only"
	}
	fmt.Printf("  Mode:          %s%s%s\n", bold, modeStr, reset)

	// Performance metrics
	fmt.Printf("\n%s%sPerformance Metrics:%s\n", bold, cyan, reset)
	fmt.Printf("  IOPS:          %d ops/sec\n", metrics.IOPS)
	fmt.Printf("  Throughput:    %s/sec\n", HumanizeBytes(metrics.Throughput))
	fmt.Printf("  Latency:       %dms\n", metrics.Latency)
	fmt.Printf("  Disk Util:     %.1f%%\n", metrics.DiskUtilPct)

	// System metrics
	fmt.Printf("\n%s%sSystem Metrics:%s\n", bold, cyan, reset)

	// Change color based on load
	loadColor := green
	if metrics.SystemLoad > 1.0 {
		loadColor = yellow
	}
	if metrics.SystemLoad > 2.0 {
		loadColor = red
	}

	fmt.Printf("  System Load:   %s%.2f%s\n", loadColor, metrics.SystemLoad, reset)

	// Change color based on IO wait
	ioWaitColor := green
	if metrics.CPUWaitPercentage > 5 {
		ioWaitColor = yellow
	}
	if metrics.CPUWaitPercentage > 20 {
		ioWaitColor = red
	}

	fmt.Printf("  I/O Wait:      %s%.1f%%%s\n", ioWaitColor, metrics.CPUWaitPercentage, reset)

	// Visual bar for disk utilization
	fmt.Println()
	fmt.Print("Disk Utilization: ")
	printColorProgressBar(int(metrics.DiskSpacePct), 100, 40)

	// Footer
	fmt.Printf("\n%s%sPress Ctrl+C to stop monitoring%s\n", bold, blue, reset)
}

// printColorProgressBar renders a colored ASCII progress bar
func printColorProgressBar(current, total, width int) {
	percent := float64(current) / float64(total)
	filled := int(percent * float64(width))

	// Choose color based on percentage
	color := "\033[32m" // Green
	if percent > 0.7 {
		color = "\033[33m" // Yellow
	}
	if percent > 0.9 {
		color = "\033[31m" // Red
	}
	reset := "\033[0m"

	fmt.Print("[")
	for i := 0; i < width; i++ {
		if i < filled {
			fmt.Printf("%s#%s", color, reset)
		} else {
			fmt.Print(" ")
		}
	}

	// Use concatenation to avoid direct percent symbol in format string
	percentValue := fmt.Sprintf("%.1f", percent*100)
	fmt.Print("] ")
	fmt.Print(color)
	fmt.Print(percentValue)
	fmt.Print(" percent")
	fmt.Println(reset)
}

// highlightKeywords highlights important keywords in the log
func highlightKeywords(line string) string {
	yellow := "\033[33m"
	red := "\033[31m"
	green := "\033[32m"
	cyan := "\033[36m"
	reset := "\033[0m"

	// Highlight error messages
	if strings.Contains(strings.ToLower(line), "error") || strings.Contains(strings.ToLower(line), "fail") {
		return red + line + reset
	}

	// Highlight warning messages
	if strings.Contains(strings.ToLower(line), "warn") {
		return yellow + line + reset
	}

	// Highlight success messages
	if strings.Contains(strings.ToLower(line), "success") || strings.Contains(strings.ToLower(line), "mounted") {
		return green + line + reset
	}

	// Highlight dates and monitoring headers
	if strings.Contains(line, "PVC Monitor") || strings.Contains(line, "Stats") {
		return cyan + line + reset
	}

	return line
}
