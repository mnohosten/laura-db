package concurrent

import (
	"sync"
	"testing"
)

func TestLockFreeStack_PushPop(t *testing.T) {
	s := NewLockFreeStack()

	// Test with empty stack
	if _, ok := s.Pop(); ok {
		t.Error("Pop on empty stack should return false")
	}

	// Push some values
	s.Push(1)
	s.Push(2)
	s.Push(3)

	// Pop in reverse order (LIFO)
	if v, ok := s.Pop(); !ok || v.(int) != 3 {
		t.Errorf("Expected 3, got %v", v)
	}
	if v, ok := s.Pop(); !ok || v.(int) != 2 {
		t.Errorf("Expected 2, got %v", v)
	}
	if v, ok := s.Pop(); !ok || v.(int) != 1 {
		t.Errorf("Expected 1, got %v", v)
	}

	// Stack should be empty
	if _, ok := s.Pop(); ok {
		t.Error("Stack should be empty")
	}
}

func TestLockFreeStack_Peek(t *testing.T) {
	s := NewLockFreeStack()

	// Test with empty stack
	if _, ok := s.Peek(); ok {
		t.Error("Peek on empty stack should return false")
	}

	s.Push("hello")
	s.Push("world")

	// Peek should return top without removing
	if v, ok := s.Peek(); !ok || v.(string) != "world" {
		t.Errorf("Expected 'world', got %v", v)
	}
	if v, ok := s.Peek(); !ok || v.(string) != "world" {
		t.Errorf("Expected 'world' again, got %v", v)
	}

	// Size should still be 2
	if size := s.Size(); size != 2 {
		t.Errorf("Expected size 2, got %d", size)
	}
}

func TestLockFreeStack_IsEmpty(t *testing.T) {
	s := NewLockFreeStack()

	if !s.IsEmpty() {
		t.Error("New stack should be empty")
	}

	s.Push(1)
	if s.IsEmpty() {
		t.Error("Stack should not be empty after push")
	}

	s.Pop()
	if !s.IsEmpty() {
		t.Error("Stack should be empty after popping all elements")
	}
}

func TestLockFreeStack_Size(t *testing.T) {
	s := NewLockFreeStack()

	if size := s.Size(); size != 0 {
		t.Errorf("Expected size 0, got %d", size)
	}

	for i := 0; i < 10; i++ {
		s.Push(i)
	}

	if size := s.Size(); size != 10 {
		t.Errorf("Expected size 10, got %d", size)
	}

	for i := 0; i < 5; i++ {
		s.Pop()
	}

	if size := s.Size(); size != 5 {
		t.Errorf("Expected size 5, got %d", size)
	}
}

func TestLockFreeStack_Clear(t *testing.T) {
	s := NewLockFreeStack()

	for i := 0; i < 100; i++ {
		s.Push(i)
	}

	s.Clear()

	if !s.IsEmpty() {
		t.Error("Stack should be empty after clear")
	}
	if size := s.Size(); size != 0 {
		t.Errorf("Expected size 0 after clear, got %d", size)
	}
}

func TestLockFreeStack_ConcurrentPush(t *testing.T) {
	s := NewLockFreeStack()
	iterations := 1000
	goroutines := 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				s.Push(id*iterations + j)
			}
		}(i)
	}

	wg.Wait()

	expected := goroutines * iterations
	size := s.Size()
	if size != expected {
		t.Errorf("Expected size %d, got %d", expected, size)
	}
}

func TestLockFreeStack_ConcurrentPushPop(t *testing.T) {
	s := NewLockFreeStack()
	iterations := 1000
	goroutines := 5

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Pushers
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				s.Push(id*iterations + j)
			}
		}(i)
	}

	// Poppers
	popCount := 0
	var popMu sync.Mutex
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if _, ok := s.Pop(); ok {
					popMu.Lock()
					popCount++
					popMu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	// Total pushed = iterations * goroutines
	// Total popped should be <= total pushed
	totalPushed := iterations * goroutines
	if popCount > totalPushed {
		t.Errorf("Popped more items (%d) than pushed (%d)", popCount, totalPushed)
	}

	// Size should be totalPushed - popCount
	size := s.Size()
	expected := totalPushed - popCount
	if size != expected {
		t.Errorf("Expected final size %d, got %d", expected, size)
	}
}

func TestLockFreeStack_MixedOperations(t *testing.T) {
	s := NewLockFreeStack()
	iterations := 100
	goroutines := 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Mix of operations
				s.Push(id*iterations + j)
				s.Peek()
				if j%2 == 0 {
					s.Pop()
				}
				s.IsEmpty()
			}
		}(i)
	}

	wg.Wait()

	// Just verify no panics occurred
	// Final size is non-deterministic but should be valid
	size := s.Size()
	if size < 0 {
		t.Errorf("Invalid size: %d", size)
	}
}

func TestLockFreeStack_TypeSafety(t *testing.T) {
	s := NewLockFreeStack()

	// Test with different types
	s.Push(42)
	s.Push("string")
	s.Push([]int{1, 2, 3})
	s.Push(struct{ Name string }{"test"})

	if v, ok := s.Pop(); !ok || v.(struct{ Name string }).Name != "test" {
		t.Error("Failed to pop struct")
	}
	if v, ok := s.Pop(); !ok || len(v.([]int)) != 3 {
		t.Error("Failed to pop slice")
	}
	if v, ok := s.Pop(); !ok || v.(string) != "string" {
		t.Error("Failed to pop string")
	}
	if v, ok := s.Pop(); !ok || v.(int) != 42 {
		t.Error("Failed to pop int")
	}
}
