package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/2ykwang/mac-cleanup-go/internal/styles"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

// GroupStat represents aggregated size statistics for a group
type GroupStat struct {
	Name string
	Size int64
}

// getGroupStats aggregates scan results by group, sorted by size descending
func (m *Model) getGroupStats() []GroupStat {
	if len(m.results) == 0 {
		return nil
	}

	// Build group ID -> name map
	groupNames := make(map[string]string)
	if m.config != nil {
		for _, g := range m.config.Groups {
			groupNames[g.ID] = g.Name
		}
	}

	// Aggregate sizes by group
	groupSizes := make(map[string]int64)
	for _, r := range m.results {
		groupSizes[r.Category.Group] += r.TotalSize
	}

	// Filter zero-byte groups and convert to slice
	var stats []GroupStat
	for groupID, size := range groupSizes {
		if size > 0 {
			name := groupNames[groupID]
			if name == "" {
				name = groupID // fallback to ID if name not found
			}
			stats = append(stats, GroupStat{Name: name, Size: size})
		}
	}

	// Sort by size descending
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Size > stats[j].Size
	})

	return stats
}

// formatGroupStats formats group statistics as a single line string
func formatGroupStats(stats []GroupStat) string {
	if len(stats) == 0 {
		return ""
	}

	var parts []string
	for _, s := range stats {
		parts = append(parts, fmt.Sprintf("%s: %s", s.Name, formatSize(s.Size)))
	}
	return styles.MutedStyle.Render(strings.Join(parts, "  "))
}

func (m *Model) listHeader(showSummary bool) string {
	var b strings.Builder

	b.WriteString(styles.HeaderStyle.Render("Mac Cleanup"))
	if m.scanning {
		b.WriteString(fmt.Sprintf("  %s Scanning... (%d/%d available, %d total)",
			m.spinner.View(), m.scanCompleted, m.scanTotal, m.scanRegistered))
	}
	b.WriteString("\n")

	// Update available notification
	if m.updateAvailable && m.latestVersion != "" {
		updateMsg := fmt.Sprintf("[↑] Update available: %s → %s (run with --update)",
			m.currentVersion, m.latestVersion)
		b.WriteString(styles.SuccessStyle.Render(updateMsg))
		b.WriteString("\n")
	}

	// Permission warning
	if !m.hasFullDiskAccess {
		b.WriteString(styles.WarningStyle.Render("[!] Limited access: Grant Full Disk Access in System Settings for complete scan"))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Legend
	b.WriteString(fmt.Sprintf("%s Safe      %s\n",
		styles.SuccessStyle.Render("●"), styles.MutedStyle.Render("Auto-regenerated caches")))
	b.WriteString(fmt.Sprintf("%s Moderate  %s\n",
		styles.WarningStyle.Render("●"), styles.MutedStyle.Render("May need re-download or re-login")))
	b.WriteString(fmt.Sprintf("%s Risky     %s\n",
		styles.DangerStyle.Render("●"), styles.MutedStyle.Render("May contain important data")))
	b.WriteString("\n")

	// Summary
	if showSummary {
		summary := fmt.Sprintf("Available: %s", styles.SizeStyle.Render(formatSize(m.getAvailableSize())))
		if m.hasSelection() {
			summary += fmt.Sprintf("  │  Selected: %s (%d)",
				styles.SizeStyle.Render(formatSize(m.getSelectedSize())), m.getSelectedCount())
		}
		b.WriteString(summary + "\n")
	}

	// Group statistics
	if stats := m.getGroupStats(); len(stats) > 0 {
		b.WriteString(formatGroupStats(stats) + "\n")
	}

	b.WriteString(styles.Divider(60) + "\n")

	return b.String()
}

func (m *Model) listFooter(includeHelp bool) string {
	var b strings.Builder

	// Show scan warnings after scan completes
	if !m.scanning && len(m.scanErrors) > 0 {
		b.WriteString(styles.WarningStyle.Render("[!] Scan warnings:"))
		b.WriteString("\n")
		for _, err := range m.scanErrors {
			errMsg := err.Error
			if len(errMsg) > 50 {
				errMsg = errMsg[:47] + "..."
			}
			b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("    %s: %s", err.CategoryName, errMsg)))
			b.WriteString("\n")
		}
	}

	if includeHelp {
		b.WriteString("\n")
		b.WriteString(m.help.View(ListKeyMap))
	}

	return b.String()
}

