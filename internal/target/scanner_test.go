package target

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

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

func (m *mockScanner) Clean(_ []types.CleanableItem) (*types.CleanResult, error) {
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

	assert.NotNil(t, r)
}

func TestNewRegistry_HasEmptyScannersMap(t *testing.T) {
	r := NewRegistry()

	assert.Empty(t, r.scanners)
}

// --- Register Tests ---

func TestRegister_AddsScannerToRegistry(t *testing.T) {
	r := NewRegistry()
	scanner := newMockScanner("test-id", true)

	r.Register(scanner)

	assert.Len(t, r.scanners, 1)
}

func TestRegister_UsesCategoryIDAsKey(t *testing.T) {
	r := NewRegistry()
	scanner := newMockScanner("my-scanner", true)

	r.Register(scanner)

	_, exists := r.scanners["my-scanner"]
	assert.True(t, exists)
}

func TestRegister_OverwritesExistingScanner(t *testing.T) {
	r := NewRegistry()
	scanner1 := newMockScanner("same-id", true)
	scanner2 := newMockScanner("same-id", false)

	r.Register(scanner1)
	r.Register(scanner2)

	assert.Len(t, r.scanners, 1)
	assert.False(t, r.scanners["same-id"].IsAvailable())
}

func TestRegister_MultipleScannersWithDifferentIDs(t *testing.T) {
	r := NewRegistry()

	r.Register(newMockScanner("scanner-1", true))
	r.Register(newMockScanner("scanner-2", true))
	r.Register(newMockScanner("scanner-3", true))

	assert.Len(t, r.scanners, 3)
}

// --- Get Tests ---

func TestGet_ReturnsScanner_WhenExists(t *testing.T) {
	r := NewRegistry()
	scanner := newMockScanner("existing", true)
	r.Register(scanner)

	result, ok := r.Get("existing")

	assert.True(t, ok)
	assert.NotNil(t, result)
	assert.Equal(t, "existing", result.Category().ID)
}

func TestGet_ReturnsNilAndFalse_WhenNotExists(t *testing.T) {
	r := NewRegistry()

	result, ok := r.Get("non-existent")

	assert.False(t, ok)
	assert.Nil(t, result)
}

func TestGet_ReturnsCorrectScanner_WhenMultipleRegistered(t *testing.T) {
	r := NewRegistry()
	r.Register(newMockScanner("first", true))
	r.Register(newMockScanner("second", false))
	r.Register(newMockScanner("third", true))

	result, ok := r.Get("second")

	assert.True(t, ok)
	assert.Equal(t, "second", result.Category().ID)
	assert.False(t, result.IsAvailable())
}

// --- All Tests ---

func TestAll_ReturnsEmptySlice_WhenNoScanners(t *testing.T) {
	r := NewRegistry()

	result := r.All()

	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestAll_ReturnsAllRegisteredScanners(t *testing.T) {
	r := NewRegistry()
	r.Register(newMockScanner("a", true))
	r.Register(newMockScanner("b", false))
	r.Register(newMockScanner("c", true))

	result := r.All()

	assert.Len(t, result, 3)
}

func TestAll_IncludesBothAvailableAndUnavailable(t *testing.T) {
	r := NewRegistry()
	r.Register(newMockScanner("available", true))
	r.Register(newMockScanner("unavailable", false))

	result := r.All()

	assert.Len(t, result, 2)
}

// --- Available Tests ---

func TestAvailable_ReturnsEmptySlice_WhenNoScanners(t *testing.T) {
	r := NewRegistry()

	result := r.Available()

	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestAvailable_ReturnsOnlyAvailableScanners(t *testing.T) {
	r := NewRegistry()
	r.Register(newMockScanner("available-1", true))
	r.Register(newMockScanner("unavailable", false))
	r.Register(newMockScanner("available-2", true))

	result := r.Available()

	assert.Len(t, result, 2)
	for _, s := range result {
		assert.True(t, s.IsAvailable())
	}
}

func TestAvailable_ReturnsEmptySlice_WhenAllUnavailable(t *testing.T) {
	r := NewRegistry()
	r.Register(newMockScanner("unavailable-1", false))
	r.Register(newMockScanner("unavailable-2", false))

	result := r.Available()

	assert.Empty(t, result)
}

func TestAvailable_ReturnsAllScanners_WhenAllAvailable(t *testing.T) {
	r := NewRegistry()
	r.Register(newMockScanner("available-1", true))
	r.Register(newMockScanner("available-2", true))

	result := r.Available()

	assert.Len(t, result, 2)
}
