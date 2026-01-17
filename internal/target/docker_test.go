package target

import (
	"math"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

func TestNewDockerTarget_ReturnsNonNil(t *testing.T) {
	cat := types.Category{ID: "docker", Name: "Docker"}

	s := NewDockerTarget(cat)

	assert.NotNil(t, s)
}

func TestDockerTarget_Category_ReturnsConfiguredCategory(t *testing.T) {
	cat := types.Category{
		ID:     "docker",
		Name:   "Docker",
		Safety: types.SafetyLevelModerate,
	}

	s := NewDockerTarget(cat)
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

func TestDockerTarget_Clean_IncludesCategoryInResult(t *testing.T) {
	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerTarget(cat)

	result, err := s.Clean(nil)

	assert.NoError(t, err)
	assert.Equal(t, "docker", result.Category.ID)
}

func TestDockerTarget_Scan_Integration(t *testing.T) {
	cat := types.Category{
		ID:       "docker",
		Name:     "Docker",
		Method:   types.MethodBuiltin,
		CheckCmd: "docker",
	}

	s := NewDockerTarget(cat)
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

func TestDockerTarget_IsAvailable_ReturnsFalse_WhenDockerNotExists(t *testing.T) {
	originalCommandExists := utils.CommandExists
	defer func() { utils.CommandExists = originalCommandExists }()
	utils.CommandExists = func(_ string) bool {
		return false
	}

	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerTarget(cat)

	assert.False(t, s.IsAvailable())
}

func TestDockerTarget_IsAvailable_ReturnsFalse_WhenDockerInfoFails(t *testing.T) {
	originalCommandExists := utils.CommandExists
	original := execCommand
	defer func() {
		utils.CommandExists = originalCommandExists
		execCommand = original
	}()

	utils.CommandExists = func(_ string) bool {
		return true
	}
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("false")
	}

	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerTarget(cat)

	assert.False(t, s.IsAvailable())
}

func TestDockerTarget_IsAvailable_ReturnsTrue_WhenDockerWorks(t *testing.T) {
	originalCommandExists := utils.CommandExists
	original := execCommand
	defer func() {
		utils.CommandExists = originalCommandExists
		execCommand = original
	}()

	utils.CommandExists = func(_ string) bool {
		return true
	}
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("true")
	}

	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerTarget(cat)

	assert.True(t, s.IsAvailable())
}

func TestDockerTarget_Scan_ReturnsEmpty_WhenNotAvailable(t *testing.T) {
	originalCommandExists := utils.CommandExists
	defer func() { utils.CommandExists = originalCommandExists }()
	utils.CommandExists = func(_ string) bool {
		return false
	}

	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerTarget(cat)

	result, err := s.Scan()

	assert.NoError(t, err)
	assert.Empty(t, result.Items)
}

func TestDockerTarget_Scan_ReturnsError_WhenCommandFails(t *testing.T) {
	originalCommandExists := utils.CommandExists
	original := execCommand
	defer func() {
		utils.CommandExists = originalCommandExists
		execCommand = original
	}()

	callCount := 0
	utils.CommandExists = func(_ string) bool {
		return true
	}
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			// docker info succeeds
			return exec.Command("true")
		}
		// docker system df fails
		return exec.Command("false")
	}

	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerTarget(cat)

	result, err := s.Scan()

	assert.NoError(t, err)
	assert.NotNil(t, result.Error)
}

