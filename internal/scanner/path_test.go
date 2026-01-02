package scanner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

	if result.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", result.ID)
	}
	if result.Name != "Test Name" {
		t.Errorf("Expected Name 'Test Name', got '%s'", result.Name)
	}
	if result.Safety != types.SafetyLevelSafe {
		t.Errorf("Expected Safety 'safe', got '%s'", result.Safety)
	}
}

// --- IsAvailable Tests ---

func TestIsAvailable_ReturnsTrue_WhenCheckCmdExists(t *testing.T) {
	cat := types.Category{
		ID:       "test",
		CheckCmd: "ls", // ls command exists on all Unix systems
	}
	s := NewPathScanner(cat)

	if !s.IsAvailable() {
		t.Error("Expected true when CheckCmd exists")
	}
}

func TestIsAvailable_ReturnsFalse_WhenCheckCmdNotExists(t *testing.T) {
	cat := types.Category{
		ID:       "test",
		CheckCmd: "nonexistent-command-xyz-123",
	}
	s := NewPathScanner(cat)

	if s.IsAvailable() {
		t.Error("Expected false when CheckCmd doesn't exist")
	}
}

func TestIsAvailable_ReturnsTrue_WhenCheckPathExists(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "base-scanner-test")
	defer os.RemoveAll(tmpDir)

	cat := types.Category{
		ID:    "test",
		Check: tmpDir,
	}
	s := NewPathScanner(cat)

	if !s.IsAvailable() {
		t.Error("Expected true when Check path exists")
	}
}

func TestIsAvailable_ReturnsFalse_WhenCheckPathNotExists(t *testing.T) {
	cat := types.Category{
		ID:    "test",
		Check: "/nonexistent/path/xyz",
		Paths: []string{"/also/nonexistent/*"},
	}
	s := NewPathScanner(cat)

	if s.IsAvailable() {
		t.Error("Expected false when Check path doesn't exist and no paths match")
	}
}

func TestIsAvailable_ReturnsTrue_WhenPathsHaveMatchingFiles(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "base-scanner-test")
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	cat := types.Category{
		ID:    "test",
		Paths: []string{filepath.Join(tmpDir, "*")},
	}
	s := NewPathScanner(cat)

	if !s.IsAvailable() {
		t.Error("Expected true when paths have matching files")
	}
}

func TestIsAvailable_ReturnsTrue_WhenNoCheckAndNoPaths(t *testing.T) {
	cat := types.Category{
		ID: "test",
		// No Check, no CheckCmd, no Paths
	}
	s := NewPathScanner(cat)

	if !s.IsAvailable() {
		t.Error("Expected true when no check specified and no paths")
	}
}

// --- Scan Tests ---

func TestScan_ReturnsEmptyResult_WhenNotAvailable(t *testing.T) {
	cat := types.Category{
		ID:       "test",
		CheckCmd: "nonexistent-command-xyz",
	}
	s := NewPathScanner(cat)

	result, err := s.Scan()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(result.Items))
	}
}

func TestScan_ReturnsItems_ForMatchingPaths(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "base-scanner-test")
	defer os.RemoveAll(tmpDir)

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("world!"), 0644)

	cat := types.Category{
		ID:    "test",
		Paths: []string{filepath.Join(tmpDir, "*.txt")},
	}
	s := NewPathScanner(cat)

	result, err := s.Scan()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result.Items))
	}
}

func TestScan_CalculatesTotalSizeCorrectly(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "base-scanner-test")
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("12345"), 0644)      // 5 bytes
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("1234567890"), 0644) // 10 bytes

	cat := types.Category{
		ID:    "test",
		Paths: []string{filepath.Join(tmpDir, "*.txt")},
	}
	s := NewPathScanner(cat)

	result, err := s.Scan()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.TotalSize != 15 {
		t.Errorf("Expected TotalSize 15, got %d", result.TotalSize)
	}
}

func TestScan_FiltersByDaysOld(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "base-scanner-test")
	defer os.RemoveAll(tmpDir)

	// Create a file with recent modification time
	recentFile := filepath.Join(tmpDir, "recent.txt")
	os.WriteFile(recentFile, []byte("recent"), 0644)

	// Create a file with old modification time
	oldFile := filepath.Join(tmpDir, "old.txt")
	os.WriteFile(oldFile, []byte("old"), 0644)
	oldTime := time.Now().AddDate(0, 0, -10) // 10 days ago
	os.Chtimes(oldFile, oldTime, oldTime)

	cat := types.Category{
		ID:      "test",
		Paths:   []string{filepath.Join(tmpDir, "*.txt")},
		DaysOld: 7, // Only files older than 7 days
	}
	s := NewPathScanner(cat)

	result, err := s.Scan()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("Expected 1 item (only old file), got %d", len(result.Items))
	}
	if len(result.Items) > 0 && result.Items[0].Name != "old.txt" {
		t.Errorf("Expected 'old.txt', got '%s'", result.Items[0].Name)
	}
}

