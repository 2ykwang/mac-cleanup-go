package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) viewGuide() string {
	if m.guideCategory == nil {
		return m.viewList()
	}

	cat := m.guideCategory

	// Calculate box width: 80% of terminal width
	boxWidth := m.width * 8 / 10
	if boxWidth < 50 {
		boxWidth = 50
	}
	// Content width = box width - borders(2) - padding(4)
	contentWidth := boxWidth - 6

	var b strings.Builder

	// Title
	b.WriteString(HeaderStyle.Render(cat.Name))
	b.WriteString("\n\n")

	// Note (warning) - truncate if too long
	if cat.Note != "" {
		note := cat.Note
		if len(note) > contentWidth-4 {
			note = note[:contentWidth-7] + "..."
		}
		b.WriteString(DangerStyle.Render("⚠ " + note))
		b.WriteString("\n\n")
	}

	b.WriteString(Divider(contentWidth) + "\n\n")

	// Guide (deletion method)
	b.WriteString(TextStyle.Render("How to delete:"))
	b.WriteString("\n")
	guideText := cat.Guide
	if guideText == "" {
		guideText = "This item must be deleted manually within the app."
	}
	// Render each line of guide text
	for _, line := range strings.Split(strings.TrimSpace(guideText), "\n") {
		b.WriteString(MutedStyle.Render(line))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Paths with cursor
	if len(cat.Paths) > 0 {
		b.WriteString(TextStyle.Render("Paths:"))
		b.WriteString("\n")
		for i, path := range cat.Paths {
			// Truncate long paths from the beginning
			displayPath := path
			if len(displayPath) > contentWidth-4 {
				displayPath = "..." + displayPath[len(displayPath)-(contentWidth-7):]
			}

			if i == m.guidePathIndex {
				b.WriteString(CursorStyle.Render("▸ ") + MutedStyle.Render(displayPath))
			} else {
				b.WriteString("  " + MutedStyle.Render(displayPath))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(Divider(contentWidth) + "\n\n")

	// Key hints
	b.WriteString(HelpStyle.Render("↑↓ Select • o Open • Esc Close"))

	// Create a fixed-width box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2).
		Width(boxWidth)

	content := boxStyle.Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
