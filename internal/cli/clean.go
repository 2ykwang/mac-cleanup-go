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
	// Styles
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	dangerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	sizeStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
)

// Run executes the CLI clean mode
func Run(cfg *types.Config, dangerouslyDelete, dryRun bool) error {
	// Load user config
	userCfg, err := userconfig.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if there's a saved selection
	if !userCfg.HasLastSelection() {
		return errors.New("no saved profile found. Run mac-cleanup in TUI mode first to create a profile")
	}

	selectedIDs := userCfg.GetLastSelection()
	fmt.Printf("%s\n\n", titleStyle.Render("mac-cleanup --clean"))

	// Show warning for dangerous delete
	if dangerouslyDelete {
		fmt.Printf("%s\n\n", dangerStyle.Render("⚠ WARNING: Permanent deletion mode enabled!"))
	}

	// Initialize registry and scan
	registry := scanner.DefaultRegistry(cfg)
	fmt.Printf("%s", mutedStyle.Render("Scanning..."))

	// Scan only selected categories
	var results []*types.ScanResult
	for _, id := range selectedIDs {
		if s, ok := registry.Get(id); ok {
			if s.IsAvailable() {
				result, _ := s.Scan()
				if result != nil && result.TotalSize > 0 {
					results = append(results, result)
				}
			}
		}
	}
	fmt.Printf("\r%s\n\n", strings.Repeat(" ", 20)) // Clear "Scanning..."

	if len(results) == 0 {
		fmt.Println(mutedStyle.Render("Nothing to clean."))
		return nil
	}

	// Load excluded paths
	excluded := make(map[string]map[string]bool)
	for catID, paths := range userCfg.ExcludedPaths {
		excluded[catID] = make(map[string]bool)
		for _, path := range paths {
			excluded[catID][path] = true
		}
	}

	// Calculate totals and show preview
	var totalSize int64
	var totalItems int

	fmt.Println(titleStyle.Render("Preview:"))
	fmt.Println(strings.Repeat("─", 50))

	for _, result := range results {
		catID := result.Category.ID
		excludedMap := excluded[catID]

		var catSize int64
		var catItems int
		for _, item := range result.Items {
			if excludedMap == nil || !excludedMap[item.Path] {
				catSize += item.Size
				catItems++
			}
		}

		if catItems == 0 {
			continue
		}

		totalSize += catSize
		totalItems += catItems

		// Safety indicator
		var safetyDot string
		switch result.Category.Safety {
		case types.SafetyLevelSafe:
			safetyDot = successStyle.Render("●")
		case types.SafetyLevelModerate:
			safetyDot = warningStyle.Render("●")
		case types.SafetyLevelRisky:
			safetyDot = dangerStyle.Render("●")
		default:
			safetyDot = mutedStyle.Render("●")
		}

		size := fmt.Sprintf("%10s", utils.FormatSize(catSize))
		fmt.Printf("%s %-30s %s\n", safetyDot, result.Category.Name, sizeStyle.Render(size))
	}

	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("Total: %s (%d items)\n\n", sizeStyle.Render(utils.FormatSize(totalSize)), totalItems)

	// Dry run - just show preview
	if dryRun {
		fmt.Println(mutedStyle.Render("Dry run - no files were deleted."))
		return nil
	}

	// Ask for confirmation
	deleteMethod := "Trash"
	if dangerouslyDelete {
		deleteMethod = dangerStyle.Render("PERMANENT DELETE")
	}
	fmt.Printf("Delete method: %s\n", deleteMethod)
	fmt.Printf("Proceed? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input != "y" && input != "yes" {
		fmt.Println(mutedStyle.Render("Cancelled."))
		return nil
	}

	// Execute cleaning
	fmt.Printf("\n%s\n\n", titleStyle.Render("Cleaning..."))

	c := cleaner.New()
	startTime := time.Now()
	var report types.Report
	report.Results = make([]types.CleanResult, 0)

	for _, result := range results {
		catID := result.Category.ID
		if result.Category.Method == types.MethodManual {
			continue
		}

		var items []types.CleanableItem
		excludedMap := excluded[catID]
		for _, item := range result.Items {
			if excludedMap == nil || !excludedMap[item.Path] {
				items = append(items, item)
			}
		}

		if len(items) == 0 {
			continue
		}

		cat := result.Category
		if cat.Method == types.MethodPermanent && !dangerouslyDelete {
			cat.Method = types.MethodTrash
		}

		var cleanResult *types.CleanResult
		if cat.Method == types.MethodSpecial {
			if s, ok := registry.Get(catID); ok {
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

			// Show progress
			if cleanResult.CleanedItems > 0 || len(cleanResult.Errors) > 0 {
				if len(cleanResult.Errors) == 0 {
					size := fmt.Sprintf("%10s", utils.FormatSize(cleanResult.FreedSpace))
					fmt.Printf("%s %-30s %s\n", successStyle.Render("✓"), cat.Name, sizeStyle.Render(size))
				} else if cleanResult.CleanedItems > 0 {
					size := fmt.Sprintf("%10s", utils.FormatSize(cleanResult.FreedSpace))
					fmt.Printf("%s %-30s %s\n", warningStyle.Render("△"), cat.Name, sizeStyle.Render(size))
				} else {
					fmt.Printf("%s %-30s %s\n", dangerStyle.Render("✗"), cat.Name, mutedStyle.Render("failed"))
				}
			}
		}
	}

	duration := time.Since(startTime)

	// Show final report
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("Freed: %s\n", sizeStyle.Render(utils.FormatSize(report.FreedSpace)))
	fmt.Printf("Succeeded: %s  Failed: %s  Time: %s\n",
		successStyle.Render(fmt.Sprintf("%d", report.CleanedItems)),
		dangerStyle.Render(fmt.Sprintf("%d", report.FailedItems)),
		mutedStyle.Render(duration.Round(time.Millisecond).String()))

	return nil
}
