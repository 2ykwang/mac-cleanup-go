package scanner

import (
	"math"
	"testing"

	"mac-cleanup-go/pkg/types"
)

func TestNewDockerScanner_ReturnsNonNil(t *testing.T) {
	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerScanner(cat)

	if s == nil {
		t.Fatal("NewDockerScanner returned nil")
	}
}

func TestDockerScanner_Category_ReturnsConfiguredCategory(t *testing.T) {
	cat := types.Category{
		ID:     "docker",
		Name:   "Docker",
		Safety: types.SafetyLevelModerate,
	}
	s := NewDockerScanner(cat)

	result := s.Category()

	if result.ID != "docker" {
		t.Errorf("Expected ID 'docker', got '%s'", result.ID)
	}
	if result.Name != "Docker" {
		t.Errorf("Expected Name 'Docker', got '%s'", result.Name)
	}
}

func TestDockerScanner_IsAvailable_ReturnsBool(t *testing.T) {
	cat := types.Category{ID: "docker", CheckCmd: "docker"}
	s := NewDockerScanner(cat)

	available := s.IsAvailable()
	t.Logf("Docker available: %v", available)
}

// Test inputs from: docker system df --format "{{json .}}"
func TestParseDockerSize_RealDockerFormat(t *testing.T) {
	const KB = 1024
	const GB = 1024 * 1024 * 1024

	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{"kB with percentage", "53.25kB (100%)", 53.25 * KB},
		{"kB zero percent", "148.3kB (0%)", 148.3 * KB},
		{"GB with percentage", "2.371GB (93%)", 2.371 * GB},
		{"GB size field", "2.601GB", 2.601 * GB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDockerSize(tt.input)
			assertWithinMargin(t, tt.input, result, int64(tt.expected), 0.01)
		})
	}
}

func TestParseDockerSize_ZeroAndEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"zero bytes", "0B", 0},
		{"empty string", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDockerSize(tt.input)
			if result != tt.expected {
				t.Errorf("parseDockerSize(%q) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseDockerSize_AllUnits(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"bytes", "100B", 100},
		{"kilobytes", "1KB", 1024},
		{"megabytes", "1MB", 1024 * 1024},
		{"gigabytes", "1GB", 1024 * 1024 * 1024},
		{"terabytes", "1TB", 1024 * 1024 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDockerSize(tt.input)
			if result != tt.expected {
				t.Errorf("parseDockerSize(%q) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseDockerSize_CaseInsensitive(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1gb", 1024 * 1024 * 1024},
		{"1GB", 1024 * 1024 * 1024},
		{"1Gb", 1024 * 1024 * 1024},
		{"1mb", 1024 * 1024},
		{"1kb", 1024},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDockerSize(tt.input)
			if result != tt.expected {
				t.Errorf("parseDockerSize(%q) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseDockerSize_WithWhitespace(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"  1GB  ", 1024 * 1024 * 1024},
		{"1GB ", 1024 * 1024 * 1024},
		{" 1GB", 1024 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDockerSize(tt.input)
			if result != tt.expected {
				t.Errorf("parseDockerSize(%q) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDockerTypeName_AllTypes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"images", "Docker Images"},
		{"Images", "Docker Images"},
		{"IMAGES", "Docker Images"},
		{"containers", "Docker Containers"},
		{"Containers", "Docker Containers"},
		{"local volumes", "Docker Volumes [!DB DATA RISK]"},
		{"Local Volumes", "Docker Volumes [!DB DATA RISK]"},
		{"build cache", "Docker Build Cache"},
		{"Build Cache", "Docker Build Cache"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := dockerTypeName(tt.input)
			if result != tt.expected {
				t.Errorf("dockerTypeName(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDockerTypeName_UnknownType(t *testing.T) {
	result := dockerTypeName("unknown")

	if result != "Docker unknown" {
		t.Errorf("Expected 'Docker unknown', got '%s'", result)
	}
}

func TestDockerScanner_Clean_DryRun(t *testing.T) {
	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerScanner(cat)

	items := []types.CleanableItem{
		{Path: "docker:images", Size: 1000},
		{Path: "docker:containers", Size: 2000},
	}

	result, err := s.Clean(items, true)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.CleanedItems != 2 {
		t.Errorf("Expected CleanedItems 2, got %d", result.CleanedItems)
	}
	if result.FreedSpace != 3000 {
		t.Errorf("Expected FreedSpace 3000, got %d", result.FreedSpace)
	}
}

func TestDockerScanner_Clean_DryRun_EmptyItems(t *testing.T) {
	cat := types.Category{ID: "docker"}
	s := NewDockerScanner(cat)

	result, err := s.Clean([]types.CleanableItem{}, true)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.CleanedItems != 0 {
		t.Errorf("Expected CleanedItems 0, got %d", result.CleanedItems)
	}
}

func TestDockerScanner_Clean_IncludesCategoryInResult(t *testing.T) {
	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerScanner(cat)

	result, err := s.Clean(nil, true)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.Category.ID != "docker" {
		t.Errorf("Expected category ID 'docker', got '%s'", result.Category.ID)
	}
}

func TestDockerScanner_Scan_Integration(t *testing.T) {
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
	if result.Category.ID != "docker" {
		t.Errorf("Expected category ID 'docker', got '%s'", result.Category.ID)
	}
	t.Logf("Found %d items, total size: %d bytes", len(result.Items), result.TotalSize)
}

func assertWithinMargin(t *testing.T, input string, result, expected int64, marginPercent float64) {
	t.Helper()

	if expected == 0 {
		if result != 0 {
			t.Errorf("parseDockerSize(%q) = %d, expected 0", input, result)
		}
		return
	}

	diff := math.Abs(float64(result-expected)) / float64(expected)
	if diff > marginPercent {
		t.Errorf("parseDockerSize(%q) = %d, expected ~%d (within %.0f%%), diff was %.2f%%",
			input, result, expected, marginPercent*100, diff*100)
	}
}
