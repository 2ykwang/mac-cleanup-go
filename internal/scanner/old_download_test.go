package scanner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/pkg/types"
)

func TestOldDownloadScanner_FiltersOldFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create old file (40 days ago)
	oldFile := filepath.Join(tmpDir, "old_file.txt")
	require.NoError(t, os.WriteFile(oldFile, []byte("old content"), 0o644))
	oldTime := time.Now().AddDate(0, 0, -40)
	require.NoError(t, os.Chtimes(oldFile, oldTime, oldTime))

	// Create recent file (10 days ago)
	recentFile := filepath.Join(tmpDir, "recent_file.txt")
	require.NoError(t, os.WriteFile(recentFile, []byte("recent content"), 0o644))
	recentTime := time.Now().AddDate(0, 0, -10)
	require.NoError(t, os.Chtimes(recentFile, recentTime, recentTime))

	cat := types.Category{
		ID:     "old-downloads",
		Name:   "Old Downloads",
		Group:  "system",
		Safety: types.SafetyLevelModerate,
		Method: types.MethodBuiltin,
		Paths:  []string{filepath.Join(tmpDir, "*")},
	}

	scanner := NewOldDownloadScanner(cat, 30)

	result, err := scanner.Scan()

	require.NoError(t, err)
	assert.Len(t, result.Items, 1, "should only include files older than 30 days")
	assert.Equal(t, "old_file.txt", result.Items[0].Name)
}

func TestOldDownloadScanner_NoOldFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only recent file
	recentFile := filepath.Join(tmpDir, "recent.txt")
	require.NoError(t, os.WriteFile(recentFile, []byte("recent"), 0o644))

	cat := types.Category{
		ID:     "old-downloads",
		Name:   "Old Downloads",
		Method: types.MethodBuiltin,
		Paths:  []string{filepath.Join(tmpDir, "*")},
	}

	scanner := NewOldDownloadScanner(cat, 30)

	result, err := scanner.Scan()

	require.NoError(t, err)
	assert.Empty(t, result.Items, "should not include recent files")
}

func TestOldDownloadScanner_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	cat := types.Category{
		ID:     "old-downloads",
		Name:   "Old Downloads",
		Method: types.MethodBuiltin,
		Paths:  []string{filepath.Join(tmpDir, "*")},
	}

	scanner := NewOldDownloadScanner(cat, 30)

	result, err := scanner.Scan()

	require.NoError(t, err)
	assert.Empty(t, result.Items)
}

func TestOldDownloadScanner_ExactlyAtCutoff(t *testing.T) {
	tmpDir := t.TempDir()

	// Create file slightly newer than 30 days (29 days, 23 hours ago)
	// This should NOT be included - must be strictly OLDER than cutoff
	exactFile := filepath.Join(tmpDir, "exact.txt")
	require.NoError(t, os.WriteFile(exactFile, []byte("exact"), 0o644))
	// 29 days and 23 hours ago - just inside the 30-day window
	almostOldTime := time.Now().Add(-29*24*time.Hour - 23*time.Hour)
	require.NoError(t, os.Chtimes(exactFile, almostOldTime, almostOldTime))

	cat := types.Category{
		ID:     "old-downloads",
		Name:   "Old Downloads",
		Method: types.MethodBuiltin,
		Paths:  []string{filepath.Join(tmpDir, "*")},
	}

	scanner := NewOldDownloadScanner(cat, 30)

	result, err := scanner.Scan()

	require.NoError(t, err)
	// File newer than cutoff should not be included
	assert.Empty(t, result.Items, "files newer than 30 days should not be included")
}

func TestOldDownloadScanner_Category(t *testing.T) {
	cat := types.Category{
		ID:   "old-downloads",
		Name: "Old Downloads",
	}

	scanner := NewOldDownloadScanner(cat, 30)

	assert.Equal(t, "old-downloads", scanner.Category().ID)
}

func TestOldDownloadScanner_IsAvailable(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file so glob matches
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0o644))

	cat := types.Category{
		ID:    "old-downloads",
		Paths: []string{tmpDir + "/*"},
	}

	scanner := NewOldDownloadScanner(cat, 30)

	assert.True(t, scanner.IsAvailable())
}

func TestOldDownloadScanner_Clean_MovesToTrash(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "to_delete.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("delete me"), 0o644))

	cat := types.Category{
		ID:   "old-downloads",
		Name: "Old Downloads",
	}

	scanner := NewOldDownloadScanner(cat, 30)
	items := []types.CleanableItem{
		{Path: testFile, Size: 9, Name: "to_delete.txt"},
	}

	result, err := scanner.Clean(items)

	require.NoError(t, err)
	assert.Equal(t, 1, result.CleanedItems)
	assert.Equal(t, int64(9), result.FreedSpace)
	assert.Empty(t, result.Errors)

	// File should no longer exist at original path
	_, err = os.Stat(testFile)
	assert.True(t, os.IsNotExist(err))
}

func TestOldDownloadScanner_Clean_NonexistentFile(t *testing.T) {
	cat := types.Category{
		ID:   "old-downloads",
		Name: "Old Downloads",
	}

	scanner := NewOldDownloadScanner(cat, 30)
	items := []types.CleanableItem{
		{Path: "/nonexistent/file.txt", Size: 100, Name: "file.txt"},
	}

	result, err := scanner.Clean(items)

	require.NoError(t, err)
	assert.Equal(t, 0, result.CleanedItems)
	assert.NotEmpty(t, result.Errors)
}
