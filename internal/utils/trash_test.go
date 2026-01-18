package utils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	originalCmd := execCommandContext
	originalMoveToTrash := MoveToTrash
	defer func() {
		execCommandContext = originalCmd
		MoveToTrash = originalMoveToTrash
	}()

	// Create actual temp files
	tmpDir := t.TempDir()
	path1 := tmpDir + "/file1.txt"
	path2 := tmpDir + "/file2.txt"
	require.NoError(t, os.WriteFile(path1, []byte("test1"), 0o644))
	require.NoError(t, os.WriteFile(path2, []byte("test2"), 0o644))

	// Make batch fail
	execCommandContext = func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.Command("false")
	}

	// Mock individual MoveToTrash to always succeed
	var calledPaths []string
	MoveToTrash = func(path string) error {
		calledPaths = append(calledPaths, path)
		return nil
	}

	paths := []string{path1, path2}
	result := moveToTrashBatchImpl(paths)

	// Should have fallen back to individual calls since files exist
	assert.Equal(t, paths, calledPaths)
	assert.Equal(t, 2, len(result.Succeeded))
	assert.Empty(t, result.Failed)
}

func TestMoveToTrashBatch_PartialFailure(t *testing.T) {
	originalCmd := execCommandContext
	originalMoveToTrash := MoveToTrash
	defer func() {
		execCommandContext = originalCmd
		MoveToTrash = originalMoveToTrash
	}()

	// Create actual temp files
	tmpDir := t.TempDir()
	successPath := tmpDir + "/success.txt"
	failPath := tmpDir + "/fail.txt"
	require.NoError(t, os.WriteFile(successPath, []byte("success"), 0o644))
	require.NoError(t, os.WriteFile(failPath, []byte("fail"), 0o644))

	// Make batch fail
	execCommandContext = func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.Command("false")
	}

	// Mock individual MoveToTrash to fail for specific path
	MoveToTrash = func(path string) error {
		if path == failPath {
			return fmt.Errorf("mock error: %s", path)
		}
		return nil
	}

	paths := []string{successPath, failPath}
	result := moveToTrashBatchImpl(paths)

	assert.Contains(t, result.Succeeded, successPath)
	assert.Contains(t, result.Failed, failPath)
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

func TestMoveToTrashBatch_PathWithNewline_FailsInBatch(t *testing.T) {
	originalCmd := execCommandContext
	originalMoveToTrash := MoveToTrash
	defer func() {
		execCommandContext = originalCmd
		MoveToTrash = originalMoveToTrash
	}()

	// Create a temp file with a normal name
	tmpDir := t.TempDir()
	normalPath := tmpDir + "/normal.txt"
	require.NoError(t, os.WriteFile(normalPath, []byte("test"), 0o644))

	// Mock MoveToTrash to fail for newline paths
	MoveToTrash = func(path string) error {
		if path == normalPath {
			return nil
		}
		return ErrInvalidPath
	}

	// Path with newline fails escape validation in executeBatch
	// Normal file still exists so it falls back to individual deletion
	paths := []string{normalPath}
	result := moveToTrashBatchImpl(append([]string{"/path/with\nnewline"}, paths...))

	// The newline path doesn't exist, so it's treated as "already deleted"
	// The normal path is deleted via fallback
	assert.Len(t, result.Succeeded, 2)
	assert.Empty(t, result.Failed)
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

func TestMoveToTrashBatchImpl_PartialBatchSuccess_FileAlreadyDeleted(t *testing.T) {
	originalCmd := execCommandContext
	originalMoveToTrash := MoveToTrash
	defer func() {
		execCommandContext = originalCmd
		MoveToTrash = originalMoveToTrash
	}()

	// Create a temp file that exists
	tmpDir := t.TempDir()
	existingFile := tmpDir + "/existing.txt"
	if err := writeTestFile(existingFile); err != nil {
		t.Fatal(err)
	}

	// Simulate batch failure (e.g., one file in batch caused error)
	execCommandContext = func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.Command("false") // Always fails
	}

	// MoveToTrash succeeds for existing file
	MoveToTrash = func(_ string) error {
		return nil
	}

	// nonexistent path simulates file already deleted by batch
	paths := []string{"/nonexistent/already/deleted", existingFile}
	result := moveToTrashBatchImpl(paths)

	// Both should be in Succeeded:
	// - /nonexistent/already/deleted: os.Stat returns NotExist → treated as already deleted
	// - existingFile: os.Stat returns nil → MoveToTrash called → succeeds
	assert.Len(t, result.Succeeded, 2)
	assert.Empty(t, result.Failed)
}

func writeTestFile(path string) error {
	return os.WriteFile(path, []byte("test"), 0o644)
}
