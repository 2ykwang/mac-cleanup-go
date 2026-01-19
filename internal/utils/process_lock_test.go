package utils

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseLockedPaths_ExtractsTopLevel(t *testing.T) {
	basePath := "/Users/test/Library/Caches"
	output := strings.Join([]string{
		"p1234",
		"n/Users/test/Library/Caches/com.apple.Safari/Cache.db",
		"n/Users/test/Library/Caches/com.apple.Safari/Cache2",
		"n/Users/test/Library/Caches/Arc/Cache",
		"n/Users/test/Library/Caches",
		"n/Users/test/Other/skip",
		"",
	}, "\n")

	locked := parseLockedPaths([]byte(output), basePath)

	require.Len(t, locked, 2)
	require.True(t, locked[filepath.Join(basePath, "com.apple.Safari")])
	require.True(t, locked[filepath.Join(basePath, "Arc")])
}

func TestGetLockedPaths_NoLsof_ReturnsEmpty(t *testing.T) {
	originalLookPath := execLookPath
	execLookPath = func(_ string) (string, error) {
		return "", errors.New("not found")
	}
	defer func() { execLookPath = originalLookPath }()

	locked, err := GetLockedPaths("/tmp")

	require.NoError(t, err)
	require.Empty(t, locked)
}

func TestGetLockedPaths_ExitCodeOne_ReturnsEmpty(t *testing.T) {
	originalLookPath := execLookPath
	execLookPath = func(_ string) (string, error) {
		return "/usr/bin/lsof", nil
	}
	defer func() { execLookPath = originalLookPath }()

	originalCmd := execCommandContext
	execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "exit 1")
	}
	defer func() { execCommandContext = originalCmd }()

	locked, err := GetLockedPaths("/tmp")

	require.NoError(t, err)
	require.Empty(t, locked)
}

func TestGetLockedPaths_ExitCodeOne_ParsesOutput(t *testing.T) {
	originalLookPath := execLookPath
	execLookPath = func(_ string) (string, error) {
		return "/usr/bin/lsof", nil
	}
	defer func() { execLookPath = originalLookPath }()

	originalCmd := execCommandContext
	execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "printf 'n/tmp/base/app/file\n'; exit 1")
	}
	defer func() { execCommandContext = originalCmd }()

	locked, err := GetLockedPaths("/tmp/base")

	require.NoError(t, err)
	require.True(t, locked["/tmp/base/app"])
}
