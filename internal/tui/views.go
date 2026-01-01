package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"mac-cleanup-go/pkg/types"
	"mac-cleanup-go/pkg/utils"
)

const (
	colName = 28
	colSize = 12
	colNum  = 8
)

func (m *Model) viewList() string {
	var b strings.Builder

	b.WriteString(HeaderStyle.Render("Mac Cleanup"))
	if m.scanning {
		b.WriteString(fmt.Sprintf("  %s Scanning... (%d/%d available, %d total)",
			m.spinner.View(), m.scanCompleted, m.scanTotal, m.scanRegistered))
	}
	b.WriteString("\n")

	// Permission warning
	if !m.hasFullDiskAccess {
		b.WriteString(WarningStyle.Render("[!] Limited access: Grant Full Disk Access in System Settings for complete scan"))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Legend
	b.WriteString(fmt.Sprintf("%s Safe      %s\n",
		SuccessStyle.Render("●"), MutedStyle.Render("Auto-regenerated caches")))
	b.WriteString(fmt.Sprintf("%s Moderate  %s\n",
		WarningStyle.Render("●"), MutedStyle.Render("May need re-download or re-login")))
	b.WriteString(fmt.Sprintf("%s Risky     %s\n",
		DangerStyle.Render("●"), MutedStyle.Render("May contain important data")))
	b.WriteString("\n")

	// Summary
	var totalSize int64
	for _, r := range m.results {
		totalSize += r.TotalSize
	}

	summary := fmt.Sprintf("Available: %s", SizeStyle.Render(formatSize(totalSize)))
	if m.hasSelection() {
		summary += fmt.Sprintf("  │  Selected: %s (%d)",
			SizeStyle.Render(formatSize(m.getSelectedSize())), m.getSelectedCount())
	}
	b.WriteString(summary + "\n")
	b.WriteString(Divider(60) + "\n\n")

	// Items
	if len(m.results) == 0 {
		if m.scanning {
			b.WriteString(MutedStyle.Render("Scanning..."))
		} else {
			b.WriteString(MutedStyle.Render("No items to clean."))
		}
		b.WriteString("\n")
	} else {
		visible := m.visibleLines()
		for i, r := range m.results {
			if i < m.scroll || i >= m.scroll+visible {
				continue
			}
			b.WriteString(m.renderListItem(i, r))
		}
		if len(m.results) > visible {
			b.WriteString(MutedStyle.Render(fmt.Sprintf("\n  [%d/%d]", m.cursor+1, len(m.results))))
		}
	}

	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("↑↓ Navigate  space Select  a Select All  d Deselect All  enter Preview  q Quit"))
	return b.String()
}

func (m *Model) renderListItem(idx int, r *types.ScanResult) string {
	isCurrent := idx == m.cursor

	cursor := "  "
	if isCurrent {
		cursor = CursorStyle.Render("▸ ")
	}

	checkbox := "[ ]"
	if m.selected[r.Category.ID] {
		checkbox = SuccessStyle.Render("[✓]")
	}

	dot := safetyDot(r.Category.Safety)

	name := r.Category.Name
	// Add method badge for non-permanent methods
	switch r.Category.Method {
	case types.MethodManual:
		name += " [Manual]"
	case types.MethodTrash:
		name += " [Trash]"
	}
	name = fmt.Sprintf("%-*s", colName, name)
	if isCurrent {
		name = SelectedStyle.Render(name)
	}

	size := fmt.Sprintf("%*s", colSize, utils.FormatSize(r.TotalSize))
	count := fmt.Sprintf("%*s", colNum, fmt.Sprintf("(%d)", len(r.Items)))

	return fmt.Sprintf("%s%s %s %s %s %s\n",
		cursor, checkbox, dot, name, SizeStyle.Render(size), MutedStyle.Render(count))
}

