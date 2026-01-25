package cleaner

import (
	"github.com/2ykwang/mac-cleanup-go/internal/logger"
	"github.com/2ykwang/mac-cleanup-go/internal/target"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

// CleanJob represents a cleaning job for a category.
type CleanJob struct {
	Category types.Category
	Items    []types.CleanableItem
}

// CleanService orchestrates the cleaning process.
type CleanService struct {
	registry *target.Registry
	executor *Executor
}

// NewCleanService creates a new CleanService.
func NewCleanService(registry *target.Registry) *CleanService {
	return &CleanService{
		registry: registry,
		executor: NewExecutor(registry),
	}
}

// Clean executes the cleaning jobs and reports progress via callbacks.
func (s *CleanService) Clean(jobs []CleanJob, callbacks types.CleanCallbacks) *types.Report {
	report := &types.Report{Results: make([]types.CleanResult, 0)}

	totalItems := 0
	for _, job := range jobs {
		totalItems += len(job.Items)
	}

	logger.Info("clean started", "jobs", len(jobs), "totalItems", totalItems)

	currentItem := 0

	for _, job := range jobs {
		logger.Debug("processing job", "category", job.Category.Name, "method", job.Category.Method, "items", len(job.Items))

		var result *types.CleanResult

		switch job.Category.Method {
		case types.MethodBuiltin:
			result = s.cleanBuiltin(job, callbacks, &currentItem, totalItems)
		case types.MethodTrash:
			result = s.cleanTrashBatch(job, callbacks, &currentItem, totalItems)
		default:
			result = s.cleanItemByItem(job, callbacks, &currentItem, totalItems)
		}

		if result != nil {
			report.Results = append(report.Results, *result)
			report.FreedSpace += result.FreedSpace
			report.CleanedItems += result.CleanedItems
			report.FailedItems += len(result.Errors)

			if callbacks.OnCategoryDone != nil {
				callbacks.OnCategoryDone(types.CategoryCleanedResult{
					CategoryName: job.Category.Name,
					FreedSpace:   result.FreedSpace,
					CleanedItems: result.CleanedItems,
					ErrorCount:   len(result.Errors),
				})
			}
		}
	}

	logger.Info("clean completed",
		"freedSpace", report.FreedSpace,
		"cleanedItems", report.CleanedItems,
		"failedItems", report.FailedItems)

	return report
}

// cleanBuiltin handles builtin methods (docker, brew) with category-level progress.
func (s *CleanService) cleanBuiltin(job CleanJob, callbacks types.CleanCallbacks, currentItem *int, totalItems int) *types.CleanResult {
	if callbacks.OnProgress != nil {
		callbacks.OnProgress(types.CleanProgress{
			CategoryName: job.Category.Name,
			CurrentItem:  "",
			Current:      *currentItem,
			Total:        totalItems,
		})
	}

	result := s.executor.Clean(job.Category, job.Items)
	*currentItem += len(job.Items)
	return result
}

// cleanTrashBatch handles trash method with batch processing for performance.
func (s *CleanService) cleanTrashBatch(job CleanJob, callbacks types.CleanCallbacks, currentItem *int, totalItems int) *types.CleanResult {
	result := types.NewCleanResult(job.Category)

	items := job.Items
	for i := 0; i < len(items); i += utils.TrashBatchSize {
		end := min(i+utils.TrashBatchSize, len(items))
		batch := items[i:end]

		if callbacks.OnProgress != nil {
			callbacks.OnProgress(types.CleanProgress{
				CategoryName: job.Category.Name,
				CurrentItem:  batch[0].Name,
				Current:      *currentItem,
				Total:        totalItems,
			})
		}

		batchResult := s.executor.Clean(job.Category, batch)
		result.FreedSpace += batchResult.FreedSpace
		result.CleanedItems += batchResult.CleanedItems
		result.Errors = append(result.Errors, batchResult.Errors...)

		s.sendBatchItemCallbacks(batch, batchResult, callbacks)
		*currentItem += len(batch)

		// Send progress after batch completion to update UI
		if callbacks.OnProgress != nil {
			callbacks.OnProgress(types.CleanProgress{
				CategoryName: job.Category.Name,
				CurrentItem:  batch[len(batch)-1].Name,
				Current:      *currentItem,
				Total:        totalItems,
			})
		}
	}

	return result
}

// sendBatchItemCallbacks sends OnItemDone callbacks for batch items with error tracking.
func (s *CleanService) sendBatchItemCallbacks(batch []types.CleanableItem, batchResult *types.CleanResult, callbacks types.CleanCallbacks) {
	if callbacks.OnItemDone == nil {
		return
	}

	// Build error map from batch result errors (format: "itemPath: error")
	errorMap := make(map[string]string)
	for _, errStr := range batchResult.Errors {
		for _, item := range batch {
			prefix := item.Path + ": "
			if len(errStr) >= len(prefix) && errStr[:len(prefix)] == prefix {
				errorMap[item.Path] = errStr[len(prefix):]
				break
			}
		}
	}

	for _, item := range batch {
		errMsg, hasFailed := errorMap[item.Path]
		callbacks.OnItemDone(types.ItemCleanedResult{
			Path:    item.Path,
			Name:    item.Name,
			Size:    item.Size,
			Success: !hasFailed,
			ErrMsg:  errMsg,
		})
	}
}

// cleanItemByItem handles other methods with item-by-item processing.
func (s *CleanService) cleanItemByItem(job CleanJob, callbacks types.CleanCallbacks, currentItem *int, totalItems int) *types.CleanResult {
	result := types.NewCleanResult(job.Category)

	for _, item := range job.Items {
		*currentItem++

		if callbacks.OnProgress != nil {
			callbacks.OnProgress(types.CleanProgress{
				CategoryName: job.Category.Name,
				CurrentItem:  item.Name,
				Current:      *currentItem,
				Total:        totalItems,
			})
		}

		singleResult := s.executor.Clean(job.Category, []types.CleanableItem{item})
		result.FreedSpace += singleResult.FreedSpace
		result.CleanedItems += singleResult.CleanedItems
		result.Errors = append(result.Errors, singleResult.Errors...)

		if callbacks.OnItemDone != nil {
			success := len(singleResult.Errors) == 0
			errMsg := ""
			if !success {
				errMsg = singleResult.Errors[0]
			}
			callbacks.OnItemDone(types.ItemCleanedResult{
				Path:    item.Path,
				Name:    item.Name,
				Size:    item.Size,
				Success: success,
				ErrMsg:  errMsg,
			})
		}
	}

	return result
}

// PrepareJobs prepares clean jobs from scan results, filtering by selection and exclusion.
func (s *CleanService) PrepareJobs(
	resultMap map[string]*types.ScanResult,
	selected map[string]bool,
	excluded map[string]map[string]bool,
) []CleanJob {
	var jobs []CleanJob

	for id, sel := range selected {
		if !sel {
			continue
		}
		r, ok := resultMap[id]
		if !ok {
			continue
		}
		if r.Category.Method == types.MethodManual {
			continue
		}

		var items []types.CleanableItem
		excludedMap := excluded[id]
		for _, item := range r.Items {
			if item.Status == types.ItemStatusProcessLocked {
				continue
			}
			if excludedMap == nil || !excludedMap[item.Path] {
				items = append(items, item)
			}
		}
		if len(items) == 0 {
			continue
		}

		jobs = append(jobs, CleanJob{
			Category: r.Category,
			Items:    items,
		})
	}

	return jobs
}
