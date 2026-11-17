package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"charm.land/lipgloss/v2"
	"github.com/2ykwang/mac-cleanup-go/internal/cleaner"
	"github.com/2ykwang/mac-cleanup-go/internal/styles"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/userconfig"
)

func newTestDensityReport() *types.Report {
	report := &types.Report{}
	for i := range 20 {
		report.AddResult(&types.CleanResult{
			Category:     types.Category{Name: fmt.Sprintf("Cache %02d", i), Method: types.MethodTrash},
			PlannedBytes: 1024, TrashedBytes: 1024, ProcessedItems: 1, CleanedItems: 1,
		})
	}
	report.AddResult(&types.CleanResult{
		Category:     types.Category{Name: "Docker", Method: types.MethodBuiltin, Disposal: types.DisposalPermanent},
		PlannedBytes: 4096, PermanentlyDeletedBytes: 2048, ProcessedItems: 1, CleanedItems: 3,
		AlreadyAbsentItems: 1, UnconfirmedItems: 1, SkippedItems: 1, Errors: []string{"permission denied"},
	})
	return report
}

func assertCompactReport(t *testing.T, width int) {
	t.Helper()
	t.Setenv("COLUMNS", fmt.Sprint(width))

	output := FormatReport(newTestDensityReport(), false, styles.New(true), false)
	_, details, found := strings.Cut(output, "Details\n")

	require.True(t, found)
	assert.LessOrEqual(t, len(strings.Split(strings.TrimSpace(details), "\n")), 30)
	assert.Equal(t, 1, strings.Count(details, "Docker"))
	assert.Contains(t, details, "Permanent")
	assert.Contains(t, details, "Already absent: 1")
	assert.Contains(t, details, "Unconfirmed: 1")
	assert.Contains(t, details, "Skipped: 1")
	assert.NotContains(t, details, "Trash: 0 B")
	assert.NotContains(t, details, "Skipped: 0")
	for _, line := range strings.Split(details, "\n") {
		assert.LessOrEqual(t, lipgloss.Width(line), width, line)
	}
	t.Logf("%d-column report:\n%s", width, output)
}

func TestReportDensity_Wide(t *testing.T) {
	assertCompactReport(t, 90)
}

func TestReportDensity_Narrow(t *testing.T) {
	assertCompactReport(t, 40)
}

func TestReportSummary_OmitsZeroOutcomes(t *testing.T) {
	report := &types.Report{PlannedBytes: 1024, TrashedBytes: 1024, ProcessedItems: 1}

	output := FormatReport(report, false, styles.New(true), false)

	assert.NotContains(t, output, "Deleted permanently (est.): 0")
	assert.NotContains(t, output, "Already absent:")
	assert.NotContains(t, output, "Unconfirmed:")
	assert.NotContains(t, output, "Skipped:")
}

func TestReportNotes_NoTrailingPadding(t *testing.T) {
	result := types.CleanResult{AlreadyAbsentItems: 1, UnconfirmedItems: 2, SkippedItems: 3}

	output := renderResultNotes(result, 40)

	assert.Contains(t, output, "Already absent: 1")
	assert.Contains(t, output, "Unconfirmed: 2")
	assert.Contains(t, output, "Skipped: 3")
	for _, line := range strings.Split(output, "\n") {
		assert.Equal(t, strings.TrimRight(line, " \t"), line)
	}
}

func TestDryRunReport_MixedActionsKeepSuccessBytesZero(t *testing.T) {
	jobs := []cleaner.CleanJob{
		{Category: types.Category{ID: "cache", Method: types.MethodTrash}, Items: []types.CleanableItem{{Size: 100}}},
		{Category: types.Category{ID: "docker", Method: types.MethodBuiltin, Disposal: types.DisposalPermanent}, Items: []types.CleanableItem{{Size: 200}}},
	}

	report := buildDryRunReport(jobs)
	plan := &cleanPlan{results: map[string]*types.ScanResult{
		"cache": {Items: jobs[0].Items}, "docker": {Items: jobs[1].Items},
	}}
	out := newCleanOutput(report, plan, true)

	assert.Equal(t, int64(300), out.PlannedBytes)
	assert.Equal(t, int64(300), out.FreedBytes)
	assert.Zero(t, out.TrashedBytes)
	assert.Zero(t, out.PermanentlyDeletedBytes)
	require.Len(t, out.Targets, 2)
	assert.Equal(t, types.DisposalTrash, out.Targets[0].Disposal)
	assert.Equal(t, types.DisposalPermanent, out.Targets[1].Disposal)
}

