package database

import (
	"os"
	"testing"
	"time"

	"github.com/mnohosten/laura-db/pkg/cache"
)

func TestQueryCache(t *testing.T) {
	// Create a test database and collection
	dir := "./test_cache_query"
	defer os.RemoveAll(dir)

	db, err := Open(DefaultConfig(dir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("users")

	// Insert test data
	_, err = coll.InsertOne(map[string]interface{}{
		"name": "Alice",
		"age":  25,
		"city": "NYC",
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	_, err = coll.InsertOne(map[string]interface{}{
		"name": "Bob",
		"age":  30,
		"city": "LA",
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Query with options (this should be cached)
	filter := map[string]interface{}{"age": map[string]interface{}{"$gte": 20}}
	options := &QueryOptions{
		Limit: 10,
		Skip:  0,
	}

	// First query - should be a cache miss
	initialStats := coll.queryCache.Stats()
	initialMisses := initialStats["misses"].(uint64)

	results1, err := coll.FindWithOptions(filter, options)
	if err != nil {
		t.Fatalf("First query failed: %v", err)
	}

	if len(results1) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results1))
	}

	// Verify cache miss
	stats1 := coll.queryCache.Stats()
	misses1 := stats1["misses"].(uint64)
	if misses1 != initialMisses+1 {
		t.Errorf("Expected cache miss, got misses=%d (expected %d)", misses1, initialMisses+1)
	}

	// Second query with same parameters - should be a cache hit
	results2, err := coll.FindWithOptions(filter, options)
	if err != nil {
		t.Fatalf("Second query failed: %v", err)
	}

	if len(results2) != 2 {
		t.Errorf("Expected 2 results from cache, got %d", len(results2))
	}

	// Verify cache hit
	stats2 := coll.queryCache.Stats()
	hits2 := stats2["hits"].(uint64)
	if hits2 == 0 {
		t.Error("Expected cache hit, got 0 hits")
	}

	// Insert new document - should invalidate cache
	_, err = coll.InsertOne(map[string]interface{}{
		"name": "Charlie",
		"age":  35,
		"city": "SF",
	})
	if err != nil {
		t.Fatalf("Failed to insert third document: %v", err)
	}

	// Verify cache was cleared
	stats3 := coll.queryCache.Stats()
	size := stats3["size"].(int)
	if size != 0 {
		t.Errorf("Expected cache to be cleared after insert, got size=%d", size)
	}

	// Query again - should be cache miss and return 3 results
	results3, err := coll.FindWithOptions(filter, options)
	if err != nil {
		t.Fatalf("Third query failed: %v", err)
	}

	if len(results3) != 3 {
		t.Errorf("Expected 3 results after new insert, got %d", len(results3))
	}
}

func TestCacheInvalidationOnUpdate(t *testing.T) {
	dir := "./test_cache_update"
	defer os.RemoveAll(dir)

	db, err := Open(DefaultConfig(dir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("products")

	// Insert test data
	_, err = coll.InsertOne(map[string]interface{}{
		"name":  "Product A",
		"price": 100,
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Query to populate cache
	filter := map[string]interface{}{"price": map[string]interface{}{"$gte": 50}}
	options := &QueryOptions{Limit: 10}

	results1, err := coll.FindWithOptions(filter, options)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results1) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results1))
	}

	// Verify cache has data
	stats1 := coll.queryCache.Stats()
	size1 := stats1["size"].(int)
	if size1 == 0 {
		t.Error("Expected cache to have entries")
	}

	// Update document - should invalidate cache
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Product A"},
		map[string]interface{}{"$set": map[string]interface{}{"price": 200}},
	)
	if err != nil {
		t.Fatalf("Failed to update: %v", err)
	}

	// Verify cache was cleared
	stats2 := coll.queryCache.Stats()
	size2 := stats2["size"].(int)
	if size2 != 0 {
		t.Errorf("Expected cache to be cleared after update, got size=%d", size2)
	}
}

func TestCacheInvalidationOnDelete(t *testing.T) {
	dir := "./test_cache_delete"
	defer os.RemoveAll(dir)

	db, err := Open(DefaultConfig(dir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("items")

	// Insert test data
	_, err = coll.InsertOne(map[string]interface{}{
		"name":   "Item 1",
		"active": true,
	})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Query to populate cache
	filter := map[string]interface{}{"active": true}
	options := &QueryOptions{Limit: 10}

	results1, err := coll.FindWithOptions(filter, options)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results1) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results1))
	}

	// Verify cache has data
	stats1 := coll.queryCache.Stats()
	size1 := stats1["size"].(int)
	if size1 == 0 {
		t.Error("Expected cache to have entries")
	}

	// Delete document - should invalidate cache
	err = coll.DeleteOne(map[string]interface{}{"name": "Item 1"})
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Verify cache was cleared
	stats2 := coll.queryCache.Stats()
	size2 := stats2["size"].(int)
	if size2 != 0 {
		t.Errorf("Expected cache to be cleared after delete, got size=%d", size2)
	}
}

