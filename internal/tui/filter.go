package tui

import (
	"sort"
	"strings"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

// sortItems sorts items based on the current sort order.
// Returns a new sorted slice without modifying the original.
func (m *Model) sortItems(items []types.CleanableItem) []types.CleanableItem {
	if len(items) == 0 {
		return items
	}

	// Make a copy to avoid modifying the original slice
	sorted := make([]types.CleanableItem, len(items))
	copy(sorted, items)

	switch m.sortOrder {
	case types.SortBySize:
		// Size descending (largest first)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Size > sorted[j].Size
		})
	case types.SortByName:
		// Name ascending (A→Z)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Name < sorted[j].Name
		})
	case types.SortByAge:
		// Age ascending (oldest first = earliest ModifiedAt)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].ModifiedAt.Before(sorted[j].ModifiedAt)
		})
	}

	return sorted
}

// filterItems filters items by the given query string.
// Supports space-separated AND matching: "cache simple" matches items
// containing both "cache" AND "simple" anywhere in Name or Path.
// Case-insensitive. Returns empty slice if query is empty (no filtering).
func (m *Model) filterItems(items []types.CleanableItem, query string) []types.CleanableItem {
	if query == "" {
		return items
	}

	// Split query into terms (space = AND)
	terms := strings.Fields(strings.ToLower(query))
	if len(terms) == 0 {
		return items
	}

	var filtered []types.CleanableItem

	for _, item := range items {
		// Combine name and path for matching
		combined := strings.ToLower(item.Name + " " + item.Path)

		// All terms must match (AND logic)
		match := true
		for _, term := range terms {
			if !strings.Contains(combined, term) {
				match = false
				break
			}
		}

		if match {
			filtered = append(filtered, item)
		}
	}

	return filtered
}
