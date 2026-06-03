package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/2ykwang/mac-cleanup-go/internal/target"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/utils"
)

func newTestModelWithRegistry(r *target.Registry) *Model {
	return &Model{configState: configState{registry: r}}
}

func newTestDockerRegistry() *target.Registry {
	r := target.NewRegistry()
	r.Register(target.NewDockerTarget(types.Category{ID: "docker", Name: "Docker"}))
	return r
}

func TestDetectDockerUnreachable_True_WhenInstalledButNotAvailable(t *testing.T) {
	original := utils.CommandExists
	defer func() { utils.CommandExists = original }()
	utils.CommandExists = func(_ string) bool { return true }

	m := newTestModelWithRegistry(newTestDockerRegistry())
	available := []target.Target{target.NewPathTarget(types.Category{ID: "system-cache"})}

	result := m.detectDockerUnreachable(available)

	assert.True(t, result)
}

func TestDetectDockerUnreachable_False_WhenDockerAvailable(t *testing.T) {
	original := utils.CommandExists
	defer func() { utils.CommandExists = original }()
	utils.CommandExists = func(_ string) bool { return true }

	m := newTestModelWithRegistry(newTestDockerRegistry())
	available := []target.Target{target.NewDockerTarget(types.Category{ID: "docker"})}

	result := m.detectDockerUnreachable(available)

	assert.False(t, result)
}

func TestDetectDockerUnreachable_False_WhenDockerNotInstalled(t *testing.T) {
	original := utils.CommandExists
	defer func() { utils.CommandExists = original }()
	utils.CommandExists = func(_ string) bool { return false }

	m := newTestModelWithRegistry(newTestDockerRegistry())

	result := m.detectDockerUnreachable([]target.Target{})

	assert.False(t, result)
}

func TestDetectDockerUnreachable_False_WhenDockerNotConfigured(t *testing.T) {
	original := utils.CommandExists
	defer func() { utils.CommandExists = original }()
	utils.CommandExists = func(_ string) bool { return true }

	m := newTestModelWithRegistry(target.NewRegistry())

	result := m.detectDockerUnreachable([]target.Target{})

	assert.False(t, result)
}
