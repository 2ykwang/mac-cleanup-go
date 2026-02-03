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

// Function variables for testing
var (
	osReadFile  = os.ReadFile
	osMkdirAll  = os.MkdirAll
	osWriteFile = os.WriteFile
)

// UserConfig stores user preferences
type UserConfig struct {
	// ExcludedPaths maps category ID to list of excluded paths
	ExcludedPaths map[string][]string `yaml:"excluded_paths,omitempty"`
	// SelectedTargets stores CLI-selected category IDs
	SelectedTargets []string `yaml:"selected_targets,omitempty"`
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

	data, err := osReadFile(path)
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
	if cfg.SelectedTargets == nil {
		cfg.SelectedTargets = make([]string, 0)
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
	if err := osMkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return osWriteFile(path, data, 0o644)
}

// SetExcludedPaths sets excluded paths for a category
func (c *UserConfig) SetExcludedPaths(categoryID string, paths []string) {
	if len(paths) == 0 {
		delete(c.ExcludedPaths, categoryID)
	} else {
		c.ExcludedPaths[categoryID] = paths
	}
}

// SetSelectedTargets sets selected target IDs for CLI runs.
func (c *UserConfig) SetSelectedTargets(targets []string) {
	c.SelectedTargets = append([]string(nil), targets...)
}

// GetSelectedTargets returns selected target IDs for CLI runs.
func (c *UserConfig) GetSelectedTargets() []string {
	return c.SelectedTargets
}

// GetExcludedPaths gets excluded paths for a category
func (c *UserConfig) GetExcludedPaths(categoryID string) []string {
	return c.ExcludedPaths[categoryID]
}

// ExcludedPathsMap returns excluded paths as a nested bool map
func (c *UserConfig) ExcludedPathsMap() map[string]map[string]bool {
	result := make(map[string]map[string]bool, len(c.ExcludedPaths))
	for catID, paths := range c.ExcludedPaths {
		result[catID] = make(map[string]bool, len(paths))
		for _, p := range paths {
			result[catID][p] = true
		}
	}
	return result
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
