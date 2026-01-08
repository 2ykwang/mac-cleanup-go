package tui

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/userconfig"
)

// Test fixtures

func newTestModel() *Model {
	return &Model{
		results:        make([]*types.ScanResult, 0),
		resultMap:      make(map[string]*types.ScanResult),
		selected:       make(map[string]bool),
		excluded:       make(map[string]map[string]bool),
		drillDownStack: make([]drillDownState, 0),
		view:           ViewList,
		width:          80,
		height:         24,
		userConfig:     &userconfig.UserConfig{ExcludedPaths: make(map[string][]string)},
		recentDeleted:  NewRingBuffer[DeletedItemEntry](defaultRecentItemsCapacity),
	}
}

func newTestModelWithResults() *Model {
	m := newTestModel()
	m.results = []*types.ScanResult{
		{
			Category:  types.Category{ID: "cat1", Name: "Chrome Cache", Safety: types.SafetyLevelSafe},
			TotalSize: 1000,
			Items:     []types.CleanableItem{{Path: "/path/1", Size: 500}, {Path: "/path/2", Size: 500}},
		},
		{
			Category:  types.Category{ID: "cat2", Name: "npm Cache", Safety: types.SafetyLevelSafe},
			TotalSize: 2000,
			Items:     []types.CleanableItem{{Path: "/path/3", Size: 2000}},
		},
		{
			Category:  types.Category{ID: "cat3", Name: "Xcode Archives", Safety: types.SafetyLevelRisky},
			TotalSize: 3000,
			Items:     []types.CleanableItem{{Path: "/path/4", Size: 3000}},
		},
	}
	for _, r := range m.results {
		m.resultMap[r.Category.ID] = r
	}
	return m
}

func newTestModelWithManualCategory() *Model {
	m := newTestModel()
	m.results = []*types.ScanResult{
		{
			Category:  types.Category{ID: "cat1", Name: "Chrome Cache", Safety: types.SafetyLevelSafe},
			TotalSize: 1000,
			Items:     []types.CleanableItem{{Path: "/path/1", Size: 500}, {Path: "/path/2", Size: 500}},
		},
		{
			Category:  types.Category{ID: "cat2", Name: "npm Cache", Safety: types.SafetyLevelSafe},
			TotalSize: 2000,
			Items:     []types.CleanableItem{{Path: "/path/3", Size: 2000}},
		},
		{
			Category:  types.Category{ID: "manual-cat", Name: "Telegram DB", Safety: types.SafetyLevelRisky, Method: types.MethodManual},
			TotalSize: 5000,
			Items:     []types.CleanableItem{{Path: "/path/5", Size: 5000}},
		},
		{
			Category:  types.Category{ID: "cat3", Name: "Xcode Archives", Safety: types.SafetyLevelRisky},
			TotalSize: 3000,
			Items:     []types.CleanableItem{{Path: "/path/4", Size: 3000}},
		},
	}
	for _, r := range m.results {
		m.resultMap[r.Category.ID] = r
	}
	return m
}

// Navigation tests

func TestHandleListKey_CursorDown(t *testing.T) {
	m := newTestModelWithResults()

	m.handleListKey(tea.KeyMsg{Type: tea.KeyDown})

	assert.Equal(t, 1, m.cursor)
}

func TestHandleListKey_CursorUp(t *testing.T) {
	m := newTestModelWithResults()
	m.cursor = 2

	m.handleListKey(tea.KeyMsg{Type: tea.KeyUp})

	assert.Equal(t, 1, m.cursor)
}

func TestHandleListKey_CursorBoundsTop(t *testing.T) {
	m := newTestModelWithResults()
	m.cursor = 0

	m.handleListKey(tea.KeyMsg{Type: tea.KeyUp})

	assert.Equal(t, 0, m.cursor, "cursor should not go below 0")
}

func TestHandleListKey_CursorBoundsBottom(t *testing.T) {
	m := newTestModelWithResults()
	m.cursor = len(m.results) - 1

	m.handleListKey(tea.KeyMsg{Type: tea.KeyDown})

	assert.Equal(t, len(m.results)-1, m.cursor, "cursor should not exceed results length")
}

// Selection tests

func TestHandleListKey_ToggleSelection(t *testing.T) {
	m := newTestModelWithResults()
	m.cursor = 0

	// Select
	m.handleListKey(tea.KeyMsg{Type: tea.KeySpace})
	assert.True(t, m.selected["cat1"])

	// Deselect
	m.handleListKey(tea.KeyMsg{Type: tea.KeySpace})
	assert.False(t, m.selected["cat1"])
}

func TestHandleListKey_SelectAll(t *testing.T) {
	m := newTestModelWithResults()

	m.handleListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	assert.True(t, m.selected["cat1"])
	assert.True(t, m.selected["cat2"])
	assert.True(t, m.selected["cat3"])
}

func TestHandleListKey_DeselectAll(t *testing.T) {
	m := newTestModelWithResults()
	m.selected["cat1"] = true
	m.selected["cat2"] = true

	m.handleListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	assert.False(t, m.selected["cat1"])
	assert.False(t, m.selected["cat2"])
}

