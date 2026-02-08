package tui

import (
	"fmt"
	"strings"

	"github.com/2ykwang/mac-cleanup-go/internal/styles"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

func (m *Model) viewCleaning() string {
	var b strings.Builder

	b.WriteString(styles.HeaderStyle.Render("Cleaning..."))
	b.WriteString("\n\n")

	rowWidth := m.cleaningRowWidth()
	nameWidth, sizeWidth := m.cleaningColumnWidths(2)

	// Show completed categories
	for _, cat := range m.cleaningCompleted {
		displayName := padToWidth(truncateToWidth(cat.name, nameWidth, false), nameWidth)
		if cat.errors == 0 {
			// Full success
			size := fmt.Sprintf("%*s", sizeWidth, utils.FormatSize(cat.freedSpace))
			b.WriteString(fmt.Sprintf("%s %s %s\n",
				styles.SuccessStyle.Render("✓"),
				displayName,
				styles.SizeStyle.Render(size)))
		} else if cat.cleaned > 0 {
			// Partial success
			size := fmt.Sprintf("%*s", sizeWidth, utils.FormatSize(cat.freedSpace))
			b.WriteString(fmt.Sprintf("%s %s %s\n",
				styles.WarningStyle.Render("△"),
				displayName,
				styles.SizeStyle.Render(size)))
		} else {
			// All failed
			b.WriteString(fmt.Sprintf("%s %s %s\n",
				styles.DangerStyle.Render("✗"),
				displayName,
				styles.MutedStyle.Render("failed")))
		}
	}

	// Show current category being processed
	if m.cleaningCategory != "" {
		b.WriteString(fmt.Sprintf("%s %s\n",
			m.spinner.View(),
			m.cleaningCategory))

		if m.cleaningItem != "" {
			item := m.cleaningItem
			if len(item) > 45 {
				item = "..." + item[len(item)-42:]
			}
			b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("  └ %s\n", item)))
		}
	}

	// Show recent deletions list
	if m.recentDeleted.Len() > 0 {
		b.WriteString("\n")
		b.WriteString(styles.MutedStyle.Render("Recent:"))
		b.WriteString("\n")
		b.WriteString(m.renderRecentDeleted())
	}

	b.WriteString("\n")

	if m.cleaningTotal > 0 {
		m.cleaningProgress.Width = rowWidth
		b.WriteString(m.cleaningProgress.View())
		b.WriteString("\n")
		percent := 0
		if m.cleaningTotal > 0 {
			percent = m.cleaningCurrent * 100 / m.cleaningTotal
		}
		progress := fmt.Sprintf("%d%% (%d/%d)", percent, m.cleaningCurrent, m.cleaningTotal)
		b.WriteString(styles.MutedStyle.Render(progress))
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
			icon = styles.SuccessStyle.Render("✓")
		} else {
			icon = styles.DangerStyle.Render("✗")
		}

		displayPath := shortenPath(entry.Path, nameWidth)
		displayPath = padToWidth(displayPath, nameWidth)

		// Format size
		size := fmt.Sprintf("%*s", sizeWidth, utils.FormatSize(entry.Size))

		// Build line
		if entry.Success {
			b.WriteString(fmt.Sprintf("  %s %s %s\n",
				icon,
				displayPath,
				styles.SizeStyle.Render(size)))
		} else {
			b.WriteString(fmt.Sprintf("  %s %s %s\n",
				icon,
				styles.MutedStyle.Render(displayPath),
				styles.MutedStyle.Render(size)))
		}
	}

	return b.String()
}
