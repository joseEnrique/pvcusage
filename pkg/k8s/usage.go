package k8s

import (
	"fmt"
	"sort"
	"strconv"
)

// GetUsages retrieves and calculates PVC usage across all nodes
func GetUsages(client *Client) ([]Usage, error) {
	nodes, err := client.GetNodes()
	if err != nil {
		return nil, fmt.Errorf("error getting nodes: %v", err)
	}

	var usages []Usage
	for _, node := range nodes {
		summary, err := client.GetSummary(node)
		if err != nil {
			// Log error but continue with other nodes
			fmt.Printf("Error getting summary for node %s: %v\n", node, err)
			continue
		}

		for _, pod := range summary.Pods {
			for _, vol := range pod.Volumes {
				if vol.PVCRef != nil {
					if vol.CapacityBytes == 0 {
						continue
					}
					percentage := float64(vol.UsedBytes) / float64(vol.CapacityBytes) * 100
					usages = append(usages, Usage{
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

	return usages, nil
}

// ParseFilter parses a filter string like ">50", "<=80", "=90"
func ParseFilter(filter string) (string, float64, error) {
	if filter == "" {
		return "", 0, nil
	}

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
		operator = ">"
		value, err = strconv.ParseFloat(filter, 64)
	}

	if err != nil {
		return "", 0, fmt.Errorf("invalid filter value: %v", err)
	}

	return operator, value, nil
}

// FilterUsages applies the filter to the usages list
func FilterUsages(usages []Usage, filter string) ([]Usage, error) {
	operator, value, err := ParseFilter(filter)
	if err != nil {
		return nil, err
	}

	var filteredUsages []Usage
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

	return filteredUsages, nil
}

// LimitTopN limits the usages list to the top N entries
func LimitTopN(usages []Usage, n int) []Usage {
	if n > 0 && len(usages) > n {
		return usages[:n]
	}
	return usages
}
