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
		Errors:       []string{"err1"},
	}
	other := &CleanResult{
		CleanedItems: 3,
		SkippedItems: 2,
		FreedSpace:   200,
		Errors:       []string{"err2", "err3"},
	}

	r.Merge(other)

	assert.Equal(t, 5, r.CleanedItems)
	assert.Equal(t, 3, r.SkippedItems)
	assert.Equal(t, int64(300), r.FreedSpace)
	assert.Equal(t, []string{"err1", "err2", "err3"}, r.Errors)
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
}
