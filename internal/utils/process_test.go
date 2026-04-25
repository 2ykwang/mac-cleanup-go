package utils

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsProcessRunning_ReturnsTrue_WhenPgrepFindsProcess(t *testing.T) {
	original := execCommand
	defer func() { execCommand = original }()
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("printf", "12345\n")
	}

	assert.True(t, IsProcessRunning("Xcode"))
}

func TestIsProcessRunning_ReturnsFalse_WhenPgrepEmpty(t *testing.T) {
	original := execCommand
	defer func() { execCommand = original }()
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	assert.False(t, IsProcessRunning("Xcode"))
}

func TestIsProcessRunning_ReturnsFalse_WhenPgrepErrors(t *testing.T) {
	original := execCommand
	defer func() { execCommand = original }()
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}

	assert.False(t, IsProcessRunning("Xcode"))
}

func TestIsProcessRunning_ReturnsFalse_WhenNameEmpty(t *testing.T) {
	assert.False(t, IsProcessRunning(""))
}
