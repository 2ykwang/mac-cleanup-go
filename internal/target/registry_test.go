package target

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func TestDefaultRegistry_UnknownBuiltin_ReturnsError(t *testing.T) {
	cfg := &types.Config{
		Categories: []types.Category{
			{ID: "unknown", Name: "Unknown", Method: types.MethodBuiltin, Safety: types.SafetyLevelSafe},
		},
	}

	registry, err := DefaultRegistry(cfg)

	require.Error(t, err)
	assert.Nil(t, registry)
}
