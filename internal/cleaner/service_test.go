package cleaner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/mocks"
	"github.com/2ykwang/mac-cleanup-go/internal/target"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

// Helper function to create a test ScanResult
func newTestScanResult(id, name string, method types.CleanupMethod, items []types.CleanableItem) *types.ScanResult {
	result := types.NewScanResult(types.Category{
		ID:     id,
		Name:   name,
		Method: method,
	})
	result.Items = items
	return result
}

// Helper function to create test CleanableItems
func newTestItems(paths ...string) []types.CleanableItem {
	items := make([]types.CleanableItem, len(paths))
	for i, path := range paths {
		items[i] = types.CleanableItem{
			Path: path,
			Name: path,
			Size: int64((i + 1) * 100),
		}
	}
	return items
}

// newMockTargetForService creates a MockTarget with default behavior for service tests.
func newMockTargetForService(cat types.Category) *mocks.MockTarget {
	m := new(mocks.MockTarget)
	m.On("Category").Return(cat)
	m.On("IsAvailable").Return(true)
	m.On("Scan").Return((*types.ScanResult)(nil), nil)
	return m
}

func TestNewCleanService(t *testing.T) {
	registry := target.NewRegistry()
	service := NewCleanService(registry)

	require.NotNil(t, service)
	assert.NotNil(t, service.registry)
	assert.NotNil(t, service.executor)
}

func TestPrepareJobs_FiltersUnselected(t *testing.T) {
	service := NewCleanService(target.NewRegistry())

	resultMap := map[string]*types.ScanResult{
		"cat1": newTestScanResult("cat1", "Category 1", types.MethodTrash, newTestItems("/path1")),
		"cat2": newTestScanResult("cat2", "Category 2", types.MethodTrash, newTestItems("/path2")),
		"cat3": newTestScanResult("cat3", "Category 3", types.MethodTrash, newTestItems("/path3")),
	}

	selected := map[string]bool{
		"cat1": true,
		"cat2": false, // not selected
		"cat3": true,
	}

	jobs := service.PrepareJobs(resultMap, selected, nil)

	assert.Len(t, jobs, 2)

	// Verify cat2 is not in jobs
	for _, job := range jobs {
		assert.NotEqual(t, "cat2", job.Category.ID, "Unselected category should not be in jobs")
	}
}

func TestPrepareJobs_SkipsMissingResults(t *testing.T) {
	service := NewCleanService(target.NewRegistry())

	resultMap := map[string]*types.ScanResult{
		"cat1": newTestScanResult("cat1", "Category 1", types.MethodTrash, newTestItems("/path1")),
		// cat2 is missing from resultMap
	}

	selected := map[string]bool{
		"cat1": true,
		"cat2": true, // selected but not in resultMap
	}

	jobs := service.PrepareJobs(resultMap, selected, nil)

	assert.Len(t, jobs, 1)
	assert.Equal(t, "cat1", jobs[0].Category.ID)
}

func TestPrepareJobs_SkipsManualMethod(t *testing.T) {
	service := NewCleanService(target.NewRegistry())

	resultMap := map[string]*types.ScanResult{
		"cat1": newTestScanResult("cat1", "Category 1", types.MethodTrash, newTestItems("/path1")),
		"cat2": newTestScanResult("cat2", "Manual Category", types.MethodManual, newTestItems("/path2")),
		"cat3": newTestScanResult("cat3", "Category 3", types.MethodPermanent, newTestItems("/path3")),
	}

	selected := map[string]bool{
		"cat1": true,
		"cat2": true, // MethodManual should be skipped
		"cat3": true,
	}

	jobs := service.PrepareJobs(resultMap, selected, nil)

	assert.Len(t, jobs, 2)

	// Verify MethodManual is not in jobs
	for _, job := range jobs {
		assert.NotEqual(t, types.MethodManual, job.Category.Method,
			"MethodManual should not be in jobs")
	}
}

