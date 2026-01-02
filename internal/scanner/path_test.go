package scanner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"mac-cleanup-go/pkg/types"
)

// --- Category Tests ---

func TestCategory_ReturnsConfiguredCategory(t *testing.T) {
	cat := types.Category{
		ID:     "test-id",
		Name:   "Test Name",
		Safety: types.SafetyLevelSafe,
	}

	s := NewPathScanner(cat)
	result := s.Category()

	assert.Equal(t, "test-id", result.ID)
	assert.Equal(t, "Test Name", result.Name)
	assert.Equal(t, types.SafetyLevelSafe, result.Safety)
}

// --- IsAvailable Tests ---

func TestIsAvailable_ReturnsTrue_WhenCheckCmdExists(t *testing.T) {
	cat := types.Category{
		ID:       "test",
		CheckCmd: "ls",
	}

	s := NewPathScanner(cat)

	assert.True(t, s.IsAvailable())
}

func TestIsAvailable_ReturnsFalse_WhenCheckCmdNotExists(t *testing.T) {
	cat := types.Category{
		ID:       "test",
		CheckCmd: "nonexistent-command-xyz-123",
	}

	s := NewPathScanner(cat)

	assert.False(t, s.IsAvailable())
}

func TestIsAvailable_ReturnsTrue_WhenCheckPathExists(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "path-scanner-test")
	defer os.RemoveAll(tmpDir)

	cat := types.Category{
		ID:    "test",
		Check: tmpDir,
	}

	s := NewPathScanner(cat)

	assert.True(t, s.IsAvailable())
}

func TestIsAvailable_ReturnsFalse_WhenCheckPathNotExists(t *testing.T) {
	cat := types.Category{
		ID:    "test",
		Check: "/nonexistent/path/xyz",
		Paths: []string{"/also/nonexistent/*"},
	}

	s := NewPathScanner(cat)

	assert.False(t, s.IsAvailable())
}

func TestIsAvailable_ReturnsTrue_WhenPathsHaveMatchingFiles(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "path-scanner-test")
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0o644)

	cat := types.Category{
		ID:    "test",
		Paths: []string{filepath.Join(tmpDir, "*")},
	}

	s := NewPathScanner(cat)

	assert.True(t, s.IsAvailable())
}

func TestIsAvailable_ReturnsTrue_WhenNoCheckAndNoPaths(t *testing.T) {
	cat := types.Category{
		ID: "test",
	}

	s := NewPathScanner(cat)

	assert.True(t, s.IsAvailable())
}

// --- Scan Tests ---

func TestScan_ReturnsEmptyResult_WhenNotAvailable(t *testing.T) {
	cat := types.Category{
		ID:       "test",
		CheckCmd: "nonexistent-command-xyz",
	}

	s := NewPathScanner(cat)
	result, err := s.Scan()

	assert.NoError(t, err)
	assert.Empty(t, result.Items)
}

func TestScan_ReturnsItems_ForMatchingPaths(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "path-scanner-test")
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("hello"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("world!"), 0o644)

	cat := types.Category{
		ID:    "test",
		Paths: []string{filepath.Join(tmpDir, "*.txt")},
	}

	s := NewPathScanner(cat)
	result, err := s.Scan()

	assert.NoError(t, err)
	assert.Len(t, result.Items, 2)
}

func TestScan_CalculatesTotalSizeCorrectly(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "path-scanner-test")
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("12345"), 0o644)      // 5 bytes
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("1234567890"), 0o644) // 10 bytes

	cat := types.Category{
		ID:    "test",
		Paths: []string{filepath.Join(tmpDir, "*.txt")},
	}

	s := NewPathScanner(cat)
	result, err := s.Scan()

	assert.NoError(t, err)
	assert.Equal(t, int64(15), result.TotalSize)
}

func TestScan_FiltersByDaysOld(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "path-scanner-test")
	defer os.RemoveAll(tmpDir)

	recentFile := filepath.Join(tmpDir, "recent.txt")
	os.WriteFile(recentFile, []byte("recent"), 0o644)

	oldFile := filepath.Join(tmpDir, "old.txt")
	os.WriteFile(oldFile, []byte("old"), 0o644)
	oldTime := time.Now().AddDate(0, 0, -10)
	os.Chtimes(oldFile, oldTime, oldTime)

	cat := types.Category{
		ID:      "test",
		Paths:   []string{filepath.Join(tmpDir, "*.txt")},
		DaysOld: 7,
	}

	s := NewPathScanner(cat)
	result, err := s.Scan()

	assert.NoError(t, err)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, "old.txt", result.Items[0].Name)
}

