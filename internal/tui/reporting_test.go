package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/styles"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func newTestReportingModel(dark bool, width int) *Model {
	m := newTestModel()
	m.styles = styles.New(dark)
	m.width, m.height = width, 24
	trash := &types.ScanResult{
		Category: types.Category{ID: "cache", Name: "Cache", Method: types.MethodTrash},
		Items:    []types.CleanableItem{{Path: "/fixture/cache", Size: 1024}},
	}
	docker := &types.ScanResult{
		Category: types.Category{ID: "docker", Name: "Docker", Method: types.MethodBuiltin, Disposal: types.DisposalPermanent},
		Items:    []types.CleanableItem{{Path: "docker:image:fixture", Size: 2048}},
	}
	m.results = []*types.ScanResult{trash, docker}
	m.resultMap = map[string]*types.ScanResult{"cache": trash, "docker": docker}
	m.selected = map[string]bool{"cache": true, "docker": true}
	m.report = &types.Report{}
	m.report.AddResult(&types.CleanResult{Category: trash.Category, PlannedBytes: 1024, TrashedBytes: 1024, CleanedItems: 1, ProcessedItems: 1})
	m.report.AddResult(&types.CleanResult{Category: docker.Category, PlannedBytes: 2048, PermanentlyDeletedBytes: 2048, CleanedItems: 1, ProcessedItems: 1})
	return m
}

func assertPlainData(t *testing.T, output, value string) {
	t.Helper()
	index := strings.Index(output, value)
	require.GreaterOrEqual(t, index, 0)
	sequences := regexp.MustCompile(`\x1b\[[0-9;]*m`).FindAllString(output[:index], -1)
	if len(sequences) == 0 {
		return
	}
	last := strings.TrimSuffix(strings.TrimPrefix(sequences[len(sequences)-1], "\x1b["), "m")
	assert.NotRegexp(t, `(^|;)(3[0-8]|9[0-7])(;|$)`, last, value)
}

func assertReportingDataColors(t *testing.T, dark bool) {
	t.Helper()
	m := newTestReportingModel(dark, 80)
	m.recentDeleted.Push(DeletedItemEntry{Path: "/fixture/cache", Size: 1024, Success: true})

	header := m.reportHeader()
	recent := m.renderRecentDeleted()

	assertPlainData(t, header, "Moved to Trash: 1 KB")
	assertPlainData(t, header, "Deleted permanently (est.): 2 KB")
	assertPlainData(t, header, "Succeeded: 2")
	assertPlainData(t, recent, "/fixture/cache")
	assertPlainData(t, recent, "1 KB")
}

func TestReportingDataColors_Dark(t *testing.T) {
	assertReportingDataColors(t, true)
}

func TestReportingDataColors_Light(t *testing.T) {
	assertReportingDataColors(t, false)
}

func TestReportDensity_CategoriesAppearOnceWithNonzeroExceptions(t *testing.T) {
	m := newTestReportingModel(true, 40)
	m.report = &types.Report{}
	for i := range 20 {
		m.report.AddResult(&types.CleanResult{
			Category:     types.Category{Name: fmt.Sprintf("Cache %02d", i), Method: types.MethodTrash},
			TrashedBytes: 1024, ProcessedItems: 1, CleanedItems: 1,
		})
	}
	m.report.AddResult(&types.CleanResult{Category: types.Category{Name: "Failed cache", Method: types.MethodTrash}, Errors: []string{"permission denied"}, SkippedItems: 1})

	lines := m.buildReportLines()
	output := ansi.Strip(strings.Join(lines, "\n"))

	assert.LessOrEqual(t, len(lines), 27)
	assert.Equal(t, 1, strings.Count(output, "Failed cache"))
	assert.Contains(t, output, "Skipped: 1")
	assert.NotContains(t, output, "absent 0")
	for _, line := range strings.Split(output, "\n") {
		assert.LessOrEqual(t, lipgloss.Width(line), m.width, line)
	}
}

func TestReportHeader_OmitsZeroOutcomes(t *testing.T) {
	m := newTestReportingModel(true, 40)

	output := ansi.Strip(m.reportHeader())

	assert.NotContains(t, output, "Already absent:")
	assert.NotContains(t, output, "Unconfirmed:")
	assert.NotContains(t, output, "Skipped:")
}

func TestConfirm_PermanentOnlyPreservesNameWithoutTrashRecovery(t *testing.T) {
	m := newTestReportingModel(true, 40)
	m.selected["cache"] = false

	output := ansi.Strip(m.confirmDialog())

	assert.Contains(t, output, "Docker")
	assert.Contains(t, output, "Delete permanently: Docker")
	assert.NotContains(t, output, "Recover Trash")
}

func TestConfirm_MixedSelectionAssociatesPermanentAction(t *testing.T) {
	m := newTestReportingModel(true, 80)

	output := ansi.Strip(m.confirmDialog())

	assert.Contains(t, output, "Delete permanently: Docker")
	assert.Contains(t, output, "Cache")
	assert.NotContains(t, output, "Delete permanently: Cache")
}

func TestConfirm_NarrowPreservesFullNonTrashNames(t *testing.T) {
	m := newTestReportingModel(true, 40)
	m.resultMap["docker"].Category.Name = "Docker Images And Unused Build Cache"

	output := ansi.Strip(m.confirmDialog())
	var content []string
	for _, line := range strings.Split(output, "\n") {
		content = append(content, strings.Trim(line, "│ "))
		assert.LessOrEqual(t, lipgloss.Width(line), m.width, line)
	}

	assert.Contains(t, strings.Join(content, " "), "Delete permanently: Docker Images And Unused Build Cache")
	assert.LessOrEqual(t, lipgloss.Height(output), m.height)
}