func (m *Model) viewPreview() string {
	if len(m.drillDownStack) > 0 {
		return m.viewDrillDown()
	}

	selected := m.getSelectedResults()
	if len(selected) == 0 {
		return "No items selected."
	}

	var b strings.Builder

	b.WriteString(HeaderStyle.Render("Cleanup Preview"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("Selected: %d  │  Estimated: %s\n",
		m.getSelectedCount(), SizeStyle.Render(formatSize(m.getSelectedSize()))))
	b.WriteString(Divider(60) + "\n\n")

	// Tabs (no truncation, wrap if needed)
	catIdx := m.findSelectedCatIndex()
	b.WriteString(m.renderTabs(selected, catIdx))
	b.WriteString("\n\n")

	// Current category content
	cat := m.getPreviewCatResult()
	if cat != nil {

		badge := safetyBadge(cat.Category.Safety)
		methodBadge := methodBadge(cat.Category.Method)
		effectiveSize := m.getEffectiveSize(cat)
		b.WriteString(fmt.Sprintf("%s %s  %s  │  %d items\n",
			badge, methodBadge, SizeStyle.Render(formatSize(effectiveSize)), len(cat.Items)))
		if cat.Category.Note != "" {
			b.WriteString(MutedStyle.Render(cat.Category.Note) + "\n")
		}
		// Show guide for manual method
		if cat.Category.Method == types.MethodManual && cat.Category.Guide != "" {
			b.WriteString(WarningStyle.Render("[Manual] "+cat.Category.Guide) + "\n")
		}
		b.WriteString(Divider(60) + "\n")

		visible := m.height - 16
		if visible < 3 {
			visible = 3
		}

		// Adjust scroll
		maxScroll := len(cat.Items) - visible
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.previewScroll > maxScroll {
			m.previewScroll = maxScroll
		}
		if m.previewScroll < 0 {
			m.previewScroll = 0
		}
		if m.previewItemIndex >= 0 {
			if m.previewItemIndex < m.previewScroll {
				m.previewScroll = m.previewItemIndex
			} else if m.previewItemIndex >= m.previewScroll+visible {
				m.previewScroll = m.previewItemIndex - visible + 1
			}
		}

		endIdx := m.previewScroll + visible
		if endIdx > len(cat.Items) {
			endIdx = len(cat.Items)
		}

		pathWidth := m.width - 20
		if pathWidth < 30 {
			pathWidth = 30
		}

		for i := m.previewScroll; i < endIdx; i++ {
			item := cat.Items[i]
			isCurrent := m.previewItemIndex == i
			isExcluded := m.isExcluded(cat.Category.ID, item.Path)

			cursor := "  "
			if isCurrent {
				cursor = CursorStyle.Render("▸ ")
			}

			checkbox := SuccessStyle.Render("[✓]")
			if isExcluded {
				checkbox = MutedStyle.Render("[ ]")
			}

			icon := " "
			if item.IsDirectory {
				icon = ">"
			}

			path := shortenPath(item.Path, pathWidth-4)
			if isExcluded {
				path = MutedStyle.Render(path)
			} else if isCurrent {
				path = SelectedStyle.Render(path)
			}

			size := fmt.Sprintf("%*s", colSize, utils.FormatSize(item.Size))
			if isExcluded {
				size = MutedStyle.Render(size)
			} else {
				size = SizeStyle.Render(size)
			}

			b.WriteString(fmt.Sprintf("%s%s %s %-*s %s\n", cursor, checkbox, icon, pathWidth-4, path, size))
		}

		if len(cat.Items) > visible {
			b.WriteString(MutedStyle.Render(fmt.Sprintf("\n  [%d-%d / %d]", m.previewScroll+1, endIdx, len(cat.Items))))
		}
	}

	// Warning
	for _, r := range selected {
		if r.Category.Safety == types.SafetyLevelRisky {
			b.WriteString("\n" + DangerStyle.Render("Warning: Risky items included"))
			break
		}
	}

	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("←→ Tab  ↑↓ Move  space Toggle  a Include All  d Exclude All  y Delete  esc Back"))
	return b.String()
}

func (m *Model) renderTabs(selected []*types.ScanResult, currentIdx int) string {
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

	// Join with space, will naturally wrap if too long
	return strings.Join(tabs, " ")
}

func (m *Model) viewDrillDown() string {
	if len(m.drillDownStack) == 0 {
		return ""
	}

	state := &m.drillDownStack[len(m.drillDownStack)-1]
	var b strings.Builder

	b.WriteString(HeaderStyle.Render("Directory Browser"))
	b.WriteString("\n\n")

	b.WriteString(MutedStyle.Render("Path: ") + shortenPath(state.path, m.width-10))
	b.WriteString("\n")
	b.WriteString(Divider(60) + "\n\n")

	if len(state.items) == 0 {
		b.WriteString(MutedStyle.Render("(empty)") + "\n")
	} else {
		visible := m.height - 12
		if visible < 5 {
			visible = 5
		}

		endIdx := state.scroll + visible
		if endIdx > len(state.items) {
			endIdx = len(state.items)
		}

		nameWidth := m.width - 20
		if nameWidth < 20 {
			nameWidth = 20
		}

		for i := state.scroll; i < endIdx; i++ {
			item := state.items[i]
			isCurrent := i == state.cursor

			cursor := "  "
			if isCurrent {
				cursor = CursorStyle.Render("▸ ")
			}

			icon := " "
			if item.IsDirectory {
				icon = ">"
			}

			name := item.Name
			if len(name) > nameWidth {
				name = name[:nameWidth-2] + ".."
			}
			if isCurrent {
				name = SelectedStyle.Render(name)
			}

			size := fmt.Sprintf("%*s", colSize, utils.FormatSize(item.Size))
			b.WriteString(fmt.Sprintf("%s%s %-*s %s\n", cursor, icon, nameWidth, name, SizeStyle.Render(size)))
		}

		if len(state.items) > visible {
			b.WriteString(MutedStyle.Render(fmt.Sprintf("\n  [%d/%d]", state.cursor+1, len(state.items))))
		}
	}

	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("↑↓ Navigate  enter Enter folder  esc/backspace Back  q Quit"))
	return b.String()
}

