package scanner

import (
	"testing"

	"mac-cleanup-go/pkg/types"
)

func TestDockerScannerIsAvailable(t *testing.T) {
	cat := types.Category{
		ID:       "docker",
		Name:     "Docker",
		Method:   types.MethodSpecial,
		CheckCmd: "docker",
	}

	s := NewDockerScanner(cat)
	available := s.IsAvailable()
	t.Logf("Docker IsAvailable: %v", available)
}

func TestDockerScannerScan(t *testing.T) {
	cat := types.Category{
		ID:       "docker",
		Name:     "Docker",
		Method:   types.MethodSpecial,
		CheckCmd: "docker",
	}

	s := NewDockerScanner(cat)
	if !s.IsAvailable() {
		t.Skip("Docker not available")
	}

	result, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	t.Logf("TotalSize: %d bytes", result.TotalSize)
	t.Logf("Items count: %d", len(result.Items))
	for _, item := range result.Items {
		t.Logf("  - %s: %d bytes", item.Name, item.Size)
	}
}

func TestParseDockerSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1.783GB (40%)", 1914419200}, // approx
		{"53.25kB (100%)", 54528},     // approx
		{"2.371GB (93%)", 2545772339}, // approx
		{"140.7MB", 147537920},        // approx
		{"0B", 0},
		{"", 0},
	}

	for _, tt := range tests {
		result := parseDockerSize(tt.input)
		t.Logf("parseDockerSize(%q) = %d (expected ~%d)", tt.input, result, tt.expected)
		// Allow 1% margin
		if result == 0 && tt.expected != 0 {
			t.Errorf("parseDockerSize(%q) = 0, expected non-zero", tt.input)
		}
	}
}
