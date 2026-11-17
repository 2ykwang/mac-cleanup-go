package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func TestLoadEmbedded_DeclaresBuiltinDisposal(t *testing.T) {
	cfg, err := LoadEmbedded()
	require.NoError(t, err)
	byID := make(map[string]types.DisposalMode)
	for _, cat := range cfg.Categories {
		byID[cat.ID] = cat.DisposalMode()
	}

	assert.Equal(t, types.DisposalPermanent, byID["docker"])
	assert.Equal(t, types.DisposalMixed, byID["homebrew"])
	assert.Equal(t, types.DisposalTrash, byID["project-cache"])
	assert.Equal(t, types.DisposalTrash, byID["old-downloads"])
	assert.Equal(t, types.DisposalTrash, byID["claude-code-versions"])
	assert.Equal(t, types.DisposalTrash, byID["codex-daemon-releases"])
}

func TestValidateDisposal_RejectsUnknownValue(t *testing.T) {
	err := validateDisposal(types.Category{ID: "docker", Method: types.MethodBuiltin, Disposal: "typo"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid disposal")
}

func TestValidateDisposal_RejectsContradictoryMethod(t *testing.T) {
	err := validateDisposal(types.Category{ID: "trash", Method: types.MethodPermanent, Disposal: types.DisposalTrash})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicts with method")
}

func TestValidateDisposal_RejectsManualBuiltin(t *testing.T) {
	err := validateDisposal(types.Category{Method: types.MethodBuiltin, Disposal: types.DisposalManual})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be manual")
}

func TestValidateDisposal_AllowsUnspecifiedBuiltin(t *testing.T) {
	cat := types.Category{Method: types.MethodBuiltin}

	err := validateDisposal(cat)

	assert.NoError(t, err)
	assert.Equal(t, types.DisposalUnknown, cat.DisposalMode())
}
