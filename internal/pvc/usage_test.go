package pvc

import (
	"testing"
)

func TestParseFilter(t *testing.T) {
	tests := []struct {
		input        string
		wantOperator string
		wantValue    float64
		wantErr      bool
	}{
		{">50", ">", 50, false},
		{"<50", "<", 50, false},
		{">=50", ">=", 50, false},
		{"<=50", "<=", 50, false},
		{"=50", "=", 50, false},
		{"50", ">", 50, false},
		{"", "", 0, false},
		{"invalid", "", 0, true},
	}

	for _, tt := range tests {
		gotOperator, gotValue, err := ParseFilter(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseFilter(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if err != nil {
			continue
		}
		if gotOperator != tt.wantOperator {
			t.Errorf("ParseFilter(%q) operator = %q, want %q", tt.input, gotOperator, tt.wantOperator)
		}
		if gotValue != tt.wantValue {
			t.Errorf("ParseFilter(%q) value = %v, want %v", tt.input, gotValue, tt.wantValue)
		}
	}
}

func TestFilterUsages(t *testing.T) {
	usages := []Usage{
		{PVC: "pvc1", PercentageUsed: 10},
		{PVC: "pvc2", PercentageUsed: 50},
		{PVC: "pvc3", PercentageUsed: 90},
	}

	tests := []struct {
		filter     string
		wantLength int
	}{
		{">20", 2},
		{">80", 1},
		{"<20", 1},
		{"=50", 1},
		{">=50", 2},
		{"", 3},
	}

	for _, tt := range tests {
		got, err := FilterUsages(usages, tt.filter)
		if err != nil {
			t.Errorf("FilterUsages(%q) unexpected error: %v", tt.filter, err)
			continue
		}
		if len(got) != tt.wantLength {
			t.Errorf("FilterUsages(%q) returned %d items, want %d", tt.filter, len(got), tt.wantLength)
		}
	}
}

func TestLimitTopN(t *testing.T) {
	usages := []Usage{
		{PVC: "pvc1"},
		{PVC: "pvc2"},
		{PVC: "pvc3"},
	}

	tests := []struct {
		n          int
		wantLength int
	}{
		{1, 1},
		{2, 2},
		{3, 3},
		{4, 3},
		{0, 3},
		{-1, 3},
	}

	for _, tt := range tests {
		got := LimitTopN(usages, tt.n)
		if len(got) != tt.wantLength {
			t.Errorf("LimitTopN(%d) returned %d items, want %d", tt.n, len(got), tt.wantLength)
		}
	}
}
