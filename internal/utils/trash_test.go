package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTrashTestFile(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte("test"), 0o644))
	return path
}

func newTrashTestFiles(t *testing.T, count int) []types.CleanableItem {
	t.Helper()

	dir := t.TempDir()
	items := make([]types.CleanableItem, count)
	for i := range count {
		path := filepath.Join(dir, fmt.Sprintf("item-%03d", i))
		require.NoError(t, os.WriteFile(path, []byte("test"), 0o644))
		items[i] = types.CleanableItem{Path: path, Size: int64(i + 1)}
	}
	return items
}

func TestBatchTrash_EmptyItems(t *testing.T) {
	called := false
	recycle := func(_ []string) ([]string, error) {
		called = true
		return nil, nil
	}

	result := batchTrash(nil, types.BatchTrashOptions{}, recycle)

	assert.False(t, called)
	assert.Equal(t, 0, result.CleanedItems)
	assert.Equal(t, int64(0), result.FreedSpace)
	assert.Empty(t, result.Errors)
}

func TestBatchTrash_RejectsEmptyPath(t *testing.T) {
	items := []types.CleanableItem{{Path: "", Size: 10}}

	result := batchTrash(items, types.BatchTrashOptions{}, nil)

	assert.Equal(t, 0, result.CleanedItems)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "path is empty")
}

func TestBatchTrash_RejectsNULPath(t *testing.T) {
	items := []types.CleanableItem{{Path: "/tmp/invalid\x00path", Size: 10}}

	result := batchTrash(items, types.BatchTrashOptions{}, nil)

	assert.Equal(t, 0, result.CleanedItems)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "NUL byte")
}

func TestBatchTrash_RejectsInvalidUTF8Path(t *testing.T) {
	path := string([]byte{'/', 't', 'm', 'p', '/', 0xff})
	items := []types.CleanableItem{{Path: path, Size: 10}}

	result := batchTrash(items, types.BatchTrashOptions{}, nil)

	assert.Equal(t, 0, result.CleanedItems)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "valid UTF-8")
}

func TestBatchTrash_DeduplicatesPathAndSize(t *testing.T) {
	path := newTrashTestFile(t, "duplicate")
	items := []types.CleanableItem{
		{Path: path, Size: 100},
		{Path: path, Size: 900},
	}
	var calls int
	recycle := func(paths []string) ([]string, error) {
		calls++
		return paths, nil
	}

	result := batchTrash(items, types.BatchTrashOptions{}, recycle)

	assert.Equal(t, 1, calls)
	assert.Equal(t, 1, result.CleanedItems)
	assert.Equal(t, int64(100), result.FreedSpace)
	assert.Empty(t, result.Errors)
}

func TestBatchTrash_TreatsMissingPathAsCleaned(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing")
	called := false
	recycle := func(_ []string) ([]string, error) {
		called = true
		return nil, nil
	}
	items := []types.CleanableItem{{Path: path, Size: 42}}

	result := batchTrash(items, types.BatchTrashOptions{}, recycle)

	assert.False(t, called)
	assert.Equal(t, 1, result.CleanedItems)
	assert.Equal(t, int64(42), result.FreedSpace)
	assert.Empty(t, result.Errors)
}

func TestBatchTrash_LimitsNativeOperationToFiftyPaths(t *testing.T) {
	items := newTrashTestFiles(t, 121)
	batchSizes := make([]int, 0, 3)
	recycle := func(paths []string) ([]string, error) {
		batchSizes = append(batchSizes, len(paths))
		return paths, nil
	}

	result := batchTrash(items, types.BatchTrashOptions{}, recycle)

	assert.Equal(t, []int{50, 50, 21}, batchSizes)
	assert.Equal(t, 121, result.CleanedItems)
	assert.Empty(t, result.Errors)
}