func TestPrepareJobs_FiltersExcludedItems(t *testing.T) {
	service := NewCleanService(target.NewRegistry())

	resultMap := map[string]*types.ScanResult{
		"cat1": newTestScanResult("cat1", "Category 1", types.MethodTrash,
			newTestItems("/path1", "/path2", "/path3")),
	}

	selected := map[string]bool{
		"cat1": true,
	}

	excluded := map[string]map[string]bool{
		"cat1": {
			"/path2": true, // exclude this item
		},
	}

	jobs := service.PrepareJobs(resultMap, selected, excluded)

	require.Len(t, jobs, 1)
	assert.Len(t, jobs[0].Items, 2)

	// Verify /path2 is not in items
	for _, item := range jobs[0].Items {
		assert.NotEqual(t, "/path2", item.Path, "Excluded item should not be in jobs")
	}

	// Verify /path1 and /path3 are in items
	paths := make([]string, len(jobs[0].Items))
	for i, item := range jobs[0].Items {
		paths[i] = item.Path
	}
	assert.Contains(t, paths, "/path1")
	assert.Contains(t, paths, "/path3")
}

func TestPrepareJobs_SkipsLockedItems(t *testing.T) {
	service := NewCleanService(target.NewRegistry())

	resultMap := map[string]*types.ScanResult{
		"cat1": newTestScanResult("cat1", "Category 1", types.MethodTrash, []types.CleanableItem{
			{Path: "/path1", Name: "path1", Status: types.ItemStatusAvailable},
			{Path: "/path2", Name: "path2", Status: types.ItemStatusProcessLocked},
			{Path: "/path3", Name: "path3"},
		}),
	}

	selected := map[string]bool{
		"cat1": true,
	}

	jobs := service.PrepareJobs(resultMap, selected, nil)

	require.Len(t, jobs, 1)
	assert.Len(t, jobs[0].Items, 2)

	for _, item := range jobs[0].Items {
		assert.NotEqual(t, "/path2", item.Path, "locked item should not be in jobs")
	}
}

func TestPrepareJobs_SkipsWhenAllItemsLocked(t *testing.T) {
	service := NewCleanService(target.NewRegistry())

	resultMap := map[string]*types.ScanResult{
		"cat1": newTestScanResult("cat1", "Category 1", types.MethodTrash, []types.CleanableItem{
			{Path: "/path1", Name: "path1", Status: types.ItemStatusProcessLocked},
		}),
	}

	selected := map[string]bool{
		"cat1": true,
	}

	jobs := service.PrepareJobs(resultMap, selected, nil)

	assert.Empty(t, jobs)
}

func TestPrepareJobs_SkipsWhenAllItemsExcluded(t *testing.T) {
	service := NewCleanService(target.NewRegistry())

	resultMap := map[string]*types.ScanResult{
		"cat1": newTestScanResult("cat1", "Category 1", types.MethodTrash,
			newTestItems("/path1", "/path2")),
	}

	selected := map[string]bool{
		"cat1": true,
	}

	// Exclude all items
	excluded := map[string]map[string]bool{
		"cat1": {
			"/path1": true,
			"/path2": true,
		},
	}

	jobs := service.PrepareJobs(resultMap, selected, excluded)

	assert.Len(t, jobs, 0, "Should return empty jobs when all items are excluded")
}

func TestPrepareJobs_EmptyInputs(t *testing.T) {
	service := NewCleanService(target.NewRegistry())

	t.Run("empty resultMap", func(t *testing.T) {
		jobs := service.PrepareJobs(
			map[string]*types.ScanResult{},
			map[string]bool{"cat1": true},
			nil,
		)
		assert.Len(t, jobs, 0)
	})

	t.Run("empty selected", func(t *testing.T) {
		resultMap := map[string]*types.ScanResult{
			"cat1": newTestScanResult("cat1", "Category 1", types.MethodTrash, newTestItems("/path1")),
		}
		jobs := service.PrepareJobs(resultMap, map[string]bool{}, nil)
		assert.Len(t, jobs, 0)
	})

	t.Run("nil inputs", func(t *testing.T) {
		jobs := service.PrepareJobs(nil, nil, nil)
		assert.Len(t, jobs, 0)
	})
}

