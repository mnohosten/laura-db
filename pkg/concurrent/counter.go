package concurrent

import (
	"sync/atomic"
)

// Counter is a lock-free counter using atomic operations
type Counter struct {
	value uint64
}

// NewCounter creates a new lock-free counter
func NewCounter() *Counter {
	return &Counter{value: 0}
}

// Inc increments the counter by 1 and returns the new value
func (c *Counter) Inc() uint64 {
	return atomic.AddUint64(&c.value, 1)
}

// Add increments the counter by delta and returns the new value
func (c *Counter) Add(delta uint64) uint64 {
	return atomic.AddUint64(&c.value, delta)
}

// Dec decrements the counter by 1 and returns the new value
func (c *Counter) Dec() uint64 {
	return atomic.AddUint64(&c.value, ^uint64(0)) // Two's complement for -1
}

// Sub decrements the counter by delta and returns the new value
func (c *Counter) Sub(delta uint64) uint64 {
	return atomic.AddUint64(&c.value, ^(delta - 1)) // Two's complement for -delta
}

// Load returns the current value
func (c *Counter) Load() uint64 {
	return atomic.LoadUint64(&c.value)
}

// Store sets the counter to a specific value
func (c *Counter) Store(value uint64) {
	atomic.StoreUint64(&c.value, value)
}

// CompareAndSwap performs a CAS operation
// Returns true if the swap was successful
func (c *Counter) CompareAndSwap(old, new uint64) bool {
	return atomic.CompareAndSwapUint64(&c.value, old, new)
}

// Swap atomically stores new value and returns the old value
func (c *Counter) Swap(new uint64) uint64 {
	return atomic.SwapUint64(&c.value, new)
}

// Reset sets the counter to 0 and returns the previous value
func (c *Counter) Reset() uint64 {
	return atomic.SwapUint64(&c.value, 0)
}