func TestHandleListKey_RiskyAutoExclude(t *testing.T) {
	m := newTestModelWithResults()
	m.cursor = 2 // Xcode Archives (risky)

	m.handleListKey(tea.KeyMsg{Type: tea.KeySpace})

	assert.True(t, m.selected["cat3"])
	assert.True(t, m.excluded["cat3"]["/path/4"], "risky items should be auto-excluded")
}

// Helper function tests

func TestHasSelection(t *testing.T) {
	m := newTestModel()

	assert.False(t, m.hasSelection())

	m.selected["cat1"] = true
	assert.True(t, m.hasSelection())
}

func TestGetSelectedCount(t *testing.T) {
	m := newTestModel()
	m.selected["cat1"] = true
	m.selected["cat2"] = true
	m.selected["cat3"] = false

	assert.Equal(t, 2, m.getSelectedCount())
}

func TestGetSelectedSize(t *testing.T) {
	m := newTestModelWithResults()
	m.selected["cat1"] = true
	m.selected["cat2"] = true

	assert.Equal(t, int64(3000), m.getSelectedSize())
}

func TestGetSelectedSize_WithExclusions(t *testing.T) {
	m := newTestModelWithResults()
	m.selected["cat1"] = true
	m.excluded["cat1"] = map[string]bool{"/path/1": true}

	// cat1 has 2 items: /path/1 (500) excluded, /path/2 (500) included
	assert.Equal(t, int64(500), m.getSelectedSize())
}

func TestGetEffectiveSize(t *testing.T) {
	m := newTestModelWithResults()
	r := m.results[0] // cat1 with 1000 total

	// No exclusions
	assert.Equal(t, int64(1000), m.getEffectiveSize(r))

	// With exclusion
	m.excluded["cat1"] = map[string]bool{"/path/1": true}
	assert.Equal(t, int64(500), m.getEffectiveSize(r))
}

func TestGetSelectedResults(t *testing.T) {
	m := newTestModelWithResults()
	m.selected["cat1"] = true
	m.selected["cat3"] = true

	selected := m.getSelectedResults()

	assert.Len(t, selected, 2)
	assert.Equal(t, "cat1", selected[0].Category.ID)
	assert.Equal(t, "cat3", selected[1].Category.ID)
}

// Exclusion tests

func TestIsExcluded(t *testing.T) {
	m := newTestModel()

	assert.False(t, m.isExcluded("cat1", "/path/1"))

	m.excluded["cat1"] = map[string]bool{"/path/1": true}
	assert.True(t, m.isExcluded("cat1", "/path/1"))
	assert.False(t, m.isExcluded("cat1", "/path/2"))
}

func TestToggleExclude(t *testing.T) {
	m := newTestModel()
	m.userConfig = &userconfig.UserConfig{ExcludedPaths: make(map[string][]string)}

	m.toggleExclude("cat1", "/path/1")
	assert.True(t, m.excluded["cat1"]["/path/1"])

	m.toggleExclude("cat1", "/path/1")
	assert.False(t, m.excluded["cat1"]["/path/1"])
}

// View state tests

func TestHandleListKey_EnterPreview(t *testing.T) {
	m := newTestModelWithResults()
	m.selected["cat1"] = true

	m.handleListKey(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Equal(t, ViewPreview, m.view)
	assert.Equal(t, "cat1", m.previewCatID)
}

func TestHandleListKey_EnterPreview_NoSelection(t *testing.T) {
	m := newTestModelWithResults()

	m.handleListKey(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Equal(t, ViewList, m.view, "should stay in list view when nothing selected")
}

func TestHandlePreviewKey_Back(t *testing.T) {
	m := newTestModelWithResults()
	m.view = ViewPreview
	m.selected["cat1"] = true
	m.previewCatID = "cat1"

	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeyEsc})

	assert.Equal(t, ViewList, m.view)
}

func TestHandlePreviewKey_Confirm(t *testing.T) {
	m := newTestModelWithResults()
	m.view = ViewPreview
	m.selected["cat1"] = true
	m.previewCatID = "cat1"

	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	assert.Equal(t, ViewConfirm, m.view)
}

// Open in Finder tests
func TestHandlePreviewKey_OpenInFinder_NonExistentPath(t *testing.T) {
	m := newTestModelWithResults()
	m.view = ViewPreview
	m.selected["cat1"] = true
	m.previewCatID = "cat1"
	m.previewItemIndex = 0 // /path/1 (non-existent)

	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	assert.Equal(t, "Path not found", m.statusMessage)
}

func TestHandlePreviewKey_OpenInFinder_ClearsStatusOnSuccess(t *testing.T) {
	m := newTestModelWithResults()
	m.view = ViewPreview
	m.selected["cat1"] = true
	m.previewCatID = "cat1"
	m.previewItemIndex = 0
	m.statusMessage = "Previous error" // Pre-existing status

	// This will fail because path doesn't exist, but tests the flow
	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	// Since path doesn't exist, it should set error message
	assert.Equal(t, "Path not found", m.statusMessage)
}