func TestPrepareJobs_NilExcludedMap(t *testing.T) {
	service := NewCleanService(target.NewRegistry())

	resultMap := map[string]*types.ScanResult{
		"cat1": newTestScanResult("cat1", "Category 1", types.MethodTrash,
			newTestItems("/path1", "/path2")),
	}

	selected := map[string]bool{
		"cat1": true,
	}

	// nil excluded map should include all items
	jobs := service.PrepareJobs(resultMap, selected, nil)

	require.Len(t, jobs, 1)
	assert.Len(t, jobs[0].Items, 2, "All items should be included when excluded is nil")
}

func TestPrepareJobs_EmptyExcludedMapForCategory(t *testing.T) {
	service := NewCleanService(target.NewRegistry())

	resultMap := map[string]*types.ScanResult{
		"cat1": newTestScanResult("cat1", "Category 1", types.MethodTrash,
			newTestItems("/path1", "/path2")),
	}

	selected := map[string]bool{
		"cat1": true,
	}

	// excluded map exists but category has no excluded items
	excluded := map[string]map[string]bool{
		"cat1": {}, // empty map for this category
	}

	jobs := service.PrepareJobs(resultMap, selected, excluded)

	require.Len(t, jobs, 1)
	assert.Len(t, jobs[0].Items, 2, "All items should be included when category excluded map is empty")
}

func TestPrepareJobs_PreservesItemProperties(t *testing.T) {
	service := NewCleanService(target.NewRegistry())

	originalItems := []types.CleanableItem{
		{Path: "/path1", Name: "File 1", Size: 1000, IsDirectory: false},
		{Path: "/path2", Name: "Dir 1", Size: 2000, IsDirectory: true},
	}

	resultMap := map[string]*types.ScanResult{
		"cat1": newTestScanResult("cat1", "Category 1", types.MethodTrash, originalItems),
	}

	selected := map[string]bool{
		"cat1": true,
	}

	jobs := service.PrepareJobs(resultMap, selected, nil)

	require.Len(t, jobs, 1)
	require.Len(t, jobs[0].Items, 2)

	// Verify item properties are preserved
	assert.Equal(t, "/path1", jobs[0].Items[0].Path)
	assert.Equal(t, "File 1", jobs[0].Items[0].Name)
	assert.Equal(t, int64(1000), jobs[0].Items[0].Size)
	assert.False(t, jobs[0].Items[0].IsDirectory)

	assert.Equal(t, "/path2", jobs[0].Items[1].Path)
	assert.Equal(t, "Dir 1", jobs[0].Items[1].Name)
	assert.Equal(t, int64(2000), jobs[0].Items[1].Size)
	assert.True(t, jobs[0].Items[1].IsDirectory)
}

func TestPrepareJobs_PreservesCategoryProperties(t *testing.T) {
	service := NewCleanService(target.NewRegistry())

	resultMap := map[string]*types.ScanResult{
		"cat1": {
			Category: types.Category{
				ID:     "cat1",
				Name:   "Test Category",
				Method: types.MethodTrash,
				Safety: types.SafetyLevelSafe,
				Guide:  "Test guide",
			},
			Items: newTestItems("/path1"),
		},
	}

	selected := map[string]bool{
		"cat1": true,
	}

	jobs := service.PrepareJobs(resultMap, selected, nil)

	require.Len(t, jobs, 1)
	assert.Equal(t, "cat1", jobs[0].Category.ID)
	assert.Equal(t, "Test Category", jobs[0].Category.Name)
	assert.Equal(t, types.MethodTrash, jobs[0].Category.Method)
	assert.Equal(t, types.SafetyLevelSafe, jobs[0].Category.Safety)
	assert.Equal(t, "Test guide", jobs[0].Category.Guide)
}

