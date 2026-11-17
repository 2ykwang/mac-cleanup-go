package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

func (m *Model) reportHeader() string {
	var b strings.Builder

	b.WriteString(m.styles.HeaderStyle.Render("Cleanup Complete"))
	b.WriteString("\n\n")

	// Summary
	if m.report.TrashedBytes > 0 {
		fmt.Fprintf(&b, "Moved to Trash: %s\n", formatSize(m.report.TrashedBytes))
		b.WriteString(m.styles.MutedStyle.Render("Trash keeps data until emptied.") + "\n")
	}
	if m.report.PermanentlyDeletedBytes > 0 {
		fmt.Fprintf(&b, "Deleted permanently (est.): %s\n", formatSize(m.report.PermanentlyDeletedBytes))
	}
	b.WriteString("Scan sizes, not freed disk space.\n")
	if m.report.AlreadyAbsentItems > 0 {
		fmt.Fprintf(&b, "Already absent: %d\n", m.report.AlreadyAbsentItems)
	}
	if m.report.UnconfirmedItems > 0 {
		fmt.Fprintf(&b, "Unconfirmed: %d\n", m.report.UnconfirmedItems)
	}
	if m.report.SkippedItems > 0 {
		fmt.Fprintf(&b, "Skipped: %d\n", m.report.SkippedItems)
	}
	if m.report.ProcessedItems > 0 {
		fmt.Fprintf(&b, "Succeeded: %d\n", m.report.ProcessedItems)
	}
	if m.report.FailedItems > 0 {
		fmt.Fprintf(&b, "Failed:    %d\n", m.report.FailedItems)
	}
	fmt.Fprintf(&b, "Time:      %s\n\n", m.report.Duration.Round(time.Millisecond))

	b.WriteString(m.styles.Divider(m.reportDividerWidth()) + "\n")

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
	m.reportScroll = adjustScrollFor(m.reportScroll, m.reportScroll, visible, totalLines)

	start := m.reportScroll
	end := start + visible
	if end > totalLines {
		end = totalLines
	}

	// Show scroll indicator at top if scrolled
	if m.reportScroll > 0 {
		fmt.Fprintf(&b, "%s%d %s\n", m.styles.MutedStyle.Render("  ↑ "), m.reportScroll, m.styles.MutedStyle.Render("more lines above"))
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
		fmt.Fprintf(&b, "%s%d %s\n", m.styles.MutedStyle.Render("  ↓ "), remaining, m.styles.MutedStyle.Render("more lines below"))
	} else {
		b.WriteString("\n")
	}

	b.WriteString(footer)
	return b.String()
}

func (m *Model) buildReportLines() []string {
	var lines []string
	nameWidth, sizeWidth := m.reportColumnWidths()
	if len(m.report.Results) > 0 {
		lines = append(lines, m.styles.MutedStyle.Render("Unmarked actions move to Trash."))
	}
	for _, result := range m.report.Results {
		if result.PlannedBytes == 0 && result.CleanedItems == 0 && result.ProcessedItems == 0 && len(result.Errors) == 0 && result.AlreadyAbsentItems == 0 && result.UnconfirmedItems == 0 && result.SkippedItems == 0 {
			continue
		}
		icon := m.styles.MutedStyle.Render("-")
		switch {
		case len(result.Errors) > 0 && result.ProcessedItems > 0:
			icon = m.styles.WarningStyle.Render("△")
		case len(result.Errors) > 0:
			icon = m.styles.DangerStyle.Render("✗")
		case result.ProcessedItems > 0:
			icon = m.styles.SuccessStyle.Render("✓")
		}
		size := fmt.Sprintf("%*s", sizeWidth, utils.FormatSize(result.TrashedBytes+result.PermanentlyDeletedBytes))
		name := padToWidth(truncateToWidth(result.Category.Name, nameWidth, false), nameWidth)
		lines = append(lines, fmt.Sprintf("  %s %s %s", icon, name, size))
		if result.Category.DisposalMode() != types.DisposalTrash {
			lines = append(lines, "    "+result.Category.DisposalMode().Label())
		}
		if result.Category.DisposalMode() == types.DisposalMixed {
			lines = append(lines, "    Additional cleanup size unknown")
		}
		var exceptions []string
		if len(result.Errors) > 0 {
			exceptions = append(exceptions, fmt.Sprintf("Failed: %d", len(result.Errors)))
		}
		if result.AlreadyAbsentItems > 0 {
			exceptions = append(exceptions, fmt.Sprintf("Already absent: %d", result.AlreadyAbsentItems))
		}
		if result.UnconfirmedItems > 0 {
			exceptions = append(exceptions, fmt.Sprintf("Unconfirmed: %d", result.UnconfirmedItems))
		}
		if result.SkippedItems > 0 {
			exceptions = append(exceptions, fmt.Sprintf("Skipped: %d", result.SkippedItems))
		}
		if len(exceptions) > 0 {
			note := lipgloss.NewStyle().Width(m.width).PaddingLeft(4).Render(strings.Join(exceptions, ", "))
			lines = append(lines, strings.Split(note, "\n")...)
		}
		for _, err := range result.Errors {
			lines = append(lines, m.styles.MutedStyle.Render("    └ ")+truncateToWidth(err, max(1, m.width-6), true))
		}
	}

	return lines
}

func (m *Model) reportDividerWidth() int {
	width := m.width - 4
	if width < 30 {
		width = 30
	}
	return width
}

func (m *Model) reportColumnWidths() (int, int) {
	return m.nameSizeColumns(4, true)
}
