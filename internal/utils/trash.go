package utils

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

const trashTimeout = 30 * time.Second

// MoveToTrash moves a file or directory to macOS Trash using Finder.
// It is a variable to allow mocking in tests.
var MoveToTrash = moveToTrashImpl

func moveToTrashImpl(path string) error {
	script := fmt.Sprintf(`tell application "Finder" to delete POSIX file "%s"`, path)
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