func TestHandleDrillDownKey_OpenInFinder_NonExistentPath(t *testing.T) {
	m := newTestModelWithResults()
	m.view = ViewPreview
	m.selected["cat1"] = true
	m.previewCatID = "cat1"
	// Set up drill down state with non-existent path
	m.drillDownStack = []drillDownState{
		{
			path:   "/nonexistent/dir",
			items:  []types.CleanableItem{{Path: "/nonexistent/file", Name: "file", Size: 100, IsDirectory: false}},
			cursor: 0,
			scroll: 0,
		},
	}

	m.handleDrillDownKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	assert.Equal(t, "Path not found", m.statusMessage)
}

// Preview navigation tests

func TestFindPrevSelectedCatID(t *testing.T) {
	m := newTestModelWithResults()
	m.selected["cat1"] = true
	m.selected["cat2"] = true
	m.previewCatID = "cat2"

	assert.Equal(t, "cat1", m.findPrevSelectedCatID())
}

func TestFindNextSelectedCatID(t *testing.T) {
	m := newTestModelWithResults()
	m.selected["cat1"] = true
	m.selected["cat2"] = true
	m.previewCatID = "cat1"

	assert.Equal(t, "cat2", m.findNextSelectedCatID())
}

func TestFindSelectedCatIndex(t *testing.T) {
	m := newTestModelWithResults()
	m.selected["cat1"] = true
	m.selected["cat3"] = true
	m.previewCatID = "cat3"

	assert.Equal(t, 1, m.findSelectedCatIndex())
}

// Layout helper tests

func TestCountLines(t *testing.T) {
	assert.Equal(t, 0, countLines(""))
	assert.Equal(t, 1, countLines("hello"))
	assert.Equal(t, 2, countLines("hello\nworld"))
	assert.Equal(t, 3, countLines("a\nb\nc"))
}

func TestAvailableLines(t *testing.T) {
	m := newTestModel()
	m.height = 30

	header := "line1\nline2\nline3\n" // 4 lines
	footer := "footer1\nfooter2"      // 2 lines

	available := m.availableLines(header, footer)
	assert.Equal(t, 30-6, available)
}

func TestAvailableLines_Minimum(t *testing.T) {
	m := newTestModel()
	m.height = 5

	header := "line1\nline2\nline3\n"
	footer := "footer1\nfooter2"

	available := m.availableLines(header, footer)
	assert.Equal(t, 3, available, "should return minimum of 3")
}

func TestAdjustScrollFor(t *testing.T) {
	m := newTestModel()

	// Cursor at top, scroll should follow
	assert.Equal(t, 0, m.adjustScrollFor(0, 5, 10, 20))

	// Cursor below visible area
	assert.Equal(t, 6, m.adjustScrollFor(15, 0, 10, 20))

	// Cursor within visible area
	assert.Equal(t, 5, m.adjustScrollFor(10, 5, 10, 20))
}

// View output tests

func TestViewList_ContainsHeader(t *testing.T) {
	m := newTestModelWithResults()

	output := m.viewList()

	assert.Contains(t, output, "Mac Cleanup")
}

func TestViewList_ContainsLegend(t *testing.T) {
	m := newTestModelWithResults()

	output := m.viewList()

	assert.Contains(t, output, "Safe")
	assert.Contains(t, output, "Moderate")
	assert.Contains(t, output, "Risky")
}

func TestViewList_ContainsResults(t *testing.T) {
	m := newTestModelWithResults()

	output := m.viewList()

	assert.Contains(t, output, "Chrome Cache")
	assert.Contains(t, output, "npm Cache")
	assert.Contains(t, output, "Xcode Archives")
}

func TestViewList_ContainsHelpText(t *testing.T) {
	m := newTestModelWithResults()

	output := m.viewList()

	// Footer now uses simplified format: "↑↓ Move  space Select  enter Preview  y Delete  ? Help"
	assert.Contains(t, output, "Move")
	assert.Contains(t, output, "Select")
	assert.Contains(t, output, "? Help")
}

func TestViewList_ShowsSelectedIndicator(t *testing.T) {
	m := newTestModelWithResults()
	m.selected["cat1"] = true

	output := m.viewList()

	assert.Contains(t, output, "[✓]")
	assert.Contains(t, output, "Selected:")
}

func TestViewList_EmptyResults(t *testing.T) {
	m := newTestModel()

	output := m.viewList()

	assert.Contains(t, output, "No items to clean")
}

func TestViewPreview_ContainsHeader(t *testing.T) {
	m := newTestModelWithResults()
	m.view = ViewPreview
	m.selected["cat1"] = true
	m.previewCatID = "cat1"

	output := m.viewPreview()

	assert.Contains(t, output, "Cleanup Preview")
	assert.Contains(t, output, "Selected:")
}

func TestViewPreview_ContainsCategoryInfo(t *testing.T) {
	m := newTestModelWithResults()
	m.view = ViewPreview
	m.selected["cat1"] = true
	m.previewCatID = "cat1"

	output := m.viewPreview()

	assert.Contains(t, output, "Chrome Cache")
	assert.Contains(t, output, "[Safe]")
}

func TestViewPreview_ContainsItems(t *testing.T) {
	m := newTestModelWithResults()
	m.view = ViewPreview
	m.selected["cat1"] = true
	m.previewCatID = "cat1"

	output := m.viewPreview()

	assert.Contains(t, output, "/path/1")
	assert.Contains(t, output, "/path/2")
}

