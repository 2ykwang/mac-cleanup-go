package tui

import "strings"

// Layout calculation helpers

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// visibleLines returns the exact number of body lines available after header/footer/extra.
// It never returns a negative value.
func (m *Model) visibleLines(header, footer string, extra int) int {
	lines := m.height - countLines(header) - countLines(footer) - extra
	if lines < 0 {
		return 0
	}
	return lines
}

func (m *Model) availableLines(header, footer string) int {
	// Keep a minimum body height so the list view stays usable.
	used := countLines(header) + countLines(footer)
	available := m.height - used
	if available < 3 {
		return 3
	}
	return available
}

// pageSize returns the number of items to move for page up/down navigation.
// Uses a reasonable default based on typical terminal heights.
func (m *Model) pageSize() int {
	// Reserve space for header/footer, use about 80% of visible area
	pageSize := (m.height - 10) * 8 / 10
	if pageSize < 5 {
		return 5
	}
	return pageSize
}

func adjustScrollFor(cursor, scroll, visible, total int) int {
	if cursor < scroll {
		return cursor
	}
	if cursor >= scroll+visible {
		return cursor - visible + 1
	}
	maxScroll := total - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		return maxScroll
	}
	if scroll < 0 {
		return 0
	}
	return scroll
}
