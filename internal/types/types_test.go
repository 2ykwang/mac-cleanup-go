package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortOrder_Next_FromSize(t *testing.T) {
	result := SortBySize.Next()

	assert.Equal(t, SortByName, result)
}

func TestSortOrder_Next_FromName(t *testing.T) {
	result := SortByName.Next()

	assert.Equal(t, SortByAge, result)
}

func TestSortOrder_Next_FromAge(t *testing.T) {
	result := SortByAge.Next()

	assert.Equal(t, SortBySize, result)
}

func TestSortOrder_Label_Size(t *testing.T) {
	result := SortBySize.Label()

	assert.Equal(t, "Size ↓", result)
}

func TestSortOrder_Label_Name(t *testing.T) {
	result := SortByName.Label()

	assert.Equal(t, "Name", result)
}

func TestSortOrder_Label_Age(t *testing.T) {
	result := SortByAge.Label()

	assert.Equal(t, "Age", result)
}

func TestSortOrder_Label_Unknown(t *testing.T) {
	unknown := SortOrder("unknown")

	result := unknown.Label()

	assert.Equal(t, "Size ↓", result)
}

func TestCleanResult_Merge_AccumulatesAllFields(t *testing.T) {
	r := &CleanResult{
		CleanedItems: 2,
		SkippedItems: 1,
		FreedSpace:   100,
		PlannedBytes: 1000, TrashedBytes: 60, PermanentlyDeletedBytes: 40,
		AlreadyAbsentItems: 1, UnconfirmedItems: 2,
		Errors: []string{"err1"},
	}
	other := &CleanResult{
		CleanedItems: 3,
		SkippedItems: 2,
		FreedSpace:   200,
		PlannedBytes: 2000, TrashedBytes: 120, PermanentlyDeletedBytes: 80,
		AlreadyAbsentItems: 2, UnconfirmedItems: 3,
		Errors: []string{"err2", "err3"},
	}

	r.Merge(other)

	assert.Equal(t, 5, r.CleanedItems)
	assert.Equal(t, 3, r.SkippedItems)
	assert.Equal(t, int64(300), r.FreedSpace)
	assert.Equal(t, int64(3000), r.PlannedBytes)
	assert.Equal(t, int64(180), r.TrashedBytes)
	assert.Equal(t, int64(120), r.PermanentlyDeletedBytes)
	assert.Equal(t, 3, r.AlreadyAbsentItems)
	assert.Equal(t, 5, r.UnconfirmedItems)
	assert.Equal(t, []string{"err1", "err2", "err3"}, r.Errors)
}

func TestCategory_DisposalMode_UnknownBuiltinStaysUnknown(t *testing.T) {
	category := Category{Method: MethodBuiltin}

	mode := category.DisposalMode()

	assert.Equal(t, DisposalUnknown, mode)
	assert.Equal(t, "Unknown disposal", mode.Label())
}

func TestReport_AddResult_AccumulatesOutcomeSizes(t *testing.T) {
	report := &Report{}
	trash := &CleanResult{PlannedBytes: 100, TrashedBytes: 60, FreedSpace: 80, CleanedItems: 2, AlreadyAbsentItems: 1}
	permanent := &CleanResult{PlannedBytes: 300, PermanentlyDeletedBytes: 200, FreedSpace: 200, CleanedItems: 1, SkippedItems: 1, Errors: []string{"failed"}}

	report.AddResult(trash)
	report.AddResult(permanent)

	assert.Equal(t, int64(400), report.PlannedBytes)
	assert.Equal(t, int64(60), report.TrashedBytes)
	assert.Equal(t, int64(200), report.PermanentlyDeletedBytes)
	assert.Equal(t, int64(280), report.FreedSpace)
	assert.Equal(t, 1, report.AlreadyAbsentItems)
	assert.Equal(t, 1, report.SkippedItems)
	assert.Equal(t, 1, report.FailedItems)
	assert.Len(t, report.Results, 2)
}

func TestCleanResult_Merge_EmptyOther(t *testing.T) {
	r := &CleanResult{
		CleanedItems: 2,
		FreedSpace:   100,
		Errors:       []string{"err1"},
	}
	other := &CleanResult{Errors: []string{}}

	r.Merge(other)

	assert.Equal(t, 2, r.CleanedItems)
	assert.Equal(t, int64(100), r.FreedSpace)
	assert.Equal(t, []string{"err1"}, r.Errors)
}

func TestCleanResult_Merge_NilErrors(t *testing.T) {
	r := &CleanResult{}
	other := &CleanResult{
		CleanedItems: 1,
		Errors:       []string{"err1"},
	}

	r.Merge(other)

	assert.Equal(t, 1, r.CleanedItems)
	assert.Equal(t, []string{"err1"}, r.Errors)
}

func TestNewScanResult_InitializesDefaults(t *testing.T) {
	category := Category{ID: "cat-id", Name: "Category"}

	result := NewScanResult(category)

	assert.Equal(t, category, result.Category)
	assert.NotNil(t, result.Items)
	assert.Len(t, result.Items, 0)
	assert.Zero(t, result.TotalSize)
	assert.Zero(t, result.TotalFileCount)
	assert.Nil(t, result.Error)
}

func TestNewCleanResult_InitializesDefaults(t *testing.T) {
	category := Category{ID: "cat-id", Name: "Category"}

	result := NewCleanResult(category)

	assert.Equal(t, category, result.Category)
	assert.NotNil(t, result.Errors)
	assert.Len(t, result.Errors, 0)
	assert.Zero(t, result.CleanedItems)
	assert.Zero(t, result.SkippedItems)
	assert.Zero(t, result.FreedSpace)
	assert.Zero(t, result.PlannedBytes)
	assert.Zero(t, result.TrashedBytes)
	assert.Zero(t, result.PermanentlyDeletedBytes)
	assert.Zero(t, result.AlreadyAbsentItems)
	assert.Zero(t, result.UnconfirmedItems)
	assert.Zero(t, result.ProcessedItems)
	assert.Empty(t, result.ItemResults)
}
