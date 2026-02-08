package cleaner

import (
	"os"

	"github.com/2ykwang/mac-cleanup-go/internal/logger"
	"github.com/2ykwang/mac-cleanup-go/internal/target"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

type Executor struct {
	registry *target.Registry
}

func NewExecutor(registry *target.Registry) *Executor {
	return &Executor{registry: registry}
}

func (c *Executor) Trash(cat types.Category, items []types.CleanableItem) *types.CleanResult {
	result := types.NewCleanResult(cat)
	if !c.ensureMethod(cat, types.MethodTrash, result, items) {
		return result
	}

	c.moveToTrash(items, result)
	return result
}

func (c *Executor) Permanent(cat types.Category, items []types.CleanableItem) *types.CleanResult {
	result := types.NewCleanResult(cat)
	if !c.ensureMethod(cat, types.MethodPermanent, result, items) {
		return result
	}

	c.removePermanent(items, result)
	return result
}

func (c *Executor) Builtin(cat types.Category, items []types.CleanableItem) *types.CleanResult {
	result := types.NewCleanResult(cat)
	if !c.ensureMethod(cat, types.MethodBuiltin, result, items) {
		return result
	}

	t, ok := c.registry.Get(cat.ID)
	if !ok {
		result.Errors = append(result.Errors, "target not found: "+cat.ID)
		return result
	}

	cleaner, ok := t.(target.BuiltinCleaner)
	if !ok {
		result.Errors = append(result.Errors, "target does not support builtin clean: "+cat.ID)
		return result
	}

	builtinResult, err := cleaner.Clean(items)
	if err != nil {
		if builtinResult != nil {
			builtinResult.Errors = append(builtinResult.Errors, err.Error())
		} else {
			result.Errors = append(result.Errors, err.Error())
		}
	}
	if builtinResult != nil {
		return builtinResult
	}

	return result
}

func (c *Executor) Manual(cat types.Category, items []types.CleanableItem) *types.CleanResult {
	result := types.NewCleanResult(cat)
	if !c.ensureMethod(cat, types.MethodManual, result, items) {
		return result
	}

	// Manual methods require user action - skip all items
	result.SkippedItems = len(items)
	return result
}

func (c *Executor) ensureMethod(cat types.Category, expected types.CleanupMethod, result *types.CleanResult, items []types.CleanableItem) bool {
	if cat.Method == expected {
		return true
	}

	result.Errors = append(result.Errors, "method mismatch: expected "+string(expected)+" got "+string(cat.Method))
	result.SkippedItems += len(items)
	return false
}

func (c *Executor) moveToTrash(items []types.CleanableItem, result *types.CleanResult) {
	batchResult := utils.BatchTrash(items, types.BatchTrashOptions{
		Category: result.Category,
		Filter: func(item types.CleanableItem) bool {
			return utils.IsSIPProtected(item.Path)
		},
	})

	result.Merge(batchResult)
}

func (c *Executor) removePermanent(items []types.CleanableItem, result *types.CleanResult) {
	sipSkipped := 0

	for _, item := range items {
		// Skip SIP protected paths
		if utils.IsSIPProtected(item.Path) {
			sipSkipped++
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
			logger.Debug("permanent delete failed", "path", item.Path, "error", err)
			result.Errors = append(result.Errors, item.Path+": "+err.Error())
		} else {
			result.FreedSpace += item.Size
			result.CleanedItems++
		}
	}

	logger.Info("removePermanent completed",
		"total", len(items),
		"cleaned", result.CleanedItems,
		"sipSkipped", sipSkipped,
		"failed", len(result.Errors))
}
