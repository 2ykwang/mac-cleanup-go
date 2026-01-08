package scanner

import "github.com/2ykwang/mac-cleanup-go/internal/types"

type Scanner interface {
	Scan() (*types.ScanResult, error)
	Clean(items []types.CleanableItem) (*types.CleanResult, error)
	Category() types.Category
	IsAvailable() bool
}

type Registry struct {
	scanners map[string]Scanner
}

func NewRegistry() *Registry {
	return &Registry{scanners: make(map[string]Scanner)}
}

func (r *Registry) Register(s Scanner) {
	r.scanners[s.Category().ID] = s
}

func (r *Registry) Get(id string) (Scanner, bool) {
	s, ok := r.scanners[id]
	return s, ok
}

func (r *Registry) All() []Scanner {
	result := make([]Scanner, 0, len(r.scanners))
	for _, s := range r.scanners {
		result = append(result, s)
	}
	return result
}

func (r *Registry) Available() []Scanner {
	result := make([]Scanner, 0)
	for _, s := range r.scanners {
		if s.IsAvailable() {
			result = append(result, s)
		}
	}
	return result
}
