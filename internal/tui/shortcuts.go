package tui

import "strings"

// Shortcut represents a single key binding
type Shortcut struct {
	Key  string
	Desc string
}

// ShortcutGroup represents a group of related shortcuts
type ShortcutGroup struct {
	Name      string
	Shortcuts []Shortcut
}

// ViewShortcuts maps each View to its shortcut groups for help popup
var ViewShortcuts = map[View][]ShortcutGroup{
	ViewList: {
		{Name: "Navigation", Shortcuts: []Shortcut{
			{"↑↓", "Move"},
			{"enter", "Preview"},
		}},
		{Name: "Selection", Shortcuts: []Shortcut{
			{"space", "Select"},
			{"a", "Select All"},
			{"d", "Deselect All"},
		}},
		{Name: "Actions", Shortcuts: []Shortcut{
			{"y", "Delete selected"},
			{"q", "Quit"},
			{"?", "Help"},
		}},
	},
	ViewPreview: {
		{Name: "Navigation", Shortcuts: []Shortcut{
			{"↑↓", "Move"},
			{"←→", "Switch tab"},
			{"PgUp/Dn", "Page scroll"},
			{"Home/End", "Jump to start/end"},
		}},
		{Name: "Search & Sort", Shortcuts: []Shortcut{
			{"/", "Search"},
			{"s", "Sort toggle"},
		}},
		{Name: "Actions", Shortcuts: []Shortcut{
			{"space", "Toggle select"},
			{"o", "Open in Finder"},
			{"y", "Delete selected"},
			{"esc", "Back to list"},
			{"?", "Help"},
		}},
	},
	ViewConfirm: {
		{Name: "Actions", Shortcuts: []Shortcut{
			{"y", "Confirm delete"},
			{"n/esc", "Cancel"},
			{"?", "Help"},
		}},
	},
	ViewReport: {
		{Name: "Navigation", Shortcuts: []Shortcut{
			{"↑↓", "Scroll"},
		}},
		{Name: "Actions", Shortcuts: []Shortcut{
			{"enter", "Rescan"},
			{"q", "Quit"},
			{"?", "Help"},
		}},
	},
}

// DrillDownShortcuts are shown when in drill-down mode (inside a directory)
var DrillDownShortcuts = []ShortcutGroup{
	{Name: "Navigation", Shortcuts: []Shortcut{
		{"↑↓", "Move"},
		{"enter", "Enter folder"},
		{"esc/⌫", "Go back"},
	}},
	{Name: "Actions", Shortcuts: []Shortcut{
		{"s", "Sort toggle"},
		{"o", "Open in Finder"},
		{"q", "Quit"},
		{"?", "Help"},
	}},
}

// FilterTypingShortcuts are shown when user is typing in filter mode
var FilterTypingShortcuts = []Shortcut{
	{"enter", "Apply"},
	{"esc", "Cancel"},
}

// footerShortcutsMap defines essential shortcuts for footer display (4-5 items max)
var footerShortcutsMap = map[View][]Shortcut{
	ViewList: {
		{"↑↓", "Move"},
		{"space", "Select"},
		{"enter", "Preview"},
		{"y", "Delete"},
		{"?", "Help"},
	},
	ViewPreview: {
		{"↑↓", "Move"},
		{"←→", "Tab"},
		{"space", "Toggle"},
		{"esc", "Back"},
		{"?", "Help"},
	},
	ViewConfirm: {
		{"y", "Confirm"},
		{"n", "Cancel"},
		{"?", "Help"},
	},
	ViewReport: {
		{"↑↓", "Scroll"},
		{"enter", "Rescan"},
		{"q", "Quit"},
		{"?", "Help"},
	},
}

// drillDownFooterShortcuts are shown in drill-down mode
var drillDownFooterShortcuts = []Shortcut{
	{"↑↓", "Move"},
	{"enter", "Enter"},
	{"o", "Open"},
	{"esc", "Back"},
	{"?", "Help"},
}

// FooterShortcuts returns the essential shortcuts for footer display
func FooterShortcuts(v View) []Shortcut {
	if shortcuts, ok := footerShortcutsMap[v]; ok {
		return shortcuts
	}
	return []Shortcut{{"?", "Help"}}
}

// DrillDownFooterShortcuts returns shortcuts for drill-down mode
func DrillDownFooterShortcuts() []Shortcut {
	return drillDownFooterShortcuts
}

// FormatFooter formats shortcuts for footer display
func FormatFooter(shortcuts []Shortcut) string {
	var parts []string
	for _, s := range shortcuts {
		parts = append(parts, s.Key+" "+s.Desc)
	}
	return strings.Join(parts, "  ")
}
