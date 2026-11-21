package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func TestOlderThan_OverridesTargetDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cache := filepath.Join(home, "app", ".next")
	require.NoError(t, os.MkdirAll(cache, 0o755))
	writeFiles(t, filepath.Dir(cache), map[string]int{"next.config.js": 1})
	cached := writeFiles(t, cache, map[string]int{"data": 3})["data"]
	downloads := t.TempDir()
	download := writeFiles(t, downloads, map[string]int{"archive": 5})["archive"]
	threeDaysAgo := time.Now().Add(-3 * day)
	for _, path := range []string{cache, cached, download} {
		require.NoError(t, os.Chtimes(path, threeDaysAgo, threeDaysAgo))
	}
	cfg := testConfig(
		testCategory("project-cache", types.SafetyLevelModerate, types.MethodBuiltin),
		testCategory("old-downloads", types.SafetyLevelRisky, types.MethodBuiltin, filepath.Join(downloads, "*")),
	)
	t.Cleanup(stubSession(cfg, emptyUserConfig()))

	assert.Zero(t, runScanJSON(t).SizeBytes)
	scan := runScanJSON(t, "project-cache", "--older-than", "1d", "old-downloads", "--items")
	assert.EqualValues(t, 8, scan.SizeBytes)
	assert.EqualValues(t, 3, scan.CleanableBytes)
	clean := runCleanJSON(t, ExitOK, "project-cache", "old-downloads", "--older-than=1d", "--dry-run")
	require.Len(t, clean.Targets, 1)
	assert.Equal(t, []string{cache}, clean.Targets[0].ItemPaths)
	assert.Equal(t, skipRiskyNeedItem, skipReasons(clean)["old-downloads"])
	picked := runCleanJSON(t, ExitOK, "old-downloads", "--older-than", "1d", "--item", download, "--dry-run")
	require.Len(t, picked.Targets, 1)
	assert.Equal(t, []string{download}, picked.Targets[0].ItemPaths)
}

func TestOlderThan_CleanUsesSameItemsAsScan(t *testing.T) {
	dir := t.TempDir()
	files := writeFiles(t, dir, map[string]int{"old": 2, "recent": 4})
	old := time.Now().Add(-31 * day)
	require.NoError(t, os.Chtimes(files["old"], old, old))
	user := emptyUserConfig()
	user.SelectedTargets = []string{"logs"}
	t.Cleanup(stubSession(testConfig(testCategory("logs", types.SafetyLevelSafe, types.MethodPermanent, dir+"/*")), user))

	scan := runScanJSON(t, "logs", "--older-than=30d", "--items")
	require.Len(t, scan.Targets[0].Items, 1)
	item := scan.Targets[0].Items[0]
	assert.Equal(t, files["old"], item.Path)
	plan := runCleanJSON(t, ExitOK, "--selected", "--older-than=30d", "--dry-run")
	require.Len(t, plan.Targets, 1)
	assert.Equal(t, []string{item.Path}, plan.Targets[0].ItemPaths)
	assert.Equal(t, scan.SizeBytes, plan.FreedBytes)
	app := newTestApp("")
	require.Equal(t, ExitUsage, app.Run([]string{"clean", "logs", "--item", files["recent"], "--older-than=30d", "--yes", "--json"}))
	assert.Equal(t, "unknown_item", decodeError(t, app.err.Bytes()).Code)

	out := runCleanJSON(t, ExitOK, "--selected", "--older-than=30d", "--yes")
	assert.Equal(t, 1, out.CleanedItems)
	assert.NoFileExists(t, item.Path)
	assert.FileExists(t, files["recent"])
}

func TestOlderThan_RejectsInvalidAge(t *testing.T) {
	for _, command := range []string{"scan", "clean"} {
		for _, age := range []string{"", "1", "-1d", "1.5d", "99999999999999d"} {
			app := newTestApp("")
			code := app.Run([]string{command, "logs", "--older-than=" + age, "--json"})
			assert.Equal(t, ExitUsage, code)
			assert.Equal(t, "invalid_flag", decodeError(t, app.err.Bytes()).Code)
		}
	}
}

func TestOlderThan_ZeroDaysSkipsUndatedDockerItems(t *testing.T) {
	calls := newTestDockerSession(t)
	require.Positive(t, runScanJSON(t, "docker").Targets[0].ItemCount)

	scan := runScanJSON(t, "docker", "--older-than=0d")
	clean := runCleanJSON(t, ExitOK, "docker", "--older-than=0d", "--yes")

	assert.Zero(t, scan.Targets[0].ItemCount)
	assert.Empty(t, clean.Targets)
	assert.Empty(t, newTestDockerCalls(t, calls))
}

func TestOlderThan_RecentNestedFileKeepsDirectoryOutOfCleanup(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "cache", "nested")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	file := writeFiles(t, nested, map[string]int{"data": 3})["data"]
	older := time.Now().Add(-45 * day)
	recent := older.Add(44 * day)
	require.NoError(t, os.Chtimes(file, recent, recent))
	require.NoError(t, os.Chtimes(filepath.Dir(nested), older, older))
	cfg := testConfig(testCategory("logs", types.SafetyLevelSafe, types.MethodTrash, filepath.Join(root, "*")))
	t.Cleanup(stubSession(cfg, emptyUserConfig()))

	scan := runScanJSON(t, "logs", "--items")
	require.Len(t, scan.Targets[0].Items, 1)
	info, err := os.Lstat(file)
	require.NoError(t, err)
	assert.True(t, scan.Targets[0].Items[0].ModifiedAt.Equal(info.ModTime()))
	assert.Zero(t, runScanJSON(t, "logs", "--older-than", "30d").SizeBytes)
	assert.Empty(t, runCleanJSON(t, ExitOK, "logs", "--older-than", "30d", "--dry-run").Targets)
}
