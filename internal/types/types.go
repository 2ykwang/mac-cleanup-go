package types

import "time"

type SafetyLevel string

const (
	SafetyLevelSafe     SafetyLevel = "safe"
	SafetyLevelModerate SafetyLevel = "moderate"
	SafetyLevelRisky    SafetyLevel = "risky"
)

type CleanupMethod string

const (
	MethodTrash     CleanupMethod = "trash"
	MethodPermanent CleanupMethod = "permanent"
	MethodBuiltin   CleanupMethod = "builtin"
	MethodManual    CleanupMethod = "manual"
)

// SafetyHint represents the safety level hint for individual items.
type SafetyHint int

const (
	SafetyHintSafe    SafetyHint = iota // Safe to delete (e.g., dangling images)
	SafetyHintWarning                   // Caution needed (e.g., images with containers)
	SafetyHintDanger                    // Dangerous to delete (e.g., running containers)
)

// Column represents a dynamic column for builtin items.
type Column struct {
	Header string // Column header (e.g., "Status", "Repository")
	Value  string // Column value (e.g., "dangling", "nginx")
}

// SortOrder represents the sorting criterion for items
type SortOrder string

const (
	SortBySize SortOrder = "size" // Size descending (default)
	SortByName SortOrder = "name" // Name ascending (A→Z)
	SortByAge  SortOrder = "age"  // Age ascending (oldest first)
)

// Next returns the next sort order in the rotation cycle
func (s SortOrder) Next() SortOrder {
	switch s {
	case SortBySize:
		return SortByName
	case SortByName:
		return SortByAge
	default:
		return SortBySize
	}
}

// Label returns the display label for the sort order
func (s SortOrder) Label() string {
	switch s {
	case SortBySize:
		return "Size ↓"
	case SortByName:
		return "Name"
	case SortByAge:
		return "Age"
	default:
		return "Size ↓"
	}
}

type Category struct {
	ID       string        `yaml:"id"`
	Name     string        `yaml:"name"`
	Group    string        `yaml:"group"`
	Safety   SafetyLevel   `yaml:"safety"`
	Method   CleanupMethod `yaml:"method"`
	Note     string        `yaml:"note,omitempty"`
	Guide    string        `yaml:"guide,omitempty"`
	Paths    []string      `yaml:"paths,omitempty"`
	CheckCmd string        `yaml:"check_cmd,omitempty"`
}

type Group struct {
	ID    string `yaml:"id"`
	Name  string `yaml:"name"`
	Order int    `yaml:"order"`
}

type Config struct {
	Categories []Category `yaml:"categories"`
	Groups     []Group    `yaml:"groups"`
}

type CleanableItem struct {
	Path        string
	Size        int64
	FileCount   int64
	Name        string
	IsDirectory bool
	ModifiedAt  time.Time

	// Fields for builtin items
	// These are optional; zero values indicate a standard file-based item.
	Columns    []Column   // Dynamic columns for display (nil for file-based items)
	SafetyHint SafetyHint // Safety level hint (Safe=0 is default, suitable for files)
	Selected   bool       // Default selection state for builtin items
}

type ScanResult struct {
	Category       Category
	Items          []CleanableItem
	TotalSize      int64
	TotalFileCount int64
	Error          error
}

type CleanResult struct {
	Category     Category
	CleanedItems int
	SkippedItems int // SIP protected paths skipped during cleanup
	FreedSpace   int64
	Errors       []string
}

type Report struct {
	BeforeSize   int64
	AfterSize    int64
	FreedSpace   int64
	TotalItems   int
	CleanedItems int
	FailedItems  int
	Results      []CleanResult
	Duration     time.Duration
}
