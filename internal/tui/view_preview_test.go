package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func TestRenderBuiltinItem_SafeItem(t *testing.T) {
	m := newTestModelWithConfig()
	item := types.CleanableItem{
		Name:       "nginx:latest",
		Size:       100 * 1024 * 1024,
		Path:       "abc123",
		SafetyHint: types.SafetyHintSafe,
		Columns: []types.Column{
			{Header: "Repository", Value: "nginx"},
			{Header: "Tag", Value: "latest"},
			{Header: "Status", Value: "unused"},
		},
	}

	result := m.renderBuiltinItem("  ", "[x]", item, false, false, 20, 10)

	assert.Contains(t, result, "nginx:latest")
	assert.Contains(t, result, "[unused]")
}

func TestRenderBuiltinItem_WarningItem(t *testing.T) {
	m := newTestModelWithConfig()
	item := types.CleanableItem{
		Name:       "postgres:15",
		Size:       200 * 1024 * 1024,
		Path:       "def456",
		SafetyHint: types.SafetyHintWarning,
		Columns: []types.Column{
			{Header: "Repository", Value: "postgres"},
			{Header: "Tag", Value: "15"},
			{Header: "Status", Value: "in-use"},
		},
	}

	result := m.renderBuiltinItem("▸ ", "[x]", item, true, false, 20, 10)

	assert.Contains(t, result, "postgres:15")
	assert.Contains(t, result, "[in-use]")
}

func TestRenderBuiltinItem_ExcludedItem(t *testing.T) {
	m := newTestModelWithConfig()
	item := types.CleanableItem{
		Name:       "node:20",
		Size:       50 * 1024 * 1024,
		Path:       "ghi789",
		SafetyHint: types.SafetyHintSafe,
		Columns: []types.Column{
			{Header: "Status", Value: "dangling"},
		},
	}

	result := m.renderBuiltinItem("  ", "[ ]", item, false, true, 20, 10)

	assert.Contains(t, result, "node:20")
	assert.Contains(t, result, "[dangling]")
}

func TestRenderBuiltinItem_NoStatusColumn(t *testing.T) {
	m := newTestModelWithConfig()
	item := types.CleanableItem{
		Name:       "myimage",
		Size:       10 * 1024 * 1024,
		Path:       "jkl012",
		SafetyHint: types.SafetyHintSafe,
		Columns: []types.Column{
			{Header: "Repository", Value: "myimage"},
		},
	}

	result := m.renderBuiltinItem("  ", "[x]", item, false, false, 20, 10)

	assert.Contains(t, result, "myimage")
	// No [status] suffix when Status column is missing
	assert.NotContains(t, result, "[unused]")
	assert.NotContains(t, result, "[dangling]")
	assert.NotContains(t, result, "[in-use]")
}
