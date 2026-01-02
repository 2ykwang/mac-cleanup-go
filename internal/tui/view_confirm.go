package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"mac-cleanup-go/internal/utils"
)

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
