package scanner

import (
	"strings"

	"mac-cleanup-go/internal/utils"
	"mac-cleanup-go/pkg/types"
)

// SystemCacheScanner scans system cache excluding paths defined in other categories
type SystemCacheScanner struct {
	*BaseScanner
	excludePaths []string
}

func NewSystemCacheScanner(cat types.Category, allCategories []types.Category) *SystemCacheScanner {
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

	return &SystemCacheScanner{
		BaseScanner:  NewBaseScanner(cat),
		excludePaths: excludes,
	}
}

func (s *SystemCacheScanner) Scan() (*types.ScanResult, error) {
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
		}
	}

	return result, nil
}

func (s *SystemCacheScanner) isExcluded(path string) bool {
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
