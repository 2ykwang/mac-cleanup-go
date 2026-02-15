package target

import (
	"fmt"

	"github.com/2ykwang/mac-cleanup-go/internal/logger"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

// BuiltinFactory is a function that creates a Target from a category and optional categories list.
type BuiltinFactory func(cat types.Category, categories []types.Category) Target

// builtinIDs is the static set of known builtin target IDs.
// Used for config validation independent of factory registration.
var builtinIDs = map[string]struct{}{
	"homebrew":      {},
	"docker":        {},
	"old-downloads": {},
	"system-cache":  {},
	"project-cache": {},
}

var builtinFactories = map[string]BuiltinFactory{}

// RegisterBuiltin registers a builtin target factory.
func RegisterBuiltin(id string, factory BuiltinFactory) {
	builtinFactories[id] = factory
}

func IsBuiltinID(id string) bool {
	_, ok := builtinIDs[id]
	return ok
}

func registerAllBuiltins() {
	builtinFactories = map[string]BuiltinFactory{}

	RegisterBuiltin("homebrew", func(cat types.Category, _ []types.Category) Target {
		return NewBrewTarget(cat)
	})
	RegisterBuiltin("docker", func(cat types.Category, _ []types.Category) Target {
		return NewDockerTarget(cat)
	})
	RegisterBuiltin("old-downloads", func(cat types.Category, _ []types.Category) Target {
		return NewOldDownloadTarget(cat, defaultDaysOld)
	})
	RegisterBuiltin("system-cache", func(cat types.Category, categories []types.Category) Target {
		return NewSystemCacheTarget(cat, categories)
	})
	RegisterBuiltin("project-cache", func(cat types.Category, _ []types.Category) Target {
		return NewProjectCacheTarget(cat)
	})
}

func DefaultRegistry(cfg *types.Config) (*Registry, error) {
	registerAllBuiltins()
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
