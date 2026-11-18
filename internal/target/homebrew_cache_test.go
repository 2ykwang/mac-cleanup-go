package target

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func newTestBrewCommand(t *testing.T, output string) *int {
	t.Helper()
	bin := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(bin, "brew"), []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", bin)
	original := execCommand
	t.Cleanup(func() { execCommand = original })
	calls := new(int)
	execCommand = func(name string, args ...string) *exec.Cmd {
		assert.Equal(t, "brew", name)
		assert.Equal(t, []string{"--cache"}, args)
		*calls++
		return exec.Command("/bin/echo", output)
	}
	return calls
}

func newTestCacheFile(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("cache"), 0o644))
}

func newTestCacheRegistry(t *testing.T, caches string) *Registry {
	t.Helper()
	original := getLockedPaths
	getLockedPaths = func(string) (map[string]bool, error) { return nil, nil }
	t.Cleanup(func() { getLockedPaths = original })
	registry, err := DefaultRegistry(&types.Config{Categories: []types.Category{
		{ID: "system-cache", Method: types.MethodTrash, Paths: []string{filepath.Join(caches, "*")}},
		{ID: "homebrew", Method: types.MethodBuiltin, Safety: types.SafetyLevelSafe},
	}})
	require.NoError(t, err)
	return registry
}

func newTestScanCache(t *testing.T, registry *Registry, id string) *types.ScanResult {
	t.Helper()
	tgt, ok := registry.Get(id)
	require.True(t, ok)
	result, err := tgt.Scan()
	require.NoError(t, err)
	return result
}

func TestDefaultRegistry_HomebrewCacheCountedOnce(t *testing.T) {
	caches := filepath.Join(t.TempDir(), "Library", "Caches")
	root := filepath.Join(caches, "Homebrew")
	newTestCacheFile(t, filepath.Join(root, "package.tar.gz"))
	newTestCacheFile(t, filepath.Join(caches, "HomebrewOther", "data"))
	calls := newTestBrewCommand(t, root)
	registry := newTestCacheRegistry(t, caches)

	system := newTestScanCache(t, registry, "system-cache")
	brew := newTestScanCache(t, registry, "homebrew")

	require.Len(t, system.Items, 1)
	assert.Equal(t, "HomebrewOther", system.Items[0].Name)
	require.Len(t, brew.Items, 1)
	assert.Equal(t, int64(10), system.TotalSize+brew.TotalSize)
	assert.Equal(t, int64(2), system.TotalFileCount+brew.TotalFileCount)
	assert.Equal(t, 1, *calls)
}

func TestBrewTarget_CleanNeverRunsBrewCleanup(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Homebrew")
	newTestCacheFile(t, filepath.Join(root, "package"))
	calls := newTestBrewCommand(t, root)
	tgt := NewBrewTarget(types.Category{ID: "homebrew"})
	scan, err := tgt.Scan()
	require.NoError(t, err)
	require.Len(t, scan.Items, 1)
	require.NoError(t, os.RemoveAll(root))

	result, err := tgt.Clean(scan.Items)

	require.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Equal(t, 1, result.CleanedItems)
	assert.Equal(t, 1, *calls)
}

func TestDefaultRegistry_AbsentBrewPreservesSystemCaches(t *testing.T) {
	caches := t.TempDir()
	newTestCacheFile(t, filepath.Join(caches, "Homebrew", "data"))
	t.Setenv("PATH", t.TempDir())
	registry := newTestCacheRegistry(t, caches)

	system := newTestScanCache(t, registry, "system-cache")
	brew := newTestScanCache(t, registry, "homebrew")

	assert.Equal(t, int64(5), system.TotalSize)
	assert.Empty(t, brew.Items)
}
