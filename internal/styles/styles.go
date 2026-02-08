// Package styles define shared color and style constants used across TUI and CLI packages.
package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Colors — shared palette for both TUI and CLI output.
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

// Styles — common lipgloss styles.
var (
	TextStyle               = lipgloss.NewStyle().Foreground(ColorText)
	MutedStyle              = lipgloss.NewStyle().Foreground(ColorMuted)
	SuccessStyle            = lipgloss.NewStyle().Foreground(ColorSuccess)
	WarningStyle            = lipgloss.NewStyle().Foreground(ColorWarning)
	DangerStyle             = lipgloss.NewStyle().Foreground(ColorDanger)
	SelectedStyle           = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	CursorStyle             = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
	SizeStyle               = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
	HelpStyle               = lipgloss.NewStyle().Foreground(ColorMuted).MarginTop(1)
	HeaderStyle             = lipgloss.NewStyle().Bold(true).Foreground(ColorText).Background(ColorPrimary).Padding(0, 2).MarginBottom(1)
	DividerStyle            = lipgloss.NewStyle().Foreground(ColorBorder)
	ButtonStyle             = lipgloss.NewStyle().Padding(0, 2).Border(lipgloss.RoundedBorder()).BorderForeground(ColorBorder).Foreground(ColorText)
	ButtonActiveStyle       = ButtonStyle.BorderForeground(ColorPrimary).Foreground(ColorText).Bold(true)
	ButtonDangerStyle       = ButtonStyle
	ButtonDangerActiveStyle = ButtonStyle.BorderForeground(ColorDanger).Foreground(ColorDanger).Bold(true)
)

// Divider renders a horizontal line of the given width.
func Divider(width int) string {
	return DividerStyle.Render(strings.Repeat("─", width))
}
