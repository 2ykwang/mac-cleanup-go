package tui

// RingBuffer is a fixed-size circular buffer for recent items
type RingBuffer[T any] struct {
	items []T
	head  int // Next write position
	size  int // Current number of items
	cap   int // Maximum capacity
}

// NewRingBuffer creates a new ring buffer with the given capacity
func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	if capacity < 1 {
		capacity = 1
	}
	return &RingBuffer[T]{
		items: make([]T, capacity),
		cap:   capacity,
	}
}

// Push adds an item to the buffer, overwriting the oldest if full
func (r *RingBuffer[T]) Push(item T) {
	r.items[r.head] = item
	r.head = (r.head + 1) % r.cap
	if r.size < r.cap {
		r.size++
	}
}

// Items returns all items in the buffer from oldest to newest
func (r *RingBuffer[T]) Items() []T {
	if r.size == 0 {
		return nil
	}
	result := make([]T, r.size)
	if r.size < r.cap {
		// Buffer not full yet, items start at 0
		copy(result, r.items[:r.size])
	} else {
		// Buffer is full, oldest item is at head
		copy(result, r.items[r.head:])
		copy(result[r.cap-r.head:], r.items[:r.head])
	}
	return result
}

// Len returns the current number of items in the buffer
func (r *RingBuffer[T]) Len() int {
	return r.size
}

// Clear removes all items from the buffer
func (r *RingBuffer[T]) Clear() {
	r.head = 0
	r.size = 0
}
