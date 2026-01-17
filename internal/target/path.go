package target

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

// getMaxWorkers returns the optimal number of workers based on CPU cores
func getMaxWorkers(numCPU int) int {
	if numCPU > 16 {
		return 16
	}
	if numCPU < 4 {
		return 4
	}
	return numCPU
}

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

	paths := s.collectPaths()
	if len(paths) == 0 {
		return result, nil
	}

	result.Items, result.TotalSize, result.TotalFileCount = s.scanPathsParallel(paths)
	return result, nil
}

// collectPaths gathers all paths from glob patterns, filtering out SIP protected paths
func (s *PathTarget) collectPaths() []string {
	var paths []string
	for _, pattern := range s.category.Paths {
		matched, err := utils.GlobPaths(pattern)
		if err != nil {
			continue
		}
		for _, p := range matched {
			if !utils.IsSIPProtected(p) {
				paths = append(paths, p)
			}
		}
	}
	return paths
}

// scanPathsParallel scans multiple paths concurrently using a worker pool
func (s *PathTarget) scanPathsParallel(paths []string) ([]types.CleanableItem, int64, int64) {
	var (
		wg         sync.WaitGroup
		mu         sync.Mutex
		items      []types.CleanableItem
		totalSize  int64
		totalCount int64
	)

	sem := make(chan struct{}, getMaxWorkers(runtime.NumCPU()))

	for _, path := range paths {
		sem <- struct{}{}
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }()

			item, err := s.scanPath(p)
			if err != nil {
				return
			}

			mu.Lock()
			items = append(items, item)
			totalSize += item.Size
			totalCount += item.FileCount
			mu.Unlock()
		}(path)
	}
	wg.Wait()

	// Sort for consistent ordering
	sort.Slice(items, func(i, j int) bool {
		return items[i].Path < items[j].Path
	})

	return items, totalSize, totalCount
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
