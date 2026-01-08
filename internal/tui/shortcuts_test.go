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


