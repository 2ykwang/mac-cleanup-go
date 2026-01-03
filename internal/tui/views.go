package tui

import (
	"path/filepath"
	"strings"

	"github.com/2ykwang/mac-cleanup-go/internal/utils"
	"github.com/2ykwang/mac-cleanup-go/pkg/types"
)

const (
	colName = 28
	colSize = 12
	colNum  = 8

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
	case types.MethodCommand:
		return MutedStyle.Render("[Command]")
	case types.MethodBuiltin:
		return MutedStyle.Render("[Builtin]")
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
