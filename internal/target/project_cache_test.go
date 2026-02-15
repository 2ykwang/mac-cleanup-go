package target

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func newTestProjectCacheTarget(scanRoot string) *ProjectCacheTarget {
	return &ProjectCacheTarget{
		category: types.Category{
			ID:     "project-cache",
			Name:   "Project Build Cache",
			Safety: types.SafetyLevelModerate,
			Method: types.MethodBuiltin,
		},
		scanRoot:  scanRoot,
		patterns:  defaultPatterns,
		staleDays: 0, // no stale filter by default in tests
	}
}

// makeStale sets a directory's mtime to 30 days ago.
func makeStale(t *testing.T, path string) {
	t.Helper()
	old := time.Now().AddDate(0, 0, -30)
	require.NoError(t, os.Chtimes(path, old, old))
}

// --- Scan: Detection ---

func TestScan_DetectsNodeModulesWithMarker(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "myapp")
	require.NoError(t, os.MkdirAll(filepath.Join(project, "node_modules", "express"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(project, "package.json"), []byte("{}"), 0o644))

	target := newTestProjectCacheTarget(root)
	result, err := target.Scan()

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "myapp/node_modules", result.Items[0].DisplayName)
}

func TestScan_DetectsVenvWithMarker(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "myapp")
	require.NoError(t, os.MkdirAll(filepath.Join(project, ".venv", "lib"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(project, "pyproject.toml"), []byte(""), 0o644))

	target := newTestProjectCacheTarget(root)
	result, err := target.Scan()

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "myapp/.venv", result.Items[0].DisplayName)
}

func TestScan_DetectsMultiplePatterns(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "myapp")
	require.NoError(t, os.MkdirAll(filepath.Join(project, ".venv"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(project, "node_modules"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(project, "pyproject.toml"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(project, "package.json"), []byte("{}"), 0o644))

	target := newTestProjectCacheTarget(root)
	result, err := target.Scan()

	require.NoError(t, err)
	assert.Len(t, result.Items, 2)
}

func TestScan_DetectsCargoTarget(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "rustapp")
	require.NoError(t, os.MkdirAll(filepath.Join(project, "target", "debug"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(project, "Cargo.toml"), []byte(""), 0o644))

	target := newTestProjectCacheTarget(root)
	result, err := target.Scan()

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "rustapp/target", result.Items[0].DisplayName)
}

// --- Scan: Ignore without marker ---

func TestScan_IgnoresWithoutMarker(t *testing.T) {
	root := t.TempDir()
	orphan := filepath.Join(root, "orphan")
	require.NoError(t, os.MkdirAll(filepath.Join(orphan, "node_modules"), 0o755))
	// no package.json

	target := newTestProjectCacheTarget(root)
	result, err := target.Scan()

	require.NoError(t, err)
	assert.Empty(t, result.Items)
}

func TestScan_IgnoresTargetWithoutCargoOrPom(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "generic")
	require.NoError(t, os.MkdirAll(filepath.Join(project, "target"), 0o755))
	// no Cargo.toml or pom.xml

	target := newTestProjectCacheTarget(root)
	result, err := target.Scan()

	require.NoError(t, err)
	assert.Empty(t, result.Items)
}

// --- Scan: Exclusion ---

