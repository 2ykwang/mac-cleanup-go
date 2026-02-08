package target

import "github.com/2ykwang/mac-cleanup-go/internal/types"

type Target interface {
	Scan() (*types.ScanResult, error)
	Category() types.Category
	IsAvailable() bool
}

// BuiltinCleaner is implemented by targets that have their own cleanup logic (e.g. brew, docker).
type BuiltinCleaner interface {
	Clean(items []types.CleanableItem) (*types.CleanResult, error)
}

type Registry struct {
	targets map[string]Target
}

func NewRegistry() *Registry {
	return &Registry{targets: make(map[string]Target)}
}

func (r *Registry) Register(s Target) {
	r.targets[s.Category().ID] = s
}

func (r *Registry) Get(id string) (Target, bool) {
	s, ok := r.targets[id]
	return s, ok
}

func (r *Registry) All() []Target {
	result := make([]Target, 0, len(r.targets))
	for _, s := range r.targets {
		result = append(result, s)
	}
	return result
}

func (r *Registry) Available() []Target {
	result := make([]Target, 0)
	for _, s := range r.targets {
		if s.IsAvailable() {
			result = append(result, s)
		}
	}
	return result
}