func TestPrepareJobs_MultipleCategories(t *testing.T) {
	service := NewCleanService(target.NewRegistry())

	resultMap := map[string]*types.ScanResult{
		"cat1": newTestScanResult("cat1", "Category 1", types.MethodTrash, newTestItems("/a1", "/a2")),
		"cat2": newTestScanResult("cat2", "Category 2", types.MethodPermanent, newTestItems("/b1")),
		"cat3": newTestScanResult("cat3", "Category 3", types.MethodBuiltin, newTestItems("/c1", "/c2", "/c3")),
	}

	selected := map[string]bool{
		"cat1": true,
		"cat2": true,
		"cat3": true,
	}

	excluded := map[string]map[string]bool{
		"cat1": {"/a1": true}, // exclude 1 item from cat1
		"cat3": {"/c2": true}, // exclude 1 item from cat3
	}

	jobs := service.PrepareJobs(resultMap, selected, excluded)

	assert.Len(t, jobs, 3)

	// Find jobs by category ID (order is not guaranteed due to map iteration)
	jobMap := make(map[string]CleanJob)
	for _, job := range jobs {
		jobMap[job.Category.ID] = job
	}

	// cat1: 2 items - 1 excluded = 1 item
	require.Contains(t, jobMap, "cat1")
	assert.Len(t, jobMap["cat1"].Items, 1)
	assert.Equal(t, "/a2", jobMap["cat1"].Items[0].Path)

	// cat2: 1 item, none excluded
	require.Contains(t, jobMap, "cat2")
	assert.Len(t, jobMap["cat2"].Items, 1)

	// cat3: 3 items - 1 excluded = 2 items
	require.Contains(t, jobMap, "cat3")
	assert.Len(t, jobMap["cat3"].Items, 2)
}

func TestPrepareJobs_ResultWithEmptyItems(t *testing.T) {
	service := NewCleanService(target.NewRegistry())

	resultMap := map[string]*types.ScanResult{
		"cat1": newTestScanResult("cat1", "Category 1", types.MethodTrash, []types.CleanableItem{}),
	}

	selected := map[string]bool{
		"cat1": true,
	}

	jobs := service.PrepareJobs(resultMap, selected, nil)

	assert.Len(t, jobs, 0, "Category with empty items should not produce a job")
}

// =============================================================================
// Clean Method Tests
// =============================================================================

func TestClean_EmptyJobs(t *testing.T) {
	service := NewCleanService(target.NewRegistry())

	report := service.Clean([]CleanJob{}, Callbacks{})

	require.NotNil(t, report)
	assert.Equal(t, int64(0), report.FreedSpace)
	assert.Equal(t, 0, report.CleanedItems)
	assert.Equal(t, 0, report.FailedItems)
	assert.Len(t, report.Results, 0)
}

func TestClean_NilCallbacks(t *testing.T) {
	// Setup: use MoveToTrashBatch mock to avoid actual file operations
	original := utils.MoveToTrashBatch
	defer func() { utils.MoveToTrashBatch = original }()
	utils.MoveToTrashBatch = func(paths []string) utils.TrashBatchResult {
		return utils.TrashBatchResult{Succeeded: paths, Failed: make(map[string]error)}
	}

	service := NewCleanService(target.NewRegistry())

	jobs := []CleanJob{
		{
			Category: types.Category{
				ID:     "cat1",
				Name:   "Test Category",
				Method: types.MethodTrash,
			},
			Items: newTestItems("/path1", "/path2"),
		},
	}

	// Should not panic with nil callbacks
	report := service.Clean(jobs, Callbacks{})

	require.NotNil(t, report)
	assert.Equal(t, 2, report.CleanedItems)
}

func TestClean_CallsOnProgress_TrashMethod(t *testing.T) {
	original := utils.MoveToTrashBatch
	defer func() { utils.MoveToTrashBatch = original }()
	utils.MoveToTrashBatch = func(paths []string) utils.TrashBatchResult {
		return utils.TrashBatchResult{Succeeded: paths, Failed: make(map[string]error)}
	}

	service := NewCleanService(target.NewRegistry())

	jobs := []CleanJob{
		{
			Category: types.Category{
				ID:     "cat1",
				Name:   "Test Category",
				Method: types.MethodTrash,
			},
			Items: newTestItems("/path1", "/path2", "/path3"),
		},
	}

	var progressCalls []Progress
	callbacks := Callbacks{
		OnProgress: func(p Progress) {
			progressCalls = append(progressCalls, p)
		},
	}

	service.Clean(jobs, callbacks)

	// For trash method, OnProgress is called before and after each batch
	// 3 items < batch size 50, so 2 progress calls (start + end)
	assert.Len(t, progressCalls, 2)

	// Verify progress at batch start
	assert.Equal(t, 0, progressCalls[0].Current)
	assert.Equal(t, 3, progressCalls[0].Total)
	assert.Equal(t, "Test Category", progressCalls[0].CategoryName)
	assert.Equal(t, "/path1", progressCalls[0].CurrentItem) // First item in batch

	// Verify progress at batch end
	assert.Equal(t, 3, progressCalls[1].Current)
	assert.Equal(t, 3, progressCalls[1].Total)
	assert.Equal(t, "/path3", progressCalls[1].CurrentItem) // Last item in batch
}

