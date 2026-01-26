package target

import (
	"math"
	"os/exec"
	"strings"
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

	utils.CommandExists = func(_ string) bool {
		return true
	}
	execCommand = func(_ string, args ...string) *exec.Cmd {
		if len(args) > 0 && args[0] == "version" {
			return exec.Command("true")
		}
		if len(args) >= 3 && args[0] == "system" && args[1] == "df" && args[2] == "-v" {
			return exec.Command("false")
		}
		return exec.Command("true")
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
	verboseOutput := `{"Images":[{"ID":"sha256:img1","Repository":"repo","Tag":"latest","Size":"1GB"}],"Containers":[{"Image":"repo:latest","Names":"web","Mounts":"vol1"}],"Volumes":[{"Name":"vol1","Size":"500MB"}],"BuildCache":[]}`
	summaryOutput := `{"Type":"Build Cache","TotalCount":"1","Active":"0","Size":"1GB","Reclaimable":"1GB (100%)"}`
	execCommand = func(_ string, args ...string) *exec.Cmd {
		if len(args) > 0 && args[0] == "version" {
			return exec.Command("true")
		}
		if len(args) >= 3 && args[0] == "system" && args[1] == "df" && args[2] == "-v" {
			return exec.Command("echo", verboseOutput)
		}
		if len(args) >= 2 && args[0] == "system" && args[1] == "df" {
			return exec.Command("echo", summaryOutput)
		}
		return exec.Command("true")
	}

	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerTarget(cat)

	result, err := s.Scan()

	assert.NoError(t, err)
	assert.Nil(t, result.Error)
	assert.Len(t, result.Items, 3)

	var imageItem, volumeItem, cacheItem *types.CleanableItem
	for i := range result.Items {
		item := &result.Items[i]
		switch {
		case strings.HasPrefix(item.Path, "docker:image:"):
			imageItem = item
		case strings.HasPrefix(item.Path, "docker:volume:"):
			volumeItem = item
		case item.Path == "docker:build-cache":
			cacheItem = item
		}
	}

	if assert.NotNil(t, imageItem) {
		assert.Contains(t, imageItem.DisplayName, "Image: repo:latest")
		assert.Contains(t, imageItem.DisplayName, "Used By: web")
		assert.Equal(t, "docker:image:sha256:img1", imageItem.Path)
		assert.Equal(t, types.ItemStatusProcessLocked, imageItem.Status)
	}
	if assert.NotNil(t, volumeItem) {
		assert.Contains(t, volumeItem.DisplayName, "Volume: vol1")
		assert.Contains(t, volumeItem.DisplayName, "Used By: web")
		assert.Equal(t, types.ItemStatusProcessLocked, volumeItem.Status)
	}
	assert.NotNil(t, cacheItem)
}

func TestDockerTarget_Scan_EmptyVerboseOutput(t *testing.T) {
	originalCommandExists := utils.CommandExists
	original := execCommand
	defer func() {
		utils.CommandExists = originalCommandExists
		execCommand = original
	}()

	utils.CommandExists = func(_ string) bool {
		return true
	}
	summaryOutput := `{"Type":"Build Cache","TotalCount":"1","Active":"0","Size":"1GB","Reclaimable":"1GB (100%)"}`
	execCommand = func(_ string, args ...string) *exec.Cmd {
		if len(args) > 0 && args[0] == "version" {
			return exec.Command("true")
		}
		if len(args) >= 3 && args[0] == "system" && args[1] == "df" && args[2] == "-v" {
			return exec.Command("echo", "")
		}
		if len(args) >= 2 && args[0] == "system" && args[1] == "df" {
			return exec.Command("echo", summaryOutput)
		}
		return exec.Command("true")
	}

	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerTarget(cat)

	result, err := s.Scan()

	assert.NoError(t, err)
	assert.Nil(t, result.Error)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, "docker:build-cache", result.Items[0].Path)
}

func TestDockerTarget_Scan_InvalidVerboseJSON(t *testing.T) {
	originalCommandExists := utils.CommandExists
	original := execCommand
	defer func() {
		utils.CommandExists = originalCommandExists
		execCommand = original
	}()

	utils.CommandExists = func(_ string) bool {
		return true
	}
	execCommand = func(_ string, args ...string) *exec.Cmd {
		if len(args) > 0 && args[0] == "version" {
			return exec.Command("true")
		}
		if len(args) >= 3 && args[0] == "system" && args[1] == "df" && args[2] == "-v" {
			return exec.Command("echo", "not-json")
		}
		return exec.Command("true")
	}

	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerTarget(cat)

	result, err := s.Scan()

	assert.NoError(t, err)
	assert.NotNil(t, result.Error)
	assert.Empty(t, result.Items)
}

