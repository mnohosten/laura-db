package concurrent

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestShardedLRUCache_Basic(t *testing.T) {
	cache := NewShardedLRUCache(100, 5*time.Minute, 4)

	// Test Put and Get
	cache.Put("key1", "value1")
	if v, ok := cache.Get("key1"); !ok || v.(string) != "value1" {
		t.Errorf("Expected 'value1', got %v", v)
	}

	// Test non-existent key
	if _, ok := cache.Get("key2"); ok {
		t.Error("Get on non-existent key should return false")
	}
}

func TestShardedLRUCache_Overwrite(t *testing.T) {
	cache := NewShardedLRUCache(100, 5*time.Minute, 4)

	cache.Put("key1", "value1")
	cache.Put("key1", "value2")

	if v, ok := cache.Get("key1"); !ok || v.(string) != "value2" {
		t.Errorf("Expected 'value2', got %v", v)
	}
}

func TestShardedLRUCache_Eviction(t *testing.T) {
	cache := NewShardedLRUCache(4, 5*time.Minute, 2) // 4 capacity, 2 shards = 2 per shard

	// Fill the cache
	cache.Put("key1", "value1")
	cache.Put("key2", "value2")
	cache.Put("key3", "value3")
	cache.Put("key4", "value4")

	// Add one more - should trigger eviction
	cache.Put("key5", "value5")

	// Size should be at most capacity
	size := cache.Size()
	if size > 4 {
		t.Errorf("Expected size <= 4, got %d", size)
	}
}

func TestShardedLRUCache_TTL(t *testing.T) {
	cache := NewShardedLRUCache(100, 100*time.Millisecond, 4)

	cache.Put("key1", "value1")

	// Should be available immediately
	if _, ok := cache.Get("key1"); !ok {
		t.Error("Key should be available")
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	if _, ok := cache.Get("key1"); ok {
		t.Error("Key should be expired")
	}
}

func TestShardedLRUCache_Clear(t *testing.T) {
	cache := NewShardedLRUCache(100, 5*time.Minute, 4)

	for i := 0; i < 10; i++ {
		cache.Put(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
	}

	cache.Clear()

	if size := cache.Size(); size != 0 {
		t.Errorf("Expected size 0 after clear, got %d", size)
	}
}

func TestShardedLRUCache_Size(t *testing.T) {
	cache := NewShardedLRUCache(100, 5*time.Minute, 4)

	for i := 0; i < 50; i++ {
		cache.Put(fmt.Sprintf("key%d", i), i)
	}

	if size := cache.Size(); size != 50 {
		t.Errorf("Expected size 50, got %d", size)
	}
}

func TestShardedLRUCache_Stats(t *testing.T) {
	cache := NewShardedLRUCache(100, 5*time.Minute, 4)

	// Add some data
	cache.Put("key1", "value1")
	cache.Put("key2", "value2")

	// Perform some Gets
	cache.Get("key1") // hit
	cache.Get("key2") // hit
	cache.Get("key3") // miss

	stats := cache.Stats()

	if stats["hits"].(uint64) != 2 {
		t.Errorf("Expected 2 hits, got %v", stats["hits"])
	}
	if stats["misses"].(uint64) != 1 {
		t.Errorf("Expected 1 miss, got %v", stats["misses"])
	}
	if stats["shard_count"].(int) != 4 {
		t.Errorf("Expected 4 shards, got %v", stats["shard_count"])
	}
}

func TestShardedLRUCache_CleanupExpired(t *testing.T) {
	cache := NewShardedLRUCache(100, 100*time.Millisecond, 4)

	for i := 0; i < 10; i++ {
		cache.Put(fmt.Sprintf("key%d", i), i)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	removed := cache.CleanupExpired()

	if removed != 10 {
		t.Errorf("Expected 10 removed, got %d", removed)
	}
	if size := cache.Size(); size != 0 {
		t.Errorf("Expected size 0 after cleanup, got %d", size)
	}
}

func TestShardedLRUCache_ConcurrentPut(t *testing.T) {
	cache := NewShardedLRUCache(2000, 5*time.Minute, 8) // Increased capacity to avoid evictions
	iterations := 100
	goroutines := 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cache.Put(key, j)
			}
		}(i)
	}

	wg.Wait()

	expected := goroutines * iterations
	size := cache.Size()
	if size != expected {
		t.Errorf("Expected size %d, got %d", expected, size)
	}
}

func TestShardedLRUCache_ConcurrentPutGet(t *testing.T) {
	cache := NewShardedLRUCache(1000, 5*time.Minute, 8)
	iterations := 100
	goroutines := 10

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Writers
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cache.Put(key, j)
			}
		}(i)
	}

	// Readers
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cache.Get(key) // May or may not exist
			}
		}(i)
	}

	wg.Wait()

	// Verify no panics and reasonable size
	size := cache.Size()
	if size < 0 || size > 1000 {
		t.Errorf("Unexpected size: %d", size)
	}
}

func TestShardedLRUCache_LRUBehavior(t *testing.T) {
	cache := NewShardedLRUCache(3, 5*time.Minute, 1) // Single shard for predictable LRU

	cache.Put("key1", "value1")
	cache.Put("key2", "value2")
	cache.Put("key3", "value3")

	// Access key1 to make it most recent
	cache.Get("key1")

	// Add key4 - should evict key2 (least recently used)
	cache.Put("key4", "value4")

	// key1 and key3 and key4 should exist
	if _, ok := cache.Get("key1"); !ok {
		t.Error("key1 should exist")
	}
	if _, ok := cache.Get("key3"); !ok {
		t.Error("key3 should exist")
	}
	if _, ok := cache.Get("key4"); !ok {
		t.Error("key4 should exist")
	}
}

func TestShardedLRUCache_ShardDistribution(t *testing.T) {
	cache := NewShardedLRUCache(1000, 5*time.Minute, 8)

	// Add many keys and verify they're distributed across shards
	for i := 0; i < 1000; i++ {
		cache.Put(fmt.Sprintf("key%d", i), i)
	}

	// Check that multiple shards have items
	nonEmptyShards := 0
	for _, shard := range cache.shards {
		shard.mu.RLock()
		if len(shard.items) > 0 {
			nonEmptyShards++
		}
		shard.mu.RUnlock()
	}

	// With 1000 items and 8 shards, we expect all shards to have items
	// (statistically very unlikely for any shard to be empty)
	if nonEmptyShards < 6 {
		t.Errorf("Expected at least 6 non-empty shards, got %d", nonEmptyShards)
	}
}

func TestNextPowerOfTwo(t *testing.T) {
	tests := []struct {
		input    uint32
		expected uint32
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 4},
		{4, 4},
		{5, 8},
		{7, 8},
		{8, 8},
		{9, 16},
		{15, 16},
		{16, 16},
		{17, 32},
		{100, 128},
	}

	for _, test := range tests {
		result := nextPowerOfTwo(test.input)
		if result != test.expected {
			t.Errorf("nextPowerOfTwo(%d) = %d, expected %d", test.input, result, test.expected)
		}
	}
}

func TestFNV32Hash(t *testing.T) {
	// Test hash consistency
	key := "test-key-123"
	hash1 := fnv32(key)
	hash2 := fnv32(key)

	if hash1 != hash2 {
		t.Error("Hash should be consistent")
	}

	// Test different keys produce different hashes
	hash3 := fnv32("different-key")
	if hash1 == hash3 {
		t.Error("Different keys should produce different hashes")
	}
}
