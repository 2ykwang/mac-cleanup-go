package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

// Confirm dialog view

func (m *Model) viewConfirm() string {
	var b strings.Builder

	b.WriteString("\n\n")
	b.WriteString(HeaderStyle.Render("Confirm Deletion"))
	b.WriteString("\n\n")

	b.WriteString(SuccessStyle.Render("  → Files will be moved to Trash"))
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
	b.WriteString(MutedStyle.Render("  Items can be recovered from Trash"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  %s Press y or Enter to delete\n", SuccessStyle.Render("▸")))
	b.WriteString(fmt.Sprintf("  %s Press n or Esc to cancel\n", DangerStyle.Render("▸")))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, b.String())
}

// Help dialog view

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

// Guide dialog view

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
