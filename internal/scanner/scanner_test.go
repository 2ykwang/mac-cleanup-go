package scanner

import (
	"testing"

	"mac-cleanup-go/pkg/types"
)

// mockScanner implements Scanner interface for testing
type mockScanner struct {
	cat       types.Category
	available bool
}

func newMockScanner(id string, available bool) *mockScanner {
	return &mockScanner{
		cat:       types.Category{ID: id, Name: "Mock " + id},
		available: available,
	}
}

func (m *mockScanner) Scan() (*types.ScanResult, error) {
	return &types.ScanResult{Category: m.cat}, nil
}

func (m *mockScanner) Clean(items []types.CleanableItem, dryRun bool) (*types.CleanResult, error) {
	return &types.CleanResult{Category: m.cat}, nil
}

func (m *mockScanner) Category() types.Category {
	return m.cat
}

func (m *mockScanner) IsAvailable() bool {
	return m.available
}

// --- NewRegistry Tests ---

func TestNewRegistry_ReturnsNonNil(t *testing.T) {
	r := NewRegistry()

	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}
}

func TestNewRegistry_HasEmptyScannersMap(t *testing.T) {
	r := NewRegistry()

	if len(r.scanners) != 0 {
		t.Errorf("Expected empty scanners map, got %d items", len(r.scanners))
	}
}

// --- Register Tests ---

func TestRegister_AddsScannerToRegistry(t *testing.T) {
	r := NewRegistry()
	scanner := newMockScanner("test-id", true)

	r.Register(scanner)

	if len(r.scanners) != 1 {
		t.Errorf("Expected 1 scanner, got %d", len(r.scanners))
	}
}

func TestRegister_UsesCategoryIDAsKey(t *testing.T) {
	r := NewRegistry()
	scanner := newMockScanner("my-scanner", true)

	r.Register(scanner)

	if _, exists := r.scanners["my-scanner"]; !exists {
		t.Error("Scanner not found with category ID as key")
	}
}

func TestRegister_OverwritesExistingScanner(t *testing.T) {
	r := NewRegistry()
	scanner1 := newMockScanner("same-id", true)
	scanner2 := newMockScanner("same-id", false)

	r.Register(scanner1)
	r.Register(scanner2)

	if len(r.scanners) != 1 {
		t.Errorf("Expected 1 scanner after overwrite, got %d", len(r.scanners))
	}
	// Verify it's the second scanner (available=false)
	if r.scanners["same-id"].IsAvailable() {
		t.Error("Expected second scanner to overwrite first")
	}
}

func TestRegister_MultipleScannersWithDifferentIDs(t *testing.T) {
	r := NewRegistry()

	r.Register(newMockScanner("scanner-1", true))
	r.Register(newMockScanner("scanner-2", true))
	r.Register(newMockScanner("scanner-3", true))

	if len(r.scanners) != 3 {
		t.Errorf("Expected 3 scanners, got %d", len(r.scanners))
	}
}

// --- Get Tests ---

func TestGet_ReturnsScanner_WhenExists(t *testing.T) {
	r := NewRegistry()
	scanner := newMockScanner("existing", true)
	r.Register(scanner)

	result, ok := r.Get("existing")

	if !ok {
		t.Error("Expected ok=true for existing scanner")
	}
	if result == nil {
		t.Error("Expected non-nil scanner")
	}
	if result.Category().ID != "existing" {
		t.Errorf("Expected ID 'existing', got '%s'", result.Category().ID)
	}
}

func TestGet_ReturnsNilAndFalse_WhenNotExists(t *testing.T) {
	r := NewRegistry()

	result, ok := r.Get("non-existent")

	if ok {
		t.Error("Expected ok=false for non-existent scanner")
	}
	if result != nil {
		t.Error("Expected nil scanner for non-existent ID")
	}
}

func TestGet_ReturnsCorrectScanner_WhenMultipleRegistered(t *testing.T) {
	r := NewRegistry()
	r.Register(newMockScanner("first", true))
	r.Register(newMockScanner("second", false))
	r.Register(newMockScanner("third", true))

	result, ok := r.Get("second")

	if !ok {
		t.Fatal("Expected ok=true")
	}
	if result.Category().ID != "second" {
		t.Errorf("Expected 'second', got '%s'", result.Category().ID)
	}
	if result.IsAvailable() {
		t.Error("Expected IsAvailable()=false for 'second'")
	}
}

// --- All Tests ---

func TestAll_ReturnsEmptySlice_WhenNoScanners(t *testing.T) {
	r := NewRegistry()

	result := r.All()

	if result == nil {
		t.Error("Expected non-nil slice")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty slice, got %d items", len(result))
	}
}

func TestAll_ReturnsAllRegisteredScanners(t *testing.T) {
	r := NewRegistry()
	r.Register(newMockScanner("a", true))
	r.Register(newMockScanner("b", false))
	r.Register(newMockScanner("c", true))

	result := r.All()

	if len(result) != 3 {
		t.Errorf("Expected 3 scanners, got %d", len(result))
	}
}

func TestAll_IncludesBothAvailableAndUnavailable(t *testing.T) {
	r := NewRegistry()
	r.Register(newMockScanner("available", true))
	r.Register(newMockScanner("unavailable", false))

	result := r.All()

	if len(result) != 2 {
		t.Errorf("Expected 2 scanners (both available and unavailable), got %d", len(result))
	}
}

// --- Available Tests ---

func TestAvailable_ReturnsEmptySlice_WhenNoScanners(t *testing.T) {
	r := NewRegistry()

	result := r.Available()

	if result == nil {
		t.Error("Expected non-nil slice")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty slice, got %d items", len(result))
	}
}

func TestAvailable_ReturnsOnlyAvailableScanners(t *testing.T) {
	r := NewRegistry()
	r.Register(newMockScanner("available-1", true))
	r.Register(newMockScanner("unavailable", false))
	r.Register(newMockScanner("available-2", true))

	result := r.Available()

	if len(result) != 2 {
		t.Errorf("Expected 2 available scanners, got %d", len(result))
	}
	for _, s := range result {
		if !s.IsAvailable() {
			t.Errorf("Scanner '%s' should be available", s.Category().ID)
		}
	}
}

func TestAvailable_ReturnsEmptySlice_WhenAllUnavailable(t *testing.T) {
	r := NewRegistry()
	r.Register(newMockScanner("unavailable-1", false))
	r.Register(newMockScanner("unavailable-2", false))

	result := r.Available()

	if len(result) != 0 {
		t.Errorf("Expected 0 available scanners, got %d", len(result))
	}
}

func TestAvailable_ReturnsAllScanners_WhenAllAvailable(t *testing.T) {
	r := NewRegistry()
	r.Register(newMockScanner("available-1", true))
	r.Register(newMockScanner("available-2", true))

	result := r.Available()

	if len(result) != 2 {
		t.Errorf("Expected 2 available scanners, got %d", len(result))
	}
}
