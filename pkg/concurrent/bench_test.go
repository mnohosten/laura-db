package concurrent

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// Benchmark Counter operations

func BenchmarkCounter_Inc(b *testing.B) {
	c := NewCounter()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Inc()
	}
}

func BenchmarkCounter_IncParallel(b *testing.B) {
	c := NewCounter()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

func BenchmarkCounter_Load(b *testing.B) {
	c := NewCounter()
	c.Store(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Load()
	}
}

func BenchmarkCounter_LoadParallel(b *testing.B) {
	c := NewCounter()
	c.Store(100)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Load()
		}
	})
}

func BenchmarkCounter_CompareAndSwap(b *testing.B) {
	c := NewCounter()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		old := c.Load()
		c.CompareAndSwap(old, old+1)
	}
}

// Benchmark LockFreeStack operations

func BenchmarkStack_Push(b *testing.B) {
	s := NewLockFreeStack()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Push(i)
	}
}

func BenchmarkStack_PushParallel(b *testing.B) {
	s := NewLockFreeStack()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			s.Push(i)
			i++
		}
	})
}

func BenchmarkStack_Pop(b *testing.B) {
	s := NewLockFreeStack()
	// Pre-fill the stack
	for i := 0; i < b.N; i++ {
		s.Push(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Pop()
	}
}

func BenchmarkStack_PopParallel(b *testing.B) {
	s := NewLockFreeStack()
	// Pre-fill the stack
	for i := 0; i < 1000000; i++ {
		s.Push(i)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Pop()
		}
	})
}

func BenchmarkStack_PushPop(b *testing.B) {
	s := NewLockFreeStack()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Push(i)
		s.Pop()
	}
}

func BenchmarkStack_PushPopParallel(b *testing.B) {
	s := NewLockFreeStack()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			s.Push(i)
			s.Pop()
			i++
		}
	})
}

// Benchmark ShardedLRUCache operations

func BenchmarkShardedLRU_Put(b *testing.B) {
	cache := NewShardedLRUCache(10000, 5*time.Minute, 8)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i)
		cache.Put(key, i)
	}
}

func BenchmarkShardedLRU_PutParallel(b *testing.B) {
	cache := NewShardedLRUCache(10000, 5*time.Minute, 8)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key%d", i)
			cache.Put(key, i)
			i++
		}
	})
}

func BenchmarkShardedLRU_Get(b *testing.B) {
	cache := NewShardedLRUCache(10000, 5*time.Minute, 8)
	// Pre-populate
	for i := 0; i < 1000; i++ {
		cache.Put(fmt.Sprintf("key%d", i), i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%1000)
		cache.Get(key)
	}
}

func BenchmarkShardedLRU_GetParallel(b *testing.B) {
	cache := NewShardedLRUCache(10000, 5*time.Minute, 8)
	// Pre-populate
	for i := 0; i < 1000; i++ {
		cache.Put(fmt.Sprintf("key%d", i), i)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key%d", i%1000)
			cache.Get(key)
			i++
		}
	})
}

func BenchmarkShardedLRU_Mixed(b *testing.B) {
	cache := NewShardedLRUCache(10000, 5*time.Minute, 8)
	// Pre-populate
	for i := 0; i < 1000; i++ {
		cache.Put(fmt.Sprintf("key%d", i), i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%1000)
		if i%5 == 0 {
			cache.Put(key, i)
		} else {
			cache.Get(key)
		}
	}
}

func BenchmarkShardedLRU_MixedParallel(b *testing.B) {
	cache := NewShardedLRUCache(10000, 5*time.Minute, 8)
	// Pre-populate
	for i := 0; i < 1000; i++ {
		cache.Put(fmt.Sprintf("key%d", i), i)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key%d", i%1000)
			if i%5 == 0 {
				cache.Put(key, i)
			} else {
				cache.Get(key)
			}
			i++
		}
	})
}

// Comparison benchmarks: Lock-free vs Mutex-based counter

type MutexCounter struct {
	mu    sync.Mutex
	value uint64
}

func (c *MutexCounter) Inc() uint64 {
	c.mu.Lock()
	c.value++
	v := c.value
	c.mu.Unlock()
	return v
}

func BenchmarkMutexCounter_Inc(b *testing.B) {
	c := &MutexCounter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Inc()
	}
}

func BenchmarkMutexCounter_IncParallel(b *testing.B) {
	c := &MutexCounter{}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

// Benchmark different shard counts

func BenchmarkShardedLRU_Shards1(b *testing.B) {
	cache := NewShardedLRUCache(10000, 5*time.Minute, 1)
	benchmarkShardedCache(b, cache)
}

func BenchmarkShardedLRU_Shards2(b *testing.B) {
	cache := NewShardedLRUCache(10000, 5*time.Minute, 2)
	benchmarkShardedCache(b, cache)
}

func BenchmarkShardedLRU_Shards4(b *testing.B) {
	cache := NewShardedLRUCache(10000, 5*time.Minute, 4)
	benchmarkShardedCache(b, cache)
}

func BenchmarkShardedLRU_Shards8(b *testing.B) {
	cache := NewShardedLRUCache(10000, 5*time.Minute, 8)
	benchmarkShardedCache(b, cache)
}

func BenchmarkShardedLRU_Shards16(b *testing.B) {
	cache := NewShardedLRUCache(10000, 5*time.Minute, 16)
	benchmarkShardedCache(b, cache)
}

func BenchmarkShardedLRU_Shards32(b *testing.B) {
	cache := NewShardedLRUCache(10000, 5*time.Minute, 32)
	benchmarkShardedCache(b, cache)
}

func benchmarkShardedCache(b *testing.B, cache *ShardedLRUCache) {
	// Pre-populate
	for i := 0; i < 1000; i++ {
		cache.Put(fmt.Sprintf("key%d", i), i)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key%d", i%1000)
			if i%5 == 0 {
				cache.Put(key, i)
			} else {
				cache.Get(key)
			}
			i++
		}
	})
}
