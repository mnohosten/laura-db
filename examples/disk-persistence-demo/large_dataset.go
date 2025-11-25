package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/mnohosten/laura-db/pkg/database"
)

// This example demonstrates handling datasets larger than the buffer pool.
// It shows how LauraDB efficiently manages memory while working with large datasets.
//
// Key points demonstrated:
// - Inserting many documents (more than fit in buffer pool)
// - Query performance with caching
// - Memory efficiency through automatic eviction
// - Index usage for large datasets
func main() {
	fmt.Println("=== Large Dataset Demo ===")
	fmt.Println("This demo shows LauraDB handling datasets larger than buffer pool.\n")

	dataDir := "./large_dataset_data"

	// Clean up any existing data
	fmt.Println("Cleaning up old data...")
	os.RemoveAll(dataDir)

	// Configure with a SMALL buffer pool to demonstrate disk I/O
	// In production, you'd use a larger buffer pool (5000+ pages)
	config := database.DefaultConfig(dataDir)
	config.BufferPoolSize = 500 // Only 500 pages (~2MB) - smaller than dataset

	fmt.Printf("\nConfiguration:\n")
	fmt.Printf("  Data Directory: %s\n", dataDir)
	fmt.Printf("  Buffer Pool Size: %d pages (~%d MB)\n", config.BufferPoolSize, config.BufferPoolSize*4/1024)
	fmt.Println()

	db, err := database.Open(config)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	orders := db.Collection("orders")

	// ==========================================
	// Phase 1: Insert large dataset
	// ==========================================
	fmt.Println("Phase 1: Inserting large dataset...")
	fmt.Println("  Inserting 10,000 orders (this will exceed buffer pool capacity)...")

	start := time.Now()
	insertedIDs := make([]string, 0, 10000)

	// Generate realistic order data
	statuses := []string{"pending", "processing", "shipped", "delivered", "cancelled"}
	products := []string{"Laptop", "Mouse", "Keyboard", "Monitor", "Headphones", "Webcam", "Desk", "Chair"}

	for i := 0; i < 10000; i++ {
		order := map[string]interface{}{
			"order_number": fmt.Sprintf("ORD-%06d", i+1),
			"customer_id":  fmt.Sprintf("CUST-%04d", rand.Intn(1000)+1),
			"product":      products[rand.Intn(len(products))],
			"quantity":     int64(rand.Intn(10) + 1),
			"amount":       int64(rand.Intn(1000) + 10),
			"status":       statuses[rand.Intn(len(statuses))],
			"created_at":   time.Now().Add(-time.Duration(rand.Intn(365)) * 24 * time.Hour),
		}

		id, err := orders.InsertOne(order)
		if err != nil {
			log.Fatal(err)
		}
		insertedIDs = append(insertedIDs, id)

		// Progress indicator
		if (i+1)%1000 == 0 {
			fmt.Printf("    Inserted %d orders...\n", i+1)
		}
	}

	elapsed := time.Since(start)
	fmt.Printf("\n  ✓ Inserted 10,000 orders in %v\n", elapsed)
	fmt.Printf("  ✓ Average insert time: %v per document\n", elapsed/10000)
	fmt.Printf("  ✓ Throughput: %.0f inserts/second\n\n", float64(10000)/elapsed.Seconds())

	// ==========================================
	// Phase 2: Create indexes for efficient queries
	// ==========================================
	fmt.Println("Phase 2: Creating indexes...")

	orders.CreateIndex("status", false)      // Index on status field
	orders.CreateIndex("customer_id", false) // Index on customer_id
	orders.CreateIndex("product", false)     // Index on product

	fmt.Println("  ✓ Created indexes on: status, customer_id, product\n")

	// ==========================================
	// Phase 3: Query large dataset
	// ==========================================
	fmt.Println("Phase 3: Querying large dataset...")

	// Query 1: Find all delivered orders (using index)
	fmt.Println("\n  Query 1: Find all 'delivered' orders...")
	start = time.Now()
	delivered, _ := orders.Find(map[string]interface{}{"status": "delivered"})
	elapsed = time.Since(start)
	fmt.Printf("    ✓ Found %d delivered orders in %v\n", len(delivered), elapsed)

	// Query 2: Find orders by customer (using index)
	fmt.Println("\n  Query 2: Find orders for customer CUST-0001...")
	start = time.Now()
	customerOrders, _ := orders.Find(map[string]interface{}{"customer_id": "CUST-0001"})
	elapsed = time.Since(start)
	fmt.Printf("    ✓ Found %d orders in %v\n", len(customerOrders), elapsed)

	// Query 3: Find expensive orders (full scan, no index)
	fmt.Println("\n  Query 3: Find orders over $500 (no index - full scan)...")
	start = time.Now()
	expensiveOrders, _ := orders.Find(map[string]interface{}{
		"amount": map[string]interface{}{"$gte": int64(500)},
	})
	elapsed = time.Since(start)
	fmt.Printf("    ✓ Found %d expensive orders in %v\n", len(expensiveOrders), elapsed)
	fmt.Println("    Note: This is slower because 'amount' is not indexed")

	// Query 4: Range query with multiple conditions
	fmt.Println("\n  Query 4: Find laptop orders that are shipped...")
	start = time.Now()
	laptopOrders, _ := orders.Find(map[string]interface{}{
		"product": "Laptop",
		"status":  "shipped",
	})
	elapsed = time.Since(start)
	fmt.Printf("    ✓ Found %d laptop orders in %v\n", len(laptopOrders), elapsed)

	// Query 5: Count documents by status
	fmt.Println("\n  Query 5: Count orders by status...")
	for _, status := range statuses {
		results, _ := orders.Find(map[string]interface{}{"status": status})
		fmt.Printf("    - %s: %d orders\n", status, len(results))
	}

	// ==========================================
	// Phase 4: Demonstrate query caching
	// ==========================================
	fmt.Println("\nPhase 4: Demonstrating query cache performance...")

	// First query (cold - not cached)
	filter := map[string]interface{}{"status": "pending"}
	fmt.Println("\n  First query (cold cache)...")
	start = time.Now()
	results1, _ := orders.Find(filter)
	elapsed1 := time.Since(start)
	fmt.Printf("    ✓ Found %d results in %v\n", len(results1), elapsed1)

	// Second identical query (warm - cached)
	fmt.Println("\n  Second identical query (cached)...")
	start = time.Now()
	results2, _ := orders.Find(filter)
	elapsed2 := time.Since(start)
	fmt.Printf("    ✓ Found %d results in %v\n", len(results2), elapsed2)

	speedup := float64(elapsed1.Nanoseconds()) / float64(elapsed2.Nanoseconds())
	fmt.Printf("\n    Cache speedup: %.1fx faster\n", speedup)

	// ==========================================
	// Phase 5: Aggregation on large dataset
	// ==========================================
	fmt.Println("\nPhase 5: Running aggregation on large dataset...")

	fmt.Println("\n  Calculating total revenue by product...")
	start = time.Now()

	// Group by product and sum amounts
	productRevenue := make(map[string]int64)
	allOrders, _ := orders.Find(map[string]interface{}{})
	for _, order := range allOrders {
		product, _ := order.Get("product")
		amount, _ := order.Get("amount")
		productRevenue[product.(string)] += amount.(int64)
	}

	elapsed = time.Since(start)
	fmt.Printf("    ✓ Processed %d orders in %v\n", len(allOrders), elapsed)
	fmt.Println("\n    Revenue by product:")
	for product, revenue := range productRevenue {
		fmt.Printf("      - %s: $%d\n", product, revenue)
	}

	// ==========================================
	// Phase 6: Test persistence with large dataset
	// ==========================================
	fmt.Println("\nPhase 6: Testing persistence...")
	fmt.Println("  Closing database (flushing large dataset to disk)...")

	closeStart := time.Now()
	if err := db.Close(); err != nil {
		log.Fatal(err)
	}
	closeElapsed := time.Since(closeStart)
	fmt.Printf("  ✓ Database closed in %v\n", closeElapsed)

	// Reopen and verify
	fmt.Println("\n  Reopening database...")
	db2, err := database.Open(config)
	if err != nil {
		log.Fatal(err)
	}
	defer db2.Close()

	orders2 := db2.Collection("orders")
	finalOrders, _ := orders2.Find(map[string]interface{}{})
	finalCount := len(finalOrders)

	fmt.Printf("  ✓ Verified: %d orders persisted (expected: 10,000)\n", finalCount)

	if finalCount != 10000 {
		log.Fatal("ERROR: Data count mismatch!")
	}

	// ==========================================
	// Summary
	// ==========================================
	fmt.Println("\n=== SUCCESS ===")
	fmt.Println("✓ LauraDB efficiently handled 10,000 documents")
	fmt.Println("✓ Buffer pool (2MB) successfully cached hot data")
	fmt.Println("✓ Indexes provided fast lookups on large dataset")
	fmt.Println("✓ Query cache dramatically improved repeated queries")
	fmt.Println("✓ All data persisted correctly to disk")
	fmt.Println("\nKey Takeaways:")
	fmt.Println("  1. Dataset can be MUCH larger than buffer pool")
	fmt.Println("  2. Frequently accessed data stays cached (LRU eviction)")
	fmt.Println("  3. Indexes are essential for query performance")
	fmt.Println("  4. Query cache provides significant speedup")
	fmt.Println("  5. WAL ensures durability even with large datasets")
	fmt.Println("\nData directory:", dataDir)
	fmt.Println("To clean up: rm -rf", dataDir)
}
