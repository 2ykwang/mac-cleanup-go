package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const trashTimeout = 30 * time.Second

// batchSize is the maximum number of files to process in a single AppleScript call.
const batchSize = 50

// TrashBatchResult holds the result of batch trash operation.
type TrashBatchResult struct {
	Succeeded []string
	Failed    map[string]error
}

// MoveToTrash moves a file or directory to macOS Trash using Finder.
// It is a variable to allow mocking in tests.
var MoveToTrash = moveToTrashImpl

func moveToTrashImpl(path string) error {
	escaped, err := EscapeForAppleScript(path)
	if err != nil {
		return fmt.Errorf("move to trash: invalid path: %w", err)
	}

	script := fmt.Sprintf(`tell application "Finder" to delete POSIX file "%s"`, escaped)
	ctx, cancel := context.WithTimeout(context.Background(), trashTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("move to trash timeout: %s", path)
		}
		return fmt.Errorf("move to trash: %s: %w", path, err)
	}
	return nil
}

// MoveToTrashBatch moves multiple files to Trash in batches.
// It is a variable to allow mocking in tests.
var MoveToTrashBatch = moveToTrashBatchImpl

func moveToTrashBatchImpl(paths []string) TrashBatchResult {
	result := TrashBatchResult{
		Succeeded: make([]string, 0, len(paths)),
		Failed:    make(map[string]error),
	}

	if len(paths) == 0 {
		return result
	}

	// Process in batches
	for i := 0; i < len(paths); i += batchSize {
		end := i + batchSize
		if end > len(paths) {
			end = len(paths)
		}
		batch := paths[i:end]

		if err := executeBatch(batch); err != nil {
			// Batch failed, fallback to individual deletion
			for _, p := range batch {
				if individualErr := MoveToTrash(p); individualErr != nil {
					result.Failed[p] = individualErr
				} else {
					result.Succeeded = append(result.Succeeded, p)
				}
			}
		} else {
			result.Succeeded = append(result.Succeeded, batch...)
		}
	}

	return result
}

// executeBatch executes a single batch of files using AppleScript.
func executeBatch(paths []string) error {
	if len(paths) == 0 {
		return nil
	}

	// Build AppleScript with proper escaping
	var script strings.Builder
	script.Grow(len(paths) * 120) // Pre-allocate ~120 bytes per path

	script.WriteString(`tell application "Finder"`)
	script.WriteString("\n")

	for _, p := range paths {
		escaped, err := EscapeForAppleScript(p)
		if err != nil {
			return fmt.Errorf("invalid path %s: %w", p, err)
		}
		script.WriteString(fmt.Sprintf(`  delete POSIX file "%s"`, escaped))
		script.WriteString("\n")
	}

	script.WriteString("end tell")

	ctx, cancel := context.WithTimeout(context.Background(), trashTimeout)
	defer cancel()

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "osascript", "-e", script.String())
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("timeout after %v", trashTimeout)
		}
		return fmt.Errorf("osascript: %w, stderr: %s", err, stderr.String())
	}
	return nil
}
