package pvc

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

// Usage holds the PVC-related usage information for output.
type Usage struct {
	Namespace      string
	PVC            string
	CapacityBytes  int64
	UsedBytes      int64
	AvailableBytes int64
	PercentageUsed float64
}
