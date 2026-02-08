package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/2ykwang/mac-cleanup-go/internal/styles"
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
	if query := m.currentFilterQuery(); query != "" {
		items = m.filterItems(items, query)
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

	b.WriteString(styles.HeaderStyle.Render("Cleanup Preview"))
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("Selected: %d  │  Estimated: %s  │  Sort: %s\n",
		m.getSelectedCount(), styles.SizeStyle.Render(formatSize(m.getSelectedSize())), m.sortOrder.Label()))
	b.WriteString(styles.Divider(60) + "\n")

	// Tabs
	catIdx := m.findSelectedCatIndex()
	b.WriteString(styles.TextStyle.Render("Categories") + "\n")
	b.WriteString(m.renderTabs(selected, catIdx))
	b.WriteString("\n\n")

	// Current category info
	if cat != nil {
		badge := safetyBadge(cat.Category.Safety)
		mBadge := m.methodBadge(cat.Category.Method)
		effectiveSize := m.getEffectiveSize(cat)
		if mBadge != "" {
			b.WriteString(fmt.Sprintf("%s %s  %s  │  %d files\n",
				badge, mBadge, styles.SizeStyle.Render(formatSize(effectiveSize)), cat.TotalFileCount))
		} else {
			b.WriteString(fmt.Sprintf("%s  %s  │  %d files\n",
				badge, styles.SizeStyle.Render(formatSize(effectiveSize)), cat.TotalFileCount))
		}
		if cat.Category.Note != "" {
			// Auto-wrap note text to fit terminal width
			noteStyle := styles.MutedStyle.Width(m.width - 4)
			b.WriteString(noteStyle.Render(cat.Category.Note) + "\n")
		}
		if cat.Category.Method == types.MethodManual && cat.Category.Guide != "" {
			guideStyle := styles.WarningStyle.Width(m.width - 4)
			b.WriteString(guideStyle.Render("[Manual] "+cat.Category.Guide) + "\n")
		}
		b.WriteString(styles.Divider(60) + "\n")
	}

	return b.String()
}

func (m *Model) previewFooter(selected []*types.ScanResult) string {
	var b strings.Builder

	// Warning for risky items
	for _, r := range selected {
		if r.Category.Safety == types.SafetyLevelRisky {
			b.WriteString("\n" + styles.DangerStyle.Render("Warning: Risky items included"))
			break
		}
	}

	// Status message (e.g., error messages)
	if m.statusMessage != "" {
		b.WriteString("\n" + styles.WarningStyle.Render(m.statusMessage))
	}

	b.WriteString("\n\n")

	// Context-specific footer for filter mode
	if m.filterState == FilterTyping {
		b.WriteString(styles.HelpStyle.Render(FormatFooter(FilterTypingShortcuts)))
	} else {
		b.WriteString(m.help.View(PreviewKeyMap))
	}

	return b.String()
}

// itemRowOpts holds parameters for rendering a single item row.
type itemRowOpts struct {
	item       types.CleanableItem
	isCurrent  bool
	isExcluded bool // only relevant when showCheck is true
	isLocked   bool // only relevant when showCheck is true
	showCheck  bool // true = preview mode (checkbox + shortenPath), false = drilldown mode
	pathWidth  int
	sizeWidth  int
	ageWidth   int
}