func TestClean_CallsOnProgress_PermanentMethod(t *testing.T) {
	service := NewCleanService(target.NewRegistry())

	jobs := []CleanJob{
		{
			Category: types.Category{
				ID:     "cat1",
				Name:   "Test Category",
				Method: types.MethodPermanent, // Non-trash, non-builtin
			},
			Items: newTestItems("/path1", "/path2", "/path3"),
		},
	}

	var progressCalls []Progress
	var itemDoneCalls []ItemResult
	callbacks := Callbacks{
		OnProgress: func(p Progress) {
			progressCalls = append(progressCalls, p)
		},
		OnItemDone: func(r ItemResult) {
			itemDoneCalls = append(itemDoneCalls, r)
		},
	}

	service.Clean(jobs, callbacks)

	// Permanent method processes items one by one
	assert.Len(t, progressCalls, 3)

	// Verify progress increments
	assert.Equal(t, 1, progressCalls[0].Current)
	assert.Equal(t, 3, progressCalls[0].Total)
	assert.Equal(t, "/path1", progressCalls[0].CurrentItem)

	assert.Equal(t, 2, progressCalls[1].Current)
	assert.Equal(t, "/path2", progressCalls[1].CurrentItem)

	assert.Equal(t, 3, progressCalls[2].Current)
	assert.Equal(t, "/path3", progressCalls[2].CurrentItem)

	// OnItemDone called for each item
	assert.Len(t, itemDoneCalls, 3)
	// Files don't exist, so deletePermanent fails
	for _, r := range itemDoneCalls {
		assert.False(t, r.Success)
		assert.NotEmpty(t, r.ErrMsg)
	}
}

func TestClean_CallsOnProgress_Builtin(t *testing.T) {
	registry := target.NewRegistry()
	cat := types.Category{
		ID:     "docker",
		Name:   "Docker",
		Method: types.MethodBuiltin,
	}

	mockTarget := newMockTargetForService(cat)
	cleanResult := types.NewCleanResult(cat)
	cleanResult.CleanedItems = 3
	cleanResult.FreedSpace = 300
	cleanResult.Errors = []string{}
	mockTarget.On("Clean", mock.Anything).Return(cleanResult, nil)
	registry.Register(mockTarget)

	service := NewCleanService(registry)

	jobs := []CleanJob{
		{
			Category: cat,
			Items:    newTestItems("/path1", "/path2", "/path3"),
		},
	}

	var progressCalls []Progress
	callbacks := Callbacks{
		OnProgress: func(p Progress) {
			progressCalls = append(progressCalls, p)
		},
	}

	service.Clean(jobs, callbacks)

	// For builtin methods, OnProgress is called once per category (batch processing)
	assert.Len(t, progressCalls, 1)
	assert.Equal(t, 0, progressCalls[0].Current) // starts at 0 for builtin
	assert.Equal(t, 3, progressCalls[0].Total)
	assert.Equal(t, "Docker", progressCalls[0].CategoryName)
	assert.Equal(t, "", progressCalls[0].CurrentItem) // empty for builtin batch
}

