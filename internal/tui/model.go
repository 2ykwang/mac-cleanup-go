package tui

import (
	"sort"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"mac-cleanup-go/internal/cleaner"
	"mac-cleanup-go/internal/scanner"
	"mac-cleanup-go/internal/userconfig"
	"mac-cleanup-go/internal/utils"
	"mac-cleanup-go/pkg/types"
)

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

	err error

	// Scan errors for display
	scanErrors []scanErrorInfo
}

// NewModel creates a new model
func NewModel(cfg *types.Config) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

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

	return &Model{
		config:            cfg,
		registry:          scanner.DefaultRegistry(cfg),
		cleaner:           cleaner.New(),
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
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.startScan())
}

func (m *Model) startScan() tea.Cmd {
	m.scanRegistered = len(m.registry.All())
	scanners := m.registry.Available()
	m.scanTotal = len(scanners)
	m.scanCompleted = 0

	cmds := make([]tea.Cmd, len(scanners))
	for i, s := range scanners {
		cmds[i] = func() tea.Msg {
			result, _ := s.Scan()
			return scanResultMsg{result: result}
		}
	}
	return tea.Batch(cmds...)
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case spinner.TickMsg:
		if m.scanning || m.view == ViewCleaning {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	case scanResultMsg:
		m.handleScanResult(msg.result)
	case cleanProgressMsg:
		m.cleaningCategory = msg.categoryName
		m.cleaningItem = msg.currentItem
		m.cleaningCurrent = msg.current
		m.cleaningTotal = msg.total
		// Continue waiting for next progress or done message
		return m, m.waitForCleanProgress()
	case cleanCategoryDoneMsg:
		// Add completed category to list
		m.cleaningCompleted = append(m.cleaningCompleted, cleanedCategory{
			name:       msg.categoryName,
			freedSpace: msg.freedSpace,
			cleaned:    msg.cleanedItems,
			errors:     msg.errorCount,
		})
		m.cleaningCategory = ""
		m.cleaningItem = ""
		return m, m.waitForCleanProgress()
	case cleanDoneMsg:
		m.report = msg.report
		m.report.Duration = time.Since(m.startTime)
		m.view = ViewReport
	}
	return m, nil
}

func (m *Model) handleScanResult(result *types.ScanResult) {
	m.scanMutex.Lock()
	defer m.scanMutex.Unlock()

	if result != nil {
		// Collect scan errors for display
		if result.Error != nil {
			m.scanErrors = append(m.scanErrors, scanErrorInfo{
				CategoryName: result.Category.Name,
				Error:        result.Error.Error(),
			})
		}

		// Collect results with items
		if result.TotalSize > 0 {
			sort.Slice(result.Items, func(i, j int) bool {
				return result.Items[i].Size > result.Items[j].Size
			})
			m.results = append(m.results, result)
			m.resultMap[result.Category.ID] = result
			sort.Slice(m.results, func(i, j int) bool {
				return m.results[i].TotalSize > m.results[j].TotalSize
			})
		}
	}
	m.scanCompleted++
	if m.scanCompleted >= m.scanTotal {
		m.scanning = false
	}
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
	default:
		return m.viewList()
	}
}

func (m *Model) doClean() tea.Cmd {
	// Collect items to clean
	type cleanJob struct {
		category  types.Category
		items     []types.CleanableItem
		isSpecial bool
	}
	var jobs []cleanJob

	for id, sel := range m.selected {
		if !sel {
			continue
		}
		r, ok := m.resultMap[id]
		if !ok {
			continue
		}
		if r.Category.Method == types.MethodManual {
			continue
		}

		var items []types.CleanableItem
		excludedMap := m.excluded[id]
		for _, item := range r.Items {
			if excludedMap == nil || !excludedMap[item.Path] {
				items = append(items, item)
			}
		}
		if len(items) == 0 {
			continue
		}

		jobs = append(jobs, cleanJob{
			category:  r.Category,
			items:     items,
			isSpecial: r.Category.Method == types.MethodSpecial,
		})
	}

	// Calculate total items
	totalItems := 0
	for _, job := range jobs {
		totalItems += len(job.items)
	}
	m.cleaningTotal = totalItems
	m.cleaningCurrent = 0
	m.cleaningCompleted = nil // Reset completed list

	// Create channels for progress communication
	m.cleanProgressChan = make(chan cleanProgressMsg, 1)
	m.cleanDoneChan = make(chan cleanDoneMsg, 1)
	m.cleanCategoryDoneCh = make(chan cleanCategoryDoneMsg, 1)

	// Start cleaning in background goroutine
	go func() {
		report := &types.Report{Results: make([]types.CleanResult, 0)}
		currentItem := 0

		for _, job := range jobs {
			var result *types.CleanResult

			if job.isSpecial {
				// Send progress for special jobs
				m.cleanProgressChan <- cleanProgressMsg{
					categoryName: job.category.Name,
					currentItem:  "",
					current:      currentItem,
					total:        totalItems,
				}

				if s, ok := m.registry.Get(job.category.ID); ok {
					result, _ = s.Clean(job.items, false)
				}
				currentItem += len(job.items)
			} else {
				cat := job.category

				// Clean items one by one for progress tracking
				itemResult := &types.CleanResult{
					Category: cat,
					Errors:   make([]string, 0),
				}
				for _, item := range job.items {
					currentItem++

					// Send progress update
					m.cleanProgressChan <- cleanProgressMsg{
						categoryName: job.category.Name,
						currentItem:  item.Name,
						current:      currentItem,
						total:        totalItems,
					}

					singleResult := m.cleaner.Clean(cat, []types.CleanableItem{item}, false)
					itemResult.FreedSpace += singleResult.FreedSpace
					itemResult.CleanedItems += singleResult.CleanedItems
					itemResult.Errors = append(itemResult.Errors, singleResult.Errors...)
				}
				result = itemResult
			}

			if result != nil {
				report.Results = append(report.Results, *result)
				report.FreedSpace += result.FreedSpace
				report.CleanedItems += result.CleanedItems
				report.FailedItems += len(result.Errors)

				// Send category done message
				m.cleanCategoryDoneCh <- cleanCategoryDoneMsg{
					categoryName: job.category.Name,
					freedSpace:   result.FreedSpace,
					cleanedItems: result.CleanedItems,
					errorCount:   len(result.Errors),
				}
			}
		}

		m.cleanDoneChan <- cleanDoneMsg{report: report}
	}()

	// Return command to wait for first progress/done message
	return m.waitForCleanProgress()
}

// waitForCleanProgress returns a command that waits for the next progress or done message
func (m *Model) waitForCleanProgress() tea.Cmd {
	return func() tea.Msg {
		select {
		case progress := <-m.cleanProgressChan:
			return progress
		case catDone := <-m.cleanCategoryDoneCh:
			return catDone
		case done := <-m.cleanDoneChan:
			return done
		}
	}
}
