package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func newTestStatsFile(t *testing.T, root, name string, size int, modified time.Time) string {
	t.Helper()
	path := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, make([]byte, size), 0o644))
	require.NoError(t, os.Chtimes(path, modified, modified))
	return path
}

func newTestStatsTimes() (time.Time, time.Time) {
	cutoff := time.Unix(1700000000, 0)
	return cutoff, cutoff.Add(-24 * time.Hour)
}

func TestGetDirStats_CollectsSizeCountAndNewestFileTime(t *testing.T) {
	root := t.TempDir()
	cutoff, old := newTestStatsTimes()
	latest := old.Add(time.Hour)
	newTestStatsFile(t, root, "first", 8, old)
	newTestStatsFile(t, root, "nested/second", 12, latest)

	stats, err := GetDirStats(root, cutoff)

	require.NoError(t, err)
	assert.Equal(t, int64(20), stats.Size)
	assert.Equal(t, int64(2), stats.FileCount)
	assert.True(t, latest.Equal(stats.NewestFileTime))
	assert.False(t, stats.Recent)
}

func TestGetDirStats_IgnoredFileStillDeterminesNewestTime(t *testing.T) {
	root := t.TempDir()
	cutoff, old := newTestStatsTimes()
	latest := old.Add(time.Hour)
	newTestStatsFile(t, root, "file", 8, old)
	newTestStatsFile(t, root, ".DS_Store", 100, latest)

	stats, err := GetDirStats(root, cutoff, WithIgnoreNames(".DS_Store"))

	require.NoError(t, err)
	assert.Equal(t, int64(8), stats.Size)
	assert.Equal(t, int64(1), stats.FileCount)
	assert.True(t, latest.Equal(stats.NewestFileTime))
	assert.False(t, stats.Recent)
}

func TestGetDirStats_RecentIgnoredFileStopsScan(t *testing.T) {
	root := t.TempDir()
	cutoff, _ := newTestStatsTimes()
	newTestStatsFile(t, root, ".DS_Store", 100, cutoff.Add(time.Second))

	stats, err := GetDirStats(root, cutoff, WithIgnoreNames(".DS_Store"))

	require.NoError(t, err)
	assert.True(t, stats.Recent)
	assert.False(t, stats.NewestFileTime.Before(cutoff))
}

func TestGetDirStats_FileAtCutoffIsRecent(t *testing.T) {
	root := t.TempDir()
	cutoff, _ := newTestStatsTimes()
	newTestStatsFile(t, root, "file", 8, cutoff)

	stats, err := GetDirStats(root, cutoff)

	require.NoError(t, err)
	assert.True(t, stats.Recent)
	assert.True(t, cutoff.Equal(stats.NewestFileTime))
}

func TestGetDirStats_FileBeforeCutoffIsStale(t *testing.T) {
	root := t.TempDir()
	cutoff, _ := newTestStatsTimes()
	modified := cutoff.Add(-time.Nanosecond)
	newTestStatsFile(t, root, "file", 8, modified)

	stats, err := GetDirStats(root, cutoff)

	require.NoError(t, err)
	assert.False(t, stats.Recent)
	assert.Equal(t, int64(8), stats.Size)
	assert.True(t, modified.Equal(stats.NewestFileTime))
}

func TestGetDirStats_ZeroCutoffCollectsAllFiles(t *testing.T) {
	root := t.TempDir()
	cutoff, _ := newTestStatsTimes()
	latest := cutoff.Add(time.Hour)
	newTestStatsFile(t, root, "first", 8, cutoff)
	newTestStatsFile(t, root, "second", 12, latest)

	stats, err := GetDirStats(root, time.Time{})

	require.NoError(t, err)
	assert.False(t, stats.Recent)
	assert.Equal(t, int64(20), stats.Size)
	assert.Equal(t, int64(2), stats.FileCount)
	assert.True(t, latest.Equal(stats.NewestFileTime))
}

func TestGetDirStats_EmptyDirectoryHasNoFileTime(t *testing.T) {
	stats, err := GetDirStats(t.TempDir(), time.Now())

	require.NoError(t, err)
	assert.Zero(t, stats)
}

func TestGetDirStats_MissingRootReturnsError(t *testing.T) {
	stats, err := GetDirStats(filepath.Join(t.TempDir(), "missing"), time.Now())

	require.Error(t, err)
	assert.Zero(t, stats)
}

func TestGetDirStats_RegularFileRoot(t *testing.T) {
	cutoff, old := newTestStatsTimes()
	file := newTestStatsFile(t, t.TempDir(), "file", 8, old)

	stats, err := GetDirStats(file, cutoff)

	require.NoError(t, err)
	assert.Equal(t, int64(8), stats.Size)
	assert.Equal(t, int64(1), stats.FileCount)
	assert.True(t, old.Equal(stats.NewestFileTime))
}

