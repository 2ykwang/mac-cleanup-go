package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

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

// readDirectory reads directory contents for drill-down view
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

// View rendering

func (m *Model) previewHeader(selected []*types.ScanResult, cat *types.ScanResult) string {
	var b strings.Builder

	b.WriteString(HeaderStyle.Render("Cleanup Preview"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("Selected: %d  │  Estimated: %s  │  Sort: %s\n",
		m.getSelectedCount(), SizeStyle.Render(formatSize(m.getSelectedSize())), m.sortOrder.Label()))
	b.WriteString(Divider(60) + "\n\n")

	// Tabs
	catIdx := m.findSelectedCatIndex()
	b.WriteString(m.renderTabs(selected, catIdx))
	b.WriteString("\n\n")

	// Current category info
	if cat != nil {
		badge := safetyBadge(cat.Category.Safety)
		mBadge := m.methodBadge(cat.Category.Method)
		effectiveSize := m.getEffectiveSize(cat)
		if mBadge != "" {
			b.WriteString(fmt.Sprintf("%s %s  %s  │  %d files\n",
				badge, mBadge, SizeStyle.Render(formatSize(effectiveSize)), cat.TotalFileCount))
		} else {
			b.WriteString(fmt.Sprintf("%s  %s  │  %d files\n",
				badge, SizeStyle.Render(formatSize(effectiveSize)), cat.TotalFileCount))
		}
		if cat.Category.Note != "" {
			// Auto-wrap note text to fit terminal width
			noteStyle := MutedStyle.Width(m.width - 4)
			b.WriteString(noteStyle.Render(cat.Category.Note) + "\n")
		}
		if cat.Category.Method == types.MethodManual && cat.Category.Guide != "" {
			guideStyle := WarningStyle.Width(m.width - 4)
			b.WriteString(guideStyle.Render("[Manual] "+cat.Category.Guide) + "\n")
		}
		b.WriteString(Divider(60) + "\n")
	}

	return b.String()
}

func (m *Model) previewFooter(selected []*types.ScanResult) string {
	var b strings.Builder

	// Warning for risky items
	for _, r := range selected {
		if r.Category.Safety == types.SafetyLevelRisky {
			b.WriteString("\n" + DangerStyle.Render("Warning: Risky items included"))
			break
		}
	}

	// Status message (e.g., error messages)
	if m.statusMessage != "" {
		b.WriteString("\n" + WarningStyle.Render(m.statusMessage))
	}

	b.WriteString("\n\n")

	// Context-specific footer for filter mode
	if m.filterState == FilterTyping {
		b.WriteString(HelpStyle.Render(FormatFooter(FilterTypingShortcuts)))
	} else {
		b.WriteString(HelpStyle.Render(FormatFooter(FooterShortcuts(ViewPreview))))
	}

	return b.String()
}

func (m *Model) viewPreview() string {
	if len(m.drillDownStack) > 0 {
		return m.viewDrillDown()
	}

	selected := m.getSelectedResults()
	if len(selected) == 0 {
		return "No items selected."
	}

	cat := m.getPreviewCatResult()
	header := m.previewHeader(selected, cat)
	footer := m.previewFooter(selected)
	visible := m.availableLines(header, footer)

	var b strings.Builder
	b.WriteString(header)

	if cat != nil {
		pathWidth := m.width - 28
		if pathWidth < 30 {
			pathWidth = 30
		}

		// Show search input if in typing mode
		// Show search input if in typing mode
		if m.filterState == FilterTyping {
			b.WriteString("Search: " + m.filterInput.View() + "\n")
		}

		// Apply filter and sort (real-time filtering during typing)
		items := cat.Items
		var filterQuery string
		switch m.filterState {
		case FilterTyping:
			filterQuery = m.filterInput.Value()
		case FilterApplied:
			filterQuery = m.filterText
		}
		if filterQuery != "" {
			items = m.filterItems(items, filterQuery)
		}
		sortedItems := m.sortItems(items)

		// Show filter info
		if filterQuery != "" {
			filterInfo := fmt.Sprintf("Filter: \"%s\" (%d items)", filterQuery, len(sortedItems))
			b.WriteString(MutedStyle.Render(filterInfo) + "\n")
		}

		colHeader := fmt.Sprintf("%*s%-*s %*s %*s",
			previewPrefixWidth, "", pathWidth-4, "Path", colSize, "Size", colAge, "Age")
		b.WriteString(MutedStyle.Render(colHeader) + "\n")

		// Handle empty filter results
		if len(sortedItems) == 0 && filterQuery != "" {
			b.WriteString(MutedStyle.Render("  No matching items\n"))
		}

		// Adjust scroll
		m.previewScroll = m.adjustScrollFor(m.previewItemIndex, m.previewScroll, visible-1, len(sortedItems))

		endIdx := m.previewScroll + visible
		if endIdx > len(sortedItems) {
			endIdx = len(sortedItems)
		}

		for i := m.previewScroll; i < endIdx; i++ {
			item := sortedItems[i]
			isCurrent := m.previewItemIndex == i
			isExcluded := m.isExcluded(cat.Category.ID, item.Path)

			cursor := "  "
			if isCurrent {
				cursor = CursorStyle.Render("▸ ")
			}

			checkbox := SuccessStyle.Render("[x]")
			if isExcluded {
				checkbox = MutedStyle.Render("[ ]")
			}

			icon := " "
			if item.IsDirectory {
				icon = ">"
			}

			// Pad using display width (not byte count) for CJK character alignment
			paddedPath := padToWidth(shortenPath(item.Path, pathWidth-4), pathWidth-4)
			if isExcluded {
				paddedPath = MutedStyle.Render(paddedPath)
			} else if isCurrent {
				paddedPath = SelectedStyle.Render(paddedPath)
			}

			size := fmt.Sprintf("%*s", colSize, utils.FormatSize(item.Size))
			age := fmt.Sprintf("%*s", colAge, utils.FormatAge(item.ModifiedAt))
			if isExcluded {
				size = MutedStyle.Render(size)
				age = MutedStyle.Render(age)
			} else {
				size = SizeStyle.Render(size)
				age = MutedStyle.Render(age)
			}

			b.WriteString(fmt.Sprintf("%s%s %s %s %s %s\n", cursor, checkbox, icon, paddedPath, size, age))
		}

		if len(sortedItems) > visible {
			b.WriteString(MutedStyle.Render(fmt.Sprintf("\n  [%d-%d / %d]", m.previewScroll+1, endIdx, len(sortedItems))))
		}
	}

	b.WriteString(footer)
	return b.String()
}

func (m *Model) renderTabs(selected []*types.ScanResult, _ int) string {
	var tabs []string

	for _, r := range selected {
		name := r.Category.Name
		isCurrent := r.Category.ID == m.previewCatID
		if isCurrent {
			tab := lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorText).
				Background(ColorPrimary).
				Padding(0, 1).
				Render(name)
			tabs = append(tabs, tab)
		} else {
			tab := lipgloss.NewStyle().
				Foreground(ColorMuted).
				Padding(0, 1).
				Render(name)
			tabs = append(tabs, tab)
		}
	}

	return strings.Join(tabs, " ")
}

