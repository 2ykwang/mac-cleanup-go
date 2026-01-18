// Package logger provides file-based structured logging for TUI compatibility.
package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

var (
	configDir = filepath.Join(os.Getenv("HOME"), ".config", "mac-cleanup-go")
	logFile   *os.File
	Log       = slog.New(slog.NewJSONHandler(io.Discard, nil))
)

// Init initializes the logger.
// - debug=true: logs all levels (DEBUG+) to file
// - debug=false: logs WARN/ERROR only to file
func Init(debug bool) error {
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}

	logPath := filepath.Join(configDir, "debug.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	logFile = f

	level := slog.LevelWarn
	if debug {
		level = slog.LevelDebug
	}

	Log = slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{Level: level}))
	return nil
}

func Debug(msg string, args ...any) { Log.Debug(msg, args...) }
func Info(msg string, args ...any)  { Log.Info(msg, args...) }
func Warn(msg string, args ...any)  { Log.Warn(msg, args...) }
func Error(msg string, args ...any) { Log.Error(msg, args...) }

func Close() {
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}