func TestClean_CallsOnItemDone(t *testing.T) {
	original := utils.MoveToTrashBatch
	defer func() { utils.MoveToTrashBatch = original }()
	utils.MoveToTrashBatch = func(paths []string) utils.TrashBatchResult {
		return utils.TrashBatchResult{Succeeded: paths, Failed: make(map[string]error)}
	}

	service := NewCleanService(target.NewRegistry())

	items := []types.CleanableItem{
		{Path: "/path1", Name: "File 1", Size: 100},
		{Path: "/path2", Name: "File 2", Size: 200},
	}

	jobs := []CleanJob{
		{
			Category: types.Category{
				ID:     "cat1",
				Name:   "Test Category",
				Method: types.MethodTrash,
			},
			Items: items,
		},
	}

	var itemResults []ItemResult
	callbacks := Callbacks{
		OnItemDone: func(r ItemResult) {
			itemResults = append(itemResults, r)
		},
	}

	service.Clean(jobs, callbacks)

	// OnItemDone is called for each item (non-builtin only)
	require.Len(t, itemResults, 2)

	assert.Equal(t, "/path1", itemResults[0].Path)
	assert.Equal(t, "File 1", itemResults[0].Name)
	assert.Equal(t, int64(100), itemResults[0].Size)
	assert.True(t, itemResults[0].Success)
	assert.Empty(t, itemResults[0].ErrMsg)

	assert.Equal(t, "/path2", itemResults[1].Path)
	assert.Equal(t, "File 2", itemResults[1].Name)
	assert.Equal(t, int64(200), itemResults[1].Size)
	assert.True(t, itemResults[1].Success)
}

func TestClean_CallsOnItemDone_WithError(t *testing.T) {
	original := utils.MoveToTrashBatch
	defer func() { utils.MoveToTrashBatch = original }()
	utils.MoveToTrashBatch = func(paths []string) utils.TrashBatchResult {
		result := utils.TrashBatchResult{
			Succeeded: make([]string, 0, len(paths)),
			Failed:    make(map[string]error),
		}
		for _, p := range paths {
			if p == "/path2" {
				result.Failed[p] = assert.AnError
			} else {
				result.Succeeded = append(result.Succeeded, p)
			}
		}
		return result
	}

	service := NewCleanService(target.NewRegistry())

	jobs := []CleanJob{
		{
			Category: types.Category{
				ID:     "cat1",
				Name:   "Test Category",
				Method: types.MethodTrash,
			},
			Items: newTestItems("/path1", "/path2"),
		},
	}

	var itemResults []ItemResult
	callbacks := Callbacks{
		OnItemDone: func(r ItemResult) {
			itemResults = append(itemResults, r)
		},
	}

	service.Clean(jobs, callbacks)

	require.Len(t, itemResults, 2)

	// First item succeeded
	assert.True(t, itemResults[0].Success)
	assert.Empty(t, itemResults[0].ErrMsg)

	// Second item failed
	assert.False(t, itemResults[1].Success)
	assert.NotEmpty(t, itemResults[1].ErrMsg)
}

func TestClean_CallsOnCategoryDone(t *testing.T) {
	original := utils.MoveToTrashBatch
	defer func() { utils.MoveToTrashBatch = original }()
	utils.MoveToTrashBatch = func(paths []string) utils.TrashBatchResult {
		return utils.TrashBatchResult{Succeeded: paths, Failed: make(map[string]error)}
	}

	service := NewCleanService(target.NewRegistry())

	jobs := []CleanJob{
		{
			Category: types.Category{
				ID:     "cat1",
				Name:   "Category 1",
				Method: types.MethodTrash,
			},
			Items: newTestItems("/path1", "/path2"),
		},
		{
			Category: types.Category{
				ID:     "cat2",
				Name:   "Category 2",
				Method: types.MethodTrash,
			},
			Items: newTestItems("/path3"),
		},
	}

	var categoryResults []CategoryResult
	callbacks := Callbacks{
		OnCategoryDone: func(r CategoryResult) {
			categoryResults = append(categoryResults, r)
		},
	}

	service.Clean(jobs, callbacks)

	require.Len(t, categoryResults, 2)

	assert.Equal(t, "Category 1", categoryResults[0].CategoryName)
	assert.Equal(t, 2, categoryResults[0].CleanedItems)
	assert.Equal(t, int64(300), categoryResults[0].FreedSpace) // 100 + 200
	assert.Equal(t, 0, categoryResults[0].ErrorCount)

	assert.Equal(t, "Category 2", categoryResults[1].CategoryName)
	assert.Equal(t, 1, categoryResults[1].CleanedItems)
	assert.Equal(t, int64(100), categoryResults[1].FreedSpace)
	assert.Equal(t, 0, categoryResults[1].ErrorCount)
}

