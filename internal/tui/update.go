package tui

import (
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/2ykwang/mac-cleanup-go/internal/logger"
	"github.com/2ykwang/mac-cleanup-go/internal/target"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func (m *Model) startScan() tea.Cmd {
	m.scanRegistered = len(m.registry.All())
	scanners := m.registry.Available()
	m.scanTotal = len(scanners)
	m.scanCompleted = 0

	logger.Info("scan started", "registered", m.scanRegistered, "available", m.scanTotal)

	m.initScanResults(scanners)
	if len(scanners) == 0 {
		m.scanning = false
		logger.Info("scan skipped: no available scanners")
		return nil
	}

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
		return m.handleWindowSize(msg)
	case spinner.TickMsg:
		return m.handleSpinnerTick(msg)
	case scanResultMsg:
		m.handleScanResult(msg.result)
	case cleanProgressMsg:
		return m.handleCleanProgress(msg)
	case progress.FrameMsg:
		var cmd tea.Cmd
		progressModel, cmd := m.cleaningProgress.Update(msg)
		m.cleaningProgress = progressModel.(progress.Model)
		return m, cmd
	case cleanItemDoneMsg:
		return m.handleCleanItemDone(msg)
	case cleanCategoryDoneMsg:
		return m.handleCleanCategoryDone(msg)
	case cleanDoneMsg:
		m.handleCleanDone(msg)
	case versionCheckMsg:
		m.latestVersion = msg.latestVersion
		m.updateAvailable = msg.updateAvailable
	}
	return m, nil
}

func (m *Model) initScanResults(scanners []target.Target) {
	m.results = m.results[:0]
	m.resultMap = make(map[string]*types.ScanResult)

	available := make(map[string]target.Target, len(scanners))
	for _, s := range scanners {
		available[s.Category().ID] = s
	}

	if m.config != nil {
		for _, cat := range m.config.Categories {
			if _, ok := available[cat.ID]; !ok {
				continue
			}
			result := types.NewScanResult(cat)
			m.results = append(m.results, result)
			m.resultMap[cat.ID] = result
		}
		return
	}

	for _, s := range scanners {
		cat := s.Category()
		result := types.NewScanResult(cat)
		m.results = append(m.results, result)
		m.resultMap[cat.ID] = result
	}
}

func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width, m.height = msg.Width, msg.Height
	m.help.Width = msg.Width
	return m, nil
}

func (m *Model) handleSpinnerTick(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	if m.scanning || m.view == ViewCleaning {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) handleCleanProgress(msg cleanProgressMsg) (tea.Model, tea.Cmd) {
	m.cleaningCategory = msg.categoryName
	m.cleaningItem = msg.currentItem
	m.cleaningCurrent = msg.current
	m.cleaningTotal = msg.total

	// Update progress bar
	if m.cleaningTotal > 0 {
		percent := float64(m.cleaningCurrent) / float64(m.cleaningTotal)
		cmd := m.cleaningProgress.SetPercent(percent)
		return m, tea.Batch(cmd, m.waitForCleanProgress())
	}
	return m, m.waitForCleanProgress()
}

func (m *Model) handleCleanItemDone(msg cleanItemDoneMsg) (tea.Model, tea.Cmd) {
	// Add deleted item to recent deletions list
	m.recentDeleted.Push(DeletedItemEntry{
		Path:    msg.path,
		Name:    msg.name,
		Size:    msg.size,
		Success: msg.success,
		ErrMsg:  msg.errMsg,
	})
	return m, m.waitForCleanProgress()
}

func (m *Model) handleCleanCategoryDone(msg cleanCategoryDoneMsg) (tea.Model, tea.Cmd) {
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
}

func (m *Model) handleCleanDone(msg cleanDoneMsg) {
	m.report = msg.report
	m.report.Duration = time.Since(m.startTime)
	m.recentDeleted.Clear()
	m.view = ViewReport

	logger.Info("cleaning completed",
		"freedSpace", m.report.FreedSpace,
		"cleanedItems", m.report.CleanedItems,
		"failedItems", m.report.FailedItems,
		"duration", m.report.Duration.String())
}

func (m *Model) handleScanResult(result *types.ScanResult) {
	if result != nil {
		m.scanDoneIDs[result.Category.ID] = true

		// Collect scan errors for display
		if result.Error != nil {
			m.scanErrors = append(m.scanErrors, scanErrorInfo{
				CategoryName: result.Category.Name,
				Error:        result.Error.Error(),
			})
		}

		if len(result.Items) > 0 {
			sort.Slice(result.Items, func(i, j int) bool {
				return result.Items[i].Size > result.Items[j].Size
			})
		}

		if existing, ok := m.resultMap[result.Category.ID]; ok {
			existing.Items = result.Items
			existing.TotalSize = result.TotalSize
			existing.TotalFileCount = result.TotalFileCount
			existing.Error = result.Error
		} else {
			m.results = append(m.results, result)
			m.resultMap[result.Category.ID] = result
		}

		sort.Slice(m.results, func(i, j int) bool {
			return m.results[i].TotalSize > m.results[j].TotalSize
		})
	}
	m.scanCompleted++
	if m.scanCompleted >= m.scanTotal {
		m.scanning = false
		m.finalizeScanResults()
	}
}

func (m *Model) finalizeScanResults() {
	filtered := make([]*types.ScanResult, 0, len(m.results))
	m.resultMap = make(map[string]*types.ScanResult)
	var totalSize int64
	for _, result := range m.results {
		if result.TotalSize <= 0 {
			continue
		}
		filtered = append(filtered, result)
		m.resultMap[result.Category.ID] = result
		totalSize += result.TotalSize
	}
	m.results = filtered

	logger.Info("scan completed",
		"categories", len(filtered),
		"totalSize", totalSize,
		"errors", len(m.scanErrors))
}
