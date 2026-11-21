package target

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func TestAgeFilter_StrictCutoffExcludesUndatedItems(t *testing.T) {
	cutoff := time.Unix(1000, 0)
	older := types.CleanableItem{ModifiedAt: cutoff.Add(-time.Nanosecond), Size: 4, FileCount: 2}
	result := &types.ScanResult{Items: []types.CleanableItem{
		{}, {ModifiedAt: cutoff}, older, {ModifiedAt: cutoff.Add(time.Nanosecond)},
	}, TotalSize: 50, TotalFileCount: 12}

	ageFilter{cutoff: cutoff}.apply(result)

	assert.Equal(t, []types.CleanableItem{older}, result.Items)
	assert.EqualValues(t, 4, result.TotalSize)
	assert.EqualValues(t, 2, result.TotalFileCount)
}
