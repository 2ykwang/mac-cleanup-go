package tui

import (
	"path/filepath"
	"strings"

	"mac-cleanup-go/internal/utils"
	"mac-cleanup-go/pkg/types"
)

const (
	colName = 28
	colSize = 12
	colNum  = 8
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
	case types.MethodCommand:
		return MutedStyle.Render("[Command]")
	case types.MethodSpecial:
		return MutedStyle.Render("[Special]")
	default:
		return "" // Trash is default, no badge needed
	}
}

func shortenPath(path string, maxLen int) string {
	home, _ := filepath.Abs(utils.ExpandPath("~"))
	if strings.HasPrefix(path, home) {
		path = "~" + path[len(home):]
	}
	if len(path) > maxLen {
		path = "..." + path[len(path)-(maxLen-3):]
	}
	return path
}
