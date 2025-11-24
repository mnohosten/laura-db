package concurrent

import (
	"sync/atomic"
	"unsafe"
)

// LockFreeStack is a lock-free stack implementation using CAS operations
// Based on Treiber's stack algorithm
type LockFreeStack struct {
	head unsafe.Pointer // *stackNode
}

// stackNode represents a node in the stack
type stackNode struct {
	value interface{}
	next  unsafe.Pointer // *stackNode
}

// NewLockFreeStack creates a new lock-free stack
func NewLockFreeStack() *LockFreeStack {
	return &LockFreeStack{
		head: nil,
	}
}

// Push adds a value to the top of the stack
func (s *LockFreeStack) Push(value interface{}) {
	node := &stackNode{value: value}

	for {
		// Load current head
		oldHead := atomic.LoadPointer(&s.head)
		node.next = oldHead

		// Try to CAS the head to point to new node
		if atomic.CompareAndSwapPointer(&s.head, oldHead, unsafe.Pointer(node)) {
			return
		}
		// If CAS failed, retry
	}
}

// Pop removes and returns the value from the top of the stack
// Returns (nil, false) if the stack is empty
func (s *LockFreeStack) Pop() (interface{}, bool) {
	for {
		// Load current head
		oldHead := atomic.LoadPointer(&s.head)
		if oldHead == nil {
			return nil, false
		}

		node := (*stackNode)(oldHead)
		newHead := atomic.LoadPointer(&node.next)

		// Try to CAS the head to point to next node
		if atomic.CompareAndSwapPointer(&s.head, oldHead, newHead) {
			return node.value, true
		}
		// If CAS failed, retry
	}
}

// Peek returns the value at the top of the stack without removing it
// Returns (nil, false) if the stack is empty
func (s *LockFreeStack) Peek() (interface{}, bool) {
	head := atomic.LoadPointer(&s.head)
	if head == nil {
		return nil, false
	}

	node := (*stackNode)(head)
	return node.value, true
}

// IsEmpty returns true if the stack is empty
func (s *LockFreeStack) IsEmpty() bool {
	return atomic.LoadPointer(&s.head) == nil
}

// Size returns the approximate number of elements in the stack
// Note: This is not an atomic snapshot and may be inaccurate under high concurrency
func (s *LockFreeStack) Size() int {
	count := 0
	current := atomic.LoadPointer(&s.head)

	for current != nil {
		count++
		node := (*stackNode)(current)
		current = atomic.LoadPointer(&node.next)
	}

	return count
}

// Clear removes all elements from the stack
func (s *LockFreeStack) Clear() {
	atomic.StorePointer(&s.head, nil)
}
