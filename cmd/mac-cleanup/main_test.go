package main

import (
	"testing"

	"mac-cleanup-go/internal/userconfig"
	"mac-cleanup-go/pkg/types"
)

func TestMergeConfig_NilUserConfig(t *testing.T) {
	cfg := &types.Config{
		Categories: []types.Category{
			{ID: "test", Name: "Test"},
		},
	}

	result := mergeConfig(cfg, &userconfig.UserConfig{})
	if len(result.Categories) != 1 {
		t.Errorf("Expected 1 category, got %d", len(result.Categories))
	}
}

func TestMergeConfig_CustomTargets(t *testing.T) {
	cfg := &types.Config{
		Categories: []types.Category{
			{ID: "existing", Name: "Existing"},
		},
	}

	userCfg := &userconfig.UserConfig{
		CustomTargets: []userconfig.CustomTarget{
			{
				ID:     "custom-test",
				Name:   "Custom Test",
				Group:  "dev",
				Safety: "safe",
				Method: "trash",
				Paths:  []string{"~/test/*"},
			},
		},
	}

	result := mergeConfig(cfg, userCfg)
	if len(result.Categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(result.Categories))
	}

	// Verify custom target was added
	found := false
	for _, cat := range result.Categories {
		if cat.ID == "custom-test" {
			found = true
			if cat.Name != "Custom Test" {
				t.Errorf("Expected name 'Custom Test', got '%s'", cat.Name)
			}
			if cat.Safety != types.SafetyLevelSafe {
				t.Errorf("Expected safety 'safe', got '%s'", cat.Safety)
			}
			break
		}
	}
	if !found {
		t.Error("Custom target 'custom-test' not found")
	}
}

func TestMergeConfig_CustomTargetOverridesExisting(t *testing.T) {
	cfg := &types.Config{
		Categories: []types.Category{
			{ID: "test", Name: "Original", Safety: types.SafetyLevelSafe},
		},
	}

	userCfg := &userconfig.UserConfig{
		CustomTargets: []userconfig.CustomTarget{
			{
				ID:     "test", // Same ID as existing
				Name:   "User Override",
				Safety: "moderate",
				Method: "trash",
			},
		},
	}

	result := mergeConfig(cfg, userCfg)
	if len(result.Categories) != 1 {
		t.Errorf("Expected 1 category, got %d", len(result.Categories))
	}

	if result.Categories[0].Name != "User Override" {
		t.Errorf("Expected user override to take precedence, got '%s'", result.Categories[0].Name)
	}
}

func TestMergeConfig_TargetOverrides_Disabled(t *testing.T) {
	disabled := true
	cfg := &types.Config{
		Categories: []types.Category{
			{ID: "trash", Name: "Trash"},
			{ID: "cache", Name: "Cache"},
		},
	}

	userCfg := &userconfig.UserConfig{
		TargetOverrides: map[string]userconfig.CategoryOverride{
			"trash": {Disabled: &disabled},
		},
	}

	result := mergeConfig(cfg, userCfg)
	if len(result.Categories) != 1 {
		t.Errorf("Expected 1 category (trash disabled), got %d", len(result.Categories))
	}

	if result.Categories[0].ID != "cache" {
		t.Errorf("Expected 'cache' to remain, got '%s'", result.Categories[0].ID)
	}
}

func TestMergeConfig_TargetOverrides_AddPaths(t *testing.T) {
	cfg := &types.Config{
		Categories: []types.Category{
			{ID: "npm", Name: "npm", Paths: []string{"~/.npm/*"}},
		},
	}

	userCfg := &userconfig.UserConfig{
		TargetOverrides: map[string]userconfig.CategoryOverride{
			"npm": {Paths: []string{"~/.npm/_npx/*"}},
		},
	}

	result := mergeConfig(cfg, userCfg)
	if len(result.Categories[0].Paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(result.Categories[0].Paths))
	}
}

func TestMergeConfig_TargetOverrides_Note(t *testing.T) {
	newNote := "Updated note"
	cfg := &types.Config{
		Categories: []types.Category{
			{ID: "test", Name: "Test", Note: "Original note"},
		},
	}

	userCfg := &userconfig.UserConfig{
		TargetOverrides: map[string]userconfig.CategoryOverride{
			"test": {Note: &newNote},
		},
	}

	result := mergeConfig(cfg, userCfg)
	if result.Categories[0].Note != "Updated note" {
		t.Errorf("Expected 'Updated note', got '%s'", result.Categories[0].Note)
	}
}

func TestConvertCustomTarget_Valid(t *testing.T) {
	ct := userconfig.CustomTarget{
		ID:     "test",
		Name:   "Test",
		Group:  "dev",
		Safety: "moderate",
		Method: "permanent",
		Paths:  []string{"~/test/*"},
	}

	cat, err := convertCustomTarget(ct)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cat.ID != "test" {
		t.Errorf("Expected ID 'test', got '%s'", cat.ID)
	}
	if cat.Safety != types.SafetyLevelModerate {
		t.Errorf("Expected safety 'moderate', got '%s'", cat.Safety)
	}
	if cat.Method != types.MethodPermanent {
		t.Errorf("Expected method 'permanent', got '%s'", cat.Method)
	}
}

func TestConvertCustomTarget_Defaults(t *testing.T) {
	ct := userconfig.CustomTarget{
		ID:   "test",
		Name: "Test",
		// Safety, Method, Group are empty - should use defaults
	}

	cat, err := convertCustomTarget(ct)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cat.Safety != types.SafetyLevelSafe {
		t.Errorf("Expected default safety 'safe', got '%s'", cat.Safety)
	}
	if cat.Method != types.MethodTrash {
		t.Errorf("Expected default method 'trash', got '%s'", cat.Method)
	}
	if cat.Group != "app" {
		t.Errorf("Expected default group 'app', got '%s'", cat.Group)
	}
}

func TestConvertCustomTarget_InvalidSafety(t *testing.T) {
	ct := userconfig.CustomTarget{
		ID:     "test",
		Name:   "Test",
		Safety: "invalid",
	}

	_, err := convertCustomTarget(ct)
	if err == nil {
		t.Error("Expected error for invalid safety level")
	}
}

func TestConvertCustomTarget_InvalidMethod(t *testing.T) {
	ct := userconfig.CustomTarget{
		ID:     "test",
		Name:   "Test",
		Method: "invalid",
	}

	_, err := convertCustomTarget(ct)
	if err == nil {
		t.Error("Expected error for invalid method")
	}
}

func TestConvertCustomTarget_MissingID(t *testing.T) {
	ct := userconfig.CustomTarget{
		Name: "Test",
	}

	_, err := convertCustomTarget(ct)
	if err == nil {
		t.Error("Expected error for missing ID")
	}
}

func TestConvertCustomTarget_MissingName(t *testing.T) {
	ct := userconfig.CustomTarget{
		ID: "test",
	}

	_, err := convertCustomTarget(ct)
	if err == nil {
		t.Error("Expected error for missing Name")
	}
}
