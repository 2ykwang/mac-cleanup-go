package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func (m *Model) doClean() tea.Cmd {
	// Prepare jobs using CleanService (handles filtering and excluded items)
	jobs := m.cleanService.PrepareJobs(m.resultMap, m.selected, m.excluded)

	// Calculate total items for progress tracking
	totalItems := 0
	for _, job := range jobs {
		totalItems += len(job.Items)
	}
	m.cleaningTotal = totalItems
	m.cleaningCurrent = 0
	m.cleaningCompleted = nil // Reset completed list

	// Create channels for progress communication
	m.cleanProgressChan = make(chan cleanProgressMsg, 1)
	m.cleanDoneChan = make(chan cleanDoneMsg, 1)
	m.cleanCategoryDoneCh = make(chan cleanCategoryDoneMsg, 1)
	m.cleanItemDoneChan = make(chan cleanItemDoneMsg, 1)

	// Create callbacks that bridge CleanService to TUI channels
	callbacks := types.CleanCallbacks{
		OnProgress: func(p types.CleanProgress) {
			m.cleanProgressChan <- cleanProgressMsg{
				categoryName: p.CategoryName,
				currentItem:  p.CurrentItem,
				current:      p.Current,
				total:        p.Total,
			}
		},
		OnItemDone: func(r types.ItemCleanedResult) {
			m.cleanItemDoneChan <- cleanItemDoneMsg{
				path:    r.Path,
				name:    r.Name,
				size:    r.Size,
				success: r.Success,
				errMsg:  r.ErrMsg,
			}
		},
		OnCategoryDone: func(r types.CategoryCleanedResult) {
			m.cleanCategoryDoneCh <- cleanCategoryDoneMsg{
				categoryName: r.CategoryName,
				freedSpace:   r.FreedSpace,
				cleanedItems: r.CleanedItems,
				errorCount:   r.ErrorCount,
			}
		},
	}

	// Start cleaning in background goroutine
	go func() {
		report := m.cleanService.Clean(jobs, callbacks)
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
		case itemDone := <-m.cleanItemDoneChan:
			return itemDone
		case catDone := <-m.cleanCategoryDoneCh:
			return catDone
		case done := <-m.cleanDoneChan:
			return done
		}
	}
}
