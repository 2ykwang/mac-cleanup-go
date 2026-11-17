package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/config"
)

func newTestDockerSession(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	logPath := filepath.Join(dir, "docker-calls")
	t.Setenv("DOCKER_TEST_CALLS", logPath)
	script := `#!/bin/sh
case "$*" in
  version) exit 0 ;;
  'system df -v --format {{json .}}')
    echo '{"Images":[{"ID":"sha256:unused","Repository":"postgres","Tag":"latest","Size":"1GB"},{"ID":"sha256:used","Repository":"redis","Tag":"latest","Size":"1GB"}],"Containers":[{"Names":"running-db","Image":"redis:latest","Mounts":"locked-data"}],"Volumes":[{"Name":"postgres-data","Size":"2GB"},{"Name":"other-data","Size":"1GB"},{"Name":"locked-data","Size":"1GB"}]}' ;;
  'system df --format {{json .}}')
    echo '{"Type":"Build Cache","Reclaimable":"1GB"}' ;;
  'image rm '*|'volume rm '*|'builder prune -af')
    printf '%s\n' "$*" >> "$DOCKER_TEST_CALLS" ;;
  *) exit 1 ;;
esac
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0o755))
	require.NoError(t, os.WriteFile(logPath, nil, 0o600))
	cfg, err := config.LoadEmbedded()
	require.NoError(t, err)
	t.Cleanup(stubSession(cfg, emptyUserConfig()))
	return logPath
}

func newTestDockerCalls(t *testing.T, logPath string) []string {
	t.Helper()
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	if len(data) == 0 {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

func TestRunClean_Docker_NeverDeletesUnusedNamedDatabaseVolume(t *testing.T) {
	logPath := newTestDockerSession(t)

	out := runCleanJSON(t, ExitOK, "docker", "--yes")

	assert.Equal(t, 2, out.CleanedItems)
	assert.Equal(t, []string{"image rm sha256:unused", "builder prune -af"}, newTestDockerCalls(t, logPath))
}

func TestRunClean_DockerVolumes_RequiresExplicitItem(t *testing.T) {
	logPath := newTestDockerSession(t)

	out := runCleanJSON(t, ExitOK, "docker-volumes", "--yes")

	assert.Empty(t, out.Targets)
	assert.Equal(t, skipRiskyNeedItem, skipReasons(out)["docker-volumes"])
	assert.Empty(t, newTestDockerCalls(t, logPath))
}

func TestRunClean_DockerVolumes_OnlyDeletesApprovedVolume(t *testing.T) {
	logPath := newTestDockerSession(t)

	out := runCleanJSON(t, ExitOK, "docker-volumes", "--item", "docker:volume:postgres-data", "--yes")

	assert.Equal(t, 1, out.CleanedItems)
	assert.Equal(t, []string{"volume rm postgres-data"}, newTestDockerCalls(t, logPath))
}