func TestBatchTrash_UsesReportedPathsAndFinalState(t *testing.T) {
	items := newTrashTestFiles(t, 3)
	recycleErr := errors.New("partial recycle")
	recycle := func(paths []string) ([]string, error) {
		require.NoError(t, os.Remove(paths[1]))
		return []string{paths[0]}, recycleErr
	}

	result := batchTrash(items, types.BatchTrashOptions{}, recycle)

	assert.Equal(t, 2, result.CleanedItems)
	assert.Equal(t, items[0].Size+items[1].Size, result.FreedSpace)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], items[2].Path)
	assert.Contains(t, result.Errors[0], recycleErr.Error())
}

func TestBatchTrash_FailsUnreportedExistingPath(t *testing.T) {
	path := newTrashTestFile(t, "unreported")
	items := []types.CleanableItem{{Path: path, Size: 10}}
	recycle := func(_ []string) ([]string, error) {
		return nil, nil
	}

	result := batchTrash(items, types.BatchTrashOptions{}, recycle)

	assert.Equal(t, 0, result.CleanedItems)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "not reported as recycled")
}

func TestBatchTrash_PreservesPathTextAndSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "대상 file\n.txt")
	symlink := filepath.Join(dir, "링크 path\n")
	require.NoError(t, os.WriteFile(target, []byte("test"), 0o644))
	require.NoError(t, os.Symlink(target, symlink))
	items := []types.CleanableItem{
		{Path: target, Size: 4},
		{Path: symlink, Size: 1},
	}
	var received []string
	recycle := func(paths []string) ([]string, error) {
		received = append(received, paths...)
		return paths, nil
	}

	result := batchTrash(items, types.BatchTrashOptions{}, recycle)

	assert.Equal(t, []string{target, symlink}, received)
	assert.Equal(t, 2, result.CleanedItems)
	assert.Equal(t, int64(5), result.FreedSpace)
	assert.Empty(t, result.Errors)
}

func TestBatchTrash_FiltersItem(t *testing.T) {
	path := newTrashTestFile(t, "filtered")
	items := []types.CleanableItem{{Path: path, Size: 10}}
	opts := types.BatchTrashOptions{
		Filter: func(_ types.CleanableItem) bool { return true },
	}

	result := batchTrash(items, opts, nil)

	assert.Equal(t, 0, result.CleanedItems)
	assert.Equal(t, 1, result.SkippedItems)
	assert.Empty(t, result.Errors)
}

func TestBatchTrash_ValidatesItem(t *testing.T) {
	path := newTrashTestFile(t, "rejected")
	items := []types.CleanableItem{{Path: path, Size: 10}}
	opts := types.BatchTrashOptions{
		Validate: func(_ types.CleanableItem) error { return errors.New("outside allowed root") },
	}

	result := batchTrash(items, opts, nil)

	assert.Equal(t, 0, result.CleanedItems)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "outside allowed root")
}

func TestBatchTrash_SerializesNativeOperations(t *testing.T) {
	items := newTrashTestFiles(t, 8)
	var stateMu sync.Mutex
	active := 0
	maximum := 0
	recycle := func(paths []string) ([]string, error) {
		stateMu.Lock()
		active++
		if active > maximum {
			maximum = active
		}
		stateMu.Unlock()

		time.Sleep(5 * time.Millisecond)

		stateMu.Lock()
		active--
		stateMu.Unlock()
		return paths, errors.New("reported operation warning")
	}

	var wait sync.WaitGroup
	for _, item := range items {
		wait.Add(1)
		go func(candidate types.CleanableItem) {
			defer wait.Done()
			result := batchTrash([]types.CleanableItem{candidate}, types.BatchTrashOptions{}, recycle)
			assert.Equal(t, 1, result.CleanedItems)
		}(item)
	}
	wait.Wait()

	assert.Equal(t, 1, maximum)
}

func TestBatchTrash_PublicAPIHandlesMissingPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing")
	items := []types.CleanableItem{{Path: path, Size: 64}}

	result := BatchTrash(items, types.BatchTrashOptions{})

	assert.Equal(t, 1, result.CleanedItems)
	assert.Equal(t, int64(64), result.FreedSpace)
	assert.Empty(t, result.Errors)
}
