package tui

import (
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"mac-cleanup-go/internal/cleaner"
	"mac-cleanup-go/internal/scanner"
	"mac-cleanup-go/pkg/types"
	"mac-cleanup-go/pkg/utils"
)

// View state
type View int

const (
	ViewList View = iota
	ViewPreview
	ViewConfirm
	ViewCleaning
	ViewReport
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

	report    *types.Report
	startTime time.Time
	scroll    int

	hasFullDiskAccess bool

	err error
}

type drillDownState struct {
	path   string
	items  []types.CleanableItem
	cursor int
	scroll int
}

// Messages
type scanResultMsg struct{ result *types.ScanResult }
type cleanDoneMsg struct{ report *types.Report }

// NewModel creates a new model
func NewModel(cfg *types.Config) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	return &Model{
		config:            cfg,
		registry:          scanner.DefaultRegistry(cfg),
		cleaner:           cleaner.New(),
		selected:          make(map[string]bool),
		excluded:          make(map[string]map[string]bool),
		resultMap:         make(map[string]*types.ScanResult),
		results:           make([]*types.ScanResult, 0),
		drillDownStack:    make([]drillDownState, 0),
		view:              ViewList,
		spinner:           s,
		scanning:          true,
		hasFullDiskAccess: utils.CheckFullDiskAccess(),
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
		s := s
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
		if m.scanning {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	case scanResultMsg:
		m.handleScanResult(msg.result)
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

	if result != nil && result.TotalSize > 0 {
		sort.Slice(result.Items, func(i, j int) bool {
			return result.Items[i].Size > result.Items[j].Size
		})
		m.results = append(m.results, result)
		m.resultMap[result.Category.ID] = result
		sort.Slice(m.results, func(i, j int) bool {
			return m.results[i].TotalSize > m.results[j].TotalSize
		})
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

// Helper methods
func (m *Model) getSelectedResults() []*types.ScanResult {
	var selected []*types.ScanResult
	for _, r := range m.results {
		if m.selected[r.Category.ID] {
			selected = append(selected, r)
		}
	}
	return selected
}

func (m *Model) hasSelection() bool {
	for _, v := range m.selected {
		if v {
			return true
		}
	}
	return false
}

func (m *Model) getSelectedSize() int64 {
	var total int64
	for id, sel := range m.selected {
		if sel {
			if r, ok := m.resultMap[id]; ok {
				total += m.getEffectiveSize(r)
			}
		}
	}
	return total
}

func (m *Model) getEffectiveSize(r *types.ScanResult) int64 {
	excludedMap := m.excluded[r.Category.ID]
	if excludedMap == nil {
		return r.TotalSize
	}
	var total int64
	for _, item := range r.Items {
		if !excludedMap[item.Path] {
			total += item.Size
		}
	}
	return total
}

func (m *Model) getSelectedCount() int {
	count := 0
	for _, v := range m.selected {
		if v {
			count++
		}
	}
	return count
}

// getPreviewCatResult returns the current preview category's ScanResult
func (m *Model) getPreviewCatResult() *types.ScanResult {
	if m.previewCatID == "" {
		return nil
	}
	return m.resultMap[m.previewCatID]
}

// findSelectedCatIndex returns the index of current preview category within selected results
func (m *Model) findSelectedCatIndex() int {
	selected := m.getSelectedResults()
	for i, r := range selected {
		if r.Category.ID == m.previewCatID {
			return i
		}
	}
	return -1
}

// findPrevSelectedCatID returns the previous selected category ID
func (m *Model) findPrevSelectedCatID() string {
	selected := m.getSelectedResults()
	for i, r := range selected {
		if r.Category.ID == m.previewCatID && i > 0 {
			return selected[i-1].Category.ID
		}
	}
	return m.previewCatID
}

// findNextSelectedCatID returns the next selected category ID
func (m *Model) findNextSelectedCatID() string {
	selected := m.getSelectedResults()
	for i, r := range selected {
		if r.Category.ID == m.previewCatID && i < len(selected)-1 {
			return selected[i+1].Category.ID
		}
	}
	return m.previewCatID
}

func (m *Model) visibleLines() int {
	lines := m.height - 10
	if lines < 5 {
		return 5
	}
	return lines
}

func (m *Model) adjustScroll() {
	visible := m.visibleLines()
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	} else if m.cursor >= m.scroll+visible {
		m.scroll = m.cursor - visible + 1
	}
}

func (m *Model) readDirectory(path string) []types.CleanableItem {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	var items []types.CleanableItem
	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		item := types.CleanableItem{
			Path:        fullPath,
			Name:        entry.Name(),
			IsDirectory: entry.IsDir(),
			ModifiedAt:  info.ModTime(),
		}

		if entry.IsDir() {
			item.Size, _ = utils.GetDirSize(fullPath)
		} else {
			item.Size = info.Size()
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Size > items[j].Size
	})
	return items
}

func (m *Model) tryDrillDown() bool {
	r := m.getPreviewCatResult()
	if r == nil {
		return false
	}

	if m.previewItemIndex < 0 || m.previewItemIndex >= len(r.Items) {
		return false
	}

	item := r.Items[m.previewItemIndex]
	if !item.IsDirectory {
		return false
	}

	items := m.readDirectory(item.Path)
	if len(items) == 0 {
		return false
	}

	m.drillDownStack = append(m.drillDownStack, drillDownState{
		path:   item.Path,
		items:  items,
		cursor: 0,
		scroll: 0,
	})
	return true
}

func (m *Model) doClean() tea.Cmd {
	return func() tea.Msg {
		report := &types.Report{Results: make([]types.CleanResult, 0)}

		for id, sel := range m.selected {
			if !sel {
				continue
			}
			r, ok := m.resultMap[id]
			if !ok {
				continue
			}

			// Skip manual method - user must clean via app
			if r.Category.Method == types.MethodManual {
				continue
			}

			// Filter out excluded items
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

			var result *types.CleanResult

			// For special method (Docker), use the scanner's Clean method
			if r.Category.Method == types.MethodSpecial {
				if s, ok := m.registry.Get(id); ok {
					result, _ = s.Clean(items, false)
				}
			} else {
				// For other methods, use the cleaner
				result = m.cleaner.Clean(r.Category, items, false)
			}

			if result != nil {
				report.Results = append(report.Results, *result)
				report.FreedSpace += result.FreedSpace
				report.CleanedItems += result.CleanedItems
				report.FailedItems += len(result.Errors)
			}
		}

		return cleanDoneMsg{report: report}
	}
}

// isExcluded checks if an item is excluded
func (m *Model) isExcluded(catID, path string) bool {
	if m.excluded[catID] == nil {
		return false
	}
	return m.excluded[catID][path]
}

// toggleExclude toggles exclusion for an item
func (m *Model) toggleExclude(catID, path string) {
	if m.excluded[catID] == nil {
		m.excluded[catID] = make(map[string]bool)
	}
	m.excluded[catID][path] = !m.excluded[catID][path]
}

// autoExcludeCategory marks all items in a category as excluded
func (m *Model) autoExcludeCategory(catID string, r *types.ScanResult) {
	if m.excluded[catID] == nil {
		m.excluded[catID] = make(map[string]bool)
	}
	for _, item := range r.Items {
		m.excluded[catID][item.Path] = true
	}
}
