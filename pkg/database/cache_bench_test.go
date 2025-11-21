package database

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/mnohosten/laura-db/pkg/cache"
)

func BenchmarkQueryWithCache(b *testing.B) {
	dir := "./bench_cache"
	defer os.RemoveAll(dir)

	db, err := Open(DefaultConfig(dir))
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("users")

	// Insert test data
	for i := 0; i < 1000; i++ {
		coll.InsertOne(map[string]interface{}{
			"name": fmt.Sprintf("User%d", i),
			"age":  20 + (i % 50),
			"city": []string{"NYC", "LA", "SF", "CHI", "BOS"}[i%5],
		})
	}

	filter := map[string]interface{}{"age": map[string]interface{}{"$gte": 30}}
	options := &QueryOptions{Limit: 100}

	// Warm up cache
	coll.FindWithOptions(filter, options)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.FindWithOptions(filter, options)
	}
}

func BenchmarkQueryWithoutCache(b *testing.B) {
	dir := "./bench_no_cache"
	defer os.RemoveAll(dir)

	db, err := Open(DefaultConfig(dir))
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("users")

	// Insert test data
	for i := 0; i < 1000; i++ {
		coll.InsertOne(map[string]interface{}{
			"name": fmt.Sprintf("User%d", i),
			"age":  20 + (i % 50),
			"city": []string{"NYC", "LA", "SF", "CHI", "BOS"}[i%5],
		})
	}

	filter := map[string]interface{}{"age": map[string]interface{}{"$gte": 30}}
	options := &QueryOptions{Limit: 100}

	// Clear cache before each iteration to simulate no cache
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.queryCache.Clear()
		coll.FindWithOptions(filter, options)
	}
}

func BenchmarkCacheHitRate(b *testing.B) {
	dir := "./bench_hit_rate"
	defer os.RemoveAll(dir)

	db, err := Open(DefaultConfig(dir))
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("products")

	// Insert test data
	for i := 0; i < 500; i++ {
		coll.InsertOne(map[string]interface{}{
			"name":     fmt.Sprintf("Product%d", i),
			"price":    100 + (i % 200),
			"category": []string{"Electronics", "Books", "Clothing", "Food"}[i%4],
		})
	}

	// Define multiple different queries (simulating real workload)
	queries := []map[string]interface{}{
		{"price": map[string]interface{}{"$gte": 150}},
		{"category": "Electronics"},
		{"price": map[string]interface{}{"$lt": 200}},
		{"category": "Books"},
	}

	options := &QueryOptions{Limit: 50}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Query with 75% hit rate (repeat some queries)
		filter := queries[i%len(queries)]
		coll.FindWithOptions(filter, options)
	}

	stats := coll.queryCache.Stats()
	b.Logf("Cache Stats: hits=%d, misses=%d, hit_rate=%v",
		stats["hits"], stats["misses"], stats["hit_rate"])
}

func BenchmarkCacheEviction(b *testing.B) {
	dir := "./bench_eviction"
	defer os.RemoveAll(dir)

	db, err := Open(DefaultConfig(dir))
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("items")

	// Use small cache to trigger evictions
	coll.queryCache = cache.NewLRUCache(10, 5*time.Minute)

	// Insert test data
	for i := 0; i < 100; i++ {
		coll.InsertOne(map[string]interface{}{
			"value": i,
		})
	}

	options := &QueryOptions{Limit: 10}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Each query is slightly different, causing evictions
		filter := map[string]interface{}{"value": map[string]interface{}{"$gte": i % 50}}
		coll.FindWithOptions(filter, options)
	}

	stats := coll.queryCache.Stats()
	b.Logf("Evictions: %d, Cache size: %d", stats["evictions"], stats["size"])
}
