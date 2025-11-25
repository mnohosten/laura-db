package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mnohosten/laura-db/pkg/database"
)

// This example demonstrates performance tuning techniques for disk-based storage.
// It compares different configurations and shows how to optimize for your workload.
//
// Topics covered:
// - Buffer pool sizing impact
// - Index usage vs full scans
// - Batch operations vs individual operations
// - Query optimization with Explain()
// - Memory vs disk trade-offs
func main() {
	fmt.Println("=== Performance Tuning Demo ===")
	fmt.Println("This demo shows how to optimize LauraDB for your workload.\n")

	// ==========================================
	// Part 1: Buffer Pool Sizing
	// ==========================================
	fmt.Println("PART 1: Buffer Pool Sizing Impact")
	fmt.Println("==================================\n")

	testBufferPoolSizes := []int{100, 500, 1000, 5000}

	for _, bufferSize := range testBufferPoolSizes {
		dataDir := fmt.Sprintf("./perf_test_%d", bufferSize)
		os.RemoveAll(dataDir)

		config := database.DefaultConfig(dataDir)
		config.BufferPoolSize = bufferSize

		fmt.Printf("Testing with BufferPoolSize = %d pages (~%d MB)\n", bufferSize, bufferSize*4/1024)

		db, err := database.Open(config)
		if err != nil {
			log.Fatal(err)
		}

		coll := db.Collection("test")

		// Insert 1000 documents
		start := time.Now()
		for i := 0; i < 1000; i++ {
			coll.InsertOne(map[string]interface{}{
				"id":    int64(i),
				"name":  fmt.Sprintf("Document %d", i),
				"value": int64(i * 100),
			})
		}
		insertTime := time.Since(start)

		// Query all documents
		start = time.Now()
		results, _ := coll.Find(map[string]interface{}{})
		queryTime := time.Since(start)

		fmt.Printf("  Insert time: %v (%v per doc)\n", insertTime, insertTime/1000)
		fmt.Printf("  Query time:  %v (found %d docs)\n", queryTime, len(results))

		db.Close()
		os.RemoveAll(dataDir)
		fmt.Println()
	}

	fmt.Println("Key Insight: Larger buffer pool = faster queries, but more memory usage")
	fmt.Println("Recommendation: Start with 1000-2000 pages, increase if needed\n")

	// ==========================================
	// Part 2: Index Usage
	// ==========================================
	fmt.Println("\nPART 2: Index Usage Impact")
	fmt.Println("==========================\n")

	dataDir := "./perf_index_test"
	os.RemoveAll(dataDir)

	config := database.DefaultConfig(dataDir)
	config.BufferPoolSize = 1000
	db, err := database.Open(config)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	users := db.Collection("users")

	// Insert test data
	fmt.Println("Inserting 5,000 users...")
	for i := 0; i < 5000; i++ {
		users.InsertOne(map[string]interface{}{
			"user_id":  int64(i),
			"username": fmt.Sprintf("user%d", i),
			"email":    fmt.Sprintf("user%d@example.com", i),
			"age":      int64(20 + (i % 50)),
			"city":     []string{"NYC", "SF", "LA", "Boston", "Chicago"}[i%5],
		})
	}
	fmt.Println("✓ Data inserted\n")

	// Test 1: Query WITHOUT index
	fmt.Println("Test 1: Query by email WITHOUT index (full scan)...")
	start := time.Now()
	result1, _ := users.FindOne(map[string]interface{}{"email": "user2500@example.com"})
	time1 := time.Since(start)
	fmt.Printf("  Time: %v\n", time1)
	if result1 != nil {
		username, _ := result1.Get("username")
		fmt.Printf("  Found user: %v\n", username)
	}

	// Explain query plan
	fmt.Println("\n  Query Plan (Explain):")
	plan := users.Explain(map[string]interface{}{"email": "user2500@example.com"})
	if indexName, ok := plan["indexName"]; ok && indexName != nil {
		fmt.Printf("    Using index: %v\n", indexName)
	} else {
		fmt.Println("    Strategy: FULL COLLECTION SCAN (slow!)")
	}

	// Test 2: Create index and query WITH index
	fmt.Println("\n  Creating index on 'email' field...")
	users.CreateIndex("email", true) // unique index
	fmt.Println("  ✓ Index created\n")

	fmt.Println("Test 2: Query by email WITH index...")
	start = time.Now()
	result2, _ := users.FindOne(map[string]interface{}{"email": "user2500@example.com"})
	time2 := time.Since(start)
	fmt.Printf("  Time: %v\n", time2)
	if result2 != nil {
		username, _ := result2.Get("username")
		fmt.Printf("  Found user: %v\n", username)
	}

	// Explain query plan with index
	fmt.Println("\n  Query Plan (Explain):")
	plan2 := users.Explain(map[string]interface{}{"email": "user2500@example.com"})
	if indexName, ok := plan2["indexName"]; ok && indexName != nil {
		fmt.Printf("    Using index: %v\n", indexName)
		fmt.Println("    Strategy: INDEX LOOKUP (fast!)")
	}

	speedup := float64(time1.Nanoseconds()) / float64(time2.Nanoseconds())
	fmt.Printf("\n  ⚡ Speedup with index: %.1fx faster\n\n", speedup)

	fmt.Println("Key Insight: Indexes dramatically improve query performance")
	fmt.Println("Recommendation: Create indexes on frequently queried fields\n")

	// ==========================================
	// Part 3: Batch Operations
	// ==========================================
	fmt.Println("\nPART 3: Batch Operations")
	fmt.Println("========================\n")

	orders := db.Collection("orders")

	// Test 1: Individual inserts
	fmt.Println("Test 1: Inserting 1,000 documents individually...")
	start = time.Now()
	for i := 0; i < 1000; i++ {
		orders.InsertOne(map[string]interface{}{
			"order_id": int64(i),
			"amount":   int64(100 + i),
		})
	}
	individualTime := time.Since(start)
	fmt.Printf("  Time: %v\n", individualTime)
	fmt.Printf("  Average: %v per insert\n", individualTime/1000)

	// Clear collection
	allOrders, _ := orders.Find(map[string]interface{}{})
	for _, order := range allOrders {
		id, _ := order.Get("_id")
		orders.DeleteOne(map[string]interface{}{"_id": id})
	}

	// Test 2: Batch insert
	fmt.Println("\nTest 2: Inserting 1,000 documents in batch...")
	docs := make([]map[string]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		docs[i] = map[string]interface{}{
			"order_id": int64(i),
			"amount":   int64(100 + i),
		}
	}

	start = time.Now()
	orders.InsertMany(docs)
	batchTime := time.Since(start)
	fmt.Printf("  Time: %v\n", batchTime)
	fmt.Printf("  Average: %v per insert\n", batchTime/1000)

	speedup = float64(individualTime.Nanoseconds()) / float64(batchTime.Nanoseconds())
	fmt.Printf("\n  ⚡ Batch insert speedup: %.1fx faster\n\n", speedup)

	fmt.Println("Key Insight: Batch operations reduce overhead")
	fmt.Println("Recommendation: Use InsertMany() for bulk inserts\n")

	// ==========================================
	// Part 4: Query Optimization
	// ==========================================
	fmt.Println("\nPART 4: Query Optimization")
	fmt.Println("==========================\n")

	products := db.Collection("products")

	// Insert products with multiple fields
	fmt.Println("Inserting 1,000 products...")
	for i := 0; i < 1000; i++ {
		products.InsertOne(map[string]interface{}{
			"product_id": int64(i),
			"name":       fmt.Sprintf("Product %d", i),
			"price":      int64(10 + (i % 100)),
			"category":   []string{"Electronics", "Books", "Clothing", "Food", "Toys"}[i%5],
			"rating":     float64(1 + (i % 5)),
			"stock":      int64(i % 100),
		})
	}

	// Create compound index
	products.CreateIndex("category", false)
	products.CreateIndex("price", false)
	fmt.Println("✓ Created indexes on category and price\n")

	// Test different query patterns
	queries := []struct {
		name   string
		filter map[string]interface{}
	}{
		{
			name:   "Simple equality (indexed)",
			filter: map[string]interface{}{"category": "Electronics"},
		},
		{
			name: "Range query (indexed)",
			filter: map[string]interface{}{
				"price": map[string]interface{}{
					"$gte": int64(50),
					"$lte": int64(75),
				},
			},
		},
		{
			name: "Complex query (multiple conditions)",
			filter: map[string]interface{}{
				"category": "Electronics",
				"price": map[string]interface{}{
					"$gte": int64(50),
				},
			},
		},
	}

	for _, q := range queries {
		fmt.Printf("Query: %s\n", q.name)

		// First run (cold)
		start = time.Now()
		results, _ := products.Find(q.filter)
		coldTime := time.Since(start)
		fmt.Printf("  First run (cold):  %v - found %d results\n", coldTime, len(results))

		// Second run (cached)
		start = time.Now()
		results, _ = products.Find(q.filter)
		warmTime := time.Since(start)
		fmt.Printf("  Second run (warm): %v - found %d results\n", warmTime, len(results))

		// Show query plan
		plan := products.Explain(q.filter)
		if indexName, ok := plan["indexName"]; ok && indexName != nil {
			fmt.Printf("  Index used: %v\n", indexName)
		} else {
			fmt.Println("  Index used: none (full scan)")
		}

		fmt.Println()
	}

	// ==========================================
	// Part 5: Memory Monitoring
	// ==========================================
	fmt.Println("\nPART 5: Configuration Recommendations")
	fmt.Println("======================================\n")

	fmt.Println("Buffer Pool Sizing Guidelines:")
	fmt.Println("  - Small dataset (<10K docs):     100-500 pages   (~0.4-2 MB)")
	fmt.Println("  - Medium dataset (10K-100K):     1000-2000 pages (~4-8 MB)")
	fmt.Println("  - Large dataset (100K-1M):       5000-10000 pages (~20-40 MB)")
	fmt.Println("  - Very large dataset (>1M):      10000+ pages    (~40+ MB)")
	fmt.Println()

	fmt.Println("Index Strategy:")
	fmt.Println("  ✓ Always create indexes on frequently queried fields")
	fmt.Println("  ✓ Use Explain() to verify index usage")
	fmt.Println("  ✓ Create compound indexes for multi-field queries")
	fmt.Println("  ✓ Consider unique indexes for unique fields (faster lookups)")
	fmt.Println()

	fmt.Println("Query Optimization:")
	fmt.Println("  ✓ Use projections to fetch only needed fields")
	fmt.Println("  ✓ Add Limit to queries when you don't need all results")
	fmt.Println("  ✓ Leverage query cache (5-minute TTL by default)")
	fmt.Println("  ✓ Use batch operations (InsertMany, UpdateMany) when possible")
	fmt.Println()

	fmt.Println("Example Optimal Configuration:")
	fmt.Println("  config := database.DefaultConfig(\"./data\")")
	fmt.Println("  config.BufferPoolSize = 5000  // 20MB buffer pool")
	fmt.Println()

	// ==========================================
	// Summary
	// ==========================================
	fmt.Println("\n=== Performance Tuning Summary ===")
	fmt.Println("✓ Larger buffer pool = better performance (more memory)")
	fmt.Println("✓ Indexes provide 10-100x speedup for lookups")
	fmt.Println("✓ Batch operations are significantly faster")
	fmt.Println("✓ Query cache provides major speedup for repeated queries")
	fmt.Println("✓ Use Explain() to verify your queries use indexes")
	fmt.Println("\nNext Steps:")
	fmt.Println("  1. Profile your workload to understand query patterns")
	fmt.Println("  2. Create indexes on frequently queried fields")
	fmt.Println("  3. Adjust buffer pool size based on dataset size")
	fmt.Println("  4. Monitor query performance with Explain()")
	fmt.Println("  5. Use batch operations for bulk data loading")
	fmt.Println("\nTo clean up test data:")
	fmt.Println("  rm -rf ./perf_*")
	fmt.Println("  rm -rf", dataDir)
}
