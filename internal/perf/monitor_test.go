package perf

import (
	"testing"
)

func TestParseHumanSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"0", 0},
		{"100", 100},
		{"1K", 1024},
		{"1M", 1024 * 1024},
		{"1G", 1024 * 1024 * 1024},
		{"1.5G", int64(1.5 * 1024 * 1024 * 1024)},
		{"500M", 500 * 1024 * 1024},
		{"", 0},
		{"invalid", 0},
	}

	for _, test := range tests {
		result := parseHumanSize(test.input)
		if result != test.expected {
			t.Errorf("parseHumanSize(%q) = %d; want %d", test.input, result, test.expected)
		}
	}
}