func TestDockerTarget_Scan_BuildCacheMissing(t *testing.T) {
	originalCommandExists := utils.CommandExists
	original := execCommand
	defer func() {
		utils.CommandExists = originalCommandExists
		execCommand = original
	}()

	utils.CommandExists = func(_ string) bool {
		return true
	}
	verboseOutput := `{"Images":[{"ID":"sha256:img1","Repository":"repo","Tag":"latest","Size":"1GB"}],"Containers":[],"Volumes":[{"Name":"vol1","Size":"500MB"}]}`
	summaryOutput := `{"Type":"Images","TotalCount":"1","Active":"0","Size":"1GB","Reclaimable":"1GB (100%)"}`
	execCommand = func(_ string, args ...string) *exec.Cmd {
		if len(args) > 0 && args[0] == "version" {
			return exec.Command("true")
		}
		if len(args) >= 3 && args[0] == "system" && args[1] == "df" && args[2] == "-v" {
			return exec.Command("echo", verboseOutput)
		}
		if len(args) >= 2 && args[0] == "system" && args[1] == "df" {
			return exec.Command("echo", summaryOutput)
		}
		return exec.Command("true")
	}

	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerTarget(cat)

	result, err := s.Scan()

	assert.NoError(t, err)
	assert.Nil(t, result.Error)
	assert.Len(t, result.Items, 2)
	for _, item := range result.Items {
		assert.NotEqual(t, "docker:build-cache", item.Path)
	}
}

func TestDockerTarget_Scan_UntaggedImageLabel(t *testing.T) {
	originalCommandExists := utils.CommandExists
	original := execCommand
	defer func() {
		utils.CommandExists = originalCommandExists
		execCommand = original
	}()

	utils.CommandExists = func(_ string) bool {
		return true
	}
	verboseOutput := `{"Images":[{"ID":"sha256:img1","Repository":"<none>","Tag":"<none>","Size":"1GB"}],"Containers":[{"Image":"sha256:img1","Names":"worker","Mounts":""}],"Volumes":[]}`
	execCommand = func(_ string, args ...string) *exec.Cmd {
		if len(args) > 0 && args[0] == "version" {
			return exec.Command("true")
		}
		if len(args) >= 3 && args[0] == "system" && args[1] == "df" && args[2] == "-v" {
			return exec.Command("echo", verboseOutput)
		}
		if len(args) >= 2 && args[0] == "system" && args[1] == "df" {
			return exec.Command("echo", `{"Type":"Build Cache","TotalCount":"0","Active":"0","Size":"0B","Reclaimable":"0B"}`)
		}
		return exec.Command("true")
	}

	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerTarget(cat)

	result, err := s.Scan()

	assert.NoError(t, err)
	assert.Nil(t, result.Error)
	if assert.Len(t, result.Items, 1) {
		assert.Contains(t, result.Items[0].DisplayName, "Image: untagged@img1")
		assert.Contains(t, result.Items[0].DisplayName, "Used By: worker")
		assert.Equal(t, "docker:image:sha256:img1", result.Items[0].Path)
	}
}

func TestDockerTarget_Scan_MultiTagImagesHaveUniquePaths(t *testing.T) {
	originalCommandExists := utils.CommandExists
	original := execCommand
	defer func() {
		utils.CommandExists = originalCommandExists
		execCommand = original
	}()

	utils.CommandExists = func(_ string) bool {
		return true
	}
	verboseOutput := `{"Images":[{"ID":"sha256:img1","Repository":"repo","Tag":"latest","Size":"1GB"},{"ID":"sha256:img1","Repository":"repo","Tag":"dev","Size":"1GB"}],"Containers":[],"Volumes":[]}`
	execCommand = func(_ string, args ...string) *exec.Cmd {
		if len(args) > 0 && args[0] == "version" {
			return exec.Command("true")
		}
		if len(args) >= 3 && args[0] == "system" && args[1] == "df" && args[2] == "-v" {
			return exec.Command("echo", verboseOutput)
		}
		if len(args) >= 2 && args[0] == "system" && args[1] == "df" {
			return exec.Command("echo", `{"Type":"Build Cache","TotalCount":"0","Active":"0","Size":"0B","Reclaimable":"0B"}`)
		}
		return exec.Command("true")
	}

	cat := types.Category{ID: "docker", Name: "Docker"}
	s := NewDockerTarget(cat)

	result, err := s.Scan()

	assert.NoError(t, err)
	if assert.Len(t, result.Items, 1) {
		assert.Contains(t, result.Items[0].DisplayName, "Image: repo:latest")
		assert.Contains(t, result.Items[0].DisplayName, "(+1 tags)")
		assert.Equal(t, "docker:image:sha256:img1", result.Items[0].Path)
		assert.Equal(t, parseDockerSize("1GB"), result.TotalSize)
	}
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
		{Path: "docker:image:sha256:abc", Size: 1000},
		{Path: "docker:volume:vol1", Size: 3000},
		{Path: "docker:build-cache", Size: 4000},
	}

	result, err := s.Clean(items)

	assert.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Equal(t, int64(8000), result.FreedSpace)
	assert.Equal(t, 3, result.CleanedItems)
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
		{Path: "docker:image:sha256:abc", Size: 1000},
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