func assertReportingFixture(t *testing.T, m *Model) {
	t.Helper()
	confirm := ansi.Strip(m.confirmDialog())
	report := ansi.Strip(m.viewReport())

	assert.Contains(t, confirm, "Move to Trash")
	assert.Contains(t, confirm, "Delete permanently")
	assert.Contains(t, confirm, "Delete permanently: Docker")
	assert.Contains(t, confirm, "Planned (estimate): 3 KB")
	assert.NotContains(t, confirm, "Files will be moved to Trash")
	assert.Contains(t, confirm, "Cache")
	assert.Contains(t, confirm, "Docker")
	assert.Contains(t, report, "Moved to Trash: 1 KB")
	assert.Contains(t, report, "Deleted permanently (est.): 2 KB")
	assert.NotContains(t, report, "Freed:")
	assert.LessOrEqual(t, lipgloss.Height(confirm), m.height)
	for _, line := range strings.Split(confirm+"\n"+report, "\n") {
		assert.LessOrEqual(t, lipgloss.Width(line), m.width, line)
	}
	t.Logf("Confirm fixture:\n%s\nReport fixture:\n%s", confirm, report)
}

func assertNeutralRecentOutcome(t *testing.T, outcome types.ItemOutcome) {
	t.Helper()
	m := newTestReportingModel(true, 80)
	m.recentDeleted.Push(DeletedItemEntry{Path: "/fixture/cache", Outcome: outcome, ErrMsg: "permission denied"})

	output := ansi.Strip(m.renderRecentDeleted())

	assert.Contains(t, output, "-")
	assert.NotContains(t, output, "✗")
	assert.NotContains(t, output, "✓")
}

func TestRecentOutcome_SkippedIsNeutral(t *testing.T) {
	assertNeutralRecentOutcome(t, types.ItemOutcomeSkipped)
}

func TestRecentOutcome_AbsentIsNeutral(t *testing.T) {
	assertNeutralRecentOutcome(t, types.ItemOutcomeAbsent)
}

func TestRecentOutcome_UnconfirmedIsNeutral(t *testing.T) {
	assertNeutralRecentOutcome(t, types.ItemOutcomeUnconfirmed)
}

func TestReportingFixture_Dark(t *testing.T) {
	m := newTestReportingModel(true, 80)

	assertReportingFixture(t, m)
}

func TestReportingFixture_Light(t *testing.T) {
	m := newTestReportingModel(false, 80)

	assertReportingFixture(t, m)
}

func TestReportingFixture_DarkNarrow(t *testing.T) {
	m := newTestReportingModel(true, 40)

	assertReportingFixture(t, m)
}

func TestReportingFixture_LightNarrow(t *testing.T) {
	m := newTestReportingModel(false, 40)

	assertReportingFixture(t, m)
}

func TestConfirmActions_UnknownAndZeroSizeBuiltin(t *testing.T) {
	m := newTestReportingModel(true, 80)
	m.resultMap["docker"].Category.Disposal = ""
	m.resultMap["docker"].Items[0].Size = 0

	lines := m.confirmActions()

	assert.Contains(t, lines, "Unknown disposal: Docker")
	assert.NotContains(t, lines, "Delete permanently")
}

func TestConfirmActions_ExcludesLockedAndExcludedItems(t *testing.T) {
	m := newTestReportingModel(true, 80)
	m.resultMap["docker"].Items[0].Status = types.ItemStatusProcessLocked
	m.excluded["cache"] = map[string]bool{"/fixture/cache": true}

	lines := m.confirmActions()

	assert.Empty(t, lines)
}

func TestConfirmActions_HomebrewMixedIsExplicit(t *testing.T) {
	m := newTestReportingModel(true, 80)
	m.resultMap["docker"].Category.Disposal = types.DisposalMixed

	lines := m.confirmActions()

	assert.Contains(t, lines, "Trash + permanent deletion: Docker")
	assert.Contains(t, lines, "Extra cleanup size unknown")
}

func TestConfirm_MixedActionsFitNarrowTerminal(t *testing.T) {
	m := newTestReportingModel(true, 40)
	r := &types.ScanResult{
		Category: types.Category{ID: "homebrew", Name: "Homebrew", Method: types.MethodBuiltin, Disposal: types.DisposalMixed},
		Items:    []types.CleanableItem{{Path: "/fixture/brew", Size: 123}},
	}
	m.results = append(m.results, r)
	m.resultMap["homebrew"] = r
	m.selected["homebrew"] = true

	output := ansi.Strip(m.confirmDialog())

	assert.LessOrEqual(t, lipgloss.Height(output), m.height)
	assert.Contains(t, output, "Trash + permanent deletion")
	assert.Contains(t, output, "Extra cleanup size unknown")
	assert.Contains(t, output, "Cancel")
	assert.Contains(t, output, "Delete")
}

func TestReport_HomebrewWithNoConfirmedMovementStillExplainsMixedAction(t *testing.T) {
	m := newTestReportingModel(true, 80)
	m.report = &types.Report{Results: []types.CleanResult{{Category: types.Category{ID: "homebrew", Name: "Homebrew", Method: types.MethodBuiltin, Disposal: types.DisposalMixed}, AlreadyAbsentItems: 1}}}

	lines := strings.Join(m.buildReportLines(), "\n")

	assert.Contains(t, lines, "Trash + permanent deletion")
	assert.Contains(t, lines, "Additional cleanup size unknown")
	assert.NotContains(t, lines, "Succeeded:")
}
