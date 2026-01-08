package scanner

import "github.com/2ykwang/mac-cleanup-go/internal/types"

func DefaultRegistry(cfg *types.Config) *Registry {
	r := NewRegistry()

	for _, cat := range cfg.Categories {
		var s Scanner
		switch {
		case cat.Method == types.MethodBuiltin && cat.ID == "docker":
			s = NewDockerScanner(cat)
		case cat.Method == types.MethodBuiltin && cat.ID == "homebrew":
			s = NewBrewScanner(cat)
		case cat.Method == types.MethodBuiltin && cat.ID == "old-downloads":
			s = NewOldDownloadScanner(cat, defaultDaysOld)
		case cat.ID == "system-cache":
			s = NewSystemCacheScanner(cat, cfg.Categories)
		default:
			s = NewPathScanner(cat)
		}
		r.Register(s)
	}

	return r
}