func TestScan_SkipsExcludedDirs(t *testing.T) {
	root := t.TempDir()
	// Create a project inside an excluded dir name
	excluded := filepath.Join(root, "Library", "myapp")
	require.NoError(t, os.MkdirAll(filepath.Join(excluded, "node_modules"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(excluded, "package.json"), []byte("{}"), 0o644))

	target := newTestProjectCacheTarget(root)
	result, err := target.Scan()

	require.NoError(t, err)
	assert.Empty(t, result.Items)
}

func TestScan_DoesNotExcludeAtDeeperDepth(t *testing.T) {
	root := t.TempDir()
	// "Library" at depth 2 should not be excluded
	project := filepath.Join(root, "projects", "Library")
	require.NoError(t, os.MkdirAll(filepath.Join(project, "node_modules"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(project, "package.json"), []byte("{}"), 0o644))

	target := newTestProjectCacheTarget(root)
	result, err := target.Scan()

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
}

// --- Scan: Prune ---

func TestScan_PrunesCacheInsideCache(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "myapp")
	// nested node_modules inside node_modules
	require.NoError(t, os.MkdirAll(filepath.Join(project, "node_modules", "dep", "node_modules"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(project, "package.json"), []byte("{}"), 0o644))
	// venv inside .tox
	require.NoError(t, os.MkdirAll(filepath.Join(project, ".tox", "py39", ".venv"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(project, "tox.ini"), []byte(""), 0o644))

	target := newTestProjectCacheTarget(root)
	result, err := target.Scan()

	require.NoError(t, err)
	// Should find node_modules and .tox, not nested ones inside them
	assert.Len(t, result.Items, 2)
	names := make(map[string]bool)
	for _, item := range result.Items {
		names[filepath.Base(item.Path)] = true
	}
	assert.True(t, names["node_modules"])
	assert.True(t, names[".tox"])
}

// --- Scan: Depth limit ---

func TestScan_RespectsMaxDepth(t *testing.T) {
	root := t.TempDir()
	// Create project at depth 9 (exceeds maxScanDepth=8)
	deep := filepath.Join(root, "a", "b", "c", "d", "e", "f", "g", "h", "project")
	require.NoError(t, os.MkdirAll(filepath.Join(deep, "node_modules"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(deep, "package.json"), []byte("{}"), 0o644))

	target := newTestProjectCacheTarget(root)
	result, err := target.Scan()

	require.NoError(t, err)
	assert.Empty(t, result.Items)
}

// --- Scan: Stale filter ---

func TestScan_FiltersStaleCachesOnly(t *testing.T) {
	root := t.TempDir()

	// Stale project (30 days old)
	staleProject := filepath.Join(root, "stale")
	require.NoError(t, os.MkdirAll(filepath.Join(staleProject, ".venv"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(staleProject, "requirements.txt"), []byte(""), 0o644))
	makeStale(t, filepath.Join(staleProject, ".venv"))

	// Active project (just created)
	activeProject := filepath.Join(root, "active")
	require.NoError(t, os.MkdirAll(filepath.Join(activeProject, ".venv"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(activeProject, "requirements.txt"), []byte(""), 0o644))

	target := newTestProjectCacheTarget(root)
	target.staleDays = 7

	result, err := target.Scan()

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Contains(t, result.Items[0].DisplayName, "stale")
}

// --- Scan: Edge cases ---

func TestScan_EmptyScanRoot_ReturnsEmpty(t *testing.T) {
	target := &ProjectCacheTarget{scanRoot: ""}

	result, err := target.Scan()

	require.NoError(t, err)
	assert.Empty(t, result.Items)
}

func TestScan_EmptyDirectory_ReturnsEmpty(t *testing.T) {
	root := t.TempDir()

	target := newTestProjectCacheTarget(root)
	result, err := target.Scan()

	require.NoError(t, err)
	assert.Empty(t, result.Items)
}

// --- hasMarker ---

func TestHasMarker_ReturnsTrue_WhenMarkerExists(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o644))

	assert.True(t, hasMarker(dir, []string{"package.json"}))
}

func TestHasMarker_ReturnsFalse_WhenNoMarker(t *testing.T) {
	dir := t.TempDir()

	assert.False(t, hasMarker(dir, []string{"package.json", "pyproject.toml"}))
}

func TestHasMarker_ReturnsTrue_WhenAnyMarkerExists(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "setup.py"), []byte(""), 0o644))

	assert.True(t, hasMarker(dir, []string{"pyproject.toml", "requirements.txt", "setup.py"}))
}

// --- formatDisplayName ---

func TestFormatDisplayName_ReturnsRelativePath(t *testing.T) {
	assert.Equal(t, "Projects/myapp/.venv", formatDisplayName("/home/user", "/home/user/Projects/myapp/.venv"))
}

func TestFormatDisplayName_FallsBackToBasename(t *testing.T) {
	// When Rel fails (different roots), should fallback
	result := formatDisplayName("/home/user", "/other/root/.venv")
	assert.Equal(t, ".venv", filepath.Base(result))
}

// --- Clean ---

func TestClean_EmptyItems_ReturnsZero(t *testing.T) {
	target := newTestProjectCacheTarget(t.TempDir())
	result, err := target.Clean([]types.CleanableItem{})

	require.NoError(t, err)
	assert.Equal(t, 0, result.CleanedItems)
	assert.Empty(t, result.Errors)
}
