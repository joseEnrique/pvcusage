package display

import (
	"testing"
)

func TestHumanizeBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0B"},
		{100, "100B"},
		{1023, "1023B"},
		{1024, "1.0KiB"},
		{1536, "1.5KiB"},
		{1024 * 1024, "1.0MiB"},
		{1024 * 1024 * 2.5, "2.5MiB"},
		{1024 * 1024 * 1024, "1.0GiB"},
		{1024 * 1024 * 1024 * 1024, "1.0TiB"},
		{1024 * 1024 * 1024 * 1024 * 1024, "1.0PiB"},
	}

	for _, test := range tests {
		result := HumanizeBytes(test.input)
		if result != test.expected {
			t.Errorf("HumanizeBytes(%d) = %s; want %s", test.input, result, test.expected)
		}
	}
}
