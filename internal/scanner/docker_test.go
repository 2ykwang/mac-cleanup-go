package scanner

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/2ykwang/mac-cleanup-go/pkg/types"
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

func TestParseDockerSize_KBWithPercentage(t *testing.T) {
	result := parseDockerSize("53.25kB (100%)")

	kb := float64(1024)
	expected := int64(53.25 * kb)
	assertWithinMargin(t, "53.25kB (100%)", result, expected, 0.01)
}

func TestParseDockerSize_GBWithPercentage(t *testing.T) {
	result := parseDockerSize("2.371GB (93%)")

	gb := float64(1024 * 1024 * 1024)
	expected := int64(2.371 * gb)
	assertWithinMargin(t, "2.371GB (93%)", result, expected, 0.01)
}

func TestParseDockerSize_ZeroBytes(t *testing.T) {
	result := parseDockerSize("0B")

	assert.Equal(t, int64(0), result)
}

func TestParseDockerSize_EmptyString(t *testing.T) {
	result := parseDockerSize("")

	assert.Equal(t, int64(0), result)
}

func TestParseDockerSize_Bytes(t *testing.T) {
	assert.Equal(t, int64(100), parseDockerSize("100B"))
}

func TestParseDockerSize_KB(t *testing.T) {
	assert.Equal(t, int64(1024), parseDockerSize("1KB"))
}

func TestParseDockerSize_MB(t *testing.T) {
	assert.Equal(t, int64(1024*1024), parseDockerSize("1MB"))
}

func TestParseDockerSize_GB(t *testing.T) {
	assert.Equal(t, int64(1024*1024*1024), parseDockerSize("1GB"))
}

func TestParseDockerSize_TB(t *testing.T) {
	assert.Equal(t, int64(1024*1024*1024*1024), parseDockerSize("1TB"))
}

func TestParseDockerSize_CaseInsensitive(t *testing.T) {
	expected := int64(1024 * 1024 * 1024)

	assert.Equal(t, expected, parseDockerSize("1gb"))
	assert.Equal(t, expected, parseDockerSize("1GB"))
	assert.Equal(t, expected, parseDockerSize("1Gb"))
}

func TestParseDockerSize_WithWhitespace(t *testing.T) {
	expected := int64(1024 * 1024 * 1024)

	assert.Equal(t, expected, parseDockerSize("  1GB  "))
	assert.Equal(t, expected, parseDockerSize("1GB "))
	assert.Equal(t, expected, parseDockerSize(" 1GB"))
}

func TestDockerTypeName_Images(t *testing.T) {
	assert.Equal(t, "Docker Images", dockerTypeName("images"))
	assert.Equal(t, "Docker Images", dockerTypeName("Images"))
	assert.Equal(t, "Docker Images", dockerTypeName("IMAGES"))
}

func TestDockerTypeName_Containers(t *testing.T) {
	assert.Equal(t, "Docker Containers", dockerTypeName("containers"))
	assert.Equal(t, "Docker Containers", dockerTypeName("Containers"))
}

func TestDockerTypeName_Volumes(t *testing.T) {
	assert.Equal(t, "Docker Volumes [!DB DATA RISK]", dockerTypeName("local volumes"))
	assert.Equal(t, "Docker Volumes [!DB DATA RISK]", dockerTypeName("Local Volumes"))
}

func TestDockerTypeName_BuildCache(t *testing.T) {
	assert.Equal(t, "Docker Build Cache", dockerTypeName("build cache"))
	assert.Equal(t, "Docker Build Cache", dockerTypeName("Build Cache"))
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
