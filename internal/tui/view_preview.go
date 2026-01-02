package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"mac-cleanup-go/internal/utils"
	"mac-cleanup-go/pkg/types"
)

func (m *Model) viewPreview() string {
	if len(m.drillDownStack) > 0 {
		return m.viewDrillDown()
	}

	selected := m.getSelectedResults()
	if len(selected) == 0 {
		return "No items selected."
	}

	var b strings.Builder

	b.WriteString(HeaderStyle.Render("Cleanup Preview"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("Selected: %d  │  Estimated: %s\n",
		m.getSelectedCount(), SizeStyle.Render(formatSize(m.getSelectedSize()))))
	b.WriteString(Divider(60) + "\n\n")

	// Tabs (no truncation, wrap if needed)
	catIdx := m.findSelectedCatIndex()
	b.WriteString(m.renderTabs(selected, catIdx))
	b.WriteString("\n\n")

	// Current category content
	cat := m.getPreviewCatResult()
	if cat != nil {

		badge := safetyBadge(cat.Category.Safety)
		mBadge := m.methodBadge(cat.Category.Method)
		effectiveSize := m.getEffectiveSize(cat)
		if mBadge != "" {
			b.WriteString(fmt.Sprintf("%s %s  %s  │  %d items\n",
				badge, mBadge, SizeStyle.Render(formatSize(effectiveSize)), len(cat.Items)))
		} else {
			b.WriteString(fmt.Sprintf("%s  %s  │  %d items\n",
				badge, SizeStyle.Render(formatSize(effectiveSize)), len(cat.Items)))
		}
		if cat.Category.Note != "" {
			b.WriteString(MutedStyle.Render(cat.Category.Note) + "\n")
		}
		// Show guide for manual method
		if cat.Category.Method == types.MethodManual && cat.Category.Guide != "" {
			b.WriteString(WarningStyle.Render("[Manual] "+cat.Category.Guide) + "\n")
		}
		b.WriteString(Divider(60) + "\n")

		visible := m.height - 16
		if visible < 3 {
			visible = 3
		}

		// Adjust scroll
		maxScroll := len(cat.Items) - visible
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.previewScroll > maxScroll {
			m.previewScroll = maxScroll
		}
		if m.previewScroll < 0 {
			m.previewScroll = 0
		}
		if m.previewItemIndex >= 0 {
			if m.previewItemIndex < m.previewScroll {
				m.previewScroll = m.previewItemIndex
			} else if m.previewItemIndex >= m.previewScroll+visible {
				m.previewScroll = m.previewItemIndex - visible + 1
			}
		}

		endIdx := m.previewScroll + visible
		if endIdx > len(cat.Items) {
			endIdx = len(cat.Items)
		}

		pathWidth := m.width - 20
		if pathWidth < 30 {
			pathWidth = 30
		}

		for i := m.previewScroll; i < endIdx; i++ {
			item := cat.Items[i]
			isCurrent := m.previewItemIndex == i
			isExcluded := m.isExcluded(cat.Category.ID, item.Path)

			cursor := "  "
			if isCurrent {
				cursor = CursorStyle.Render("▸ ")
			}

			checkbox := SuccessStyle.Render("[✓]")
			if isExcluded {
				checkbox = MutedStyle.Render("[ ]")
			}

			icon := " "
			if item.IsDirectory {
				icon = ">"
			}

			path := shortenPath(item.Path, pathWidth-4)
			if isExcluded {
				path = MutedStyle.Render(path)
			} else if isCurrent {
				path = SelectedStyle.Render(path)
			}

			size := fmt.Sprintf("%*s", colSize, utils.FormatSize(item.Size))
			if isExcluded {
				size = MutedStyle.Render(size)
			} else {
				size = SizeStyle.Render(size)
			}

			b.WriteString(fmt.Sprintf("%s%s %s %-*s %s\n", cursor, checkbox, icon, pathWidth-4, path, size))
		}

		if len(cat.Items) > visible {
			b.WriteString(MutedStyle.Render(fmt.Sprintf("\n  [%d-%d / %d]", m.previewScroll+1, endIdx, len(cat.Items))))
		}
	}

	// Warning
	for _, r := range selected {
		if r.Category.Safety == types.SafetyLevelRisky {
			b.WriteString("\n" + DangerStyle.Render("Warning: Risky items included"))
			break
		}
	}

	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("←→ Tab  ↑↓ Move  space Toggle  a Include All  d Exclude All  y Delete  esc Back"))
	return b.String()
}

func (m *Model) renderTabs(selected []*types.ScanResult, currentIdx int) string {
	var tabs []string

	for _, r := range selected {
		name := r.Category.Name
		isCurrent := r.Category.ID == m.previewCatID
		if isCurrent {
			tab := lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorText).
				Background(ColorPrimary).
				Padding(0, 1).
				Render(name)
			tabs = append(tabs, tab)
		} else {
			tab := lipgloss.NewStyle().
				Foreground(ColorMuted).
				Padding(0, 1).
				Render(name)
			tabs = append(tabs, tab)
		}
	}

	// Join with space, will naturally wrap if too long
	return strings.Join(tabs, " ")
}

func (m *Model) viewDrillDown() string {
	if len(m.drillDownStack) == 0 {
		return ""
	}

	state := &m.drillDownStack[len(m.drillDownStack)-1]
	var b strings.Builder

	b.WriteString(HeaderStyle.Render("Directory Browser"))
	b.WriteString("\n\n")

	b.WriteString(MutedStyle.Render("Path: ") + shortenPath(state.path, m.width-10))
	b.WriteString("\n")
	b.WriteString(Divider(60) + "\n\n")

	if len(state.items) == 0 {
		b.WriteString(MutedStyle.Render("(empty)") + "\n")
	} else {
		visible := m.height - 12
		if visible < 5 {
			visible = 5
		}

		endIdx := state.scroll + visible
		if endIdx > len(state.items) {
			endIdx = len(state.items)
		}

		nameWidth := m.width - 20
		if nameWidth < 20 {
			nameWidth = 20
		}

		for i := state.scroll; i < endIdx; i++ {
			item := state.items[i]
			isCurrent := i == state.cursor

			cursor := "  "
			if isCurrent {
				cursor = CursorStyle.Render("▸ ")
			}

			icon := " "
			if item.IsDirectory {
				icon = ">"
			}

			name := item.Name
			if len(name) > nameWidth {
				name = name[:nameWidth-2] + ".."
			}
			if isCurrent {
				name = SelectedStyle.Render(name)
			}

			size := fmt.Sprintf("%*s", colSize, utils.FormatSize(item.Size))
			b.WriteString(fmt.Sprintf("%s%s %-*s %s\n", cursor, icon, nameWidth, name, SizeStyle.Render(size)))
		}

		if len(state.items) > visible {
			b.WriteString(MutedStyle.Render(fmt.Sprintf("\n  [%d/%d]", state.cursor+1, len(state.items))))
		}
	}

	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("↑↓ Navigate  enter Enter folder  esc/backspace Back  q Quit"))
	return b.String()
}
