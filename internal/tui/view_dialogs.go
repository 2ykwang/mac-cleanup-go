package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/2ykwang/mac-cleanup-go/internal/styles"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

// Confirm dialog view

func (m *Model) viewConfirm() string {
	base := lipgloss.NewStyle().Faint(true).Render(m.viewPreview())
	return overlayCentered(base, m.confirmDialog(), m.width, m.height)
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
	b.WriteString(styles.HeaderStyle.Render(cat.Name))
	b.WriteString("\n\n")

	// Note (warning) - truncate if too long
	if cat.Note != "" {
		note := cat.Note
		if len(note) > contentWidth-4 {
			note = note[:contentWidth-7] + "..."
		}
		b.WriteString(styles.DangerStyle.Render("⚠ " + note))
		b.WriteString("\n\n")
	}

	b.WriteString(styles.Divider(contentWidth) + "\n\n")

	// Guide (deletion method)
	b.WriteString(styles.TextStyle.Render("How to delete:"))
	b.WriteString("\n")
	guideText := cat.Guide
	if guideText == "" {
		guideText = "This item must be deleted manually within the app."
	}
	// Render each line of guide text
	for _, line := range strings.Split(strings.TrimSpace(guideText), "\n") {
		b.WriteString(styles.MutedStyle.Render(line))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Paths with cursor
	if len(cat.Paths) > 0 {
		b.WriteString(styles.TextStyle.Render("Paths:"))
		b.WriteString("\n")
		for i, path := range cat.Paths {
			// Truncate long paths from the beginning
			displayPath := path
			if len(displayPath) > contentWidth-4 {
				displayPath = "..." + displayPath[len(displayPath)-(contentWidth-7):]
			}

			if i == m.guidePathIndex {
				b.WriteString(styles.CursorStyle.Render("▸ ") + styles.MutedStyle.Render(displayPath))
			} else {
				b.WriteString("  " + styles.MutedStyle.Render(displayPath))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(styles.Divider(contentWidth) + "\n\n")

	// Key hints
	b.WriteString(styles.HelpStyle.Render("↑↓ Select • o Open • Esc Close"))

	// Create a fixed-width box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorBorder).
		Padding(1, 2).
		Width(boxWidth)

	content := boxStyle.Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m *Model) confirmDialog() string {
	boxWidth := min(72, m.width-4)
	if boxWidth < 40 {
		boxWidth = min(m.width-2, 40)
	}
	if boxWidth < 24 {
		boxWidth = m.width
	}
	if boxWidth < 0 {
		boxWidth = m.width
	}
	if boxWidth > m.width {
		boxWidth = m.width
	}
	contentWidth := boxWidth - 6
	if contentWidth < 0 {
		contentWidth = 0
	}

	maxBoxHeight := m.height - 2
	if maxBoxHeight < 0 {
		maxBoxHeight = 0
	}
	contentHeight := maxBoxHeight - 4
	if contentHeight < 0 {
		contentHeight = 0
	}

	buttons := m.confirmButtons()
	if contentWidth > 0 {
		buttons = lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, buttons)
	}
	help := m.help
	if contentWidth > 0 {
		help.Width = contentWidth
	}
	helpView := help.View(ConfirmKeyMap)

	buildSections := func(compact bool) ([]string, []string) {
		var head []string
		var tail []string

		head = append(head, styles.HeaderStyle.Render("Confirm Deletion"))
		if !compact {
			head = append(head, "")
		}
		head = append(head, styles.SuccessStyle.Render("Files will be moved to Trash"))
		if !compact {
			head = append(head, "")
		}
		if contentWidth > 0 && !compact {
			head = append(head, styles.Divider(contentWidth), "")
		}
		head = append(head, fmt.Sprintf("Total %s will be deleted.",
			styles.DangerStyle.Render(formatSize(m.getSelectedSize()))))
		if !compact {
			head = append(head, "")
		}

		if contentWidth > 0 && !compact {
			tail = append(tail, styles.Divider(contentWidth), "")
		}
		tail = append(tail, styles.MutedStyle.Render("Items can be recovered from Trash"))
		if !compact {
			tail = append(tail, "")
		}
		tail = append(tail, buttons)
		tail = append(tail, splitLines(helpView)...)

		return head, tail
	}

	head, tail := buildSections(false)
	availableForItems := contentHeight - len(head) - len(tail)
	if availableForItems < 0 && contentHeight > 0 {
		head, tail = buildSections(true)
		availableForItems = contentHeight - len(head) - len(tail)
	}
	if availableForItems < 0 {
		availableForItems = 0
	}

	var itemLines []string
	selected := m.getSelectedResults()
	totalSelected := len(selected)
	nameWidth, sizeWidth := confirmItemColumns(contentWidth)
	showScrollInfo := false
	visibleItems := availableForItems
	if totalSelected > availableForItems && availableForItems >= 2 {
		showScrollInfo = true
		visibleItems = availableForItems - 1
	}
	if visibleItems < 0 {
		visibleItems = 0
	}

	if nameWidth > 0 && sizeWidth > 0 && visibleItems > 0 {
		maxScroll := totalSelected - visibleItems
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.confirmScroll > maxScroll {
			m.confirmScroll = maxScroll
		}
		if m.confirmScroll < 0 {
			m.confirmScroll = 0
		}

		start := m.confirmScroll
		end := start + visibleItems
		if end > totalSelected {
			end = totalSelected
		}

		for i := start; i < end; i++ {
			r := selected[i]
			dot := safetyDot(r.Category.Safety)
			effectiveSize := m.getEffectiveSize(r)
			size := fmt.Sprintf("%*s", sizeWidth, utils.FormatSize(effectiveSize))
			name := padToWidth(truncateToWidth(r.Category.Name, nameWidth, false), nameWidth)
			itemLines = append(itemLines, fmt.Sprintf("  %s %s %s", dot, name, size))
		}
	}

	lines := append([]string{}, head...)
	lines = append(lines, itemLines...)
	if showScrollInfo && totalSelected > 0 && visibleItems > 0 && len(itemLines) > 0 {
		start := m.confirmScroll + 1
		end := m.confirmScroll + len(itemLines)
		if end > totalSelected {
			end = totalSelected
		}
		info := fmt.Sprintf("Showing %d-%d of %d", start, end, totalSelected)
		if contentWidth > 0 {
			info = truncateToWidth(info, contentWidth, false)
		}
		lines = append(lines, styles.MutedStyle.Render(info))
	}
	lines = append(lines, tail...)
	content := strings.Join(lines, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorBorder).
		Padding(1, 2).
		Width(boxWidth)

	return boxStyle.Render(content)
}

func (m *Model) confirmButtons() string {
	cancelStyle := styles.ButtonStyle
	if m.confirmChoice == confirmCancel {
		cancelStyle = styles.ButtonActiveStyle
	}
	deleteStyle := styles.ButtonDangerStyle
	if m.confirmChoice == confirmDelete {
		deleteStyle = styles.ButtonDangerActiveStyle
	}

	cancel := cancelStyle.Render("Cancel")
	del := deleteStyle.Render("Delete")
	return lipgloss.JoinHorizontal(lipgloss.Top, cancel, "  ", del)
}

func confirmItemColumns(contentWidth int) (int, int) {
	if contentWidth <= 0 {
		return 0, 0
	}

	indentWidth := 2
	sizeWidth := colSize
	nameWidth := contentWidth - indentWidth - 3 - sizeWidth
	if nameWidth < 8 {
		nameWidth = 8
		sizeWidth = contentWidth - indentWidth - 3 - nameWidth
	}
	if sizeWidth < 6 {
		sizeWidth = 6
		nameWidth = contentWidth - indentWidth - 3 - sizeWidth
	}
	if nameWidth < 1 {
		nameWidth = 1
	}
	if sizeWidth < 1 {
		sizeWidth = 1
	}

	return nameWidth, sizeWidth
}
