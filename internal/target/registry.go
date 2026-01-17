package target

import (
	"fmt"

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

	for _, cat := range cfg.Categories {
		var s Target
		if factory, ok := builtinFactories[cat.ID]; ok {
			// Use registered factory regardless of method type
			s = factory(cat, cfg.Categories)
		} else if cat.Method == types.MethodBuiltin {
			// method: builtin but no factory registered
			return nil, fmt.Errorf("unknown builtin target id: %s", cat.ID)
		} else {
			s = NewPathTarget(cat)
		}
		r.Register(s)
	}

	return r, nil
}
