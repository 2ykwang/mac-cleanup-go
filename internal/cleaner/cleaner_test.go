package cleaner

import (
	"os"
	"path/filepath"
	"testing"

	"mac-cleanup-go/pkg/types"
)

func TestNew(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatal("New() returned nil")
	}
}

func TestClean_DryRun(t *testing.T) {
	c := New()

	items := []types.CleanableItem{
		{Path: "/fake/path1", Name: "file1", Size: 1000},
		{Path: "/fake/path2", Name: "file2", Size: 2000},
		{Path: "/fake/path3", Name: "file3", Size: 3000},
	}

	cat := types.Category{
		ID:     "test",
		Name:   "Test Category",
		Method: types.MethodTrash,
	}

	result := c.Clean(cat, items, true) // dry-run

	if result.CleanedItems != 3 {
		t.Errorf("Expected 3 cleaned items, got %d", result.CleanedItems)
	}

	if result.FreedSpace != 6000 {
		t.Errorf("Expected 6000 bytes freed, got %d", result.FreedSpace)
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %v", result.Errors)
	}
}

func TestClean_DryRun_EmptyItems(t *testing.T) {
	c := New()

	cat := types.Category{
		ID:     "test",
		Name:   "Test Category",
		Method: types.MethodTrash,
	}

	result := c.Clean(cat, []types.CleanableItem{}, true)

	if result.CleanedItems != 0 {
		t.Errorf("Expected 0 cleaned items, got %d", result.CleanedItems)
	}

	if result.FreedSpace != 0 {
		t.Errorf("Expected 0 bytes freed, got %d", result.FreedSpace)
	}
}

func TestClean_PermanentDelete_File(t *testing.T) {
	c := New()

	// Create temp file
	tmpFile, err := os.CreateTemp("", "cleaner-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.WriteString("test content")
	tmpFile.Close()

	// Get file info for size
	info, _ := os.Stat(tmpPath)

	items := []types.CleanableItem{
		{
			Path:        tmpPath,
			Name:        filepath.Base(tmpPath),
			Size:        info.Size(),
			IsDirectory: false,
		},
	}

	cat := types.Category{
		ID:     "test",
		Name:   "Test Category",
		Method: types.MethodPermanent,
	}

	result := c.Clean(cat, items, false)

	if result.CleanedItems != 1 {
		t.Errorf("Expected 1 cleaned item, got %d", result.CleanedItems)
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %v", result.Errors)
	}

	// Verify file is deleted
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("File should have been deleted")
		os.Remove(tmpPath) // cleanup
	}
}

func TestClean_PermanentDelete_Directory(t *testing.T) {
	c := New()

	// Create temp directory with files
	tmpDir, err := os.MkdirTemp("", "cleaner-test-dir-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create a file inside
	tmpFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(tmpFile, []byte("test"), 0644)

	items := []types.CleanableItem{
		{
			Path:        tmpDir,
			Name:        filepath.Base(tmpDir),
			Size:        1000,
			IsDirectory: true,
		},
	}

	cat := types.Category{
		ID:     "test",
		Name:   "Test Category",
		Method: types.MethodPermanent,
	}

	result := c.Clean(cat, items, false)

	if result.CleanedItems != 1 {
		t.Errorf("Expected 1 cleaned item, got %d", result.CleanedItems)
	}

	// Verify directory is deleted
	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		t.Error("Directory should have been deleted")
		os.RemoveAll(tmpDir) // cleanup
	}
}

func TestClean_PermanentDelete_NonExistent(t *testing.T) {
	c := New()

	items := []types.CleanableItem{
		{
			Path:        "/nonexistent/path/file.txt",
			Name:        "file.txt",
			Size:        1000,
			IsDirectory: false,
		},
	}

	cat := types.Category{
		ID:     "test",
		Name:   "Test Category",
		Method: types.MethodPermanent,
	}

	result := c.Clean(cat, items, false)

	if result.CleanedItems != 0 {
		t.Errorf("Expected 0 cleaned items, got %d", result.CleanedItems)
	}

	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}
}

func TestClean_Command_Success(t *testing.T) {
	c := New()

	cat := types.Category{
		ID:      "test",
		Name:    "Test Category",
		Method:  types.MethodCommand,
		Command: "echo hello",
	}

	result := c.Clean(cat, nil, false)

	if result.CleanedItems != 1 {
		t.Errorf("Expected 1 cleaned item, got %d", result.CleanedItems)
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %v", result.Errors)
	}
}

func TestClean_Command_Failure(t *testing.T) {
	c := New()

	cat := types.Category{
		ID:      "test",
		Name:    "Test Category",
		Method:  types.MethodCommand,
		Command: "exit 1",
	}

	result := c.Clean(cat, nil, false)

	if result.CleanedItems != 0 {
		t.Errorf("Expected 0 cleaned items, got %d", result.CleanedItems)
	}

	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}
}

func TestClean_Command_Empty(t *testing.T) {
	c := New()

	cat := types.Category{
		ID:      "test",
		Name:    "Test Category",
		Method:  types.MethodCommand,
		Command: "",
	}

	result := c.Clean(cat, nil, false)

	if result.CleanedItems != 0 {
		t.Errorf("Expected 0 cleaned items, got %d", result.CleanedItems)
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %v", result.Errors)
	}
}

func TestClean_CategoryInResult(t *testing.T) {
	c := New()

	cat := types.Category{
		ID:     "test-id",
		Name:   "Test Category",
		Method: types.MethodTrash,
		Safety: types.SafetyLevelSafe,
	}

	result := c.Clean(cat, []types.CleanableItem{}, true)

	if result.Category.ID != "test-id" {
		t.Errorf("Expected category ID 'test-id', got '%s'", result.Category.ID)
	}

	if result.Category.Name != "Test Category" {
		t.Errorf("Expected category name 'Test Category', got '%s'", result.Category.Name)
	}
}
