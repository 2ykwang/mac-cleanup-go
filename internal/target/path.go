package target

import (
	"os"
	"path/filepath"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

type PathTarget struct {
	category types.Category
}

func NewPathTarget(cat types.Category) *PathTarget {
	return &PathTarget{category: cat}
}

func (s *PathTarget) Category() types.Category {
	return s.category
}

func (s *PathTarget) IsAvailable() bool {
	// For command-based methods, check if command exists
	if s.category.CheckCmd != "" {
		return utils.CommandExists(s.category.CheckCmd)
	}

	// For path-based methods, check if any of the paths have matching files
	for _, pattern := range s.category.Paths {
		paths, err := utils.GlobPaths(pattern)
		if err == nil && len(paths) > 0 {
			return true
		}
	}

	// No paths configured - nothing to scan
	return false
}

func (s *PathTarget) Scan() (*types.ScanResult, error) {
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
			if utils.IsSIPProtected(path) {
				continue
			}

			item, err := s.scanPath(path)
			if err != nil {
				continue
			}

			result.Items = append(result.Items, item)
			result.TotalSize += item.Size
			result.TotalFileCount += item.FileCount
		}
	}

	return result, nil
}

func (s *PathTarget) scanPath(path string) (types.CleanableItem, error) {
	info, err := os.Stat(path)
	if err != nil {
		return types.CleanableItem{}, err
	}

	var size, fileCount int64
	if info.IsDir() {
		size, fileCount, _ = utils.GetDirSizeWithCount(path)
	} else {
		size = info.Size()
		fileCount = 1
	}

	return types.CleanableItem{
		Path:        path,
		Size:        size,
		FileCount:   fileCount,
		Name:        filepath.Base(path),
		IsDirectory: info.IsDir(),
		ModifiedAt:  info.ModTime(),
	}, nil
}

func (s *PathTarget) Clean(_ []types.CleanableItem) (*types.CleanResult, error) {
	result := &types.CleanResult{
		Category: s.category,
		Errors:   make([]string, 0),
	}

	return result, nil
}
