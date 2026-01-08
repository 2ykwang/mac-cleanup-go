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
}

func TestListKeys_FullHelp_ReturnsGroupedBindings(t *testing.T) {
	groups := ListKeyMap.FullHelp()

	assert.Len(t, groups, 3)
	assert.Len(t, groups[0], 2) // Up, Down
	assert.Len(t, groups[1], 2) // Select, Enter
	assert.Len(t, groups[2], 3) // Delete, Quit, Help
}

func TestListKeys_FullHelp_ContainsAllKeys(t *testing.T) {
	groups := ListKeyMap.FullHelp()

	assert.Equal(t, "↑/k", groups[0][0].Help().Key)
	assert.Equal(t, "↓/j", groups[0][1].Help().Key)
	assert.Equal(t, "space", groups[1][0].Help().Key)
	assert.Equal(t, "enter", groups[1][1].Help().Key)
	assert.Equal(t, "y", groups[2][0].Help().Key)
	assert.Equal(t, "q", groups[2][1].Help().Key)
	assert.Equal(t, "?", groups[2][2].Help().Key)
}

// PreviewKeys tests

func TestPreviewKeys_ShortHelp_ReturnsExpectedBindings(t *testing.T) {
	bindings := PreviewKeyMap.ShortHelp()

	assert.Len(t, bindings, 5)
	assert.Equal(t, "↑/k", bindings[0].Help().Key)
	assert.Equal(t, "/", bindings[3].Help().Key)
}

func TestPreviewKeys_FullHelp_ReturnsGroupedBindings(t *testing.T) {
	groups := PreviewKeyMap.FullHelp()

	assert.Len(t, groups, 3)
	assert.Len(t, groups[0], 4) // Up, Down, Left, Right
	assert.Len(t, groups[1], 4) // Select, Enter, Back, Open
	assert.Len(t, groups[2], 4) // Search, Sort, Delete, Help
}

func TestPreviewKeys_FullHelp_ContainsAllKeys(t *testing.T) {
	groups := PreviewKeyMap.FullHelp()

	assert.Equal(t, "←/h", groups[0][2].Help().Key)
	assert.Equal(t, "→/l", groups[0][3].Help().Key)
	assert.Equal(t, "esc", groups[1][2].Help().Key)
	assert.Equal(t, "o", groups[1][3].Help().Key)
	assert.Equal(t, "s", groups[2][1].Help().Key)
}

// ConfirmKeys tests

func TestConfirmKeys_ShortHelp_ReturnsExpectedBindings(t *testing.T) {
	bindings := ConfirmKeyMap.ShortHelp()

	assert.Len(t, bindings, 3)
	assert.Equal(t, "y", bindings[0].Help().Key)
	assert.Equal(t, "n/esc", bindings[1].Help().Key)
	assert.Equal(t, "?", bindings[2].Help().Key)
}

func TestConfirmKeys_FullHelp_ReturnsGroupedBindings(t *testing.T) {
	groups := ConfirmKeyMap.FullHelp()

	assert.Len(t, groups, 1)
	assert.Len(t, groups[0], 3)
}

func TestConfirmKeys_FullHelp_ContainsAllKeys(t *testing.T) {
	groups := ConfirmKeyMap.FullHelp()

	assert.Equal(t, "y", groups[0][0].Help().Key)
	assert.Equal(t, "confirm", groups[0][0].Help().Desc)
	assert.Equal(t, "n/esc", groups[0][1].Help().Key)
	assert.Equal(t, "cancel", groups[0][1].Help().Desc)
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
