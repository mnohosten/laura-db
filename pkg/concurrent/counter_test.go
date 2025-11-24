package concurrent

import (
	"sync"
	"testing"
)

func TestCounter_Inc(t *testing.T) {
	c := NewCounter()

	if v := c.Inc(); v != 1 {
		t.Errorf("Expected 1, got %d", v)
	}
	if v := c.Inc(); v != 2 {
		t.Errorf("Expected 2, got %d", v)
	}
	if v := c.Load(); v != 2 {
		t.Errorf("Expected 2, got %d", v)
	}
}

func TestCounter_Dec(t *testing.T) {
	c := NewCounter()
	c.Store(10)

	if v := c.Dec(); v != 9 {
		t.Errorf("Expected 9, got %d", v)
	}
	if v := c.Dec(); v != 8 {
		t.Errorf("Expected 8, got %d", v)
	}
	if v := c.Load(); v != 8 {
		t.Errorf("Expected 8, got %d", v)
	}
}

func TestCounter_Add(t *testing.T) {
	c := NewCounter()

	if v := c.Add(5); v != 5 {
		t.Errorf("Expected 5, got %d", v)
	}
	if v := c.Add(10); v != 15 {
		t.Errorf("Expected 15, got %d", v)
	}
}

func TestCounter_Sub(t *testing.T) {
	c := NewCounter()
	c.Store(20)

	if v := c.Sub(5); v != 15 {
		t.Errorf("Expected 15, got %d", v)
	}
	if v := c.Sub(10); v != 5 {
		t.Errorf("Expected 5, got %d", v)
	}
}

func TestCounter_CompareAndSwap(t *testing.T) {
	c := NewCounter()
	c.Store(10)

	// Successful CAS
	if !c.CompareAndSwap(10, 20) {
		t.Error("CAS should succeed")
	}
	if v := c.Load(); v != 20 {
		t.Errorf("Expected 20, got %d", v)
	}

	// Failed CAS
	if c.CompareAndSwap(10, 30) {
		t.Error("CAS should fail")
	}
	if v := c.Load(); v != 20 {
		t.Errorf("Expected 20, got %d", v)
	}
}

func TestCounter_Swap(t *testing.T) {
	c := NewCounter()
	c.Store(10)

	old := c.Swap(20)
	if old != 10 {
		t.Errorf("Expected old value 10, got %d", old)
	}
	if v := c.Load(); v != 20 {
		t.Errorf("Expected 20, got %d", v)
	}
}

func TestCounter_Reset(t *testing.T) {
	c := NewCounter()
	c.Store(100)

	old := c.Reset()
	if old != 100 {
		t.Errorf("Expected old value 100, got %d", old)
	}
	if v := c.Load(); v != 0 {
		t.Errorf("Expected 0, got %d", v)
	}
}

func TestCounter_Concurrent(t *testing.T) {
	c := NewCounter()
	iterations := 1000
	goroutines := 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				c.Inc()
			}
		}()
	}

	wg.Wait()

	expected := uint64(goroutines * iterations)
	if v := c.Load(); v != expected {
		t.Errorf("Expected %d, got %d", expected, v)
	}
}

func TestCounter_ConcurrentIncDec(t *testing.T) {
	c := NewCounter()
	c.Store(1000000)
	iterations := 1000
	goroutines := 10

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Incrementers
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				c.Inc()
			}
		}()
	}

	// Decrementers
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				c.Dec()
			}
		}()
	}

	wg.Wait()

	// Should be back to initial value
	expected := uint64(1000000)
	if v := c.Load(); v != expected {
		t.Errorf("Expected %d, got %d", expected, v)
	}
}
