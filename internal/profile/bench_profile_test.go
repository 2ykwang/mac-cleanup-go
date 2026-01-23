//go:build perf

package profile

import (
	"fmt"
	"os"
	"testing"

	"github.com/2ykwang/mac-cleanup-go/internal/benchutil"
	"github.com/2ykwang/mac-cleanup-go/internal/config"
	"github.com/2ykwang/mac-cleanup-go/internal/target"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

// Benchmark tree configuration
const (
	benchFilesPerDir = 100
	benchFanout      = 3
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
		"bench-",
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

func BenchmarkPathTargetScan(b *testing.B) {
	for _, bd := range benchDirs {
		b.Run(bd.Name, func(b *testing.B) {
			tgt := target.NewPathTarget(types.Category{ID: bd.Name, Paths: []string{bd.Dir}})
			_, _ = tgt.Scan()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = tgt.Scan()
			}
		})
	}
}

func BenchmarkGetDirSizeWithCount(b *testing.B) {
	for _, bd := range benchDirs {
		b.Run(bd.Name, func(b *testing.B) {
			_, _, _ = utils.GetDirSizeWithCount(bd.Dir)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, _ = utils.GetDirSizeWithCount(bd.Dir)
			}
		})
	}
}

func BenchmarkRegistryAvailable(b *testing.B) {
	cfg, err := config.LoadEmbedded()
	if err != nil {
		b.Fatal(err)
	}
	registry, err := target.DefaultRegistry(cfg)
	if err != nil {
		b.Fatal(err)
	}

	_ = registry.Available()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.Available()
	}
}

func BenchmarkFullScanFlow(b *testing.B) {
	cfg, err := config.LoadEmbedded()
	if err != nil {
		b.Fatal(err)
	}
	registry, err := target.DefaultRegistry(cfg)
	if err != nil {
		b.Fatal(err)
	}

	available := registry.Available()
	for _, t := range available {
		_, _ = t.Scan()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		available := registry.Available()
		for _, t := range available {
			_, _ = t.Scan()
		}
	}
}
