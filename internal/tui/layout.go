package tui

import "strings"

// Layout calculation helpers

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func (m *Model) availableLines(header, footer string) int {
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

func (m *Model) adjustScrollFor(cursor, scroll, visible, _ int) int {
	if cursor < scroll {
		return cursor
	}
	if cursor >= scroll+visible {
		return cursor - visible + 1
	}
	return scroll
}
