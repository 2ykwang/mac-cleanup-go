package cleaner

import (
	"github.com/2ykwang/mac-cleanup-go/internal/target"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

// CleanJob represents a cleaning job for a category.
type CleanJob struct {
	Category types.Category
	Items    []types.CleanableItem
}

// Progress represents the current cleaning progress.
type Progress struct {
	CategoryName string
	CurrentItem  string
	Current      int
	Total        int
}

// ItemResult represents the result of cleaning a single item.
type ItemResult struct {
	Path    string
	Name    string
	Size    int64
	Success bool
	ErrMsg  string
}

// CategoryResult represents the result of cleaning a category.
type CategoryResult struct {
	CategoryName string
	FreedSpace   int64
	CleanedItems int
	ErrorCount   int
}

// Callbacks holds callback functions for cleaning progress.
type Callbacks struct {
	OnProgress     func(Progress)
	OnItemDone     func(ItemResult)
	OnCategoryDone func(CategoryResult)
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
// Returns the final report.
func (s *CleanService) Clean(jobs []CleanJob, callbacks Callbacks) *types.Report {
	report := &types.Report{Results: make([]types.CleanResult, 0)}

	// Calculate total items
	totalItems := 0
	for _, job := range jobs {
		totalItems += len(job.Items)
	}

	currentItem := 0

	for _, job := range jobs {
		var result *types.CleanResult
		cat := job.Category

		// Builtin methods: batch processing (category-level progress)
		// Other methods: item-by-item processing (item-level progress)
		if cat.Method == types.MethodBuiltin {
			if callbacks.OnProgress != nil {
				callbacks.OnProgress(Progress{
					CategoryName: cat.Name,
					CurrentItem:  "",
					Current:      currentItem,
					Total:        totalItems,
				})
			}

			result = s.executor.Clean(cat, job.Items)
			currentItem += len(job.Items)
		} else {
			// Clean items one by one for progress tracking
			itemResult := &types.CleanResult{
				Category: cat,
				Errors:   make([]string, 0),
			}

			for _, item := range job.Items {
				currentItem++

				// Send progress update
				if callbacks.OnProgress != nil {
					callbacks.OnProgress(Progress{
						CategoryName: cat.Name,
						CurrentItem:  item.Name,
						Current:      currentItem,
						Total:        totalItems,
					})
				}

				singleResult := s.executor.Clean(cat, []types.CleanableItem{item})
				itemResult.FreedSpace += singleResult.FreedSpace
				itemResult.CleanedItems += singleResult.CleanedItems
				itemResult.Errors = append(itemResult.Errors, singleResult.Errors...)

				// Send item done callback
				if callbacks.OnItemDone != nil {
					success := len(singleResult.Errors) == 0
					errMsg := ""
					if !success && len(singleResult.Errors) > 0 {
						errMsg = singleResult.Errors[0]
					}
					callbacks.OnItemDone(ItemResult{
						Path:    item.Path,
						Name:    item.Name,
						Size:    item.Size,
						Success: success,
						ErrMsg:  errMsg,
					})
				}
			}
			result = itemResult
		}

		if result != nil {
			report.Results = append(report.Results, *result)
			report.FreedSpace += result.FreedSpace
			report.CleanedItems += result.CleanedItems
			report.FailedItems += len(result.Errors)

			// Send category done callback
			if callbacks.OnCategoryDone != nil {
				callbacks.OnCategoryDone(CategoryResult{
					CategoryName: cat.Name,
					FreedSpace:   result.FreedSpace,
					CleanedItems: result.CleanedItems,
					ErrorCount:   len(result.Errors),
				})
			}
		}
	}

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
