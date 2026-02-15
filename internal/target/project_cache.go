package target

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/2ykwang/mac-cleanup-go/internal/logger"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

// cachePattern defines a build artifact directory and its project marker files.
// A directory is identified as a project cache only when:
//  1. Its name matches DirName, AND
//  2. At least one MarkerFiles exists in the parent directory
//
// This two-step validation distinguishes project caches (safe to delete)
// from system tool dependencies (e.g. ~/.nvm/**/node_modules).
type cachePattern struct {
	DirName     string
	MarkerFiles []string
}

var defaultPatterns = []cachePattern{
	{"node_modules", []string{"package.json"}},                                                    // npm install
	{".venv", []string{"pyproject.toml", "requirements.txt", "setup.py", "setup.cfg", "Pipfile"}}, // python -m venv && pip install
	{".tox", []string{"tox.ini", "pyproject.toml", "setup.cfg"}},                                  // tox
	{".mypy_cache", []string{"pyproject.toml", "mypy.ini", "setup.cfg"}},                          // mypy
	{".pytest_cache", []string{"pyproject.toml", "pytest.ini", "setup.cfg", "conftest.py"}},       // pytest
	{".next", []string{"next.config.js", "next.config.mjs", "next.config.ts"}},                    // next build
	{"target", []string{"Cargo.toml"}},                                                            // cargo build
	{"target", []string{"pom.xml"}},                                                               // mvn compile
	{".gradle", []string{"build.gradle", "build.gradle.kts", "settings.gradle"}},                  // gradle build
}

// excludeDirs: top-level directories under $HOME to skip during walk.
var excludeDirs = map[string]struct{}{
	// macOS system
	"Library": {}, "Applications": {}, ".Trash": {},
	"Music": {}, "Movies": {}, "Pictures": {}, "Public": {},
	// package manager / toolchain installations
	".npm": {}, ".nvm": {}, ".yarn": {}, ".pnpm": {},
	".cargo": {}, ".rustup": {}, ".gradle": {},
	".local": {}, ".cache": {}, ".docker": {},
	// editor plugins (contain node_modules with package.json)
	".vscode": {}, ".cursor": {}, ".hyper_plugins": {}, ".claude": {},
}

const (
	maxScanDepth     = 8
	defaultStaleDays = 7
)

type foundCache struct {
	path    string
	pattern cachePattern
}

// ProjectCacheTarget scans $HOME recursively for stale build caches
// inside project directories, using marker-file validation to avoid false positives.
type ProjectCacheTarget struct {
	category  types.Category
	scanRoot  string // overridable for testing
	patterns  []cachePattern
	staleDays int
}

func NewProjectCacheTarget(cat types.Category) *ProjectCacheTarget {
	home, _ := os.UserHomeDir()
	return &ProjectCacheTarget{
		category:  cat,
		scanRoot:  home,
		patterns:  defaultPatterns,
		staleDays: defaultStaleDays,
	}
}

func (t *ProjectCacheTarget) Category() types.Category { return t.category }
func (t *ProjectCacheTarget) IsAvailable() bool        { return t.scanRoot != "" }

func (t *ProjectCacheTarget) Scan() (*types.ScanResult, error) {
	result := types.NewScanResult(t.category)
	if !t.IsAvailable() {
		return result, nil
	}

	start := time.Now()

	patternMap := make(map[string][]cachePattern)
	for _, p := range t.patterns {
		patternMap[p.DirName] = append(patternMap[p.DirName], p)
	}

	var found []foundCache

	//nolint:errcheck // WalkDir errors are handled per-entry
	filepath.WalkDir(t.scanRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == t.scanRoot {
			return nil
		}

		name := d.Name()

		rel, _ := filepath.Rel(t.scanRoot, path)
		depth := strings.Count(rel, string(filepath.Separator)) + 1
		if depth > maxScanDepth {
			return fs.SkipDir
		}

		if depth == 1 {
			if _, excluded := excludeDirs[name]; excluded {
				return fs.SkipDir
			}
		}

		if patterns, ok := patternMap[name]; ok {
			parentDir := filepath.Dir(path)
			for _, p := range patterns {
				if hasMarker(parentDir, p.MarkerFiles) {
					found = append(found, foundCache{path: path, pattern: p})
					break // first matching pattern wins (e.g. target/ → Cargo before Maven)
				}
			}
			// Always prune cache-named dirs even without marker —
			// they are likely tool dependencies, and recursing into them is expensive.
			return fs.SkipDir
		}

		return nil
	})

	walkDuration := time.Since(start)
	logger.Info("project cache walk complete",
		"found", len(found),
		"walk_ms", walkDuration.Milliseconds())

	allItems, _, _ := t.calculateSizes(found)

	// Filter to stale caches only — protect active projects
	cutoff := time.Now().AddDate(0, 0, -t.staleDays)
	for _, item := range allItems {
		if item.ModifiedAt.Before(cutoff) {
			result.Items = append(result.Items, item)
			result.TotalSize += item.Size
			result.TotalFileCount += item.FileCount
		}
	}

	logger.Info("project cache scan complete",
		"found", len(allItems),
		"stale", len(result.Items),
		"total_size", result.TotalSize,
		"total_ms", time.Since(start).Milliseconds())

	return result, nil
}

func (t *ProjectCacheTarget) calculateSizes(found []foundCache) ([]types.CleanableItem, int64, int64) {
	var (
		wg         sync.WaitGroup
		mu         sync.Mutex
		items      []types.CleanableItem
		totalSize  int64
		totalCount int64
	)

	sem := make(chan struct{}, utils.DefaultWorkers())

	for _, fc := range found {
		sem <- struct{}{}
		wg.Add(1)
		go func(fc foundCache) {
			defer wg.Done()
			defer func() { <-sem }()

			size, count, err := utils.GetDirSizeWithCount(fc.path)
			if err != nil {
				return
			}
			info, err := os.Stat(fc.path)
			if err != nil {
				return
			}

			item := types.CleanableItem{
				Path:        fc.path,
				Size:        size,
				FileCount:   count,
				Name:        filepath.Base(fc.path),
				DisplayName: formatDisplayName(t.scanRoot, fc.path),
				IsDirectory: true,
				ModifiedAt:  info.ModTime(),
			}

			mu.Lock()
			items = append(items, item)
			totalSize += size
			totalCount += count
			mu.Unlock()
		}(fc)
	}
	wg.Wait()

	sort.Slice(items, func(i, j int) bool {
		return items[i].Path < items[j].Path
	})

	return items, totalSize, totalCount
}

func hasMarker(parentDir string, markers []string) bool {
	for _, m := range markers {
		if _, err := os.Stat(filepath.Join(parentDir, m)); err == nil {
			return true
		}
	}
	return false
}

func formatDisplayName(scanRoot, cachePath string) string {
	rel, err := filepath.Rel(scanRoot, cachePath)
	if err != nil {
		return filepath.Base(cachePath)
	}
	return rel
}

func (t *ProjectCacheTarget) Clean(items []types.CleanableItem) (*types.CleanResult, error) {
	result := types.NewCleanResult(t.category)
	if len(items) == 0 {
		return result, nil
	}

	batchResult := utils.BatchTrash(items, types.BatchTrashOptions{
		Category: t.category,
	})
	result.Merge(batchResult)
	return result, nil
}