func (m *Model) viewList() string {
	showSidePanel := false
	bodyWidth := m.width
	sideWidth := 0
	if m.width >= 100 {
		sideWidth = min(32, m.width/3)
		if m.width-sideWidth-2 >= 60 {
			bodyWidth = m.width - sideWidth - 2
			showSidePanel = true
		} else {
			sideWidth = 0
		}
	}

	header := m.listHeader(!showSidePanel)
	showHelpInFooter := !showSidePanel
	footer := m.listFooter(showHelpInFooter)
	helpContent := ""
	helpLines := 0
	if !showHelpInFooter {
		helpContent = m.help.View(ListKeyMap)
		if helpContent != "" {
			helpLines = countLines(helpContent) + 1
		}
	}
	visible := m.visibleLines(header, footer, helpLines)
	if visible < 16 {
		showSidePanel = false
		showHelpInFooter = true
		footer = m.listFooter(showHelpInFooter)
		helpContent = ""
		visible = m.visibleLines(header, footer, 0)
	}

	listContent := m.renderListBody(visible)
	if showSidePanel {
		if helpContent != "" {
			listContent = strings.TrimRight(listContent, "\n") + "\n\n" + helpContent
		}
		sideContent := m.renderListSidePanel(sideWidth)
		sideHeight := min(visible, lipgloss.Height(sideContent)+2)
		if sideHeight < 1 {
			sideHeight = 1
		}
		nameWidth, sizeWidth, countWidth := m.listColumnWidths()
		listContentWidth := listPrefixWidth + nameWidth + sizeWidth + countWidth + 2
		gapWidth := 8
		listWidth := listContentWidth
		if listWidth > bodyWidth {
			listWidth = bodyWidth
		}
		if listWidth < 1 {
			listWidth = 1
		}
		listStyle := lipgloss.NewStyle().Width(listWidth)
		sideStyle := lipgloss.NewStyle().
			Width(sideWidth).
			Height(sideHeight).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.ColorBorder).
			Padding(0, 1)
		spacer := strings.Repeat(" ", gapWidth)
		listContent = lipgloss.JoinHorizontal(lipgloss.Top, listStyle.Render(listContent), spacer, sideStyle.Render(sideContent))
	}

	var b strings.Builder
	b.WriteString(header)
	b.WriteString(listContent)
	if listContent != "" && footer != "" {
		b.WriteString("\n")
	}
	b.WriteString(footer)
	return b.String()
}

func (m *Model) renderListItem(idx int, r *types.ScanResult, nameWidth, sizeWidth, countWidth int) string {
	isCurrent := idx == m.cursor
	isManual := r.Category.Method == types.MethodManual

	cursor := "  "
	if isCurrent {
		cursor = styles.CursorStyle.Render("▸ ")
	}

	checkbox := "[ ]"
	if isManual {
		// Manual items cannot be selected - always show muted unchecked box
		checkbox = styles.MutedStyle.Render(" - ")
	} else if m.selected[r.Category.ID] {
		checkbox = styles.SuccessStyle.Render("[✓]")
	}

	dot := safetyDot(r.Category.Safety)

	name := r.Category.Name

	switch r.Category.Method {
	case types.MethodManual:
		name += " [Manual]"
	}
	// Truncate and pad using display width for consistent alignment
	name = padToWidth(truncateToWidth(name, nameWidth, false), nameWidth)
	if isManual {
		name = styles.MutedStyle.Render(name)
	} else if isCurrent {
		name = styles.SelectedStyle.Render(name)
	}

	sizeText := utils.FormatSize(r.TotalSize)
	countText := fmt.Sprintf("%d", r.TotalFileCount)
	if m.scanning && r.TotalSize == 0 && r.TotalFileCount == 0 && len(r.Items) == 0 {
		sizeText = "-"
		countText = "-"
	}

	size := fmt.Sprintf("%*s", sizeWidth, sizeText)
	count := fmt.Sprintf("%*s", countWidth, countText)

	if isManual {
		size = styles.MutedStyle.Render(size)
		count = styles.MutedStyle.Render(count)
	} else {
		size = styles.SizeStyle.Render(size)
		count = styles.MutedStyle.Render(count)
	}

	return fmt.Sprintf("%s%s %s %s %s %s\n",
		cursor, checkbox, dot, name, size, count)
}