func TestCleanOutput_JSONKeepsLegacyBytesAndAddsExplicitOutcomes(t *testing.T) {
	report := &types.Report{}
	report.AddResult(&types.CleanResult{
		Category:   types.Category{ID: "cache", Method: types.MethodTrash},
		FreedSpace: 42, PlannedBytes: 42, AlreadyAbsentItems: 1, CleanedItems: 1,
	})
	var buf bytes.Buffer

	err := writeJSON(&buf, newCleanOutput(report, &cleanPlan{}, false))

	require.NoError(t, err)
	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(buf.Bytes(), &raw))
	assert.JSONEq(t, `42`, string(raw["freed_bytes"]))
	assert.JSONEq(t, `42`, string(raw["planned_bytes"]))
	assert.JSONEq(t, `0`, string(raw["trashed_bytes"]))
	assert.JSONEq(t, `0`, string(raw["permanently_deleted_bytes"]))
	assert.JSONEq(t, `1`, string(raw["already_absent_items"]))
	assert.Contains(t, string(raw["targets"]), `"freed_bytes":42`)
	assert.True(t, strings.HasPrefix(buf.String(), `{"dry_run":`))
	assert.Contains(t, buf.String(), `"targets":[{"id":"cache"`)
}

func TestCleanOutput_LegacySkipAliasMatchesPolicySkips(t *testing.T) {
	report := &types.Report{}
	report.AddResult(&types.CleanResult{Category: types.Category{ID: "cache"}, SkippedItems: 2})
	var buf bytes.Buffer

	err := writeJSON(&buf, newCleanOutput(report, &cleanPlan{}, false))

	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"sip_skipped_items":2`)
	assert.Contains(t, buf.String(), `"skipped_items":2`)
}

func TestConfirm_ExplainsBuiltinPermanentAndUnknownModes(t *testing.T) {
	var errOut bytes.Buffer
	a := &App{In: strings.NewReader("n\n"), Err: &errOut}
	jobs := []cleaner.CleanJob{
		{Category: types.Category{ID: "docker", Method: types.MethodBuiltin, Disposal: types.DisposalPermanent}},
		{Category: types.Category{ID: "other", Method: types.MethodBuiltin}},
	}

	accepted := a.confirm(&cleanPlan{}, jobs)

	assert.False(t, accepted)
	assert.Contains(t, errOut.String(), "docker: Delete permanently")
	assert.Contains(t, errOut.String(), "other: Unknown disposal")
}

func TestNewScanEntry_ExposesActualBuiltinDisposal(t *testing.T) {
	s := scanned{cat: types.Category{Method: types.MethodBuiltin, Disposal: types.DisposalTrash}}

	entry := newScanEntry(s, &userconfig.UserConfig{}, false, 0)

	assert.Equal(t, types.MethodBuiltin, entry.Method)
	assert.Equal(t, types.DisposalTrash, entry.Disposal)
}

func TestFormatReport_MixedOutcomesUseHonestLabels(t *testing.T) {
	t.Setenv("COLUMNS", "70")
	report := &types.Report{
		PlannedBytes: 4096, TrashedBytes: 1024, PermanentlyDeletedBytes: 2048,
		Results: []types.CleanResult{{Category: types.Category{Name: "Docker", Method: types.MethodBuiltin, Disposal: types.DisposalPermanent}, CleanedItems: 1, PermanentlyDeletedBytes: 2048}},
	}

	output := FormatReport(report, false, styles.New(false), false)

	assert.Contains(t, output, "Moved to Trash: 1 KB")
	assert.Contains(t, output, "Deleted permanently (est.): 2 KB")
	assert.Contains(t, output, "Permanent")
	assert.Contains(t, output, "not freed disk space")
	assert.NotContains(t, output, "Recovered:")
}
