package tui

import (
	"os"
	"path/filepath"
	"sort"

	"mac-cleanup-go/internal/utils"
	"mac-cleanup-go/pkg/types"
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

// Scroll helpers

func (m *Model) visibleLines() int {
	lines := m.height - 10
	if lines < 5 {
		return 5
	}
	return lines
}

func (m *Model) reportVisibleLines() int {
	lines := m.height - 12
	if lines < 5 {
		return 5
	}
	return lines
}

func (m *Model) adjustScroll() {
	visible := m.visibleLines()
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	} else if m.cursor >= m.scroll+visible {
		m.scroll = m.cursor - visible + 1
	}
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
			item.Size, _ = utils.GetDirSize(fullPath)
		} else {
			item.Size = info.Size()
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Size > items[j].Size
	})
	return items
}

func (m *Model) tryDrillDown() bool {
	r := m.getPreviewCatResult()
	if r == nil {
		return false
	}

	if m.previewItemIndex < 0 || m.previewItemIndex >= len(r.Items) {
		return false
	}

	item := r.Items[m.previewItemIndex]
	if !item.IsDirectory {
		return false
	}

	items := m.readDirectory(item.Path)
	if len(items) == 0 {
		return false
	}

	m.drillDownStack = append(m.drillDownStack, drillDownState{
		path:   item.Path,
		items:  items,
		cursor: 0,
		scroll: 0,
	})
	return true
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
