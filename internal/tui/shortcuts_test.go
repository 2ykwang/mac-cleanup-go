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

// PreviewKeys tests

func TestPreviewKeys_ShortHelp_ReturnsExpectedBindings(t *testing.T) {
	bindings := PreviewKeyMap.ShortHelp()

	assert.Len(t, bindings, 5)
	assert.Equal(t, "↑/k", bindings[0].Help().Key)
	assert.Equal(t, "tab/]", bindings[1].Help().Key)
	assert.Equal(t, "enter", bindings[2].Help().Key)
	assert.Equal(t, "y", bindings[3].Help().Key)
	assert.Equal(t, "?", bindings[4].Help().Key)
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

// ReportKeys tests

func TestReportKeys_ShortHelp_ReturnsExpectedBindings(t *testing.T) {
	bindings := ReportKeyMap.ShortHelp()

	assert.Len(t, bindings, 4)
	assert.Equal(t, "↑/k", bindings[0].Help().Key)
	assert.Equal(t, "enter", bindings[1].Help().Key)
	assert.Equal(t, "q", bindings[2].Help().Key)
}
