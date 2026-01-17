package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/2ykwang/mac-cleanup-go/internal/version"
)

// checkVersion starts async version check
func (m *Model) checkVersion() tea.Cmd {
	currentVersion := m.currentVersion
	return func() tea.Msg {
		result := version.CheckForUpdate(currentVersion)
		return versionCheckMsg{
			latestVersion:   result.LatestVersion,
			updateAvailable: result.UpdateAvailable,
		}
	}
}
