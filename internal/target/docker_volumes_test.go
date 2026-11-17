package target

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func TestDockerVolumesTarget_Scan_NamedAndAnonymousVolumesOnly(t *testing.T) {
	anonymous := strings.Repeat("a", 64)
	verbose := `{"Images":[{"ID":"sha256:img1","Size":"1GB"}],"Containers":[{"Names":"db","Mounts":"postgres-data"}],"Volumes":[{"Name":"postgres-data","Size":"2GB"},{"Name":"` + anonymous + `","Size":"1GB"}]}`
	defer stubDockerScan(t, verbose, `{"Type":"Build Cache","Reclaimable":"1GB"}`)()
	original := execCommand
	var calls [][]string
	execCommand = func(name string, args ...string) *exec.Cmd {
		calls = append(calls, append([]string{name}, args...))
		return original(name, args...)
	}
	defer func() { execCommand = original }()
	s := NewDockerVolumesTarget(types.Category{ID: "docker-volumes"})

	result, err := s.Scan()

	require.NoError(t, err)
	require.NoError(t, result.Error)
	require.Len(t, result.Items, 2)
	assert.Equal(t, dockerPathPrefixVolume+"postgres-data", result.Items[0].Path)
	assert.Equal(t, types.ItemStatusProcessLocked, result.Items[0].Status)
	assert.Contains(t, result.Items[0].DisplayName, "Used By: db")
	assert.Equal(t, dockerPathPrefixVolume+anonymous, result.Items[1].Path)
	assert.Equal(t, types.ItemStatusAvailable, result.Items[1].Status)
	assert.Equal(t, parseDockerSize("3GB"), result.TotalSize)
	assert.Equal(t, [][]string{{"docker", "system", "df", "-v", "--format", "{{json .}}"}}, calls)
}

func TestDockerVolumesTarget_Clean_OnlyExplicitVolume(t *testing.T) {
	original := execCommand
	defer func() { execCommand = original }()
	var calls [][]string
	execCommand = func(name string, args ...string) *exec.Cmd {
		calls = append(calls, append([]string{name}, args...))
		return exec.Command("true")
	}
	s := NewDockerVolumesTarget(types.Category{ID: "docker-volumes"})

	result, err := s.Clean([]types.CleanableItem{{Path: dockerPathPrefixVolume + "postgres-data", Size: 123}})

	require.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Equal(t, [][]string{{"docker", "volume", "rm", "postgres-data"}}, calls)
	assert.Equal(t, int64(123), result.FreedSpace)
	assert.Equal(t, 1, result.CleanedItems)
}

func TestDockerVolumesTarget_Clean_RejectsImagesAndBuildCache(t *testing.T) {
	original := execCommand
	defer func() { execCommand = original }()
	var calls [][]string
	execCommand = func(name string, args ...string) *exec.Cmd {
		calls = append(calls, append([]string{name}, args...))
		return exec.Command("true")
	}
	s := NewDockerVolumesTarget(types.Category{ID: "docker-volumes"})

	result, err := s.Clean([]types.CleanableItem{
		{Path: dockerPathPrefixImage + "sha256:abc", Size: 100},
		{Path: dockerPathBuildCache, Size: 100},
		{Path: dockerPathPrefixVolume, Size: 100},
	})

	require.NoError(t, err)
	assert.Empty(t, calls)
	require.Len(t, result.Errors, 3)
	assert.Contains(t, result.Errors[0], "docker:image:sha256:abc: unsupported item for docker-volumes")
	assert.Contains(t, result.Errors[1], "docker:build-cache: unsupported item for docker-volumes")
	assert.Zero(t, result.CleanedItems)
	assert.Zero(t, result.FreedSpace)
}
