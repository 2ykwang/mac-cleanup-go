package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) viewHelp() string {
	// Get shortcuts for the previous view (before help was opened)
	var groups []ShortcutGroup

	switch m.helpPreviousView {
	case ViewList:
		groups = ViewShortcuts[ViewList]
	case ViewPreview:
		if len(m.drillDownStack) > 0 {
			groups = DrillDownShortcuts
		} else {
			groups = ViewShortcuts[ViewPreview]
		}
	case ViewConfirm:
		groups = ViewShortcuts[ViewConfirm]
	case ViewReport:
		groups = ViewShortcuts[ViewReport]
	default:
		groups = ViewShortcuts[ViewList]
	}

	// Calculate box width: 60% of terminal width, min 40, max 60
	boxWidth := m.width * 6 / 10
	if boxWidth < 40 {
		boxWidth = 40
	}
	if boxWidth > 60 {
		boxWidth = 60
	}
	contentWidth := boxWidth - 6 // borders(2) + padding(4)

	var b strings.Builder

	// Title
	b.WriteString(HeaderStyle.Render("Keyboard Shortcuts"))
	b.WriteString("\n\n")

	// Render each group
	for i, group := range groups {
		// Group name
		b.WriteString(TextStyle.Bold(true).Render(group.Name))
		b.WriteString("\n")

		// Shortcuts in group
		for _, shortcut := range group.Shortcuts {
			keyStyle := lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Width(12)
			descStyle := lipgloss.NewStyle().
				Foreground(ColorText)

			line := keyStyle.Render(shortcut.Key) + descStyle.Render(shortcut.Desc)
			b.WriteString("  " + line + "\n")
		}

		// Add spacing between groups (except last)
		if i < len(groups)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(Divider(contentWidth) + "\n\n")

	// Footer hint
	b.WriteString(HelpStyle.Render("Press Esc or ? to close"))

	// Create box style (similar to view_guide.go)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2).
		Width(boxWidth)

	content := boxStyle.Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
