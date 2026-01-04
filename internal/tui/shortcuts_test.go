package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestViewShortcuts_AllKeysNotEmpty(t *testing.T) {
	for view, groups := range ViewShortcuts {
		for _, group := range groups {
			assert.NotEmpty(t, group.Name, "Group name should not be empty for view %d", view)
			assert.NotEmpty(t, group.Shortcuts, "Shortcuts should not be empty for group %s", group.Name)

			for _, shortcut := range group.Shortcuts {
				assert.NotEmpty(t, shortcut.Key, "Key should not be empty in group %s", group.Name)
				assert.NotEmpty(t, shortcut.Desc, "Desc should not be empty for key %s", shortcut.Key)
			}
		}
	}
}

func TestDrillDownShortcuts_AllKeysNotEmpty(t *testing.T) {
	for _, group := range DrillDownShortcuts {
		assert.NotEmpty(t, group.Name)
		assert.NotEmpty(t, group.Shortcuts)

		for _, shortcut := range group.Shortcuts {
			assert.NotEmpty(t, shortcut.Key)
			assert.NotEmpty(t, shortcut.Desc)
		}
	}
}

func TestFilterTypingShortcuts_AllKeysNotEmpty(t *testing.T) {
	for _, shortcut := range FilterTypingShortcuts {
		assert.NotEmpty(t, shortcut.Key)
		assert.NotEmpty(t, shortcut.Desc)
	}
}

func TestFooterShortcuts_ReturnsShortcuts(t *testing.T) {
	shortcuts := FooterShortcuts(ViewList)

	assert.NotEmpty(t, shortcuts)
	assert.LessOrEqual(t, len(shortcuts), 6, "Footer should have at most 6 shortcuts")
}

func TestFooterShortcuts_UnknownViewReturnsHelpOnly(t *testing.T) {
	shortcuts := FooterShortcuts(View(999))

	assert.Len(t, shortcuts, 1)
	assert.Equal(t, "?", shortcuts[0].Key)
	assert.Equal(t, "Help", shortcuts[0].Desc)
}

func TestFormatFooter_FormatsCorrectly(t *testing.T) {
	shortcuts := []Shortcut{
		{"↑↓", "Move"},
		{"space", "Select"},
		{"?", "Help"},
	}

	result := FormatFooter(shortcuts)

	assert.Equal(t, "↑↓ Move  space Select  ? Help", result)
}

func TestFormatFooter_EmptyShortcuts(t *testing.T) {
	result := FormatFooter([]Shortcut{})

	assert.Empty(t, result)
}

func TestDrillDownFooterShortcuts_ReturnsShortcuts(t *testing.T) {
	shortcuts := DrillDownFooterShortcuts()

	assert.NotEmpty(t, shortcuts)
	assert.LessOrEqual(t, len(shortcuts), 6)
}

func TestFooterShortcuts_AllViewsHaveHelpKey(t *testing.T) {
	views := []View{ViewList, ViewPreview, ViewConfirm, ViewReport}

	for _, view := range views {
		shortcuts := FooterShortcuts(view)
		hasHelp := false
		for _, s := range shortcuts {
			if s.Key == "?" {
				hasHelp = true
				break
			}
		}
		assert.True(t, hasHelp, "View %d should have ? Help shortcut in footer", view)
	}
}
