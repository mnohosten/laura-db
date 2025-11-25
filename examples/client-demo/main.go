package main

import (
	"fmt"
	"log"
	"time"

	"github.com/mnohosten/laura-db/pkg/client"
)

func main() {
	fmt.Println("LauraDB Go Client Library Demo")
	fmt.Println("================================\n")

	// NOTE: This demo requires a running LauraDB server
	// Start the server with: ./bin/laura-server -port 8080

	// Create a client with default configuration
	fmt.Println("1. Connecting to LauraDB server...")
	c := client.NewDefaultClient()
	defer c.Close()

	// Check server health
	fmt.Println("\n2. Checking server health...")
	health, err := c.Health()
	if err != nil {
		log.Fatalf("Health check failed: %v", err)
	}
	fmt.Printf("   Status: %s\n", health.Status)
	fmt.Printf("   Uptime: %s\n", health.Uptime)

	// Create a collection
	collName := "demo_users"
	fmt.Printf("\n3. Creating collection '%s'...\n", collName)
	err = c.CreateCollection(collName)
	if err != nil {
		log.Printf("   Warning: %v (collection may already exist)\n", err)
	}

	// Get collection handle
	users := c.Collection(collName)

	// Insert documents
	fmt.Println("\n4. Inserting documents...")

	alice := map[string]interface{}{
		"name":   "Alice Johnson",
		"email":  "alice@example.com",
		"age":    int64(30),
		"city":   "New York",
		"status": "active",
	}
	aliceID, err := users.InsertOne(alice)
	if err != nil {
		log.Fatalf("Insert failed: %v", err)
	}
	fmt.Printf("   Inserted Alice with ID: %s\n", aliceID)

	bob := map[string]interface{}{
		"name":   "Bob Smith",
		"email":  "bob@example.com",
		"age":    int64(25),
		"city":   "San Francisco",
		"status": "active",
	}
	bobID, err := users.InsertOne(bob)
	if err != nil {
		log.Fatalf("Insert failed: %v", err)
	}
	fmt.Printf("   Inserted Bob with ID: %s\n", bobID)

	charlie := map[string]interface{}{
		"name":   "Charlie Brown",
		"email":  "charlie@example.com",
		"age":    int64(35),
		"city":   "New York",
		"status": "inactive",
	}
	charlieID, err := users.InsertOne(charlie)
	if err != nil {
		log.Fatalf("Insert failed: %v", err)
	}
	fmt.Printf("   Inserted Charlie with ID: %s\n", charlieID)

	// Find a document by ID
	fmt.Printf("\n5. Finding document by ID: %s\n", aliceID)
	doc, err := users.FindOne(aliceID)
	if err != nil {
		log.Fatalf("FindOne failed: %v", err)
	}
	fmt.Printf("   Found: %v\n", doc)

	// Count documents
	fmt.Println("\n6. Counting all documents...")
	count, err := users.Count(nil)
	if err != nil {
		log.Fatalf("Count failed: %v", err)
	}
	fmt.Printf("   Total documents: %d\n", count)

	// Search with filter
	fmt.Println("\n7. Searching for users in New York...")
	filter := map[string]interface{}{
		"city": "New York",
	}
	results, err := users.Find(filter)
	if err != nil {
		log.Fatalf("Find failed: %v", err)
	}
	fmt.Printf("   Found %d user(s):\n", len(results))
	for _, r := range results {
		fmt.Printf("   - %s (%s)\n", r["name"], r["city"])
	}

	// Search with complex filter
	fmt.Println("\n8. Searching for active users over 25...")
	filter = map[string]interface{}{
		"age": map[string]interface{}{
			"$gt": int64(25),
		},
		"status": "active",
	}
	results, err = users.Find(filter)
	if err != nil {
		log.Fatalf("Find failed: %v", err)
	}
	fmt.Printf("   Found %d user(s):\n", len(results))
	for _, r := range results {
		fmt.Printf("   - %s (age: %.0f, status: %s)\n", r["name"], r["age"], r["status"])
	}

	// Update a document
	fmt.Printf("\n9. Updating Bob's age to 26...\n")
	update := map[string]interface{}{
		"$set": map[string]interface{}{
			"age": int64(26),
		},
	}
	err = users.UpdateOne(bobID, update)
	if err != nil {
		log.Fatalf("UpdateOne failed: %v", err)
	}
	fmt.Println("   Updated successfully")

	// Verify update
	doc, _ = users.FindOne(bobID)
	fmt.Printf("   Bob's new age: %.0f\n", doc["age"])

	// Create an index
	fmt.Println("\n10. Creating index on 'email' field...")
	err = users.CreateBTreeIndex("email_idx", "email", true)
	if err != nil {
		log.Printf("   Warning: %v (index may already exist)\n", err)
	} else {
		fmt.Println("   Index created successfully")
	}

	// List indexes
	fmt.Println("\n11. Listing all indexes...")
	indexes, err := users.ListIndexes()
	if err != nil {
		log.Fatalf("ListIndexes failed: %v", err)
	}
	fmt.Printf("   Found %d index(es):\n", len(indexes))
	for _, idx := range indexes {
		fmt.Printf("   - %s (type: %s, unique: %v)\n", idx.Name, idx.Type, idx.Unique)
	}

	// Aggregation pipeline
	fmt.Println("\n12. Running aggregation pipeline...")
	fmt.Println("    Grouping users by city and counting...")

	pipeline := client.NewPipeline().
		Group("$city", map[string]interface{}{
			"count":  client.Count(),
			"avgAge": client.Avg("age"),
		}).
		Sort(map[string]interface{}{"count": -1}).
		Build()

	aggResults, err := users.Aggregate(pipeline)
	if err != nil {
		log.Fatalf("Aggregate failed: %v", err)
	}
	fmt.Printf("   Results:\n")
	for _, r := range aggResults {
		fmt.Printf("   - City: %s, Count: %.0f, Avg Age: %.1f\n",
			r["_id"], r["count"], r["avgAge"])
	}

	// Bulk operations
	fmt.Println("\n13. Performing bulk operations...")
	bulkOps := []client.BulkOperation{
		{
			Operation: "insert",
			Document: map[string]interface{}{
				"name":   "David Lee",
				"email":  "david@example.com",
				"age":    int64(28),
				"city":   "Boston",
				"status": "active",
			},
		},
		{
			Operation: "insert",
			Document: map[string]interface{}{
				"name":   "Eve Wilson",
				"email":  "eve@example.com",
				"age":    int64(32),
				"city":   "Seattle",
				"status": "active",
			},
		},
	}

	bulkResult, err := users.Bulk(bulkOps)
	if err != nil {
		log.Fatalf("Bulk failed: %v", err)
	}
	fmt.Printf("   Inserted: %d, Updated: %d, Deleted: %d, Failed: %d\n",
		bulkResult.Inserted, bulkResult.Updated, bulkResult.Deleted, bulkResult.Failed)

	// Collection statistics
	fmt.Println("\n14. Getting collection statistics...")
	stats, err := users.Stats()
	if err != nil {
		log.Fatalf("Stats failed: %v", err)
	}
	fmt.Printf("   Collection: %s\n", stats.Name)
	fmt.Printf("   Documents: %d\n", stats.Count)
	fmt.Printf("   Indexes: %d\n", stats.Indexes)

	// Database statistics
	fmt.Println("\n15. Getting database statistics...")
	dbStats, err := c.Stats()
	if err != nil {
		log.Fatalf("Stats failed: %v", err)
	}
	fmt.Printf("   Database: %s\n", dbStats.Name)
	fmt.Printf("   Collections: %d\n", dbStats.Collections)
	fmt.Printf("   Active Transactions: %d\n", dbStats.ActiveTransactions)
	fmt.Printf("   Buffer Pool Hit Rate: %.2f%%\n", dbStats.StorageStats.BufferPool.HitRate*100)

	// List all collections
	fmt.Println("\n16. Listing all collections...")
	collections, err := c.ListCollections()
	if err != nil {
		log.Fatalf("ListCollections failed: %v", err)
	}
	fmt.Printf("   Collections: %v\n", collections)

	// Clean up - delete a document
	fmt.Printf("\n17. Deleting Charlie's document...\n")
	err = users.DeleteOne(charlieID)
	if err != nil {
		log.Fatalf("DeleteOne failed: %v", err)
	}
	fmt.Println("   Deleted successfully")

	// Verify deletion
	count, _ = users.Count(nil)
	fmt.Printf("   Remaining documents: %d\n", count)

	fmt.Println("\n18. Demo completed successfully!")
	fmt.Println("\nTo clean up, you can drop the collection:")
	fmt.Printf("   users.Drop()\n")
}

// Demonstration of custom client configuration
func exampleCustomConfig() {
	// Create a client with custom configuration
	config := &client.Config{
		Host:            "example.com",
		Port:            9090,
		Timeout:         10 * time.Second,
		MaxIdleConns:    20,
		MaxConnsPerHost: 20,
	}

	c := client.NewClient(config)
	defer c.Close()

	// Use the client...
	_ = c
}
