package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/2ykwang/mac-cleanup-go/internal/userconfig"
)

func newTestModelForViewCleaning() *Model {
	m := &Model{}
	m.width = 80
	m.height = 24
	m.recentDeleted = NewRingBuffer[DeletedItemEntry](defaultRecentItemsCapacity)
	m.userConfig = &userconfig.UserConfig{ExcludedPaths: make(map[string][]string)}
	return m
}

func TestRenderRecentDeleted_SuccessIcon(t *testing.T) {
	m := newTestModelForViewCleaning()
	m.recentDeleted.Push(DeletedItemEntry{
		Path:    "/path/to/file.txt",
		Name:    "file.txt",
		Size:    1024,
		Success: true,
	})

	output := m.renderRecentDeleted()

	assert.Contains(t, output, "✓", "success item should have checkmark icon")
	assert.Contains(t, output, "file.txt", "should contain file path")
}

func TestRenderRecentDeleted_FailureIcon(t *testing.T) {
	m := newTestModelForViewCleaning()
	m.recentDeleted.Push(DeletedItemEntry{
		Path:    "/path/to/locked.txt",
		Name:    "locked.txt",
		Size:    512,
		Success: false,
		ErrMsg:  "permission denied",
	})

	output := m.renderRecentDeleted()

	assert.Contains(t, output, "✗", "failed item should have X icon")
	assert.Contains(t, output, "locked.txt", "should contain file path")
}

func TestRenderRecentDeleted_MixedItems(t *testing.T) {
	m := newTestModelForViewCleaning()
	m.recentDeleted.Push(DeletedItemEntry{
		Path:    "/path/to/success.txt",
		Name:    "success.txt",
		Size:    1024,
		Success: true,
	})
	m.recentDeleted.Push(DeletedItemEntry{
		Path:    "/path/to/failed.txt",
		Name:    "failed.txt",
		Size:    512,
		Success: false,
		ErrMsg:  "error",
	})

	output := m.renderRecentDeleted()

	assert.Contains(t, output, "✓", "should have success icon")
	assert.Contains(t, output, "✗", "should have failure icon")
	assert.Contains(t, output, "success.txt", "should contain success path")
	assert.Contains(t, output, "failed.txt", "should contain failed path")
}

func TestRenderRecentDeleted_FileSize(t *testing.T) {
	m := newTestModelForViewCleaning()
	m.recentDeleted.Push(DeletedItemEntry{
		Path:    "/path/to/file.txt",
		Name:    "file.txt",
		Size:    1024 * 1024, // 1 MB
		Success: true,
	})

	output := m.renderRecentDeleted()

	// utils.FormatSize(1024*1024) returns "1.0 MB"
	assert.Contains(t, output, "1.0 MB", "should display formatted file size")
}

func TestRenderRecentDeleted_LongPathTruncation(t *testing.T) {
	m := newTestModelForViewCleaning()
	longPath := "/very/long/path/that/should/be/truncated/to/fit/display/width/file.txt"
	m.recentDeleted.Push(DeletedItemEntry{
		Path:    longPath,
		Name:    "file.txt",
		Size:    1024,
		Success: true,
	})

	output := m.renderRecentDeleted()

	// shortenPath(longPath, 40) should truncate with "..."
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "✓") {
			// The displayed path should be shorter than original
			assert.True(t, len(line) < len(longPath)+20, "long path should be truncated")
			break
		}
	}
}

func TestRenderRecentDeleted_EmptyBuffer(t *testing.T) {
	m := newTestModelForViewCleaning()

	output := m.renderRecentDeleted()

	assert.Empty(t, output, "empty buffer should produce empty output")
}
