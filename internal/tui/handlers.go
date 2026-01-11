package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.view {
	case ViewList:
		return m.handleListKey(msg)
	case ViewPreview:
		return m.handlePreviewKey(msg)
	case ViewConfirm:
		return m.handleConfirmKey(msg)
	case ViewGuide:
		return m.handleGuideKey(msg)
	case ViewCleaning:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case ViewReport:
		return m.handleReportKey(msg)
	}
	return m, nil
}

func (m *Model) handleReportKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?":
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		m.reportScroll = clamp(m.reportScroll-1, 0, m.maxReportScroll())
	case "down", "j":
		m.reportScroll = clamp(m.reportScroll+1, 0, m.maxReportScroll())
	case "enter", " ":
		// Return to main screen and rescan
		return m, m.startRescanCmd()
	}
	return m, nil
}

func (m *Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?":
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.results)-1 {
			m.cursor++
		}
	case " ":
		if len(m.results) > 0 && m.cursor < len(m.results) {
			r := m.results[m.cursor]
			// Open guide popup for manual categories
			if r.Category.Method == types.MethodManual {
				m.guideCategory = &r.Category
				m.guidePathIndex = 0
				m.view = ViewGuide
				break
			}
			id := r.Category.ID
			wasSelected := m.selected[id]
			m.selected[id] = !wasSelected

			// Auto-exclude all items for risky categories when newly selected
			if !wasSelected && r.Category.Safety == types.SafetyLevelRisky {
				m.autoExcludeCategory(id, r)
			}
		}
	case "a", "A":
		// Select all (excluding manual categories)
		for _, r := range m.results {
			// Skip manual categories - they cannot be selected
			if r.Category.Method == types.MethodManual {
				continue
			}
			wasSelected := m.selected[r.Category.ID]
			m.selected[r.Category.ID] = true

			// Auto-exclude all items for risky categories when newly selected
			if !wasSelected && r.Category.Safety == types.SafetyLevelRisky {
				m.autoExcludeCategory(r.Category.ID, r)
			}
		}
	case "d", "D":
		// Deselect all
		for _, r := range m.results {
			m.selected[r.Category.ID] = false
		}
	case "enter", "p":
		if m.hasSelection() {
			m.previewScroll = 0
			m.previewCatID = ""
			m.previewItemIndex = 0
			m.drillDownStack = m.drillDownStack[:0]
			// Find first selected category
			for _, r := range m.results {
				if m.selected[r.Category.ID] {
					m.previewCatID = r.Category.ID
					break
				}
			}
			m.view = ViewPreview
		}
	}
	return m, nil
}

func (m *Model) handlePreviewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle filter typing mode first
	if m.filterState == FilterTyping {
		return m.handleFilterTypingKey(msg)
	}

	if len(m.drillDownStack) > 0 {
		return m.handleDrillDownKey(msg)
	}

	switch msg.String() {
	case "?":
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc", "n":
		// Clear filter if applied, otherwise go back
		if m.filterState == FilterApplied {
			m.clearFilter()
			m.resetPreviewSelection()
			return m, nil
		}
		m.view = ViewList
	case "up", "k":
		m.movePreviewCursor(-1)
	case "down", "j":
		m.movePreviewCursor(1)
	case "left", "h":
		prevID := m.findPrevSelectedCatID()
		if prevID != m.previewCatID {
			m.previewCatID = prevID
			m.resetPreviewSelection()
			m.clearFilter()
		}
	case "right", "l":
		nextID := m.findNextSelectedCatID()
		if nextID != m.previewCatID {
			m.previewCatID = nextID
			m.resetPreviewSelection()
			m.clearFilter()
		}
	case " ":
		// Toggle exclusion for current item (use visible items after filter/sort)
		r := m.getPreviewCatResult()
		item := m.getCurrentPreviewItem()
		if r != nil && item != nil {
			m.toggleExclude(r.Category.ID, item.Path)
		}
	case "enter":
		// Drill down into directory (use visible items after filter/sort)
		item := m.getCurrentPreviewItem()
		if item != nil && item.IsDirectory {
			items := m.readDirectory(item.Path)
			if len(items) > 0 {
				m.drillDownStack = append(m.drillDownStack, drillDownState{
					path:   item.Path,
					items:  items,
					cursor: 0,
					scroll: 0,
				})
			}
		}
	case "y":
		m.view = ViewConfirm
	case "a", "A":
		// Include all items in current category (clear exclusions)
		if r := m.getPreviewCatResult(); r != nil {
			m.clearExcludeCategory(r.Category.ID)
		}
	case "d", "D":
		// Exclude all items in current category
		if r := m.getPreviewCatResult(); r != nil {
			m.autoExcludeCategory(r.Category.ID, r)
		}
	case "o":
		// Open in Finder (use visible items after filter/sort)
		item := m.getCurrentPreviewItem()
		if item != nil {
			if err := utils.OpenInFinder(item.Path); err != nil {
				m.statusMessage = "Path not found"
			} else {
				m.statusMessage = ""
			}
		}
	case "s":
		// Toggle sort order
		m.sortOrder = m.sortOrder.Next()
		m.resetPreviewSelection()
	case "pgdown":
		// Page down
		m.movePreviewCursor(m.pageSize())
	case "pgup":
		// Page up
		m.movePreviewCursor(-m.pageSize())
	case "home":
		// Go to first item
		m.setPreviewCursor(0, 0)
	case "end":
		// Go to last item
		r := m.getPreviewCatResult()
		if r != nil && len(r.Items) > 0 {
			m.setPreviewCursor(len(r.Items)-1, len(r.Items)-1)
		}
	case "/":
		// Enter search mode
		return m, m.startFilterTyping()
	}
	return m, nil
}

