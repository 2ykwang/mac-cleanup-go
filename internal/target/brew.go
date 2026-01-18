package target

import (
	"os"
	"strings"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

type BrewTarget struct {
	category  types.Category
	cachePath string
}

func init() {
	RegisterBuiltin("homebrew", func(cat types.Category, _ []types.Category) Target {
		return NewBrewTarget(cat)
	})
}

func NewBrewTarget(cat types.Category) *BrewTarget {
	return &BrewTarget{category: cat}
}

func (s *BrewTarget) Category() types.Category {
	return s.category
}

func (s *BrewTarget) IsAvailable() bool {
	return utils.CommandExists("brew")
}

// getBrewCachePath returns the brew cache directory path.
func (s *BrewTarget) getBrewCachePath() string {
	if s.cachePath != "" {
		return s.cachePath
	}

	cmd := execCommand("brew", "--cache")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	s.cachePath = strings.TrimSpace(string(output))
	return s.cachePath
}

func (s *BrewTarget) Scan() (*types.ScanResult, error) {
	result := types.NewScanResult(s.category)

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

func (s *BrewTarget) Clean(items []types.CleanableItem) (*types.CleanResult, error) {
	result := types.NewCleanResult(s.category)

	if len(items) == 0 {
		return result, nil
	}

	cmd := execCommand("brew", "cleanup", "--prune=all", "-s")
	_ = cmd.Run()

	// Collect valid paths and build path-to-item map
	paths := make([]string, 0, len(items))
	pathToItem := make(map[string]types.CleanableItem, len(items))
	cachePath := s.getBrewCachePath()

	for _, item := range items {
		// Verify the path is within brew cache (safety check)
		if cachePath == "" || !strings.HasPrefix(item.Path, cachePath) {
			result.Errors = append(result.Errors, "invalid path: "+item.Path)
			continue
		}
		paths = append(paths, item.Path)
		pathToItem[item.Path] = item
	}

	if len(paths) == 0 {
		return result, nil
	}

	// Batch delete
	batchResult := utils.MoveToTrashBatch(paths)

	// Process succeeded items
	for _, p := range batchResult.Succeeded {
		item := pathToItem[p]
		result.FreedSpace += item.Size
		result.CleanedItems++
	}

	// Process failed items
	for p, err := range batchResult.Failed {
		result.Errors = append(result.Errors, p+": "+err.Error())
	}

	return result, nil
}