func (m *Model) renderListBody(visible int) string {
	var b strings.Builder

	if visible < 1 {
		return ""
	}

	if len(m.results) == 0 {
		if m.scanning {
			b.WriteString(styles.MutedStyle.Render("Scanning..."))
		} else {
			b.WriteString(styles.MutedStyle.Render("No items to clean."))
		}
		b.WriteString("\n")
		return b.String()
	}

	linesRemaining := visible
	nameWidth, sizeWidth, countWidth := m.listColumnWidths()
	if visible >= 2 {
		colHeader := fmt.Sprintf("%*s%-*s %*s %*s",
			listPrefixWidth, "",
			nameWidth, "Name", sizeWidth, "Size", countWidth, "Count")
		b.WriteString(styles.MutedStyle.Render(colHeader) + "\n")
		linesRemaining--
	}

	showPager := false
	itemsVisible := linesRemaining
	if len(m.results) > itemsVisible && itemsVisible > 0 {
		showPager = true
		itemsVisible--
	}

	if itemsVisible > 0 {
		// Adjust scroll
		m.scroll = adjustScrollFor(m.cursor, m.scroll, itemsVisible, len(m.results))

		for i, r := range m.results {
			if i < m.scroll || i >= m.scroll+itemsVisible {
				continue
			}
			b.WriteString(m.renderListItem(i, r, nameWidth, sizeWidth, countWidth))
		}
	}

	if showPager {
		b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("\n  [%d/%d]", m.cursor+1, len(m.results))))
	}
	return b.String()
}

func (m *Model) renderListSidePanel(width int) string {
	var b strings.Builder

	b.WriteString(styles.HeaderStyle.Render("Summary"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s %s", styles.MutedStyle.Render("Available:"), styles.SizeStyle.Render(formatSize(m.getAvailableSize()))))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s %s (%s)",
		styles.MutedStyle.Render("Selected:"),
		styles.SizeStyle.Render(formatSize(m.getSelectedSize())),
		styles.TextStyle.Render(fmt.Sprintf("%d", m.getSelectedCount())),
	))
	b.WriteString("\n")
	if m.hasSelection() {
		b.WriteString("\n")
		b.WriteString(styles.MutedStyle.Render("Selected Items"))
		b.WriteString("\n")
		b.WriteString(m.renderSelectedMiniList(width))
		b.WriteString("\n")
	}
	b.WriteString(styles.Divider(min(width-2, 30)))
	return b.String()
}

func (m *Model) renderSelectedMiniList(width int) string {
	selected := m.getSelectedResults()
	if len(selected) == 0 {
		return ""
	}

	contentWidth := width - 2
	if contentWidth < 10 {
		contentWidth = 10
	}

	var b strings.Builder
	limit := 6
	for i, r := range selected {
		if i >= limit {
			break
		}
		sizeStr := formatSize(r.TotalSize)
		nameWidth := contentWidth - lipgloss.Width(sizeStr) - 3
		if nameWidth < 8 {
			nameWidth = 8
		}
		name := truncateToWidth(r.Category.Name, nameWidth, false)
		name = padToWidth(name, nameWidth)
		dot := safetyDot(r.Category.Safety)
		b.WriteString(fmt.Sprintf("%s %s %s\n", dot, name, styles.SizeStyle.Render(sizeStr)))
	}

	if len(selected) > limit {
		b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("+%d more", len(selected)-limit)))
	}

	return strings.TrimRight(b.String(), "\n")
}
