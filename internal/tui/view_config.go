package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/2ykwang/mac-cleanup-go/internal/styles"
	"github.com/2ykwang/mac-cleanup-go/internal/target"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/userconfig"
)

type configItem struct {
	category types.Category
	disabled bool
	size     int64
	scanned  bool
}

type configScanResultMsg struct {
	categoryID string
	size       int64
}

// ConfigModel is a standalone TUI for CLI configuration.
type ConfigModel struct {
	cfg      *types.Config
	userCfg  *userconfig.UserConfig
	registry *target.Registry

	items    []configItem
	selected map[string]bool
	cursor   int
	scroll   int

	width  int
	height int

	showIntro bool
	status    string
	err       error
}

// NewConfigModel creates a new config TUI model.
func NewConfigModel(cfg *types.Config) *ConfigModel {
	userCfg, _ := userconfig.Load()

	registry, err := target.DefaultRegistry(cfg)
	if err != nil {
		registry = target.NewRegistry()
	}

	m := &ConfigModel{
		cfg:      cfg,
		userCfg:  userCfg,
		registry: registry,
		items:    make([]configItem, 0),
		selected: make(map[string]bool),
		err:      err,
	}

	m.initItems()
	m.initSelection()
	m.showIntro = true

	return m
}

func (m *ConfigModel) initItems() {
	for _, cat := range m.cfg.Categories {
		if cat.Safety == types.SafetyLevelRisky {
			continue
		}
		disabled := cat.Method == types.MethodManual
		m.items = append(m.items, configItem{
			category: cat,
			disabled: disabled,
		})
	}
}

func (m *ConfigModel) initSelection() {
	itemIDs := make(map[string]bool, len(m.items))
	for _, item := range m.items {
		if !item.disabled {
			itemIDs[item.category.ID] = true
		}
	}

	for _, id := range m.userCfg.GetSelectedTargets() {
		if itemIDs[id] {
			m.selected[id] = true
		}
	}
}

// Init implements tea.Model.
func (m *ConfigModel) Init() tea.Cmd {
	return m.startScan()
}

func (m *ConfigModel) startScan() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(m.items))
	for _, item := range m.items {
		cat := item.category
		t, ok := m.registry.Get(cat.ID)
		if !ok {
			id := cat.ID
			cmds = append(cmds, func() tea.Msg {
				return configScanResultMsg{categoryID: id, size: 0}
			})
			continue
		}
		cmds = append(cmds, func() tea.Msg {
			result, _ := t.Scan()
			if result == nil {
				return configScanResultMsg{categoryID: t.Category().ID, size: 0}
			}
			return configScanResultMsg{categoryID: t.Category().ID, size: result.TotalSize}
		})
	}
	return tea.Batch(cmds...)
}

// Update implements tea.Model.
func (m *ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.showIntro {
			return m.handleIntroKey(msg)
		}
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case configScanResultMsg:
		for i, item := range m.items {
			if item.category.ID == msg.categoryID {
				m.items[i].size = msg.size
				m.items[i].scanned = true
				break
			}
		}
	}
	return m, nil
}

func (m *ConfigModel) handleIntroKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", " ", "esc", "q":
		m.showIntro = false
	}
	return m, nil
}

func (m *ConfigModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		m.status = ""
	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
		m.status = ""
	case "?":
		m.showIntro = true
	case " ":
		m.toggleSelection()
	case "enter", "s":
		if err := m.saveSelection(); err != nil {
			m.status = "Save failed: " + err.Error()
			return m, nil
		}
		return m, tea.Quit
	}
	return m, nil
}

func (m *ConfigModel) toggleSelection() {
	if len(m.items) == 0 || m.cursor >= len(m.items) {
		return
	}
	item := m.items[m.cursor]
	if item.disabled {
		m.status = "This target can't be selected."
		return
	}
	id := item.category.ID
	if m.selected[id] {
		delete(m.selected, id)
	} else {
		m.selected[id] = true
	}
}

func (m *ConfigModel) saveSelection() error {
	var selected []string
	for _, item := range m.items {
		if m.selected[item.category.ID] {
			selected = append(selected, item.category.ID)
		}
	}
	m.userCfg.SetSelectedTargets(selected)
	return m.userCfg.Save()
}

// View implements tea.Model.
func (m *ConfigModel) View() string {
	if m.err != nil {
		return "Error: " + m.err.Error() + "\n\nPress q to quit."
	}

	base := m.viewList()
	if m.showIntro {
		return overlayCentered(base, m.introDialog(), m.width, m.height)
	}
	return base
}

