package target

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

type testReleaseFixture struct {
	base       string
	root       string
	activeLink string
}

// newTestReleaseFixture creates a versions directory holding one file per
// release name and a symlink that points at the active release.
func newTestReleaseFixture(t *testing.T, active string, names ...string) testReleaseFixture {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "versions")
	require.NoError(t, os.Mkdir(root, 0o755))
	for _, name := range append([]string{active}, names...) {
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte(name), 0o755))
	}
	link := filepath.Join(base, "active")
	require.NoError(t, os.Symlink(filepath.Join(root, active), link))

	realBase, err := filepath.EvalSymlinks(base)
	require.NoError(t, err)
	return testReleaseFixture{
		base:       realBase,
		root:       filepath.Join(realBase, "versions"),
		activeLink: link,
	}
}

func newTestOldReleaseTarget(fx testReleaseFixture, policy releasePolicy) *OldReleaseTarget {
	policy.activeLink = fx.activeLink
	cat := types.Category{
		ID:     "claude-code-versions",
		Method: types.MethodBuiltin,
		Paths:  []string{fx.root + "/*"},
	}
	return NewOldReleaseTarget(cat, policy)
}

func stubReleaseLocks(t *testing.T, locked map[string]bool, err error) {
	t.Helper()
	originalLocked := getLockedPaths
	originalExists := utils.CommandExists
	t.Cleanup(func() {
		getLockedPaths = originalLocked
		utils.CommandExists = originalExists
	})
	utils.CommandExists = func(string) bool { return true }
	getLockedPaths = func(string) (map[string]bool, error) { return locked, err }
}

func itemPaths(items []types.CleanableItem) []string {
	paths := make([]string, len(items))
	for i, item := range items {
		paths[i] = item.Path
	}
	return paths
}

func TestOldReleaseTargetScan_OffersReleasesOlderThanActive(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.2.0", "1.0.0", "1.1.0")
	stubReleaseLocks(t, nil, nil)
	target := newTestOldReleaseTarget(fx, releasePolicy{})

	result, err := target.Scan()

	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join(fx.root, "1.0.0"),
		filepath.Join(fx.root, "1.1.0"),
	}, itemPaths(result.Items))
}

func TestOldReleaseTargetScan_KeepsPreviousReleasesForRollback(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.2.0", "1.0.0", "1.1.0")
	stubReleaseLocks(t, nil, nil)
	target := newTestOldReleaseTarget(fx, releasePolicy{keepPrevious: 1})

	result, err := target.Scan()

	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(fx.root, "1.0.0")}, itemPaths(result.Items))
}

func TestOldReleaseTargetScan_ComparesVersionsNumerically(t *testing.T) {
	fx := newTestReleaseFixture(t, "2.1.10", "2.1.9", "2.1.2")
	stubReleaseLocks(t, nil, nil)
	target := newTestOldReleaseTarget(fx, releasePolicy{keepPrevious: 1})

	result, err := target.Scan()

	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(fx.root, "2.1.2")}, itemPaths(result.Items))
}

func TestOldReleaseTargetScan_KeepsReleasesAtOrAboveActive(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.1.0", "1.0.0", "1.2.0")
	stubReleaseLocks(t, nil, nil)
	target := newTestOldReleaseTarget(fx, releasePolicy{})

	result, err := target.Scan()

	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(fx.root, "1.0.0")}, itemPaths(result.Items))
}

func TestOldReleaseTargetScan_KeepsPinnedRelease(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.2.0", "1.0.0", "1.1.0")
	pinFile := filepath.Join(fx.base, "auto-update-version")
	require.NoError(t, os.WriteFile(pinFile, []byte("1.1.0\n"), 0o644))
	stubReleaseLocks(t, nil, nil)
	target := newTestOldReleaseTarget(fx, releasePolicy{pinFiles: []string{pinFile}})

	result, err := target.Scan()

	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(fx.root, "1.0.0")}, itemPaths(result.Items))
}

func TestOldReleaseTargetScan_ReturnsNothing_WhenPinFileUnreadable(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.2.0", "1.0.0")
	pinDir := filepath.Join(fx.base, "auto-update-version")
	require.NoError(t, os.Mkdir(pinDir, 0o755))
	stubReleaseLocks(t, nil, nil)
	target := newTestOldReleaseTarget(fx, releasePolicy{pinFiles: []string{pinDir}})

	result, err := target.Scan()

	require.NoError(t, err)
	assert.Empty(t, result.Items)
}

func TestOldReleaseTargetScan_ReturnsNothing_WhenActiveLinkMissing(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.2.0", "1.0.0")
	require.NoError(t, os.Remove(fx.activeLink))
	stubReleaseLocks(t, nil, nil)
	target := newTestOldReleaseTarget(fx, releasePolicy{})

	result, err := target.Scan()

	require.NoError(t, err)
	assert.Empty(t, result.Items)
}

