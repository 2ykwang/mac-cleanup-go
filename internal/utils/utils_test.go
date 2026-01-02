package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/test", filepath.Join(home, "test")},
		{"~/", filepath.Join(home, "")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		result := ExpandPath(tt.input)
		if result != tt.expected {
			t.Errorf("ExpandPath(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
		{1610612736, "1.5 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		result := FormatSize(tt.bytes)
		if result != tt.expected {
			t.Errorf("FormatSize(%d) = %q, expected %q", tt.bytes, result, tt.expected)
		}
	}
}

func TestPathExists(t *testing.T) {
	// Test existing path
	if !PathExists("/tmp") {
		t.Error("PathExists(/tmp) should be true")
	}

	// Test non-existing path
	if PathExists("/nonexistent/path/12345") {
		t.Error("PathExists(/nonexistent/path/12345) should be false")
	}

	// Test home directory expansion
	if !PathExists("~/") {
		t.Error("PathExists(~/) should be true")
	}
}

func TestCommandExists(t *testing.T) {
	// Test common command
	if !CommandExists("ls") {
		t.Error("CommandExists(ls) should be true")
	}

	// Test non-existing command
	if CommandExists("nonexistentcommand12345") {
		t.Error("CommandExists(nonexistentcommand12345) should be false")
	}
}

func TestGetDirSize(t *testing.T) {
	// Create temp directory with files
	tmpDir, err := os.MkdirTemp("", "test-dir-size")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files with known sizes
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	if err := os.WriteFile(file1, make([]byte, 100), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, make([]byte, 200), 0o644); err != nil {
		t.Fatal(err)
	}

	size, err := GetDirSize(tmpDir)
	if err != nil {
		t.Errorf("GetDirSize error: %v", err)
	}

	if size != 300 {
		t.Errorf("GetDirSize = %d, expected 300", size)
	}
}

func TestGetFileSize(t *testing.T) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "test-file-size")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := make([]byte, 1024)
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	size, err := GetFileSize(tmpFile.Name())
	if err != nil {
		t.Errorf("GetFileSize error: %v", err)
	}

	if size != 1024 {
		t.Errorf("GetFileSize = %d, expected 1024", size)
	}
}

func TestGlobPaths(t *testing.T) {
	// Create temp directory with files
	tmpDir, err := os.MkdirTemp("", "test-glob")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	for _, name := range []string{"a.txt", "b.txt", "c.log"} {
		f, _ := os.Create(filepath.Join(tmpDir, name))
		f.Close()
	}

	// Test glob
	pattern := filepath.Join(tmpDir, "*.txt")
	paths, err := GlobPaths(pattern)
	if err != nil {
		t.Errorf("GlobPaths error: %v", err)
	}

	if len(paths) != 2 {
		t.Errorf("GlobPaths found %d files, expected 2", len(paths))
	}
}
