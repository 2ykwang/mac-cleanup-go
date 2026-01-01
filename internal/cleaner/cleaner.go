package cleaner

import (
	"fmt"
	"os"
	"os/exec"

	"mac-cleanup-go/pkg/types"
)

type Cleaner struct{}

func New() *Cleaner {
	return &Cleaner{}
}

func (c *Cleaner) Clean(cat types.Category, items []types.CleanableItem, dryRun bool) *types.CleanResult {
	result := &types.CleanResult{
		Category: cat,
		Errors:   make([]string, 0),
	}

	if dryRun {
		for _, item := range items {
			result.FreedSpace += item.Size
			result.CleanedItems++
		}
		return result
	}

	switch cat.Method {
	case types.MethodTrash:
		c.moveToTrash(items, result)
	case types.MethodPermanent:
		c.deletePermanent(items, result)
	case types.MethodCommand:
		c.runCommand(cat, result)
	}

	return result
}

func (c *Cleaner) moveToTrash(items []types.CleanableItem, result *types.CleanResult) {
	for _, item := range items {
		script := fmt.Sprintf(`tell application "Finder" to delete POSIX file "%s"`, item.Path)
		cmd := exec.Command("osascript", "-e", script)
		if err := cmd.Run(); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", item.Name, err))
		} else {
			result.FreedSpace += item.Size
			result.CleanedItems++
		}
	}
}

func (c *Cleaner) deletePermanent(items []types.CleanableItem, result *types.CleanResult) {
	for _, item := range items {
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

	var cmd *exec.Cmd
	if cat.Sudo {
		cmd = exec.Command("sudo", "sh", "-c", cat.Command)
	} else {
		cmd = exec.Command("sh", "-c", cat.Command)
	}

	if err := cmd.Run(); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("command failed: %v", err))
	} else {
		result.CleanedItems = 1
	}
}
