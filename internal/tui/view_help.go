package tui

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/2ykwang/mac-cleanup-go/internal/styles"
)

const githubURL = "https://github.com/2ykwang/mac-cleanup-go"

func (m *Model) helpContent(contentWidth int) string {
	section := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorSecondary)
	keyStyle := lipgloss.NewStyle().Foreground(styles.ColorSecondary)
	colStyle := lipgloss.NewStyle().Width(14)

	styledKey := func(s string) string {
		parts := strings.Split(s, " / ")
		if len(parts) == 1 {
			return keyStyle.Render(s)
		}
		rendered := make([]string, len(parts))
		for i, p := range parts {
			rendered[i] = keyStyle.Render(p)
		}
		return strings.Join(rendered, styles.MutedStyle.Render(" / "))
	}

	writeRow := func(b *strings.Builder, key, desc string) {
		b.WriteString("  " + colStyle.Render(styledKey(key)) + styles.MutedStyle.Render(desc) + "\n")
	}

	var b strings.Builder

	muted := styles.MutedStyle
	div := styles.Divider(contentWidth)

	// Title
	b.WriteString(styles.HeaderStyle.Render("Help"))
	b.WriteString("\n" + div + "\n\n")

	// Overview
	b.WriteString(section.Render("Overview"))
	b.WriteString("\n")
	b.WriteString(styles.TextStyle.Render("Scans and removes macOS system caches, app logs, old"))
	b.WriteString("\n")
	b.WriteString(styles.TextStyle.Render("downloads, and dev tool leftovers to free disk space."))
	b.WriteString("\n")
	b.WriteString(muted.Render("Files are moved to Trash by default — always recoverable."))
	b.WriteString("\n")
	b.WriteString(muted.Render("Scan → Select → Preview → Confirm → Clean"))
	b.WriteString("\n\n")

	// Shortcuts
	b.WriteString(div + "\n")
	b.WriteString(section.Render("Shortcuts"))
	b.WriteString("\n\n")
	b.WriteString(section.Render("List"))
	b.WriteString("\n")
	for _, k := range [][2]string{
		{"↑↓ / jk", "Navigate"},
		{"space", "Select category"},
		{"enter", "Preview selected"},
		{"a / d", "Select all / Deselect all"},
		{"o", "Open GitHub"},
		{"q", "Quit"},
	} {
		writeRow(&b, k[0], k[1])
	}
	b.WriteString("\n")
	b.WriteString(section.Render("Preview"))
	b.WriteString("\n")
	for _, k := range [][2]string{
		{"↑↓ / jk", "Navigate items"},
		{"tab / ]", "Next section"},
		{"space", "Toggle exclusion"},
		{"/", "Search files"},
		{"s", "Sort order"},
		{"enter", "Open directory"},
		{"o", "Open in Finder"},
		{"y", "Proceed to delete"},
		{"esc", "Back to list"},
	} {
		writeRow(&b, k[0], k[1])
	}
	b.WriteString("\n")

	// Tips
	b.WriteString(div + "\n")
	b.WriteString(section.Render("Tips"))
	b.WriteString("\n")
	b.WriteString("  • " + keyStyle.Render("space") + muted.Render(" on manual item → opens cleanup guide") + "\n")
	b.WriteString("  • " + keyStyle.Render("/") + muted.Render(" in preview → search files by name") + "\n")
	b.WriteString("  • " + styles.DangerStyle.Render("Risky") + muted.Render(" items are auto-excluded for safety") + "\n")
	b.WriteString("\n")

	// GitHub
	b.WriteString(div + "\n")
	b.WriteString(section.Render("GitHub"))
	b.WriteString("\n")
	b.WriteString("  " + muted.Render(githubURL))

	return b.String()
}

func (m *Model) helpDialog() string {
	boxWidth := min(72, m.width*8/10)
	if boxWidth < 48 {
		boxWidth = min(m.width-2, 48)
	}
	if boxWidth < 28 {
		boxWidth = m.width
	}
	if boxWidth <= 0 {
		boxWidth = 60
	}

	contentWidth := max(boxWidth-6, 0)

	fullContent := m.helpContent(contentWidth)
	lines := splitLines(fullContent)

	// Available height inside the box: terminal - border(2) - padding(2) - margin(2)
	viewHeight := m.height - 6
	if viewHeight < 3 {
		viewHeight = 3
	}
	// Reserve 1 line for footer hint
	scrollViewHeight := viewHeight - 1
	if scrollViewHeight < 1 {
		scrollViewHeight = 1
	}

	scrollable := len(lines) > scrollViewHeight

	maxScroll := len(lines) - scrollViewHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	// Local clamp — no state mutation in View
	scroll := clamp(m.helpScroll, maxScroll)

	end := scroll + scrollViewHeight
	if end > len(lines) {
		end = len(lines)
	}

	visible := strings.Join(lines[scroll:end], "\n")

	// Footer hint
	footer := "esc close  o github"
	if scrollable {
		footer = "esc close  ↑↓ scroll  o github"
	}
	visible += "\n" + styles.MutedStyle.Render(footer)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorBorder).
		Padding(1, 2).
		Width(boxWidth)

	return boxStyle.Render(visible)
}

func (m *Model) maxHelpScroll() int {
	boxWidth := min(72, m.width*8/10)
	if boxWidth < 48 {
		boxWidth = min(m.width-2, 48)
	}
	if boxWidth < 28 {
		boxWidth = m.width
	}
	if boxWidth <= 0 {
		boxWidth = 60
	}
	contentWidth := max(boxWidth-6, 0)

	lines := splitLines(m.helpContent(contentWidth))
	viewHeight := m.height - 6
	if viewHeight < 3 {
		viewHeight = 3
	}
	scrollViewHeight := viewHeight - 1
	if scrollViewHeight < 1 {
		scrollViewHeight = 1
	}
	maxScroll := len(lines) - scrollViewHeight
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
}

func (m *Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "?", "enter":
		m.showHelp = false
		m.helpScroll = 0
		return m, nil
	case "up", "k":
		m.helpScroll = clamp(m.helpScroll-1, m.maxHelpScroll())
	case "down", "j":
		m.helpScroll = clamp(m.helpScroll+1, m.maxHelpScroll())
	case "o":
		_ = exec.Command("open", githubURL).Start()
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}
