package scanner

import (
	"os"
	"path/filepath"
	"testing"

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

	if !s.isExcluded("/tmp/test/Caches/Arc/cache.db") {
		t.Error("Expected path in other category to be excluded")
	}
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

	if s.isExcluded("/tmp/test/Caches/RandomApp/data") {
		t.Error("Expected path not in any category to not be excluded")
	}
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

	if s.isExcluded("") {
		t.Error("Expected empty path to not be excluded")
	}
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

	if len(s.excludePaths) != 3 {
		t.Errorf("Expected 3 exclude paths, got %d", len(s.excludePaths))
	}
}

func TestNewSystemCacheScanner_DoesNotIncludeOwnPaths(t *testing.T) {
	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{"/tmp/test/Caches/*"},
	}
	allCategories := []types.Category{systemCache}
	s := NewSystemCacheScanner(systemCache, allCategories)

	if len(s.excludePaths) != 0 {
		t.Errorf("Expected 0 exclude paths when only self exists, got %d", len(s.excludePaths))
	}
}

func TestNewSystemCacheScanner_WhenNoCategoriesProvided_CreatesEmptyExcludes(t *testing.T) {
	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{"/tmp/test/Caches/*"},
	}
	s := NewSystemCacheScanner(systemCache, nil)

	if s.excludePaths == nil {
		// excludePaths should be initialized, not nil
		t.Log("excludePaths is nil, but empty slice would be preferred")
	}
	if s.isExcluded("/tmp/test/Caches/AnyApp/file") {
		t.Error("Expected no exclusions when no categories provided")
	}
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

	if !s.isExcluded("/tmp/test/Caches/App/sub/deep/file") {
		t.Error("Expected deeply nested path to be excluded")
	}
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

	// "AppOther" starts with "App" but should NOT be excluded
	if s.isExcluded("/tmp/test/Caches/AppOther/data") {
		t.Error("Expected path with similar prefix but different directory to not be excluded")
	}
}

func TestScan_ExcludesPathsFromOtherCategories(t *testing.T) {
	// Arrange: Create temp directory structure
	tmpDir, err := os.MkdirTemp("", "systemcache-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cachesDir := filepath.Join(tmpDir, "Caches")
	arcDir := filepath.Join(cachesDir, "Arc")
	randomDir := filepath.Join(cachesDir, "RandomApp")
	jetbrainsDir := filepath.Join(cachesDir, "JetBrains")

	for _, dir := range []string{arcDir, randomDir, jetbrainsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "cache.dat"), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
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

	// Act
	result, err := s.Scan()
	// Assert
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("Expected 1 item (RandomApp only), got %d", len(result.Items))
		for _, item := range result.Items {
			t.Logf("  - %s", item.Path)
		}
		return
	}
	if result.Items[0].Name != "RandomApp" {
		t.Errorf("Expected RandomApp, got %s", result.Items[0].Name)
	}
}

func TestScan_WhenNoMatchingPaths_ReturnsEmptyResult(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "systemcache-empty-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	systemCache := types.Category{
		ID:    "system-cache",
		Paths: []string{filepath.Join(tmpDir, "NonExistent", "*")},
	}
	s := NewSystemCacheScanner(systemCache, []types.Category{systemCache})

	result, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(result.Items))
	}
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
			if result != tt.expected {
				t.Errorf("isExcluded(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}