func TestViewPreview_NoSelection(t *testing.T) {
	m := newTestModelWithResults()
	m.view = ViewPreview

	output := m.viewPreview()

	assert.Contains(t, output, "No items selected")
}

func TestViewConfirm_ContainsWarning(t *testing.T) {
	m := newTestModelWithResults()
	m.view = ViewConfirm
	m.selected["cat1"] = true

	output := m.viewConfirm()

	assert.Contains(t, output, "Confirm Deletion")
	assert.Contains(t, output, "Trash")
	assert.Contains(t, output, "Chrome Cache")
}

func TestViewReport_ContainsSummary(t *testing.T) {
	m := newTestModel()
	m.view = ViewReport
	m.report = &types.Report{
		FreedSpace:   1024 * 1024 * 100, // 100 MB
		CleanedItems: 10,
		FailedItems:  2,
	}

	output := m.viewReport()

	assert.Contains(t, output, "Cleanup Complete")
	assert.Contains(t, output, "Freed:")
	assert.Contains(t, output, "Succeeded:")
	assert.Contains(t, output, "10")
}

// cleanItemDoneMsg handling tests
func TestUpdate_CleanItemDoneMsg_AddsToRecentDeleted(t *testing.T) {
	m := newTestModel()
	m.view = ViewCleaning
	m.cleanProgressChan = make(chan cleanProgressMsg, 1)
	m.cleanItemDoneChan = make(chan cleanItemDoneMsg, 1)
	m.cleanCategoryDoneCh = make(chan cleanCategoryDoneMsg, 1)
	m.cleanDoneChan = make(chan cleanDoneMsg, 1)

	msg := cleanItemDoneMsg{
		path:    "/path/to/file.txt",
		name:    "file.txt",
		size:    1024,
		success: true,
		errMsg:  "",
	}

	m.Update(msg)

	assert.Equal(t, 1, m.recentDeleted.Len())
	items := m.recentDeleted.Items()
	assert.Equal(t, "file.txt", items[0].Name)
	assert.Equal(t, int64(1024), items[0].Size)
	assert.True(t, items[0].Success)
}

func TestUpdate_CleanItemDoneMsg_FailedItem(t *testing.T) {
	m := newTestModel()
	m.view = ViewCleaning
	m.cleanProgressChan = make(chan cleanProgressMsg, 1)
	m.cleanItemDoneChan = make(chan cleanItemDoneMsg, 1)
	m.cleanCategoryDoneCh = make(chan cleanCategoryDoneMsg, 1)
	m.cleanDoneChan = make(chan cleanDoneMsg, 1)

	msg := cleanItemDoneMsg{
		path:    "/path/to/locked.txt",
		name:    "locked.txt",
		size:    512,
		success: false,
		errMsg:  "permission denied",
	}

	m.Update(msg)

	items := m.recentDeleted.Items()
	assert.Equal(t, 1, len(items))
	assert.False(t, items[0].Success)
	assert.Equal(t, "permission denied", items[0].ErrMsg)
}

func TestUpdate_CleanCategoryDoneMsg_RecentDeletedPersists(t *testing.T) {
	m := newTestModel()
	m.view = ViewCleaning
	m.cleanProgressChan = make(chan cleanProgressMsg, 1)
	m.cleanItemDoneChan = make(chan cleanItemDoneMsg, 1)
	m.cleanCategoryDoneCh = make(chan cleanCategoryDoneMsg, 1)
	m.cleanDoneChan = make(chan cleanDoneMsg, 1)

	// Add items to recentDeleted first
	m.recentDeleted.Push(DeletedItemEntry{
		Path:    "/path/to/file1.txt",
		Name:    "file1.txt",
		Size:    1024,
		Success: true,
	})
	m.recentDeleted.Push(DeletedItemEntry{
		Path:    "/path/to/file2.txt",
		Name:    "file2.txt",
		Size:    2048,
		Success: true,
	})

	// Send category done message
	msg := cleanCategoryDoneMsg{
		categoryName: "Test Category",
		freedSpace:   3072,
		cleanedItems: 2,
		errorCount:   0,
	}

	m.Update(msg)

	// recentDeleted should still have the items
	assert.Equal(t, 2, m.recentDeleted.Len(), "recentDeleted should persist after cleanCategoryDoneMsg")
	items := m.recentDeleted.Items()
	assert.Equal(t, "file1.txt", items[0].Name)
	assert.Equal(t, "file2.txt", items[1].Name)
}

func TestUpdate_CleanDoneMsg_ClearsRecentDeleted(t *testing.T) {
	m := newTestModel()
	m.view = ViewCleaning

	// Add items to recentDeleted
	m.recentDeleted.Push(DeletedItemEntry{
		Path:    "/path/to/file.txt",
		Name:    "file.txt",
		Size:    1024,
		Success: true,
	})

	assert.Equal(t, 1, m.recentDeleted.Len(), "should have 1 item before cleanDoneMsg")

	// Send done message
	msg := cleanDoneMsg{
		report: &types.Report{
			FreedSpace:   1024,
			CleanedItems: 1,
			FailedItems:  0,
		},
	}

	m.Update(msg)

	// recentDeleted should be cleared
	assert.Equal(t, 0, m.recentDeleted.Len(), "recentDeleted should be cleared after cleanDoneMsg")
	assert.Equal(t, ViewReport, m.view, "view should change to ViewReport")
}