func TestCacheDifferentQueries(t *testing.T) {
	dir := "./test_cache_different"
	defer os.RemoveAll(dir)

	db, err := Open(DefaultConfig(dir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("users")

	// Insert test data
	for i := 0; i < 5; i++ {
		_, err := coll.InsertOne(map[string]interface{}{
			"name": string(rune('A' + i)),
			"age":  20 + i*5,
		})
		if err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}
	}

	// Different queries should have different cache keys
	filter1 := map[string]interface{}{"age": map[string]interface{}{"$gte": 25}}
	filter2 := map[string]interface{}{"age": map[string]interface{}{"$gte": 30}}
	options := &QueryOptions{Limit: 10}

	// Query 1
	results1, err := coll.FindWithOptions(filter1, options)
	if err != nil {
		t.Fatalf("Query 1 failed: %v", err)
	}

	// Query 2
	results2, err := coll.FindWithOptions(filter2, options)
	if err != nil {
		t.Fatalf("Query 2 failed: %v", err)
	}

	// Different results expected
	if len(results1) == len(results2) {
		t.Log("Warning: different queries returned same number of results (might be coincidence)")
	}

	// Cache should have 2 entries
	stats := coll.queryCache.Stats()
	size := stats["size"].(int)
	if size != 2 {
		t.Errorf("Expected cache to have 2 entries, got %d", size)
	}

	// Query 1 again - should be cache hit
	initialHits := stats["hits"].(uint64)
	results3, err := coll.FindWithOptions(filter1, options)
	if err != nil {
		t.Fatalf("Query 3 failed: %v", err)
	}

	stats2 := coll.queryCache.Stats()
	newHits := stats2["hits"].(uint64)
	if newHits != initialHits+1 {
		t.Errorf("Expected cache hit, hits went from %d to %d", initialHits, newHits)
	}

	if len(results3) != len(results1) {
		t.Errorf("Cached results different from original: %d vs %d", len(results3), len(results1))
	}
}

func TestCacheTTL(t *testing.T) {
	// This test verifies TTL works, but uses a short TTL
	// Note: The actual collection uses 5 minute TTL, but we can test the cache directly
	dir := "./test_cache_ttl"
	defer os.RemoveAll(dir)

	db, err := Open(DefaultConfig(dir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("temp")

	// Replace the cache with one that has a short TTL for testing
	// (In production, TTL is 5 minutes)
	coll.queryCache = cache.NewLRUCache(100, 100*time.Millisecond)

	// Insert and query
	_, err = coll.InsertOne(map[string]interface{}{"name": "Test"})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	filter := map[string]interface{}{"name": "Test"}
	options := &QueryOptions{Limit: 10}

	// First query
	results1, err := coll.FindWithOptions(filter, options)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results1) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results1))
	}

	// Verify cached
	stats1 := coll.queryCache.Stats()
	size1 := stats1["size"].(int)
	if size1 != 1 {
		t.Errorf("Expected 1 cache entry, got %d", size1)
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Query again - should be a cache miss due to TTL expiration
	initialMisses := stats1["misses"].(uint64)
	results2, err := coll.FindWithOptions(filter, options)
	if err != nil {
		t.Fatalf("Query after TTL failed: %v", err)
	}
	if len(results2) != 1 {
		t.Errorf("Expected 1 result after TTL, got %d", len(results2))
	}

	// Verify cache miss (TTL expired)
	stats2 := coll.queryCache.Stats()
	newMisses := stats2["misses"].(uint64)
	if newMisses != initialMisses+1 {
		t.Errorf("Expected cache miss due to TTL, misses went from %d to %d", initialMisses, newMisses)
	}
}
