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
