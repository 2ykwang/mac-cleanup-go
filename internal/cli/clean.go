package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"mac-cleanup-go/internal/cleaner"
	"mac-cleanup-go/internal/scanner"
	"mac-cleanup-go/internal/userconfig"
	"mac-cleanup-go/internal/utils"
	"mac-cleanup-go/pkg/types"
)

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	dangerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	sizeStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
)

// filterItems returns items that are not in the excluded map
func filterItems(items []types.CleanableItem, excluded map[string]bool) []types.CleanableItem {
	if excluded == nil {
		return items
	}
	var filtered []types.CleanableItem
	for _, item := range items {
		if !excluded[item.Path] {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// buildExcludedMap converts userconfig excluded paths to a lookup map
func buildExcludedMap(userCfg *userconfig.UserConfig) map[string]map[string]bool {
	excluded := make(map[string]map[string]bool)
	for catID, paths := range userCfg.ExcludedPaths {
		excluded[catID] = make(map[string]bool)
		for _, path := range paths {
			excluded[catID][path] = true
		}
	}
	return excluded
}

// filterSafeCategories separates safe categories from moderate/risky ones
func filterSafeCategories(selectedIDs []string, categoryMap map[string]types.Category) (safeIDs, skippedNames []string) {
	for _, id := range selectedIDs {
		if cat, ok := categoryMap[id]; ok {
			if cat.Safety == types.SafetyLevelSafe {
				safeIDs = append(safeIDs, id)
			} else {
				skippedNames = append(skippedNames, cat.Name)
			}
		}
	}
	return
}

// scanCategories scans the given category IDs and returns results with size > 0 and any warnings
func scanCategories(ids []string, registry *scanner.Registry) ([]*types.ScanResult, []string) {
	var results []*types.ScanResult
	var warnings []string
	for _, id := range ids {
		if s, ok := registry.Get(id); ok {
			if s.IsAvailable() {
				result, _ := s.Scan()
				if result != nil {
					// Collect scan errors as warnings
					if result.Error != nil {
						warnings = append(warnings,
							fmt.Sprintf("%s: %s", result.Category.Name, result.Error.Error()))
					}
					if result.TotalSize > 0 {
						results = append(results, result)
					}
				}
			}
		}
	}
	return results, warnings
}

// showPreview displays the cleanup preview and returns totals
func showPreview(results []*types.ScanResult, excluded map[string]map[string]bool) (totalSize int64, totalItems int) {
	fmt.Println(titleStyle.Render("Preview:"))
	fmt.Println(strings.Repeat("─", 50))

	for _, result := range results {
		items := filterItems(result.Items, excluded[result.Category.ID])
		if len(items) == 0 {
			continue
		}

		var catSize int64
		for _, item := range items {
			catSize += item.Size
		}

		totalSize += catSize
		totalItems += len(items)

		safetyDot := successStyle.Render("●") // CLI only has safe categories
		size := fmt.Sprintf("%10s", utils.FormatSize(catSize))
		fmt.Printf("%s %-30s %s\n", safetyDot, result.Category.Name, sizeStyle.Render(size))
	}

	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("Total: %s (%d items)\n\n", sizeStyle.Render(utils.FormatSize(totalSize)), totalItems)
	return
}

// confirmCleanup asks for user confirmation
func confirmCleanup(dangerouslyDelete bool) bool {
	deleteMethod := "Trash"
	if dangerouslyDelete {
		deleteMethod = dangerStyle.Render("PERMANENT DELETE")
	}
	fmt.Printf("Delete method: %s\n", deleteMethod)
	fmt.Printf("Proceed? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	return input == "y" || input == "yes"
}

// executeCleanup performs the actual cleanup and returns report
func executeCleanup(
	results []*types.ScanResult,
	excluded map[string]map[string]bool,
	registry *scanner.Registry,
	dangerouslyDelete bool,
) types.Report {
	c := cleaner.New()
	var report types.Report
	report.Results = make([]types.CleanResult, 0)

	for _, result := range results {
		if result.Category.Method == types.MethodManual {
			continue
		}

		items := filterItems(result.Items, excluded[result.Category.ID])
		if len(items) == 0 {
			continue
		}

		cat := result.Category
		if cat.Method == types.MethodPermanent && !dangerouslyDelete {
			cat.Method = types.MethodTrash
		}

		var cleanResult *types.CleanResult
		if cat.Method == types.MethodSpecial {
			if s, ok := registry.Get(cat.ID); ok {
				cleanResult, _ = s.Clean(items, false)
			}
		} else {
			cleanResult = c.Clean(cat, items, false)
		}

		if cleanResult != nil {
			report.Results = append(report.Results, *cleanResult)
			report.FreedSpace += cleanResult.FreedSpace
			report.CleanedItems += cleanResult.CleanedItems
			report.FailedItems += len(cleanResult.Errors)

			printCleanResult(cat.Name, cleanResult)
		}
	}
	return report
}

// printCleanResult prints the result of cleaning a single category
func printCleanResult(name string, result *types.CleanResult) {
	if result.CleanedItems == 0 && len(result.Errors) == 0 {
		return
	}

	size := fmt.Sprintf("%10s", utils.FormatSize(result.FreedSpace))
	switch {
	case len(result.Errors) == 0:
		fmt.Printf("%s %-30s %s\n", successStyle.Render("✓"), name, sizeStyle.Render(size))
	case result.CleanedItems > 0:
		fmt.Printf("%s %-30s %s\n", warningStyle.Render("△"), name, sizeStyle.Render(size))
	default:
		fmt.Printf("%s %-30s %s\n", dangerStyle.Render("✗"), name, mutedStyle.Render("failed"))
	}
}

// showReport displays the final cleanup report
func showReport(report types.Report, duration time.Duration) {
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("Freed: %s\n", sizeStyle.Render(utils.FormatSize(report.FreedSpace)))
	fmt.Printf("Succeeded: %s  Failed: %s  Time: %s\n",
		successStyle.Render(fmt.Sprintf("%d", report.CleanedItems)),
		dangerStyle.Render(fmt.Sprintf("%d", report.FailedItems)),
		mutedStyle.Render(duration.Round(time.Millisecond).String()))
}

// Run executes the CLI clean mode
func Run(cfg *types.Config, dangerouslyDelete, dryRun bool) error {
	// Load user config
	userCfg, err := userconfig.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !userCfg.HasLastSelection() {
		return errors.New("no saved profile found. Run mac-cleanup in TUI mode first to create a profile")
	}

	fmt.Printf("%s\n\n", titleStyle.Render("mac-cleanup --clean"))

	if dangerouslyDelete {
		fmt.Printf("%s\n\n", dangerStyle.Render("⚠ WARNING: Permanent deletion mode enabled!"))
	}

	// Build category map
	categoryMap := make(map[string]types.Category)
	for _, cat := range cfg.Categories {
		categoryMap[cat.ID] = cat
	}

	// Filter safe categories only
	safeIDs, skippedNames := filterSafeCategories(userCfg.GetLastSelection(), categoryMap)

	if len(skippedNames) > 0 {
		fmt.Printf("%s\n", mutedStyle.Render("Skipped (CLI only supports safe categories):"))
		for _, name := range skippedNames {
			fmt.Printf("  %s %s\n", mutedStyle.Render("⊘"), mutedStyle.Render(name))
		}
		fmt.Println()
	}

	if len(safeIDs) == 0 {
		fmt.Println(mutedStyle.Render("No safe categories to clean. Use TUI mode for moderate/risky categories."))
		return nil
	}

	// Scan
	fmt.Printf("%s", mutedStyle.Render("Scanning..."))
	registry := scanner.DefaultRegistry(cfg)
	results, warnings := scanCategories(safeIDs, registry)
	fmt.Printf("\r%s\n\n", strings.Repeat(" ", 20))

	// Show scan warnings
	if len(warnings) > 0 {
		fmt.Println(warningStyle.Render("Scan warnings:"))
		for _, w := range warnings {
			fmt.Printf("  %s\n", mutedStyle.Render(w))
		}
		fmt.Println()
	}

	if len(results) == 0 {
		fmt.Println(mutedStyle.Render("Nothing to clean."))
		return nil
	}

	// Build excluded paths map
	excluded := buildExcludedMap(userCfg)

	// Show preview
	totalSize, _ := showPreview(results, excluded)
	if totalSize == 0 {
		fmt.Println(mutedStyle.Render("Nothing to clean."))
		return nil
	}

	// Dry run - stop here
	if dryRun {
		fmt.Println(mutedStyle.Render("Dry run - no files were deleted."))
		return nil
	}

	// Confirm
	if !confirmCleanup(dangerouslyDelete) {
		fmt.Println(mutedStyle.Render("Cancelled."))
		return nil
	}

	// Execute cleanup
	fmt.Printf("\n%s\n\n", titleStyle.Render("Cleaning..."))
	startTime := time.Now()
	report := executeCleanup(results, excluded, registry, dangerouslyDelete)
	duration := time.Since(startTime)

	// Show report
	showReport(report, duration)

	return nil
}
