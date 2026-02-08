package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/2ykwang/mac-cleanup-go/internal/cleaner"
	"github.com/2ykwang/mac-cleanup-go/internal/logger"
	"github.com/2ykwang/mac-cleanup-go/internal/styles"
	"github.com/2ykwang/mac-cleanup-go/internal/target"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/userconfig"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

// defaultRecentItemsCapacity is the maximum number of recent deleted items to display
const (
	defaultRecentItemsCapacity = 10
	maxContentWidth            = 140
	maxListContentWidth        = 120
	maxPreviewContentWidth     = 120
	maxReportContentWidth      = 120
	maxCleaningContentWidth    = 120
)

// Model is the main TUI model
type Model struct {
	configState
	dataState
	selectionState
	layoutState
	scanState
	previewState
	confirmState
	filterStateData
	cleaningState
	reportState
	versionState
}

// NewModel creates a new model
func NewModel(cfg *types.Config, currentVersion string) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.ColorPrimary)

	// Initialize filter input
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 100
	ti.Width = 30

	// Load user config
	userCfg, _ := userconfig.Load()

	// Initialize excluded from saved config
	excluded := userCfg.ExcludedPathsMap()

	registry, err := target.DefaultRegistry(cfg)
	if err != nil {
		logger.Warn("registry initialization failed", "error", err)
		// Prevent nil registry when we surface a fatal config error.
		registry = target.NewRegistry()
	}

	logger.Info("model initialized",
		"categories", len(cfg.Categories),
		"hasFullDiskAccess", utils.CheckFullDiskAccess())

	// Initialize progress bar
	prog := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)

	return &Model{
		configState: configState{
			config:            cfg,
			registry:          registry,
			cleanService:      cleaner.NewCleanService(registry),
			hasFullDiskAccess: utils.CheckFullDiskAccess(),
			userConfig:        userCfg,
			err:               err,
		},
		dataState: dataState{
			results:   make([]*types.ScanResult, 0),
			resultMap: make(map[string]*types.ScanResult),
		},
		selectionState: selectionState{
			selected:      make(map[string]bool),
			selectedOrder: make([]string, 0),
			excluded:      excluded,
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
		confirmState: confirmState{
			confirmChoice: confirmCancel,
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
		versionState: versionState{
			currentVersion: currentVersion,
		},
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.startScan(), m.checkVersion())
}

// View renders the UI
func (m *Model) View() string {
	if m.err != nil {
		return m.renderCentered(func() string {
			return "Error: " + m.err.Error() + "\n\nPress q to quit."
		}, maxContentWidth)
	}

	switch m.view {
	case ViewList:
		return m.renderCentered(m.viewList, maxListContentWidth)
	case ViewPreview:
		return m.renderCentered(m.viewPreview, maxPreviewContentWidth)
	case ViewConfirm:
		return m.renderCentered(m.viewConfirm, maxPreviewContentWidth)
	case ViewCleaning:
		return m.renderCentered(m.viewCleaning, maxCleaningContentWidth)
	case ViewReport:
		return m.renderCentered(m.viewReport, maxReportContentWidth)
	case ViewGuide:
		return m.renderCentered(m.viewGuide, maxContentWidth)
	default:
		return m.renderCentered(m.viewList, maxListContentWidth)
	}
}

func (m *Model) renderCentered(render func() string, maxWidth int) string {
	if maxWidth <= 0 || m.width <= maxWidth {
		return render()
	}

	contentWidth := maxWidth
	padding := (m.width - contentWidth) / 2
	if padding <= 0 {
		return render()
	}

	originalWidth := m.width
	originalHelpWidth := m.help.Width
	m.width = contentWidth
	m.help.Width = contentWidth

	output := render()

	m.width = originalWidth
	m.help.Width = originalHelpWidth

	return indentLines(output, strings.Repeat(" ", padding))
}
