package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

func newTestHomebrewSession(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFiles(t, root, map[string]int{"package": 10})
	bin := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(bin, "brew"),
		[]byte("#!/bin/sh\n[ \"$#\" = 1 ] && [ \"$1\" = --cache ] || exit 1\n/bin/echo \"$HOMEBREW_CACHE\"\n"), 0o755))
	t.Setenv("PATH", bin)
	t.Setenv("HOMEBREW_CACHE", root)
	t.Cleanup(stubSession(testConfig(testCategory("homebrew", types.SafetyLevelSafe, types.MethodBuiltin)), emptyUserConfig()))
	return root
}

func TestRunClean_HomebrewAcceptsScannedCacheItem(t *testing.T) {
	root := newTestHomebrewSession(t)

	out := runCleanJSON(t, ExitOK, "homebrew", "--item", root, "--dry-run")

	require.Len(t, out.Targets, 1)
	assert.Equal(t, []string{root}, out.Targets[0].ItemPaths)
	assert.Equal(t, int64(10), out.FreedBytes)
	assert.FileExists(t, filepath.Join(root, "package"))
}