func TestGetDirStats_IgnoredRootStillContributesToRecency(t *testing.T) {
	cutoff, _ := newTestStatsTimes()
	file := newTestStatsFile(t, t.TempDir(), ".DS_Store", 8, cutoff)

	stats, err := GetDirStats(file, cutoff, WithIgnoreNames(".DS_Store"))

	require.NoError(t, err)
	assert.Zero(t, stats.Size)
	assert.Zero(t, stats.FileCount)
	assert.True(t, stats.Recent)
}

func TestGetDirStats_DoesNotFollowSymlinkDirectory(t *testing.T) {
	root := t.TempDir()
	destination := t.TempDir()
	cutoff, old := newTestStatsTimes()
	newTestStatsFile(t, destination, "recent", 100, cutoff)
	link := filepath.Join(root, "link")
	require.NoError(t, os.Symlink(destination, link))
	times := []unix.Timespec{unix.NsecToTimespec(old.UnixNano()), unix.NsecToTimespec(old.UnixNano())}
	require.NoError(t, unix.UtimesNanoAt(unix.AT_FDCWD, link, times, unix.AT_SYMLINK_NOFOLLOW))
	info, err := os.Lstat(link)
	require.NoError(t, err)

	stats, err := GetDirStats(root, cutoff)

	require.NoError(t, err)
	assert.Equal(t, info.Size(), stats.Size)
	assert.Equal(t, int64(1), stats.FileCount)
	assert.True(t, old.Equal(stats.NewestFileTime))
	assert.False(t, stats.Recent)
}

func TestGetDirStats_SymlinkRootCountsTheLink(t *testing.T) {
	root := t.TempDir()
	cutoff, _ := newTestStatsTimes()
	file := newTestStatsFile(t, root, "file", 100, cutoff)
	link := filepath.Join(root, "link")
	require.NoError(t, os.Symlink(file, link))
	info, err := os.Lstat(link)
	require.NoError(t, err)

	stats, err := GetDirStats(link, time.Time{})

	require.NoError(t, err)
	assert.Equal(t, info.Size(), stats.Size)
	assert.Equal(t, int64(1), stats.FileCount)
	assert.True(t, info.ModTime().Equal(stats.NewestFileTime))
}

func TestGetDirStats_SequentialMatchesParallel(t *testing.T) {
	root := t.TempDir()
	cutoff, old := newTestStatsTimes()
	newTestStatsFile(t, root, "file", 8, old)
	newTestStatsFile(t, root, "nested/.DS_Store", 100, old.Add(time.Hour))
	previous := runtime.GOMAXPROCS(4)
	defer runtime.GOMAXPROCS(previous)
	parallel, err := GetDirStats(root, cutoff, WithIgnoreNames(".DS_Store"))
	require.NoError(t, err)
	runtime.GOMAXPROCS(1)

	sequential, err := GetDirStats(root, cutoff, WithIgnoreNames(".DS_Store"))

	require.NoError(t, err)
	assert.Equal(t, parallel, sequential)
}

func TestGetDirStats_SequentialStopsAtRecentFile(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	root := t.TempDir()
	cutoff, _ := newTestStatsTimes()
	newTestStatsFile(t, root, "nested/.DS_Store", 100, cutoff)

	stats, err := GetDirStats(root, cutoff, WithIgnoreNames(".DS_Store"))

	require.NoError(t, err)
	assert.True(t, stats.Recent)
}

func TestGetDirStats_ConcurrentScansDoNotShareCancellation(t *testing.T) {
	root := t.TempDir()
	cutoff, old := newTestStatsTimes()
	stale := filepath.Join(root, "stale")
	active := filepath.Join(root, "active")
	for i := range 32 {
		name := filepath.Join(string(rune('a'+i)), "file")
		newTestStatsFile(t, stale, name, 2, old)
		newTestStatsFile(t, active, name, 2, cutoff)
	}
	var staleStats, recentStats DirStats
	var staleErr, recentErr error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() { defer wg.Done(); staleStats, staleErr = GetDirStats(stale, cutoff) }()
	go func() { defer wg.Done(); recentStats, recentErr = GetDirStats(active, cutoff) }()
	wg.Wait()

	require.NoError(t, staleErr)
	require.NoError(t, recentErr)
	assert.Equal(t, int64(64), staleStats.Size)
	assert.Equal(t, int64(32), staleStats.FileCount)
	assert.False(t, staleStats.Recent)
	assert.True(t, recentStats.Recent)
}

func TestGetDirStats_UnreadableChildKeepsReadableTotals(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read directories without permission bits")
	}
	root := t.TempDir()
	cutoff, old := newTestStatsTimes()
	newTestStatsFile(t, root, "visible", 8, old)
	hidden := newTestStatsFile(t, root, "blocked/hidden", 100, old)
	blocked := filepath.Dir(hidden)
	require.NoError(t, os.Chmod(blocked, 0o000))
	t.Cleanup(func() { require.NoError(t, os.Chmod(blocked, 0o755)) })

	stats, err := GetDirStats(root, cutoff)

	require.NoError(t, err)
	assert.Equal(t, int64(8), stats.Size)
	assert.Equal(t, int64(1), stats.FileCount)
	assert.False(t, stats.Recent)
}
