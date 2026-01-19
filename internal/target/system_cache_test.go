package target

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func newSystemCacheTargetWithArcCategory() *SystemCacheTarget {
	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{"/tmp/test/Caches/*"},
	}
	otherCategories := []types.Category{
		{ID: "browser-arc", Paths: []string{"/tmp/test/Caches/Arc/*"}},
	}
	allCategories := append([]types.Category{systemCache}, otherCategories...)
	return NewSystemCacheTarget(systemCache, allCategories)
}

func TestIsExcluded_WhenPathMatchesOtherCategory_ReturnsTrue(t *testing.T) {
	s := newSystemCacheTargetWithArcCategory()

	assert.True(t, s.isExcluded("/tmp/test/Caches/Arc/cache.db"))
}

func TestIsExcluded_WhenPathNotInAnyCategory_ReturnsFalse(t *testing.T) {
	s := newSystemCacheTargetWithArcCategory()

	assert.False(t, s.isExcluded("/tmp/test/Caches/RandomApp/data"))
}

func TestIsExcluded_WhenEmptyPath_ReturnsFalse(t *testing.T) {
	s := newSystemCacheTargetWithArcCategory()

	assert.False(t, s.isExcluded(""))
}

func TestNewSystemCacheTarget_CollectsPathsFromOtherCategories(t *testing.T) {
	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{"/tmp/test/Caches/*"},
	}
	otherCategories := []types.Category{
		{ID: "browser-arc", Paths: []string{"/tmp/test/Caches/Arc/*", "/tmp/test/Caches/company.thebrowser.Browser/*"}},
		{ID: "homebrew", Paths: []string{"/tmp/test/Caches/Homebrew/*"}},
	}
	allCategories := append([]types.Category{systemCache}, otherCategories...)
	s := NewSystemCacheTarget(systemCache, allCategories)

	assert.Len(t, s.excludePaths, 3)
}

func TestNewSystemCacheTarget_DoesNotIncludeOwnPaths(t *testing.T) {
	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{"/tmp/test/Caches/*"},
	}
	allCategories := []types.Category{systemCache}
	s := NewSystemCacheTarget(systemCache, allCategories)

	assert.Empty(t, s.excludePaths, "should not include own paths")
}

func TestNewSystemCacheTarget_WhenNoCategoriesProvided_CreatesEmptyExcludes(t *testing.T) {
	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{"/tmp/test/Caches/*"},
	}
	s := NewSystemCacheTarget(systemCache, nil)

	assert.False(t, s.isExcluded("/tmp/test/Caches/AnyApp/file"), "no exclusions when no categories provided")
}

func TestIsExcluded_WhenNestedPath_ReturnsTrue(t *testing.T) {
	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{"/tmp/test/Caches/*"},
	}
	otherCategories := []types.Category{
		{ID: "app", Paths: []string{"/tmp/test/Caches/App/*"}},
	}
	allCategories := append([]types.Category{systemCache}, otherCategories...)
	s := NewSystemCacheTarget(systemCache, allCategories)

	assert.True(t, s.isExcluded("/tmp/test/Caches/App/sub/deep/file"), "deeply nested path should be excluded")
}

func TestIsExcluded_WhenSimilarPrefix_ReturnsFalse(t *testing.T) {
	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{"/tmp/test/Caches/*"},
	}
	otherCategories := []types.Category{
		{ID: "app", Paths: []string{"/tmp/test/Caches/App/*"}},
	}
	allCategories := append([]types.Category{systemCache}, otherCategories...)
	s := NewSystemCacheTarget(systemCache, allCategories)

	assert.False(t, s.isExcluded("/tmp/test/Caches/AppOther/data"), "path with similar prefix but different directory should not be excluded")
}

func TestScan_ExcludesPathsFromOtherCategories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "systemcache-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cachesDir := filepath.Join(tmpDir, "Caches")
	arcDir := filepath.Join(cachesDir, "Arc")
	randomDir := filepath.Join(cachesDir, "RandomApp")
	jetbrainsDir := filepath.Join(cachesDir, "JetBrains")

	for _, dir := range []string{arcDir, randomDir, jetbrainsDir} {
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "cache.dat"), []byte("test"), 0o644))
	}

	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{filepath.Join(cachesDir, "*")},
	}
	otherCategories := []types.Category{
		{ID: "browser-arc", Paths: []string{filepath.Join(cachesDir, "Arc", "*")}},
		{ID: "jetbrains", Paths: []string{filepath.Join(cachesDir, "JetBrains", "*")}},
	}
	allCategories := append([]types.Category{systemCache}, otherCategories...)
	s := NewSystemCacheTarget(systemCache, allCategories)

	result, err := s.Scan()
	require.NoError(t, err)
	require.Len(t, result.Items, 1, "should only include RandomApp")
	assert.Equal(t, "RandomApp", result.Items[0].Name)
}

func TestScan_WhenNoMatchingPaths_ReturnsEmptyResult(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "systemcache-empty-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{filepath.Join(tmpDir, "NonExistent", "*")},
	}
	s := NewSystemCacheTarget(systemCache, []types.Category{systemCache})

	result, err := s.Scan()
	require.NoError(t, err)
	assert.Empty(t, result.Items)
}

func newSystemCacheTargetForExclusionTest() *SystemCacheTarget {
	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{"/tmp/test/Caches/*"},
	}
	otherCategories := []types.Category{
		{ID: "with-star", Paths: []string{"/tmp/test/Caches/App1/*"}},
		{ID: "with-double-star", Paths: []string{"/tmp/test/Caches/App2/**"}},
		{ID: "no-trailing", Paths: []string{"/tmp/test/Caches/App3"}},
	}
	allCategories := append([]types.Category{systemCache}, otherCategories...)
	return NewSystemCacheTarget(systemCache, allCategories)
}

