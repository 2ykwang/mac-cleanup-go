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
	MethodCommand   CleanupMethod = "command"
	MethodBuiltin   CleanupMethod = "builtin"
	MethodManual    CleanupMethod = "manual"
)

type Category struct {
	ID       string        `yaml:"id"`
	Name     string        `yaml:"name"`
	Group    string        `yaml:"group"`
	Safety   SafetyLevel   `yaml:"safety"`
	Method   CleanupMethod `yaml:"method"`
	Note     string        `yaml:"note,omitempty"`
	Guide    string        `yaml:"guide,omitempty"`
	Paths    []string      `yaml:"paths,omitempty"`
	Command  string        `yaml:"command,omitempty"`
	Check    string        `yaml:"check,omitempty"`
	CheckCmd string        `yaml:"check_cmd,omitempty"`
	Sudo     bool          `yaml:"sudo,omitempty"`
	DaysOld  int           `yaml:"days_old,omitempty"`
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
