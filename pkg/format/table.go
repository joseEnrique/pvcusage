package format

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/joseEnrique/pvcusage/pkg/k8s"
)

// Table represents a formatted table display
type Table struct {
	writer *tabwriter.Writer
}

// NewTable creates a new table formatter
func NewTable() *Table {
	return &Table{
		writer: tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0),
	}
}

// Show displays the PVC usage table
func (t *Table) Show(usages []k8s.Usage) {
	fmt.Fprintln(t.writer, "Namespace\tPVC\tSize\tUsed\tAvail\tUse%")
	for _, u := range usages {
		fmt.Fprintf(t.writer, "%s\t%s\t%s\t%s\t%s\t%.0f%%\n",
			u.Namespace,
			u.PVC,
			HumanizeBytes(u.CapacityBytes),
			HumanizeBytes(u.UsedBytes),
			HumanizeBytes(u.AvailableBytes),
			u.PercentageUsed)
	}
	t.writer.Flush()
}

// ClearScreen clears the terminal
func ClearScreen() {
	fmt.Print("\033[H\033[2J")
}
