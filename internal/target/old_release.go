package target

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/2ykwang/mac-cleanup-go/internal/logger"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

// releasePolicy describes a tool that installs releases side by side in one
// directory and switches between them with a symlink.
type releasePolicy struct {
	// activeLink resolves to the release the tool runs now.
	activeLink string
	// pinFiles each hold one release name the tool may switch to next.
	pinFiles []string
	// keepPrevious is the number of releases older than the active one kept for rollback.
	keepPrevious int
}

var (
	errReleaseRootConfig  = errors.New(`old release category needs exactly one "<dir>/*" path`)
	errReleaseNotOld      = errors.New("no longer an unused old release")
	releaseVersionPattern = regexp.MustCompile(`^\d+(\.\d+)*`)
)

// OldReleaseTarget offers releases older than the active one. Releases that
// are pinned, kept for rollback, or open by a process are never offered.
type OldReleaseTarget struct {
	*PathTarget
	policy releasePolicy
}

func NewOldReleaseTarget(cat types.Category, policy releasePolicy) *OldReleaseTarget {
	return &OldReleaseTarget{
		PathTarget: NewPathTarget(cat),
		policy:     policy,
	}
}

func (t *OldReleaseTarget) Scan() (*types.ScanResult, error) {
	result := types.NewScanResult(t.category)
	if !t.IsAvailable() {
		return result, nil
	}

	root, err := t.releaseRoot()
	if err != nil {
		result.Error = err
		return result, nil
	}
	paths, err := t.oldReleases(root)
	if err != nil {
		logger.Warn("old release scan failed", "category", t.category.ID, "error", err)
		return result, nil
	}
	if len(paths) == 0 {
		return result, nil
	}

	result.Items, result.TotalSize, result.TotalFileCount = t.scanPathsParallel(paths)
	t.markInUse(root, result)
	return result, nil
}

// Clean re-plans and re-checks open files right before trashing, because the
// tool may switch releases or start a session after the scan.
func (t *OldReleaseTarget) Clean(items []types.CleanableItem) (*types.CleanResult, error) {
	result := types.NewCleanResult(t.category)
	if len(items) == 0 {
		return result, nil
	}

	allowed, err := t.removableNow()
	if err != nil {
		result.SkippedItems = len(items)
		return result, fmt.Errorf("re-checking %s releases: %w", t.category.ID, err)
	}

	batchResult := utils.BatchTrash(items, types.BatchTrashOptions{
		Category: t.category,
		Filter: func(item types.CleanableItem) bool {
			return utils.IsSIPProtected(item.Path)
		},
		Validate: func(item types.CleanableItem) error {
			if allowed[item.Path] {
				return nil
			}
			if _, err := os.Lstat(item.Path); os.IsNotExist(err) {
				return nil
			}
			return errReleaseNotOld
		},
	})
	result.Merge(batchResult)
	return result, nil
}

func (t *OldReleaseTarget) removableNow() (map[string]bool, error) {
	root, err := t.releaseRoot()
	if err != nil {
		return nil, err
	}
	paths, err := t.oldReleases(root)
	if err != nil {
		return nil, err
	}
	locked, err := t.lockedReleases(root)
	if err != nil {
		return nil, err
	}

	allowed := make(map[string]bool, len(paths))
	for _, p := range paths {
		if !locked[p] {
			allowed[p] = true
		}
	}
	return allowed, nil
}

// releaseRoot returns the resolved directory named by the category's single
// "<dir>/*" path. Paths are resolved so they compare equal to lsof output.
func (t *OldReleaseTarget) releaseRoot() (string, error) {
	if len(t.category.Paths) != 1 || !strings.HasSuffix(t.category.Paths[0], "/*") {
		return "", errReleaseRootConfig
	}
	dir := utils.ExpandPath(strings.TrimSuffix(t.category.Paths[0], "/*"))
	return filepath.EvalSymlinks(dir)
}

