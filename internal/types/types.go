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

// DisposalMode describes recovery behavior independently of cleanup dispatch.
type DisposalMode string

const (
	DisposalTrash     DisposalMode = "trash"
	DisposalPermanent DisposalMode = "permanent"
	DisposalMixed     DisposalMode = "mixed"
	DisposalManual    DisposalMode = "manual"
	DisposalUnknown   DisposalMode = "unknown"
)

func (d DisposalMode) Label() string {
	switch d {
	case DisposalTrash:
		return "Move to Trash"
	case DisposalPermanent:
		return "Delete permanently"
	case DisposalMixed:
		return "Trash + permanent deletion"
	case DisposalManual:
		return "Manual cleanup"
	default:
		return "Unknown disposal"
	}
}

func (d DisposalMode) ShortLabel() string {
	switch d {
	case DisposalTrash:
		return "Trash"
	case DisposalPermanent:
		return "Permanent"
	case DisposalMixed:
		return "Trash + permanent"
	case DisposalManual:
		return "Manual"
	default:
		return "Unknown"
	}
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
	Disposal DisposalMode  `yaml:"disposal,omitempty"`
	Note     string        `yaml:"note,omitempty"`
	Guide    string        `yaml:"guide,omitempty"`
	Paths    []string      `yaml:"paths,omitempty"`
	CheckCmd string        `yaml:"check_cmd,omitempty"`

	// BlockedByProcesses lists process names that, when running, make this target unavailable.
	BlockedByProcesses []string `yaml:"blocked_by_processes,omitempty"`
}

func (c Category) DisposalMode() DisposalMode {
	switch c.Method {
	case MethodTrash:
		return DisposalTrash
	case MethodPermanent:
		return DisposalPermanent
	case MethodManual:
		return DisposalManual
	case MethodBuiltin:
		switch c.Disposal {
		case DisposalTrash, DisposalPermanent, DisposalMixed:
			return c.Disposal
		}
	}
	return DisposalUnknown
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

// ItemStatus represents the availability state of a cleanable item.
type ItemStatus int

const (
	ItemStatusAvailable ItemStatus = iota
	ItemStatusProcessLocked
)

type CleanableItem struct {
	Path        string
	Size        int64
	FileCount   int64
	Name        string
	DisplayName string
	IsDirectory bool
	ModifiedAt  time.Time
	Status      ItemStatus
}

type ScanResult struct {
	Category       Category
	Items          []CleanableItem
	TotalSize      int64
	TotalFileCount int64
	Error          error
}

// CleanResult byte fields are scan estimates, never measured filesystem free space.
type CleanResult struct {
	Category     Category
	CleanedItems int
	SkippedItems int
	// FreedSpace is deprecated. It preserves legacy success estimates, including absent paths.
	FreedSpace              int64
	PlannedBytes            int64
	TrashedBytes            int64
	PermanentlyDeletedBytes int64
	AlreadyAbsentItems      int
	UnconfirmedItems        int
	ProcessedItems          int
	ItemResults             []ItemCleanedResult
	Errors                  []string
}

// Merge accumulates another CleanResult's counters and errors into this one.
func (r *CleanResult) Merge(other *CleanResult) {
	r.CleanedItems += other.CleanedItems
	r.SkippedItems += other.SkippedItems
	r.FreedSpace += other.FreedSpace
	r.PlannedBytes += other.PlannedBytes
	r.TrashedBytes += other.TrashedBytes
	r.PermanentlyDeletedBytes += other.PermanentlyDeletedBytes
	r.AlreadyAbsentItems += other.AlreadyAbsentItems
	r.UnconfirmedItems += other.UnconfirmedItems
	r.ProcessedItems += other.ProcessedItems
	r.ItemResults = append(r.ItemResults, other.ItemResults...)
	r.Errors = append(r.Errors, other.Errors...)
}

func NewScanResult(category Category) *ScanResult {
	return &ScanResult{
		Category: category,
		Items:    make([]CleanableItem, 0),
	}
}

func NewCleanResult(category Category) *CleanResult {
	return &CleanResult{
		Category: category,
		Errors:   make([]string, 0),
	}
}

type Report struct {
	BeforeSize              int64
	AfterSize               int64
	FreedSpace              int64
	PlannedBytes            int64
	TrashedBytes            int64
	PermanentlyDeletedBytes int64
	AlreadyAbsentItems      int
	UnconfirmedItems        int
	ProcessedItems          int
	SkippedItems            int
	TotalItems              int
	CleanedItems            int
	FailedItems             int
	Results                 []CleanResult
	Duration                time.Duration
}

func (r *Report) AddResult(result *CleanResult) {
	r.Results = append(r.Results, *result)
	r.FreedSpace += result.FreedSpace
	r.PlannedBytes += result.PlannedBytes
	r.TrashedBytes += result.TrashedBytes
	r.PermanentlyDeletedBytes += result.PermanentlyDeletedBytes
	r.AlreadyAbsentItems += result.AlreadyAbsentItems
	r.UnconfirmedItems += result.UnconfirmedItems
	r.ProcessedItems += result.ProcessedItems
	r.SkippedItems += result.SkippedItems
	r.CleanedItems += result.CleanedItems
	r.FailedItems += len(result.Errors)
}

func PlannedSize(items []CleanableItem) int64 {
	var size int64
	for _, item := range items {
		size += item.Size
	}
	return size
}

type CleanProgress struct {
	CategoryName string
	CurrentItem  string
	Current      int
	Total        int
}

type ItemOutcome int

const (
	ItemOutcomeUnknown ItemOutcome = iota
	ItemOutcomeProcessed
	ItemOutcomeFailed
	ItemOutcomeSkipped
	ItemOutcomeAbsent
	ItemOutcomeUnconfirmed
)

type ItemCleanedResult struct {
	Path    string
	Name    string
	Size    int64
	Success bool
	ErrMsg  string
	Outcome ItemOutcome
}

type CategoryCleanedResult struct {
	CategoryName            string
	FreedSpace              int64
	TrashedBytes            int64
	PermanentlyDeletedBytes int64
	ProcessedItems          int
	CleanedItems            int
	ErrorCount              int
}

type CleanCallbacks struct {
	OnProgress     func(CleanProgress)
	OnItemDone     func(ItemCleanedResult)
	OnCategoryDone func(CategoryCleanedResult)
}

type BatchTrashOptions struct {
	Category Category
	Filter   func(CleanableItem) bool
	Validate func(CleanableItem) error
}
