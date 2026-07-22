//go:build perf

package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/2ykwang/mac-cleanup-go/internal/benchfixtures"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

// Benchmark tree configuration.
const (
	benchFilesPerDir = 100
	benchFanout      = 3
)

var benchSpecs = []benchfixtures.BenchDirSpec{
	{Name: "Small", Depth: 3},
	{Name: "Medium", Depth: 5},
	{Name: "Large", Depth: 8},
}

// Shared benchmark directories (initialized in TestMain).
var benchDirs []benchfixtures.BenchDir

func TestMain(m *testing.M) {
	dirs, cleanup, err := benchfixtures.PrepareBenchDirs(
		"BENCH_DATA_DIR",
		"bench-utils-",
		benchSpecs,
		benchFilesPerDir,
		benchFanout,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	benchDirs = dirs
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func BenchmarkDirSizeWithCount(b *testing.B) {
	for _, bd := range benchDirs {
		b.Run(bd.Name, func(b *testing.B) {
			_, _, _ = GetDirSizeWithCount(bd.Dir)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, _ = GetDirSizeWithCount(bd.Dir)
			}
		})
	}
}

func BenchmarkBatchTrash(b *testing.B) {
	const pathCount = 500
	paths := make([]string, 0, pathCount)
	err := filepath.WalkDir(benchDirs[1].Dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		paths = append(paths, path)
		if len(paths) == pathCount {
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		b.Fatal(err)
	}
	if len(paths) != pathCount {
		b.Fatalf("expected %d benchmark paths, got %d", pathCount, len(paths))
	}
	items := make([]types.CleanableItem, len(paths))
	for i, path := range paths {
		items[i] = types.CleanableItem{Path: path, Size: int64(i + 1)}
	}
	recycle := func(paths []string) ([]string, error) {
		return paths, nil
	}

	b.ResetTimer()
	for b.Loop() {
		result := batchTrash(items, types.BatchTrashOptions{}, recycle)
		if len(result.Errors) != 0 {
			b.Fatal(result.Errors)
		}
	}
}
