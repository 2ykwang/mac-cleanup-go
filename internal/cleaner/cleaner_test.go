package cleaner

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/scanner"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

// mockScanner implements scanner.Scanner for testing
type mockScanner struct {
	category    types.Category
	cleanCalled bool
	cleanItems  []types.CleanableItem
	cleanResult *types.CleanResult
	cleanErr    error
}

func (m *mockScanner) Scan() (*types.ScanResult, error) {
	return nil, nil
}

func (m *mockScanner) Clean(items []types.CleanableItem) (*types.CleanResult, error) {
	m.cleanCalled = true
	m.cleanItems = items
	if m.cleanResult != nil {
		return m.cleanResult, m.cleanErr
	}
	return &types.CleanResult{
		Category:     m.category,
		CleanedItems: len(items),
		FreedSpace:   100,
		Errors:       []string{},
	}, m.cleanErr
}

func (m *mockScanner) Category() types.Category {
	return m.category
}

func (m *mockScanner) IsAvailable() bool {
	return true
}

func TestNew(t *testing.T) {
	registry := scanner.NewRegistry()
	c := New(registry)
	require.NotNil(t, c)
}

func TestClean_CategoryInResult(t *testing.T) {
	c := New(nil)

	cat := types.Category{
		ID:     "test-id",
		Name:   "Test Category",
		Method: types.MethodTrash,
		Safety: types.SafetyLevelSafe,
	}

	result := c.Clean(cat, []types.CleanableItem{})

	assert.Equal(t, "test-id", result.Category.ID)
	assert.Equal(t, "Test Category", result.Category.Name)
}

func TestClean_SkipsSIPProtectedPaths(t *testing.T) {
	c := New(nil)

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

	assert.Equal(t, 0, result.CleanedItems, "SIP protected items should be skipped")
	assert.Equal(t, 2, result.SkippedItems)
}

func TestClean_Permanent_RemovesFiles(t *testing.T) {
	c := New(nil)

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

	result := c.Clean(cat, items)

	assert.Equal(t, 1, result.CleanedItems)
	assert.Empty(t, result.Errors)

	_, err = os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(err), "file should be deleted")
}

func TestClean_Permanent_RemovesDirectories(t *testing.T) {
	c := New(nil)

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

	result := c.Clean(cat, items)

	assert.Equal(t, 1, result.CleanedItems)

	_, err = os.Stat(tmpDir)
	assert.True(t, os.IsNotExist(err), "directory should be deleted")
}

func TestClean_Permanent_SkipsSIPProtectedPaths(t *testing.T) {
	c := New(nil)

	items := []types.CleanableItem{
		{Path: "/System/Library/Caches/test", Name: "sip-protected", Size: 1000},
	}

	cat := types.Category{
		ID:     "test",
		Name:   "Test Category",
		Method: types.MethodPermanent,
	}

	result := c.Clean(cat, items)

	assert.Equal(t, 1, result.SkippedItems)
}

func TestClean_MethodBuiltin_DelegatesToScanner(t *testing.T) {
	registry := scanner.NewRegistry()
	mock := &mockScanner{
		category: types.Category{
			ID:     "docker",
			Name:   "Docker",
			Method: types.MethodBuiltin,
		},
	}
	registry.Register(mock)

	c := New(registry)

	cat := types.Category{
		ID:     "docker",
		Name:   "Docker",
		Method: types.MethodBuiltin,
	}
	items := []types.CleanableItem{
		{Path: "/tmp/test1", Name: "test1", Size: 100},
		{Path: "/tmp/test2", Name: "test2", Size: 200},
	}

	result := c.Clean(cat, items)

	assert.True(t, mock.cleanCalled, "Scanner.Clean should be called for MethodBuiltin")
	assert.Equal(t, items, mock.cleanItems, "Items should be passed to Scanner.Clean")
	assert.Equal(t, 2, result.CleanedItems)
	assert.Equal(t, int64(100), result.FreedSpace)
}