// Manual item tests
func TestRenderListItem_ManualItemMuted(t *testing.T) {
	m := newTestModelWithManualCategory()

	manualResult := m.results[2]
	assert.Equal(t, types.MethodManual, manualResult.Category.Method)

	output := m.renderListItem(2, manualResult)

	assert.Contains(t, output, "[Manual]")
	assert.Contains(t, output, "Telegram DB")

	mutedCheckbox := MutedStyle.Render(" - ")
	assert.Contains(t, output, mutedCheckbox, "manual item checkbox should be rendered with MutedStyle")

	m.selected[manualResult.Category.ID] = true
	outputSelected := m.renderListItem(2, manualResult)
	assert.NotContains(t, outputSelected, "[✓]", "manual item should never show checked indicator")
}

func TestHandleListKey_ManualNotSelectable(t *testing.T) {
	m := newTestModelWithManualCategory()
	m.cursor = 2 // Manual category (Telegram DB)

	assert.Equal(t, types.MethodManual, m.results[m.cursor].Category.Method)

	m.handleListKey(tea.KeyMsg{Type: tea.KeySpace})

	assert.False(t, m.selected["manual-cat"], "manual category should not be selectable via Space key")
}

func TestHandleListKey_SelectAll_ExcludesManual(t *testing.T) {
	m := newTestModelWithManualCategory()

	m.handleListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	assert.True(t, m.selected["cat1"], "non-manual category should be selected")
	assert.True(t, m.selected["cat2"], "non-manual category should be selected")
	assert.True(t, m.selected["cat3"], "non-manual category should be selected")

	assert.False(t, m.selected["manual-cat"], "manual category should be excluded from select all")
}

// Guide popup tests (Manual item)
func TestHandleListKey_ManualSpaceOpensGuide(t *testing.T) {
	m := newTestModelWithManualCategory()
	m.cursor = 2 // Manual category (Telegram DB)

	assert.Equal(t, types.MethodManual, m.results[m.cursor].Category.Method)

	m.handleListKey(tea.KeyMsg{Type: tea.KeySpace})

	assert.Equal(t, ViewGuide, m.view, "Space on Manual item should open guide popup")
	assert.NotNil(t, m.guideCategory, "guideCategory should be set")
	assert.Equal(t, "manual-cat", m.guideCategory.ID, "guideCategory should reference the Manual category")
}

func TestHandleGuideKey_EscReturnsToList(t *testing.T) {
	m := newTestModelWithManualCategory()
	m.view = ViewGuide
	m.guideCategory = &m.results[2].Category
	m.guidePathIndex = 1

	m.handleGuideKey(tea.KeyMsg{Type: tea.KeyEsc})

	assert.Equal(t, ViewList, m.view, "Esc should return to list view")
	assert.Nil(t, m.guideCategory, "guideCategory should be cleared")
	assert.Equal(t, 0, m.guidePathIndex, "guidePathIndex should be reset")
}

