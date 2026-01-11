package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/progress"
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
	configState
	dataState
	selectionState
	layoutState
	scanState
	previewState
	filterStateData
	cleaningState
	reportState
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

	registry, err := scanner.DefaultRegistry(cfg)
	if err != nil {
		// Prevent nil registry when we surface a fatal config error.
		registry = scanner.NewRegistry()
	}

	// Initialize progress bar
	prog := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)

	return &Model{
		configState: configState{
			config:            cfg,
			registry:          registry,
			cleaner:           cleaner.New(registry),
			hasFullDiskAccess: utils.CheckFullDiskAccess(),
			userConfig:        userCfg,
			err:               err,
		},
		dataState: dataState{
			results:   make([]*types.ScanResult, 0),
			resultMap: make(map[string]*types.ScanResult),
		},
		selectionState: selectionState{
			selected: make(map[string]bool),
			excluded: excluded,
		},
		layoutState: layoutState{
			view: ViewList,
			help: help.New(),
		},
		scanState: scanState{
			scanning:   true,
			spinner:    s,
			scanErrors: make([]scanErrorInfo, 0),
		},
		previewState: previewState{
			drillDownStack: make([]drillDownState, 0),
			sortOrder:      types.SortBySize,
		},
		filterStateData: filterStateData{
			filterState: FilterNone,
			filterInput: ti,
		},
		cleaningState: cleaningState{
			cleaningProgress: prog,
			recentDeleted:    NewRingBuffer[DeletedItemEntry](defaultRecentItemsCapacity),
		},
		reportState: reportState{},
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
