package cli

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/sys/unix"

	"github.com/2ykwang/mac-cleanup-go/internal/tui"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

const (
	defaultReportWidth = 90
	minBlockWidth      = 28
	blockGap           = 2
)

// FormatReport renders a plain-text report for CLI output.
func FormatReport(report *types.Report, dryRun bool) string {
	if report == nil {
		return "No report available.\n"
	}

	styles := newReportStyles()

	title := "Cleanup Report"
	if dryRun {
		title = "Dry Run Report"
	}

	var b strings.Builder
	b.WriteString(styles.Title(title) + "\n")
	b.WriteString(styles.Muted(strings.Repeat("-", len(title))) + "\n")

	modeLine := "Mode: Clean"
	if dryRun {
		modeLine = "Mode: Dry Run"
	}
	b.WriteString(styles.Muted(modeLine) + "\n")

	results := filterResults(report.Results)
	width := reportWidth()
	layout := pickLayout(width)
	b.WriteString(renderSummaryAndHighlights(styles, report, results, dryRun, layout, width))
	b.WriteString("\n")
	if len(results) == 0 {
		b.WriteString(styles.Muted("No items to clean.") + "\n")
		return b.String()
	}

	b.WriteString("\n")
	b.WriteString(styles.Section("Details") + "\n")
	b.WriteString(renderDetails(styles, results, layout, width))

	return b.String()
}

func filterResults(results []types.CleanResult) []types.CleanResult {
	filtered := make([]types.CleanResult, 0, len(results))
	for _, r := range results {
		if r.CleanedItems == 0 && len(r.Errors) == 0 {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

func renderSummaryAndHighlights(
	styles reportStyles,
	report *types.Report,
	results []types.CleanResult,
	dryRun bool,
	layout reportLayout,
	width int,
) string {
	freedLabel := "Recovered"
	if dryRun {
		freedLabel = "Freed (dry-run)"
	}
	summaryLines := []string{
		fmt.Sprintf("%s: %s", freedLabel, styles.Success(utils.FormatSize(report.FreedSpace))),
	}

	switch layout {
	case layoutWide:
		blockWidth := (width - blockGap) / 2
		if blockWidth < minBlockWidth {
			return renderSummaryAndHighlights(styles, report, results, dryRun, layoutMedium, width)
		}
		summaryBlock := renderBlock("Summary", summaryLines, blockWidth, styles)
		highlightsBlock := renderBlock("Highlights", buildHighlights(styles, results, 3), blockWidth, styles)
		return joinBlocks(summaryBlock, highlightsBlock)
	case layoutMedium:
		summaryBlock := renderBlock("Summary", summaryLines, width, styles)
		highlightsBlock := renderBlock("Highlights", buildHighlights(styles, results, 3), width, styles)
		return summaryBlock + "\n\n" + highlightsBlock
	default:
		summaryBlock := renderBlock("Summary", summaryLines, width, styles)
		highlightsBlock := renderBlock("Highlights", buildHighlights(styles, results, 3), width, styles)
		return summaryBlock + "\n\n" + highlightsBlock
	}
}

func buildHighlights(styles reportStyles, results []types.CleanResult, limit int) []string {
	if len(results) == 0 {
		return []string{styles.Muted("No categories to summarize.")}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].FreedSpace > results[j].FreedSpace
	})

	if limit > len(results) {
		limit = len(results)
	}

	lines := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		r := results[i]
		line := fmt.Sprintf("%d. %s - %s (%d items)",
			i+1,
			r.Category.Name,
			utils.FormatSize(r.FreedSpace),
			r.CleanedItems,
		)
		lines = append(lines, styles.Line(line))
	}
	return lines
}

func renderDetails(styles reportStyles, results []types.CleanResult, layout reportLayout, width int) string {
	if layout == layoutNarrow {
		return renderDetailsStack(styles, results)
	}
	return renderDetailsTable(styles, results, width)
}

func renderDetailsTable(styles reportStyles, results []types.CleanResult, width int) string {
	if width <= 0 {
		width = defaultReportWidth
	}

	statusW := 6
	itemsW := 7
	sizeW := 10
	gap := "  "
	gaps := lipgloss.Width(gap) * 3
	minName := 16
	nameW := width - statusW - itemsW - sizeW - gaps
	if nameW < minName {
		nameW = minName
	}

	statusCol := lipgloss.NewStyle().Width(statusW).Align(lipgloss.Left)
	nameCol := lipgloss.NewStyle().Width(nameW).Align(lipgloss.Left)
	itemsCol := lipgloss.NewStyle().Width(itemsW).Align(lipgloss.Right)
	sizeCol := lipgloss.NewStyle().Width(sizeW).Align(lipgloss.Right)

	header := statusCol.Render("STATUS") +
		gap + nameCol.Render("CATEGORY") +
		gap + itemsCol.Render("ITEMS") +
		gap + sizeCol.Render("SIZE")

	var b strings.Builder
	b.WriteString(styles.Muted(header) + "\n")

	for _, r := range results {
		status := statusLabel(r)
		row := statusCol.Render(styles.Status(status)) +
			gap + nameCol.Render(truncateText(r.Category.Name, nameW)) +
			gap + itemsCol.Render(strconv.Itoa(r.CleanedItems)) +
			gap + sizeCol.Render(utils.FormatSize(r.FreedSpace))
		b.WriteString(row + "\n")

		if len(r.Errors) > 0 {
			for _, err := range r.Errors {
				b.WriteString(styles.Muted("  - "+truncateError(err, width-6)) + "\n")
			}
		}
	}

	return b.String()
}

