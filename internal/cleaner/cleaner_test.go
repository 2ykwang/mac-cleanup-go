package cleaner

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/mocks"
	"github.com/2ykwang/mac-cleanup-go/internal/target"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

// newMockTargetWithCategory creates a MockTarget with basic setup.
func newMockTargetWithCategory(cat types.Category) *mocks.MockTarget {
	m := new(mocks.MockTarget)
	m.On("Category").Return(cat)
	m.On("IsAvailable").Return(true)
	m.On("Scan").Return((*types.ScanResult)(nil), nil)
	return m
}

func TestNew(t *testing.T) {
	registry := target.NewRegistry()
	c := NewExecutor(registry)
	require.NotNil(t, c)
}

func TestClean_CategoryInResult(t *testing.T) {
	c := NewExecutor(nil)

	cat := types.Category{
		ID:     "test-id",
		Name:   "Test Category",
		Method: types.MethodTrash,
		Safety: types.SafetyLevelSafe,
	}

	result := c.Trash(cat, []types.CleanableItem{})

	assert.Equal(t, "test-id", result.Category.ID)
	assert.Equal(t, "Test Category", result.Category.Name)
}

func TestClean_SkipsSIPProtectedPaths(t *testing.T) {
	c := NewExecutor(nil)

	items := []types.CleanableItem{
		{Path: "/System/Library/Caches/test", Name: "sip-protected", Size: 1000},
		{Path: "/bin/test", Name: "bin-protected", Size: 2000},
	}

	cat := types.Category{
		ID:     "test",
		Name:   "Test Category",
		Method: types.MethodTrash,
	}

	result := c.Trash(cat, items)

	assert.Equal(t, 0, result.CleanedItems, "SIP protected items should be skipped")
	assert.Equal(t, 2, result.SkippedItems)
}

func TestClean_Permanent_RemovesFiles(t *testing.T) {
	c := NewExecutor(nil)

	tmpFile, err := os.CreateTemp("", "cleanup-test-*")
	require.NoError(t, err)
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

	result := c.Permanent(cat, items)

	assert.Equal(t, 1, result.CleanedItems)
	assert.Empty(t, result.Errors)

	_, err = os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(err), "file should be deleted")
}

func TestClean_Permanent_RemovesDirectories(t *testing.T) {
	c := NewExecutor(nil)

	tmpDir, err := os.MkdirTemp("", "cleanup-test-dir-*")
	require.NoError(t, err)

	tmpFile, err := os.CreateTemp(tmpDir, "file-*")
	require.NoError(t, err)
	tmpFile.Close()

	items := []types.CleanableItem{
		{Path: tmpDir, Name: "test-dir", Size: 100, IsDirectory: true},
	}

	cat := types.Category{
		ID:     "test",
		Name:   "Test Category",
		Method: types.MethodPermanent,
	}

	result := c.Permanent(cat, items)

	assert.Equal(t, 1, result.CleanedItems)

	_, err = os.Stat(tmpDir)
	assert.True(t, os.IsNotExist(err), "directory should be deleted")
}

func TestClean_Permanent_SkipsSIPProtectedPaths(t *testing.T) {
	c := NewExecutor(nil)

	items := []types.CleanableItem{
		{Path: "/System/Library/Caches/test", Name: "sip-protected", Size: 1000},
	}

	cat := types.Category{
		ID:     "test",
		Name:   "Test Category",
		Method: types.MethodPermanent,
	}

	result := c.Permanent(cat, items)

	assert.Equal(t, 1, result.SkippedItems)
}

func TestClean_MethodBuiltin_DelegatesToTarget(t *testing.T) {
	registry := target.NewRegistry()
	cat := types.Category{
		ID:     "docker",
		Name:   "Docker",
		Method: types.MethodBuiltin,
	}

	mockTarget := newMockTargetWithCategory(cat)
	cleanResult := types.NewCleanResult(cat)
	cleanResult.CleanedItems = 2
	cleanResult.FreedSpace = 100
	cleanResult.Errors = []string{}
	mockTarget.On("Clean", mock.Anything).Return(cleanResult, nil)
	registry.Register(mockTarget)

	c := NewExecutor(registry)
	items := []types.CleanableItem{
		{Path: "/tmp/test1", Name: "test1", Size: 100},
		{Path: "/tmp/test2", Name: "test2", Size: 200},
	}

	result := c.Builtin(cat, items)

	mockTarget.AssertCalled(t, "Clean", items)
	assert.Equal(t, 2, result.CleanedItems)
	assert.Equal(t, int64(100), result.FreedSpace)
}

func TestClean_MethodBuiltin_TargetNotFound(t *testing.T) {
	registry := target.NewRegistry()
	c := NewExecutor(registry)

	cat := types.Category{
		ID:     "nonexistent",
		Name:   "Nonexistent",
		Method: types.MethodBuiltin,
	}
	items := []types.CleanableItem{
		{Path: "/tmp/test", Name: "test", Size: 100},
	}

	result := c.Builtin(cat, items)

	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "scanner not found")
	assert.Equal(t, 0, result.CleanedItems)
}

