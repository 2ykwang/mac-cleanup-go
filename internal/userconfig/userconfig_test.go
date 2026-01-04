package userconfig

import (
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

func TestUserConfig_SaveAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "userconfig-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	cfg := &UserConfig{
		ExcludedPaths: map[string][]string{
			"chrome-cache": {"/path/one", "/path/two"},
		},
	}

	require.NoError(t, cfg.Save())

	configPath := filepath.Join(tmpDir, ".config", "mac-cleanup-go", "config.yaml")
	_, err = os.Stat(configPath)
	assert.NoError(t, err, "config file should be created")

	loaded, err := Load()
	require.NoError(t, err)
	assert.True(t, loaded.IsExcluded("chrome-cache", "/path/one"))
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "userconfig-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	configDir := filepath.Join(tmpDir, ".config", "mac-cleanup-go")
	require.NoError(t, os.MkdirAll(configDir, 0o755))

	invalidYAML := []byte("invalid: yaml: content: [unclosed")
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), invalidYAML, 0o644))

	cfg, err := Load()

	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_EmptyFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "userconfig-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	configDir := filepath.Join(tmpDir, ".config", "mac-cleanup-go")
	require.NoError(t, os.MkdirAll(configDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte{}, 0o644))

	cfg, err := Load()

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.NotNil(t, cfg.ExcludedPaths)
}

func TestUserConfig_IsExcluded_EmptyCategory(t *testing.T) {
	cfg := &UserConfig{
		ExcludedPaths: make(map[string][]string),
	}

	result := cfg.IsExcluded("nonexistent", "/some/path")

	assert.False(t, result)
}
