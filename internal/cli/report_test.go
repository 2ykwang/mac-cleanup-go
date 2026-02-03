package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func TestFormatReport_DryRunNoItems(t *testing.T) {
	t.Setenv("COLUMNS", "80")
	report := &types.Report{
		FreedSpace: 0,
		Results:    []types.CleanResult{},
		Duration:   50 * time.Millisecond,
	}

	output := FormatReport(report, true)

	assert.Contains(t, output, "Dry Run Report")
	assert.Contains(t, output, "Summary")
	assert.Contains(t, output, "Highlights")
	assert.Contains(t, output, "No items to clean.")
	assert.Contains(t, output, "Freed (dry-run)")
}

func TestFormatReport_IncludesGroups(t *testing.T) {
	t.Setenv("COLUMNS", "100")
	report := &types.Report{
		FreedSpace:   1024,
		CleanedItems: 2,
		FailedItems:  1,
		Results: []types.CleanResult{
			{
				Category:     types.Category{Name: "Cache"},
				CleanedItems: 1,
				FreedSpace:   1024,
			},
			{
				Category:     types.Category{Name: "Logs"},
				CleanedItems: 1,
				FreedSpace:   0,
				Errors:       []string{"failed to remove"},
			},
		},
	}

	output := FormatReport(report, false)

	assert.Contains(t, output, "Cleanup Report")
	assert.Contains(t, output, "Summary")
	assert.Contains(t, output, "Highlights")
	assert.Contains(t, output, "Details")
	assert.Contains(t, output, "STATUS")
	assert.Contains(t, output, "Recovered")
	assert.True(t, strings.Contains(output, "Cache") || strings.Contains(output, "Logs"))
	assert.Contains(t, output, "failed to remove")
}
