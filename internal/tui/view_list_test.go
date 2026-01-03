package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"mac-cleanup-go/internal/userconfig"
	"mac-cleanup-go/pkg/types"
)

func newTestModelWithConfig() *Model {
	return &Model{
		config: &types.Config{
			Groups: []types.Group{
				{ID: "system", Name: "System", Order: 1},
				{ID: "browser", Name: "Browsers", Order: 2},
				{ID: "dev", Name: "Development", Order: 3},
				{ID: "app", Name: "Applications", Order: 4},
			},
		},
		results:        make([]*types.ScanResult, 0),
		resultMap:      make(map[string]*types.ScanResult),
		selected:       make(map[string]bool),
		excluded:       make(map[string]map[string]bool),
		drillDownStack: make([]drillDownState, 0),
		view:           ViewList,
		width:          80,
		height:         24,
		userConfig:     &userconfig.UserConfig{ExcludedPaths: make(map[string][]string)},
		recentDeleted:  NewRingBuffer[DeletedItemEntry](defaultRecentItemsCapacity),
	}
}

func TestGetGroupStats_EmptyResults(t *testing.T) {
	m := newTestModelWithConfig()

	stats := m.getGroupStats()

	assert.Empty(t, stats)
}

func TestGetGroupStats_FiltersZeroByteGroups(t *testing.T) {
	m := newTestModelWithConfig()
	m.results = []*types.ScanResult{
		{
			Category:  types.Category{ID: "cat1", Name: "Chrome Cache", Group: "browser"},
			TotalSize: 1000,
		},
		{
			Category:  types.Category{ID: "cat2", Name: "Empty Category", Group: "system"},
			TotalSize: 0,
		},
	}

	stats := m.getGroupStats()

	assert.Len(t, stats, 1)
	assert.Equal(t, "Browsers", stats[0].Name)
}

func TestGetGroupStats_SortsBySizeDescending(t *testing.T) {
	m := newTestModelWithConfig()
	m.results = []*types.ScanResult{
		{Category: types.Category{ID: "cat1", Group: "browser"}, TotalSize: 1000},
		{Category: types.Category{ID: "cat2", Group: "dev"}, TotalSize: 5000},
		{Category: types.Category{ID: "cat3", Group: "system"}, TotalSize: 2000},
	}

	stats := m.getGroupStats()

	assert.Len(t, stats, 3)
	assert.Equal(t, "Development", stats[0].Name)
	assert.Equal(t, int64(5000), stats[0].Size)
	assert.Equal(t, "System", stats[1].Name)
	assert.Equal(t, int64(2000), stats[1].Size)
	assert.Equal(t, "Browsers", stats[2].Name)
	assert.Equal(t, int64(1000), stats[2].Size)
}

func TestGetGroupStats_AggregatesMultipleCategoriesInSameGroup(t *testing.T) {
	m := newTestModelWithConfig()
	m.results = []*types.ScanResult{
		{Category: types.Category{ID: "cat1", Group: "dev"}, TotalSize: 1000},
		{Category: types.Category{ID: "cat2", Group: "dev"}, TotalSize: 2000},
		{Category: types.Category{ID: "cat3", Group: "dev"}, TotalSize: 3000},
	}

	stats := m.getGroupStats()

	assert.Len(t, stats, 1)
	assert.Equal(t, "Development", stats[0].Name)
	assert.Equal(t, int64(6000), stats[0].Size)
}

func TestGetGroupStats_HandlesUnknownGroupID(t *testing.T) {
	m := newTestModelWithConfig()
	m.results = []*types.ScanResult{
		{Category: types.Category{ID: "cat1", Group: "unknown_group"}, TotalSize: 1000},
	}

	stats := m.getGroupStats()

	assert.Len(t, stats, 1)
	assert.Equal(t, "unknown_group", stats[0].Name)
}
