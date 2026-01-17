package target

import (
	"fmt"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

var builtinFactories = map[string]func(types.Category) Target{
	"docker":        func(cat types.Category) Target { return NewDockerTarget(cat) },
	"homebrew":      func(cat types.Category) Target { return NewBrewTarget(cat) },
	"old-downloads": func(cat types.Category) Target { return NewOldDownloadTarget(cat, defaultDaysOld) },
}

func IsBuiltinID(id string) bool {
	_, ok := builtinFactories[id]
	return ok
}

func DefaultRegistry(cfg *types.Config) (*Registry, error) {
	r := NewRegistry()

	for _, cat := range cfg.Categories {
		var s Target
		switch {
		case cat.ID == "system-cache":
			s = NewSystemCacheTarget(cat, cfg.Categories)
		case cat.Method == types.MethodBuiltin:
			if factory, ok := builtinFactories[cat.ID]; ok {
				s = factory(cat)
			} else {
				return nil, fmt.Errorf("unknown builtin scanner id: %s", cat.ID)
			}
		default:
			s = NewPathTarget(cat)
		}
		r.Register(s)
	}

	return r, nil
}
