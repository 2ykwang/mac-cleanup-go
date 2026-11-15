//go:build perf

package target

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func BenchmarkScanTargetsConcurrent(b *testing.B) {
	cfg := benchConfigFromDirs(benchDirs)
	for i := range cfg.Categories {
		cfg.Categories[i].Paths = []string{filepath.Join(benchDirs[i].Dir, "*")}
	}
	registry, err := DefaultRegistry(cfg)
	if err != nil {
		b.Fatal(err)
	}
	scanners := registry.Available()
	if len(scanners) != len(benchDirs) {
		b.Fatalf("expected %d scanners, got %d", len(benchDirs), len(scanners))
	}
	results := make([]*types.ScanResult, len(scanners))
	errors := make([]error, len(scanners))
	scan := func() {
		var wg sync.WaitGroup
		wg.Add(len(scanners))
		for i, scanner := range scanners {
			go func() { defer wg.Done(); results[i], errors[i] = scanner.Scan() }()
		}
		wg.Wait()
		for i, result := range results {
			if errors[i] != nil {
				b.Fatal(errors[i])
			}
			if result == nil {
				b.Fatal("scan returned no result")
			}
		}
	}
	scan()
	counts := make([]int64, len(results))
	sizes := make([]int64, len(results))
	for i, result := range results {
		counts[i] = result.TotalFileCount
		sizes[i] = result.TotalSize
	}
	b.ResetTimer()
	for b.Loop() {
		scan()
		for i, result := range results {
			if result.TotalFileCount != counts[i] || result.TotalSize != sizes[i] {
				b.Fatalf("scan totals changed for %s", result.Category.ID)
			}
		}
	}
}
