package scanner

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/2ykwang/mac-cleanup-go/internal/utils"
	"github.com/2ykwang/mac-cleanup-go/pkg/types"
)

type BrewScanner struct {
	category types.Category
}

func NewBrewScanner(cat types.Category) *BrewScanner {
	return &BrewScanner{category: cat}
}

func (s *BrewScanner) Category() types.Category {
	return s.category
}

func (s *BrewScanner) IsAvailable() bool {
	return utils.CommandExists("brew")
}

func (s *BrewScanner) Scan() (*types.ScanResult, error) {
	result := &types.ScanResult{
		Category: s.category,
		Items:    make([]types.CleanableItem, 0),
	}

	if !s.IsAvailable() {
		return result, nil
	}

	// Run brew cleanup --dry-run to see what would be cleaned
	cmd := exec.Command("brew", "cleanup", "--dry-run")
	output, err := cmd.Output()
	if err != nil {
		// No cleanup needed or error
		return result, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Parse output - format varies:
	// "Would remove: /path/to/file (123.4MB)"
	// or just paths
	sizeRegex := regexp.MustCompile(`\(([0-9.]+)\s*(B|KB|MB|GB)\)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Extract path
		path := line
		if strings.HasPrefix(line, "Would remove: ") {
			path = strings.TrimPrefix(line, "Would remove: ")
		}

		// Extract size if present
		var size int64
		if matches := sizeRegex.FindStringSubmatch(line); len(matches) == 3 {
			path = strings.TrimSpace(sizeRegex.ReplaceAllString(path, ""))
			val, _ := strconv.ParseFloat(matches[1], 64)
			switch matches[2] {
			case "KB":
				size = int64(val * 1024)
			case "MB":
				size = int64(val * 1024 * 1024)
			case "GB":
				size = int64(val * 1024 * 1024 * 1024)
			default:
				size = int64(val)
			}
		}

		// Skip if no valid path
		if path == "" || !strings.HasPrefix(path, "/") {
			continue
		}

		// Get actual size if not parsed
		if size == 0 {
			size, _ = utils.GetFileSize(path)
		}

		if size > 0 {
			item := types.CleanableItem{
				Path:      path,
				Size:      size,
				FileCount: 1,
				Name:      extractBrewItemName(path),
			}
			result.Items = append(result.Items, item)
			result.TotalSize += size
			result.TotalFileCount++
		}
	}

	return result, nil
}

func (s *BrewScanner) Clean(items []types.CleanableItem) (*types.CleanResult, error) {
	result := &types.CleanResult{
		Category: s.category,
		Errors:   make([]string, 0),
	}

	// Run actual cleanup
	cmd := exec.Command("brew", "cleanup", "-s")
	if err := cmd.Run(); err != nil {
		result.Errors = append(result.Errors, err.Error())
	} else {
		for _, item := range items {
			result.FreedSpace += item.Size
			result.CleanedItems++
		}
	}

	return result, nil
}

func extractBrewItemName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "@" + parts[len(parts)-1]
	}
	return parts[len(parts)-1]
}
