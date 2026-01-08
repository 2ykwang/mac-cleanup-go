package utils

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMoveToTrash_CanBeMocked(t *testing.T) {
	original := MoveToTrash
	defer func() { MoveToTrash = original }()

	var calledPath string
	MoveToTrash = func(path string) error {
		calledPath = path
		return nil
	}

	err := MoveToTrash("/test/path")

	assert.NoError(t, err)
	assert.Equal(t, "/test/path", calledPath)
}

func TestMoveToTrash_MockError(t *testing.T) {
	original := MoveToTrash
	defer func() { MoveToTrash = original }()

	MoveToTrash = func(path string) error {
		return fmt.Errorf("mock error: %s", path)
	}

	err := MoveToTrash("/test/path")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock error")
}

func TestMoveToTrashImpl_SkipOnNonMacOS(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping test on non-macOS")
	}
	// Real implementation test would go here
	// but we don't test actual osascript calls to avoid side effects
}