func TestScan_HandlesGlobErrors_Gracefully(t *testing.T) {
	cat := types.Category{
		ID:    "test",
		Paths: []string{"[invalid-glob"},
	}

	s := NewPathScanner(cat)
	result, err := s.Scan()

	assert.NoError(t, err)
	assert.Empty(t, result.Items)
}

func TestScan_IncludesCategoryInResult(t *testing.T) {
	cat := types.Category{
		ID:   "my-category",
		Name: "My Category",
	}

	s := NewPathScanner(cat)
	result, err := s.Scan()

	assert.NoError(t, err)
	assert.Equal(t, "my-category", result.Category.ID)
}

func TestScan_HandlesDirectories(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "path-scanner-test")
	defer os.RemoveAll(tmpDir)

	subDir := filepath.Join(tmpDir, "subdir")
	os.MkdirAll(subDir, 0o755)
	os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("content"), 0o644)

	cat := types.Category{
		ID:    "test",
		Paths: []string{filepath.Join(tmpDir, "*")},
	}

	s := NewPathScanner(cat)
	result, err := s.Scan()

	assert.NoError(t, err)
	assert.Len(t, result.Items, 1)
	assert.True(t, result.Items[0].IsDirectory)
	assert.Equal(t, "subdir", result.Items[0].Name)
}

func TestScan_HandlesMultiplePathPatterns(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "path-scanner-test")
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("txt"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "file.log"), []byte("log"), 0o644)

	cat := types.Category{
		ID: "test",
		Paths: []string{
			filepath.Join(tmpDir, "*.txt"),
			filepath.Join(tmpDir, "*.log"),
		},
	}

	s := NewPathScanner(cat)
	result, err := s.Scan()

	assert.NoError(t, err)
	assert.Len(t, result.Items, 2)
}

func TestScan_HandlesScanPathError_Gracefully(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "path-scanner-test")
	defer os.RemoveAll(tmpDir)

	brokenLink := filepath.Join(tmpDir, "broken.txt")
	os.Symlink("/nonexistent/path", brokenLink)

	cat := types.Category{
		ID:    "test",
		Paths: []string{filepath.Join(tmpDir, "*.txt")},
	}

	s := NewPathScanner(cat)
	result, err := s.Scan()

	assert.NoError(t, err)
	assert.Empty(t, result.Items)
}

// --- Clean Tests ---

func TestClean_DryRun_ReturnsCorrectStats(t *testing.T) {
	cat := types.Category{ID: "test", Name: "Test"}
	s := NewPathScanner(cat)

	items := []types.CleanableItem{
		{Path: "/fake/1", Size: 100},
		{Path: "/fake/2", Size: 200},
		{Path: "/fake/3", Size: 300},
	}

	result, err := s.Clean(items, true)

	assert.NoError(t, err)
	assert.Equal(t, 3, result.CleanedItems)
	assert.Equal(t, int64(600), result.FreedSpace)
}

func TestClean_DryRun_EmptyItems(t *testing.T) {
	cat := types.Category{ID: "test"}
	s := NewPathScanner(cat)

	result, err := s.Clean([]types.CleanableItem{}, true)

	assert.NoError(t, err)
	assert.Equal(t, 0, result.CleanedItems)
	assert.Equal(t, int64(0), result.FreedSpace)
}

func TestClean_NonDryRun_ReturnsEmptyResult(t *testing.T) {
	cat := types.Category{ID: "test"}
	s := NewPathScanner(cat)

	items := []types.CleanableItem{
		{Path: "/fake/1", Size: 100},
	}

	result, err := s.Clean(items, false)

	assert.NoError(t, err)
	assert.Equal(t, 0, result.CleanedItems)
}

func TestClean_IncludesCategoryInResult(t *testing.T) {
	cat := types.Category{ID: "my-cat", Name: "My Category"}
	s := NewPathScanner(cat)

	result, err := s.Clean(nil, true)

	assert.NoError(t, err)
	assert.Equal(t, "my-cat", result.Category.ID)
}
