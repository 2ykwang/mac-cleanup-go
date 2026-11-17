package cleaner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/target"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func TestPermanent_AbsentDirectoryKeepsLegacyEstimate(t *testing.T) {
	item := types.CleanableItem{Path: filepath.Join(t.TempDir(), "absent"), Size: 123, IsDirectory: true}

	result := NewExecutor(nil).Permanent(types.Category{Method: types.MethodPermanent}, []types.CleanableItem{item})

	assert.Equal(t, int64(123), result.FreedSpace)
	assert.Zero(t, result.PermanentlyDeletedBytes)
	assert.Equal(t, 1, result.AlreadyAbsentItems)
	assert.Empty(t, result.Errors)
}

func TestTrash_MismatchKeepsPlannedBytes(t *testing.T) {
	items := []types.CleanableItem{{Size: 12}, {Size: 34}}

	result := NewExecutor(nil).Trash(types.Category{Method: types.MethodPermanent}, items)

	assert.Equal(t, int64(46), result.PlannedBytes)
	assert.Equal(t, 2, result.SkippedItems)
	assert.Zero(t, result.ProcessedItems)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "method mismatch")
}

func TestBuiltin_MismatchKeepsPlannedBytes(t *testing.T) {
	items := []types.CleanableItem{{Size: 12}, {Size: 34}}

	result := NewExecutor(nil).Builtin(types.Category{Method: types.MethodTrash}, items)

	assert.Equal(t, int64(46), result.PlannedBytes)
	assert.Equal(t, 2, result.SkippedItems)
	assert.Zero(t, result.ProcessedItems)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "method mismatch")
}

func TestManual_KeepsPlannedBytesWithoutCleanup(t *testing.T) {
	items := []types.CleanableItem{{Size: 12}, {Size: 34}}

	result := NewExecutor(nil).Manual(types.Category{Method: types.MethodManual}, items)

	assert.Equal(t, int64(46), result.PlannedBytes)
	assert.Equal(t, 2, result.SkippedItems)
	assert.Zero(t, result.ProcessedItems)
	assert.Empty(t, result.Errors)
}

func TestClean_UnsupportedMethodKeepsPlannedBytes(t *testing.T) {
	job := CleanJob{Category: types.Category{Method: "unsupported"}, Items: []types.CleanableItem{{Size: 12}, {Size: 34}}}

	report := NewCleanService(nil).Clean([]CleanJob{job}, types.CleanCallbacks{})

	require.Len(t, report.Results, 1)
	assert.Equal(t, int64(46), report.PlannedBytes)
	assert.Equal(t, int64(46), report.Results[0].PlannedBytes)
	assert.Equal(t, 1, report.FailedItems)
	assert.Zero(t, report.ProcessedItems)
}

func TestPermanent_AbsentRegularFileKeepsOnlyLegacyError(t *testing.T) {
	item := types.CleanableItem{Path: filepath.Join(t.TempDir(), "absent"), Size: 123}

	result := NewExecutor(nil).Permanent(types.Category{Method: types.MethodPermanent}, []types.CleanableItem{item})

	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "no such file or directory")
	assert.Zero(t, result.AlreadyAbsentItems)
	assert.Zero(t, result.CleanedItems)
	assert.Zero(t, result.FreedSpace)
	assert.Zero(t, result.PermanentlyDeletedBytes)
	assert.Equal(t, int64(123), result.PlannedBytes)
}

func TestBatchCallbacks_LookUpNamesOnceAndKeepFirstDuplicate(t *testing.T) {
	items := []types.CleanableItem{{Path: "/fixture/a", Name: "first"}, {Path: "/fixture/b", Name: "second"}, {Path: "/fixture/a", Name: "duplicate"}}
	result := &types.CleanResult{ItemResults: []types.ItemCleanedResult{{Path: "/fixture/b"}, {Path: "/fixture/a"}}}
	var names []string

	NewCleanService(nil).sendBatchItemCallbacks(items, result, types.CleanCallbacks{OnItemDone: func(result types.ItemCleanedResult) {
		names = append(names, result.Name)
	}})

	assert.Equal(t, []string{"second", "first"}, names)
}

func TestPermanent_AbsentDirectoryCallbackIsNeutral(t *testing.T) {
	item := types.CleanableItem{Path: filepath.Join(t.TempDir(), "absent"), IsDirectory: true, Size: 123}
	var outcome types.ItemCleanedResult

	report := NewCleanService(nil).Clean([]CleanJob{{Category: types.Category{Method: types.MethodPermanent}, Items: []types.CleanableItem{item}}},
		types.CleanCallbacks{OnItemDone: func(result types.ItemCleanedResult) { outcome = result }})

	assert.Equal(t, types.ItemOutcomeAbsent, outcome.Outcome)
	assert.False(t, outcome.Success)
	assert.Zero(t, outcome.Size)
	assert.Equal(t, int64(123), report.PlannedBytes)
}

