package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

func (m *Model) ensurePreviewCategory(selected []*types.ScanResult) {
	if len(selected) == 0 {
		m.previewCatID = ""
		return
	}

	for _, r := range selected {
		if r.Category.ID == m.previewCatID {
			return
		}
	}
	m.previewCatID = selected[0].Category.ID
}

func (m *Model) initializePreviewSections() {
	selected := m.getSelectedResults()
	m.previewCollapsed = make(map[string]bool, len(selected))
	m.ensurePreviewCategory(selected)
	for _, r := range selected {
		m.previewCollapsed[r.Category.ID] = true
	}
	m.previewItemIndex = -1
}

func (m *Model) isSectionCollapsed(catID string) bool {
	if m.previewCollapsed == nil {
		return false
	}
	collapsed, ok := m.previewCollapsed[catID]
	if !ok {
		return catID != m.previewCatID
	}
	return collapsed
}

func (m *Model) collapseCurrentSection() {
	if m.previewCatID == "" {
		return
	}
	if m.previewCollapsed == nil {
		m.previewCollapsed = make(map[string]bool)
	}
	m.previewCollapsed[m.previewCatID] = true
	m.previewItemIndex = -1
	m.previewScroll = 0
}

func (m *Model) expandCurrentSection() {
	if m.previewCatID == "" {
		return
	}
	if m.previewCollapsed == nil {
		m.previewCollapsed = make(map[string]bool)
	}
	m.previewCollapsed[m.previewCatID] = false
}

func (m *Model) toggleCurrentSection() {
	if m.previewCatID == "" {
		return
	}
	if m.isSectionCollapsed(m.previewCatID) {
		m.expandCurrentSection()
		return
	}
	m.collapseCurrentSection()
}

func (m *Model) movePreviewSection(delta int) bool {
	selected := m.getSelectedResults()
	if len(selected) == 0 || delta == 0 {
		return false
	}

	m.ensurePreviewCategory(selected)
	current := 0
	for i, r := range selected {
		if r.Category.ID == m.previewCatID {
			current = i
			break
		}
	}
	next := clamp(current+delta, len(selected)-1)
	if next == current {
		return false
	}

	m.previewCatID = selected[next].Category.ID
	m.previewItemIndex = -1
	return true
}

func (m *Model) previewItemsCount() int {
	return len(m.getVisiblePreviewItems())
}

// getVisiblePreviewItems returns the sorted and filtered items for the current preview.
// This is what the user actually sees on screen.
func (m *Model) getVisiblePreviewItems() []types.CleanableItem {
	return m.getVisiblePreviewItemsFor(m.previewCatID)
}