func (m *Model) viewConfirm() string {
	var b strings.Builder

	b.WriteString("\n\n")
	b.WriteString(HeaderStyle.Render("Confirm Deletion"))
	b.WriteString("\n\n")
	b.WriteString(Divider(50) + "\n\n")

	b.WriteString(fmt.Sprintf("  Total %s will be deleted.\n\n",
		DangerStyle.Render(formatSize(m.getSelectedSize()))))

	selected := m.getSelectedResults()
	for _, r := range selected {
		dot := safetyDot(r.Category.Safety)
		effectiveSize := m.getEffectiveSize(r)
		size := fmt.Sprintf("%*s", colSize, utils.FormatSize(effectiveSize))
		b.WriteString(fmt.Sprintf("  %s %-24s %s\n", dot, r.Category.Name, size))
	}

	b.WriteString("\n" + Divider(50) + "\n\n")
	b.WriteString(WarningStyle.Render("  This action cannot be undone!"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  %s Press y or Enter to delete\n", SuccessStyle.Render("▸")))
	b.WriteString(fmt.Sprintf("  %s Press n or Esc to cancel\n", DangerStyle.Render("▸")))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, b.String())
}

func (m *Model) viewCleaning() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(Logo())
	b.WriteString("\n\n")
	b.WriteString(m.spinner.View() + " ")
	b.WriteString(TitleStyle.Render("Cleaning..."))
	b.WriteString("\n\n")
	b.WriteString(MutedStyle.Render("Removing selected items."))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, b.String())
}

func (m *Model) viewReport() string {
	var b strings.Builder

	b.WriteString(HeaderStyle.Render("Cleanup Complete"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("Freed:   %s\n", SizeStyle.Render(formatSize(m.report.FreedSpace))))
	b.WriteString(fmt.Sprintf("Items:   %d\n", m.report.CleanedItems))
	if m.report.FailedItems > 0 {
		b.WriteString(DangerStyle.Render(fmt.Sprintf("Failed:  %d\n", m.report.FailedItems)))
	}
	b.WriteString(fmt.Sprintf("Time:    %s\n\n", m.report.Duration.Round(time.Millisecond)))

	b.WriteString(Divider(50) + "\n\n")

	for _, result := range m.report.Results {
		status := SuccessStyle.Render("✓")
		if len(result.Errors) > 0 {
			status = DangerStyle.Render("✗")
		}

		size := fmt.Sprintf("%*s", colSize, utils.FormatSize(result.FreedSpace))
		b.WriteString(fmt.Sprintf("%s %-28s %s\n", status, result.Category.Name, SizeStyle.Render(size)))

		for _, err := range result.Errors {
			b.WriteString(DangerStyle.Render(fmt.Sprintf("  └ %s\n", err)))
		}
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("enter Rescan  q Quit"))
	return b.String()
}

// Helper functions
func formatSize(bytes int64) string {
	return utils.FormatSize(bytes)
}

func safetyDot(level types.SafetyLevel) string {
	switch level {
	case types.SafetyLevelSafe:
		return SuccessStyle.Render("●")
	case types.SafetyLevelModerate:
		return WarningStyle.Render("●")
	case types.SafetyLevelRisky:
		return DangerStyle.Render("●")
	default:
		return MutedStyle.Render("●")
	}
}

func safetyBadge(level types.SafetyLevel) string {
	switch level {
	case types.SafetyLevelSafe:
		return SuccessStyle.Render("[Safe]")
	case types.SafetyLevelModerate:
		return WarningStyle.Render("[Moderate]")
	case types.SafetyLevelRisky:
		return DangerStyle.Render("[Risky]")
	default:
		return ""
	}
}

func methodBadge(method types.CleanupMethod) string {
	switch method {
	case types.MethodTrash:
		return MutedStyle.Render("[Trash]")
	case types.MethodManual:
		return WarningStyle.Render("[Manual]")
	case types.MethodCommand:
		return MutedStyle.Render("[Command]")
	case types.MethodSpecial:
		return MutedStyle.Render("[Special]")
	default:
		return MutedStyle.Render("[Delete]")
	}
}

func shortenPath(path string, maxLen int) string {
	home, _ := filepath.Abs(utils.ExpandPath("~"))
	if strings.HasPrefix(path, home) {
		path = "~" + path[len(home):]
	}
	if len(path) > maxLen {
		path = "..." + path[len(path)-(maxLen-3):]
	}
	return path
}
