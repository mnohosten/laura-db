package main

import (
	"fmt"
	"log"
	"os"

	"github.com/mnohosten/laura-db/pkg/database"
)

// This example demonstrates that data persists across database restarts.
// It performs the following steps:
// 1. Creates a database and inserts documents
// 2. Closes the database
// 3. Reopens the database
// 4. Verifies that all data is still present
func main() {
	fmt.Println("=== Disk Persistence Demo ===")
	fmt.Println("This demo shows that data survives database restarts.\n")

	dataDir := "./persistence_demo_data"

	// Clean up any existing data from previous runs
	fmt.Println("Step 1: Cleaning up old data...")
	os.RemoveAll(dataDir)

	// ==========================================
	// Phase 1: Create database and insert data
	// ==========================================
	fmt.Println("\nPhase 1: Creating database and inserting documents...")

	config := database.DefaultConfig(dataDir)
	db, err := database.Open(config)
	if err != nil {
		log.Fatal(err)
	}

	users := db.Collection("users")
	products := db.Collection("products")

	// Insert users
	fmt.Println("  Inserting 5 users...")
	userIDs := make([]string, 0)

	id1, _ := users.InsertOne(map[string]interface{}{
		"name":  "Alice Johnson",
		"email": "alice@example.com",
		"age":   int64(30),
		"role":  "admin",
	})
	userIDs = append(userIDs, id1)

	id2, _ := users.InsertOne(map[string]interface{}{
		"name":  "Bob Smith",
		"email": "bob@example.com",
		"age":   int64(25),
		"role":  "user",
	})
	userIDs = append(userIDs, id2)

	id3, _ := users.InsertOne(map[string]interface{}{
		"name":  "Charlie Brown",
		"email": "charlie@example.com",
		"age":   int64(35),
		"role":  "user",
	})
	userIDs = append(userIDs, id3)

	id4, _ := users.InsertOne(map[string]interface{}{
		"name":  "Diana Prince",
		"email": "diana@example.com",
		"age":   int64(28),
		"role":  "moderator",
	})
	userIDs = append(userIDs, id4)

	id5, _ := users.InsertOne(map[string]interface{}{
		"name":  "Eve Anderson",
		"email": "eve@example.com",
		"age":   int64(32),
		"role":  "user",
	})
	userIDs = append(userIDs, id5)

	fmt.Printf("  ✓ Inserted 5 users with IDs: %v\n", userIDs)

	// Insert products
	fmt.Println("  Inserting 3 products...")
	productIDs := make([]string, 0)

	pid1, _ := products.InsertOne(map[string]interface{}{
		"name":     "Laptop",
		"price":    int64(999),
		"category": "electronics",
		"stock":    int64(50),
	})
	productIDs = append(productIDs, pid1)

	pid2, _ := products.InsertOne(map[string]interface{}{
		"name":     "Mouse",
		"price":    int64(29),
		"category": "electronics",
		"stock":    int64(200),
	})
	productIDs = append(productIDs, pid2)

	pid3, _ := products.InsertOne(map[string]interface{}{
		"name":     "Desk",
		"price":    int64(299),
		"category": "furniture",
		"stock":    int64(25),
	})
	productIDs = append(productIDs, pid3)

	fmt.Printf("  ✓ Inserted 3 products with IDs: %v\n", productIDs)

	// Verify data before closing
	allUsersBefore, _ := users.Find(map[string]interface{}{})
	allProductsBefore, _ := products.Find(map[string]interface{}{})
	fmt.Printf("\n  Data summary before close:\n")
	fmt.Printf("    Users: %d\n", len(allUsersBefore))
	fmt.Printf("    Products: %d\n", len(allProductsBefore))

	// Close database (flushes all data to disk)
	fmt.Println("\nPhase 2: Closing database (flushing to disk)...")
	if err := db.Close(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("  ✓ Database closed successfully")
	fmt.Println("  ✓ All data written to disk at:", dataDir)
	fmt.Println("    Files created: data.db, wal.log, collections/")

	// ==========================================
	// Phase 3: Reopen database and verify data
	// ==========================================
	fmt.Println("\nPhase 3: Reopening database from disk...")

	// Reopen the same database
	config2 := database.DefaultConfig(dataDir)
	db2, err := database.Open(config2)
	if err != nil {
		log.Fatal(err)
	}
	defer db2.Close()

	fmt.Println("  ✓ Database reopened successfully")

	// Get collections again
	users2 := db2.Collection("users")
	products2 := db2.Collection("products")

	// Verify all data is still there
	fmt.Println("\n  Verifying data integrity...")

	allUsers, _ := users2.Find(map[string]interface{}{})
	allProducts, _ := products2.Find(map[string]interface{}{})

	fmt.Printf("\n  Data summary after reopen:\n")
	fmt.Printf("    Users: %d (expected: 5)\n", len(allUsers))
	fmt.Printf("    Products: %d (expected: 3)\n", len(allProducts))

	if len(allUsers) != 5 || len(allProducts) != 3 {
		log.Fatal("ERROR: Data count mismatch!")
	}

	// Verify specific documents by ID
	fmt.Println("\n  Verifying specific documents by ID...")
	for i, id := range userIDs {
		doc, err := users2.FindOne(map[string]interface{}{"_id": id})
		if err != nil || doc == nil {
			log.Fatalf("ERROR: User with ID %s not found!", id)
		}
		name, _ := doc.Get("name")
		fmt.Printf("    ✓ User %d: %s (ID: %s)\n", i+1, name, id)
	}

	for i, id := range productIDs {
		doc, err := products2.FindOne(map[string]interface{}{"_id": id})
		if err != nil || doc == nil {
			log.Fatalf("ERROR: Product with ID %s not found!", id)
		}
		name, _ := doc.Get("name")
		fmt.Printf("    ✓ Product %d: %s (ID: %s)\n", i+1, name, id)
	}

	// Perform queries to verify data integrity
	fmt.Println("\n  Performing queries on persisted data...")

	// Query 1: Find admins
	admins, _ := users2.Find(map[string]interface{}{"role": "admin"})
	fmt.Printf("    ✓ Found %d admin(s)\n", len(admins))

	// Query 2: Find electronics
	electronics, _ := products2.Find(map[string]interface{}{"category": "electronics"})
	fmt.Printf("    ✓ Found %d electronic product(s)\n", len(electronics))

	// Query 3: Users over 30
	over30, _ := users2.Find(map[string]interface{}{
		"age": map[string]interface{}{"$gte": int64(30)},
	})
	fmt.Printf("    ✓ Found %d user(s) aged 30 or older\n", len(over30))

	// ==========================================
	// Phase 4: Modify data and verify again
	// ==========================================
	fmt.Println("\nPhase 4: Modifying data...")

	// Update a user
	err = users2.UpdateOne(
		map[string]interface{}{"name": "Alice Johnson"},
		map[string]interface{}{
			"$set": map[string]interface{}{
				"age":        int64(31),
				"last_login": "2025-01-15",
			},
		},
	)
	if err == nil {
		fmt.Printf("    ✓ Updated user successfully\n")
	}

	// Delete a product
	err = products2.DeleteOne(map[string]interface{}{"name": "Mouse"})
	if err == nil {
		fmt.Printf("    ✓ Deleted product successfully\n")
	}

	// Insert a new document
	newID, _ := users2.InsertOne(map[string]interface{}{
		"name":  "Frank Wilson",
		"email": "frank@example.com",
		"age":   int64(40),
		"role":  "user",
	})
	fmt.Printf("    ✓ Inserted new user: %s\n", newID)

	fmt.Println("\n=== SUCCESS ===")
	fmt.Println("✓ All data persisted correctly across database restart!")
	fmt.Println("✓ CRUD operations work seamlessly with disk storage")
	fmt.Println("✓ Data directory:", dataDir)
	fmt.Println("\nYou can inspect the files in the data directory:")
	fmt.Println("  - data.db: Main database file (contains all documents)")
	fmt.Println("  - wal.log: Write-ahead log (ensures durability)")
	fmt.Println("  - collections/: Collection metadata")
	fmt.Println("\nTo clean up: rm -rf", dataDir)
}
