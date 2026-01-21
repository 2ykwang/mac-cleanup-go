package utils

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const lsofTimeout = 10 * time.Second

// GetLockedPaths returns top-level paths under basePath that are in use by processes.
func GetLockedPaths(basePath string) (map[string]bool, error) {
	locked := make(map[string]bool)
	if basePath == "" || !CommandExists("lsof") {
		return locked, nil
	}

	expanded := filepath.Clean(ExpandPath(basePath))
	ctx, cancel := context.WithTimeout(context.Background(), lsofTimeout)
	defer cancel()

	args := []string{"-nP", "-F", "n"}
	cmd := execCommandContext(ctx, "lsof", args...)
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		// lsof returns exit 1 when no results - not an error.
		exitErr := &exec.ExitError{}
		if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
			return nil, err
		}
	}
	if len(output) == 0 {
		return locked, nil
	}

	return parseLockedPaths(output, expanded), nil
}

func parseLockedPaths(output []byte, basePath string) map[string]bool {
	locked := make(map[string]bool)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if !strings.HasPrefix(line, "n") {
			continue
		}
		path := strings.TrimPrefix(line, "n")

		rel, err := filepath.Rel(basePath, path)
		if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
			continue
		}

		topLevel := strings.Split(rel, string(filepath.Separator))[0]
		if topLevel == "" || topLevel == "." || topLevel == ".." {
			continue
		}

		fullPath := filepath.Join(basePath, topLevel)
		locked[fullPath] = true
	}

	return locked
}
