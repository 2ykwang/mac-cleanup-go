package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"

	"github.com/2ykwang/mac-cleanup-go/internal/styles"
)

// newStyledHelp returns a help.Model whose styles render keys as keycaps
func newStyledHelp(theme styles.Styles) help.Model {
	h := help.New()
	h.Styles = helpStyles(theme)
	return h
}

func helpStyles(theme styles.Styles) help.Styles {
	key := lipgloss.NewStyle().Foreground(theme.Text).Bold(true)
	muted := lipgloss.NewStyle().Foreground(theme.Muted)

	return help.Styles{
		Ellipsis:       muted,
		ShortKey:       key,
		ShortDesc:      muted,
		ShortSeparator: muted,
		FullKey:        key,
		FullDesc:       muted,
		FullSeparator:  muted,
	}
}
