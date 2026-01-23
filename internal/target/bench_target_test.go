//go:build perf

package target

import (
	"fmt"
	"os"
	"testing"

	"github.com/2ykwang/mac-cleanup-go/internal/benchutil"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

// Benchmark tree configuration
const (
	benchInternalFilesPerDir = 100
	benchInternalFanout      = 3
)

var benchSpecs = []benchutil.BenchDirSpec{
	{Name: "Small", Depth: 3},
	{Name: "Medium", Depth: 5},
	{Name: "Large", Depth: 8},
}

// Shared benchmark directories (initialized in TestMain).
var benchDirs []benchutil.BenchDir

func TestMain(m *testing.M) {
	dirs, cleanup, err := benchutil.PrepareBenchDirs(
		"BENCH_DATA_DIR",
		"bench-internal-",
		benchSpecs,
		benchInternalFilesPerDir,
		benchInternalFanout,
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

func BenchmarkScanPathsParallel(b *testing.B) {
	for _, bd := range benchDirs {
		b.Run(bd.Name, func(b *testing.B) {
			target := NewPathTarget(types.Category{ID: bd.Name, Paths: []string{bd.Dir}})
			paths := target.collectPaths()
			_, _, _ = target.scanPathsParallel(paths)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, _ = target.scanPathsParallel(paths)
			}
		})
	}
}
