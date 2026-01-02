package userconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_NoConfigFile(t *testing.T) {
	// Load should return empty config when file doesn't exist
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	if cfg.ExcludedPaths == nil {
		t.Error("ExcludedPaths should be initialized")
	}
}

func TestUserConfig_ExcludedPaths(t *testing.T) {
	cfg := &UserConfig{
		ExcludedPaths: make(map[string][]string),
	}

	// Initially no excluded paths
	paths := cfg.GetExcludedPaths("chrome-cache")
	if len(paths) != 0 {
		t.Error("Expected no excluded paths initially")
	}

	if cfg.IsExcluded("chrome-cache", "/some/path") {
		t.Error("Path should not be excluded initially")
	}

	// Set excluded paths
	cfg.SetExcludedPaths("chrome-cache", []string{"/path/one", "/path/two"})

	paths = cfg.GetExcludedPaths("chrome-cache")
	if len(paths) != 2 {
		t.Errorf("Expected 2 excluded paths, got %d", len(paths))
	}

	if !cfg.IsExcluded("chrome-cache", "/path/one") {
		t.Error("/path/one should be excluded")
	}

	if !cfg.IsExcluded("chrome-cache", "/path/two") {
		t.Error("/path/two should be excluded")
	}

	if cfg.IsExcluded("chrome-cache", "/path/three") {
		t.Error("/path/three should not be excluded")
	}

	// Different category should not be affected
	if cfg.IsExcluded("safari-cache", "/path/one") {
		t.Error("safari-cache should not have excluded paths")
	}
}

func TestUserConfig_SetExcludedPaths_Empty(t *testing.T) {
	cfg := &UserConfig{
		ExcludedPaths: make(map[string][]string),
	}

	// Set some paths
	cfg.SetExcludedPaths("chrome-cache", []string{"/path/one"})
	if len(cfg.GetExcludedPaths("chrome-cache")) != 1 {
		t.Error("Expected 1 excluded path")
	}

	// Clear by setting empty slice
	cfg.SetExcludedPaths("chrome-cache", []string{})
	paths := cfg.GetExcludedPaths("chrome-cache")
	if len(paths) != 0 {
		t.Errorf("Expected 0 excluded paths after clearing, got %d", len(paths))
	}
}

func TestUserConfig_SaveAndLoad(t *testing.T) {
	// Create temp directory for test config
	tmpDir, err := os.MkdirTemp("", "userconfig-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override config path for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create and save config
	cfg := &UserConfig{
		ExcludedPaths: map[string][]string{
			"chrome-cache": {"/path/one", "/path/two"},
		},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify file was created
	configPath := filepath.Join(tmpDir, ".config", "mac-cleanup-go", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load and verify
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if !loaded.IsExcluded("chrome-cache", "/path/one") {
		t.Error("Loaded config should have excluded path")
	}
}
