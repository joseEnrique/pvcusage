package display

import (
	"fmt"
	"os"
	"text/tabwriter"

	"pvcusage/internal/pvc"
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