func TestHandleGuideKey_EnterReturnsToList(t *testing.T) {
	m := newTestModelWithManualCategory()
	m.view = ViewGuide
	m.guideCategory = &m.results[2].Category

	m.handleGuideKey(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Equal(t, ViewList, m.view, "Enter should return to list view")
	assert.Nil(t, m.guideCategory, "guideCategory should be cleared")
}

func TestHandleGuideKey_SpaceReturnsToList(t *testing.T) {
	m := newTestModelWithManualCategory()
	m.view = ViewGuide
	m.guideCategory = &m.results[2].Category

	m.handleGuideKey(tea.KeyMsg{Type: tea.KeySpace})

	assert.Equal(t, ViewList, m.view, "Space should return to list view")
}

func TestViewGuide_EmptyGuideShowsDefaultMessage(t *testing.T) {
	m := newTestModel()
	m.width = 80
	m.height = 24
	m.view = ViewGuide
	m.guideCategory = &types.Category{
		ID:     "test-manual",
		Name:   "Test Manual Item",
		Method: types.MethodManual,
		Guide:  "", // Empty guide
		Paths:  []string{"~/Library/Test"},
	}

	output := m.viewGuide()

	assert.Contains(t, output, "Test Manual Item", "should show category name")
	assert.Contains(t, output, "must be deleted manually", "should show default message when guide is empty")
	assert.Contains(t, output, "~/Library/Test", "should show paths")
}

func TestViewGuide_EmptyNoteHidesSection(t *testing.T) {
	m := newTestModel()
	m.width = 80
	m.height = 24
	m.view = ViewGuide
	m.guideCategory = &types.Category{
		ID:     "test-manual",
		Name:   "Test Manual Item",
		Method: types.MethodManual,
		Note:   "", // Empty note
		Guide:  "Open app and clear data",
	}

	output := m.viewGuide()

	assert.NotContains(t, output, "⚠", "should not show warning icon when note is empty")
	assert.Contains(t, output, "Open app and clear data", "should show guide")
}

func TestViewGuide_EmptyPathsHidesSection(t *testing.T) {
	m := newTestModel()
	m.width = 80
	m.height = 24
	m.view = ViewGuide
	m.guideCategory = &types.Category{
		ID:     "test-manual",
		Name:   "Test Manual Item",
		Method: types.MethodManual,
		Guide:  "Open app settings",
		Paths:  []string{}, // Empty paths
	}

	output := m.viewGuide()

	assert.NotContains(t, output, "Paths:", "should not show Paths section when empty")
	assert.Contains(t, output, "Open app settings", "should show guide")
}

func TestHandleGuideKey_PathNavigation(t *testing.T) {
	m := newTestModel()
	m.view = ViewGuide
	m.guideCategory = &types.Category{
		ID:    "test-manual",
		Name:  "Test Item",
		Paths: []string{"~/path1", "~/path2", "~/path3"},
	}
	m.guidePathIndex = 0

	m.handleGuideKey(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.guidePathIndex, "down should increase path index")

	m.handleGuideKey(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, m.guidePathIndex, "down should increase path index")

	m.handleGuideKey(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, m.guidePathIndex, "should not exceed paths length")

	m.handleGuideKey(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 1, m.guidePathIndex, "up should decrease path index")

	m.guidePathIndex = 0
	m.handleGuideKey(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, m.guidePathIndex, "should not go below 0")
}

func newTestModelWithFilter() *Model {
	m := newTestModel()
	ti := textinput.New()
	ti.Placeholder = "Search..."
	m.filterInput = ti
	m.filterState = FilterTyping
	return m
}

func TestHandleFilterTypingKey_EnterAppliesFilter(t *testing.T) {
	m := newTestModelWithFilter()
	m.filterInput.SetValue("chrome")
	m.previewItemIndex = 5
	m.previewScroll = 10

	m.handleFilterTypingKey(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Equal(t, FilterApplied, m.filterState)
	assert.Equal(t, "chrome", m.filterText)
	assert.Equal(t, 0, m.previewItemIndex)
	assert.Equal(t, 0, m.previewScroll)
}

func TestHandleFilterTypingKey_EscCancelsFilter(t *testing.T) {
	m := newTestModelWithFilter()
	m.filterInput.SetValue("test")

	m.handleFilterTypingKey(tea.KeyMsg{Type: tea.KeyEsc})

	assert.Equal(t, FilterNone, m.filterState)
	assert.Equal(t, "", m.filterText)
}

func TestHandleFilterTypingKey_CtrlCQuits(t *testing.T) {
	m := newTestModelWithFilter()

	_, cmd := m.handleFilterTypingKey(tea.KeyMsg{Type: tea.KeyCtrlC})

	require.NotNil(t, cmd)
	msg := cmd()
	assert.IsType(t, tea.QuitMsg{}, msg)
}

func TestHandleFilterTypingKey_RegularKeyResetsScroll(t *testing.T) {
	m := newTestModelWithFilter()
	m.previewItemIndex = 5
	m.previewScroll = 10

	m.handleFilterTypingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	assert.Equal(t, 0, m.previewItemIndex)
	assert.Equal(t, 0, m.previewScroll)
}

func TestHandleConfirmKey_YesStartsCleaning(t *testing.T) {
	m := newTestModelWithResults()
	m.view = ViewConfirm
	beforeTime := time.Now()

	_, cmd := m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	assert.Equal(t, ViewCleaning, m.view)
	assert.False(t, m.startTime.IsZero())
	assert.True(t, m.startTime.After(beforeTime) || m.startTime.Equal(beforeTime))
	assert.NotNil(t, cmd)
}

func TestHandleConfirmKey_EnterStartsCleaning(t *testing.T) {
	m := newTestModelWithResults()
	m.view = ViewConfirm

	m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Equal(t, ViewCleaning, m.view)
}

func TestHandleConfirmKey_NoReturnsToPreview(t *testing.T) {
	m := newTestModelWithResults()
	m.view = ViewConfirm

	m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	assert.Equal(t, ViewPreview, m.view)
}

func TestHandleConfirmKey_EscReturnsToPreview(t *testing.T) {
	m := newTestModelWithResults()
	m.view = ViewConfirm

	m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyEsc})

	assert.Equal(t, ViewPreview, m.view)
}

func TestHandleConfirmKey_QuestionMarkOpensHelp(t *testing.T) {
	m := newTestModelWithResults()
	m.view = ViewConfirm

	m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	assert.Equal(t, ViewHelp, m.view)
	assert.Equal(t, ViewConfirm, m.helpPreviousView)
}

func TestHandleHelpKey_EscReturnsToList(t *testing.T) {
	m := newTestModel()
	m.view = ViewHelp
	m.helpPreviousView = ViewList

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyEsc})

	assert.Equal(t, ViewList, m.view)
}

func TestHandleHelpKey_EnterReturnsToPreview(t *testing.T) {
	m := newTestModel()
	m.view = ViewHelp
	m.helpPreviousView = ViewPreview

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Equal(t, ViewPreview, m.view)
}

