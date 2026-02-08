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
	assert.Contains(t, err.Error(), "unknown builtin target id")
}

func TestDefaultRegistry_RegisteredFactory_UsedRegardlessOfMethod(t *testing.T) {
	// system-cache has method: trash but is registered in builtinFactories
	// It should use the registered factory, not PathTarget
	cfg := &types.Config{
		Categories: []types.Category{
			{
				ID:     "system-cache",
				Name:   "App Caches",
				Method: types.MethodTrash, // NOT builtin
				Safety: types.SafetyLevelModerate,
				Paths:  []string{"~/Library/Caches/**"},
			},
		},
	}

	registry, err := DefaultRegistry(cfg)

	require.NoError(t, err)
	require.NotNil(t, registry)

	target, ok := registry.Get("system-cache")
	require.True(t, ok)
	_, isSystemCache := target.(*SystemCacheTarget)
	assert.True(t, isSystemCache, "system-cache should use SystemCacheTarget even with method: trash")
}

func TestDefaultRegistry_NonBuiltin_UsesPathTarget(t *testing.T) {
	cfg := &types.Config{
		Categories: []types.Category{
			{
				ID:     "custom-category",
				Name:   "Custom",
				Method: types.MethodTrash,
				Safety: types.SafetyLevelSafe,
				Paths:  []string{"/tmp/test"},
			},
		},
	}

	registry, err := DefaultRegistry(cfg)

	require.NoError(t, err)
	require.NotNil(t, registry)

	target, ok := registry.Get("custom-category")
	require.True(t, ok)
	_, isPathTarget := target.(*PathTarget)
	assert.True(t, isPathTarget, "non-registered category should use PathTarget")
}

func TestIsBuiltinID_RegisteredBuiltin_ReturnsTrue(t *testing.T) {
	registerAllBuiltins()

	assert.True(t, IsBuiltinID("docker"))
	assert.True(t, IsBuiltinID("homebrew"))
	assert.True(t, IsBuiltinID("system-cache"))
	assert.True(t, IsBuiltinID("old-downloads"))
}

func TestIsBuiltinID_UnregisteredID_ReturnsFalse(t *testing.T) {
	assert.False(t, IsBuiltinID("unknown"))
	assert.False(t, IsBuiltinID("custom-category"))
	assert.False(t, IsBuiltinID(""))
}
