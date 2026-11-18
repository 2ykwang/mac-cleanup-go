package target

import (
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func newTestScannedBrew(t *testing.T) (*BrewTarget, *types.ScanResult) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "Homebrew")
	newTestCacheFile(t, filepath.Join(root, "package"))
	newTestBrewCommand(t, root)
	tgt := NewBrewTarget(types.Category{ID: "homebrew", Name: "Homebrew Cache"})
	result, err := tgt.Scan()
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	return tgt, result
}

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
	t.Setenv("PATH", t.TempDir())
	tgt := NewBrewTarget(types.Category{ID: "homebrew"})

	result, err := tgt.Scan()

	require.NoError(t, err)
	assert.Empty(t, result.Items)
	assert.False(t, tgt.IsAvailable())
}

func TestBrewTarget_GetBrewCachePath_CachesConcurrentResolution(t *testing.T) {
	root := t.TempDir()
	calls := newTestBrewCommand(t, root)
	tgt := NewBrewTarget(types.Category{ID: "homebrew"})
	var wg sync.WaitGroup
	paths := make([]string, 8)

	for i := range paths {
		wg.Add(1)
		go func() {
			defer wg.Done()
			paths[i] = tgt.getBrewCachePath()
		}()
	}
	wg.Wait()

	for _, path := range paths {
		assert.Equal(t, root, path)
	}
	assert.Equal(t, 1, *calls)
}

func TestBrewTarget_GetBrewCachePath_CommandFailureIsCached(t *testing.T) {
	calls := newTestBrewCommand(t, "")
	execCommand = func(string, ...string) *exec.Cmd {
		*calls++
		return exec.Command("/bin/sh", "-c", "exit 1")
	}
	tgt := NewBrewTarget(types.Category{ID: "homebrew"})

	first := tgt.getBrewCachePath()
	second := tgt.getBrewCachePath()

	assert.Empty(t, first)
	assert.Empty(t, second)
	assert.Equal(t, 1, *calls)
}

func TestBrewTarget_Scan_MeasuresOnlyCacheFiles(t *testing.T) {
	tgt, _ := newTestScannedBrew(t)
	newTestCacheFile(t, filepath.Join(tgt.cachePath, ".DS_Store"))

	result, err := tgt.Scan()

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "Homebrew Cache", result.Items[0].Name)
	assert.True(t, result.Items[0].IsDirectory)
	assert.Equal(t, int64(5), result.TotalSize)
	assert.Equal(t, int64(1), result.TotalFileCount)
}

func TestBrewTarget_Scan_NonexistentCacheIsEmpty(t *testing.T) {
	newTestBrewCommand(t, filepath.Join(t.TempDir(), "missing"))
	tgt := NewBrewTarget(types.Category{ID: "homebrew"})

	result, err := tgt.Scan()

	require.NoError(t, err)
	assert.Empty(t, result.Items)
}

func TestBrewTarget_Clean_ReturnsEmptyWhenNoItems(t *testing.T) {
	calls := newTestBrewCommand(t, t.TempDir())
	tgt := NewBrewTarget(types.Category{ID: "homebrew"})

	result, err := tgt.Clean(nil)

	require.NoError(t, err)
	assert.Zero(t, result.CleanedItems)
	assert.Zero(t, *calls)
}

func TestBrewTarget_Clean_RejectsUnscannedPaths(t *testing.T) {
	tgt, scan := newTestScannedBrew(t)
	root := scan.Items[0].Path
	items := []types.CleanableItem{
		{Path: root + "-other"},
		{Path: filepath.Join(root, "package")},
		{Path: filepath.Join(root, "..", "unrelated")},
		{Path: filepath.Dir(root)},
	}

	result, err := tgt.Clean(items)

	require.NoError(t, err)
	assert.Len(t, result.Errors, len(items))
	assert.Zero(t, result.CleanedItems)
	assert.DirExists(t, root)
}
