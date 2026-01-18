package logger

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit_DebugTrue_LogsAllLevels(t *testing.T) {
	tmpDir := t.TempDir()
	originalConfigDir := configDir
	configDir = tmpDir
	defer func() { configDir = originalConfigDir }()

	err := Init(true)
	require.NoError(t, err)

	Debug("debug msg")
	Info("info msg")
	Warn("warn msg")

	logPath := filepath.Join(tmpDir, "debug.log")
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "debug msg")
	assert.Contains(t, string(content), "info msg")
	assert.Contains(t, string(content), "warn msg")

	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

func TestInit_DebugFalse_LogsWarnErrorOnly(t *testing.T) {
	tmpDir := t.TempDir()
	originalConfigDir := configDir
	configDir = tmpDir
	defer func() { configDir = originalConfigDir }()

	err := Init(false)
	require.NoError(t, err)

	Debug("debug msg")
	Info("info msg")
	Warn("warn msg")
	Error("error msg")

	logPath := filepath.Join(tmpDir, "debug.log")
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	assert.NotContains(t, string(content), "debug msg")
	assert.NotContains(t, string(content), "info msg")
	assert.Contains(t, string(content), "warn msg")
	assert.Contains(t, string(content), "error msg")

	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

func TestInit_CreatesConfigDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "nested", "config")
	originalConfigDir := configDir
	configDir = nestedDir
	defer func() { configDir = originalConfigDir }()

	err := Init(true)

	require.NoError(t, err)
	_, err = os.Stat(nestedDir)
	assert.NoError(t, err, "config directory should be created")

	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

func TestInit_CalledTwice_ClosesOldFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalConfigDir := configDir
	configDir = tmpDir
	defer func() { configDir = originalConfigDir }()

	err1 := Init(true)
	require.NoError(t, err1)
	Info("first log")

	err2 := Init(true)
	require.NoError(t, err2)
	Info("second log")

	logPath := filepath.Join(tmpDir, "debug.log")
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(content), "second log"))

	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

func TestDebug_WithDiscard_DoesNotPanic(t *testing.T) {
	Log = slog.New(slog.NewJSONHandler(bytes.NewBuffer(nil), nil))

	assert.NotPanics(t, func() {
		Debug("test message")
		Info("test message")
		Warn("test message")
		Error("test message")
	})
}
