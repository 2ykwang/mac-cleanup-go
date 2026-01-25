package target

import (
	"fmt"
	"os"
	"strings"

	"github.com/2ykwang/mac-cleanup-go/internal/logger"
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
		logger.Warn("brew --cache failed", "error", err)
		return ""
	}

	s.cachePath = strings.TrimSpace(string(output))
	logger.Debug("brew cache path resolved", "path", s.cachePath)
	return s.cachePath
}

func (s *BrewTarget) Scan() (*types.ScanResult, error) {
	result := types.NewScanResult(s.category)

	if !s.IsAvailable() {
		logger.Debug("brew not available, skipping scan")
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

	logger.Info("brew scan completed",
		"cachePath", cachePath,
		"totalSize", result.TotalSize,
		"fileCount", result.TotalFileCount)

	return result, nil
}

func (s *BrewTarget) Clean(items []types.CleanableItem) (*types.CleanResult, error) {
	result := types.NewCleanResult(s.category)

	if len(items) == 0 {
		return result, nil
	}

	// Run brew cleanup first
	cmd := execCommand("brew", "cleanup", "--prune=all", "-s")
	if err := cmd.Run(); err != nil {
		logger.Warn("brew cleanup command failed", "error", err)
	} else {
		logger.Debug("brew cleanup command completed")
	}

	cachePath := s.getBrewCachePath()

	batchResult := utils.BatchTrash(items, types.BatchTrashOptions{
		Category: result.Category,
		Validate: func(item types.CleanableItem) error {
			if cachePath == "" || !strings.HasPrefix(item.Path, cachePath) {
				return fmt.Errorf("invalid path: %s", item.Path)
			}
			return nil
		},
	})

	result.CleanedItems += batchResult.CleanedItems
	result.FreedSpace += batchResult.FreedSpace
	result.Errors = append(result.Errors, batchResult.Errors...)

	logger.Info("brew clean completed",
		"cleanedItems", result.CleanedItems,
		"freedSpace", result.FreedSpace,
		"errors", len(result.Errors))

	return result, nil
}
