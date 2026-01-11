package cleaner

import (
	"fmt"
	"os"

	"github.com/2ykwang/mac-cleanup-go/internal/scanner"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

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

func (c *Cleaner) moveToTrash(items []types.CleanableItem, result *types.CleanResult) {
	for _, item := range items {
		if utils.IsSIPProtected(item.Path) {
			result.SkippedItems++
			continue
		}

		if err := utils.MoveToTrash(item.Path); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", item.Name, err))
		} else {
			result.FreedSpace += item.Size
			result.CleanedItems++
		}
	}
}

func (c *Cleaner) removePermanent(items []types.CleanableItem, result *types.CleanResult) {
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
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", item.Name, err))
		} else {
			result.FreedSpace += item.Size
			result.CleanedItems++
		}
	}
}
