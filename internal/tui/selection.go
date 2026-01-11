package tui

import "github.com/2ykwang/mac-cleanup-go/internal/types"

// Selection state management

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

// Exclusion management

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
