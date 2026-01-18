package cleaner

import (
	"fmt"
	"os"

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

func (c *Executor) Clean(cat types.Category, items []types.CleanableItem) *types.CleanResult {
	result := types.NewCleanResult(cat)

	switch cat.Method {
	case types.MethodTrash:
		c.moveToTrash(items, result)
	case types.MethodPermanent:
		c.removePermanent(items, result)
	case types.MethodBuiltin:
		if s, ok := c.registry.Get(cat.ID); ok {
			builtinResult, err := s.Clean(items)
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
		} else {
			result.Errors = append(result.Errors, "scanner not found: "+cat.ID)
		}
	case types.MethodManual:
		// Manual methods require user action - skip all items
		result.SkippedItems = len(items)
	}

	return result
}

func (c *Executor) moveToTrash(items []types.CleanableItem, result *types.CleanResult) {
	// SIP filtering and path collection
	paths := make([]string, 0, len(items))
	pathToItem := make(map[string]types.CleanableItem, len(items))

	for _, item := range items {
		if utils.IsSIPProtected(item.Path) {
			result.SkippedItems++
			continue
		}
		paths = append(paths, item.Path)
		pathToItem[item.Path] = item
	}

	if len(paths) == 0 {
		return
	}

	// Batch delete
	batchResult := utils.MoveToTrashBatch(paths)

	// Process succeeded items
	for _, p := range batchResult.Succeeded {
		item := pathToItem[p]
		result.FreedSpace += item.Size
		result.CleanedItems++
	}

	// Process failed items
	for p, err := range batchResult.Failed {
		item := pathToItem[p]
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", item.Path, err))
	}
}

func (c *Executor) removePermanent(items []types.CleanableItem, result *types.CleanResult) {
	for _, item := range items {
		// Skip SIP protected paths
		if utils.IsSIPProtected(item.Path) {
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
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", item.Path, err))
		} else {
			result.FreedSpace += item.Size
			result.CleanedItems++
		}
	}
}
