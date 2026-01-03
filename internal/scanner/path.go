package scanner

import (
	"os"
	"path/filepath"
	"time"

	"mac-cleanup-go/internal/utils"
	"mac-cleanup-go/pkg/types"
)

type PathScanner struct {
	category types.Category
}

func NewPathScanner(cat types.Category) *PathScanner {
	return &PathScanner{category: cat}
}

func (s *PathScanner) Category() types.Category {
	return s.category
}

func (s *PathScanner) IsAvailable() bool {
	// For command-based methods, check if command exists
	if s.category.CheckCmd != "" {
		return utils.CommandExists(s.category.CheckCmd)
	}

	// For path-based methods, check if check path OR any of the paths exist
	// This handles cases where app is uninstalled but cache folder remains
	if s.category.Check != "" && utils.PathExists(s.category.Check) {
		return true
	}

	// Check if any of the paths have matching files
	for _, pattern := range s.category.Paths {
		paths, err := utils.GlobPaths(pattern)
		if err == nil && len(paths) > 0 {
			return true
		}
	}

	// No check specified and no paths found
	return s.category.Check == "" && len(s.category.Paths) == 0
}

func (s *PathScanner) Scan() (*types.ScanResult, error) {
	result := &types.ScanResult{
		Category: s.category,
		Items:    make([]types.CleanableItem, 0),
	}

	if !s.IsAvailable() {
		return result, nil
	}

	for _, pattern := range s.category.Paths {
		paths, err := utils.GlobPaths(pattern)
		if err != nil {
			continue
		}

		for _, path := range paths {
			// Skip SIP protected paths
			if IsSIPProtected(path) {
				continue
			}

			item, err := s.scanPath(path)
			if err != nil {
				continue
			}

			if s.category.DaysOld > 0 {
				cutoff := time.Now().AddDate(0, 0, -s.category.DaysOld)
				if item.ModifiedAt.After(cutoff) {
					continue
				}
			}

			result.Items = append(result.Items, item)
			result.TotalSize += item.Size
		}
	}

	return result, nil
}

func (s *PathScanner) scanPath(path string) (types.CleanableItem, error) {
	info, err := os.Stat(path)
	if err != nil {
		return types.CleanableItem{}, err
	}

	var size int64
	if info.IsDir() {
		size, _ = utils.GetDirSize(path)
	} else {
		size = info.Size()
	}

	return types.CleanableItem{
		Path:        path,
		Size:        size,
		Name:        filepath.Base(path),
		IsDirectory: info.IsDir(),
		ModifiedAt:  info.ModTime(),
	}, nil
}

func (s *PathScanner) Clean(items []types.CleanableItem, dryRun bool) (*types.CleanResult, error) {
	result := &types.CleanResult{
		Category: s.category,
		Errors:   make([]string, 0),
	}

	if dryRun {
		for _, item := range items {
			result.FreedSpace += item.Size
			result.CleanedItems++
		}
		return result, nil
	}

	return result, nil
}
