package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRingBuffer_NewRingBuffer(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
		wantCap  int
	}{
		{"normal capacity", 10, 10},
		{"zero capacity becomes 1", 0, 1},
		{"negative capacity becomes 1", -5, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := NewRingBuffer[int](tt.capacity)
			assert.Equal(t, tt.wantCap, rb.cap)
			assert.Equal(t, 0, rb.Len())
		})
	}
}

func TestRingBuffer_Push(t *testing.T) {
	rb := NewRingBuffer[int](3)

	rb.Push(1)
	assert.Equal(t, 1, rb.Len())

	rb.Push(2)
	assert.Equal(t, 2, rb.Len())

	rb.Push(3)
	assert.Equal(t, 3, rb.Len(), "buffer should be full")

	rb.Push(4)
	assert.Equal(t, 3, rb.Len(), "length should stay at capacity after overflow")
}

func TestRingBuffer_Items(t *testing.T) {
	t.Run("empty buffer returns nil", func(t *testing.T) {
		rb := NewRingBuffer[int](3)
		assert.Nil(t, rb.Items())
	})

	t.Run("not full buffer returns items in order", func(t *testing.T) {
		rb := NewRingBuffer[int](5)
		rb.Push(1)
		rb.Push(2)
		rb.Push(3)

		items := rb.Items()
		assert.Equal(t, []int{1, 2, 3}, items)
	})

	t.Run("full buffer with overflow returns items in correct order", func(t *testing.T) {
		rb := NewRingBuffer[int](3)
		rb.Push(1)
		rb.Push(2)
		rb.Push(3)
		rb.Push(4) // Overwrites 1
		rb.Push(5) // Overwrites 2

		items := rb.Items()
		assert.Equal(t, []int{3, 4, 5}, items)
	})
}

func TestRingBuffer_Clear(t *testing.T) {
	rb := NewRingBuffer[int](3)
	rb.Push(1)
	rb.Push(2)
	rb.Push(3)

	rb.Clear()

	assert.Equal(t, 0, rb.Len())
	assert.Nil(t, rb.Items())

	rb.Push(10)
	assert.Equal(t, 1, rb.Len())
	assert.Equal(t, []int{10}, rb.Items())
}

func TestRingBuffer_WithDeletedItemEntry(t *testing.T) {
	rb := NewRingBuffer[DeletedItemEntry](2)

	entry1 := DeletedItemEntry{
		Path:    "/path/to/file1",
		Name:    "file1",
		Size:    1024,
		Success: true,
	}
	entry2 := DeletedItemEntry{
		Path:    "/path/to/file2",
		Name:    "file2",
		Size:    2048,
		Success: false,
		ErrMsg:  "permission denied",
	}
	entry3 := DeletedItemEntry{
		Path:    "/path/to/file3",
		Name:    "file3",
		Size:    512,
		Success: true,
	}

	rb.Push(entry1)
	rb.Push(entry2)
	rb.Push(entry3) // Overwrites entry1

	items := rb.Items()
	require.Len(t, items, 2)

	assert.Equal(t, "file2", items[0].Name)
	assert.False(t, items[0].Success)

	assert.Equal(t, "file3", items[1].Name)
	assert.True(t, items[1].Success)
}
