package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func TestHelp_ToggleHelp_OpensAndCloses(t *testing.T) {
	m := newTestModel()
	assert.False(t, m.showHelp)

	m.toggleHelp()
	assert.True(t, m.showHelp)
	assert.Equal(t, 0, m.helpScroll)

	m.toggleHelp()
	assert.False(t, m.showHelp)
}

func TestHelp_ToggleHelp_ResetsScroll(t *testing.T) {
	m := newTestModel()
	m.showHelp = true
	m.helpScroll = 5

	m.toggleHelp()
	assert.False(t, m.showHelp)
	assert.Equal(t, 0, m.helpScroll)
}

func TestHelp_HandleHelpKey_EscCloses(t *testing.T) {
	m := newTestModel()
	m.showHelp = true
	m.helpScroll = 3

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyEscape})

	assert.False(t, m.showHelp)
	assert.Equal(t, 0, m.helpScroll)
}

func TestHelp_HandleHelpKey_QuestionMarkCloses(t *testing.T) {
	m := newTestModel()
	m.showHelp = true

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	assert.False(t, m.showHelp)
}

func TestHelp_HandleHelpKey_EnterCloses(t *testing.T) {
	m := newTestModel()
	m.showHelp = true

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyEnter})

	assert.False(t, m.showHelp)
}

func TestHelp_HandleHelpKey_ScrollDown(t *testing.T) {
	m := newTestModel()
	m.showHelp = true
	m.helpScroll = 0

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.helpScroll)

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 2, m.helpScroll)
}

func TestHelp_HandleHelpKey_ScrollUp(t *testing.T) {
	m := newTestModel()
	m.showHelp = true
	m.helpScroll = 3

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 2, m.helpScroll)

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 1, m.helpScroll)
}

func TestHelp_HandleHelpKey_ScrollUpClampsToZero(t *testing.T) {
	m := newTestModel()
	m.showHelp = true
	m.helpScroll = 0

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, m.helpScroll)
}

func TestHelp_HandleKey_RoutesToHelpWhenOpen(t *testing.T) {
	m := newTestModel()
	m.view = ViewList
	m.showHelp = true
	m.helpScroll = 2

	// "esc" should close help, not affect the list view
	m.handleKey(tea.KeyMsg{Type: tea.KeyEscape})

	assert.False(t, m.showHelp)
	assert.Equal(t, ViewList, m.view) // view unchanged
}

func TestHelp_HandleKey_PreservesViewWhenHelpOpen(t *testing.T) {
	views := []View{ViewList, ViewPreview, ViewReport}

	for _, v := range views {
		m := newTestModel()
		m.view = v
		m.showHelp = true

		// down key should scroll help, not change underlying view state
		m.handleKey(tea.KeyMsg{Type: tea.KeyDown})

		assert.Equal(t, v, m.view, "view should be preserved for %v", v)
		assert.True(t, m.showHelp, "help should remain open for %v", v)
	}
}

func TestHelp_HelpDialog_NoPanic(t *testing.T) {
	m := newTestModel()
	m.width = 80
	m.height = 24

	// Should not panic
	output := m.helpDialog()
	assert.NotEmpty(t, output)
}

func TestHelp_HelpDialog_ContainsHelp(t *testing.T) {
	m := newTestModel()
	m.width = 80
	m.height = 24

	output := m.helpDialog()
	assert.True(t, strings.Contains(output, "Help"), "helpDialog should contain 'Help'")
}

func TestHelp_HelpDialog_SmallTerminal(t *testing.T) {
	m := newTestModel()
	m.width = 30
	m.height = 10

	// Should not panic even with very small terminal
	output := m.helpDialog()
	assert.NotEmpty(t, output)
}

func TestHelp_HelpDialog_ZeroSize(t *testing.T) {
	m := newTestModel()
	m.width = 0
	m.height = 0

	// Should not panic
	output := m.helpDialog()
	assert.NotEmpty(t, output)
}

func TestHelp_View_OverlayOnViewList(t *testing.T) {
	m := newTestModel()
	m.view = ViewList
	m.showHelp = true
	m.width = 80
	m.height = 24

	output := m.View()
	assert.True(t, strings.Contains(output, "Help"), "overlay should contain 'Help' on ViewList")
}

func TestHelp_View_OverlayOnViewPreview(t *testing.T) {
	m := newTestModel()
	m.view = ViewPreview
	m.showHelp = true
	m.width = 80
	m.height = 24
	// Need at least one selected result for preview
	m.results = []*types.ScanResult{
		{Category: types.Category{ID: "test", Name: "Test"}, TotalSize: 100},
	}
	m.selected = map[string]bool{"test": true}
	m.selectedOrder = []string{"test"}
	m.previewCatID = "test"

	output := m.View()
	assert.True(t, strings.Contains(output, "Help"), "overlay should contain 'Help' on ViewPreview")
}

func TestHelp_View_OverlayOnViewReport(t *testing.T) {
	m := newTestModel()
	m.view = ViewReport
	m.showHelp = true
	m.width = 80
	m.height = 24
	m.report = &types.Report{}

	output := m.View()
	assert.True(t, strings.Contains(output, "Help"), "overlay should contain 'Help' on ViewReport")
}
