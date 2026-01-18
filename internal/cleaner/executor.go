package cleaner

import (
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
	result := &types.CleanResult{
		Category: cat,
		Errors:   make([]string, 0),
	}

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
	batchResult := utils.BatchTrash(items, utils.BatchTrashOptions{
		Category: result.Category,
		Filter: func(item types.CleanableItem) bool {
			return utils.IsSIPProtected(item.Path)
		},
	})

	result.CleanedItems += batchResult.CleanedItems
	result.SkippedItems += batchResult.SkippedItems
	result.FreedSpace += batchResult.FreedSpace
	result.Errors = append(result.Errors, batchResult.Errors...)
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
			result.Errors = append(result.Errors, item.Path+": "+err.Error())
		} else {
			result.FreedSpace += item.Size
			result.CleanedItems++
		}
	}
}
