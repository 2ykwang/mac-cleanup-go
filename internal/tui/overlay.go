package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m *Model) overlayCentered(base, overlay string) string {
	if m.width <= 0 || m.height <= 0 {
		return base
	}

	baseLines := normalizeOverlayLines(base, m.width, m.height)
	overlayLines := splitLines(overlay)
	overlayWidth := lipgloss.Width(overlay)
	overlayHeight := len(overlayLines)
	if overlayWidth <= 0 || overlayHeight <= 0 {
		return strings.Join(baseLines, "\n")
	}
	if overlayWidth > m.width {
		overlayWidth = m.width
	}
	if overlayHeight > m.height {
		overlayHeight = m.height
	}

	x := (m.width - overlayWidth) / 2
	y := (m.height - overlayHeight) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	for i := 0; i < overlayHeight && y+i < len(baseLines); i++ {
		overlayLine := ""
		if i < len(overlayLines) {
			overlayLine = overlayLines[i]
		}
		overlayLine = ansi.Truncate(overlayLine, overlayWidth, "")
		overlayLine = padToWidth(overlayLine, overlayWidth)
		line := baseLines[y+i]
		left := ansi.Cut(line, 0, x)
		right := ansi.Cut(line, x+overlayWidth, m.width)
		baseLines[y+i] = left + overlayLine + right
	}

	return strings.Join(baseLines, "\n")
}

func normalizeOverlayLines(s string, width, height int) []string {
	lines := splitLines(s)
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], width, "")
		lines[i] = padToWidth(lines[i], width)
	}

	blank := strings.Repeat(" ", width)
	for len(lines) < height {
		lines = append(lines, blank)
	}
	if len(lines) > height {
		lines = lines[:height]
	}

	return lines
}

func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, "\n")
}
