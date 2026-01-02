package scanner

import "mac-cleanup-go/pkg/types"

func DefaultRegistry(cfg *types.Config) *Registry {
	r := NewRegistry()

	for _, cat := range cfg.Categories {
		var s Scanner
		switch {
		case cat.Method == types.MethodSpecial && cat.ID == "docker":
			s = NewDockerScanner(cat)
		case cat.Method == types.MethodSpecial && cat.ID == "homebrew":
			s = NewBrewScanner(cat)
		case cat.ID == "system-cache":
			s = NewSystemCacheScanner(cat, cfg.Categories)
		default:
			s = NewPathScanner(cat)
		}
		r.Register(s)
	}

	return r
}
