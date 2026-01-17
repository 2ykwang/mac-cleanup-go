package target

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

func TestNewBrewTarget_ReturnsNonNil(t *testing.T) {
	cat := types.Category{ID: "homebrew", Name: "Homebrew"}

	s := NewBrewTarget(cat)

	assert.NotNil(t, s)
}

func TestNewBrewTarget_StoresCategory(t *testing.T) {
	cat := types.Category{
		ID:     "homebrew",
		Name:   "Homebrew Cache",
		Safety: types.SafetyLevelSafe,
	}

	s := NewBrewTarget(cat)

	assert.Equal(t, "homebrew", s.category.ID)
	assert.Equal(t, "Homebrew Cache", s.category.Name)
}

func TestBrewTarget_Category_ReturnsConfiguredCategory(t *testing.T) {
	cat := types.Category{
		ID:     "homebrew",
		Name:   "Homebrew",
		Safety: types.SafetyLevelModerate,
	}
	s := NewBrewTarget(cat)

	result := s.Category()

	assert.Equal(t, "homebrew", result.ID)
	assert.Equal(t, "Homebrew", result.Name)
	assert.Equal(t, types.SafetyLevelModerate, result.Safety)
}

func TestBrewTarget_Scan_ReturnsEmptyWhenNotAvailable(t *testing.T) {
	cat := types.Category{ID: "homebrew", Name: "Homebrew"}
	s := NewBrewTarget(cat)

	result, err := s.Scan()

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "homebrew", result.Category.ID)
}

func TestBrewTarget_GetBrewCachePath_CachesResult(t *testing.T) {
	cat := types.Category{ID: "homebrew", Name: "Homebrew"}
	s := NewBrewTarget(cat)

	// First call
	path1 := s.getBrewCachePath()
	// Second call should return cached value
	path2 := s.getBrewCachePath()

	assert.Equal(t, path1, path2)
}

func TestBrewTarget_Clean_ReturnsResult(t *testing.T) {
	cat := types.Category{ID: "homebrew", Name: "Homebrew"}
	s := NewBrewTarget(cat)
	s.cachePath = "/nonexistent/path"

	items := []types.CleanableItem{
		{Path: "/nonexistent/path", Size: 1000, Name: "Homebrew Cache", IsDirectory: true},
	}

	result, err := s.Clean(items)

	assert.NoError(t, err)
	assert.Equal(t, "homebrew", result.Category.ID)
	// MoveToTrash will fail for nonexistent path, but Clean should not return error
	assert.NotEmpty(t, result.Errors)
}

func TestBrewTarget_Scan_WithMockCachePath(t *testing.T) {
	if !utils.CommandExists("brew") {
		t.Skip("brew not installed")
	}

	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "Homebrew")
	require.NoError(t, os.MkdirAll(cacheDir, 0o755))

	testFile := filepath.Join(cacheDir, "test-package.tar.gz")
	require.NoError(t, os.WriteFile(testFile, []byte("test content for brew cache"), 0o644))

	cat := types.Category{ID: "homebrew", Name: "Homebrew"}
	s := NewBrewTarget(cat)
	s.cachePath = cacheDir

	result, err := s.Scan()

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, "Homebrew Cache", result.Items[0].Name)
	assert.True(t, result.Items[0].IsDirectory)
	assert.Greater(t, result.TotalSize, int64(0))
}

func TestBrewTarget_Scan_NonexistentCachePath(t *testing.T) {
	if !utils.CommandExists("brew") {
		t.Skip("brew not installed")
	}

	cat := types.Category{ID: "homebrew", Name: "Homebrew"}
	s := NewBrewTarget(cat)
	s.cachePath = "/nonexistent/path/that/does/not/exist"

	result, err := s.Scan()

	assert.NoError(t, err)
	assert.Empty(t, result.Items)
}
