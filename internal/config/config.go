package config

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/2ykwang/mac-cleanup-go/internal/logger"
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
		if err := validateDisposal(cat); err != nil {
			return err
		}
		// Validate method
		if !validMethods[cat.Method] {
			logger.Warn("config validation failed: invalid method",
				"category", cat.ID, "method", cat.Method)
			return fmt.Errorf("category '%s': invalid method '%s'", cat.ID, cat.Method)
		}

		// Validate safety
		if !validSafety[cat.Safety] {
			logger.Warn("config validation failed: invalid safety",
				"category", cat.ID, "safety", cat.Safety)
			return fmt.Errorf("category '%s': invalid safety '%s'", cat.ID, cat.Safety)
		}

		// Method-specific validations
		if cat.Method == types.MethodBuiltin && !target.IsBuiltinID(cat.ID) {
			logger.Warn("config validation failed: unknown builtin ID",
				"category", cat.ID)
			return fmt.Errorf("category '%s': unknown builtin ID", cat.ID)
		}
	}

	logger.Info("config validated", "categories", len(cfg.Categories))
	return nil
}

func validateDisposal(cat types.Category) error {
	if cat.Disposal == "" {
		return nil
	}
	switch cat.Disposal {
	case types.DisposalTrash, types.DisposalPermanent, types.DisposalMixed, types.DisposalManual, types.DisposalUnknown:
	default:
		return fmt.Errorf("category '%s': invalid disposal '%s'", cat.ID, cat.Disposal)
	}
	if cat.Method == types.MethodBuiltin {
		if cat.Disposal == types.DisposalManual {
			return fmt.Errorf("category '%s': builtin disposal cannot be manual", cat.ID)
		}
		return nil
	}
	if cat.Disposal != cat.DisposalMode() {
		return fmt.Errorf("category '%s': disposal '%s' conflicts with method '%s'", cat.ID, cat.Disposal, cat.Method)
	}
	return nil
}