func TestClean_MixedBuiltinAndPermanentOutcomes(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(file, []byte("abc"), 0o600))
	trashCat := types.Category{ID: "cache", Method: types.MethodBuiltin, Disposal: types.DisposalTrash}
	registry := target.NewRegistry()
	tgt := newMockTargetForService(trashCat)
	tgt.On("Clean", mock.Anything).Return(&types.CleanResult{
		Category: trashCat, FreedSpace: 100, TrashedBytes: 60,
		CleanedItems: 2, ProcessedItems: 1, AlreadyAbsentItems: 1, Errors: []string{"one failed"},
	}, nil)
	registry.Register(tgt)
	jobs := []CleanJob{
		{Category: trashCat, Items: []types.CleanableItem{{Size: 60}, {Size: 40}, {Size: 50}}},
		{Category: types.Category{ID: "permanent", Method: types.MethodPermanent}, Items: []types.CleanableItem{{Path: file, Size: 3}}},
	}
	var categories []types.CategoryCleanedResult

	report := NewCleanService(registry).Clean(jobs, types.CleanCallbacks{OnCategoryDone: func(result types.CategoryCleanedResult) {
		categories = append(categories, result)
	}})

	assert.Equal(t, int64(153), report.PlannedBytes)
	assert.Equal(t, int64(60), report.TrashedBytes)
	assert.Equal(t, int64(3), report.PermanentlyDeletedBytes)
	assert.Equal(t, int64(103), report.FreedSpace)
	assert.Equal(t, 2, report.ProcessedItems)
	assert.Equal(t, 1, report.FailedItems)
	require.Len(t, categories, 2)
	assert.Equal(t, int64(60), categories[0].TrashedBytes)
	assert.Equal(t, 1, report.AlreadyAbsentItems)
	assert.Equal(t, 1, categories[0].ProcessedItems)
	assert.Equal(t, int64(3), categories[1].PermanentlyDeletedBytes)
}

func TestBuiltin_UnknownOutcomesRemainUnconfirmed(t *testing.T) {
	cat := types.Category{ID: "unknown", Method: types.MethodBuiltin}
	registry := target.NewRegistry()
	tgt := newMockTargetForService(cat)
	tgt.On("Clean", mock.Anything).Return(&types.CleanResult{Category: cat, FreedSpace: 20, CleanedItems: 1}, nil)
	registry.Register(tgt)

	result := NewExecutor(registry).Builtin(cat, []types.CleanableItem{{Size: 20}})

	assert.Equal(t, int64(20), result.FreedSpace)
	assert.Equal(t, int64(20), result.PlannedBytes)
	assert.Zero(t, result.TrashedBytes)
	assert.Zero(t, result.PermanentlyDeletedBytes)
	assert.Equal(t, 1, result.UnconfirmedItems)
}

func TestClean_TrashSkippedCallbackDoesNotClaimSuccess(t *testing.T) {
	jobs := []CleanJob{{
		Category: types.Category{Method: types.MethodTrash},
		Items:    []types.CleanableItem{{Path: "/System/Library/Caches/fixture", Size: 123}},
	}}
	var outcomes []types.ItemCleanedResult

	report := NewCleanService(target.NewRegistry()).Clean(jobs, types.CleanCallbacks{OnItemDone: func(result types.ItemCleanedResult) {
		outcomes = append(outcomes, result)
	}})

	require.Len(t, outcomes, 1)
	assert.False(t, outcomes[0].Success)
	assert.Equal(t, types.ItemOutcomeSkipped, outcomes[0].Outcome)
	assert.Zero(t, outcomes[0].Size)
	assert.Equal(t, 1, report.SkippedItems)
	assert.Equal(t, int64(123), report.PlannedBytes)
	assert.Zero(t, report.TrashedBytes)
}

func TestPermanent_PartialFailureCountsOnlySuccessfulSize(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(file, []byte("abc"), 0o600))
	items := []types.CleanableItem{{Path: file, Size: 3}, {Path: filepath.Join(dir, "absent"), Size: 10}}

	result := NewExecutor(nil).Permanent(types.Category{Method: types.MethodPermanent}, items)

	assert.Equal(t, int64(3), result.PermanentlyDeletedBytes)
	assert.Equal(t, int64(3), result.FreedSpace)
	assert.Zero(t, result.AlreadyAbsentItems)
	assert.Len(t, result.Errors, 1)
}
