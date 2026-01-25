//go:build perf

package target

import (
	"fmt"
	"testing"

	"github.com/2ykwang/mac-cleanup-go/internal/benchfixtures"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func BenchmarkScanPathTarget(b *testing.B) {
	for _, bd := range benchDirs {
		b.Run(bd.Name, func(b *testing.B) {
			tgt := NewPathTarget(types.Category{ID: bd.Name, Paths: []string{bd.Dir}})
			_, _ = tgt.Scan()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = tgt.Scan()
			}
		})
	}
}

func BenchmarkAvailableTargets(b *testing.B) {
	cfg := benchConfigFromDirs(benchDirs)
	registry, err := DefaultRegistry(cfg)
	if err != nil {
		b.Fatal(err)
	}

	_ = registry.Available()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.Available()
	}
}

func BenchmarkScanAllTargets(b *testing.B) {
	cfg := benchConfigFromDirs(benchDirs)
	registry, err := DefaultRegistry(cfg)
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

func benchConfigFromDirs(dirs []benchfixtures.BenchDir) *types.Config {
	categories := make([]types.Category, 0, len(dirs))
	for _, dir := range dirs {
		categories = append(categories, types.Category{
			ID:     fmt.Sprintf("bench-%s", dir.Name),
			Name:   fmt.Sprintf("Bench %s", dir.Name),
			Method: types.MethodTrash,
			Safety: types.SafetyLevelSafe,
			Paths:  []string{dir.Dir},
		})
	}
	return &types.Config{Categories: categories}
}
