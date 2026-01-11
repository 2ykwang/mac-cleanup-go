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
		if m.reportScroll > 0 {
			m.reportScroll--
		}
	case "down", "j":
		// Estimate visible lines (header ~8, footer ~2)
		visible := m.height - 10
		if visible < 5 {
			visible = 5
		}
		maxScroll := len(m.reportLines) - visible
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.reportScroll < maxScroll {
			m.reportScroll++
		}
	case "enter", " ":
		// Return to main screen and rescan
		m.resetForRescan()
		return m, tea.Batch(m.spinner.Tick, m.startScan())
	}
	return m, nil
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
			m.filterState = FilterNone
			m.filterText = ""
			m.previewItemIndex = 0
			m.previewScroll = 0
			return m, nil
		}
		m.view = ViewList
	case "up", "k":
		if m.previewItemIndex > 0 {
			m.previewItemIndex--
		}
	case "down", "j":
		r := m.getPreviewCatResult()
		if r != nil {
			maxItem := len(r.Items) - 1
			if m.previewItemIndex < maxItem {
				m.previewItemIndex++
			}
		}
	case "left", "h":
		prevID := m.findPrevSelectedCatID()
		if prevID != m.previewCatID {
			m.previewCatID = prevID
			m.previewItemIndex = 0
			m.previewScroll = 0
			// Clear filter on tab change
			m.filterState = FilterNone
			m.filterText = ""
		}
	case "right", "l":
		nextID := m.findNextSelectedCatID()
		if nextID != m.previewCatID {
			m.previewCatID = nextID
			m.previewItemIndex = 0
			m.previewScroll = 0
			// Clear filter on tab change
			m.filterState = FilterNone
			m.filterText = ""
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
		r := m.getPreviewCatResult()
		if r != nil {
			m.clearExcludeCategory(r.Category.ID)
		}
	case "d", "D":
		// Exclude all items in current category
		r := m.getPreviewCatResult()
		if r != nil {
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
		m.previewItemIndex = 0
		m.previewScroll = 0
	case "pgdown":
		// Page down
		r := m.getPreviewCatResult()
		if r != nil {
			m.previewItemIndex += m.pageSize()
			if m.previewItemIndex >= len(r.Items) {
				m.previewItemIndex = len(r.Items) - 1
			}
		}
	case "pgup":
		// Page up
		m.previewItemIndex -= m.pageSize()
		if m.previewItemIndex < 0 {
			m.previewItemIndex = 0
		}
	case "home":
		// Go to first item
		m.previewItemIndex = 0
		m.previewScroll = 0
	case "end":
		// Go to last item
		r := m.getPreviewCatResult()
		if r != nil && len(r.Items) > 0 {
			m.previewItemIndex = len(r.Items) - 1
		}
	case "/":
		// Enter search mode
		m.filterState = FilterTyping
		m.filterInput.SetValue("")
		m.filterInput.Focus()
		return m, m.filterInput.Focus()
	}
	return m, nil
}

func (m *Model) handleFilterTypingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Apply the filter
		m.filterText = m.filterInput.Value()
		m.filterState = FilterApplied
		m.filterInput.Blur()
		m.previewItemIndex = 0
		m.previewScroll = 0
		return m, nil
	case "esc":
		// Cancel search
		m.filterState = FilterNone
		m.filterText = ""
		m.filterInput.Blur()
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	}

	// Pass other keys to textinput and reset cursor for real-time filtering
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	// Reset cursor position when filter changes
	m.previewItemIndex = 0
	m.previewScroll = 0
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
		if state.cursor > 0 {
			state.cursor--
		}
	case "down", "j":
		if state.cursor < len(state.items)-1 {
			state.cursor++
		}
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
		state.cursor += m.pageSize()
		if state.cursor >= len(state.items) {
			state.cursor = len(state.items) - 1
		}
	case "pgup":
		// Page up
		state.cursor -= m.pageSize()
		if state.cursor < 0 {
			state.cursor = 0
		}
	case "home":
		// Go to first item
		state.cursor = 0
		state.scroll = 0
	case "end":
		// Go to last item
		if len(state.items) > 0 {
			state.cursor = len(state.items) - 1
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