func (m *Model) handleFilterTypingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Apply the filter
		m.applyFilter()
		return m, nil
	case "esc":
		// Cancel search
		m.clearFilter()
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	}

	// Pass other keys to textinput and reset cursor for real-time filtering
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	// Reset cursor position when filter changes
	m.resetPreviewSelection()
	return m, cmd
}

func (m *Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?":
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	case "ctrl+c", "q":
		return m, tea.Quit
	case "y", "Y", "enter":
		m.view = ViewCleaning
		m.startTime = time.Now()
		return m, tea.Batch(m.spinner.Tick, m.doClean())
	case "n", "N", "esc":
		m.view = ViewPreview
	}
	return m, nil
}

func (m *Model) handleDrillDownKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	state := &m.drillDownStack[len(m.drillDownStack)-1]

	switch msg.String() {
	case "?":
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc", "backspace", "n":
		m.drillDownStack = m.drillDownStack[:len(m.drillDownStack)-1]
	case "up", "k":
		m.moveDrillCursor(state, -1)
	case "down", "j":
		m.setDrillCursor(state, state.cursor+1, len(state.items)-1)
	case "enter":
		if state.cursor < len(state.items) {
			item := state.items[state.cursor]
			if item.IsDirectory {
				items := m.readDirectory(item.Path)
				if len(items) > 0 {
					m.drillDownStack = append(m.drillDownStack, drillDownState{
						path:   item.Path,
						items:  items,
						cursor: 0,
						scroll: 0,
					})
				}
			}
		}
	case "o":
		// Open in Finder
		if state.cursor < len(state.items) {
			path := state.items[state.cursor].Path
			if err := utils.OpenInFinder(path); err != nil {
				m.statusMessage = "Path not found"
			} else {
				m.statusMessage = ""
			}
		}
	case "s":
		// Toggle sort order
		m.sortOrder = m.sortOrder.Next()
		state.cursor = 0
		state.scroll = 0
	case "pgdown":
		// Page down
		m.setDrillCursor(state, state.cursor+m.pageSize(), len(state.items)-1)
	case "pgup":
		// Page up
		m.moveDrillCursor(state, -m.pageSize())
	case "home":
		// Go to first item
		m.setDrillCursor(state, 0, 0)
	case "end":
		// Go to last item
		if len(state.items) > 0 {
			m.setDrillCursor(state, len(state.items)-1, len(state.items)-1)
		}
	}
	return m, nil
}

// handleGuideKey handles key events in the guide popup view
func (m *Model) handleGuideKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc", "enter", " ":
		m.guideCategory = nil
		m.guidePathIndex = 0
		m.view = ViewList
	case "up", "k":
		if m.guidePathIndex > 0 {
			m.guidePathIndex--
		}
	case "down", "j":
		if m.guideCategory != nil && m.guidePathIndex < len(m.guideCategory.Paths)-1 {
			m.guidePathIndex++
		}
	case "o":
		// Open selected path in Finder
		if m.guideCategory != nil && len(m.guideCategory.Paths) > 0 {
			path := m.guideCategory.Paths[m.guidePathIndex]
			// Strip glob patterns to find openable parent directory
			path = utils.StripGlobPattern(path)
			if err := utils.OpenInFinder(path); err != nil {
				m.statusMessage = "Path not found"
			} else {
				m.statusMessage = ""
			}
		}
	}
	return m, nil
}

func (m *Model) startRescanCmd() tea.Cmd {
	m.resetForRescan()
	return tea.Batch(m.spinner.Tick, m.startScan())
}

func (m *Model) resetForRescan() {
	m.view = ViewList
	m.selected = make(map[string]bool)
	m.excluded = make(map[string]map[string]bool)
	m.results = make([]*types.ScanResult, 0)
	m.resultMap = make(map[string]*types.ScanResult)
	m.cursor = 0
	m.scroll = 0
	m.reportScroll = 0
	m.reportLines = nil
	m.scanning = true
}

func (m *Model) resetPreviewSelection() {
	m.previewItemIndex = 0
	m.previewScroll = 0
}

func (m *Model) clearFilter() {
	m.filterState = FilterNone
	m.filterText = ""
	m.filterInput.Blur()
}

func (m *Model) applyFilter() {
	m.filterText = m.filterInput.Value()
	m.filterState = FilterApplied
	m.filterInput.Blur()
	m.resetPreviewSelection()
}

func (m *Model) startFilterTyping() tea.Cmd {
	m.filterState = FilterTyping
	m.filterInput.SetValue("")
	m.filterInput.Focus()
	return m.filterInput.Focus()
}

func (m *Model) maxReportScroll() int {
	// Estimate visible lines (header ~8, footer ~2)
	visible := m.height - 10
	if visible < 5 {
		visible = 5
	}
	maxScroll := len(m.reportLines) - visible
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func (m *Model) movePreviewCursor(delta int) {
	r := m.getPreviewCatResult()
	if r == nil || len(r.Items) == 0 {
		return
	}
	m.setPreviewCursor(m.previewItemIndex+delta, len(r.Items)-1)
}

func (m *Model) setPreviewCursor(index int, max int) {
	if index < 0 {
		index = 0
	}
	if index > max {
		index = max
	}
	m.previewItemIndex = index
	m.previewScroll = 0
}

func (m *Model) moveDrillCursor(state *drillDownState, delta int) {
	if state == nil || len(state.items) == 0 {
		return
	}
	m.setDrillCursor(state, state.cursor+delta, len(state.items)-1)
}

func (m *Model) setDrillCursor(state *drillDownState, index int, max int) {
	if index < 0 {
		index = 0
	}
	if index > max {
		index = max
	}
	state.cursor = index
	state.scroll = 0
}
