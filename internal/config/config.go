package config

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/2ykwang/mac-cleanup-go/internal/target"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

//go:embed targets.yaml
var embeddedConfig []byte

// LoadEmbedded loads the embedded config
func LoadEmbedded() (*types.Config, error) {
	return loadConfig(embeddedConfig)
}

func loadConfig(data []byte) (*types.Config, error) {
	var cfg types.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// validateConfig validates the configuration for correctness
func validateConfig(cfg *types.Config) error {
	validMethods := map[types.CleanupMethod]bool{
		types.MethodTrash:     true,
		types.MethodPermanent: true,
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
		if cat.Method == types.MethodBuiltin && !target.IsBuiltinID(cat.ID) {
			return fmt.Errorf("category '%s': unknown builtin ID", cat.ID)
		}
	}

	return nil
}
