//go:build !darwin || !cgo

package utils

import (
	"os"
	"testing"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchTrash_ReturnsUnsupportedPlatformError(t *testing.T) {
	path := newTrashTestFile(t, "existing")
	items := []types.CleanableItem{{Path: path, Size: 10}}

	result := BatchTrash(items, types.BatchTrashOptions{})

	assert.Equal(t, 0, result.CleanedItems)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "requires macOS with cgo enabled")
	_, err := os.Lstat(path)
	assert.NoError(t, err)
}