func TestClean_MethodBuiltin_TargetReturnsError(t *testing.T) {
	registry := target.NewRegistry()
	cat := types.Category{
		ID:     "docker",
		Name:   "Docker",
		Method: types.MethodBuiltin,
	}

	mockTarget := newMockTargetWithCategory(cat)
	cleanResult := types.NewCleanResult(cat)
	cleanResult.CleanedItems = 1
	cleanResult.FreedSpace = 50
	cleanResult.Errors = []string{"partial failure"}
	mockTarget.On("Clean", mock.Anything).Return(cleanResult, nil)
	registry.Register(mockTarget)

	c := NewExecutor(registry)
	items := []types.CleanableItem{
		{Path: "/tmp/test", Name: "test", Size: 100},
	}

	result := c.Builtin(cat, items)

	mockTarget.AssertCalled(t, "Clean", mock.Anything)
	assert.Equal(t, 1, result.CleanedItems)
	assert.Equal(t, int64(50), result.FreedSpace)
	assert.Contains(t, result.Errors, "partial failure")
}

func TestClean_MethodBuiltin_TargetErrorPropagates(t *testing.T) {
	registry := target.NewRegistry()
	cat := types.Category{
		ID:     "docker",
		Name:   "Docker",
		Method: types.MethodBuiltin,
	}

	mockTarget := newMockTargetWithCategory(cat)
	cleanResult := types.NewCleanResult(cat)
	cleanResult.CleanedItems = 0
	cleanResult.FreedSpace = 0
	cleanResult.Errors = []string{}
	mockTarget.On("Clean", mock.Anything).Return(cleanResult, fmt.Errorf("scanner failed"))
	registry.Register(mockTarget)

	c := NewExecutor(registry)

	result := c.Builtin(cat, []types.CleanableItem{})

	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "scanner failed")
}

func TestClean_MethodBuiltin_TargetNilResultWithError(t *testing.T) {
	registry := target.NewRegistry()
	cat := types.Category{
		ID:     "docker",
		Name:   "Docker",
		Method: types.MethodBuiltin,
	}

	mockTarget := newMockTargetWithCategory(cat)
	mockTarget.On("Clean", mock.Anything).Return(nil, errors.New("scanner failed"))
	registry.Register(mockTarget)

	c := NewExecutor(registry)

	result := c.Builtin(cat, []types.CleanableItem{{Path: "/tmp/test", Name: "test"}})

	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "scanner failed")
}

func TestClean_Manual_SkipsWithGuide(t *testing.T) {
	c := NewExecutor(nil)

	cat := types.Category{
		ID:     "test-manual",
		Name:   "Manual Task",
		Method: types.MethodManual,
		Guide:  "Open Finder and delete manually",
	}

	items := []types.CleanableItem{
		{Path: "/some/path", Name: "manual-item", Size: 1000},
	}

	result := c.Manual(cat, items)

	assert.Equal(t, 0, result.CleanedItems, "manual methods should skip all items")
	assert.Equal(t, 1, result.SkippedItems)
}

func TestClean_Trash_MovesToTrash(t *testing.T) {
	original := utils.MoveToTrashBatch
	defer func() { utils.MoveToTrashBatch = original }()

	var trashedPaths []string
	utils.MoveToTrashBatch = func(paths []string) utils.TrashBatchResult {
		trashedPaths = paths
		return utils.TrashBatchResult{
			Succeeded: paths,
			Failed:    make(map[string]error),
		}
	}

	c := NewExecutor(nil)
	cat := types.Category{
		ID:     "test-trash",
		Name:   "Test Trash",
		Method: types.MethodTrash,
	}
	items := []types.CleanableItem{
		{Path: "/tmp/test1", Name: "test1", Size: 100},
		{Path: "/tmp/test2", Name: "test2", Size: 200},
	}

	result := c.Trash(cat, items)

	assert.Equal(t, 2, result.CleanedItems)
	assert.Equal(t, int64(300), result.FreedSpace)
	assert.Equal(t, []string{"/tmp/test1", "/tmp/test2"}, trashedPaths)
	assert.Empty(t, result.Errors)
}

func TestClean_Trash_PartialFailure(t *testing.T) {
	original := utils.MoveToTrashBatch
	defer func() { utils.MoveToTrashBatch = original }()

	utils.MoveToTrashBatch = func(paths []string) utils.TrashBatchResult {
		result := utils.TrashBatchResult{
			Succeeded: make([]string, 0, len(paths)),
			Failed:    make(map[string]error),
		}
		for _, p := range paths {
			if p == "/tmp/test2" {
				result.Failed[p] = fmt.Errorf("permission denied")
			} else {
				result.Succeeded = append(result.Succeeded, p)
			}
		}
		return result
	}

	c := NewExecutor(nil)
	cat := types.Category{
		ID:     "test-trash",
		Name:   "Test Trash",
		Method: types.MethodTrash,
	}
	items := []types.CleanableItem{
		{Path: "/tmp/test1", Name: "test1", Size: 100},
		{Path: "/tmp/test2", Name: "test2", Size: 200},
		{Path: "/tmp/test3", Name: "test3", Size: 300},
	}

	result := c.Trash(cat, items)

	assert.Equal(t, 2, result.CleanedItems)
	assert.Equal(t, int64(400), result.FreedSpace)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "permission denied")
}
