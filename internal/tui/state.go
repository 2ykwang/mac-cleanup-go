package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"

	"github.com/2ykwang/mac-cleanup-go/internal/cleaner"
	"github.com/2ykwang/mac-cleanup-go/internal/scanner"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/userconfig"
)

type configState struct {
	config            *types.Config
	registry          *scanner.Registry
	cleaner           *cleaner.Cleaner
	hasFullDiskAccess bool
	userConfig        *userconfig.UserConfig
	err               error
}

type dataState struct {
	results   []*types.ScanResult
	resultMap map[string]*types.ScanResult
}

type selectionState struct {
	selected map[string]bool
	excluded map[string]map[string]bool // categoryID -> itemPath -> excluded
	cursor   int
}

type layoutState struct {
	view          View
	width         int
	height        int
	scroll        int
	statusMessage string
	help          help.Model
}

type scanState struct {
	scanCompleted  int
	scanTotal      int
	scanRegistered int
	scanning       bool
	spinner        spinner.Model
	scanErrors     []scanErrorInfo
}

type previewState struct {
	previewCatID     string
	previewItemIndex int
	previewScroll    int
	drillDownStack   []drillDownState
	sortOrder        types.SortOrder
	guideCategory    *types.Category
	guidePathIndex   int
}

type filterStateData struct {
	filterState FilterState
	filterText  string
	filterInput textinput.Model
}

type cleaningState struct {
	cleaningCategory    string
	cleaningItem        string
	cleaningCurrent     int
	cleaningTotal       int
	cleaningCompleted   []cleanedCategory
	cleaningProgress    progress.Model
	cleanProgressChan   chan cleanProgressMsg
	cleanDoneChan       chan cleanDoneMsg
	cleanCategoryDoneCh chan cleanCategoryDoneMsg
	cleanItemDoneChan   chan cleanItemDoneMsg
	recentDeleted       *RingBuffer[DeletedItemEntry]
}

type reportState struct {
	report       *types.Report
	startTime    time.Time
	reportScroll int
	reportLines  []string
}
