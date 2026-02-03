package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/userconfig"
)

func TestRunner_Run_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "sample.log")
	data := []byte("cleanup")
	require.NoError(t, os.WriteFile(tmpFile, data, 0o644))

	cfg := &types.Config{
		Categories: []types.Category{
			{
				ID:     "logs",
				Name:   "Logs",
				Safety: types.SafetyLevelSafe,
				Method: types.MethodTrash,
				Paths:  []string{tmpFile},
			},
		},
	}

	userCfg := &userconfig.UserConfig{
		ExcludedPaths:   make(map[string][]string),
		SelectedTargets: []string{"logs"},
	}

	runner, err := NewRunner(cfg, userCfg)
	require.NoError(t, err)

	report, warnings, err := runner.Run(true)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	require.NotNil(t, report)
	assert.Equal(t, int64(len(data)), report.FreedSpace)
	assert.Equal(t, 1, report.CleanedItems)
}

func TestRunner_Run_NoSelection(t *testing.T) {
	cfg := &types.Config{
		Categories: []types.Category{
			{
				ID:     "logs",
				Name:   "Logs",
				Safety: types.SafetyLevelSafe,
				Method: types.MethodTrash,
				Paths:  []string{"/tmp/does-not-matter"},
			},
		},
	}

	userCfg := &userconfig.UserConfig{
		ExcludedPaths: make(map[string][]string),
	}

	runner, err := NewRunner(cfg, userCfg)
	require.NoError(t, err)

	report, warnings, err := runner.Run(true)
	assert.ErrorIs(t, err, ErrNoSelection)
	assert.Nil(t, report)
	assert.Nil(t, warnings)
}
