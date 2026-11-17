package tui

import (
	"fmt"
	"strings"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

func (m *Model) viewCleaning() string {
	var b strings.Builder

	b.WriteString(m.styles.HeaderStyle.Render("Cleaning..."))
	b.WriteString("\n\n")
	b.WriteString("Processed sizes (scan estimates)\n\n")

	rowWidth := m.cleaningRowWidth()
	nameWidth, sizeWidth := m.cleaningColumnWidths(2)

	// Show completed categories
	for _, cat := range m.cleaningCompleted {
		displayName := padToWidth(truncateToWidth(cat.name, nameWidth, false), nameWidth)
		if cat.errors == 0 && cat.cleaned > 0 {
			// Full success
			size := fmt.Sprintf("%*s", sizeWidth, utils.FormatSize(cat.freedSpace))
			fmt.Fprintf(&b, "%s %s %s\n",
				m.styles.SuccessStyle.Render("✓"),
				displayName,
				size)
		} else if cat.cleaned > 0 {
			// Partial success
			size := fmt.Sprintf("%*s", sizeWidth, utils.FormatSize(cat.freedSpace))
			fmt.Fprintf(&b, "%s %s %s\n",
				m.styles.WarningStyle.Render("△"),
				displayName,
				size)
		} else if cat.errors > 0 {
			// All failed
			fmt.Fprintf(&b, "%s %s %s\n",
				m.styles.DangerStyle.Render("✗"),
				displayName,
				m.styles.MutedStyle.Render("failed"))
		} else {
			fmt.Fprintf(&b, "%s %s %s\n", m.styles.WarningStyle.Render("-"), displayName, "no confirmed changes")
		}
	}

	// Show current category being processed
	if m.cleaningCategory != "" {
		fmt.Fprintf(&b, "%s %s\n",
			m.spinner.View(),
			m.cleaningCategory)

		if m.cleaningItem != "" {
			item := m.cleaningItem
			if len(item) > 45 {
				item = "..." + item[len(item)-42:]
			}
			fmt.Fprintf(&b, "%s%s\n", m.styles.MutedStyle.Render("  └ "), item)
		}
	}

	// Show recent deletions list
	if m.recentDeleted.Len() > 0 {
		b.WriteString("\n")
		b.WriteString(m.styles.MutedStyle.Render("Recent:"))
		b.WriteString("\n")
		b.WriteString(m.renderRecentDeleted())
	}

	b.WriteString("\n")

	if m.cleaningTotal > 0 {
		m.cleaningProgress.SetWidth(rowWidth)
		b.WriteString(m.cleaningProgress.View())
		b.WriteString("\n")
		percent := 0
		if m.cleaningTotal > 0 {
			percent = m.cleaningCurrent * 100 / m.cleaningTotal
		}
		progress := fmt.Sprintf("%d%% (%d/%d)", percent, m.cleaningCurrent, m.cleaningTotal)
		b.WriteString(progress)
	}

	return b.String()
}

func (m *Model) cleaningRowWidth() int {
	width := m.width
	if width < 40 {
		width = 40
	}
	return width
}

func (m *Model) cleaningColumnWidths(overhead int) (int, int) {
	return m.nameSizeColumns(overhead, true)
}

// renderRecentDeleted renders the recent deletions list with status icons and sizes
func (m *Model) renderRecentDeleted() string {
	var b strings.Builder
	items := m.recentDeleted.Items()
	nameWidth, sizeWidth := m.cleaningColumnWidths(5)

	for _, entry := range items {
		// Status icon
		var icon string
		if entry.Success {
			icon = m.styles.SuccessStyle.Render("✓")
		} else if entry.Outcome == types.ItemOutcomeSkipped || entry.Outcome == types.ItemOutcomeAbsent || entry.Outcome == types.ItemOutcomeUnconfirmed {
			icon = m.styles.MutedStyle.Render("-")
		} else {
			icon = m.styles.DangerStyle.Render("✗")
		}

		displayPath := shortenPath(entry.Path, nameWidth)
		displayPath = padToWidth(displayPath, nameWidth)

		// Format size
		size := fmt.Sprintf("%*s", sizeWidth, utils.FormatSize(entry.Size))

		fmt.Fprintf(&b, "  %s %s %s\n", icon, displayPath, size)
	}

	return b.String()
}
