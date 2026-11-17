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
	executor *Executor
}

// NewCleanService creates a new CleanService.
func NewCleanService(registry *target.Registry) *CleanService {
	return &CleanService{
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
		case types.MethodPermanent:
			result = s.cleanByItem(job, callbacks, &currentItem, totalItems, s.executor.Permanent)
		default:
			result = s.cleanUnsupported(job)
		}

		if result != nil {
			report.AddResult(result)

			if callbacks.OnCategoryDone != nil {
				callbacks.OnCategoryDone(types.CategoryCleanedResult{
					CategoryName:            job.Category.Name,
					FreedSpace:              result.FreedSpace,
					TrashedBytes:            result.TrashedBytes,
					PermanentlyDeletedBytes: result.PermanentlyDeletedBytes,
					CleanedItems:            result.CleanedItems,
					ProcessedItems:          result.ProcessedItems,
					ErrorCount:              len(result.Errors),
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
		if job, ok := buildJob(resultMap, excluded, id); ok {
			jobs = append(jobs, job)
		}
	}

	return jobs
}

// PrepareJobsWithOrder prepares clean jobs in a deterministic order.
func (s *CleanService) PrepareJobsWithOrder(
	resultMap map[string]*types.ScanResult,
	selected map[string]bool,
	excluded map[string]map[string]bool,
	order []string,
) []CleanJob {
	if len(order) == 0 {
		return s.PrepareJobs(resultMap, selected, excluded)
	}

	var jobs []CleanJob
	for _, id := range order {
		if !selected[id] {
			continue
		}
		if job, ok := buildJob(resultMap, excluded, id); ok {
			jobs = append(jobs, job)
		}
	}

	return jobs
}

// buildJob creates a CleanJob for the given category ID, filtering out manual,
// locked, and excluded items. Returns false if no cleanable items remain.
func buildJob(
	resultMap map[string]*types.ScanResult,
	excluded map[string]map[string]bool,
	id string,
) (CleanJob, bool) {
	r, ok := resultMap[id]
	if !ok {
		return CleanJob{}, false
	}
	if r.Category.Method == types.MethodManual {
		return CleanJob{}, false
	}
	for _, proc := range r.Category.BlockedByProcesses {
		if utils.IsProcessRunning(proc) {
			logger.Warn("skipping target: blocking process running", "id", id, "process", proc)
			return CleanJob{}, false
		}
	}

	excludedMap := excluded[id]
	var items []types.CleanableItem
	for _, item := range r.Items {
		if item.Status == types.ItemStatusProcessLocked {
			continue
		}
		if excludedMap == nil || !excludedMap[item.Path] {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return CleanJob{}, false
	}

	return CleanJob{Category: r.Category, Items: items}, true
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

	result := s.executor.Builtin(job.Category, job.Items)
	*currentItem += len(job.Items)
	return result
}

// cleanTrashBatch handles a trash category and reports progress around the operation.
func (s *CleanService) cleanTrashBatch(job CleanJob, callbacks types.CleanCallbacks, currentItem *int, totalItems int) *types.CleanResult {
	items := job.Items
	if len(items) == 0 {
		return types.NewCleanResult(job.Category)
	}

	if callbacks.OnProgress != nil {
		callbacks.OnProgress(types.CleanProgress{
			CategoryName: job.Category.Name,
			CurrentItem:  items[0].Name,
			Current:      *currentItem,
			Total:        totalItems,
		})
	}

	result := s.executor.Trash(job.Category, items)
	s.sendBatchItemCallbacks(items, result, callbacks)
	*currentItem += len(items)

	if callbacks.OnProgress != nil {
		callbacks.OnProgress(types.CleanProgress{
			CategoryName: job.Category.Name,
			CurrentItem:  items[len(items)-1].Name,
			Current:      *currentItem,
			Total:        totalItems,
		})
	}

	return result
}

// sendBatchItemCallbacks sends OnItemDone callbacks for batch items with error tracking.
func (s *CleanService) sendBatchItemCallbacks(batch []types.CleanableItem, batchResult *types.CleanResult, callbacks types.CleanCallbacks) {
	if callbacks.OnItemDone == nil {
		return
	}

	names := make(map[string]string, len(batch))
	for _, item := range batch {
		if _, exists := names[item.Path]; !exists {
			names[item.Path] = item.Name
		}
	}
	for _, outcome := range batchResult.ItemResults {
		outcome.Name = names[outcome.Path]
		callbacks.OnItemDone(outcome)
	}
}

type itemCleaner func(types.Category, []types.CleanableItem) *types.CleanResult

// cleanByItem handles item-by-item processing with a provided executor function.
func (s *CleanService) cleanByItem(job CleanJob, callbacks types.CleanCallbacks, currentItem *int, totalItems int, exec itemCleaner) *types.CleanResult {
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

		singleResult := exec(job.Category, []types.CleanableItem{item})
		result.Merge(singleResult)

		if callbacks.OnItemDone != nil {
			success := singleResult.ProcessedItems > 0
			outcome := types.ItemOutcomeFailed
			switch {
			case success:
				outcome = types.ItemOutcomeProcessed
			case singleResult.SkippedItems > 0:
				outcome = types.ItemOutcomeSkipped
			case singleResult.AlreadyAbsentItems > 0:
				outcome = types.ItemOutcomeAbsent
			case singleResult.UnconfirmedItems > 0:
				outcome = types.ItemOutcomeUnconfirmed
			}
			errMsg := ""
			if len(singleResult.Errors) > 0 {
				errMsg = singleResult.Errors[0]
			} else if !success {
				errMsg = "skipped or already absent"
			}
			size := singleResult.PermanentlyDeletedBytes
			callbacks.OnItemDone(types.ItemCleanedResult{
				Path:    item.Path,
				Name:    item.Name,
				Size:    size,
				Success: success,
				ErrMsg:  errMsg,
				Outcome: outcome,
			})
		}
	}

	return result
}

func (s *CleanService) cleanUnsupported(job CleanJob) *types.CleanResult {
	logger.Info("unsupported clean method",
		"category", job.Category.Name,
		"method", job.Category.Method,
		"items", len(job.Items))

	result := types.NewCleanResult(job.Category)
	result.PlannedBytes = types.PlannedSize(job.Items)
	result.Errors = append(result.Errors, "unsupported method: "+string(job.Category.Method))
	return result
}