func TestClean_MethodBuiltin_ScannerNotFound(t *testing.T) {
	registry := scanner.NewRegistry()
	c := New(registry)

	cat := types.Category{
		ID:     "nonexistent",
		Name:   "Nonexistent",
		Method: types.MethodBuiltin,
	}
	items := []types.CleanableItem{
		{Path: "/tmp/test", Name: "test", Size: 100},
	}

	result := c.Clean(cat, items)

	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "scanner not found")
	assert.Equal(t, 0, result.CleanedItems)
}

func TestClean_MethodBuiltin_ScannerReturnsError(t *testing.T) {
	registry := scanner.NewRegistry()
	mock := &mockScanner{
		category: types.Category{
			ID:     "docker",
			Name:   "Docker",
			Method: types.MethodBuiltin,
		},
		cleanResult: &types.CleanResult{
			Category:     types.Category{ID: "docker"},
			CleanedItems: 1,
			FreedSpace:   50,
			Errors:       []string{"partial failure"},
		},
	}
	registry.Register(mock)

	c := New(registry)

	cat := types.Category{
		ID:     "docker",
		Name:   "Docker",
		Method: types.MethodBuiltin,
	}
	items := []types.CleanableItem{
		{Path: "/tmp/test", Name: "test", Size: 100},
	}

	result := c.Clean(cat, items)

	assert.True(t, mock.cleanCalled)
	assert.Equal(t, 1, result.CleanedItems)
	assert.Equal(t, int64(50), result.FreedSpace)
	assert.Contains(t, result.Errors, "partial failure")
}

func TestClean_MethodBuiltin_ScannerErrorPropagates(t *testing.T) {
	registry := scanner.NewRegistry()
	mock := &mockScanner{
		category: types.Category{
			ID:     "docker",
			Name:   "Docker",
			Method: types.MethodBuiltin,
		},
		cleanErr: fmt.Errorf("scanner failed"),
	}
	registry.Register(mock)

	c := New(registry)

	cat := types.Category{
		ID:     "docker",
		Name:   "Docker",
		Method: types.MethodBuiltin,
	}

	result := c.Clean(cat, []types.CleanableItem{})

	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "scanner failed")
}

func TestClean_Manual_SkipsWithGuide(t *testing.T) {
	c := New(nil)

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

	assert.Equal(t, 0, result.CleanedItems, "manual methods should skip all items")
	assert.Equal(t, 1, result.SkippedItems)
}

func TestClean_Trash_MovesToTrash(t *testing.T) {
	original := utils.MoveToTrash
	defer func() { utils.MoveToTrash = original }()

	var trashedPaths []string
	utils.MoveToTrash = func(path string) error {
		trashedPaths = append(trashedPaths, path)
		return nil
	}

	c := New(nil)
	cat := types.Category{
		ID:     "test-trash",
		Name:   "Test Trash",
		Method: types.MethodTrash,
	}
	items := []types.CleanableItem{
		{Path: "/tmp/test1", Name: "test1", Size: 100},
		{Path: "/tmp/test2", Name: "test2", Size: 200},
	}

	result := c.Clean(cat, items)

	assert.Equal(t, 2, result.CleanedItems)
	assert.Equal(t, int64(300), result.FreedSpace)
	assert.Equal(t, []string{"/tmp/test1", "/tmp/test2"}, trashedPaths)
	assert.Empty(t, result.Errors)
}

func TestClean_Trash_PartialFailure(t *testing.T) {
	original := utils.MoveToTrash
	defer func() { utils.MoveToTrash = original }()

	utils.MoveToTrash = func(path string) error {
		if path == "/tmp/test2" {
			return fmt.Errorf("permission denied")
		}
		return nil
	}

	c := New(nil)
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

	result := c.Clean(cat, items)

	assert.Equal(t, 2, result.CleanedItems)
	assert.Equal(t, int64(400), result.FreedSpace)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "permission denied")
}
