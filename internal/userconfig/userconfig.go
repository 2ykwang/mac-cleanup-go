package userconfig

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	configDir  = ".config/mac-cleanup-go"
	configFile = "config.yaml"
)

// CustomTarget defines a user-defined cleanup target
type CustomTarget struct {
	ID       string   `yaml:"id"`
	Name     string   `yaml:"name"`
	Group    string   `yaml:"group"`
	Safety   string   `yaml:"safety"` // safe, moderate, risky
	Method   string   `yaml:"method"` // trash, permanent, command, special, manual
	Note     string   `yaml:"note,omitempty"`
	Paths    []string `yaml:"paths,omitempty"`
	Command  string   `yaml:"command,omitempty"`
	CheckCmd string   `yaml:"check_cmd,omitempty"`
}

// CategoryOverride allows partial override of category properties
type CategoryOverride struct {
	Disabled *bool    `yaml:"disabled,omitempty"`
	Paths    []string `yaml:"paths,omitempty"`
	Note     *string  `yaml:"note,omitempty"`
}

// UserConfig stores user preferences
type UserConfig struct {
	// ExcludedPaths maps category ID to list of excluded paths
	ExcludedPaths map[string][]string `yaml:"excluded_paths,omitempty"`
	// LastSelection stores the last selected category IDs for --clean mode
	LastSelection []string `yaml:"last_selection,omitempty"`
	// CustomTargets defines user-defined cleanup targets
	CustomTargets []CustomTarget `yaml:"custom_targets,omitempty"`
	// TargetOverrides overrides specific fields of existing targets (by ID)
	TargetOverrides map[string]CategoryOverride `yaml:"target_overrides,omitempty"`
}

// SetLastSelection saves the selected category IDs
func (c *UserConfig) SetLastSelection(categoryIDs []string) {
	c.LastSelection = categoryIDs
}

// GetLastSelection returns the last selected category IDs
func (c *UserConfig) GetLastSelection() []string {
	return c.LastSelection
}

// HasLastSelection checks if there's a saved selection
func (c *UserConfig) HasLastSelection() bool {
	return len(c.LastSelection) > 0
}

// configPath returns the full path to the config file
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDir, configFile), nil
}

// Load loads user config from disk
func Load() (*UserConfig, error) {
	path, err := configPath()
	if err != nil {
		return &UserConfig{ExcludedPaths: make(map[string][]string)}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &UserConfig{ExcludedPaths: make(map[string][]string)}, nil
		}
		return nil, err
	}

	var cfg UserConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.ExcludedPaths == nil {
		cfg.ExcludedPaths = make(map[string][]string)
	}
	if cfg.CustomTargets == nil {
		cfg.CustomTargets = make([]CustomTarget, 0)
	}
	if cfg.TargetOverrides == nil {
		cfg.TargetOverrides = make(map[string]CategoryOverride)
	}

	return &cfg, nil
}

// Save saves user config to disk
func (c *UserConfig) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	// Create config directory if not exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// SetExcludedPaths sets excluded paths for a category
func (c *UserConfig) SetExcludedPaths(categoryID string, paths []string) {
	if len(paths) == 0 {
		delete(c.ExcludedPaths, categoryID)
	} else {
		c.ExcludedPaths[categoryID] = paths
	}
}

// GetExcludedPaths gets excluded paths for a category
func (c *UserConfig) GetExcludedPaths(categoryID string) []string {
	return c.ExcludedPaths[categoryID]
}

// IsExcluded checks if a path is excluded for a category
func (c *UserConfig) IsExcluded(categoryID, path string) bool {
	for _, p := range c.ExcludedPaths[categoryID] {
		if p == path {
			return true
		}
	}
	return false
}
