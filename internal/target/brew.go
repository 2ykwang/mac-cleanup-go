package target

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/2ykwang/mac-cleanup-go/internal/logger"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

type BrewTarget struct {
	category  types.Category
	cacheOnce sync.Once
	cachePath string
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
// The system-cache scan calls it concurrently with Scan.
func (s *BrewTarget) getBrewCachePath() string {
	s.cacheOnce.Do(func() {
		if !s.IsAvailable() {
			return
		}
		output, err := execCommand("brew", "--cache").Output()
		if err != nil {
			logger.Warn("brew --cache failed", "error", err)
			return
		}
		s.cachePath = strings.TrimSpace(string(output))
		logger.Debug("brew cache path resolved", "path", s.cachePath)
	})
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
	stats, err := utils.GetDirStats(cachePath, utils.WithIgnoreNames(ignoredNames...))
	if err == nil && stats.Size > 0 {
		item := types.CleanableItem{
			Path:        cachePath,
			Size:        stats.Size,
			FileCount:   stats.FileCount,
			Name:        "Homebrew Cache",
			IsDirectory: true,
			ModifiedAt:  stats.ModifiedAt,
		}
		result.Items = append(result.Items, item)
		result.TotalSize = stats.Size
		result.TotalFileCount = stats.FileCount
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

	cachePath := s.getBrewCachePath()

	batchResult := utils.BatchTrash(items, types.BatchTrashOptions{
		Category: result.Category,
		Validate: func(item types.CleanableItem) error {
			if cachePath == "" || item.Path != cachePath {
				return fmt.Errorf("invalid path: %s", item.Path)
			}
			return nil
		},
	})

	result.Merge(batchResult)

	logger.Info("brew clean completed",
		"cleanedItems", result.CleanedItems,
		"freedSpace", result.FreedSpace,
		"errors", len(result.Errors))

	return result, nil
}
