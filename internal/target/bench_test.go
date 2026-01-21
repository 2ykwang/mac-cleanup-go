//go:build perf

package target

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

const (
	benchDepthSmall    = 3
	benchDepthMedium   = 5
	benchDepthLarge    = 8
	benchFilesPerDir   = 100
	benchFanout        = 3
	benchMarkerName    = ".bench_marker"
	benchMarkerVersion = "v1"
)

// Shared benchmark directories created once in TestMain (reused if BENCH_DATA_DIR is set).
var (
	benchDirSmall  string
	benchDirMedium string
	benchDirLarge  string
)

func TestMain(m *testing.M) {
	root := os.Getenv("BENCH_DATA_DIR")
	cleanup := func() {}
	if root == "" {
		tempRoot, err := os.MkdirTemp("", "bench-trees-")
		if err != nil {
			panic("failed to create temp root: " + err.Error())
		}
		root = tempRoot
		cleanup = func() { _ = os.RemoveAll(tempRoot) }
	}

	if err := ensureBenchmarkTrees(root); err != nil {
		panic("failed to prepare benchmark trees: " + err.Error())
	}

	// Run benchmarks
	code := m.Run()

	// Cleanup
	cleanup()

	os.Exit(code)
}

func ensureBenchmarkTrees(root string) error {
	markerPath := filepath.Join(root, benchMarkerName)
	marker := benchMarkerContent()

	if data, err := os.ReadFile(markerPath); err == nil && string(data) == marker && benchDirsExist(root) {
		benchDirSmall = filepath.Join(root, "small")
		benchDirMedium = filepath.Join(root, "medium")
		benchDirLarge = filepath.Join(root, "large")
		return nil
	}

	if err := os.RemoveAll(root); err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}

	// Create all trees once (in parallel for speed).
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		benchDirSmall = createBenchmarkTreeOnce(filepath.Join(root, "small"), benchDepthSmall, benchFilesPerDir)
	}()
	go func() {
		defer wg.Done()
		benchDirMedium = createBenchmarkTreeOnce(filepath.Join(root, "medium"), benchDepthMedium, benchFilesPerDir)
	}()
	go func() {
		defer wg.Done()
		benchDirLarge = createBenchmarkTreeOnce(filepath.Join(root, "large"), benchDepthLarge, benchFilesPerDir)
	}()

	wg.Wait()

	if err := os.WriteFile(markerPath, []byte(marker), 0o644); err != nil {
		return err
	}

	return nil
}

func benchMarkerContent() string {
	return fmt.Sprintf(
		"version=%s\nfanout=%d\nfiles=%d\ndepthSmall=%d\ndepthMedium=%d\ndepthLarge=%d\n",
		benchMarkerVersion,
		benchFanout,
		benchFilesPerDir,
		benchDepthSmall,
		benchDepthMedium,
		benchDepthLarge,
	)
}

func benchDirsExist(root string) bool {
	dirs := []string{
		filepath.Join(root, "small"),
		filepath.Join(root, "medium"),
		filepath.Join(root, "large"),
	}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); err != nil {
			return false
		}
	}
	return true
}

// createBenchmarkTreeOnce creates a directory tree with parallel file creation.
func createBenchmarkTreeOnce(root string, depth, filesPerDir int) string {
	if err := os.MkdirAll(root, 0o755); err != nil {
		panic("failed to create root: " + err.Error())
	}

	// Collect all directories first
	var dirs []string
	var collectDirs func(path string, currentDepth int)
	collectDirs = func(path string, currentDepth int) {
		dirs = append(dirs, path)
		if currentDepth < depth {
			for i := 0; i < benchFanout; i++ {
				subDir := filepath.Join(path, "dir"+string(rune('a'+i)))
				if err := os.MkdirAll(subDir, 0o755); err != nil {
					panic("failed to create dir: " + err.Error())
				}
				collectDirs(subDir, currentDepth+1)
			}
		}
	}
	collectDirs(root, 1)

	// Create files in parallel (one goroutine per directory)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 32) // Limit concurrent goroutines
	data := make([]byte, 1024)     // 1KB, shared read-only

	for _, dir := range dirs {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			for i := 0; i < filesPerDir; i++ {
				filePath := filepath.Join(d, "file"+string(rune('a'+i))+".txt")
				if err := os.WriteFile(filePath, data, 0o644); err != nil {
					panic("failed to create file: " + err.Error())
				}
			}
		}(dir)
	}
	wg.Wait()

	return root
}

// BenchmarkPathTargetScan_Small benchmarks PathTarget.Scan
func BenchmarkPathTargetScan_Small(b *testing.B) {
	cat := types.Category{
		ID:    "bench-small",
		Paths: []string{benchDirSmall},
	}
	target := NewPathTarget(cat)

	// Warm up (ensure page cache is hot for this benchmark)
	_, _ = target.Scan()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = target.Scan()
	}
}

// BenchmarkPathTargetScan_Medium benchmarks PathTarget.Scan
func BenchmarkPathTargetScan_Medium(b *testing.B) {
	cat := types.Category{
		ID:    "bench-medium",
		Paths: []string{benchDirMedium},
	}
	target := NewPathTarget(cat)

	// Warm up
	_, _ = target.Scan()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = target.Scan()
	}
}

// BenchmarkPathTargetScan_Large benchmarks PathTarget.Scan
func BenchmarkPathTargetScan_Large(b *testing.B) {
	cat := types.Category{
		ID:    "bench-large",
		Paths: []string{benchDirLarge},
	}
	target := NewPathTarget(cat)

	// Warm up
	_, _ = target.Scan()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = target.Scan()
	}
}

// BenchmarkGetDirSizeWithCount_Small benchmarks GetDirSizeWithCount alone
func BenchmarkGetDirSizeWithCount_Small(b *testing.B) {
	// Warm up
	_, _, _ = utils.GetDirSizeWithCount(benchDirSmall)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = utils.GetDirSizeWithCount(benchDirSmall)
	}
}

func BenchmarkGetDirSizeWithCount_Medium(b *testing.B) {
	// Warm up
	_, _, _ = utils.GetDirSizeWithCount(benchDirMedium)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = utils.GetDirSizeWithCount(benchDirMedium)
	}
}

func BenchmarkGetDirSizeWithCount_Large(b *testing.B) {
	// Warm up
	_, _, _ = utils.GetDirSizeWithCount(benchDirLarge)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = utils.GetDirSizeWithCount(benchDirLarge)
	}
}

// BenchmarkScanPathsParallel_Small benchmarks scanPathsParallel alone
func BenchmarkScanPathsParallel_Small(b *testing.B) {
	cat := types.Category{ID: "bench-small", Paths: []string{benchDirSmall}}
	target := NewPathTarget(cat)
	paths := target.collectPaths()

	// Warm up
	_, _, _ = target.scanPathsParallel(paths)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = target.scanPathsParallel(paths)
	}
}

func BenchmarkScanPathsParallel_Medium(b *testing.B) {
	cat := types.Category{ID: "bench-medium", Paths: []string{benchDirMedium}}
	target := NewPathTarget(cat)
	paths := target.collectPaths()

	// Warm up
	_, _, _ = target.scanPathsParallel(paths)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = target.scanPathsParallel(paths)
	}
}

func BenchmarkScanPathsParallel_Large(b *testing.B) {
	cat := types.Category{ID: "bench-large", Paths: []string{benchDirLarge}}
	target := NewPathTarget(cat)
	paths := target.collectPaths()

	// Warm up
	_, _, _ = target.scanPathsParallel(paths)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = target.scanPathsParallel(paths)
	}
}