func TestOldReleaseTargetScan_ReturnsNothing_WhenActiveOutsideRoot(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.2.0", "1.0.0")
	outside := filepath.Join(fx.base, "2.0.0")
	require.NoError(t, os.WriteFile(outside, []byte("x"), 0o755))
	require.NoError(t, os.Remove(fx.activeLink))
	require.NoError(t, os.Symlink(outside, fx.activeLink))
	stubReleaseLocks(t, nil, nil)
	target := newTestOldReleaseTarget(fx, releasePolicy{})

	result, err := target.Scan()

	require.NoError(t, err)
	assert.Empty(t, result.Items)
}

func TestOldReleaseTargetScan_SkipsHardLinkedRelease(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.2.0", "1.0.0", "1.1.0")
	require.NoError(t, os.Link(filepath.Join(fx.root, "1.0.0"), filepath.Join(fx.base, "app-binary")))
	stubReleaseLocks(t, nil, nil)
	target := newTestOldReleaseTarget(fx, releasePolicy{})

	result, err := target.Scan()

	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(fx.root, "1.1.0")}, itemPaths(result.Items))
}

func TestOldReleaseTargetScan_SkipsSymlinkRelease(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.2.0", "1.1.0")
	require.NoError(t, os.Symlink(filepath.Join(fx.root, "1.1.0"), filepath.Join(fx.root, "1.0.0")))
	stubReleaseLocks(t, nil, nil)
	target := newTestOldReleaseTarget(fx, releasePolicy{})

	result, err := target.Scan()

	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(fx.root, "1.1.0")}, itemPaths(result.Items))
}

func TestOldReleaseTargetScan_SkipsUnversionedAndHiddenEntries(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.2.0", "1.0.0", "staging", ".1.1.0")
	stubReleaseLocks(t, nil, nil)
	target := newTestOldReleaseTarget(fx, releasePolicy{})

	result, err := target.Scan()

	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(fx.root, "1.0.0")}, itemPaths(result.Items))
}

func TestOldReleaseTargetScan_OffersDirectoryReleasesWithSuffix(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "releases")
	for _, name := range []string{"0.168.0-aarch64-apple-darwin", "0.169.0-aarch64-apple-darwin"} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, name, "bin"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, name, "bin", "codex"), []byte(name), 0o755))
	}
	link := filepath.Join(base, "current")
	require.NoError(t, os.Symlink(filepath.Join(root, "0.169.0-aarch64-apple-darwin"), link))
	realRoot, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)
	stubReleaseLocks(t, nil, nil)
	target := newTestOldReleaseTarget(testReleaseFixture{base: base, root: realRoot, activeLink: link}, releasePolicy{})

	result, err := target.Scan()

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, filepath.Join(realRoot, "0.168.0-aarch64-apple-darwin"), result.Items[0].Path)
	assert.True(t, result.Items[0].IsDirectory)
}

func TestOldReleaseTargetScan_LocksReleaseInUse(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.2.0", "1.0.0", "1.1.0")
	stubReleaseLocks(t, map[string]bool{filepath.Join(fx.root, "1.0.0"): true}, nil)
	target := newTestOldReleaseTarget(fx, releasePolicy{})

	result, err := target.Scan()

	require.NoError(t, err)
	require.Len(t, result.Items, 2)
	assert.Equal(t, types.ItemStatusProcessLocked, result.Items[0].Status)
	assert.Equal(t, types.ItemStatusAvailable, result.Items[1].Status)
}

func TestOldReleaseTargetScan_LocksAll_WhenLockCheckFails(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.2.0", "1.0.0", "1.1.0")
	stubReleaseLocks(t, nil, errors.New("lsof timeout"))
	target := newTestOldReleaseTarget(fx, releasePolicy{})

	result, err := target.Scan()

	require.NoError(t, err)
	require.Len(t, result.Items, 2)
	for _, item := range result.Items {
		assert.Equal(t, types.ItemStatusProcessLocked, item.Status)
	}
}

func TestOldReleaseTargetScan_LocksAll_WhenLsofMissing(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.2.0", "1.0.0")
	stubReleaseLocks(t, nil, nil)
	utils.CommandExists = func(string) bool { return false }
	target := newTestOldReleaseTarget(fx, releasePolicy{})

	result, err := target.Scan()

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, types.ItemStatusProcessLocked, result.Items[0].Status)
}

func TestOldReleaseTargetScan_ReportsError_WhenPathsMisconfigured(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.2.0", "1.0.0")
	cat := types.Category{
		ID:     "claude-code-versions",
		Method: types.MethodBuiltin,
		Paths:  []string{fx.root + "/*", fx.base + "/other/*"},
	}
	target := NewOldReleaseTarget(cat, releasePolicy{activeLink: fx.activeLink})

	result, err := target.Scan()

	require.NoError(t, err)
	assert.ErrorIs(t, result.Error, errReleaseRootConfig)
	assert.Empty(t, result.Items)
}

