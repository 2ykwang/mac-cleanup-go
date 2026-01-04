package utils

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandPath_TildePath(t *testing.T) {
	home, _ := os.UserHomeDir()

	result := ExpandPath("~/test")

	assert.Equal(t, filepath.Join(home, "test"), result)
}

func TestExpandPath_TildeOnly(t *testing.T) {
	home, _ := os.UserHomeDir()

	result := ExpandPath("~/")

	assert.Equal(t, filepath.Join(home, ""), result)
}

func TestExpandPath_AbsolutePath(t *testing.T) {
	result := ExpandPath("/absolute/path")

	assert.Equal(t, "/absolute/path", result)
}

func TestExpandPath_RelativePath(t *testing.T) {
	result := ExpandPath("relative/path")

	assert.Equal(t, "relative/path", result)
}

func TestFormatSize_Zero(t *testing.T) {
	assert.Equal(t, "0 B", FormatSize(0))
}

func TestFormatSize_Bytes(t *testing.T) {
	assert.Equal(t, "512 B", FormatSize(512))
}

func TestFormatSize_KB(t *testing.T) {
	assert.Equal(t, "1.0 KB", FormatSize(1024))
	assert.Equal(t, "1.5 KB", FormatSize(1536))
}

func TestFormatSize_MB(t *testing.T) {
	assert.Equal(t, "1.0 MB", FormatSize(1048576))
	assert.Equal(t, "1.5 MB", FormatSize(1572864))
}

func TestFormatSize_GB(t *testing.T) {
	assert.Equal(t, "1.0 GB", FormatSize(1073741824))
	assert.Equal(t, "1.5 GB", FormatSize(1610612736))
}

func TestFormatSize_TB(t *testing.T) {
	assert.Equal(t, "1.0 TB", FormatSize(1099511627776))
}

func TestPathExists(t *testing.T) {
	assert.True(t, PathExists("/tmp"), "existing path should return true")
	assert.False(t, PathExists("/nonexistent/path/12345"), "non-existing path should return false")
	assert.True(t, PathExists("~/"), "home directory should exist")
}

func TestCommandExists(t *testing.T) {
	assert.True(t, CommandExists("ls"), "common command should exist")
	assert.False(t, CommandExists("nonexistentcommand12345"), "non-existing command should return false")
}

func TestGetDirSize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-dir-size")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	require.NoError(t, os.WriteFile(file1, make([]byte, 100), 0o644))
	require.NoError(t, os.WriteFile(file2, make([]byte, 200), 0o644))

	size, err := GetDirSize(tmpDir)
	assert.NoError(t, err)
	assert.Equal(t, int64(300), size)
}

func TestGetDirSizeWithCount(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-dir-size-count")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create files in root
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	require.NoError(t, os.WriteFile(file1, make([]byte, 100), 0o644))
	require.NoError(t, os.WriteFile(file2, make([]byte, 200), 0o644))

	// Create subdirectory with files
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0o755))
	file3 := filepath.Join(subDir, "file3.txt")
	require.NoError(t, os.WriteFile(file3, make([]byte, 50), 0o644))

	size, count, err := GetDirSizeWithCount(tmpDir)
	assert.NoError(t, err)
	assert.Equal(t, int64(350), size)
	assert.Equal(t, int64(3), count)
}

func TestGetFileSize(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-file-size")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := make([]byte, 1024)
	_, err = tmpFile.Write(content)
	require.NoError(t, err)
	tmpFile.Close()

	size, err := GetFileSize(tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, int64(1024), size)
}

func TestGlobPaths(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-glob")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	for _, name := range []string{"a.txt", "b.txt", "c.log"} {
		f, _ := os.Create(filepath.Join(tmpDir, name))
		f.Close()
	}

	pattern := filepath.Join(tmpDir, "*.txt")
	paths, err := GlobPaths(pattern)
	assert.NoError(t, err)
	assert.Len(t, paths, 2)
}

// Note: Testing with existing paths would actually open Finder windows,
// which is not suitable for automated tests. Manual verification required.

func TestOpenInFinder_NonExistentPath(t *testing.T) {
	err := OpenInFinder("/nonexistent/path/12345")

	assert.Error(t, err, "non-existent path should return error")
}

func TestOpenInFinder_TildeExpansion(t *testing.T) {
	// Test that tilde path that doesn't exist returns error
	err := OpenInFinder("~/nonexistent/path/12345")

	assert.Error(t, err, "non-existent home path should return error")
}

func TestFormatAge_ZeroTime(t *testing.T) {
	result := FormatAge(time.Time{})

	assert.Equal(t, "-", result)
}

func TestFormatAge_LessThanMinute(t *testing.T) {
	now := time.Now()

	result := FormatAge(now.Add(-30 * time.Second))

	assert.Equal(t, "<1m", result)
}

func TestFormatAge_Minutes(t *testing.T) {
	now := time.Now()

	result := FormatAge(now.Add(-5 * time.Minute))

	assert.Equal(t, "5m", result)
}

func TestFormatAge_Hours(t *testing.T) {
	now := time.Now()

	result := FormatAge(now.Add(-3 * time.Hour))

	assert.Equal(t, "3h", result)
}

func TestFormatAge_Days(t *testing.T) {
	now := time.Now()

	result := FormatAge(now.Add(-7 * 24 * time.Hour))

	assert.Equal(t, "7d", result)
}

func TestFormatAge_Months(t *testing.T) {
	now := time.Now()

	result := FormatAge(now.Add(-60 * 24 * time.Hour)) // ~2 months

	assert.Equal(t, "2mo", result)
}

func TestFormatAge_Years(t *testing.T) {
	now := time.Now()

	result := FormatAge(now.Add(-400 * 24 * time.Hour)) // ~1 year

	assert.Equal(t, "1y", result)
}

func TestStripGlobPattern_ExistingPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-strip-glob")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	result := StripGlobPattern(tmpDir)

	assert.Equal(t, tmpDir, result)
}

func TestStripGlobPattern_GlobInPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-strip-glob")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pattern := filepath.Join(tmpDir, "*.txt")

	result := StripGlobPattern(pattern)

	assert.Equal(t, tmpDir, result)
}

func TestStripGlobPattern_NestedGlob(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-strip-glob")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0o755))

	pattern := filepath.Join(subDir, "*", "*.log")

	result := StripGlobPattern(pattern)

	assert.Equal(t, subDir, result)
}

func TestStripGlobPattern_NonExistentPath(t *testing.T) {
	pattern := "/nonexistent/path/*.txt"

	result := StripGlobPattern(pattern)

	assert.Equal(t, "/nonexistent/path", result)
}
