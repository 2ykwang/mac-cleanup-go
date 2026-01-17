package target

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

type mockTarget struct {
	cat       types.Category
	available bool
}

func newMockTarget(id string, available bool) *mockTarget {
	return &mockTarget{
		cat:       types.Category{ID: id, Name: "Mock " + id},
		available: available,
	}
}

func (m *mockTarget) Scan() (*types.ScanResult, error) {
	return &types.ScanResult{Category: m.cat}, nil
}

func (m *mockTarget) Clean(_ []types.CleanableItem) (*types.CleanResult, error) {
	return &types.CleanResult{Category: m.cat}, nil
}

func (m *mockTarget) Category() types.Category {
	return m.cat
}

func (m *mockTarget) IsAvailable() bool {
	return m.available
}

// --- NewRegistry Tests ---

func TestNewRegistry_ReturnsNonNil(t *testing.T) {
	r := NewRegistry()

	assert.NotNil(t, r)
}

func TestNewRegistry_HasEmptyTargetsMap(t *testing.T) {
	r := NewRegistry()

	assert.Empty(t, r.targets)
}

// --- Register Tests ---

func TestRegister_AddsTargetToRegistry(t *testing.T) {
	r := NewRegistry()
	scanner := newMockTarget("test-id", true)

	r.Register(scanner)

	assert.Len(t, r.targets, 1)
}

func TestRegister_UsesCategoryIDAsKey(t *testing.T) {
	r := NewRegistry()
	scanner := newMockTarget("my-scanner", true)

	r.Register(scanner)

	_, exists := r.targets["my-scanner"]
	assert.True(t, exists)
}

func TestRegister_OverwritesExistingTarget(t *testing.T) {
	r := NewRegistry()
	scanner1 := newMockTarget("same-id", true)
	scanner2 := newMockTarget("same-id", false)

	r.Register(scanner1)
	r.Register(scanner2)

	assert.Len(t, r.targets, 1)
	assert.False(t, r.targets["same-id"].IsAvailable())
}

func TestRegister_MultipleTargetsWithDifferentIDs(t *testing.T) {
	r := NewRegistry()

	r.Register(newMockTarget("scanner-1", true))
	r.Register(newMockTarget("scanner-2", true))
	r.Register(newMockTarget("scanner-3", true))

	assert.Len(t, r.targets, 3)
}

// --- Get Tests ---

func TestGet_ReturnsTarget_WhenExists(t *testing.T) {
	r := NewRegistry()
	scanner := newMockTarget("existing", true)
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

func TestGet_ReturnsCorrectTarget_WhenMultipleRegistered(t *testing.T) {
	r := NewRegistry()
	r.Register(newMockTarget("first", true))
	r.Register(newMockTarget("second", false))
	r.Register(newMockTarget("third", true))

	result, ok := r.Get("second")

	assert.True(t, ok)
	assert.Equal(t, "second", result.Category().ID)
	assert.False(t, result.IsAvailable())
}

// --- All Tests ---

func TestAll_ReturnsEmptySlice_WhenNoTargets(t *testing.T) {
	r := NewRegistry()

	result := r.All()

	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestAll_ReturnsAllRegisteredTargets(t *testing.T) {
	r := NewRegistry()
	r.Register(newMockTarget("a", true))
	r.Register(newMockTarget("b", false))
	r.Register(newMockTarget("c", true))

	result := r.All()

	assert.Len(t, result, 3)
}

func TestAll_IncludesBothAvailableAndUnavailable(t *testing.T) {
	r := NewRegistry()
	r.Register(newMockTarget("available", true))
	r.Register(newMockTarget("unavailable", false))

	result := r.All()

	assert.Len(t, result, 2)
}

// --- Available Tests ---

func TestAvailable_ReturnsEmptySlice_WhenNoTargets(t *testing.T) {
	r := NewRegistry()

	result := r.Available()

	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestAvailable_ReturnsOnlyAvailableTargets(t *testing.T) {
	r := NewRegistry()
	r.Register(newMockTarget("available-1", true))
	r.Register(newMockTarget("unavailable", false))
	r.Register(newMockTarget("available-2", true))

	result := r.Available()

	assert.Len(t, result, 2)
	for _, s := range result {
		assert.True(t, s.IsAvailable())
	}
}

func TestAvailable_ReturnsEmptySlice_WhenAllUnavailable(t *testing.T) {
	r := NewRegistry()
	r.Register(newMockTarget("unavailable-1", false))
	r.Register(newMockTarget("unavailable-2", false))

	result := r.Available()

	assert.Empty(t, result)
}

func TestAvailable_ReturnsAllTargets_WhenAllAvailable(t *testing.T) {
	r := NewRegistry()
	r.Register(newMockTarget("available-1", true))
	r.Register(newMockTarget("available-2", true))

	result := r.Available()

	assert.Len(t, result, 2)
}
