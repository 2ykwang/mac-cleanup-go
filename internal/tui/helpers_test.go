package tui

import (
	"testing"
	"time"

	"github.com/2ykwang/mac-cleanup-go/pkg/types"
	"github.com/stretchr/testify/assert"
)

func newTestModelWithSortOrder(order types.SortOrder) *Model {
	return &Model{
		sortOrder: order,
	}
}

func TestSortItems_EmptySlice(t *testing.T) {
	m := newTestModelWithSortOrder(types.SortBySize)

	result := m.sortItems([]types.CleanableItem{})

	assert.Empty(t, result)
}

func TestSortItems_BySize(t *testing.T) {
	m := newTestModelWithSortOrder(types.SortBySize)
	items := []types.CleanableItem{
		{Name: "small", Size: 100},
		{Name: "large", Size: 1000},
		{Name: "medium", Size: 500},
	}

	result := m.sortItems(items)

	assert.Equal(t, "large", result[0].Name)
	assert.Equal(t, "medium", result[1].Name)
	assert.Equal(t, "small", result[2].Name)
}

func TestSortItems_ByName(t *testing.T) {
	m := newTestModelWithSortOrder(types.SortByName)
	items := []types.CleanableItem{
		{Name: "zebra", Size: 100},
		{Name: "alpha", Size: 1000},
		{Name: "beta", Size: 500},
	}

	result := m.sortItems(items)

	assert.Equal(t, "alpha", result[0].Name)
	assert.Equal(t, "beta", result[1].Name)
	assert.Equal(t, "zebra", result[2].Name)
}

func TestSortItems_ByAge(t *testing.T) {
	m := newTestModelWithSortOrder(types.SortByAge)
	now := time.Now()
	items := []types.CleanableItem{
		{Name: "recent", ModifiedAt: now.Add(-1 * time.Hour)},
		{Name: "old", ModifiedAt: now.Add(-30 * 24 * time.Hour)},
		{Name: "middle", ModifiedAt: now.Add(-7 * 24 * time.Hour)},
	}

	result := m.sortItems(items)

	assert.Equal(t, "old", result[0].Name)
	assert.Equal(t, "middle", result[1].Name)
	assert.Equal(t, "recent", result[2].Name)
}

func TestSortItems_DoesNotModifyOriginal(t *testing.T) {
	m := newTestModelWithSortOrder(types.SortBySize)
	items := []types.CleanableItem{
		{Name: "small", Size: 100},
		{Name: "large", Size: 1000},
	}

	_ = m.sortItems(items)

	assert.Equal(t, "small", items[0].Name)
	assert.Equal(t, "large", items[1].Name)
}

func TestFilterItems_EmptyQuery(t *testing.T) {
	m := &Model{}
	items := []types.CleanableItem{
		{Name: "file1.txt", Path: "/path/to/file1.txt"},
		{Name: "file2.txt", Path: "/path/to/file2.txt"},
	}

	result := m.filterItems(items, "")

	assert.Len(t, result, 2)
}

func TestFilterItems_MatchByName(t *testing.T) {
	m := &Model{}
	items := []types.CleanableItem{
		{Name: "cache.db", Path: "/path/cache.db"},
		{Name: "logs.txt", Path: "/path/logs.txt"},
		{Name: "data.json", Path: "/path/data.json"},
	}

	result := m.filterItems(items, "cache")

	assert.Len(t, result, 1)
	assert.Equal(t, "cache.db", result[0].Name)
}

func TestFilterItems_CaseInsensitive(t *testing.T) {
	m := &Model{}
	items := []types.CleanableItem{
		{Name: "Cache.db", Path: "/path/Cache.db"},
		{Name: "CACHE.txt", Path: "/path/CACHE.txt"},
		{Name: "other.json", Path: "/path/other.json"},
	}

	result := m.filterItems(items, "cache")

	assert.Len(t, result, 2)
}

func TestFilterItems_MatchByPath(t *testing.T) {
	m := &Model{}
	items := []types.CleanableItem{
		{Name: "file1.txt", Path: "/Library/Caches/file1.txt"},
		{Name: "file2.txt", Path: "/tmp/file2.txt"},
	}

	result := m.filterItems(items, "Library")

	assert.Len(t, result, 1)
	assert.Equal(t, "file1.txt", result[0].Name)
}

func TestFilterItems_PartialMatch(t *testing.T) {
	m := &Model{}
	items := []types.CleanableItem{
		{Name: "application_cache.db", Path: "/path/application_cache.db"},
		{Name: "logs.txt", Path: "/path/logs.txt"},
	}

	result := m.filterItems(items, "app")

	assert.Len(t, result, 1)
	assert.Equal(t, "application_cache.db", result[0].Name)
}

func TestFilterItems_NoMatch(t *testing.T) {
	m := &Model{}
	items := []types.CleanableItem{
		{Name: "file1.txt", Path: "/path/file1.txt"},
		{Name: "file2.txt", Path: "/path/file2.txt"},
	}

	result := m.filterItems(items, "xyz")

	assert.Empty(t, result)
}
