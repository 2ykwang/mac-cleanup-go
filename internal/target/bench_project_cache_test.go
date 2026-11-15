//go:build perf

package target

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

const (
	benchProjectCount             = 8
	benchProjectDirectories       = 16
	benchProjectFilesPerDirectory = 32
)

func newBenchmarkProjectCache(b *testing.B, activeEvery int) *ProjectCacheTarget {
	b.Helper()
	root := b.TempDir()
	old := time.Now().AddDate(0, 0, -30)
	for project := range benchProjectCount {
		parent := filepath.Join(root, fmt.Sprintf("project-%d", project))
		cache := filepath.Join(parent, "node_modules")
		if err := os.MkdirAll(cache, 0o755); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(parent, "package.json"), []byte("{}"), 0o644); err != nil {
			b.Fatal(err)
		}
		for directory := range benchProjectDirectories {
			dir := filepath.Join(cache, fmt.Sprintf("dep-%d", directory))
			if err := os.Mkdir(dir, 0o755); err != nil {
				b.Fatal(err)
			}
			for file := range benchProjectFilesPerDirectory {
				path := filepath.Join(dir, fmt.Sprintf("file-%d", file))
				if err := os.WriteFile(path, []byte("cache"), 0o644); err != nil {
					b.Fatal(err)
				}
				if err := os.Chtimes(path, old, old); err != nil {
					b.Fatal(err)
				}
			}
		}
		if activeEvery > 0 && project%activeEvery == 0 {
			if err := os.WriteFile(filepath.Join(cache, ".DS_Store"), []byte("active"), 0o644); err != nil {
				b.Fatal(err)
			}
		}
	}
	target := NewProjectCacheTarget(types.Category{ID: "project-cache"})
	target.scanRoot = root
	return target
}

func benchmarkProjectCache(b *testing.B, activeEvery, expectedItems int) {
	target := newBenchmarkProjectCache(b, activeEvery)
	result, err := target.Scan()
	if err != nil {
		b.Fatal(err)
	}
	if len(result.Items) != expectedItems {
		b.Fatalf("expected %d stale caches, got %d", expectedItems, len(result.Items))
	}
	expectedFiles := int64(expectedItems * benchProjectDirectories * benchProjectFilesPerDirectory)
	if result.TotalFileCount != expectedFiles || result.TotalSize != expectedFiles*int64(len("cache")) {
		b.Fatal("unexpected project cache totals")
	}
	b.ResetTimer()
	for b.Loop() {
		result, err := target.Scan()
		if err != nil {
			b.Fatal(err)
		}
		if len(result.Items) != expectedItems {
			b.Fatalf("expected %d stale caches, got %d", expectedItems, len(result.Items))
		}
		if result.TotalFileCount != expectedFiles || result.TotalSize != expectedFiles*int64(len("cache")) {
			b.Fatal("project cache totals changed")
		}
	}
}

func BenchmarkProjectCacheScanStale(b *testing.B)  { benchmarkProjectCache(b, 0, benchProjectCount) }
func BenchmarkProjectCacheScanMixed(b *testing.B)  { benchmarkProjectCache(b, 2, benchProjectCount/2) }
func BenchmarkProjectCacheScanActive(b *testing.B) { benchmarkProjectCache(b, 1, 0) }
