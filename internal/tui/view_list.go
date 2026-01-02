package tui

import (
	"fmt"
	"strings"

	"mac-cleanup-go/internal/utils"
	"mac-cleanup-go/pkg/types"
)

func (m *Model) listHeader() string {
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
	b.WriteString(Divider(60) + "\n")

	return b.String()
}

func (m *Model) listFooter() string {
	var b strings.Builder

	// Show scan warnings after scan completes
	if !m.scanning && len(m.scanErrors) > 0 {
		b.WriteString(WarningStyle.Render("[!] Scan warnings:"))
		b.WriteString("\n")
		for _, err := range m.scanErrors {
			errMsg := err.Error
			if len(errMsg) > 50 {
				errMsg = errMsg[:47] + "..."
			}
			b.WriteString(MutedStyle.Render(fmt.Sprintf("    %s: %s", err.CategoryName, errMsg)))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("↑↓ Navigate  space Select  a Select All  d Deselect All  enter Preview  q Quit"))

	return b.String()
}

func (m *Model) viewList() string {
	header := m.listHeader()
	footer := m.listFooter()
	visible := m.availableLines(header, footer)

	var b strings.Builder
	b.WriteString(header)

	// Items
	if len(m.results) == 0 {
		if m.scanning {
			b.WriteString(MutedStyle.Render("Scanning..."))
		} else {
			b.WriteString(MutedStyle.Render("No items to clean."))
		}
		b.WriteString("\n")
	} else {
		// Adjust scroll
		m.scroll = m.adjustScrollFor(m.cursor, m.scroll, visible, len(m.results))

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

	b.WriteString("\n")
	b.WriteString(footer)
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
	// Add method badge only for special methods
	switch r.Category.Method {
	case types.MethodManual:
		name += " [Manual]"
	case types.MethodCommand:
		name += " [Command]"
	case types.MethodSpecial:
		name += " [Special]"
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
