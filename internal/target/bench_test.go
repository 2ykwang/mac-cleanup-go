//go:build perf

package target

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

// Benchmark tree configuration
const (
	benchFilesPerDir = 100
	benchFanout      = 3
)

// Shared benchmark directories (initialized in TestMain)
var benchDirs = []struct {
	name  string
	dir   string
	depth int
}{
	{"Small", "", 3},
	{"Medium", "", 5},
	{"Large", "", 8},
}

func TestMain(m *testing.M) {
	root := os.Getenv("BENCH_DATA_DIR")
	shouldCleanup := root == ""
	if shouldCleanup {
		root, _ = os.MkdirTemp("", "bench-")
	}

	for i := range benchDirs {
		benchDirs[i].dir = filepath.Join(root, benchDirs[i].name)
		if _, err := os.Stat(benchDirs[i].dir); err != nil {
			createTree(benchDirs[i].dir, benchDirs[i].depth)
		}
	}

	code := m.Run()

	if shouldCleanup {
		_ = os.RemoveAll(root)
	}
	os.Exit(code)
}

// createTree creates a directory tree with the given depth.
func createTree(root string, depth int) {
	data := make([]byte, 1024)
	var create func(path string, d int)
	create = func(path string, d int) {
		_ = os.MkdirAll(path, 0o755)
		for i := 0; i < benchFilesPerDir; i++ {
			_ = os.WriteFile(filepath.Join(path, string(rune('a'+i))+".txt"), data, 0o644)
		}
		if d < depth {
			for i := 0; i < benchFanout; i++ {
				create(filepath.Join(path, "d"+string(rune('a'+i))), d+1)
			}
		}
	}
	create(root, 1)
}

func BenchmarkPathTargetScan(b *testing.B) {
	for _, bd := range benchDirs {
		b.Run(bd.name, func(b *testing.B) {
			target := NewPathTarget(types.Category{ID: bd.name, Paths: []string{bd.dir}})
			_, _ = target.Scan()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = target.Scan()
			}
		})
	}
}

func BenchmarkGetDirSizeWithCount(b *testing.B) {
	for _, bd := range benchDirs {
		b.Run(bd.name, func(b *testing.B) {
			_, _, _ = utils.GetDirSizeWithCount(bd.dir)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, _ = utils.GetDirSizeWithCount(bd.dir)
			}
		})
	}
}

func BenchmarkScanPathsParallel(b *testing.B) {
	for _, bd := range benchDirs {
		b.Run(bd.name, func(b *testing.B) {
			target := NewPathTarget(types.Category{ID: bd.name, Paths: []string{bd.dir}})
			paths := target.collectPaths()
			_, _, _ = target.scanPathsParallel(paths)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, _ = target.scanPathsParallel(paths)
			}
		})
	}
}
