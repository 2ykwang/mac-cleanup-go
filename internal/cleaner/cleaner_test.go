package cleaner

import (
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

func TestClean_DryRun_SkipsSIPProtectedPaths(t *testing.T) {
	c := New()

	items := []types.CleanableItem{
		{Path: "/System/Library/Caches/test", Name: "sip-protected", Size: 1000},
		{Path: "/usr/bin/test", Name: "usr-protected", Size: 2000},
		{Path: "/Users/test/Library/Caches", Name: "normal", Size: 3000},
	}

	cat := types.Category{
		ID:     "test",
		Name:   "Test Category",
		Method: types.MethodTrash,
	}

	result := c.Clean(cat, items, true) // dry-run

	// Only the non-SIP path should be counted
	if result.CleanedItems != 1 {
		t.Errorf("Expected 1 cleaned item, got %d", result.CleanedItems)
	}

	if result.SkippedItems != 2 {
		t.Errorf("Expected 2 skipped items, got %d", result.SkippedItems)
	}

	if result.FreedSpace != 3000 {
		t.Errorf("Expected 3000 bytes freed, got %d", result.FreedSpace)
	}
}

func TestClean_RealRun_SkipsSIPProtectedPaths(t *testing.T) {
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

	result := c.Clean(cat, items, false) // real run

	// Both should be skipped (SIP protected)
	if result.CleanedItems != 0 {
		t.Errorf("Expected 0 cleaned items, got %d", result.CleanedItems)
	}

	if result.SkippedItems != 2 {
		t.Errorf("Expected 2 skipped items, got %d", result.SkippedItems)
	}
}
