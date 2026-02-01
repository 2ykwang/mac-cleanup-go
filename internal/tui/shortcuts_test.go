package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterTypingShortcuts_AllKeysNotEmpty(t *testing.T) {
	for _, shortcut := range FilterTypingShortcuts {
		assert.NotEmpty(t, shortcut.Key)
		assert.NotEmpty(t, shortcut.Desc)
	}
}

func TestFormatFooter_FormatsCorrectly(t *testing.T) {
	shortcuts := []Shortcut{
		{"↑↓", "Move"},
		{"space", "Select"},
		{"q", "Quit"},
	}

	result := FormatFooter(shortcuts)

	assert.Equal(t, "↑↓ Move  space Select  q Quit", result)
}

func TestFormatFooter_EmptyShortcuts(t *testing.T) {
	result := FormatFooter([]Shortcut{})

	assert.Empty(t, result)
}

func TestFormatFooter_SingleShortcut(t *testing.T) {
	shortcuts := []Shortcut{{"q", "Quit"}}

	result := FormatFooter(shortcuts)

	assert.Equal(t, "q Quit", result)
}

// ListKeys tests

func TestListKeys_ShortHelp_ReturnsExpectedBindings(t *testing.T) {
	bindings := ListKeyMap.ShortHelp()

	assert.Len(t, bindings, 5)
	assert.Equal(t, "↑/k", bindings[0].Help().Key)
	assert.Equal(t, "space", bindings[1].Help().Key)
	assert.Equal(t, "enter", bindings[2].Help().Key)
	assert.Equal(t, "q", bindings[3].Help().Key)
}

func TestListKeys_FullHelp_ReturnsGroupedBindings(t *testing.T) {
	groups := ListKeyMap.FullHelp()

	assert.Len(t, groups, 3)
	assert.Len(t, groups[0], 2) // Up, Down
	assert.Len(t, groups[1], 2) // Select, Enter
	assert.Len(t, groups[2], 2) // Quit, Help
}

func TestListKeys_FullHelp_ContainsAllKeys(t *testing.T) {
	groups := ListKeyMap.FullHelp()

	assert.Equal(t, "↑/k", groups[0][0].Help().Key)
	assert.Equal(t, "↓/j", groups[0][1].Help().Key)
	assert.Equal(t, "space", groups[1][0].Help().Key)
	assert.Equal(t, "enter", groups[1][1].Help().Key)
	assert.Equal(t, "q", groups[2][0].Help().Key)
	assert.Equal(t, "?", groups[2][1].Help().Key)
}

// PreviewKeys tests

func TestPreviewKeys_ShortHelp_ReturnsExpectedBindings(t *testing.T) {
	bindings := PreviewKeyMap.ShortHelp()

	assert.Len(t, bindings, 7)
	assert.Equal(t, "↑/k", bindings[0].Help().Key)
	assert.Equal(t, "y", bindings[3].Help().Key)
	assert.Equal(t, "o", bindings[4].Help().Key)
	assert.Equal(t, "/", bindings[5].Help().Key)
	assert.Equal(t, "enter", bindings[6].Help().Key)
}

func TestPreviewKeys_FullHelp_ReturnsGroupedBindings(t *testing.T) {
	groups := PreviewKeyMap.FullHelp()

	assert.Len(t, groups, 6)
	assert.Len(t, groups[0], 2) // Up, Down
	assert.Len(t, groups[1], 2) // Left, Right
	assert.Len(t, groups[2], 2) // Select, Delete
	assert.Len(t, groups[3], 2) // Open, Search
	assert.Len(t, groups[4], 3) // Enter, Sort, Back
	assert.Len(t, groups[5], 2) // Quit, Help
}

func TestPreviewKeys_FullHelp_ContainsAllKeys(t *testing.T) {
	groups := PreviewKeyMap.FullHelp()

	assert.Equal(t, "←/h", groups[1][0].Help().Key)
	assert.Equal(t, "→/l", groups[1][1].Help().Key)
	assert.Equal(t, "o", groups[3][0].Help().Key)
	assert.Equal(t, "/", groups[3][1].Help().Key)
	assert.Equal(t, "enter", groups[4][0].Help().Key)
	assert.Equal(t, "s", groups[4][1].Help().Key)
	assert.Equal(t, "esc", groups[4][2].Help().Key)
}

// ConfirmKeys tests

func TestConfirmKeys_ShortHelp_ReturnsExpectedBindings(t *testing.T) {
	bindings := ConfirmKeyMap.ShortHelp()

	assert.Len(t, bindings, 5)
	assert.Equal(t, "↑/↓", bindings[0].Help().Key)
	assert.Equal(t, "←/→/tab", bindings[1].Help().Key)
	assert.Equal(t, "enter", bindings[2].Help().Key)
	assert.Equal(t, "esc", bindings[3].Help().Key)
	assert.Equal(t, "?", bindings[4].Help().Key)
}

func TestConfirmKeys_FullHelp_ReturnsGroupedBindings(t *testing.T) {
	groups := ConfirmKeyMap.FullHelp()

	assert.Len(t, groups, 1)
	assert.Len(t, groups[0], 5)
}

func TestConfirmKeys_FullHelp_ContainsAllKeys(t *testing.T) {
	groups := ConfirmKeyMap.FullHelp()

	assert.Equal(t, "↑/↓", groups[0][0].Help().Key)
	assert.Equal(t, "scroll", groups[0][0].Help().Desc)
	assert.Equal(t, "←/→/tab", groups[0][1].Help().Key)
	assert.Equal(t, "switch", groups[0][1].Help().Desc)
	assert.Equal(t, "enter", groups[0][2].Help().Key)
	assert.Equal(t, "select", groups[0][2].Help().Desc)
	assert.Equal(t, "esc", groups[0][3].Help().Key)
	assert.Equal(t, "cancel", groups[0][3].Help().Desc)
}

// ReportKeys tests

func TestReportKeys_ShortHelp_ReturnsExpectedBindings(t *testing.T) {
	bindings := ReportKeyMap.ShortHelp()

	assert.Len(t, bindings, 4)
	assert.Equal(t, "↑/k", bindings[0].Help().Key)
	assert.Equal(t, "enter", bindings[1].Help().Key)
	assert.Equal(t, "q", bindings[2].Help().Key)
}

func TestReportKeys_FullHelp_ReturnsGroupedBindings(t *testing.T) {
	groups := ReportKeyMap.FullHelp()

	assert.Len(t, groups, 2)
	assert.Len(t, groups[0], 2) // Up, Down
	assert.Len(t, groups[1], 3) // Enter, Quit, Help
}

func TestReportKeys_FullHelp_ContainsAllKeys(t *testing.T) {
	groups := ReportKeyMap.FullHelp()

	assert.Equal(t, "↑/k", groups[0][0].Help().Key)
	assert.Equal(t, "↓/j", groups[0][1].Help().Key)
	assert.Equal(t, "enter", groups[1][0].Help().Key)
	assert.Equal(t, "rescan", groups[1][0].Help().Desc)
}
