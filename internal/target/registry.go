package target

import (
	"fmt"

	"github.com/2ykwang/mac-cleanup-go/internal/logger"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

// BuiltinFactory is a function that creates a Target from a category and optional categories list.
type BuiltinFactory func(cat types.Category, categories []types.Category) Target

var builtinFactories = map[string]BuiltinFactory{}

// RegisterBuiltin registers a builtin target factory.
func RegisterBuiltin(id string, factory BuiltinFactory) {
	builtinFactories[id] = factory
}

func IsBuiltinID(id string) bool {
	_, ok := builtinFactories[id]
	return ok
}

func DefaultRegistry(cfg *types.Config) (*Registry, error) {
	r := NewRegistry()

	builtinCount := 0
	pathCount := 0
	for _, cat := range cfg.Categories {
		var s Target
		if factory, ok := builtinFactories[cat.ID]; ok {
			// Use registered factory regardless of method type
			s = factory(cat, cfg.Categories)
			builtinCount++
		} else if cat.Method == types.MethodBuiltin {
			// method: builtin but no factory registered
			logger.Warn("unknown builtin target", "id", cat.ID)
			return nil, fmt.Errorf("unknown builtin target id: %s", cat.ID)
		} else {
			s = NewPathTarget(cat)
			pathCount++
		}
		r.Register(s)
	}

	logger.Info("registry initialized",
		"total", len(cfg.Categories),
		"builtin", builtinCount,
		"path", pathCount)

	return r, nil
}
