package config

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/2ykwang/mac-cleanup-go/pkg/types"
)

//go:embed targets.yaml
var embeddedConfig []byte

// LoadEmbedded loads the embedded config
func LoadEmbedded() (*types.Config, error) {
	var cfg types.Config
	if err := yaml.Unmarshal(embeddedConfig, &cfg); err != nil {
		return nil, err
	}
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// validBuiltinIDs defines the known builtin scanner IDs
var validBuiltinIDs = map[string]bool{
	"docker":   true,
	"homebrew": true,
}

// validateConfig validates the configuration for correctness
func validateConfig(cfg *types.Config) error {
	validMethods := map[types.CleanupMethod]bool{
		types.MethodTrash:     true,
		types.MethodPermanent: true,
		types.MethodCommand:   true,
		types.MethodBuiltin:   true,
		types.MethodManual:    true,
	}
	validSafety := map[types.SafetyLevel]bool{
		types.SafetyLevelSafe:     true,
		types.SafetyLevelModerate: true,
		types.SafetyLevelRisky:    true,
	}

	for _, cat := range cfg.Categories {
		// Validate method
		if !validMethods[cat.Method] {
			return fmt.Errorf("category '%s': invalid method '%s'", cat.ID, cat.Method)
		}

		// Validate safety
		if !validSafety[cat.Safety] {
			return fmt.Errorf("category '%s': invalid safety '%s'", cat.ID, cat.Safety)
		}

		// Method-specific validations
		switch cat.Method {
		case types.MethodCommand:
			if cat.Command == "" {
				return fmt.Errorf("category '%s': command field required for method 'command'", cat.ID)
			}
		case types.MethodBuiltin:
			if !validBuiltinIDs[cat.ID] {
				return fmt.Errorf("category '%s': unknown builtin ID, expected one of: docker, homebrew", cat.ID)
			}
		}
	}

	return nil
}
