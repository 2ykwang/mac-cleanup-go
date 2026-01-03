package cleaner

import (
	"os"
	"testing"

	"mac-cleanup-go/pkg/types"
)

func TestNew(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatal("New() returned nil")
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

	result := c.Clean(cat, nil)

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

	result := c.Clean(cat, nil)

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

	result := c.Clean(cat, nil)

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
		Method: types.MethodCommand,
		Safety: types.SafetyLevelSafe,
	}

	result := c.Clean(cat, []types.CleanableItem{})

	if result.Category.ID != "test-id" {
		t.Errorf("Expected category ID 'test-id', got '%s'", result.Category.ID)
	}

	if result.Category.Name != "Test Category" {
		t.Errorf("Expected category name 'Test Category', got '%s'", result.Category.Name)
	}
}

func TestClean_SkipsSIPProtectedPaths(t *testing.T) {
	c := New()

	items := []types.CleanableItem{
		{Path: "/System/Library/Caches/test", Name: "sip-protected", Size: 1000},
		{Path: "/bin/test", Name: "bin-protected", Size: 2000},
	}

	cat := types.Category{
		ID:     "test",
		Name:   "Test Category",
		Method: types.MethodTrash,
	}

	result := c.Clean(cat, items)

	// Both should be skipped (SIP protected)
	if result.CleanedItems != 0 {
		t.Errorf("Expected 0 cleaned items, got %d", result.CleanedItems)
	}

	if result.SkippedItems != 2 {
		t.Errorf("Expected 2 skipped items, got %d", result.SkippedItems)
	}
}

func TestClean_Permanent_RemovesFiles(t *testing.T) {
	c := New()

	// Create a temp file to test permanent deletion
	tmpFile, err := os.CreateTemp("", "cleanup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.WriteString("test content")
	tmpFile.Close()

	items := []types.CleanableItem{
		{Path: tmpPath, Name: "test-file", Size: 12},
	}

	cat := types.Category{
		ID:     "test",
		Name:   "Test Category",
		Method: types.MethodPermanent,
	}

	result := c.Clean(cat, items)

	if result.CleanedItems != 1 {
		t.Errorf("Expected 1 cleaned item, got %d", result.CleanedItems)
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %v", result.Errors)
	}

	// Verify file is deleted
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("Expected file to be deleted, but it still exists")
		os.Remove(tmpPath) // cleanup
	}
}

func TestClean_Permanent_RemovesDirectories(t *testing.T) {
	c := New()

	// Create a temp directory with files
	tmpDir, err := os.MkdirTemp("", "cleanup-test-dir-*")
	if err != nil {
		t.Fatal(err)
	}

	// Create a file inside
	tmpFile, err := os.CreateTemp(tmpDir, "file-*")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	items := []types.CleanableItem{
		{Path: tmpDir, Name: "test-dir", Size: 100, IsDirectory: true},
	}

	cat := types.Category{
		ID:     "test",
		Name:   "Test Category",
		Method: types.MethodPermanent,
	}

	result := c.Clean(cat, items)

	if result.CleanedItems != 1 {
		t.Errorf("Expected 1 cleaned item, got %d", result.CleanedItems)
	}

	// Verify directory is deleted
	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		t.Errorf("Expected directory to be deleted, but it still exists")
		os.RemoveAll(tmpDir) // cleanup
	}
}

func TestClean_Permanent_SkipsSIPProtectedPaths(t *testing.T) {
	c := New()

	items := []types.CleanableItem{
		{Path: "/System/Library/Caches/test", Name: "sip-protected", Size: 1000},
	}

	cat := types.Category{
		ID:     "test",
		Name:   "Test Category",
		Method: types.MethodPermanent,
	}

	result := c.Clean(cat, items)

	if result.SkippedItems != 1 {
		t.Errorf("Expected 1 skipped item, got %d", result.SkippedItems)
	}
}

func TestClean_Builtin_ReturnsNoActionResult(t *testing.T) {
	c := New()

	cat := types.Category{
		ID:     "docker",
		Name:   "Docker",
		Method: types.MethodBuiltin,
	}

	result := c.Clean(cat, nil)

	if result.CleanedItems != 0 {
		t.Errorf("Expected 0 cleaned items for builtin, got %d", result.CleanedItems)
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %v", result.Errors)
	}
}

func TestClean_Manual_SkipsWithGuide(t *testing.T) {
	c := New()

	cat := types.Category{
		ID:     "test-manual",
		Name:   "Manual Task",
		Method: types.MethodManual,
		Guide:  "Open Finder and delete manually",
	}

	items := []types.CleanableItem{
		{Path: "/some/path", Name: "manual-item", Size: 1000},
	}

	result := c.Clean(cat, items)

	// Manual methods should skip all items
	if result.CleanedItems != 0 {
		t.Errorf("Expected 0 cleaned items for manual, got %d", result.CleanedItems)
	}

	if result.SkippedItems != 1 {
		t.Errorf("Expected 1 skipped item for manual, got %d", result.SkippedItems)
	}
}