// renderItemRow renders a single item row for both preview and drilldown views.
func (m *Model) renderItemRow(opts itemRowOpts) string {
	item := opts.item

	cursor := "  "
	if opts.isCurrent {
		cursor = styles.CursorStyle.Render("▸ ")
	}

	checkbox := ""
	if opts.showCheck {
		switch {
		case opts.isLocked:
			checkbox = styles.MutedStyle.Render(" - ")
		case opts.isExcluded:
			checkbox = styles.MutedStyle.Render("[ ]")
		default:
			checkbox = styles.SuccessStyle.Render("[x]")
		}
		checkbox += " "
	}

	icon := " "
	if item.IsDirectory {
		icon = ">"
	}
	if opts.showCheck {
		icon = padToWidth(icon, previewIconWidth)
		if opts.isLocked {
			icon = styles.MutedStyle.Render(icon)
		}
	}

	var paddedName string
	if opts.showCheck {
		displayPath := item.Path
		if item.DisplayName != "" {
			displayPath = item.DisplayName
		}
		if displayPath == item.Path {
			paddedName = shortenPath(displayPath, opts.pathWidth)
		} else {
			paddedName = truncateToWidth(displayPath, opts.pathWidth, false)
		}
	} else {
		paddedName = truncateToWidth(item.Name, opts.pathWidth, false)
	}
	paddedName = padToWidth(paddedName, opts.pathWidth)

	switch {
	case opts.isLocked || opts.isExcluded:
		paddedName = styles.MutedStyle.Render(paddedName)
	case opts.isCurrent:
		paddedName = styles.SelectedStyle.Render(paddedName)
	}

	size := fmt.Sprintf("%*s", opts.sizeWidth, utils.FormatSize(item.Size))
	age := fmt.Sprintf("%*s", opts.ageWidth, utils.FormatAge(item.ModifiedAt))
	if opts.isLocked || opts.isExcluded {
		size = styles.MutedStyle.Render(size)
		age = styles.MutedStyle.Render(age)
	} else {
		size = styles.SizeStyle.Render(size)
		age = styles.MutedStyle.Render(age)
	}

	return fmt.Sprintf("%s%s%s %s %s %s\n", cursor, checkbox, icon, paddedName, size, age)
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
		pathWidth, sizeWidth, ageWidth := m.previewColumnWidths()

		// Show search input if in typing mode
		if m.filterState == FilterTyping {
			b.WriteString("Search: " + m.filterInput.View() + "\n")
		}

		filterQuery := m.currentFilterQuery()
		sortedItems := m.getVisiblePreviewItems()

		// Show filter info
		if filterQuery != "" {
			filterInfo := fmt.Sprintf("Filter: \"%s\" (%d items)", filterQuery, len(sortedItems))
			b.WriteString(styles.MutedStyle.Render(filterInfo) + "\n")
		}

		colHeader := fmt.Sprintf("%*s%-*s %*s %*s",
			previewPrefixWidth, "", pathWidth, "Path", sizeWidth, "Size", ageWidth, "Age")
		b.WriteString(styles.MutedStyle.Render(colHeader) + "\n")

		// Handle empty filter results
		if len(sortedItems) == 0 && filterQuery != "" {
			b.WriteString(styles.MutedStyle.Render("  No matching items\n"))
		}

		// Adjust scroll
		m.previewScroll = adjustScrollFor(m.previewItemIndex, m.previewScroll, visible-1, len(sortedItems))

		endIdx := m.previewScroll + visible
		if endIdx > len(sortedItems) {
			endIdx = len(sortedItems)
		}

		for i := m.previewScroll; i < endIdx; i++ {
			item := sortedItems[i]
			b.WriteString(m.renderItemRow(itemRowOpts{
				item:       item,
				isCurrent:  m.previewItemIndex == i,
				isExcluded: m.isExcluded(cat.Category.ID, item.Path),
				isLocked:   item.Status == types.ItemStatusProcessLocked,
				showCheck:  true,
				pathWidth:  pathWidth,
				sizeWidth:  sizeWidth,
				ageWidth:   ageWidth,
			}))
		}

		if len(sortedItems) > visible {
			b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("\n\n  [%d-%d / %d]", m.previewScroll+1, endIdx, len(sortedItems))))
		}
	}

	b.WriteString(footer)
	return b.String()
}

