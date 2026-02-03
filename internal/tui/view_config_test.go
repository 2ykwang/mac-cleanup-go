package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/userconfig"
)

func newTestConfig(t *testing.T) *types.Config {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	tmpFile := filepath.Join(tmpDir, "target.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("data"), 0o644))

	cfg := &types.Config{
		Categories: []types.Category{
			{
				ID:     "safe",
				Name:   "Safe",
				Safety: types.SafetyLevelSafe,
				Method: types.MethodTrash,
				Paths:  []string{tmpFile},
			},
			{
				ID:     "risky",
				Name:   "Risky",
				Safety: types.SafetyLevelRisky,
				Method: types.MethodTrash,
				Paths:  []string{tmpFile},
			},
			{
				ID:     "manual",
				Name:   "Manual",
				Safety: types.SafetyLevelSafe,
				Method: types.MethodManual,
				Paths:  []string{tmpFile},
			},
			{
				ID:     "moderate",
				Name:   "Moderate",
				Safety: types.SafetyLevelModerate,
				Method: types.MethodTrash,
				Paths:  []string{tmpFile},
			},
		},
	}
	return cfg
}

func TestConfigModel_InitItems_ExcludesRisky(t *testing.T) {
	cfg := newTestConfig(t)
	m := NewConfigModel(cfg)

	for _, item := range m.items {
		assert.NotEqual(t, types.SafetyLevelRisky, item.category.Safety,
			"risky items should not appear in items list")
	}
	// safe + manual + moderate = 3 (risky excluded)
	assert.Len(t, m.items, 3)
}

func TestConfigModel_InitItems_ManualIsDisabled(t *testing.T) {
	cfg := newTestConfig(t)
	m := NewConfigModel(cfg)

	for _, item := range m.items {
		if item.category.Method == types.MethodManual {
			assert.True(t, item.disabled, "manual items should be disabled")
		} else {
			assert.False(t, item.disabled)
		}
	}
}

func TestConfigModel_InitSelection_SkipsDisabled(t *testing.T) {
	cfg := newTestConfig(t)

	seed := &userconfig.UserConfig{
		ExcludedPaths:   make(map[string][]string),
		SelectedTargets: []string{"safe", "risky", "manual"},
	}
	require.NoError(t, seed.Save())

	m := NewConfigModel(cfg)

	assert.True(t, m.selected["safe"])
	assert.False(t, m.selected["risky"], "risky not in items, so not selected")
	assert.False(t, m.selected["manual"], "manual is disabled, so deselected")
}

func TestConfigModel_SaveSelection(t *testing.T) {
	cfg := newTestConfig(t)

	m := NewConfigModel(cfg)
	m.selected["safe"] = true

	require.NoError(t, m.saveSelection())

	loaded, err := userconfig.Load()
	require.NoError(t, err)
	assert.Equal(t, []string{"safe"}, loaded.SelectedTargets)
}

func TestConfigModel_IntroAlwaysShown(t *testing.T) {
	cfg := newTestConfig(t)

	m := NewConfigModel(cfg)
	require.True(t, m.showIntro)

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, m.showIntro, "intro dismissed after keypress")

	// Intro should always show on next launch regardless of prior dismissal
	m2 := NewConfigModel(cfg)
	assert.True(t, m2.showIntro)
}

func TestConfigModel_HandleConfigScanResult(t *testing.T) {
	cfg := newTestConfig(t)
	m := NewConfigModel(cfg)

	// Initially not scanned
	for _, item := range m.items {
		assert.False(t, item.scanned)
		assert.Equal(t, int64(0), item.size)
	}

	// Simulate scan result for "safe"
	m.Update(configScanResultMsg{categoryID: "safe", size: 1024})

	for _, item := range m.items {
		if item.category.ID == "safe" {
			assert.True(t, item.scanned)
			assert.Equal(t, int64(1024), item.size)
		}
	}
}

func TestConfigModel_HandleConfigScanResult_ZeroSize(t *testing.T) {
	cfg := newTestConfig(t)
	m := NewConfigModel(cfg)

	m.Update(configScanResultMsg{categoryID: "moderate", size: 0})

	for _, item := range m.items {
		if item.category.ID == "moderate" {
			assert.True(t, item.scanned)
			assert.Equal(t, int64(0), item.size)
		}
	}
}

func TestConfigModel_RenderItemLine_ShowsScanningIndicator(t *testing.T) {
	cfg := newTestConfig(t)
	m := NewConfigModel(cfg)
	m.width = 80

	// Before scan: should show "..."
	line := m.renderItemLine(0, m.items[0], m.width)
	assert.Contains(t, line, "...")
}

func TestConfigModel_RenderItemLine_ShowsSize(t *testing.T) {
	cfg := newTestConfig(t)
	m := NewConfigModel(cfg)
	m.width = 80

	// Simulate scan complete
	m.items[0].scanned = true
	m.items[0].size = 1024 * 1024 // 1 MB

	line := m.renderItemLine(0, m.items[0], m.width)
	assert.Contains(t, line, "1.0 MB")
}
