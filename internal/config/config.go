package config

import (
	_ "embed"

	"gopkg.in/yaml.v3"

	"mac-cleanup-go/pkg/types"
)

//go:embed targets.yaml
var embeddedConfig []byte

// LoadEmbedded loads the embedded config
func LoadEmbedded() (*types.Config, error) {
	var cfg types.Config
	if err := yaml.Unmarshal(embeddedConfig, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
