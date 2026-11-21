package target

import (
	"time"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

type ageScanner interface {
	scanWithAge(ageFilter) (*types.ScanResult, error)
}

type ageFilter struct {
	cutoff      time.Time
	defaultDays int
}

func (f ageFilter) apply(result *types.ScanResult) {
	useDefault := f.cutoff.IsZero()
	if useDefault {
		f.cutoff = time.Now().AddDate(0, 0, -f.defaultDays)
	}
	items := result.Items[:0]
	result.TotalSize, result.TotalFileCount = 0, 0
	for _, item := range result.Items {
		if (!useDefault && item.ModifiedAt.IsZero()) || !item.ModifiedAt.Before(f.cutoff) {
			continue
		}
		items = append(items, item)
		result.TotalSize += item.Size
		result.TotalFileCount += item.FileCount
	}
	clear(result.Items[len(items):])
	result.Items = items
}

// ScanBefore uses the target's default age policy when cutoff is zero.
func ScanBefore(t Target, cutoff time.Time) (*types.ScanResult, error) {
	if cutoff.IsZero() {
		return t.Scan()
	}
	filter := ageFilter{cutoff: cutoff}
	if s, ok := t.(ageScanner); ok {
		return s.scanWithAge(filter)
	}
	result, err := t.Scan()
	if err == nil && result != nil && result.Error == nil {
		filter.apply(result)
	}
	return result, err
}
