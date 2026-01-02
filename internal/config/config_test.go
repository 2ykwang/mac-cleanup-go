package config

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"mac-cleanup-go/pkg/types"
)

func TestLoadEmbedded_ReturnsNonNil(t *testing.T) {
	cfg, err := LoadEmbedded()

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestLoadEmbedded_HasCategories(t *testing.T) {
	cfg, err := LoadEmbedded()

	assert.NoError(t, err)
	assert.NotEmpty(t, cfg.Categories)
}

func TestLoadEmbedded_KnownCategoriesExist(t *testing.T) {
	cfg, _ := LoadEmbedded()

	categoryMap := make(map[string]bool)
	for _, cat := range cfg.Categories {
		categoryMap[cat.ID] = true
	}

	assert.True(t, categoryMap["system-cache"])
	assert.True(t, categoryMap["docker"])
	assert.True(t, categoryMap["homebrew"])
}

func TestLoadEmbedded_CategoriesHaveRequiredFields(t *testing.T) {
	cfg, _ := LoadEmbedded()

	for _, cat := range cfg.Categories {
		assert.NotEmpty(t, cat.ID)
		assert.NotEmpty(t, cat.Name, "Category '%s' has empty Name", cat.ID)
		assert.NotEmpty(t, cat.Safety, "Category '%s' has empty Safety", cat.ID)
		assert.NotEmpty(t, cat.Method, "Category '%s' has empty Method", cat.ID)
	}
}

func TestLoadEmbedded_SafetyLevelsAreValid(t *testing.T) {
	cfg, _ := LoadEmbedded()

	validSafety := map[types.SafetyLevel]bool{
		types.SafetyLevelSafe:     true,
		types.SafetyLevelModerate: true,
		types.SafetyLevelRisky:    true,
	}

	for _, cat := range cfg.Categories {
		assert.True(t, validSafety[cat.Safety], "Category '%s' has invalid safety: %s", cat.ID, cat.Safety)
	}
}

func TestLoadEmbedded_MethodsAreValid(t *testing.T) {
	cfg, _ := LoadEmbedded()

	validMethods := map[types.CleanupMethod]bool{
		types.MethodTrash:   true,
		types.MethodCommand: true,
		types.MethodSpecial: true,
		types.MethodManual:  true,
	}

	for _, cat := range cfg.Categories {
		assert.True(t, validMethods[cat.Method], "Category '%s' has invalid method: %s", cat.ID, cat.Method)
	}
}
