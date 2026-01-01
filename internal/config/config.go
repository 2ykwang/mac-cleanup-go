package config

import (
	"os"

	"gopkg.in/yaml.v3"
	"mac-cleanup-go/pkg/types"
)

func Load(path string) (*types.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg types.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
