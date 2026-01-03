package tui

import (
	"testing"
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
			if rb.cap != tt.wantCap {
				t.Errorf("NewRingBuffer(%d).cap = %d, want %d", tt.capacity, rb.cap, tt.wantCap)
			}
			if rb.Len() != 0 {
				t.Errorf("NewRingBuffer(%d).Len() = %d, want 0", tt.capacity, rb.Len())
			}
		})
	}
}

func TestRingBuffer_Push(t *testing.T) {
	rb := NewRingBuffer[int](3)

	// Push first item
	rb.Push(1)
	if rb.Len() != 1 {
		t.Errorf("after Push(1), Len() = %d, want 1", rb.Len())
	}

	// Push second item
	rb.Push(2)
	if rb.Len() != 2 {
		t.Errorf("after Push(2), Len() = %d, want 2", rb.Len())
	}

	// Push third item (buffer full)
	rb.Push(3)
	if rb.Len() != 3 {
		t.Errorf("after Push(3), Len() = %d, want 3", rb.Len())
	}

	// Push fourth item (should overwrite oldest)
	rb.Push(4)
	if rb.Len() != 3 {
		t.Errorf("after Push(4), Len() = %d, want 3", rb.Len())
	}
}

func TestRingBuffer_Items(t *testing.T) {
	t.Run("empty buffer returns nil", func(t *testing.T) {
		rb := NewRingBuffer[int](3)
		items := rb.Items()
		if items != nil {
			t.Errorf("empty buffer Items() = %v, want nil", items)
		}
	})

	t.Run("not full buffer returns items in order", func(t *testing.T) {
		rb := NewRingBuffer[int](5)
		rb.Push(1)
		rb.Push(2)
		rb.Push(3)

		items := rb.Items()
		expected := []int{1, 2, 3}

		if len(items) != len(expected) {
			t.Errorf("Items() length = %d, want %d", len(items), len(expected))
		}
		for i, v := range expected {
			if items[i] != v {
				t.Errorf("Items()[%d] = %d, want %d", i, items[i], v)
			}
		}
	})

	t.Run("full buffer with overflow returns items in correct order", func(t *testing.T) {
		rb := NewRingBuffer[int](3)
		rb.Push(1)
		rb.Push(2)
		rb.Push(3)
		rb.Push(4) // Overwrites 1
		rb.Push(5) // Overwrites 2

		items := rb.Items()
		expected := []int{3, 4, 5}

		if len(items) != len(expected) {
			t.Errorf("Items() length = %d, want %d", len(items), len(expected))
		}
		for i, v := range expected {
			if items[i] != v {
				t.Errorf("Items()[%d] = %d, want %d", i, items[i], v)
			}
		}
	})
}

func TestRingBuffer_Clear(t *testing.T) {
	rb := NewRingBuffer[int](3)
	rb.Push(1)
	rb.Push(2)
	rb.Push(3)

	rb.Clear()

	if rb.Len() != 0 {
		t.Errorf("after Clear(), Len() = %d, want 0", rb.Len())
	}
	if rb.Items() != nil {
		t.Errorf("after Clear(), Items() = %v, want nil", rb.Items())
	}

	// Verify can push after clear
	rb.Push(10)
	if rb.Len() != 1 {
		t.Errorf("after Clear() and Push(10), Len() = %d, want 1", rb.Len())
	}
	items := rb.Items()
	if len(items) != 1 || items[0] != 10 {
		t.Errorf("after Clear() and Push(10), Items() = %v, want [10]", items)
	}
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
	if len(items) != 2 {
		t.Fatalf("Items() length = %d, want 2", len(items))
	}

	// Oldest item should be entry2
	if items[0].Name != "file2" {
		t.Errorf("Items()[0].Name = %s, want file2", items[0].Name)
	}
	if items[0].Success {
		t.Errorf("Items()[0].Success = true, want false")
	}

	// Newest item should be entry3
	if items[1].Name != "file3" {
		t.Errorf("Items()[1].Name = %s, want file3", items[1].Name)
	}
	if !items[1].Success {
		t.Errorf("Items()[1].Success = false, want true")
	}
}
