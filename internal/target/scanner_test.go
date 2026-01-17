package target

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/2ykwang/mac-cleanup-go/internal/mocks"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

// newMockTarget creates a MockTarget with common setup.
func newMockTarget(id string, available bool) *mocks.MockTarget {
	m := new(mocks.MockTarget)
	cat := types.Category{ID: id, Name: "Mock " + id}
	m.On("Category").Return(cat)
	m.On("IsAvailable").Return(available)
	m.On("Scan").Return(&types.ScanResult{Category: cat}, nil)
	m.On("Clean", mock.Anything).Return(&types.CleanResult{Category: cat}, nil)
	return m
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
	target := newMockTarget("test-id", true)

	r.Register(target)

	assert.Len(t, r.targets, 1)
}

func TestRegister_UsesCategoryIDAsKey(t *testing.T) {
	r := NewRegistry()
	target := newMockTarget("my-scanner", true)

	r.Register(target)

	_, exists := r.targets["my-scanner"]
	assert.True(t, exists)
}

func TestRegister_OverwritesExistingTarget(t *testing.T) {
	r := NewRegistry()
	target1 := newMockTarget("same-id", true)
	target2 := newMockTarget("same-id", false)

	r.Register(target1)
	r.Register(target2)

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
	target := newMockTarget("existing", true)
	r.Register(target)

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