func (m *Model) renderTabs(selected []*types.ScanResult, currentIdx int) string {
	if len(selected) == 0 {
		return ""
	}

	maxWidth := m.width - 2
	if maxWidth < 10 {
		maxWidth = 10
	}

	if currentIdx < 0 || currentIdx >= len(selected) {
		currentIdx = 0
	}

	type tabItem struct {
		text  string
		width int
	}

	items := make([]tabItem, 0, len(selected))
	for _, r := range selected {
		name := r.Category.Name
		isCurrent := r.Category.ID == m.previewCatID
		tabName := truncateToWidth(name, maxWidth-2, false)
		var tab string
		if isCurrent {
			tab = lipgloss.NewStyle().
				Bold(true).
				Foreground(styles.ColorText).
				Background(styles.ColorPrimary).
				Padding(0, 1).
				Render(tabName)
		} else {
			tab = lipgloss.NewStyle().
				Foreground(styles.ColorText).
				Background(styles.ColorBorder).
				Padding(0, 1).
				Render(tabName)
		}
		items = append(items, tabItem{text: tab, width: lipgloss.Width(tab)})
	}

	joinTabs := func(start, end int, left, right bool) (string, int) {
		parts := make([]string, 0, end-start+3)
		if left {
			parts = append(parts, styles.MutedStyle.Render("…"))
		}
		for i := start; i <= end; i++ {
			parts = append(parts, items[i].text)
		}
		if right {
			parts = append(parts, styles.MutedStyle.Render("…"))
		}
		line := strings.Join(parts, " ")
		return line, lipgloss.Width(line)
	}

	start := currentIdx
	end := currentIdx
	for {
		expanded := false
		if start > 0 {
			_, width := joinTabs(start-1, end, start-1 > 0, end < len(items)-1)
			if width <= maxWidth {
				start--
				expanded = true
			}
		}
		if end < len(items)-1 {
			_, width := joinTabs(start, end+1, start > 0, end+1 < len(items)-1)
			if width <= maxWidth {
				end++
				expanded = true
			}
		}
		if !expanded {
			break
		}
	}

	left := start > 0
	right := end < len(items)-1
	line, width := joinTabs(start, end, left, right)
	for width > maxWidth && start < end {
		if currentIdx-start > end-currentIdx {
			start++
		} else {
			end--
		}
		left = start > 0
		right = end < len(items)-1
		line, width = joinTabs(start, end, left, right)
	}

	return line
}

func (m *Model) drillDownHeader(path string) string {
	var b strings.Builder

	b.WriteString(styles.HeaderStyle.Render("Directory Browser"))
	b.WriteString("\n\n")

	b.WriteString(styles.MutedStyle.Render("Path: ") + shortenPath(path, m.width-10))
	b.WriteString("\n")
	b.WriteString(styles.Divider(60) + "\n")

	return b.String()
}

func (m *Model) drillDownFooter() string {
	var b strings.Builder

	// Status message (e.g., error messages)
	if m.statusMessage != "" {
		b.WriteString("\n" + styles.WarningStyle.Render(m.statusMessage))
	}

	b.WriteString("\n\n")
	b.WriteString(m.help.View(PreviewKeyMap))

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
		b.WriteString(styles.MutedStyle.Render("(empty)") + "\n")
	} else {
		// Sort items based on current sort order
		sortedItems := m.sortItems(state.items)

		// Adjust scroll
		state.scroll = adjustScrollFor(state.cursor, state.scroll, visible, len(sortedItems))

		endIdx := state.scroll + visible
		if endIdx > len(sortedItems) {
			endIdx = len(sortedItems)
		}

		_, sizeWidth, ageWidth := m.previewColumnWidths()
		nameWidth := m.width - (sizeWidth + ageWidth + 6)
		if nameWidth < 20 {
			nameWidth = 20
		}

		for i := state.scroll; i < endIdx; i++ {
			b.WriteString(m.renderItemRow(itemRowOpts{
				item:      sortedItems[i],
				isCurrent: i == state.cursor,
				showCheck: false,
				pathWidth: nameWidth,
				sizeWidth: sizeWidth,
				ageWidth:  ageWidth,
			}))
		}

		if len(sortedItems) > visible {
			b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("\n  [%d/%d]", state.cursor+1, len(sortedItems))))
		}
	}

	b.WriteString(footer)
	return b.String()
}