func TestClean_BuiltinBatchProcessing(t *testing.T) {
	registry := target.NewRegistry()
	cat := types.Category{
		ID:     "docker",
		Name:   "Docker",
		Method: types.MethodBuiltin,
	}

	mockTarget := newMockTargetForService(cat)
	cleanResult := types.NewCleanResult(cat)
	cleanResult.CleanedItems = 3
	cleanResult.FreedSpace = 300
	cleanResult.Errors = []string{}
	mockTarget.On("Clean", mock.Anything).Return(cleanResult, nil)
	registry.Register(mockTarget)

	service := NewCleanService(registry)

	jobs := []CleanJob{
		{
			Category: cat,
			Items:    newTestItems("/path1", "/path2", "/path3"),
		},
	}

	var itemDoneCalls int
	callbacks := Callbacks{
		OnItemDone: func(_ ItemResult) {
			itemDoneCalls++
		},
	}

	report := service.Clean(jobs, callbacks)

	// Builtin processes all items in one batch
	mockTarget.AssertNumberOfCalls(t, "Clean", 1)

	// Verify items passed to Clean
	cleanCall := mockTarget.Calls[len(mockTarget.Calls)-1]
	items := cleanCall.Arguments.Get(0).([]types.CleanableItem)
	assert.Len(t, items, 3, "All items should be passed in single call")

	// OnItemDone is NOT called for builtin (batch processing)
	assert.Equal(t, 0, itemDoneCalls, "OnItemDone should not be called for builtin")

	assert.Equal(t, 3, report.CleanedItems)
}

func TestClean_TrashMethodBatchProcessing(t *testing.T) {
	original := utils.MoveToTrashBatch
	defer func() { utils.MoveToTrashBatch = original }()

	var batchCalls [][]string
	utils.MoveToTrashBatch = func(paths []string) utils.TrashBatchResult {
		batchCalls = append(batchCalls, paths)
		return utils.TrashBatchResult{Succeeded: paths, Failed: make(map[string]error)}
	}

	service := NewCleanService(target.NewRegistry())

	jobs := []CleanJob{
		{
			Category: types.Category{
				ID:     "cat1",
				Name:   "Test Category",
				Method: types.MethodTrash,
			},
			Items: newTestItems("/path1", "/path2", "/path3"),
		},
	}

	var itemDoneCalls int
	callbacks := Callbacks{
		OnItemDone: func(_ ItemResult) {
			itemDoneCalls++
		},
	}

	report := service.Clean(jobs, callbacks)

	// Trash method processes items in batches (batch size 50)
	// 3 items < batch size, so single batch call
	require.Len(t, batchCalls, 1, "Should call MoveToTrashBatch once for batch")
	assert.Equal(t, []string{"/path1", "/path2", "/path3"}, batchCalls[0])

	// OnItemDone is called for each item after batch completes
	assert.Equal(t, 3, itemDoneCalls, "OnItemDone should be called for each item")

	assert.Equal(t, 3, report.CleanedItems)
}

