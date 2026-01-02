package config

import (
	"testing"

	"mac-cleanup-go/pkg/types"
)

func TestLoadEmbedded_ReturnsNonNil(t *testing.T) {
	cfg, err := LoadEmbedded()

	if err != nil {
		t.Fatalf("LoadEmbedded() error: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadEmbedded() returned nil config")
	}
}

func TestLoadEmbedded_HasCategories(t *testing.T) {
	cfg, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() error: %v", err)
	}

	if len(cfg.Categories) == 0 {
		t.Error("Expected categories, got none")
	}
}

func TestLoadEmbedded_KnownCategoriesExist(t *testing.T) {
	cfg, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() error: %v", err)
	}

	knownIDs := []string{"system-cache", "docker", "homebrew"}
	categoryMap := make(map[string]bool)
	for _, cat := range cfg.Categories {
		categoryMap[cat.ID] = true
	}

	for _, id := range knownIDs {
		if !categoryMap[id] {
			t.Errorf("Expected category '%s' not found", id)
		}
	}
}

func TestLoadEmbedded_CategoriesHaveRequiredFields(t *testing.T) {
	cfg, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() error: %v", err)
	}

	for _, cat := range cfg.Categories {
		if cat.ID == "" {
			t.Error("Category has empty ID")
		}
		if cat.Name == "" {
			t.Errorf("Category '%s' has empty Name", cat.ID)
		}
		if cat.Safety == "" {
			t.Errorf("Category '%s' has empty Safety", cat.ID)
		}
		if cat.Method == "" {
			t.Errorf("Category '%s' has empty Method", cat.ID)
		}
	}
}

func TestLoadEmbedded_SafetyLevelsAreValid(t *testing.T) {
	cfg, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() error: %v", err)
	}

	validSafety := map[types.SafetyLevel]bool{
		types.SafetyLevelSafe:     true,
		types.SafetyLevelModerate: true,
		types.SafetyLevelRisky:    true,
	}

	for _, cat := range cfg.Categories {
		if !validSafety[cat.Safety] {
			t.Errorf("Category '%s' has invalid safety: %s", cat.ID, cat.Safety)
		}
	}
}

func TestLoadEmbedded_MethodsAreValid(t *testing.T) {
	cfg, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() error: %v", err)
	}

	validMethods := map[types.CleanupMethod]bool{
		types.MethodTrash:   true,
		types.MethodCommand: true,
		types.MethodSpecial: true,
		types.MethodManual:  true,
	}

	for _, cat := range cfg.Categories {
		if !validMethods[cat.Method] {
			t.Errorf("Category '%s' has invalid method: %s", cat.ID, cat.Method)
		}
	}
}
