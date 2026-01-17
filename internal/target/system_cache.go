package target

import (
	"strings"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

// SystemCacheTarget scans system cache excluding paths defined in other categories
type SystemCacheTarget struct {
	*PathTarget
	excludePaths []string
}

func NewSystemCacheTarget(cat types.Category, allCategories []types.Category) *SystemCacheTarget {
	var excludes []string
	for _, other := range allCategories {
		if other.ID == cat.ID {
			continue
		}
		for _, p := range other.Paths {
			expanded := utils.ExpandPath(p)
			// Remove trailing patterns for prefix matching
			expanded = strings.TrimSuffix(expanded, "/**")
			expanded = strings.TrimSuffix(expanded, "/*")
			expanded = strings.TrimSuffix(expanded, "*")
			// Ensure path ends with / for proper prefix matching
			if !strings.HasSuffix(expanded, "/") {
				expanded = expanded + "/"
			}
			excludes = append(excludes, expanded)
		}
	}

	return &SystemCacheTarget{
		PathTarget:   NewPathTarget(cat),
		excludePaths: excludes,
	}
}

func (s *SystemCacheTarget) Scan() (*types.ScanResult, error) {
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
			if s.isExcluded(path) {
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

func (s *SystemCacheTarget) isExcluded(path string) bool {
	// Normalize path to end with / for consistent matching
	pathWithSlash := path
	if !strings.HasSuffix(pathWithSlash, "/") {
		pathWithSlash = path + "/"
	}

	for _, exclude := range s.excludePaths {
		// Check if path is the exclude dir itself or a child of it
		if strings.HasPrefix(pathWithSlash, exclude) {
			return true
		}
	}
	return false
}