func TestHandleHelpKey_SpaceReturnsToPreviousView(t *testing.T) {
	m := newTestModel()
	m.view = ViewHelp
	m.helpPreviousView = ViewConfirm

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeySpace})

	assert.Equal(t, ViewConfirm, m.view)
}

func TestHandleHelpKey_QuestionMarkCloses(t *testing.T) {
	m := newTestModel()
	m.view = ViewHelp
	m.helpPreviousView = ViewList

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	assert.Equal(t, ViewList, m.view)
}

func newTestModelForPreview() *Model {
	m := newTestModelWithResults()
	m.view = ViewPreview
	m.previewCatID = "cat1"
	m.selected["cat1"] = true
	m.selected["cat2"] = true
	return m
}

func TestHandlePreviewKey_CursorUpDecreases(t *testing.T) {
	m := newTestModelForPreview()
	m.previewItemIndex = 1

	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeyUp})

	assert.Equal(t, 0, m.previewItemIndex)
}

func TestHandlePreviewKey_CursorUpBoundsAtZero(t *testing.T) {
	m := newTestModelForPreview()
	m.previewItemIndex = 0

	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeyUp})

	assert.Equal(t, 0, m.previewItemIndex)
}

func TestHandlePreviewKey_CursorDownIncreases(t *testing.T) {
	m := newTestModelForPreview()
	m.previewItemIndex = 0

	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeyDown})

	assert.Equal(t, 1, m.previewItemIndex)
}

func TestHandlePreviewKey_SortToggle(t *testing.T) {
	m := newTestModelForPreview()
	m.previewItemIndex = 5
	m.previewScroll = 10
	initialSort := m.sortOrder
	expectedSort := initialSort.Next()

	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	assert.Equal(t, expectedSort, m.sortOrder)
	assert.Equal(t, 0, m.previewItemIndex)
	assert.Equal(t, 0, m.previewScroll)
}

func TestHandlePreviewKey_HomeGoesToFirst(t *testing.T) {
	m := newTestModelForPreview()
	m.previewItemIndex = 1
	m.previewScroll = 5

	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeyHome})

	assert.Equal(t, 0, m.previewItemIndex)
	assert.Equal(t, 0, m.previewScroll)
}

func TestHandlePreviewKey_EndGoesToLast(t *testing.T) {
	m := newTestModelForPreview()
	m.previewItemIndex = 0

	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeyEnd})

	assert.Equal(t, 1, m.previewItemIndex)
}

func TestHandlePreviewKey_SlashEntersSearchMode(t *testing.T) {
	m := newTestModelForPreview()
	ti := textinput.New()
	m.filterInput = ti

	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	assert.Equal(t, FilterTyping, m.filterState)
}

func TestHandlePreviewKey_EscClearsAppliedFilter(t *testing.T) {
	m := newTestModelForPreview()
	m.filterState = FilterApplied
	m.filterText = "test"

	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeyEsc})

	assert.Equal(t, FilterNone, m.filterState)
	assert.Equal(t, "", m.filterText)
	assert.Equal(t, ViewPreview, m.view)
}

func TestHandlePreviewKey_EscReturnsToListWhenNoFilter(t *testing.T) {
	m := newTestModelForPreview()
	m.filterState = FilterNone

	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeyEsc})

	assert.Equal(t, ViewList, m.view)
}

func TestHandlePreviewKey_YEntersConfirm(t *testing.T) {
	m := newTestModelForPreview()

	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	assert.Equal(t, ViewConfirm, m.view)
}

func TestHandleDrillDownKey_EscPopsStack(t *testing.T) {
	m := newTestModelForPreview()
	m.drillDownStack = append(m.drillDownStack, drillDownState{
		path:   "/test/path",
		items:  []types.CleanableItem{{Path: "/test/path/a"}},
		cursor: 0,
	})

	m.handleDrillDownKey(tea.KeyMsg{Type: tea.KeyEsc})

	assert.Empty(t, m.drillDownStack)
}

func TestHandleDrillDownKey_CursorNavigation(t *testing.T) {
	m := newTestModelForPreview()
	m.drillDownStack = append(m.drillDownStack, drillDownState{
		path:   "/test/path",
		items:  []types.CleanableItem{{Path: "/a"}, {Path: "/b"}, {Path: "/c"}},
		cursor: 0,
	})

	m.handleDrillDownKey(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.drillDownStack[0].cursor)

	m.handleDrillDownKey(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, m.drillDownStack[0].cursor)
}

func TestHandleDrillDownKey_QuestionMarkOpensHelp(t *testing.T) {
	m := newTestModelForPreview()
	m.drillDownStack = append(m.drillDownStack, drillDownState{
		path:   "/test/path",
		items:  []types.CleanableItem{{Path: "/a"}},
		cursor: 0,
	})

	m.handleDrillDownKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	assert.Equal(t, ViewHelp, m.view)
	assert.Equal(t, ViewPreview, m.helpPreviousView)
}

func TestHandleConfirmKey_CtrlCQuits(t *testing.T) {
	m := newTestModelWithResults()
	m.view = ViewConfirm

	_, cmd := m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyCtrlC})

	require.NotNil(t, cmd)
	msg := cmd()
	assert.IsType(t, tea.QuitMsg{}, msg)
}