func TestDockerTarget_Scan_ParsesOutput(t *testing.T) {
	originalCommandExists := utils.CommandExists
	original := execCommand
	defer func() {
		utils.CommandExists = originalCommandExists
		execCommand = original
	}()

	utils.CommandExists = func(_ string) bool {
		return true
	}
	callCount := 0
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			// docker info
			return exec.Command("true")
		}
		// docker system df
		output := `{"Type":"Images","TotalCount":"5","Active":"2","Size":"2.371GB","Reclaimable":"1.5GB (63%)"}
{"Type":"Containers","TotalCount":"3","Active":"1","Size":"100MB","Reclaimable":"50MB (50%)"}
{"Type":"Local Volumes","TotalCount":"2","Active":"0","Size":"500MB","Reclaimable":"500MB (100%)"}
{"Type":"Build Cache","TotalCount":"10","Active":"0","Size":"1GB","Reclaimable":"1GB (100%)"}`
		return exec.Command("echo", output)
	}

	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerTarget(cat)

	result, err := s.Scan()

	assert.NoError(t, err)
	assert.Nil(t, result.Error)
	assert.Len(t, result.Items, 4)
	assert.Equal(t, "docker:images", result.Items[0].Path)
	assert.Equal(t, "Docker Images", result.Items[0].Name)
}

func TestDockerTarget_Scan_SkipsEmptyLines(t *testing.T) {
	originalCommandExists := utils.CommandExists
	original := execCommand
	defer func() {
		utils.CommandExists = originalCommandExists
		execCommand = original
	}()

	utils.CommandExists = func(_ string) bool {
		return true
	}
	callCount := 0
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			return exec.Command("true")
		}
		output := `{"Type":"Images","TotalCount":"5","Active":"2","Size":"2GB","Reclaimable":"1GB"}

{"Type":"Build Cache","TotalCount":"10","Active":"0","Size":"1GB","Reclaimable":"1GB"}`
		return exec.Command("echo", output)
	}

	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerTarget(cat)

	result, err := s.Scan()

	assert.NoError(t, err)
	assert.Len(t, result.Items, 2)
}

func TestDockerTarget_Scan_SkipsZeroReclaimable(t *testing.T) {
	originalCommandExists := utils.CommandExists
	original := execCommand
	defer func() {
		utils.CommandExists = originalCommandExists
		execCommand = original
	}()

	utils.CommandExists = func(_ string) bool {
		return true
	}
	callCount := 0
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			return exec.Command("true")
		}
		output := `{"Type":"Images","TotalCount":"5","Active":"2","Size":"2GB","Reclaimable":"0B"}`
		return exec.Command("echo", output)
	}

	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerTarget(cat)

	result, err := s.Scan()

	assert.NoError(t, err)
	assert.Empty(t, result.Items)
}

func TestDockerTarget_Clean_AllTypes(t *testing.T) {
	original := execCommand
	defer func() { execCommand = original }()

	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("true")
	}

	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerTarget(cat)

	items := []types.CleanableItem{
		{Path: "docker:images", Size: 1000},
		{Path: "docker:containers", Size: 2000},
		{Path: "docker:local volumes", Size: 3000},
		{Path: "docker:build cache", Size: 4000},
	}

	result, err := s.Clean(items)

	assert.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Equal(t, int64(10000), result.FreedSpace)
	assert.Equal(t, 4, result.CleanedItems)
}

func TestDockerTarget_Clean_RecordsErrors(t *testing.T) {
	original := execCommand
	defer func() { execCommand = original }()

	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("false")
	}

	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerTarget(cat)

	items := []types.CleanableItem{
		{Path: "docker:images", Size: 1000},
	}

	result, err := s.Clean(items)

	assert.NoError(t, err)
	assert.NotEmpty(t, result.Errors)
	assert.Equal(t, int64(0), result.FreedSpace)
	assert.Equal(t, 0, result.CleanedItems)
}

func TestDockerTarget_Clean_SkipsUnknownPath(t *testing.T) {
	original := execCommand
	defer func() { execCommand = original }()

	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("true")
	}

	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerTarget(cat)

	items := []types.CleanableItem{
		{Path: "docker:unknown", Size: 1000},
	}

	result, err := s.Clean(items)

	assert.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Equal(t, int64(0), result.FreedSpace)
	assert.Equal(t, 0, result.CleanedItems)
}
