package cache

import (
	"testing"
	"time"
)

func TestLRUCacheBasicOperations(t *testing.T) {
	cache := NewLRUCache(3, 5*time.Minute)

	// Test Put and Get
	cache.Put("key1", "value1")
	value, found := cache.Get("key1")
	if !found {
		t.Error("Expected to find key1")
	}
	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}

	// Test miss
	_, found = cache.Get("nonexistent")
	if found {
		t.Error("Should not find nonexistent key")
	}
}

func TestLRUCacheEviction(t *testing.T) {
	cache := NewLRUCache(3, 5*time.Minute)

	// Fill cache to capacity
	cache.Put("key1", "value1")
	cache.Put("key2", "value2")
	cache.Put("key3", "value3")

	// Add fourth key, should evict oldest (key1 - hasn't been accessed)
	cache.Put("key4", "value4")

	// key1 should be evicted (it was the oldest and never accessed)
	if _, found := cache.Get("key1"); found {
		t.Error("key1 should have been evicted")
	}

	// Other keys should exist
	if _, found := cache.Get("key2"); !found {
		t.Error("key2 should exist")
	}
	if _, found := cache.Get("key3"); !found {
		t.Error("key3 should exist")
	}
	if _, found := cache.Get("key4"); !found {
		t.Error("key4 should exist")
	}

	// Verify size
	if cache.Size() != 3 {
		t.Errorf("Expected size 3, got %d", cache.Size())
	}
}

func TestLRUCacheLRUBehavior(t *testing.T) {
	cache := NewLRUCache(3, 5*time.Minute)

	// Fill cache
	cache.Put("key1", "value1")
	cache.Put("key2", "value2")
	cache.Put("key3", "value3")

	// Access key1 to make it most recently used
	cache.Get("key1")

	// Add key4, should evict key2 (oldest)
	cache.Put("key4", "value4")

	// key2 should be evicted, key1 should still exist
	if _, found := cache.Get("key2"); found {
		t.Error("key2 should have been evicted")
	}
	if _, found := cache.Get("key1"); !found {
		t.Error("key1 should still exist (was accessed recently)")
	}
}

func TestLRUCacheTTL(t *testing.T) {
	cache := NewLRUCache(10, 100*time.Millisecond)

	// Add entry
	cache.Put("key1", "value1")

	// Should exist immediately
	if _, found := cache.Get("key1"); !found {
		t.Error("key1 should exist")
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Should not exist after TTL
	if _, found := cache.Get("key1"); found {
		t.Error("key1 should have expired")
	}
}

func TestLRUCacheUpdate(t *testing.T) {
	cache := NewLRUCache(3, 5*time.Minute)

	// Add entry
	cache.Put("key1", "value1")

	// Update entry
	cache.Put("key1", "value2")

	// Should have new value
	value, found := cache.Get("key1")
	if !found {
		t.Error("key1 should exist")
	}
	if value != "value2" {
		t.Errorf("Expected value2, got %v", value)
	}

	// Size should still be 1
	if cache.Size() != 1 {
		t.Errorf("Expected size 1, got %d", cache.Size())
	}
}

func TestLRUCacheClear(t *testing.T) {
	cache := NewLRUCache(10, 5*time.Minute)

	// Add entries
	cache.Put("key1", "value1")
	cache.Put("key2", "value2")
	cache.Put("key3", "value3")

	// Verify size
	if cache.Size() != 3 {
		t.Errorf("Expected size 3, got %d", cache.Size())
	}

	// Clear cache
	cache.Clear()

	// Verify empty
	if cache.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", cache.Size())
	}

	// Verify keys don't exist
	if _, found := cache.Get("key1"); found {
		t.Error("key1 should not exist after clear")
	}
}

func TestLRUCacheStats(t *testing.T) {
	cache := NewLRUCache(10, 5*time.Minute)

	// Add some entries
	cache.Put("key1", "value1")
	cache.Put("key2", "value2")

	// Generate hits
	cache.Get("key1")
	cache.Get("key1")
	cache.Get("key2")

	// Generate misses
	cache.Get("key3")
	cache.Get("key4")

	stats := cache.Stats()

	if stats["hits"].(uint64) != 3 {
		t.Errorf("Expected 3 hits, got %v", stats["hits"])
	}
	if stats["misses"].(uint64) != 2 {
		t.Errorf("Expected 2 misses, got %v", stats["misses"])
	}
	if stats["size"].(int) != 2 {
		t.Errorf("Expected size 2, got %v", stats["size"])
	}
}

func TestLRUCacheCleanupExpired(t *testing.T) {
	cache := NewLRUCache(10, 100*time.Millisecond)

	// Add entries
	cache.Put("key1", "value1")
	cache.Put("key2", "value2")
	cache.Put("key3", "value3")

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Add one more entry (should not expire yet)
	cache.Put("key4", "value4")

	// Cleanup expired entries
	removed := cache.CleanupExpired()

	if removed != 3 {
		t.Errorf("Expected to remove 3 expired entries, got %d", removed)
	}

	// key4 should still exist
	if _, found := cache.Get("key4"); !found {
		t.Error("key4 should still exist")
	}

	// Others should be gone
	if _, found := cache.Get("key1"); found {
		t.Error("key1 should be removed")
	}
}

func TestGenerateKey(t *testing.T) {
	// Test that same inputs generate same key
	filter1 := map[string]interface{}{"age": 25}
	sort1 := []interface{}{"name"}
	key1 := GenerateKey(filter1, sort1, 0, 10, nil)
	key2 := GenerateKey(filter1, sort1, 0, 10, nil)

	if key1 != key2 {
		t.Error("Same inputs should generate same key")
	}

	// Test that different inputs generate different keys
	filter2 := map[string]interface{}{"age": 30}
	key3 := GenerateKey(filter2, sort1, 0, 10, nil)

	if key1 == key3 {
		t.Error("Different filters should generate different keys")
	}

	// Test with different skip/limit
	key4 := GenerateKey(filter1, sort1, 10, 20, nil)
	if key1 == key4 {
		t.Error("Different skip/limit should generate different keys")
	}
}

func TestLRUCacheConcurrency(t *testing.T) {
	cache := NewLRUCache(100, 5*time.Minute)

	// Run concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				key := string(rune('a' + (id+j)%26))
				cache.Put(key, id*100+j)
				cache.Get(key)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify cache still works
	cache.Put("test", "value")
	if _, found := cache.Get("test"); !found {
		t.Error("Cache should still work after concurrent operations")
	}
}

func BenchmarkLRUCachePut(b *testing.B) {
	cache := NewLRUCache(1000, 5*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Put(string(rune(i%1000)), i)
	}
}

func BenchmarkLRUCacheGet(b *testing.B) {
	cache := NewLRUCache(1000, 5*time.Minute)

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		cache.Put(string(rune(i)), i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(string(rune(i % 1000)))
	}
}

func BenchmarkGenerateKey(b *testing.B) {
	filter := map[string]interface{}{
		"age":  25,
		"city": "NYC",
		"name": "Alice",
	}
	sort := []interface{}{"name", "age"}
	projection := map[string]bool{"name": true, "age": true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateKey(filter, sort, 0, 10, projection)
	}
}
