package utils

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMoveToTrash_CanBeMocked(t *testing.T) {
	original := MoveToTrash
	defer func() { MoveToTrash = original }()

	var calledPath string
	MoveToTrash = func(path string) error {
		calledPath = path
		return nil
	}

	err := MoveToTrash("/test/path")

	assert.NoError(t, err)
	assert.Equal(t, "/test/path", calledPath)
}

func TestMoveToTrash_MockError(t *testing.T) {
	original := MoveToTrash
	defer func() { MoveToTrash = original }()

	MoveToTrash = func(path string) error {
		return fmt.Errorf("mock error: %s", path)
	}

	err := MoveToTrash("/test/path")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock error")
}

func TestMoveToTrashImpl_SkipOnNonMacOS(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping test on non-macOS")
	}
	// Real implementation test would go here
	// but we don't test actual osascript calls to avoid side effects
}

func TestMoveToTrashBatch_CanBeMocked(t *testing.T) {
	original := MoveToTrashBatch
	defer func() { MoveToTrashBatch = original }()

	var calledPaths []string
	MoveToTrashBatch = func(paths []string) TrashBatchResult {
		calledPaths = paths
		return TrashBatchResult{
			Succeeded: paths,
			Failed:    make(map[string]error),
		}
	}

	result := MoveToTrashBatch([]string{"/test/path1", "/test/path2"})

	assert.Equal(t, []string{"/test/path1", "/test/path2"}, calledPaths)
	assert.Equal(t, 2, len(result.Succeeded))
	assert.Empty(t, result.Failed)
}

func TestMoveToTrashBatch_EmptyPaths(t *testing.T) {
	original := MoveToTrashBatch
	defer func() { MoveToTrashBatch = original }()

	// Use original implementation for this test
	MoveToTrashBatch = moveToTrashBatchImpl

	result := MoveToTrashBatch([]string{})

	assert.Empty(t, result.Succeeded)
	assert.Empty(t, result.Failed)
}

func TestMoveToTrashBatch_FallbackOnBatchFailure(t *testing.T) {
	original := MoveToTrash
	defer func() { MoveToTrash = original }()

	// Mock individual MoveToTrash to always succeed
	var calledPaths []string
	MoveToTrash = func(path string) error {
		calledPaths = append(calledPaths, path)
		return nil
	}

	// Test with non-existent paths which should fail in executeBatch
	// and trigger fallback to individual deletion
	paths := []string{"/non/existent/path1", "/non/existent/path2"}

	// Call the real implementation
	result := moveToTrashBatchImpl(paths)

	// Should have fallen back to individual calls
	assert.Equal(t, paths, calledPaths)
	assert.Equal(t, 2, len(result.Succeeded))
	assert.Empty(t, result.Failed)
}

func TestMoveToTrashBatch_PartialFailure(t *testing.T) {
	original := MoveToTrash
	defer func() { MoveToTrash = original }()

	// Mock individual MoveToTrash to fail for specific path
	MoveToTrash = func(path string) error {
		if path == "/fail/this/path" {
			return fmt.Errorf("mock error: %s", path)
		}
		return nil
	}

	paths := []string{"/success/path", "/fail/this/path"}
	result := moveToTrashBatchImpl(paths)

	assert.Contains(t, result.Succeeded, "/success/path")
	assert.Contains(t, result.Failed, "/fail/this/path")
}
