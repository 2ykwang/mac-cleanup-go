package utils

import (
	"io/fs"
	"path/filepath"
	"time"

	"github.com/2ykwang/mac-cleanup-go/internal/logger"
)

// DirStats describes non-directory entries without following symbolic links.
// Recent means the scan stopped early; its totals and newest time are incomplete.
type DirStats struct {
	Size           int64
	FileCount      int64
	NewestFileTime time.Time
	Recent         bool
}

// GetDirStats collects size, count and newest file time in one traversal.
// A nonzero cutoff stops the scan at a file modified at or after that time.
// Ignored names affect size and count, but still contribute to recency.
func GetDirStats(path string, cutoff time.Time, opts ...SizeOption) (DirStats, error) {
	cfg := sizeOpts{collectNewest: true, cutoff: cutoff}
	for _, opt := range opts {
		opt(&cfg)
	}
	return getDirStats(path, cfg)
}

func getDirStatsSequential(path string, cfg sizeOpts) (DirStats, error) {
	var stats DirStats
	err := filepath.WalkDir(path, func(walkPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			logger.Debug("GetDirStats walk error", "path", walkPath, "error", err)
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			logger.Debug("GetDirStats stat failed", "path", walkPath, "error", err)
			return nil
		}
		modified := info.ModTime()
		if modified.After(stats.NewestFileTime) {
			stats.NewestFileTime = modified
		}
		if !cfg.cutoff.IsZero() && !modified.Before(cfg.cutoff) {
			stats.Recent = true
			return filepath.SkipAll
		}
		if cfg.ignoreNames[info.Name()] {
			return nil
		}
		stats.Size += info.Size()
		stats.FileCount++
		return nil
	})
	return stats, err
}
