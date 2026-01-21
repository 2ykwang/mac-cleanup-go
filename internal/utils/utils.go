package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	osUserHomeDir = os.UserHomeDir
	osReadDir     = os.ReadDir
	execCommand   = exec.Command
	execLookPath  = exec.LookPath
)

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := osUserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatAge formats a time.Time as a human-readable age string
// Examples: "5m", "3h", "7d", "2mo", "1y"
func FormatAge(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	duration := time.Since(t)

	minutes := int(duration.Minutes())
	hours := int(duration.Hours())
	days := hours / 24
	months := days / 30
	years := days / 365

	switch {
	case hours < 1:
		if minutes < 1 {
			return "<1m"
		}
		return fmt.Sprintf("%dm", minutes)
	case hours < 24:
		return fmt.Sprintf("%dh", hours)
	case days < 30:
		return fmt.Sprintf("%dd", days)
	case months < 12:
		return fmt.Sprintf("%dmo", months)
	default:
		return fmt.Sprintf("%dy", years)
	}
}

func PathExists(path string) bool {
	expanded := ExpandPath(path)
	_, err := os.Stat(expanded)
	return err == nil
}

var CommandExists = func(cmd string) bool {
	_, err := execLookPath(cmd)
	return err == nil
}

func GetDirSizeWithCount(path string) (int64, int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, 0, err
	}
	if !info.IsDir() {
		return info.Size(), 1, nil
	}

	dirWorkers := runtime.GOMAXPROCS(0)
	if dirWorkers < 2 {
		return getDirSizeWithCountSequential(path)
	}
	if dirWorkers > 16 {
		dirWorkers = 16
	}

	var size, count int64
	var (
		mu      sync.Mutex
		cond    = sync.NewCond(&mu)
		queue   = []string{path}
		pending = 1
		done    = false
	)

	var wg sync.WaitGroup
	wg.Add(dirWorkers)
	for i := 0; i < dirWorkers; i++ {
		go func() {
			defer wg.Done()
			for {
				mu.Lock()
				for len(queue) == 0 && !done {
					cond.Wait()
				}
				if done {
					mu.Unlock()
					return
				}
				dir := queue[len(queue)-1]
				queue = queue[:len(queue)-1]
				mu.Unlock()

				entries, err := os.ReadDir(dir)
				var (
					localSize  int64
					localCount int64
					subDirs    []string
				)
				if err == nil {
					for _, entry := range entries {
						entryPath := filepath.Join(dir, entry.Name())
						entryType := entry.Type()
						if entryType == 0 || entryType&os.ModeSymlink != 0 {
							info, err := os.Lstat(entryPath)
							if err != nil {
								continue
							}
							if info.IsDir() {
								subDirs = append(subDirs, entryPath)
								continue
							}
							localSize += info.Size()
							localCount++
							continue
						}
						if entryType.IsDir() {
							subDirs = append(subDirs, entryPath)
							continue
						}
						entryInfo, err := entry.Info()
						if err != nil {
							continue
						}
						localSize += entryInfo.Size()
						localCount++
					}
				}

				if localSize != 0 {
					atomic.AddInt64(&size, localSize)
				}
				if localCount != 0 {
					atomic.AddInt64(&count, localCount)
				}

				mu.Lock()
				if len(subDirs) > 0 {
					queue = append(queue, subDirs...)
					pending += len(subDirs)
					cond.Broadcast()
				}
				pending--
				if pending == 0 {
					done = true
					cond.Broadcast()
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	return size, count, nil
}

func getDirSizeWithCountSequential(path string) (int64, int64, error) {
	var size, count int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
			count++
		}
		return nil
	})
	return size, count, err
}

func GetDirSize(path string) (int64, error) {
	size, _, err := GetDirSizeWithCount(path)
	return size, err
}

func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if info.IsDir() {
		return GetDirSize(path)
	}
	return info.Size(), nil
}

func GlobPaths(pattern string) ([]string, error) {
	expanded := ExpandPath(pattern)
	return filepath.Glob(expanded)
}

// CheckFullDiskAccess checks if the app has Full Disk Access permission
// by attempting to read the Trash directory
func CheckFullDiskAccess() bool {
	trashPath := ExpandPath("~/.Trash")
	_, err := osReadDir(trashPath)
	return err == nil
}

// StripGlobPattern removes glob patterns from a path to find an existing parent directory
func StripGlobPattern(path string) string {
	expanded := ExpandPath(path)

	// Try the path as-is first
	if _, err := os.Stat(expanded); err == nil {
		return expanded
	}

	// Strip trailing glob patterns and find existing parent
	for strings.ContainsAny(expanded, "*?[") {
		// Check for glob characters

		// Get parent directory
		parent := filepath.Dir(expanded)
		if parent == expanded {
			break
		}
		expanded = parent

		// Check if parent exists
		if _, err := os.Stat(expanded); err == nil {
			return expanded
		}
	}

	return expanded
}

// OpenInFinder opens the specified path in macOS Finder.
// For files, it opens the parent directory with the file selected (-R flag).
// For directories, it opens the directory directly.
func OpenInFinder(path string) error {
	expanded := ExpandPath(path)

	info, err := os.Stat(expanded)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return execCommand("open", expanded).Run()
	}
	return execCommand("open", "-R", expanded).Run()
}
