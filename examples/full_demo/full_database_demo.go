package main

import (
	"fmt"
	"log"

	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/query"
)

func main() {
	fmt.Println("=== Document Database Demo ===\n")

	// Open database
	config := database.DefaultConfig("./demo_data")
	db, err := database.Open(config)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Get collection
	users := db.Collection("users")

	// Example 1: Insert documents
	fmt.Println("--- Inserting Documents ---")
	id1, _ := users.InsertOne(map[string]interface{}{
		"name":  "Alice",
		"email": "alice@example.com",
		"age":   int64(30),
		"city":  "New York",
		"tags":  []interface{}{"admin", "developer"},
	})
	fmt.Printf("Inserted Alice with ID: %s\n", id1)

	id2, _ := users.InsertOne(map[string]interface{}{
		"name":  "Bob",
		"email": "bob@example.com",
		"age":   int64(25),
		"city":  "San Francisco",
		"tags":  []interface{}{"developer"},
	})
	fmt.Printf("Inserted Bob with ID: %s\n", id2)

	users.InsertOne(map[string]interface{}{
		"name":  "Charlie",
		"email": "charlie@example.com",
		"age":   int64(35),
		"city":  "New York",
		"tags":  []interface{}{"manager"},
	})

	users.InsertOne(map[string]interface{}{
		"name":  "Diana",
		"email": "diana@example.com",
		"age":   int64(28),
		"city":  "Boston",
		"tags":  []interface{}{"developer", "designer"},
	})

	// Example 2: Simple queries
	fmt.Println("\n--- Simple Queries ---")

	// Find all users
	allUsers, _ := users.Find(map[string]interface{}{})
	fmt.Printf("Total users: %d\n", len(allUsers))

	// Find by exact match
	aliceDoc, _ := users.FindOne(map[string]interface{}{
		"name": "Alice",
	})
	if aliceDoc != nil {
		fmt.Printf("Found: %v\n", aliceDoc.ToMap())
	}

	// Example 3: Comparison operators
	fmt.Println("\n--- Comparison Operators ---")

	// Find users older than 28
	olderUsers, _ := users.Find(map[string]interface{}{
		"age": map[string]interface{}{
			"$gt": int64(28),
		},
	})
	fmt.Printf("Users older than 28: %d\n", len(olderUsers))
	for _, user := range olderUsers {
		name, _ := user.Get("name")
		age, _ := user.Get("age")
		fmt.Printf("  - %s (age: %v)\n", name, age)
	}

	// Find users between 25 and 30
	midAgeUsers, _ := users.Find(map[string]interface{}{
		"age": map[string]interface{}{
			"$gte": int64(25),
			"$lte": int64(30),
		},
	})
	fmt.Printf("Users aged 25-30: %d\n", len(midAgeUsers))

	// Example 4: Logical operators
	fmt.Println("\n--- Logical Operators ---")

	// Find users in New York OR older than 30
	orResults, _ := users.Find(map[string]interface{}{
		"$or": []interface{}{
			map[string]interface{}{"city": "New York"},
			map[string]interface{}{"age": map[string]interface{}{"$gt": int64(30)}},
		},
	})
	fmt.Printf("Users in New York OR age > 30: %d\n", len(orResults))
	for _, user := range orResults {
		name, _ := user.Get("name")
		city, _ := user.Get("city")
		age, _ := user.Get("age")
		fmt.Printf("  - %s from %s (age: %v)\n", name, city, age)
	}

	// Example 5: Query with options
	fmt.Println("\n--- Query with Options ---")

	// Find with projection (only name and email)
	options := &database.QueryOptions{
		Projection: map[string]bool{
			"name":  true,
			"email": true,
		},
		Sort: []query.SortField{
			{Field: "age", Ascending: false}, // Sort by age descending
		},
		Limit: 2,
	}
	limitedResults, _ := users.FindWithOptions(map[string]interface{}{}, options)
	fmt.Println("Top 2 oldest users (name and email only):")
	for _, user := range limitedResults {
		fmt.Printf("  %v\n", user.ToMap())
	}

	// Example 6: Update operations
	fmt.Println("\n--- Update Operations ---")

	// Update one document
	err = users.UpdateOne(
		map[string]interface{}{"name": "Alice"},
		map[string]interface{}{
			"$set": map[string]interface{}{
				"age":  int64(31),
				"city": "Los Angeles",
			},
		},
	)
	if err == nil {
		fmt.Println("Updated Alice's age and city")
		updatedAlice, _ := users.FindOne(map[string]interface{}{"name": "Alice"})
		fmt.Printf("  Alice now: %v\n", updatedAlice.ToMap())
	}

	// Increment age
	users.UpdateOne(
		map[string]interface{}{"name": "Bob"},
		map[string]interface{}{
			"$inc": map[string]interface{}{
				"age": int64(1),
			},
		},
	)
	fmt.Println("Incremented Bob's age by 1")

	// Update many
	count, _ := users.UpdateMany(
		map[string]interface{}{
			"city": "New York",
		},
		map[string]interface{}{
			"$set": map[string]interface{}{
				"timezone": "EST",
			},
		},
	)
	fmt.Printf("Added timezone to %d users in New York\n", count)

	// Example 7: Indexes
	fmt.Println("\n--- Indexes ---")

	// Create index on email (unique)
	err = users.CreateIndex("email", true)
	if err != nil {
		fmt.Printf("Index creation error: %v\n", err)
	} else {
		fmt.Println("Created unique index on email")
	}

	// Create index on city (non-unique)
	users.CreateIndex("city", false)
	fmt.Println("Created index on city")

	// List indexes
	indexes := users.ListIndexes()
	fmt.Printf("Total indexes: %d\n", len(indexes))
	for _, idx := range indexes {
		fmt.Printf("  - %s on %s (unique: %v)\n",
			idx["name"], idx["field_path"], idx["unique"])
	}

	// Example 8: Delete operations
	fmt.Println("\n--- Delete Operations ---")

	// Delete one document
	err = users.DeleteOne(map[string]interface{}{"name": "Bob"})
	if err == nil {
		fmt.Println("Deleted Bob")
	}

	// Count remaining users
	count, _ = users.Count(map[string]interface{}{})
	fmt.Printf("Remaining users: %d\n", count)

	// Delete many
	deletedCount, _ := users.DeleteMany(map[string]interface{}{
		"age": map[string]interface{}{
			"$lt": int64(30),
		},
	})
	fmt.Printf("Deleted %d users younger than 30\n", deletedCount)

	// Example 9: Multiple collections
	fmt.Println("\n--- Multiple Collections ---")

	products := db.Collection("products")
	products.InsertOne(map[string]interface{}{
		"name":  "Laptop",
		"price": 999.99,
		"stock": int64(50),
	})
	products.InsertOne(map[string]interface{}{
		"name":  "Mouse",
		"price": 29.99,
		"stock": int64(200),
	})

	collections := db.ListCollections()
	fmt.Printf("Collections in database: %v\n", collections)

	// Example 10: Database statistics
	fmt.Println("\n--- Database Statistics ---")
	stats := db.Stats()
	fmt.Printf("Database: %s\n", stats["name"])
	fmt.Printf("Total collections: %v\n", stats["collections"])
	fmt.Printf("Active transactions: %v\n", stats["active_transactions"])

	// Collection stats
	userStats := users.Stats()
	fmt.Printf("\nUsers collection:\n")
	fmt.Printf("  Document count: %v\n", userStats["count"])
	fmt.Printf("  Index count: %v\n", userStats["indexes"])

	fmt.Println("\n=== Demo Complete ===")
}
