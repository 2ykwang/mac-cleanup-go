package target

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
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
		Paths: []string{tmpDir + "/*"},
	}

	// Create a file so glob matches
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0o644)

	s := NewPathScanner(cat)

	assert.True(t, s.IsAvailable())
}

func TestIsAvailable_ReturnsFalse_WhenPathsNotExists(t *testing.T) {
	cat := types.Category{
		ID:    "test",
		Paths: []string{"/nonexistent/path/xyz/*"},
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

func TestClean_ReturnsEmptyResult(t *testing.T) {
	cat := types.Category{ID: "test"}
	s := NewPathScanner(cat)

	items := []types.CleanableItem{
		{Path: "/fake/1", Size: 100},
	}

	result, err := s.Clean(items)

	assert.NoError(t, err)
	assert.Equal(t, 0, result.CleanedItems)
}

func TestClean_IncludesCategoryInResult(t *testing.T) {
	cat := types.Category{ID: "my-cat", Name: "My Category"}
	s := NewPathScanner(cat)

	result, err := s.Clean(nil)

	assert.NoError(t, err)
	assert.Equal(t, "my-cat", result.Category.ID)
}

// --- SIP Filtering Tests ---
func TestScan_ExcludesSIPProtectedPaths(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "path-scanner-test")
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "regular.txt"), []byte("test"), 0o644)

	cat := types.Category{
		ID:    "test",
		Paths: []string{filepath.Join(tmpDir, "*")},
	}

	s := NewPathScanner(cat)
	result, err := s.Scan()

	assert.NoError(t, err)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, "regular.txt", result.Items[0].Name)
}
