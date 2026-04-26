package tui

import (
	"strings"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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
	themeState
}

// NewModel creates a new model
func NewModel(cfg *types.Config, currentVersion string) *Model {
	theme := styles.New(true)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.Muted)

	// Initialize filter input
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 100
	ti.SetWidth(30)

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
		progress.WithDefaultBlend(),
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
			help: newStyledHelp(theme),
		},
		scanState: scanState{
			scanning:    true,
			spinner:     s,
			scanDoneIDs: make(map[string]bool),
			scanErrors:  make([]scanErrorInfo, 0),
		},
		previewState: previewState{
			drillDownStack:   make([]drillDownState, 0),
			sortOrder:        types.SortBySize,
			previewCollapsed: make(map[string]bool),
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
		themeState: themeState{styles: theme},
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.startScan(), m.checkVersion(), tea.RequestBackgroundColor)
}

// View renders the UI
func (m *Model) View() tea.View {
	if m.err != nil {
		return tea.View{
			Content: m.renderCentered(func() string {
				return "Error: " + m.err.Error() + "\n\nPress q to quit."
			}, maxContentWidth),
			AltScreen:       true,
			ForegroundColor: m.styles.Text,
		}
	}

	var output string
	switch m.view {
	case ViewList:
		output = m.renderCentered(m.viewList, maxListContentWidth)
	case ViewPreview:
		output = m.renderCentered(m.viewPreview, maxPreviewContentWidth)
	case ViewConfirm:
		output = m.renderCentered(m.viewConfirm, maxPreviewContentWidth)
	case ViewCleaning:
		output = m.renderCentered(m.viewCleaning, maxCleaningContentWidth)
	case ViewReport:
		output = m.renderCentered(m.viewReport, maxReportContentWidth)
	case ViewGuide:
		output = m.renderCentered(m.viewGuide, maxContentWidth)
	default:
		output = m.renderCentered(m.viewList, maxListContentWidth)
	}

	if m.showHelp {
		base := lipgloss.NewStyle().Faint(true).Render(output)
		output = overlayCentered(base, m.helpDialog(), m.width, m.height)
	}
	if m.showHint {
		base := lipgloss.NewStyle().Faint(true).Render(output)
		output = overlayCentered(base, m.hintDialog(), m.width, m.height)
	}
	return tea.View{Content: output, AltScreen: true, ForegroundColor: m.styles.Text}
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
	originalHelpWidth := m.help.Width()
	m.width = contentWidth
	m.help.SetWidth(contentWidth)

	output := render()

	m.width = originalWidth
	m.help.SetWidth(originalHelpWidth)

	return indentLines(output, strings.Repeat(" ", padding))
}
