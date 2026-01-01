package cli

import (
	"testing"

	"mac-cleanup-go/internal/userconfig"
	"mac-cleanup-go/pkg/types"
)

func TestFilterItems(t *testing.T) {
	items := []types.CleanableItem{
		{Path: "/path/a", Size: 100},
		{Path: "/path/b", Size: 200},
		{Path: "/path/c", Size: 300},
	}

	t.Run("nil excluded map returns all items", func(t *testing.T) {
		result := filterItems(items, nil)
		if len(result) != 3 {
			t.Errorf("expected 3 items, got %d", len(result))
		}
	})

	t.Run("empty excluded map returns all items", func(t *testing.T) {
		result := filterItems(items, map[string]bool{})
		if len(result) != 3 {
			t.Errorf("expected 3 items, got %d", len(result))
		}
	})

	t.Run("excludes matching paths", func(t *testing.T) {
		excluded := map[string]bool{
			"/path/b": true,
		}
		result := filterItems(items, excluded)
		if len(result) != 2 {
			t.Errorf("expected 2 items, got %d", len(result))
		}
		for _, item := range result {
			if item.Path == "/path/b" {
				t.Error("excluded path should not be in result")
			}
		}
	})

	t.Run("excludes multiple paths", func(t *testing.T) {
		excluded := map[string]bool{
			"/path/a": true,
			"/path/c": true,
		}
		result := filterItems(items, excluded)
		if len(result) != 1 {
			t.Errorf("expected 1 item, got %d", len(result))
		}
		if result[0].Path != "/path/b" {
			t.Errorf("expected /path/b, got %s", result[0].Path)
		}
	})

	t.Run("all excluded returns empty", func(t *testing.T) {
		excluded := map[string]bool{
			"/path/a": true,
			"/path/b": true,
			"/path/c": true,
		}
		result := filterItems(items, excluded)
		if len(result) != 0 {
			t.Errorf("expected 0 items, got %d", len(result))
		}
	})

	t.Run("empty items returns empty", func(t *testing.T) {
		result := filterItems([]types.CleanableItem{}, nil)
		if len(result) != 0 {
			t.Errorf("expected 0 items, got %d", len(result))
		}
	})
}

func TestFilterSafeCategories(t *testing.T) {
	categoryMap := map[string]types.Category{
		"safe-cat": {
			ID:     "safe-cat",
			Name:   "Safe Category",
			Safety: types.SafetyLevelSafe,
		},
		"moderate-cat": {
			ID:     "moderate-cat",
			Name:   "Moderate Category",
			Safety: types.SafetyLevelModerate,
		},
		"risky-cat": {
			ID:     "risky-cat",
			Name:   "Risky Category",
			Safety: types.SafetyLevelRisky,
		},
	}

	t.Run("filters safe categories only", func(t *testing.T) {
		selectedIDs := []string{"safe-cat", "moderate-cat", "risky-cat"}
		safeIDs, skippedNames := filterSafeCategories(selectedIDs, categoryMap)

		if len(safeIDs) != 1 {
			t.Errorf("expected 1 safe ID, got %d", len(safeIDs))
		}
		if safeIDs[0] != "safe-cat" {
			t.Errorf("expected safe-cat, got %s", safeIDs[0])
		}

		if len(skippedNames) != 2 {
			t.Errorf("expected 2 skipped names, got %d", len(skippedNames))
		}
	})

	t.Run("all safe categories", func(t *testing.T) {
		selectedIDs := []string{"safe-cat"}
		safeIDs, skippedNames := filterSafeCategories(selectedIDs, categoryMap)

		if len(safeIDs) != 1 {
			t.Errorf("expected 1 safe ID, got %d", len(safeIDs))
		}
		if len(skippedNames) != 0 {
			t.Errorf("expected 0 skipped names, got %d", len(skippedNames))
		}
	})

	t.Run("no safe categories", func(t *testing.T) {
		selectedIDs := []string{"moderate-cat", "risky-cat"}
		safeIDs, skippedNames := filterSafeCategories(selectedIDs, categoryMap)

		if len(safeIDs) != 0 {
			t.Errorf("expected 0 safe IDs, got %d", len(safeIDs))
		}
		if len(skippedNames) != 2 {
			t.Errorf("expected 2 skipped names, got %d", len(skippedNames))
		}
	})

	t.Run("unknown category ID is ignored", func(t *testing.T) {
		selectedIDs := []string{"unknown-cat", "safe-cat"}
		safeIDs, skippedNames := filterSafeCategories(selectedIDs, categoryMap)

		if len(safeIDs) != 1 {
			t.Errorf("expected 1 safe ID, got %d", len(safeIDs))
		}
		if len(skippedNames) != 0 {
			t.Errorf("expected 0 skipped (unknown should be ignored), got %d", len(skippedNames))
		}
	})

	t.Run("empty selection", func(t *testing.T) {
		safeIDs, skippedNames := filterSafeCategories([]string{}, categoryMap)

		if len(safeIDs) != 0 {
			t.Errorf("expected 0 safe IDs, got %d", len(safeIDs))
		}
		if len(skippedNames) != 0 {
			t.Errorf("expected 0 skipped names, got %d", len(skippedNames))
		}
	})
}

func TestBuildExcludedMap(t *testing.T) {
	t.Run("builds map correctly", func(t *testing.T) {
		userCfg := &userconfig.UserConfig{
			ExcludedPaths: map[string][]string{
				"cat1": {"/path/a", "/path/b"},
				"cat2": {"/path/c"},
			},
		}

		excluded := buildExcludedMap(userCfg)

		if len(excluded) != 2 {
			t.Errorf("expected 2 categories, got %d", len(excluded))
		}

		if !excluded["cat1"]["/path/a"] {
			t.Error("/path/a should be excluded in cat1")
		}
		if !excluded["cat1"]["/path/b"] {
			t.Error("/path/b should be excluded in cat1")
		}
		if !excluded["cat2"]["/path/c"] {
			t.Error("/path/c should be excluded in cat2")
		}
	})

	t.Run("empty excluded paths", func(t *testing.T) {
		userCfg := &userconfig.UserConfig{
			ExcludedPaths: map[string][]string{},
		}

		excluded := buildExcludedMap(userCfg)

		if len(excluded) != 0 {
			t.Errorf("expected 0 categories, got %d", len(excluded))
		}
	})

	t.Run("nil excluded paths", func(t *testing.T) {
		userCfg := &userconfig.UserConfig{
			ExcludedPaths: nil,
		}

		excluded := buildExcludedMap(userCfg)

		if excluded == nil {
			t.Error("should return empty map, not nil")
		}
		if len(excluded) != 0 {
			t.Errorf("expected 0 categories, got %d", len(excluded))
		}
	})
}
