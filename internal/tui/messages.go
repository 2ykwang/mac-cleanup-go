package tui

import "github.com/2ykwang/mac-cleanup-go/pkg/types"

// View state
type View int

const (
	ViewList View = iota
	ViewPreview
	ViewConfirm
	ViewCleaning
	ViewReport
	ViewGuide // Manual item guide popup
)

// FilterState represents the search/filter mode state
type FilterState int

const (
	FilterNone    FilterState = iota // No filter active
	FilterTyping                     // User is typing search query
	FilterApplied                    // Filter is applied
)

// Messages
type (
	scanResultMsg    struct{ result *types.ScanResult }
	cleanDoneMsg     struct{ report *types.Report }
	cleanProgressMsg struct {
		categoryName string
		currentItem  string
		current      int
		total        int
	}
	cleanCategoryDoneMsg struct {
		categoryName string
		freedSpace   int64
		cleanedItems int
		errorCount   int
	}
)

// scanErrorInfo holds scan error information for display
type scanErrorInfo struct {
	CategoryName string
	Error        string
}

// drillDownState holds state for directory drill-down navigation
type drillDownState struct {
	path   string
	items  []types.CleanableItem
	cursor int
	scroll int
}

// cleanedCategory tracks a completed category during cleaning
type cleanedCategory struct {
	name       string
	freedSpace int64
	cleaned    int
	errors     int
}

// DeletedItemEntry represents a single deleted item in the progress list
type DeletedItemEntry struct {
	Path    string // Full path of the deleted item
	Name    string // Display name (filename or directory name)
	Size    int64  // Size in bytes
	Success bool   // true if deletion succeeded
	ErrMsg  string // Error message if failed (empty if success)
}

// cleanItemDoneMsg signals that a single item deletion is complete
type cleanItemDoneMsg struct {
	path    string
	name    string
	size    int64
	success bool
	errMsg  string
}