func TestOldReleaseTargetClean_TrashesOfferedRelease(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.2.0", "1.0.0")
	stubReleaseLocks(t, nil, nil)
	target := newTestOldReleaseTarget(fx, releasePolicy{})
	old := filepath.Join(fx.root, "1.0.0")

	result, err := target.Clean([]types.CleanableItem{{Path: old, Size: 5}})

	require.NoError(t, err)
	assert.Equal(t, 1, result.CleanedItems)
	assert.Empty(t, result.Errors)
	assert.NoFileExists(t, old)
}

func TestOldReleaseTargetClean_RefusesActiveRelease(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.2.0", "1.0.0")
	stubReleaseLocks(t, nil, nil)
	target := newTestOldReleaseTarget(fx, releasePolicy{})
	active := filepath.Join(fx.root, "1.2.0")

	result, err := target.Clean([]types.CleanableItem{{Path: active, Size: 5}})

	require.NoError(t, err)
	assert.Equal(t, 0, result.CleanedItems)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], errReleaseNotOld.Error())
	assert.FileExists(t, active)
}

func TestOldReleaseTargetClean_RefusesRelease_WhenActiveSwitchedAfterScan(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.2.0", "1.0.0")
	stubReleaseLocks(t, nil, nil)
	target := newTestOldReleaseTarget(fx, releasePolicy{})
	old := filepath.Join(fx.root, "1.0.0")
	require.NoError(t, os.Remove(fx.activeLink))
	require.NoError(t, os.Symlink(old, fx.activeLink))

	result, err := target.Clean([]types.CleanableItem{{Path: old, Size: 5}})

	require.NoError(t, err)
	assert.Equal(t, 0, result.CleanedItems)
	assert.Len(t, result.Errors, 1)
	assert.FileExists(t, old)
}

func TestOldReleaseTargetClean_RefusesRelease_WhenOpenedAfterScan(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.2.0", "1.0.0")
	old := filepath.Join(fx.root, "1.0.0")
	stubReleaseLocks(t, map[string]bool{old: true}, nil)
	target := newTestOldReleaseTarget(fx, releasePolicy{})

	result, err := target.Clean([]types.CleanableItem{{Path: old, Size: 5}})

	require.NoError(t, err)
	assert.Equal(t, 0, result.CleanedItems)
	assert.Len(t, result.Errors, 1)
	assert.FileExists(t, old)
}

func TestOldReleaseTargetClean_SkipsAll_WhenLockCheckFails(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.2.0", "1.0.0")
	stubReleaseLocks(t, nil, errors.New("lsof timeout"))
	target := newTestOldReleaseTarget(fx, releasePolicy{})
	old := filepath.Join(fx.root, "1.0.0")

	result, err := target.Clean([]types.CleanableItem{{Path: old, Size: 5}})

	require.Error(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.SkippedItems)
	assert.Equal(t, 0, result.CleanedItems)
	assert.FileExists(t, old)
}

func TestOldReleaseTargetClean_CountsAlreadyRemovedRelease(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.2.0", "1.0.0")
	stubReleaseLocks(t, nil, nil)
	target := newTestOldReleaseTarget(fx, releasePolicy{})
	gone := filepath.Join(fx.root, "0.9.0")

	result, err := target.Clean([]types.CleanableItem{{Path: gone, Size: 5}})

	require.NoError(t, err)
	assert.Equal(t, 1, result.CleanedItems)
	assert.Empty(t, result.Errors)
}

func TestOldReleaseTargetClean_EmptyItems(t *testing.T) {
	fx := newTestReleaseFixture(t, "1.2.0", "1.0.0")
	target := newTestOldReleaseTarget(fx, releasePolicy{})

	result, err := target.Clean(nil)

	require.NoError(t, err)
	assert.Equal(t, 0, result.CleanedItems)
	assert.Empty(t, result.Errors)
}

func TestReleaseVersion_ParsesLeadingDottedNumber(t *testing.T) {
	version, ok := releaseVersion("0.168.0-aarch64-apple-darwin")

	require.True(t, ok)
	assert.Equal(t, []int{0, 168, 0}, version)
}

func TestReleaseVersion_RejectsNameWithoutVersion(t *testing.T) {
	_, ok := releaseVersion("staging")

	assert.False(t, ok)
}

func TestCompareVersions_TreatsMissingPartsAsZero(t *testing.T) {
	result := compareVersions([]int{1, 2}, []int{1, 2, 0})

	assert.Equal(t, 0, result)
}
