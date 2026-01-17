package target

import (
	"time"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

const defaultDaysOld = 30

// OldDownloadTarget scans for old files in the Downloads folder.
type OldDownloadTarget struct {
	*PathTarget
	daysOld int
}

func init() {
	RegisterBuiltin("old-downloads", func(cat types.Category, _ []types.Category) Target {
		return NewOldDownloadTarget(cat, defaultDaysOld)
	})
}

func NewOldDownloadTarget(cat types.Category, daysOld int) *OldDownloadTarget {
	return &OldDownloadTarget{
		PathTarget: NewPathTarget(cat),
		daysOld:    daysOld,
	}
}

// Scan returns files older than the configured days' threshold.
func (s *OldDownloadTarget) Scan() (*types.ScanResult, error) {
	// Get all items from PathTarget
	result, err := s.PathTarget.Scan()
	if err != nil {
		return nil, err
	}

	// Filter to only include old files
	cutoff := time.Now().AddDate(0, 0, -s.daysOld)
	filtered := make([]types.CleanableItem, 0)
	var totalSize int64
	var totalFileCount int64

	for _, item := range result.Items {
		if item.ModifiedAt.Before(cutoff) {
			filtered = append(filtered, item)
			totalSize += item.Size
			totalFileCount += item.FileCount
		}
	}

	result.Items = filtered
	result.TotalSize = totalSize
	result.TotalFileCount = totalFileCount

	return result, nil
}

// Clean moves the selected items to trash.
func (s *OldDownloadTarget) Clean(items []types.CleanableItem) (*types.CleanResult, error) {
	result := &types.CleanResult{
		Category: s.category,
		Errors:   make([]string, 0),
	}

	for _, item := range items {
		if err := utils.MoveToTrash(item.Path); err != nil {
			result.Errors = append(result.Errors, err.Error())
			continue
		}
		result.FreedSpace += item.Size
		result.CleanedItems++
	}

	return result, nil
}
