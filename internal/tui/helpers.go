package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/2ykwang/mac-cleanup-go/internal/utils"
	"github.com/2ykwang/mac-cleanup-go/pkg/types"
)

// Selection helpers

func (m *Model) getSelectedResults() []*types.ScanResult {
	var selected []*types.ScanResult
	for _, r := range m.results {
		if m.selected[r.Category.ID] {
			selected = append(selected, r)
		}
	}
	return selected
}

func (m *Model) hasSelection() bool {
	for _, v := range m.selected {
		if v {
			return true
		}
	}
	return false
}

func (m *Model) getSelectedSize() int64 {
	var total int64
	for id, sel := range m.selected {
		if sel {
			if r, ok := m.resultMap[id]; ok {
				total += m.getEffectiveSize(r)
			}
		}
	}
	return total
}

func (m *Model) getEffectiveSize(r *types.ScanResult) int64 {
	excludedMap := m.excluded[r.Category.ID]
	if excludedMap == nil {
		return r.TotalSize
	}
	var total int64
	for _, item := range r.Items {
		if !excludedMap[item.Path] {
			total += item.Size
		}
	}
	return total
}

func (m *Model) getSelectedCount() int {
	count := 0
	for _, v := range m.selected {
		if v {
			count++
		}
	}
	return count
}

// Preview navigation helpers

func (m *Model) getPreviewCatResult() *types.ScanResult {
	if m.previewCatID == "" {
		return nil
	}
	return m.resultMap[m.previewCatID]
}

func (m *Model) findSelectedCatIndex() int {
	selected := m.getSelectedResults()
	for i, r := range selected {
		if r.Category.ID == m.previewCatID {
			return i
		}
	}
	return -1
}

func (m *Model) findPrevSelectedCatID() string {
	selected := m.getSelectedResults()
	for i, r := range selected {
		if r.Category.ID == m.previewCatID && i > 0 {
			return selected[i-1].Category.ID
		}
	}
	return m.previewCatID
}

func (m *Model) findNextSelectedCatID() string {
	selected := m.getSelectedResults()
	for i, r := range selected {
		if r.Category.ID == m.previewCatID && i < len(selected)-1 {
			return selected[i+1].Category.ID
		}
	}
	return m.previewCatID
}

// Layout helpers

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func (m *Model) availableLines(header, footer string) int {
	used := countLines(header) + countLines(footer)
	available := m.height - used
	if available < 3 {
		return 3
	}
	return available
}

// pageSize returns the number of items to move for page up/down navigation.
// Uses a reasonable default based on typical terminal heights.
func (m *Model) pageSize() int {
	// Reserve space for header/footer, use about 80% of visible area
	pageSize := (m.height - 10) * 8 / 10
	if pageSize < 5 {
		return 5
	}
	return pageSize
}

// getVisiblePreviewItems returns the sorted and filtered items for the current preview.
// This is what the user actually sees on screen.
func (m *Model) getVisiblePreviewItems() []types.CleanableItem {
	r := m.getPreviewCatResult()
	if r == nil {
		return nil
	}

	items := r.Items

	// Apply filter if active
	if m.filterState == FilterTyping {
		query := m.filterInput.Value()
		if query != "" {
			items = m.filterItems(items, query)
		}
	} else if m.filterState == FilterApplied && m.filterText != "" {
		items = m.filterItems(items, m.filterText)
	}

	// Apply sort
	return m.sortItems(items)
}

// getCurrentPreviewItem returns the item at the current cursor position
// after applying filter and sort. Returns nil if no valid item.
func (m *Model) getCurrentPreviewItem() *types.CleanableItem {
	items := m.getVisiblePreviewItems()
	if items == nil || m.previewItemIndex < 0 || m.previewItemIndex >= len(items) {
		return nil
	}
	return &items[m.previewItemIndex]
}

func (m *Model) adjustScrollFor(cursor, scroll, visible, _ int) int {
	if cursor < scroll {
		return cursor
	}
	if cursor >= scroll+visible {
		return cursor - visible + 1
	}
	return scroll
}

// Directory helpers

func (m *Model) readDirectory(path string) []types.CleanableItem {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	var items []types.CleanableItem
	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		item := types.CleanableItem{
			Path:        fullPath,
			Name:        entry.Name(),
			IsDirectory: entry.IsDir(),
			ModifiedAt:  info.ModTime(),
		}

		if entry.IsDir() {
			item.Size, item.FileCount, _ = utils.GetDirSizeWithCount(fullPath)
		} else {
			item.Size = info.Size()
			item.FileCount = 1
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Size > items[j].Size
	})
	return items
}

// Exclusion helpers

func (m *Model) isExcluded(catID, path string) bool {
	if m.excluded[catID] == nil {
		return false
	}
	return m.excluded[catID][path]
}

func (m *Model) toggleExclude(catID, path string) {
	if m.excluded[catID] == nil {
		m.excluded[catID] = make(map[string]bool)
	}
	m.excluded[catID][path] = !m.excluded[catID][path]
	m.saveExcludedPaths()
}

func (m *Model) saveExcludedPaths() {
	for catID, pathMap := range m.excluded {
		var paths []string
		for path, excluded := range pathMap {
			if excluded {
				paths = append(paths, path)
			}
		}
		m.userConfig.SetExcludedPaths(catID, paths)
	}
	_ = m.userConfig.Save()
}

func (m *Model) autoExcludeCategory(catID string, r *types.ScanResult) {
	if m.excluded[catID] == nil {
		m.excluded[catID] = make(map[string]bool)
	}
	for _, item := range r.Items {
		m.excluded[catID][item.Path] = true
	}
	m.saveExcludedPaths()
}

func (m *Model) clearExcludeCategory(catID string) {
	if m.excluded[catID] != nil {
		m.excluded[catID] = make(map[string]bool)
		m.saveExcludedPaths()
	}
}

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