func TestClean_AggregatesResults(t *testing.T) {
	original := utils.MoveToTrashBatch
	defer func() { utils.MoveToTrashBatch = original }()
	utils.MoveToTrashBatch = func(paths []string) utils.TrashBatchResult {
		result := utils.TrashBatchResult{
			Succeeded: make([]string, 0, len(paths)),
			Failed:    make(map[string]error),
		}
		for _, p := range paths {
			if p == "/fail" {
				result.Failed[p] = assert.AnError
			} else {
				result.Succeeded = append(result.Succeeded, p)
			}
		}
		return result
	}

	service := NewCleanService(target.NewRegistry())

	jobs := []CleanJob{
		{
			Category: types.Category{ID: "cat1", Name: "Cat 1", Method: types.MethodTrash},
			Items: []types.CleanableItem{
				{Path: "/path1", Name: "File 1", Size: 100},
				{Path: "/fail", Name: "Fail", Size: 50},
			},
		},
		{
			Category: types.Category{ID: "cat2", Name: "Cat 2", Method: types.MethodTrash},
			Items: []types.CleanableItem{
				{Path: "/path2", Name: "File 2", Size: 200},
			},
		},
	}

	report := service.Clean(jobs, Callbacks{})

	// Aggregate results
	assert.Equal(t, int64(300), report.FreedSpace) // 100 + 200 (fail doesn't count)
	assert.Equal(t, 2, report.CleanedItems)        // 2 succeeded
	assert.Equal(t, 1, report.FailedItems)         // 1 failed
	assert.Len(t, report.Results, 2)               // 2 categories
}

func TestClean_MultipleCategories(t *testing.T) {
	original := utils.MoveToTrashBatch
	defer func() { utils.MoveToTrashBatch = original }()
	utils.MoveToTrashBatch = func(paths []string) utils.TrashBatchResult {
		return utils.TrashBatchResult{Succeeded: paths, Failed: make(map[string]error)}
	}

	registry := target.NewRegistry()
	cat := types.Category{
		ID:     "docker",
		Name:   "Docker",
		Method: types.MethodBuiltin,
	}

	mockTarget := newMockTargetForService(cat)
	cleanResult := types.NewCleanResult(cat)
	cleanResult.CleanedItems = 3
	cleanResult.FreedSpace = 300
	cleanResult.Errors = []string{}
	mockTarget.On("Clean", mock.Anything).Return(cleanResult, nil)
	registry.Register(mockTarget)

	service := NewCleanService(registry)

	jobs := []CleanJob{
		{
			Category: types.Category{ID: "cat1", Name: "Trash Cat", Method: types.MethodTrash},
			Items:    newTestItems("/trash1", "/trash2"),
		},
		{
			Category: cat,
			Items:    newTestItems("/docker1"),
		},
	}

	var progressCalls []Progress
	var categoryDoneCalls []CategoryResult

	callbacks := Callbacks{
		OnProgress: func(p Progress) {
			progressCalls = append(progressCalls, p)
		},
		OnCategoryDone: func(r CategoryResult) {
			categoryDoneCalls = append(categoryDoneCalls, r)
		},
	}

	report := service.Clean(jobs, callbacks)

	// Total items = 3, progress calls = 2 (trash batch start+end) + 1 (docker batch) = 3
	assert.Len(t, progressCalls, 3)

	// Category done called for each category
	assert.Len(t, categoryDoneCalls, 2)

	// Report aggregates all
	assert.Equal(t, 5, report.CleanedItems)
	assert.Len(t, report.Results, 2)
}

func TestClean_ReturnsCorrectReport(t *testing.T) {
	original := utils.MoveToTrashBatch
	defer func() { utils.MoveToTrashBatch = original }()
	utils.MoveToTrashBatch = func(paths []string) utils.TrashBatchResult {
		return utils.TrashBatchResult{Succeeded: paths, Failed: make(map[string]error)}
	}

	service := NewCleanService(target.NewRegistry())

	jobs := []CleanJob{
		{
			Category: types.Category{
				ID:     "cat1",
				Name:   "Test Category",
				Method: types.MethodTrash,
				Safety: types.SafetyLevelSafe,
			},
			Items: []types.CleanableItem{
				{Path: "/path1", Name: "File 1", Size: 1000},
				{Path: "/path2", Name: "File 2", Size: 2000},
			},
		},
	}

	report := service.Clean(jobs, Callbacks{})

	require.NotNil(t, report)
	assert.Equal(t, int64(3000), report.FreedSpace)
	assert.Equal(t, 2, report.CleanedItems)
	assert.Equal(t, 0, report.FailedItems)

	require.Len(t, report.Results, 1)
	assert.Equal(t, "cat1", report.Results[0].Category.ID)
	assert.Equal(t, int64(3000), report.Results[0].FreedSpace)
	assert.Equal(t, 2, report.Results[0].CleanedItems)
}
