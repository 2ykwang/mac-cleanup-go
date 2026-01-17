package version

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBrewVersion_ValidOutput(t *testing.T) {
	output := `==> 2ykwang/2ykwang/mac-cleanup-go: stable 1.3.1
Interactive TUI for cleaning macOS caches, logs, and temporary files
https://github.com/2ykwang/mac-cleanup-go
Installed
/opt/homebrew/Cellar/mac-cleanup-go/1.3.1 (6 files, 4.1MB) *`

	got := parseBrewVersion(output)
	assert.Equal(t, "1.3.1", got)
}

func TestParseBrewVersion_InvalidOutput(t *testing.T) {
	output := "some invalid output"
	got := parseBrewVersion(output)
	assert.Equal(t, "", got)
}

func TestParseBrewVersion_EmptyOutput(t *testing.T) {
	got := parseBrewVersion("")
	assert.Equal(t, "", got)
}

func TestIsNewerVersion_NewerMajor(t *testing.T) {
	assert.True(t, isNewerVersion("2.0.0", "1.9.9"))
}

func TestIsNewerVersion_NewerMinor(t *testing.T) {
	assert.True(t, isNewerVersion("1.2.0", "1.1.9"))
}

func TestIsNewerVersion_NewerPatch(t *testing.T) {
	assert.True(t, isNewerVersion("1.1.2", "1.1.1"))
}

func TestIsNewerVersion_SameVersion(t *testing.T) {
	assert.False(t, isNewerVersion("1.0.0", "1.0.0"))
}

func TestIsNewerVersion_OlderVersion(t *testing.T) {
	assert.False(t, isNewerVersion("1.0.0", "1.1.0"))
}

func TestIsNewerVersion_WithVPrefix(t *testing.T) {
	assert.True(t, isNewerVersion("v1.2.0", "v1.1.0"))
	assert.True(t, isNewerVersion("1.2.0", "v1.1.0"))
	assert.True(t, isNewerVersion("v1.2.0", "1.1.0"))
}

func TestCheckForUpdate_DevVersion(t *testing.T) {
	result := CheckForUpdate("dev")
	// dev 버전도 체크함 - brew 있으면 latestVersion이 채워짐
	assert.Equal(t, "dev", result.CurrentVersion)
	// dev는 숫자가 아니라서 버전 비교 시 updateAvailable=true가 됨
	// (brew가 없는 환경에서는 false)
}

func TestCheckForUpdate_EmptyVersion(t *testing.T) {
	result := CheckForUpdate("")
	assert.False(t, result.UpdateAvailable)
}

func TestCheckForUpdate_BrewNotFound(t *testing.T) {
	original := execLookPath
	defer func() { execLookPath = original }()
	execLookPath = func(file string) (string, error) {
		return "", errors.New("executable not found")
	}

	result := CheckForUpdate("1.0.0")

	assert.Equal(t, "1.0.0", result.CurrentVersion)
	assert.False(t, result.UpdateAvailable)
	assert.Empty(t, result.LatestVersion)
}

func TestCheckForUpdate_BrewCommandSuccess(t *testing.T) {
	originalLookPath := execLookPath
	originalCommand := execCommand
	defer func() {
		execLookPath = originalLookPath
		execCommand = originalCommand
	}()

	execLookPath = func(file string) (string, error) {
		return "/opt/homebrew/bin/brew", nil
	}
	execCommand = func(name string, args ...string) *exec.Cmd {
		// Return a command that outputs valid brew info
		return exec.Command("echo", "mac-cleanup-go: stable 2.0.0")
	}

	result := CheckForUpdate("1.0.0")

	assert.Equal(t, "1.0.0", result.CurrentVersion)
	assert.Equal(t, "2.0.0", result.LatestVersion)
	assert.True(t, result.UpdateAvailable)
	assert.NoError(t, result.Error)
}

func TestCheckForUpdate_BrewCommandError(t *testing.T) {
	originalLookPath := execLookPath
	originalCommand := execCommand
	defer func() {
		execLookPath = originalLookPath
		execCommand = originalCommand
	}()

	execLookPath = func(file string) (string, error) {
		return "/opt/homebrew/bin/brew", nil
	}
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false") // always fails
	}

	result := CheckForUpdate("1.0.0")

	assert.Equal(t, "1.0.0", result.CurrentVersion)
	assert.Error(t, result.Error)
}

func TestCheckForUpdate_NoUpdateAvailable(t *testing.T) {
	originalLookPath := execLookPath
	originalCommand := execCommand
	defer func() {
		execLookPath = originalLookPath
		execCommand = originalCommand
	}()

	execLookPath = func(file string) (string, error) {
		return "/opt/homebrew/bin/brew", nil
	}
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "mac-cleanup-go: stable 1.0.0")
	}

	result := CheckForUpdate("1.0.0")

	assert.Equal(t, "1.0.0", result.CurrentVersion)
	assert.Equal(t, "1.0.0", result.LatestVersion)
	assert.False(t, result.UpdateAvailable)
}

func TestRunUpdate_Success(t *testing.T) {
	original := execCommand
	defer func() { execCommand = original }()
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	err := RunUpdate()

	assert.NoError(t, err)
}

func TestRunUpdate_Error(t *testing.T) {
	original := execCommand
	defer func() { execCommand = original }()
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}

	err := RunUpdate()

	assert.Error(t, err)
}