func TestIsExcluded_PathWithStarPattern(t *testing.T) {
	s := newSystemCacheTargetForExclusionTest()

	result := s.isExcluded("/tmp/test/Caches/App1/file")

	assert.True(t, result)
}

func TestIsExcluded_PathWithDoubleStarPattern(t *testing.T) {
	s := newSystemCacheTargetForExclusionTest()

	result := s.isExcluded("/tmp/test/Caches/App2/file")

	assert.True(t, result)
}

func TestIsExcluded_PathWithNoTrailingPattern(t *testing.T) {
	s := newSystemCacheTargetForExclusionTest()

	result := s.isExcluded("/tmp/test/Caches/App3/file")

	assert.True(t, result)
}

func TestIsExcluded_PathNotInAnyCategory(t *testing.T) {
	s := newSystemCacheTargetForExclusionTest()

	result := s.isExcluded("/tmp/test/Caches/App4/file")

	assert.False(t, result)
}

func TestSystemCacheScan_CalculatesTotalFileCount(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "systemcache-filecount-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cachesDir := filepath.Join(tmpDir, "Caches")
	appDir := filepath.Join(cachesDir, "TestApp")
	require.NoError(t, os.MkdirAll(appDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(appDir, "file1.dat"), []byte("test"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(appDir, "file2.dat"), []byte("test"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(appDir, "file3.dat"), []byte("test"), 0o644))
	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{filepath.Join(cachesDir, "*")},
	}
	s := NewSystemCacheTarget(systemCache, []types.Category{systemCache})

	result, err := s.Scan()

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, int64(3), result.Items[0].FileCount)
	assert.Equal(t, int64(3), result.TotalFileCount)
}

func TestScan_MarksLockedItems(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "systemcache-locked-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cachesDir := filepath.Join(tmpDir, "Caches")
	appDir := filepath.Join(cachesDir, "App")
	otherDir := filepath.Join(cachesDir, "Other")

	for _, dir := range []string{appDir, otherDir} {
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "cache.dat"), []byte("test"), 0o644))
	}

	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{filepath.Join(cachesDir, "*")},
	}
	s := NewSystemCacheTarget(systemCache, []types.Category{systemCache})

	originalGetLockedPaths := getLockedPaths
	getLockedPaths = func(basePath string) (map[string]bool, error) {
		assert.Equal(t, cachesDir, basePath)
		return map[string]bool{
			appDir: true,
		}, nil
	}
	defer func() { getLockedPaths = originalGetLockedPaths }()

	result, err := s.Scan()
	require.NoError(t, err)

	itemsByPath := make(map[string]types.CleanableItem)
	for _, item := range result.Items {
		itemsByPath[item.Path] = item
	}

	require.Contains(t, itemsByPath, appDir)
	require.Contains(t, itemsByPath, otherDir)
	assert.Equal(t, types.ItemStatusProcessLocked, itemsByPath[appDir].Status)
	assert.Equal(t, types.ItemStatusAvailable, itemsByPath[otherDir].Status)
}

func TestMarkLockedItems_WhenNoItems_SkipsLockCheck(t *testing.T) {
	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{"/tmp/test/Caches/*"},
	}
	s := NewSystemCacheTarget(systemCache, []types.Category{systemCache})

	called := false
	originalGetLockedPaths := getLockedPaths
	getLockedPaths = func(_ string) (map[string]bool, error) {
		called = true
		return map[string]bool{}, nil
	}
	defer func() { getLockedPaths = originalGetLockedPaths }()

	s.markLockedItems(&types.ScanResult{Items: nil})

	assert.False(t, called)
}

func TestMarkLockedItems_WhenBasePathEmpty_SkipsLockCheck(t *testing.T) {
	systemCache := types.Category{
		ID:    "system-cache",
		Paths: nil,
	}
	s := NewSystemCacheTarget(systemCache, []types.Category{systemCache})

	called := false
	originalGetLockedPaths := getLockedPaths
	getLockedPaths = func(_ string) (map[string]bool, error) {
		called = true
		return map[string]bool{}, nil
	}
	defer func() { getLockedPaths = originalGetLockedPaths }()

	result := &types.ScanResult{
		Items: []types.CleanableItem{{Path: "/tmp/test/Caches/App"}},
	}
	s.markLockedItems(result)

	assert.False(t, called)
	assert.Equal(t, types.ItemStatusAvailable, result.Items[0].Status)
}

func TestMarkLockedItems_WhenLockCheckFails_DoesNotUpdate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "systemcache-lockfail-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cachesDir := filepath.Join(tmpDir, "Caches")
	appDir := filepath.Join(cachesDir, "App")
	require.NoError(t, os.MkdirAll(appDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(appDir, "cache.dat"), []byte("test"), 0o644))

	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{filepath.Join(cachesDir, "*")},
	}
	s := NewSystemCacheTarget(systemCache, []types.Category{systemCache})

	originalGetLockedPaths := getLockedPaths
	getLockedPaths = func(_ string) (map[string]bool, error) {
		return nil, errors.New("lsof failed")
	}
	defer func() { getLockedPaths = originalGetLockedPaths }()

	result := &types.ScanResult{
		Items: []types.CleanableItem{{Path: appDir}},
	}
	s.markLockedItems(result)

	assert.Equal(t, types.ItemStatusAvailable, result.Items[0].Status)
}