func (m *Model) getVisiblePreviewItemsFor(catID string) []types.CleanableItem {
	r := m.resultMap[catID]
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
	if m.isSectionCollapsed(m.previewCatID) {
		return nil
	}
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

func (m *Model) previewHeader() string {
	var b strings.Builder

	b.WriteString(styles.HeaderStyle.Render("Cleanup Preview"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("Selected: %d  │  Estimated: %s  │  Sort: %s\n",
		m.getSelectedCount(), styles.SizeStyle.Render(formatSize(m.getSelectedSize())), m.sortOrder.Label()))
	b.WriteString("\n")
	b.WriteString(styles.MutedStyle.Render("Next: press ") + styles.SelectedStyle.Render("Y") + styles.MutedStyle.Render(" to open delete confirmation") + "\n")
	b.WriteString(styles.Divider(60) + "\n")

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

func (m *Model) renderSectionLine(r *types.ScanResult, isCurrentSection bool, isFocused bool) string {
	nameWidth, badgeWidth, sizeWidth, countWidth := m.previewSectionColumnWidths()

	cursor := "  "
	if isFocused {
		cursor = styles.CursorStyle.Render("▸ ")
	}

	indicator := "▶"
	if !m.isSectionCollapsed(r.Category.ID) {
		indicator = "▼"
	}
	if isCurrentSection {
		indicator = styles.CursorStyle.Render(indicator)
	} else {
		indicator = styles.MutedStyle.Render(indicator)
	}

	name := padToWidth(truncateToWidth(r.Category.Name, nameWidth, false), nameWidth)
	if isCurrentSection {
		name = SectionActiveNameStyle.Render(name)
	}

	badgeText := ""
	switch r.Category.Safety {
	case types.SafetyLevelSafe:
		badgeText = "[Safe]"
	case types.SafetyLevelModerate:
		badgeText = "[Moderate]"
	case types.SafetyLevelRisky:
		badgeText = "[Risky]"
	}
	if r.Category.Method == types.MethodManual {
		badgeText += " [Manual]"
	}
	badgeText = padToWidth(truncateToWidth(badgeText, badgeWidth, false), badgeWidth)
	badge := styles.MutedStyle.Render(badgeText)
	switch r.Category.Safety {
	case types.SafetyLevelSafe:
		badge = styles.SuccessStyle.Render(badgeText)
	case types.SafetyLevelModerate:
		badge = styles.WarningStyle.Render(badgeText)
	case types.SafetyLevelRisky:
		badge = styles.DangerStyle.Render(badgeText)
	}

	size := fmt.Sprintf("%*s", sizeWidth, formatSize(m.getEffectiveSize(r)))
	count := fmt.Sprintf("%*s", countWidth, fmt.Sprintf("%d", r.TotalFileCount))

	return fmt.Sprintf("%s%s %s %s %s %s", cursor, indicator, name, badge, styles.SizeStyle.Render(size), styles.MutedStyle.Render(count))
}

func (m *Model) previewSectionColumnWidths() (int, int, int, int) {
	// sectionPrefixWidth: cursor(2) + indicator(1) + space(1)
	const sectionPrefixWidth = 4
	cols := columnWidths(m.width, sectionPrefixWidth, 3, []int{44, 22, 10, 10}, false)
	return cols[0], cols[1], cols[2], cols[3]
}

func (m *Model) renderSectionColumnsHeader() string {
	nameWidth, badgeWidth, sizeWidth, countWidth := m.previewSectionColumnWidths()
	name := padToWidth("Section", nameWidth)
	badge := padToWidth("Risk/Mode", badgeWidth)
	size := fmt.Sprintf("%*s", sizeWidth, "Size")
	count := fmt.Sprintf("%*s", countWidth, "Files")
	return styles.MutedStyle.Render(fmt.Sprintf("    %s %s %s %s", name, badge, size, count))
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

func (m *Model) renderPreviewItemLine(catID string, item types.CleanableItem, isCurrent bool, pathWidth, sizeWidth, ageWidth int) string {
	isExcluded := m.isExcluded(catID, item.Path)
	isLocked := item.Status == types.ItemStatusProcessLocked

	cursor := "  "
	if isCurrent {
		cursor = styles.CursorStyle.Render("▸ ")
	}

	checkbox := styles.SuccessStyle.Render("[x]")
	if isLocked {
		checkbox = styles.MutedStyle.Render(" - ")
	} else if isExcluded {
		checkbox = styles.MutedStyle.Render("[ ]")
	}

	icon := " "
	if item.IsDirectory {
		icon = ">"
	}
	icon = padToWidth(icon, previewIconWidth)
	if isLocked {
		icon = styles.MutedStyle.Render(icon)
	}

	displayPath := item.Path
	if item.DisplayName != "" {
		displayPath = item.DisplayName
	}

	var truncated string
	if displayPath == item.Path {
		truncated = shortenPath(displayPath, pathWidth)
	} else {
		truncated = truncateToWidth(displayPath, pathWidth, false)
	}

	paddedPath := padToWidth(truncated, pathWidth)
	if isLocked || isExcluded {
		paddedPath = styles.MutedStyle.Render(paddedPath)
	} else if isCurrent {
		paddedPath = styles.SelectedStyle.Render(paddedPath)
	}

	size := fmt.Sprintf("%*s", sizeWidth, utils.FormatSize(item.Size))
	age := fmt.Sprintf("%*s", ageWidth, utils.FormatAge(item.ModifiedAt))
	if isLocked || isExcluded {
		size = styles.MutedStyle.Render(size)
		age = styles.MutedStyle.Render(age)
	} else {
		size = styles.SizeStyle.Render(size)
		age = styles.MutedStyle.Render(age)
	}

	return fmt.Sprintf("%s%s %s %s %s %s", cursor, checkbox, icon, paddedPath, size, age)
}

func (m *Model) viewPreview() string {
	if len(m.drillDownStack) > 0 {
		return m.viewDrillDown()
	}

	selected := m.getSelectedResults()
	if len(selected) == 0 {
		return "No items selected."
	}
	m.ensurePreviewCategory(selected)
	if len(m.previewCollapsed) == 0 {
		m.initializePreviewSections()
	}

	header := m.previewHeader()
	footer := m.previewFooter(selected)
	visible := m.availableLines(header, footer)

	pathWidth, sizeWidth, ageWidth := m.previewColumnWidths()
	filterQuery := m.currentFilterQuery()

	bodyLines := make([]string, 0, 64)
	focusLine := -1
	addLine := func(line string, focused bool) {
		bodyLines = append(bodyLines, line)
		if focused {
			focusLine = len(bodyLines) - 1
		}
	}

	if m.filterState == FilterTyping {
		addLine("Search: "+m.filterInput.View(), false)
	}
	if filterQuery != "" {
		addLine(styles.MutedStyle.Render(fmt.Sprintf("Filter: \"%s\"", filterQuery)), false)
	}
	addLine(m.renderSectionColumnsHeader(), false)

	for _, r := range selected {
		isCurrentSection := r.Category.ID == m.previewCatID
		if isCurrentSection && m.isSectionCollapsed(r.Category.ID) {
			m.previewItemIndex = -1
		}

		sectionFocus := isCurrentSection && (m.previewItemIndex < 0 || m.isSectionCollapsed(r.Category.ID))
		addLine(m.renderSectionLine(r, isCurrentSection, sectionFocus), sectionFocus)

		if m.isSectionCollapsed(r.Category.ID) {
			continue
		}

		items := m.getVisiblePreviewItemsFor(r.Category.ID)
		if isCurrentSection {
			if len(items) == 0 {
				m.previewItemIndex = -1
			} else if m.previewItemIndex >= len(items) {
				m.previewItemIndex = len(items) - 1
			}
		}

		if len(items) == 0 {
			addLine(styles.MutedStyle.Render("    (no items)"), false)
			continue
		}

		for itemIdx, item := range items {
			isCurrentItem := isCurrentSection && m.previewItemIndex == itemIdx
			addLine("  "+m.renderPreviewItemLine(r.Category.ID, item, isCurrentItem, pathWidth, sizeWidth, ageWidth), isCurrentItem)
		}
	}

	if focusLine < 0 {
		focusLine = 0
	}

	m.previewScroll = adjustScrollFor(focusLine, m.previewScroll, visible, len(bodyLines))
	start := m.previewScroll
	end := start + visible
	if end > len(bodyLines) {
		end = len(bodyLines)
	}

	var b strings.Builder
	b.WriteString(header)
	for i := start; i < end; i++ {
		b.WriteString(bodyLines[i])
		b.WriteString("\n")
	}
	if len(bodyLines) > visible {
		b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("  … [%d-%d / %d]\n", start+1, end, len(bodyLines))))
	}
	m.updatePreviewStatusMessage()
	b.WriteString(footer)
	return b.String()
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
