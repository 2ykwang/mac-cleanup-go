package scanner

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"mac-cleanup-go/pkg/types"
)

func TestNewDockerScanner_ReturnsNonNil(t *testing.T) {
	cat := types.Category{ID: "docker", Name: "Docker"}

	s := NewDockerScanner(cat)

	assert.NotNil(t, s)
}

func TestDockerScanner_Category_ReturnsConfiguredCategory(t *testing.T) {
	cat := types.Category{
		ID:     "docker",
		Name:   "Docker",
		Safety: types.SafetyLevelModerate,
	}

	s := NewDockerScanner(cat)
	result := s.Category()

	assert.Equal(t, "docker", result.ID)
	assert.Equal(t, "Docker", result.Name)
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

			assert.Equal(t, tt.expected, result)
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

			assert.Equal(t, tt.expected, result)
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

			assert.Equal(t, tt.expected, result)
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

			assert.Equal(t, tt.expected, result)
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

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDockerTypeName_UnknownType(t *testing.T) {
	result := dockerTypeName("unknown")

	assert.Equal(t, "Docker unknown", result)
}

func TestDockerScanner_Clean_IncludesCategoryInResult(t *testing.T) {
	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerScanner(cat)

	result, err := s.Clean(nil)

	assert.NoError(t, err)
	assert.Equal(t, "docker", result.Category.ID)
}

func TestDockerScanner_Scan_Integration(t *testing.T) {
	cat := types.Category{
		ID:       "docker",
		Name:     "Docker",
		Method:   types.MethodBuiltin,
		CheckCmd: "docker",
	}

	s := NewDockerScanner(cat)
	if !s.IsAvailable() {
		t.Skip("Docker not available")
	}

	result, err := s.Scan()

	assert.NoError(t, err)
	assert.Equal(t, "docker", result.Category.ID)
	t.Logf("Found %d items, total size: %d bytes", len(result.Items), result.TotalSize)
}

func assertWithinMargin(t *testing.T, input string, result, expected int64, marginPercent float64) {
	t.Helper()

	if expected == 0 {
		assert.Equal(t, int64(0), result, "parseDockerSize(%q)", input)
		return
	}

	diff := math.Abs(float64(result-expected)) / float64(expected)
	assert.LessOrEqual(t, diff, marginPercent,
		"parseDockerSize(%q) = %d, expected ~%d (within %.0f%%)",
		input, result, expected, marginPercent*100)
}
