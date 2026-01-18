package utils

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"testing"
	"time"

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

func TestExecuteBatch_EmptyPaths(t *testing.T) {
	err := executeBatch([]string{})
	assert.NoError(t, err)
}

func TestExecuteBatch_InvalidPathWithNewline(t *testing.T) {
	// Path with newline should cause escape error
	err := executeBatch([]string{"/path/with\nnewline"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid path")
}

func TestMoveToTrashBatch_PathWithNewline_FailsGracefully(t *testing.T) {
	original := MoveToTrash
	defer func() { MoveToTrash = original }()

	// Mock MoveToTrash to also fail for newline paths
	MoveToTrash = func(_ string) error {
		return ErrInvalidPath
	}

	// Path with newline should fail in both batch and fallback
	paths := []string{"/path/with\nnewline"}
	result := moveToTrashBatchImpl(paths)

	assert.Empty(t, result.Succeeded)
	assert.Len(t, result.Failed, 1)
}

func TestMoveToTrashImpl_Success_WithMockCommand(t *testing.T) {
	original := execCommandContext
	defer func() { execCommandContext = original }()

	execCommandContext = func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.Command("true") // Always succeeds
	}

	err := moveToTrashImpl("/valid/path")

	assert.NoError(t, err)
}

func TestMoveToTrashImpl_CommandFailure_WithMockCommand(t *testing.T) {
	original := execCommandContext
	defer func() { execCommandContext = original }()

	execCommandContext = func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.Command("false") // Always fails
	}

	err := moveToTrashImpl("/valid/path")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "move to trash:")
}

func TestMoveToTrashImpl_InvalidPath_ReturnsError(t *testing.T) {
	// Path with newline should return escape error
	err := moveToTrashImpl("/path/with\nnewline")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid path")
}

func TestExecuteBatch_Success_WithMockCommand(t *testing.T) {
	original := execCommandContext
	defer func() { execCommandContext = original }()

	execCommandContext = func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.Command("true")
	}

	err := executeBatch([]string{"/path1", "/path2"})

	assert.NoError(t, err)
}

func TestExecuteBatch_CommandFailure_WithMockCommand(t *testing.T) {
	original := execCommandContext
	defer func() { execCommandContext = original }()

	execCommandContext = func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.Command("false")
	}

	err := executeBatch([]string{"/valid/path"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "osascript:")
}

func TestMoveToTrashBatchImpl_BatchSuccess_WithMockCommand(t *testing.T) {
	original := execCommandContext
	defer func() { execCommandContext = original }()

	execCommandContext = func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.Command("true")
	}

	paths := []string{"/path1", "/path2", "/path3"}
	result := moveToTrashBatchImpl(paths)

	assert.Equal(t, paths, result.Succeeded)
	assert.Empty(t, result.Failed)
}

func TestExecuteBatch_Timeout_WithMockCommand(t *testing.T) {
	originalCmd := execCommandContext
	originalTimeout := trashTimeout
	defer func() {
		execCommandContext = originalCmd
		trashTimeout = originalTimeout
	}()

	// Set very short timeout
	trashTimeout = 10 * time.Millisecond

	// Use CommandContext so the command respects context cancellation
	execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sleep", "1")
	}

	err := executeBatch([]string{"/valid/path"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout after")
}
