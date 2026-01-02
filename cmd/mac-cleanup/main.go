package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"mac-cleanup-go/internal/cli"
	"mac-cleanup-go/internal/config"
	"mac-cleanup-go/internal/tui"
	"mac-cleanup-go/internal/userconfig"
	"mac-cleanup-go/pkg/types"
)

// version is set via ldflags: go build -ldflags "-X main.version=1.0.0"
var version = "dev"

func main() {
	// Flags
	showVersion := flag.Bool("version", false, "Show version")
	shortVersion := flag.Bool("v", false, "Show version (short)")
	dangerouslyDelete := flag.Bool("dangerously-delete", false, "Enable permanent deletion (default: move to Trash)")
	cleanMode := flag.Bool("clean", false, "Run cleanup with saved profile (no TUI)")
	dryRun := flag.Bool("dry-run", false, "Show what would be cleaned without actually cleaning")
	flag.Parse()

	if *showVersion || *shortVersion {
		fmt.Printf("mac-cleanup %s\n", version)
		return
	}

	cfg, err := config.LoadEmbedded()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Load user config and merge
	userCfg, err := userconfig.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load user config: %v\n", err)
	}
	if userCfg != nil {
		cfg = mergeConfig(cfg, userCfg)
	}

	// CLI clean mode
	if *cleanMode {
		if err := cli.Run(cfg, *dangerouslyDelete, *dryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// TUI mode
	p := tea.NewProgram(
		tui.NewModel(cfg, *dangerouslyDelete),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// mergeConfig merges user config into base config
func mergeConfig(cfg *types.Config, userCfg *userconfig.UserConfig) *types.Config {
	// 1. Apply TargetOverrides (user takes precedence for ID conflicts)
	cfg.Categories = applyOverrides(cfg.Categories, userCfg.TargetOverrides)

	// 2. Add CustomTargets (with validation, skip invalid ones with warning)
	for _, ct := range userCfg.CustomTargets {
		cat, err := convertCustomTarget(ct)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping custom target '%s': %v\n", ct.ID, err)
			continue
		}

		// Check for ID conflict - user takes precedence
		found := false
		for i, existing := range cfg.Categories {
			if existing.ID == cat.ID {
				cfg.Categories[i] = cat
				found = true
				break
			}
		}
		if !found {
			cfg.Categories = append(cfg.Categories, cat)
		}
	}

	return cfg
}

// applyOverrides applies user overrides to categories
func applyOverrides(categories []types.Category, overrides map[string]userconfig.CategoryOverride) []types.Category {
	if len(overrides) == 0 {
		return categories
	}

	result := make([]types.Category, 0, len(categories))
	for _, cat := range categories {
		override, hasOverride := overrides[cat.ID]
		if !hasOverride {
			result = append(result, cat)
			continue
		}

		// Disabled: skip this category
		if override.Disabled != nil && *override.Disabled {
			continue
		}

		// Add extra paths
		if len(override.Paths) > 0 {
			cat.Paths = append(cat.Paths, override.Paths...)
		}

		// Override note
		if override.Note != nil {
			cat.Note = *override.Note
		}

		result = append(result, cat)
	}

	return result
}

// convertCustomTarget converts userconfig.CustomTarget to types.Category with validation
func convertCustomTarget(ct userconfig.CustomTarget) (types.Category, error) {
	// Validate required fields
	if ct.ID == "" {
		return types.Category{}, fmt.Errorf("id is required")
	}
	if ct.Name == "" {
		return types.Category{}, fmt.Errorf("name is required")
	}

	// Validate and convert safety level
	var safety types.SafetyLevel
	switch ct.Safety {
	case "safe":
		safety = types.SafetyLevelSafe
	case "moderate":
		safety = types.SafetyLevelModerate
	case "risky":
		safety = types.SafetyLevelRisky
	case "":
		safety = types.SafetyLevelSafe // default
	default:
		return types.Category{}, fmt.Errorf("invalid safety level '%s' (use: safe, moderate, risky)", ct.Safety)
	}

	// Validate and convert method
	var method types.CleanupMethod
	switch ct.Method {
	case "trash":
		method = types.MethodTrash
	case "permanent":
		method = types.MethodPermanent
	case "command":
		method = types.MethodCommand
	case "special":
		method = types.MethodSpecial
	case "manual":
		method = types.MethodManual
	case "":
		method = types.MethodTrash // default
	default:
		return types.Category{}, fmt.Errorf("invalid method '%s' (use: trash, permanent, command, special, manual)", ct.Method)
	}

	// Default group
	group := ct.Group
	if group == "" {
		group = "app"
	}

	return types.Category{
		ID:       ct.ID,
		Name:     ct.Name,
		Group:    group,
		Safety:   safety,
		Method:   method,
		Note:     ct.Note,
		Paths:    ct.Paths,
		Command:  ct.Command,
		CheckCmd: ct.CheckCmd,
	}, nil
}
