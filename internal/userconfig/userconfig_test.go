package userconfig

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_NoConfigFile(t *testing.T) {
	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.NotNil(t, cfg.ExcludedPaths, "ExcludedPaths should be initialized")
	assert.NotNil(t, cfg.SelectedTargets, "SelectedTargets should be initialized")
}

func TestUserConfig_ExcludedPaths(t *testing.T) {
	cfg := &UserConfig{
		ExcludedPaths: make(map[string][]string),
	}

	paths := cfg.GetExcludedPaths("chrome-cache")
	assert.Empty(t, paths, "expected no excluded paths initially")
	assert.False(t, cfg.IsExcluded("chrome-cache", "/some/path"), "path should not be excluded initially")

	cfg.SetExcludedPaths("chrome-cache", []string{"/path/one", "/path/two"})

	paths = cfg.GetExcludedPaths("chrome-cache")
	assert.Len(t, paths, 2)
	assert.True(t, cfg.IsExcluded("chrome-cache", "/path/one"))
	assert.True(t, cfg.IsExcluded("chrome-cache", "/path/two"))
	assert.False(t, cfg.IsExcluded("chrome-cache", "/path/three"))
	assert.False(t, cfg.IsExcluded("safari-cache", "/path/one"), "different category should not be affected")
}

func TestUserConfig_SetExcludedPaths_Empty(t *testing.T) {
	cfg := &UserConfig{
		ExcludedPaths: make(map[string][]string),
	}

	cfg.SetExcludedPaths("chrome-cache", []string{"/path/one"})
	assert.Len(t, cfg.GetExcludedPaths("chrome-cache"), 1)

	cfg.SetExcludedPaths("chrome-cache", []string{})
	assert.Empty(t, cfg.GetExcludedPaths("chrome-cache"), "paths should be cleared")
}

func TestUserConfig_SelectedTargets(t *testing.T) {
	cfg := &UserConfig{
		ExcludedPaths: make(map[string][]string),
	}

	cfg.SetSelectedTargets([]string{"system-cache", "browser-chrome"})
	assert.Equal(t, []string{"system-cache", "browser-chrome"}, cfg.GetSelectedTargets())
}

func TestUserConfig_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := &UserConfig{
		ExcludedPaths: map[string][]string{
			"chrome-cache": {"/path/one", "/path/two"},
		},
		SelectedTargets: []string{"chrome-cache", "system-cache"},
	}

	require.NoError(t, cfg.Save())

	configPath := filepath.Join(tmpDir, ".config", "mac-cleanup-go", "config.yaml")
	_, err := os.Stat(configPath)
	assert.NoError(t, err, "config file should be created")

	loaded, err := Load()
	require.NoError(t, err)
	assert.True(t, loaded.IsExcluded("chrome-cache", "/path/one"))
	assert.Equal(t, []string{"chrome-cache", "system-cache"}, loaded.SelectedTargets)
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".config", "mac-cleanup-go")
	require.NoError(t, os.MkdirAll(configDir, 0o755))

	invalidYAML := []byte("invalid: yaml: content: [unclosed")
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), invalidYAML, 0o644))

	cfg, err := Load()

	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".config", "mac-cleanup-go")
	require.NoError(t, os.MkdirAll(configDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte{}, 0o644))

	cfg, err := Load()

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.NotNil(t, cfg.ExcludedPaths)
	assert.NotNil(t, cfg.SelectedTargets)
}

func TestUserConfig_ExcludedPathsMap(t *testing.T) {
	cfg := &UserConfig{
		ExcludedPaths: map[string][]string{
			"logs":  {"/path/a", "/path/b"},
			"cache": {"/path/c"},
		},
	}

	result := cfg.ExcludedPathsMap()

	assert.Len(t, result, 2)
	assert.True(t, result["logs"]["/path/a"])
	assert.True(t, result["logs"]["/path/b"])
	assert.True(t, result["cache"]["/path/c"])
	assert.False(t, result["logs"]["/path/unknown"])
}

func TestUserConfig_ExcludedPathsMap_Empty(t *testing.T) {
	cfg := &UserConfig{
		ExcludedPaths: make(map[string][]string),
	}

	result := cfg.ExcludedPathsMap()

	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestUserConfig_IsExcluded_EmptyCategory(t *testing.T) {
	cfg := &UserConfig{
		ExcludedPaths: make(map[string][]string),
	}

	result := cfg.IsExcluded("nonexistent", "/some/path")

	assert.False(t, result)
}

func TestLoad_ReadPermissionError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".config", "mac-cleanup-go")
	require.NoError(t, os.MkdirAll(configDir, 0o755))

	configFile := filepath.Join(configDir, "config.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte("test: data"), 0o644))

	original := osReadFile
	defer func() { osReadFile = original }()
	osReadFile = func(_ string) ([]byte, error) {
		return nil, fs.ErrPermission
	}

	cfg, err := Load()

	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestSave_MkdirAllError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	original := osMkdirAll
	defer func() { osMkdirAll = original }()
	osMkdirAll = func(_ string, _ fs.FileMode) error {
		return errors.New("permission denied")
	}

	cfg := &UserConfig{ExcludedPaths: make(map[string][]string)}

	err := cfg.Save()

	assert.Error(t, err)
}

func TestSave_WriteFileError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	original := osWriteFile
	defer func() { osWriteFile = original }()
	osWriteFile = func(_ string, _ []byte, _ fs.FileMode) error {
		return errors.New("permission denied")
	}

	cfg := &UserConfig{ExcludedPaths: make(map[string][]string)}

	err := cfg.Save()

	assert.Error(t, err)
}
