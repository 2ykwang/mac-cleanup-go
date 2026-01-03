package cleaner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"mac-cleanup-go/internal/scanner"
	"mac-cleanup-go/pkg/types"
)

const commandTimeout = 10 * time.Second

type Cleaner struct {
	registry *scanner.Registry
}

func New(registry *scanner.Registry) *Cleaner {
	return &Cleaner{registry: registry}
}

func (c *Cleaner) Clean(cat types.Category, items []types.CleanableItem) *types.CleanResult {
	result := &types.CleanResult{
		Category: cat,
		Errors:   make([]string, 0),
	}

	switch cat.Method {
	case types.MethodTrash:
		c.moveToTrash(items, result)
	case types.MethodPermanent:
		c.removePermanent(items, result)
	case types.MethodCommand:
		c.runCommand(cat, result)
	case types.MethodBuiltin:
		if s, ok := c.registry.Get(cat.ID); ok {
			builtinResult, _ := s.Clean(items)
			if builtinResult != nil {
				return builtinResult
			}
		} else {
			result.Errors = append(result.Errors, "scanner not found: "+cat.ID)
		}
	case types.MethodManual:
		// Manual methods require user action - skip all items
		result.SkippedItems = len(items)
	}

	return result
}

func (c *Cleaner) moveToTrash(items []types.CleanableItem, result *types.CleanResult) {
	for _, item := range items {
		// Skip SIP protected paths
		if scanner.IsSIPProtected(item.Path) {
			result.SkippedItems++
			continue
		}

		script := fmt.Sprintf(`tell application "Finder" to delete POSIX file "%s"`, item.Path)
		ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
		cmd := exec.CommandContext(ctx, "osascript", "-e", script)
		err := cmd.Run()
		cancel()

		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: timeout", item.Name))
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", item.Name, err))
			}
		} else {
			result.FreedSpace += item.Size
			result.CleanedItems++
		}
	}
}

func (c *Cleaner) removePermanent(items []types.CleanableItem, result *types.CleanResult) {
	for _, item := range items {
		// Skip SIP protected paths
		if scanner.IsSIPProtected(item.Path) {
			result.SkippedItems++
			continue
		}

		var err error
		if item.IsDirectory {
			err = os.RemoveAll(item.Path)
		} else {
			err = os.Remove(item.Path)
		}

		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", item.Name, err))
		} else {
			result.FreedSpace += item.Size
			result.CleanedItems++
		}
	}
}

func (c *Cleaner) runCommand(cat types.Category, result *types.CleanResult) {
	if cat.Command == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if cat.Sudo {
		cmd = exec.CommandContext(ctx, "sudo", "sh", "-c", cat.Command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", cat.Command)
	}

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.Errors = append(result.Errors, "command timeout")
		} else {
			result.Errors = append(result.Errors, fmt.Sprintf("command failed: %v", err))
		}
	} else {
		result.CleanedItems = 1
	}
}
