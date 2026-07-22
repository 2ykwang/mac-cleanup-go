//go:build darwin && cgo

package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchTrash_RecyclesTenFilesInOneNativeOperation(t *testing.T) {
	if os.Getenv("MAC_CLEANUP_TRASH_INTEGRATION") != "1" {
		t.Skip("set MAC_CLEANUP_TRASH_INTEGRATION=1 to run the NSWorkspace integration test")
	}

	dir := t.TempDir()
	prefix := fmt.Sprintf("mac-cleanup-integration-%d", time.Now().UnixNano())
	items := make([]types.CleanableItem, 10)
	for i := range 10 {
		name := fmt.Sprintf("%s-%02d", prefix, i)
		if i == 0 {
			name = prefix + " 한글\nfile"
		}
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, []byte("test"), 0o644))
		items[i] = types.CleanableItem{Path: path, Size: 4}
	}

	trashDir := filepath.Join(ExpandPath("~"), ".Trash")
	t.Cleanup(func() {
		for _, item := range items {
			// These files are created solely by this test.
			_ = os.Remove(filepath.Join(trashDir, filepath.Base(item.Path)))
		}
	})

	var calls int
	recycle := func(paths []string) ([]string, error) {
		calls++
		return recyclePathsPlatform(paths)
	}

	result := batchTrash(items, types.BatchTrashOptions{}, recycle)

	assert.Equal(t, 1, calls)
	assert.Equal(t, 10, result.CleanedItems)
	assert.Equal(t, int64(40), result.FreedSpace)
	assert.Empty(t, result.Errors)
	for _, item := range items {
		_, err := os.Lstat(item.Path)
		assert.True(t, os.IsNotExist(err), item.Path)
	}
}