func TestScan_HandlesGlobErrors_Gracefully(t *testing.T) {
	cat := types.Category{
		ID:    "test",
		Paths: []string{"[invalid-glob"}, // Invalid glob pattern
	}
	s := NewPathScanner(cat)

	result, err := s.Scan()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("Expected 0 items for invalid glob, got %d", len(result.Items))
	}
}

func TestScan_IncludesCategoryInResult(t *testing.T) {
	cat := types.Category{
		ID:   "my-category",
		Name: "My Category",
	}
	s := NewPathScanner(cat)

	result, err := s.Scan()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.Category.ID != "my-category" {
		t.Errorf("Expected category ID 'my-category', got '%s'", result.Category.ID)
	}
}

func TestScan_HandlesDirectories(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "base-scanner-test")
	defer os.RemoveAll(tmpDir)

	// Create a subdirectory with files
	subDir := filepath.Join(tmpDir, "subdir")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("content"), 0644)

	cat := types.Category{
		ID:    "test",
		Paths: []string{filepath.Join(tmpDir, "*")},
	}
	s := NewPathScanner(cat)

	result, err := s.Scan()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(result.Items))
	}
	if !result.Items[0].IsDirectory {
		t.Error("Expected item to be a directory")
	}
	if result.Items[0].Name != "subdir" {
		t.Errorf("Expected name 'subdir', got '%s'", result.Items[0].Name)
	}
}

func TestScan_HandlesMultiplePathPatterns(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "base-scanner-test")
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("txt"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file.log"), []byte("log"), 0644)

	cat := types.Category{
		ID: "test",
		Paths: []string{
			filepath.Join(tmpDir, "*.txt"),
			filepath.Join(tmpDir, "*.log"),
		},
	}
	s := NewPathScanner(cat)

	result, err := s.Scan()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("Expected 2 items from multiple patterns, got %d", len(result.Items))
	}
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

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.CleanedItems != 3 {
		t.Errorf("Expected CleanedItems 3, got %d", result.CleanedItems)
	}
	if result.FreedSpace != 600 {
		t.Errorf("Expected FreedSpace 600, got %d", result.FreedSpace)
	}
}

func TestClean_DryRun_EmptyItems(t *testing.T) {
	cat := types.Category{ID: "test"}
	s := NewPathScanner(cat)

	result, err := s.Clean([]types.CleanableItem{}, true)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.CleanedItems != 0 {
		t.Errorf("Expected CleanedItems 0, got %d", result.CleanedItems)
	}
	if result.FreedSpace != 0 {
		t.Errorf("Expected FreedSpace 0, got %d", result.FreedSpace)
	}
}

func TestClean_NonDryRun_ReturnsEmptyResult(t *testing.T) {
	cat := types.Category{ID: "test"}
	s := NewPathScanner(cat)

	items := []types.CleanableItem{
		{Path: "/fake/1", Size: 100},
	}

	result, err := s.Clean(items, false)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Current implementation doesn't actually delete
	if result.CleanedItems != 0 {
		t.Errorf("Expected CleanedItems 0 for non-dry-run, got %d", result.CleanedItems)
	}
}

func TestClean_IncludesCategoryInResult(t *testing.T) {
	cat := types.Category{ID: "my-cat", Name: "My Category"}
	s := NewPathScanner(cat)

	result, err := s.Clean(nil, true)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.Category.ID != "my-cat" {
		t.Errorf("Expected category ID 'my-cat', got '%s'", result.Category.ID)
	}
}

func TestScan_HandlesScanPathError_Gracefully(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "base-scanner-test")
	defer os.RemoveAll(tmpDir)

	// Create a file that will be deleted before scanPath
	testFile := filepath.Join(tmpDir, "disappearing.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	cat := types.Category{
		ID:    "test",
		Paths: []string{filepath.Join(tmpDir, "*.txt")},
	}
	s := NewPathScanner(cat)

	// Delete the file after glob would find it but before stat
	// We simulate this by creating a symlink to a non-existent file
	os.Remove(testFile)
	brokenLink := filepath.Join(tmpDir, "broken.txt")
	os.Symlink("/nonexistent/path", brokenLink)

	result, err := s.Scan()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	// Should handle the error gracefully and return empty result
	if len(result.Items) != 0 {
		t.Errorf("Expected 0 items when scanPath fails, got %d", len(result.Items))
	}
}