func (m *Model) drillDownHeader(path string) string {
	var b strings.Builder

	b.WriteString(HeaderStyle.Render("Directory Browser"))
	b.WriteString("\n\n")

	b.WriteString(MutedStyle.Render("Path: ") + shortenPath(path, m.width-10))
	b.WriteString("\n")
	b.WriteString(Divider(60) + "\n")

	return b.String()
}

func (m *Model) drillDownFooter() string {
	var b strings.Builder

	// Status message (e.g., error messages)
	if m.statusMessage != "" {
		b.WriteString("\n" + WarningStyle.Render(m.statusMessage))
	}

	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render(FormatFooter(DrillDownFooterShortcuts())))

	return b.String()
}

func (m *Model) viewDrillDown() string {
	if len(m.drillDownStack) == 0 {
		return ""
	}

	state := &m.drillDownStack[len(m.drillDownStack)-1]
	header := m.drillDownHeader(state.path)
	footer := m.drillDownFooter()
	visible := m.availableLines(header, footer)

	var b strings.Builder
	b.WriteString(header)

	if len(state.items) == 0 {
		b.WriteString(MutedStyle.Render("(empty)") + "\n")
	} else {
		// Sort items based on current sort order
		sortedItems := m.sortItems(state.items)

		// Adjust scroll
		state.scroll = m.adjustScrollFor(state.cursor, state.scroll, visible, len(sortedItems))

		endIdx := state.scroll + visible
		if endIdx > len(sortedItems) {
			endIdx = len(sortedItems)
		}

		nameWidth := m.width - 20
		if nameWidth < 20 {
			nameWidth = 20
		}

		for i := state.scroll; i < endIdx; i++ {
			item := sortedItems[i]
			isCurrent := i == state.cursor

			cursor := "  "
			if isCurrent {
				cursor = CursorStyle.Render("▸ ")
			}

			icon := " "
			if item.IsDirectory {
				icon = ">"
			}

			// Truncate and pad using display width for CJK character alignment
			name := truncateToWidth(item.Name, nameWidth, false)
			paddedName := padToWidth(name, nameWidth)
			if isCurrent {
				paddedName = SelectedStyle.Render(paddedName)
			}

			size := fmt.Sprintf("%*s", colSize, utils.FormatSize(item.Size))
			age := fmt.Sprintf("%*s", colAge, utils.FormatAge(item.ModifiedAt))
			b.WriteString(fmt.Sprintf("%s%s %s %s %s\n", cursor, icon, paddedName, SizeStyle.Render(size), MutedStyle.Render(age)))
		}

		if len(sortedItems) > visible {
			b.WriteString(MutedStyle.Render(fmt.Sprintf("\n  [%d/%d]", state.cursor+1, len(sortedItems))))
		}
	}

	b.WriteString(footer)
	return b.String()
}
