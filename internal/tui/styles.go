package tui

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	ColorPrimary   = lipgloss.Color("#7C3AED")
	ColorSecondary = lipgloss.Color("#06B6D4")
	ColorSuccess   = lipgloss.Color("#10B981")
	ColorWarning   = lipgloss.Color("#F59E0B")
	ColorDanger    = lipgloss.Color("#EF4444")
	ColorMuted     = lipgloss.Color("#6B7280")
	ColorText      = lipgloss.Color("#F9FAFB")
	ColorBorder    = lipgloss.Color("#374151")
)

// Styles
var (
	TextStyle     = lipgloss.NewStyle().Foreground(ColorText)
	MutedStyle    = lipgloss.NewStyle().Foreground(ColorMuted)
	SuccessStyle  = lipgloss.NewStyle().Foreground(ColorSuccess)
	WarningStyle  = lipgloss.NewStyle().Foreground(ColorWarning)
	DangerStyle   = lipgloss.NewStyle().Foreground(ColorDanger)
	SelectedStyle = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	CursorStyle   = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
	SizeStyle     = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
	HelpStyle     = lipgloss.NewStyle().Foreground(ColorMuted).MarginTop(1)
	HeaderStyle   = lipgloss.NewStyle().Bold(true).Foreground(ColorText).Background(ColorPrimary).Padding(0, 2).MarginBottom(1)
	DividerStyle  = lipgloss.NewStyle().Foreground(ColorBorder)
)

func Divider(width int) string {
	line := ""
	for i := 0; i < width; i++ {
		line += "─"
	}
	return DividerStyle.Render(line)
}
