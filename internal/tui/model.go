package tui

import (
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/2ykwang/mac-cleanup-go/internal/cleaner"
	"github.com/2ykwang/mac-cleanup-go/internal/scanner"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/userconfig"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

// defaultRecentItemsCapacity is the maximum number of recent deleted items to display
const defaultRecentItemsCapacity = 10

// Model is the main TUI model
type Model struct {
	config   *types.Config
	registry *scanner.Registry
	cleaner  *cleaner.Cleaner

	results   []*types.ScanResult
	resultMap map[string]*types.ScanResult
	selected  map[string]bool
	excluded  map[string]map[string]bool // categoryID -> itemPath -> excluded
	cursor    int

	view   View
	width  int
	height int

	scanCompleted  int
	scanTotal      int
	scanRegistered int // Total registered categories
	scanning       bool
	spinner        spinner.Model
	scanMutex      sync.Mutex

	// Preview state
	previewCatID     string // Category ID instead of index
	previewItemIndex int
	previewScroll    int
	drillDownStack   []drillDownState
	sortOrder        types.SortOrder // Current sort order for items

	// Filter/search state
	filterState FilterState
	filterText  string
	filterInput textinput.Model

	report       *types.Report
	startTime    time.Time
	scroll       int
	reportScroll int
	reportLines  []string // Pre-rendered report lines for scrolling

	hasFullDiskAccess bool

	userConfig *userconfig.UserConfig

	// Cleaning progress
	cleaningCategory  string
	cleaningItem      string
	cleaningCurrent   int
	cleaningTotal     int
	cleaningCompleted []cleanedCategory // Completed categories

	// Channels for cleaning progress
	cleanProgressChan   chan cleanProgressMsg
	cleanDoneChan       chan cleanDoneMsg
	cleanCategoryDoneCh chan cleanCategoryDoneMsg
	cleanItemDoneChan   chan cleanItemDoneMsg

	err error

	// Scan errors for display
	scanErrors []scanErrorInfo

	// Recent deleted items for progress display
	recentDeleted *RingBuffer[DeletedItemEntry]

	// Status message for user feedback (e.g., error messages)
	statusMessage string

	// Guide popup state (for Manual items)
	guideCategory  *types.Category // Category being shown in guide popup
	guidePathIndex int             // Selected path index in guide popup

	// Help component
	help help.Model
}

// NewModel creates a new model
func NewModel(cfg *types.Config) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	// Initialize filter input
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 100
	ti.Width = 30

	// Load user config
	userCfg, _ := userconfig.Load()

	// Initialize excluded from saved config
	excluded := make(map[string]map[string]bool)
	for catID, paths := range userCfg.ExcludedPaths {
		excluded[catID] = make(map[string]bool)
		for _, path := range paths {
			excluded[catID][path] = true
		}
	}

	registry := scanner.DefaultRegistry(cfg)

	return &Model{
		config:            cfg,
		registry:          registry,
		cleaner:           cleaner.New(registry),
		selected:          make(map[string]bool),
		excluded:          excluded,
		resultMap:         make(map[string]*types.ScanResult),
		results:           make([]*types.ScanResult, 0),
		drillDownStack:    make([]drillDownState, 0),
		view:              ViewList,
		spinner:           s,
		scanning:          true,
		hasFullDiskAccess: utils.CheckFullDiskAccess(),
		userConfig:        userCfg,
		scanErrors:        make([]scanErrorInfo, 0),
		recentDeleted:     NewRingBuffer[DeletedItemEntry](defaultRecentItemsCapacity),
		sortOrder:         types.SortBySize,
		filterState:       FilterNone,
		filterInput:       ti,
		help:              help.New(),
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.startScan())
}

// View renders the UI
func (m *Model) View() string {
	if m.err != nil {
		return "Error: " + m.err.Error() + "\n\nPress q to quit."
	}

	switch m.view {
	case ViewList:
		return m.viewList()
	case ViewPreview:
		return m.viewPreview()
	case ViewConfirm:
		return m.viewConfirm()
	case ViewCleaning:
		return m.viewCleaning()
	case ViewReport:
		return m.viewReport()
	case ViewGuide:
		return m.viewGuide()
	default:
		return m.viewList()
	}
}
