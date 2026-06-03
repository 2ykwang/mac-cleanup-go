package target

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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
	{"dist", []string{"package.json", "tsconfig.json"}},                                           // JS/TS build output
	{".nox", []string{"noxfile.py"}},                                                              // python nox
	{".ruff_cache", []string{"pyproject.toml", "ruff.toml"}},                                      // ruff linter cache
	{"vendor", []string{"composer.json"}},                                                         // PHP composer install
	{"Pods", []string{"Podfile"}},                                                                 // iOS pod install
	{".expo", []string{"app.json", "app.config.js"}},                                              // expo prebuild
	{".angular", []string{"angular.json"}},                                                        // angular build cache
	{".svelte-kit", []string{"svelte.config.js", "svelte.config.ts"}},                             // sveltekit build
	{".nuxt", []string{"nuxt.config.ts", "nuxt.config.js"}},                                       // nuxt build
	{".astro", []string{"astro.config.mjs", "astro.config.ts"}},                                   // astro build
	{"coverage", []string{"jest.config.js", "jest.config.ts", "vitest.config.ts"}},                // test coverage output
	{".zig-cache", []string{"build.zig"}},                                                         // zig build cache
	{".dart_tool", []string{"pubspec.yaml"}},                                                      // dart/flutter tooling
	{"build", []string{"build.gradle", "build.gradle.kts"}},                                       // gradle build output
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
	// discoveryWorkers caps the parallel walk: per-directory work is tiny, so more
	// than a few workers only adds queue contention, not speed.
	discoveryWorkers = 4
)

type foundCache struct {
	path    string
	pattern cachePattern
}

type walkItem struct {
	path  string
	depth int
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

	found := t.discoverCaches(patternMap)

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

// discoverCaches walks scanRoot in parallel and returns the project cache dirs
// that classifyDir identifies (pattern-named dirs validated by a marker file).
func (t *ProjectCacheTarget) discoverCaches(patternMap map[string][]cachePattern) []foundCache {
	workers := discoveryWorkers
	if w := utils.DefaultWorkers(); w < workers {
		workers = w
	}

	var (
		mu      sync.Mutex
		cond    = sync.NewCond(&mu)
		found   []foundCache
		queue   = []walkItem{{path: t.scanRoot, depth: 0}}
		pending = 1
		done    bool
	)

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for {
				mu.Lock()
				for len(queue) == 0 && !done {
					cond.Wait()
				}
				if done {
					mu.Unlock()
					return
				}
				last := len(queue) - 1
				cur := queue[last]
				queue = queue[:last]
				mu.Unlock()

				children, localFound := t.classifyDir(cur.path, cur.depth, patternMap)

				mu.Lock()
				found = append(found, localFound...)
				if len(children) > 0 {
					queue = append(queue, children...)
					pending += len(children)
					cond.Broadcast()
				}
				pending--
				if pending == 0 {
					done = true
					cond.Broadcast()
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return found
}

// classifyDir reads one directory and returns its subdirectories split into
// caches to record and children to keep walking.
func (t *ProjectCacheTarget) classifyDir(dir string, depth int, patternMap map[string][]cachePattern) (children []walkItem, found []foundCache) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		logger.Debug("project cache walk readdir failed", "path", dir, "error", err)
		return nil, nil
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		childDepth := depth + 1
		if childDepth > maxScanDepth {
			continue
		}
		if childDepth == 1 {
			if _, excluded := excludeDirs[name]; excluded {
				continue
			}
		}

		childPath := filepath.Join(dir, name)
		patterns, ok := patternMap[name]
		if !ok {
			children = append(children, walkItem{path: childPath, depth: childDepth})
			continue
		}
		for _, p := range patterns {
			if hasMarker(dir, p.MarkerFiles) {
				found = append(found, foundCache{path: childPath, pattern: p})
				break // first matching pattern wins (e.g. target/ → Cargo before Maven)
			}
		}
		// cache-named dirs are pruned (not added to children) even without a marker
	}
	return children, found
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

			size, count, err := utils.GetDirSizeWithCount(fc.path, utils.WithIgnoreNames(ignoredNames...))
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
				ModifiedAt:  newestFileTime(fc.path),
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

// newestFileTime walks dir and returns the most recent file mtime.
// Directory mtime only reflects direct child add/remove, not deeper file updates,
// so we must check actual file mtimes to determine staleness.
func newestFileTime(dir string) time.Time {
	var newest time.Time
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if mt := info.ModTime(); mt.After(newest) {
			newest = mt
		}
		return nil
	})
	return newest
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
