package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mac-cleanup-go/pkg/types"
)

func TestIsExcluded_WhenPathMatchesOtherCategory_ReturnsTrue(t *testing.T) {
	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{"/tmp/test/Caches/*"},
	}
	otherCategories := []types.Category{
		{ID: "browser-arc", Paths: []string{"/tmp/test/Caches/Arc/*"}},
	}
	allCategories := append([]types.Category{systemCache}, otherCategories...)
	s := NewSystemCacheScanner(systemCache, allCategories)

	assert.True(t, s.isExcluded("/tmp/test/Caches/Arc/cache.db"), "path in other category should be excluded")
}

func TestIsExcluded_WhenPathNotInAnyCategory_ReturnsFalse(t *testing.T) {
	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{"/tmp/test/Caches/*"},
	}
	otherCategories := []types.Category{
		{ID: "browser-arc", Paths: []string{"/tmp/test/Caches/Arc/*"}},
	}
	allCategories := append([]types.Category{systemCache}, otherCategories...)
	s := NewSystemCacheScanner(systemCache, allCategories)

	assert.False(t, s.isExcluded("/tmp/test/Caches/RandomApp/data"), "path not in any category should not be excluded")
}

func TestIsExcluded_WhenEmptyPath_ReturnsFalse(t *testing.T) {
	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{"/tmp/test/Caches/*"},
	}
	otherCategories := []types.Category{
		{ID: "browser-arc", Paths: []string{"/tmp/test/Caches/Arc/*"}},
	}
	allCategories := append([]types.Category{systemCache}, otherCategories...)
	s := NewSystemCacheScanner(systemCache, allCategories)

	assert.False(t, s.isExcluded(""), "empty path should not be excluded")
}

func TestNewSystemCacheScanner_CollectsPathsFromOtherCategories(t *testing.T) {
	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{"/tmp/test/Caches/*"},
	}
	otherCategories := []types.Category{
		{ID: "browser-arc", Paths: []string{"/tmp/test/Caches/Arc/*", "/tmp/test/Caches/company.thebrowser.Browser/*"}},
		{ID: "homebrew", Paths: []string{"/tmp/test/Caches/Homebrew/*"}},
	}
	allCategories := append([]types.Category{systemCache}, otherCategories...)
	s := NewSystemCacheScanner(systemCache, allCategories)

	assert.Len(t, s.excludePaths, 3)
}

func TestNewSystemCacheScanner_DoesNotIncludeOwnPaths(t *testing.T) {
	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{"/tmp/test/Caches/*"},
	}
	allCategories := []types.Category{systemCache}
	s := NewSystemCacheScanner(systemCache, allCategories)

	assert.Empty(t, s.excludePaths, "should not include own paths")
}

func TestNewSystemCacheScanner_WhenNoCategoriesProvided_CreatesEmptyExcludes(t *testing.T) {
	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{"/tmp/test/Caches/*"},
	}
	s := NewSystemCacheScanner(systemCache, nil)

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
	s := NewSystemCacheScanner(systemCache, allCategories)

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
	s := NewSystemCacheScanner(systemCache, allCategories)

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
	s := NewSystemCacheScanner(systemCache, allCategories)

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
	s := NewSystemCacheScanner(systemCache, []types.Category{systemCache})

	result, err := s.Scan()
	require.NoError(t, err)
	assert.Empty(t, result.Items)
}

func TestIsExcluded_WithVariousPathPatterns(t *testing.T) {
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
	s := NewSystemCacheScanner(systemCache, allCategories)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"path with /* pattern", "/tmp/test/Caches/App1/file", true},
		{"path with /** pattern", "/tmp/test/Caches/App2/file", true},
		{"path with no trailing pattern", "/tmp/test/Caches/App3/file", true},
		{"path not in any category", "/tmp/test/Caches/App4/file", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.isExcluded(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}
