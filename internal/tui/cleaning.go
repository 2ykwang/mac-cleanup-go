package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/2ykwang/mac-cleanup-go/pkg/types"
)

func (m *Model) doClean() tea.Cmd {
	type cleanJob struct {
		category types.Category
		items    []types.CleanableItem
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
			category: r.Category,
			items:    items,
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
	m.cleanItemDoneChan = make(chan cleanItemDoneMsg, 1)

	// Start cleaning in background goroutine
	go func() {
		report := &types.Report{Results: make([]types.CleanResult, 0)}
		currentItem := 0

		for _, job := range jobs {
			var result *types.CleanResult
			cat := job.category

			// Builtin methods: batch processing (category-level progress)
			// Other methods: item-by-item processing (item-level progress)
			if cat.Method == types.MethodBuiltin {
				m.cleanProgressChan <- cleanProgressMsg{
					categoryName: cat.Name,
					currentItem:  "",
					current:      currentItem,
					total:        totalItems,
				}

				result = m.cleaner.Clean(cat, job.items)
				currentItem += len(job.items)
			} else {
				// Clean items one by one for progress tracking
				itemResult := &types.CleanResult{
					Category: cat,
					Errors:   make([]string, 0),
				}
				for _, item := range job.items {
					currentItem++

					// Send progress update
					m.cleanProgressChan <- cleanProgressMsg{
						categoryName: cat.Name,
						currentItem:  item.Name,
						current:      currentItem,
						total:        totalItems,
					}

					singleResult := m.cleaner.Clean(cat, []types.CleanableItem{item})
					itemResult.FreedSpace += singleResult.FreedSpace
					itemResult.CleanedItems += singleResult.CleanedItems
					itemResult.Errors = append(itemResult.Errors, singleResult.Errors...)

					// Send item done message for recent deletions list
					success := len(singleResult.Errors) == 0
					errMsg := ""
					if !success && len(singleResult.Errors) > 0 {
						errMsg = singleResult.Errors[0]
					}
					m.cleanItemDoneChan <- cleanItemDoneMsg{
						path:    item.Path,
						name:    item.Name,
						size:    item.Size,
						success: success,
						errMsg:  errMsg,
					}
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
					categoryName: cat.Name,
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
		case itemDone := <-m.cleanItemDoneChan:
			return itemDone
		case catDone := <-m.cleanCategoryDoneCh:
			return catDone
		case done := <-m.cleanDoneChan:
			return done
		}
	}
}
