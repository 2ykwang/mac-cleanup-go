// Package styles provides background-aware terminal styles. Construct
// a bundle with New for the appropriate light or dark variant.
package styles

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

type Styles struct {
	// Palette. Primary and Danger are static; the rest adapt to background.
	Primary, Danger,
	Secondary, Success, Warning,
	Muted, Text, Border color.Color

	// Composed styles.
	TextStyle               lipgloss.Style
	MutedStyle              lipgloss.Style
	SuccessStyle            lipgloss.Style
	WarningStyle            lipgloss.Style
	DangerStyle             lipgloss.Style
	SelectedStyle           lipgloss.Style
	CursorStyle             lipgloss.Style
	SizeStyle               lipgloss.Style
	HelpStyle               lipgloss.Style
	HeaderStyle             lipgloss.Style
	DividerStyle            lipgloss.Style
	ButtonStyle             lipgloss.Style
	ButtonActiveStyle       lipgloss.Style
	ButtonDangerStyle       lipgloss.Style
	ButtonDangerActiveStyle lipgloss.Style
	SectionActiveNameStyle  lipgloss.Style
}

// New returns a Styles configured for the given terminal background.
func New(isDark bool) Styles {
	ld := lipgloss.LightDark(isDark)

	s := Styles{
		Primary:   lipgloss.Color("#7C3AED"),
		Danger:    lipgloss.Color("#EF4444"),
		Secondary: ld(lipgloss.Color("#0891B2"), lipgloss.Color("#22D3EE")), // cyan-600 / cyan-400
		Success:   ld(lipgloss.Color("#059669"), lipgloss.Color("#34D399")), // emerald-600 / emerald-400
		Warning:   ld(lipgloss.Color("#D97706"), lipgloss.Color("#FBBF24")), // amber-600 / amber-400
		Muted:     ld(lipgloss.Color("#4B5563"), lipgloss.Color("#B5B5B5")), // gray-600 / 256-color 250
		Text:      ld(lipgloss.Color("#111827"), lipgloss.Color("#F9FAFB")), // gray-900 / gray-50
		Border:    ld(lipgloss.Color("#D1D5DB"), lipgloss.Color("#909090")), // gray-300 / 256-color 242
	}

	s.TextStyle = lipgloss.NewStyle().Foreground(s.Text)
	s.MutedStyle = lipgloss.NewStyle().Foreground(s.Muted)
	s.SuccessStyle = lipgloss.NewStyle().Foreground(s.Success)
	s.WarningStyle = lipgloss.NewStyle().Foreground(s.Warning)
	s.DangerStyle = lipgloss.NewStyle().Foreground(s.Danger)
	s.SelectedStyle = lipgloss.NewStyle().Foreground(s.Primary).Bold(true)
	s.CursorStyle = lipgloss.NewStyle().Foreground(s.Secondary).Bold(true)
	s.SizeStyle = lipgloss.NewStyle().Foreground(s.Secondary).Bold(true)
	s.HelpStyle = lipgloss.NewStyle().Foreground(s.Muted).MarginTop(1)
	s.HeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(s.Text).Background(s.Primary).Padding(0, 2).MarginBottom(1)
	s.DividerStyle = lipgloss.NewStyle().Foreground(s.Border)
	s.ButtonStyle = lipgloss.NewStyle().Padding(0, 2).Border(lipgloss.RoundedBorder()).BorderForeground(s.Border).Foreground(s.Text)
	s.ButtonActiveStyle = s.ButtonStyle.BorderForeground(s.Primary).Foreground(s.Text).Bold(true)
	s.ButtonDangerStyle = s.ButtonStyle
	s.ButtonDangerActiveStyle = s.ButtonStyle.BorderForeground(s.Danger).Foreground(s.Danger).Bold(true)
	s.SectionActiveNameStyle = lipgloss.NewStyle().Bold(true).Foreground(s.Primary)

	return s
}

// Divider renders a horizontal line of the given width.
func (s Styles) Divider(width int) string {
	if width <= 0 {
		return ""
	}
	return s.DividerStyle.Render(strings.Repeat("─", width))
}

func (s Styles) SafetyDot(level types.SafetyLevel) string {
	switch level {
	case types.SafetyLevelSafe:
		return s.SuccessStyle.Render("●")
	case types.SafetyLevelModerate:
		return s.WarningStyle.Render("●")
	case types.SafetyLevelRisky:
		return s.DangerStyle.Render("●")
	default:
		return s.MutedStyle.Render("●")
	}
}
