package tui

import (
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

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
	case cleanItemDoneMsg:
		// Add deleted item to recent deletions list
		m.recentDeleted.Push(DeletedItemEntry{
			Path:    msg.path,
			Name:    msg.name,
			Size:    msg.size,
			Success: msg.success,
			ErrMsg:  msg.errMsg,
		})
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
		m.recentDeleted.Clear()
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
