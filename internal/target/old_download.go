package target

import (
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

const defaultDaysOld = 30

// OldDownloadTarget scans for old files in the Downloads folder.
type OldDownloadTarget struct {
	*PathTarget
	daysOld int
}

func NewOldDownloadTarget(cat types.Category, daysOld int) *OldDownloadTarget {
	return &OldDownloadTarget{
		PathTarget: NewPathTarget(cat),
		daysOld:    daysOld,
	}
}

// Scan returns files older than the configured days' threshold.
func (s *OldDownloadTarget) Scan() (*types.ScanResult, error) {
	return s.scanWithAge(ageFilter{defaultDays: s.daysOld})
}

func (s *OldDownloadTarget) scanWithAge(filter ageFilter) (*types.ScanResult, error) {
	// Get all items from PathTarget
	result, err := s.PathTarget.Scan()
	if err != nil {
		return nil, err
	}

	filter.apply(result)

	return result, nil
}

// Clean moves the selected items to trash.
func (s *OldDownloadTarget) Clean(items []types.CleanableItem) (*types.CleanResult, error) {
	result := types.NewCleanResult(s.category)

	if len(items) == 0 {
		return result, nil
	}

	batchResult := utils.BatchTrash(items, types.BatchTrashOptions{
		Category: result.Category,
	})

	result.Merge(batchResult)

	return result, nil
}
