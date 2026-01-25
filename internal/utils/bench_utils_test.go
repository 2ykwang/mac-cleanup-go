//go:build perf

package utils

import (
	"fmt"
	"os"
	"testing"

	"github.com/2ykwang/mac-cleanup-go/internal/benchfixtures"
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
