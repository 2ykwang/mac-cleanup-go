package tui

import "github.com/2ykwang/mac-cleanup-go/internal/types"

// Selection state management

func (m *Model) getSelectedResults() []*types.ScanResult {
	var selected []*types.ScanResult
	if len(m.selectedOrder) > 0 {
		for _, id := range m.selectedOrder {
			if !m.selected[id] {
				continue
			}
			if r, ok := m.resultMap[id]; ok {
				selected = append(selected, r)
			}
		}
		return selected
	}

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

func (m *Model) getAvailableSize() int64 {
	var total int64
	for _, r := range m.results {
		total += r.TotalSize
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

func (m *Model) addSelected(id string) {
	if m.selected[id] {
		return
	}
	m.selected[id] = true
	m.selectedOrder = append(m.selectedOrder, id)
}

func (m *Model) removeSelected(id string) {
	if !m.selected[id] {
		return
	}
	m.selected[id] = false
	for i, existing := range m.selectedOrder {
		if existing == id {
			m.selectedOrder = append(m.selectedOrder[:i], m.selectedOrder[i+1:]...)
			break
		}
	}
}

func (m *Model) clearSelections() {
	m.selected = make(map[string]bool)
	m.selectedOrder = nil
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