// oldReleases returns the releases under root that may be removed, newest
// first. It returns nothing when the active release can't be determined.
func (t *OldReleaseTarget) oldReleases(root string) ([]string, error) {
	active, ok := t.activeRelease(root)
	if !ok {
		return nil, nil
	}
	activeVersion, ok := releaseVersion(active)
	if !ok {
		logger.Info("old release scan skipped: active release has no version", "category", t.category.ID, "active", active)
		return nil, nil
	}
	pinned, err := t.pinnedReleases()
	if err != nil {
		return nil, err
	}
	pinned[active] = true

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", root, err)
	}

	type release struct {
		name    string
		version []int
	}
	var older []release
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || pinned[name] {
			continue
		}
		version, ok := releaseVersion(name)
		// Releases at or above the active version may be an update in progress.
		if !ok || compareVersions(version, activeVersion) >= 0 {
			continue
		}
		older = append(older, release{name: name, version: version})
	}

	sort.Slice(older, func(i, j int) bool {
		return compareVersions(older[i].version, older[j].version) > 0
	})
	if len(older) <= t.policy.keepPrevious {
		return nil, nil
	}

	var paths []string
	for _, r := range older[t.policy.keepPrevious:] {
		p := filepath.Join(root, r.name)
		if isRemovableRelease(p) {
			paths = append(paths, p)
		}
	}
	return paths, nil
}

func (t *OldReleaseTarget) activeRelease(root string) (string, bool) {
	resolved, err := filepath.EvalSymlinks(utils.ExpandPath(t.policy.activeLink))
	if err != nil {
		logger.Info("old release scan skipped: active release unknown", "category", t.category.ID, "error", err)
		return "", false
	}
	if filepath.Dir(resolved) != root {
		logger.Info("old release scan skipped: active release outside root",
			"category", t.category.ID, "active", resolved, "root", root)
		return "", false
	}
	return filepath.Base(resolved), true
}

func (t *OldReleaseTarget) pinnedReleases() (map[string]bool, error) {
	pinned := make(map[string]bool)
	for _, file := range t.policy.pinFiles {
		data, err := os.ReadFile(utils.ExpandPath(file))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("reading pin file %s: %w", file, err)
		}
		if name := strings.TrimSpace(string(data)); name != "" {
			pinned[name] = true
		}
	}
	return pinned, nil
}

// markInUse locks releases that a running process has open. When open files
// can't be listed, every release is locked.
func (t *OldReleaseTarget) markInUse(root string, result *types.ScanResult) {
	locked, err := t.lockedReleases(root)
	for i := range result.Items {
		if err != nil || locked[result.Items[i].Path] {
			result.Items[i].Status = types.ItemStatusProcessLocked
		}
	}
	if err != nil {
		logger.Warn("old release lock check failed", "category", t.category.ID, "error", err)
	}
}

func (t *OldReleaseTarget) lockedReleases(root string) (map[string]bool, error) {
	if !utils.CommandExists("lsof") {
		return nil, errors.New("lsof not found")
	}
	return getLockedPaths(root)
}

// isRemovableRelease rejects symlinks and hard-linked files. Trashing a file
// that another path still links to frees no space.
func isRemovableRelease(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	switch {
	case info.IsDir():
		return true
	case info.Mode().IsRegular():
		st, ok := info.Sys().(*syscall.Stat_t)
		return ok && st.Nlink == 1
	default:
		return false
	}
}

// releaseVersion parses the leading dotted number of a release name, such as
// "2.1.317" or "0.168.0-aarch64-apple-darwin".
func releaseVersion(name string) ([]int, bool) {
	match := releaseVersionPattern.FindString(name)
	if match == "" {
		return nil, false
	}
	parts := strings.Split(match, ".")
	version := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, false
		}
		version[i] = n
	}
	return version, true
}

func compareVersions(a, b []int) int {
	for i := 0; i < len(a) || i < len(b); i++ {
		var x, y int
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}
