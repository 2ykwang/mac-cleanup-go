package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

func (m *Model) reportHeader() string {
	var b strings.Builder

	b.WriteString(HeaderStyle.Render("Cleanup Complete"))
	b.WriteString("\n\n")

	// Summary
	b.WriteString(fmt.Sprintf("Freed:     %s\n", SizeStyle.Render(formatSize(m.report.FreedSpace))))
	b.WriteString(fmt.Sprintf("Succeeded: %s\n", SuccessStyle.Render(fmt.Sprintf("%d", m.report.CleanedItems))))
	if m.report.FailedItems > 0 {
		b.WriteString(fmt.Sprintf("Failed:    %s\n", DangerStyle.Render(fmt.Sprintf("%d", m.report.FailedItems))))
	}
	b.WriteString(fmt.Sprintf("Time:      %s\n\n", m.report.Duration.Round(time.Millisecond)))

	b.WriteString(Divider(50) + "\n")

	return b.String()
}

func (m *Model) reportFooter() string {
	return "\n" + m.help.View(ReportKeyMap)
}

func (m *Model) viewReport() string {
	// Build report lines if not already built
	if m.reportLines == nil {
		m.reportLines = m.buildReportLines()
	}

	header := m.reportHeader()
	footer := m.reportFooter()
	visible := m.availableLines(header, footer)

	var b strings.Builder
	b.WriteString(header)

	// Adjust scroll
	totalLines := len(m.reportLines)
	m.reportScroll = m.adjustScrollFor(m.reportScroll, m.reportScroll, visible, totalLines)

	start := m.reportScroll
	end := start + visible
	if end > totalLines {
		end = totalLines
	}

	// Show scroll indicator at top if scrolled
	if m.reportScroll > 0 {
		b.WriteString(MutedStyle.Render(fmt.Sprintf("  ↑ %d more lines above\n", m.reportScroll)))
	} else {
		b.WriteString("\n")
	}

	// Display visible lines
	for i := start; i < end; i++ {
		b.WriteString(m.reportLines[i])
		b.WriteString("\n")
	}

	// Show scroll indicator at bottom if more content
	remaining := totalLines - end
	if remaining > 0 {
		b.WriteString(MutedStyle.Render(fmt.Sprintf("  ↓ %d more lines below\n", remaining)))
	} else {
		b.WriteString("\n")
	}

	b.WriteString(footer)
	return b.String()
}

func (m *Model) buildReportLines() []string {
	var lines []string

	// Successful items first (no errors at all)
	hasSuccess := false
	for _, result := range m.report.Results {
		if len(result.Errors) == 0 && result.CleanedItems > 0 {
			if !hasSuccess {
				lines = append(lines, SuccessStyle.Render("Succeeded:"))
				hasSuccess = true
			}
			size := fmt.Sprintf("%*s", colSize, utils.FormatSize(result.FreedSpace))
			lines = append(lines, fmt.Sprintf("  %s %-26s %s", SuccessStyle.Render("✓"), result.Category.Name, SizeStyle.Render(size)))
		}
	}

	// Partial success items (some succeeded, some failed)
	for _, result := range m.report.Results {
		if len(result.Errors) > 0 && result.CleanedItems > 0 {
			if !hasSuccess {
				lines = append(lines, SuccessStyle.Render("Succeeded:"))
				hasSuccess = true
			}
			size := fmt.Sprintf("%*s", colSize, utils.FormatSize(result.FreedSpace))
			lines = append(lines, fmt.Sprintf("  %s %-26s %s",
				WarningStyle.Render("△"),
				result.Category.Name,
				SizeStyle.Render(size)))
		}
	}

	// Failed items
	hasFailed := false
	for _, result := range m.report.Results {
		if len(result.Errors) > 0 {
			if !hasFailed {
				if hasSuccess {
					lines = append(lines, "") // blank line
				}
				lines = append(lines, DangerStyle.Render("Failed:"))
				hasFailed = true
			}

			// Show category with error count
			if result.CleanedItems > 0 {
				lines = append(lines, fmt.Sprintf("  %s %s: %s succeeded, %s failed",
					WarningStyle.Render("⚠"),
					result.Category.Name,
					SuccessStyle.Render(fmt.Sprintf("%d", result.CleanedItems)),
					DangerStyle.Render(fmt.Sprintf("%d", len(result.Errors)))))
			} else {
				lines = append(lines, fmt.Sprintf("  %s %s: %s failed",
					DangerStyle.Render("✗"),
					result.Category.Name,
					DangerStyle.Render(fmt.Sprintf("%d", len(result.Errors)))))
			}

			// Show individual errors (truncate long paths)
			for _, err := range result.Errors {
				displayErr := err
				if len(displayErr) > 60 {
					displayErr = "..." + displayErr[len(displayErr)-57:]
				}
				lines = append(lines, MutedStyle.Render(fmt.Sprintf("    └ %s", displayErr)))
			}
		}
	}

	return lines
}