func TestHandleConfirmKey_QQuits(t *testing.T) {
	m := newTestModelWithResults()
	m.view = ViewConfirm

	_, cmd := m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	require.NotNil(t, cmd)
	msg := cmd()
	assert.IsType(t, tea.QuitMsg{}, msg)
}

func TestHandleHelpKey_CtrlCQuits(t *testing.T) {
	m := newTestModel()
	m.view = ViewHelp

	_, cmd := m.handleHelpKey(tea.KeyMsg{Type: tea.KeyCtrlC})

	require.NotNil(t, cmd)
	msg := cmd()
	assert.IsType(t, tea.QuitMsg{}, msg)
}

func TestHandlePreviewKey_LeftSwitchesCategory(t *testing.T) {
	m := newTestModelForPreview()
	m.previewCatID = "cat2"
	m.previewItemIndex = 5
	m.previewScroll = 10
	m.filterState = FilterApplied
	m.filterText = "test"

	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeyLeft})

	assert.Equal(t, "cat1", m.previewCatID)
	assert.Equal(t, 0, m.previewItemIndex)
	assert.Equal(t, 0, m.previewScroll)
	assert.Equal(t, FilterNone, m.filterState)
	assert.Equal(t, "", m.filterText)
}

func TestHandlePreviewKey_RightSwitchesCategory(t *testing.T) {
	m := newTestModelForPreview()
	m.previewCatID = "cat1"
	m.previewItemIndex = 5
	m.previewScroll = 10
	m.filterState = FilterApplied
	m.filterText = "test"

	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeyRight})

	assert.Equal(t, "cat2", m.previewCatID)
	assert.Equal(t, 0, m.previewItemIndex)
	assert.Equal(t, 0, m.previewScroll)
	assert.Equal(t, FilterNone, m.filterState)
	assert.Equal(t, "", m.filterText)
}

func TestHandlePreviewKey_SpaceTogglesExclusion(t *testing.T) {
	m := newTestModelForPreview()
	m.excluded = make(map[string]map[string]bool)
	itemPath := m.results[0].Items[0].Path

	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeySpace})

	assert.True(t, m.excluded["cat1"][itemPath])
}

func TestHandlePreviewKey_PageDown(t *testing.T) {
	m := newTestModelForPreview()
	m.height = 20
	m.previewItemIndex = 0

	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeyPgDown})

	assert.Greater(t, m.previewItemIndex, 0)
}

func TestHandlePreviewKey_PageUp(t *testing.T) {
	m := newTestModelForPreview()
	m.previewItemIndex = 1

	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeyPgUp})

	assert.Equal(t, 0, m.previewItemIndex)
}

func TestHandlePreviewKey_AIncludesAll(t *testing.T) {
	m := newTestModelForPreview()
	m.excluded = map[string]map[string]bool{
		"cat1": {"/path/1": true, "/path/2": true},
	}

	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	assert.Empty(t, m.excluded["cat1"])
}

func TestHandlePreviewKey_DExcludesAll(t *testing.T) {
	m := newTestModelForPreview()
	m.excluded = make(map[string]map[string]bool)

	m.handlePreviewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	assert.NotEmpty(t, m.excluded["cat1"])
}

func TestHandleDrillDownKey_BackspacePopsStack(t *testing.T) {
	m := newTestModelForPreview()
	m.drillDownStack = append(m.drillDownStack, drillDownState{
		path:   "/test/path",
		items:  []types.CleanableItem{{Path: "/a"}},
		cursor: 0,
	})

	m.handleDrillDownKey(tea.KeyMsg{Type: tea.KeyBackspace})

	assert.Empty(t, m.drillDownStack)
}

func TestHandleDrillDownKey_HomeGoesToFirst(t *testing.T) {
	m := newTestModelForPreview()
	m.drillDownStack = append(m.drillDownStack, drillDownState{
		path:   "/test/path",
		items:  []types.CleanableItem{{Path: "/a"}, {Path: "/b"}, {Path: "/c"}},
		cursor: 2,
	})

	m.handleDrillDownKey(tea.KeyMsg{Type: tea.KeyHome})

	assert.Equal(t, 0, m.drillDownStack[0].cursor)
}

func TestHandleDrillDownKey_EndGoesToLast(t *testing.T) {
	m := newTestModelForPreview()
	m.drillDownStack = append(m.drillDownStack, drillDownState{
		path:   "/test/path",
		items:  []types.CleanableItem{{Path: "/a"}, {Path: "/b"}, {Path: "/c"}},
		cursor: 0,
	})

	m.handleDrillDownKey(tea.KeyMsg{Type: tea.KeyEnd})

	assert.Equal(t, 2, m.drillDownStack[0].cursor)
}

func TestHandleDrillDownKey_SortToggle(t *testing.T) {
	m := newTestModelForPreview()
	m.drillDownStack = append(m.drillDownStack, drillDownState{
		path:   "/test/path",
		items:  []types.CleanableItem{{Path: "/a"}},
		cursor: 0,
	})
	initialSort := m.sortOrder
	expectedSort := initialSort.Next()

	m.handleDrillDownKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	assert.Equal(t, expectedSort, m.sortOrder)
}