func (m *ConfigModel) viewList() string {
	width := m.width
	if width <= 0 {
		width = 80
	}

	sizeWidth := colSize
	nameWidth := max(min(width-listPrefixWidth-sizeWidth-1, colName), 10)

	var header strings.Builder
	header.WriteString(styles.HeaderStyle.Render("Target Selection") + "\n")
	header.WriteString(styles.MutedStyle.Render("Select cleanup targets") + "\n")
	header.WriteString(styles.Divider(clampWidth(width-4, 30)) + "\n")
	colHeader := fmt.Sprintf("%*s%-*s %*s",
		listPrefixWidth, "", nameWidth, "Name", sizeWidth, "Size")
	header.WriteString(styles.MutedStyle.Render(colHeader) + "\n")
	headerStr := header.String()

	var footer strings.Builder
	footer.WriteString(styles.Divider(clampWidth(width-4, 30)) + "\n")
	footer.WriteString(styles.MutedStyle.Render(fmt.Sprintf("Selected: %d", m.selectedCount())) + "\n")
	if m.status != "" {
		footer.WriteString(styles.WarningStyle.Render(m.status) + "\n")
	}
	footer.WriteString(styles.HelpStyle.Render(FormatFooter(configShortcuts)))
	footerStr := footer.String()

	visible := m.height - countLines(headerStr) - countLines(footerStr)
	total := len(m.items)

	// Reserve 1 line for scroll indicator when needed
	if total > visible && visible > 1 {
		visible--
	}
	if visible < 1 {
		visible = 1
	}

	m.scroll = adjustScrollFor(m.cursor, m.scroll, visible, total)

	var b strings.Builder
	b.WriteString(headerStr)

	if total == 0 {
		b.WriteString(styles.MutedStyle.Render("No available targets found.") + "\n")
	} else {
		end := min(m.scroll+visible, total)
		for i := m.scroll; i < end; i++ {
			b.WriteString(m.renderItemLine(i, m.items[i], width))
			b.WriteString("\n")
		}
		if total > visible+1 {
			info := fmt.Sprintf("  %d-%d of %d", m.scroll+1, end, total)
			b.WriteString(styles.MutedStyle.Render(info) + "\n")
		}
	}

	b.WriteString(footerStr)
	return b.String()
}

func (m *ConfigModel) renderItemLine(index int, item configItem, width int) string {
	cursor := "  "
	if index == m.cursor {
		cursor = styles.CursorStyle.Render("▸ ")
	}

	checkbox := styles.MutedStyle.Render("[ ]")
	if item.disabled {
		checkbox = styles.MutedStyle.Render(" - ")
	} else if m.selected[item.category.ID] {
		checkbox = styles.SuccessStyle.Render("[✓]")
	}

	dot := safetyDot(item.category.Safety)

	sizeWidth := colSize
	nameWidth := max(min(width-listPrefixWidth-sizeWidth-1, colName), 10)

	name := padToWidth(truncateToWidth(item.category.Name, nameWidth, false), nameWidth)
	if item.disabled {
		name = styles.MutedStyle.Render(name)
	}

	var sizeText string
	if !item.scanned {
		sizeText = styles.MutedStyle.Render(fmt.Sprintf("%*s", sizeWidth, "..."))
	} else if item.size > 0 {
		sizeText = styles.SizeStyle.Render(fmt.Sprintf("%*s", sizeWidth, formatSize(item.size)))
	} else {
		sizeText = styles.MutedStyle.Render(fmt.Sprintf("%*s", sizeWidth, "0 B"))
	}

	return fmt.Sprintf("%s%s %s %s %s", cursor, checkbox, dot, name, sizeText)
}

func (m *ConfigModel) selectedCount() int {
	count := 0
	for _, item := range m.items {
		if m.selected[item.category.ID] {
			count++
		}
	}
	return count
}

func clampWidth(width, min int) int {
	if width < min {
		return min
	}
	return width
}

var configShortcuts = []Shortcut{
	{"↑/↓", "Move"},
	{"space", "Select"},
	{"s", "Save"},
	{"?", "Help"},
	{"q", "Cancel"},
}

func (m *ConfigModel) introDialog() string {
	boxWidth := min(64, m.width-4)
	if boxWidth < 48 {
		boxWidth = min(m.width-2, 48)
	}
	if boxWidth < 28 {
		boxWidth = m.width
	}
	if boxWidth <= 0 {
		boxWidth = 60
	}

	contentWidth := max(boxWidth-6, 0)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorSecondary)

	var b strings.Builder

	// Title
	b.WriteString(styles.HeaderStyle.Render("Getting Started"))
	b.WriteString("\n\n")

	// Description
	b.WriteString(styles.TextStyle.Render("Configure cleanup targets for CLI mode."))
	b.WriteString("\n")
	b.WriteString(styles.TextStyle.Render("Selected targets run with ") + styles.SelectedStyle.Render("--clean") + styles.TextStyle.Render(" flag."))
	b.WriteString("\n")
	b.WriteString(styles.TextStyle.Render("Files are moved to Trash (recoverable)."))
	b.WriteString("\n\n")

	// Section: Target types
	b.WriteString(sectionStyle.Render("Target Types"))
	b.WriteString("\n")
	b.WriteString("  " + safetyDot(types.SafetyLevelSafe) + " Safe")
	b.WriteString("\n")
	b.WriteString(styles.MutedStyle.Render("    Auto-regenerated caches and logs"))
	b.WriteString("\n")
	b.WriteString("  " + safetyDot(types.SafetyLevelModerate) + " Moderate")
	b.WriteString("\n")
	b.WriteString(styles.MutedStyle.Render("    May require re-download or re-login"))
	b.WriteString("\n\n")

	b.WriteString(styles.Divider(contentWidth) + "\n\n")

	// Commands
	infoLabel := lipgloss.NewStyle().Foreground(styles.ColorMuted).Width(10)
	b.WriteString(infoLabel.Render("Clean") + styles.MutedStyle.Render("mac-cleanup ") + styles.SelectedStyle.Render("--clean"))
	b.WriteString("\n")
	b.WriteString(infoLabel.Render("Dry run") + styles.MutedStyle.Render("mac-cleanup ") + styles.SelectedStyle.Render("--clean --dry-run"))
	b.WriteString("\n\n")

	// Button
	button := styles.ButtonActiveStyle.Render("Got it")
	if contentWidth > 0 {
		button = lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, button)
	}
	b.WriteString(button)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorBorder).
		Padding(1, 2).
		Width(boxWidth)

	return boxStyle.Render(b.String())
}