func renderDetailsStack(styles reportStyles, results []types.CleanResult) string {
	var b strings.Builder
	for _, r := range results {
		status := styles.Status(statusLabel(r))
		line := fmt.Sprintf("%s %s — %s (%d items)", status, r.Category.Name, utils.FormatSize(r.FreedSpace), r.CleanedItems)
		b.WriteString(line + "\n")
		if len(r.Errors) > 0 {
			for _, err := range r.Errors {
				b.WriteString(styles.Muted("  - "+truncateError(err, 60)) + "\n")
			}
		}
	}
	return b.String()
}

func statusLabel(r types.CleanResult) string {
	if len(r.Errors) == 0 {
		return "OK"
	}
	if r.CleanedItems > 0 {
		return "WARN"
	}
	return "FAIL"
}

type reportStyles struct {
	enabled bool
	title   lipgloss.Style
	section lipgloss.Style
	success lipgloss.Style
	warn    lipgloss.Style
	danger  lipgloss.Style
	muted   lipgloss.Style
}

func newReportStyles() reportStyles {
	enabled := shouldColorize()
	return reportStyles{
		enabled: enabled,
		title:   lipgloss.NewStyle().Foreground(tui.ColorPrimary).Bold(true),
		section: lipgloss.NewStyle().Foreground(tui.ColorSecondary).Bold(true),
		success: lipgloss.NewStyle().Foreground(tui.ColorSuccess).Bold(true),
		warn:    lipgloss.NewStyle().Foreground(tui.ColorWarning).Bold(true),
		danger:  lipgloss.NewStyle().Foreground(tui.ColorDanger).Bold(true),
		muted:   lipgloss.NewStyle().Foreground(tui.ColorMuted),
	}
}

func (s reportStyles) Title(text string) string {
	if !s.enabled {
		return text
	}
	return s.title.Render(text)
}

func (s reportStyles) Section(text string) string {
	if !s.enabled {
		return text
	}
	return s.section.Render(text)
}

func (s reportStyles) Success(text string) string {
	if !s.enabled {
		return text
	}
	return s.success.Render(text)
}

func (s reportStyles) Danger(text string) string {
	if !s.enabled {
		return text
	}
	return s.danger.Render(text)
}

func (s reportStyles) Status(text string) string {
	switch text {
	case "OK":
		return s.Success(text)
	case "WARN":
		if !s.enabled {
			return text
		}
		return s.warn.Render(text)
	case "FAIL":
		return s.Danger(text)
	default:
		return text
	}
}

func (s reportStyles) Muted(text string) string {
	if !s.enabled {
		return text
	}
	return s.muted.Render(text)
}

func (s reportStyles) Line(text string) string {
	if !s.enabled {
		return text
	}
	return text
}

func shouldColorize() bool {
	return true
}

func renderBlock(title string, lines []string, width int, styles reportStyles) string {
	if width <= 0 {
		width = defaultReportWidth
	}
	lineStyle := lipgloss.NewStyle().Width(width).Align(lipgloss.Left)
	var b strings.Builder
	b.WriteString(lineStyle.Render(styles.Section(title)) + "\n")
	for _, line := range lines {
		b.WriteString(lineStyle.Render(truncateText(line, width)) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func joinBlocks(left, right string) string {
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	spacer := strings.Repeat(" ", blockGap)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, spacer, right)
}

type reportLayout int

const (
	layoutWide reportLayout = iota
	layoutMedium
	layoutNarrow
)

func pickLayout(width int) reportLayout {
	switch {
	case width >= 90:
		return layoutWide
	case width >= 70:
		return layoutMedium
	default:
		return layoutNarrow
	}
}

func reportWidth() int {
	if env := os.Getenv("COLUMNS"); env != "" {
		if value, err := strconv.Atoi(env); err == nil && value > 0 {
			if value > defaultReportWidth {
				return defaultReportWidth
			}
			return value
		}
	}
	if size, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ); err == nil {
		if size.Col > 0 {
			if size.Col > defaultReportWidth {
				return defaultReportWidth
			}
			return int(size.Col)
		}
	}
	return defaultReportWidth
}

func truncateText(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(text) <= width {
		return text
	}
	if width <= 3 {
		return string([]rune(text)[:width])
	}
	target := width - 3
	var b strings.Builder
	current := 0
	for _, r := range text {
		w := lipgloss.Width(string(r))
		if current+w > target {
			break
		}
		b.WriteRune(r)
		current += w
	}
	return b.String() + "..."
}

func truncateError(err string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(err) <= width {
		return err
	}
	if width <= 3 {
		return string([]rune(err)[:width])
	}
	runes := []rune(err)
	return "..." + string(runes[len(runes)-width+3:])
}
