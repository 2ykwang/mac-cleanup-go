package tui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

const (
	colName = 28
	colSize = 12
	colNum  = 8
	colAge  = 6 // Age column width for FormatAge output (e.g., "2mo")

	// listPrefixWidth: cursor(2) + checkbox(3) + space(1) + dot(1) + space(1)
	listPrefixWidth = 8
	// previewPrefixWidth: cursor(2) + checkbox(3) + space(1) + icon(1) + space(1)
	previewPrefixWidth = 8
)

// Helper functions
func formatSize(bytes int64) string {
	return utils.FormatSize(bytes)
}

func safetyDot(level types.SafetyLevel) string {
	switch level {
	case types.SafetyLevelSafe:
		return SuccessStyle.Render("●")
	case types.SafetyLevelModerate:
		return WarningStyle.Render("●")
	case types.SafetyLevelRisky:
		return DangerStyle.Render("●")
	default:
		return MutedStyle.Render("●")
	}
}

func safetyBadge(level types.SafetyLevel) string {
	switch level {
	case types.SafetyLevelSafe:
		return SuccessStyle.Render("[Safe]")
	case types.SafetyLevelModerate:
		return WarningStyle.Render("[Moderate]")
	case types.SafetyLevelRisky:
		return DangerStyle.Render("[Risky]")
	default:
		return ""
	}
}

func (m *Model) methodBadge(method types.CleanupMethod) string {
	switch method {
	case types.MethodManual:
		return WarningStyle.Render("[Manual]")
	case types.MethodBuiltin:
		return "" // Internal implementation detail, not shown to user
	default:
		return "" // Trash is default, no badge needed
	}
}

// shortenPath truncates path to fit within maxWidth display columns.
func shortenPath(path string, maxWidth int) string {
	home, _ := filepath.Abs(utils.ExpandPath("~"))
	if strings.HasPrefix(path, home) {
		path = "~" + path[len(home):]
	}
	return truncateToWidth(path, maxWidth, true)
}

// truncateToWidth truncates string to fit within maxWidth display columns.
func truncateToWidth(s string, maxWidth int, fromEnd bool) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}

	runes := []rune(s)
	if fromEnd {
		// Keep end, add "..." prefix
		prefix := "..."
		prefixWidth := lipgloss.Width(prefix)
		targetWidth := maxWidth - prefixWidth

		for i := len(runes) - 1; i >= 0; i-- {
			substr := string(runes[i:])
			if lipgloss.Width(substr) <= targetWidth {
				continue
			}

			return prefix + string(runes[i+1:])
		}
		return prefix + s
	}

	// Keep star add ".." suffix
	suffix := ".."
	suffixWidth := lipgloss.Width(suffix)
	targetWidth := maxWidth - suffixWidth

	for i := 1; i <= len(runes); i++ {
		substr := string(runes[:i])
		if lipgloss.Width(substr) > targetWidth {
			return string(runes[:i-1]) + suffix
		}
	}
	return s
}

// padToWidth pads string with spaces to reach exactly targetWidth display columns.
func padToWidth(s string, targetWidth int) string {
	currentWidth := lipgloss.Width(s)
	if currentWidth >= targetWidth {
		return s
	}
	return s + strings.Repeat(" ", targetWidth-currentWidth)
}
