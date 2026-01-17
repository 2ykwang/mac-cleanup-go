package target

import (
	"os"
	"os/exec"
	"strings"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

type BrewScanner struct {
	category  types.Category
	cachePath string
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

// getBrewCachePath returns the brew cache directory path.
func (s *BrewScanner) getBrewCachePath() string {
	if s.cachePath != "" {
		return s.cachePath
	}

	cmd := exec.Command("brew", "--cache")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	s.cachePath = strings.TrimSpace(string(output))
	return s.cachePath
}

func (s *BrewScanner) Scan() (*types.ScanResult, error) {
	result := &types.ScanResult{
		Category: s.category,
		Items:    make([]types.CleanableItem, 0),
	}

	if !s.IsAvailable() {
		return result, nil
	}

	cachePath := s.getBrewCachePath()
	if cachePath == "" {
		return result, nil
	}

	// Verify cache path exists
	info, err := os.Stat(cachePath)
	if err != nil || !info.IsDir() {
		return result, nil
	}

	// Scan the cache directory
	size, fileCount, _ := utils.GetDirSizeWithCount(cachePath)
	if size > 0 {
		item := types.CleanableItem{
			Path:        cachePath,
			Size:        size,
			FileCount:   fileCount,
			Name:        "Homebrew Cache",
			IsDirectory: true,
			ModifiedAt:  info.ModTime(),
		}
		result.Items = append(result.Items, item)
		result.TotalSize = size
		result.TotalFileCount = fileCount
	}

	return result, nil
}

func (s *BrewScanner) Clean(items []types.CleanableItem) (*types.CleanResult, error) {
	result := &types.CleanResult{
		Category: s.category,
		Errors:   make([]string, 0),
	}

	if len(items) == 0 {
		return result, nil
	}

	cmd := exec.Command("brew", "cleanup", "--prune=all", "-s")
	_ = cmd.Run()

	// Move selected items to trash
	for _, item := range items {
		// Verify the path is within brew cache (safety check)
		cachePath := s.getBrewCachePath()
		if cachePath == "" || !strings.HasPrefix(item.Path, cachePath) {
			result.Errors = append(result.Errors, "invalid path: "+item.Path)
			continue
		}

		if err := utils.MoveToTrash(item.Path); err != nil {
			result.Errors = append(result.Errors, err.Error())
			continue
		}

		result.FreedSpace += item.Size
		result.CleanedItems++
	}

	return result, nil
}
