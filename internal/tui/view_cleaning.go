package tui

import (
	"fmt"
	"strings"

	"mac-cleanup-go/internal/utils"
)

func (m *Model) viewCleaning() string {
	var b strings.Builder

	b.WriteString(HeaderStyle.Render("Cleaning..."))
	b.WriteString("\n\n")

	// Progress info
	if m.cleaningTotal > 0 {
		progress := fmt.Sprintf("[%d/%d]", m.cleaningCurrent, m.cleaningTotal)
		b.WriteString(SizeStyle.Render(progress))
		b.WriteString("\n\n")
	}

	b.WriteString(Divider(50))
	b.WriteString("\n\n")

	// Show completed categories
	for _, cat := range m.cleaningCompleted {
		if cat.errors == 0 {
			// Full success
			size := fmt.Sprintf("%*s", colSize, utils.FormatSize(cat.freedSpace))
			b.WriteString(fmt.Sprintf("%s %-30s %s\n",
				SuccessStyle.Render("✓"),
				cat.name,
				SizeStyle.Render(size)))
		} else if cat.cleaned > 0 {
			// Partial success
			size := fmt.Sprintf("%*s", colSize, utils.FormatSize(cat.freedSpace))
			b.WriteString(fmt.Sprintf("%s %-30s %s\n",
				WarningStyle.Render("△"),
				cat.name,
				SizeStyle.Render(size)))
		} else {
			// All failed
			b.WriteString(fmt.Sprintf("%s %-30s %s\n",
				DangerStyle.Render("✗"),
				cat.name,
				MutedStyle.Render("failed")))
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
			b.WriteString(MutedStyle.Render(fmt.Sprintf("  └ %s\n", item)))
		}
	}

	return b.String()
}
