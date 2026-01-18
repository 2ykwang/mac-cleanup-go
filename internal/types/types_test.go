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
