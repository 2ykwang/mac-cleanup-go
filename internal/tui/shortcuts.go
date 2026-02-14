package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
)

// ListKeys defines key bindings for list view
type ListKeys struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Enter  key.Binding
	Delete key.Binding
	Quit   key.Binding
	Help   key.Binding
}

func (k ListKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Select, k.Enter, k.Quit, k.Help}
}

func (k ListKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Select, k.Enter},
		{k.Quit, k.Help},
	}
}

var ListKeyMap = ListKeys{
	Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Select: key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select")),
	Enter:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "preview")),
	Delete: key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "delete")),
	Quit:   key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	Help:   key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
}

// PreviewKeys defines key bindings for preview view
type PreviewKeys struct {
	Up          key.Binding
	Down        key.Binding
	PageUp      key.Binding
	PageDown    key.Binding
	PrevSection key.Binding
	NextSection key.Binding
	Collapse    key.Binding
	Expand      key.Binding
	Select      key.Binding
	Enter       key.Binding
	Back        key.Binding
	Delete      key.Binding
	Open        key.Binding
	Search      key.Binding
	Sort        key.Binding
	Quit        key.Binding
	Help        key.Binding
}

func (k PreviewKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.NextSection, k.Enter, k.Delete, k.Help}
}

func (k PreviewKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.PageUp, k.PageDown},
		{k.PrevSection, k.NextSection},
		{k.Collapse, k.Expand},
		{k.Select, k.Delete},
		{k.Open, k.Search},
		{k.Enter, k.Sort, k.Back},
		{k.Quit, k.Help},
	}
}

var PreviewKeyMap = PreviewKeys{
	Up:          key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:        key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	PageUp:      key.NewBinding(key.WithKeys("shift+up", "K"), key.WithHelp("shift+↑/K", "page up")),
	PageDown:    key.NewBinding(key.WithKeys("shift+down", "J"), key.WithHelp("shift+↓/J", "page down")),
	PrevSection: key.NewBinding(key.WithKeys("shift+tab", "["), key.WithHelp("shift+tab/[", "prev section")),
	NextSection: key.NewBinding(key.WithKeys("tab", "]"), key.WithHelp("tab/]", "next section")),
	Collapse:    key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "collapse")),
	Expand:      key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "expand")),
	Select:      key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle")),
	Enter:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
	Back:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Delete:      key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "delete")),
	Open:        key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open")),
	Search:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	Sort:        key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
	Quit:        key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	Help:        key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
}

// ConfirmKeys defines key bindings for confirm view
type ConfirmKeys struct {
	Scroll key.Binding
	Switch key.Binding
	Select key.Binding
	Cancel key.Binding
	Help   key.Binding
}

func (k ConfirmKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Scroll, k.Switch, k.Select, k.Cancel, k.Help}
}

func (k ConfirmKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Scroll, k.Switch, k.Select, k.Cancel, k.Help},
	}
}

var ConfirmKeyMap = ConfirmKeys{
	Scroll: key.NewBinding(key.WithKeys("up", "down", "k", "j", "pgup", "pgdown"), key.WithHelp("↑/↓", "scroll")),
	Switch: key.NewBinding(key.WithKeys("left", "right", "h", "l", "tab", "shift+tab"), key.WithHelp("←/→/tab", "switch")),
	Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Cancel: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	Help:   key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
}

// ReportKeys defines key bindings for report view
type ReportKeys struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
	Quit  key.Binding
	Help  key.Binding
}

func (k ReportKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Enter, k.Quit, k.Help}
}

func (k ReportKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Enter, k.Quit, k.Help},
	}
}

var ReportKeyMap = ReportKeys{
	Up:    key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:  key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Enter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "rescan")),
	Quit:  key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	Help:  key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
}

// Shortcut represents a single key binding
type Shortcut struct {
	Key  string
	Desc string
}

// FilterTypingShortcuts are shown when user is typing in filter mode
var FilterTypingShortcuts = []Shortcut{
	{"enter", "Apply"},
	{"esc", "Cancel"},
}

// FormatFooter formats shortcuts for footer display
func FormatFooter(shortcuts []Shortcut) string {
	var parts []string
	for _, s := range shortcuts {
		parts = append(parts, s.Key+" "+s.Desc)
	}
	return strings.Join(parts, "  ")
}
