package main

import (
	"fmt"
	"log"

	"github.com/mnohosten/laura-db/pkg/database"
)

func main() {
	fmt.Println("LauraDB Cursor Demo")
	fmt.Println("===================")

	// Open database
	config := database.DefaultConfig("./data")
	db, err := database.Open(config)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Get collection
	coll := db.Collection("products")

	// Insert sample data
	fmt.Println("\n1. Inserting sample products...")
	products := []map[string]interface{}{
		{"name": "Laptop", "price": int64(1200), "category": "Electronics", "stock": int64(15)},
		{"name": "Mouse", "price": int64(25), "category": "Electronics", "stock": int64(100)},
		{"name": "Keyboard", "price": int64(75), "category": "Electronics", "stock": int64(50)},
		{"name": "Monitor", "price": int64(350), "category": "Electronics", "stock": int64(30)},
		{"name": "Desk Chair", "price": int64(250), "category": "Furniture", "stock": int64(20)},
		{"name": "Standing Desk", "price": int64(450), "category": "Furniture", "stock": int64(15)},
		{"name": "Notebook", "price": int64(5), "category": "Stationery", "stock": int64(200)},
		{"name": "Pen Set", "price": int64(15), "category": "Stationery", "stock": int64(150)},
		{"name": "Whiteboard", "price": int64(80), "category": "Office", "stock": int64(25)},
		{"name": "Projector", "price": int64(600), "category": "Electronics", "stock": int64(10)},
	}

	for _, product := range products {
		_, err := coll.InsertOne(product)
		if err != nil {
			log.Fatalf("Failed to insert product: %v", err)
		}
	}
	fmt.Printf("Inserted %d products\n", len(products))

	// Demo 1: Basic cursor usage
	fmt.Println("\n2. Basic Cursor Usage - Iterate through all products")
	fmt.Println("-----------------------------------------------------")
	cursor, err := coll.FindCursor(map[string]interface{}{}, nil)
	if err != nil {
		log.Fatalf("Failed to create cursor: %v", err)
	}
	defer cursor.Close()

	fmt.Printf("Total documents: %d\n", cursor.Count())
	count := 0
	for cursor.HasNext() {
		doc, err := cursor.Next()
		if err != nil {
			log.Fatalf("Failed to get next document: %v", err)
		}
		name, _ := doc.Get("name")
		price, _ := doc.Get("price")
		fmt.Printf("  %d. %s - $%d\n", count+1, name, price)
		count++
	}

	// Demo 2: Batch processing
	fmt.Println("\n3. Batch Processing - Fetch products in batches of 3")
	fmt.Println("----------------------------------------------------")
	options := &database.CursorOptions{
		BatchSize: 3,
	}

	cursor2, err := coll.FindCursor(map[string]interface{}{}, options)
	if err != nil {
		log.Fatalf("Failed to create cursor: %v", err)
	}
	defer cursor2.Close()

	batchNum := 1
	for !cursor2.IsExhausted() {
		batch, err := cursor2.NextBatch()
		if err != nil {
			log.Fatalf("Failed to get next batch: %v", err)
		}
		if len(batch) == 0 {
			break
		}
		fmt.Printf("Batch %d (%d documents):\n", batchNum, len(batch))
		for _, doc := range batch {
			name, _ := doc.Get("name")
			fmt.Printf("  - %s\n", name)
		}
		batchNum++
	}

	// Demo 3: Filtered cursor
	fmt.Println("\n4. Filtered Cursor - Products priced over $100")
	fmt.Println("----------------------------------------------")
	filter := map[string]interface{}{
		"price": map[string]interface{}{
			"$gt": int64(100),
		},
	}

	cursor3, err := coll.FindCursor(filter, nil)
	if err != nil {
		log.Fatalf("Failed to create cursor: %v", err)
	}
	defer cursor3.Close()

	fmt.Printf("Found %d products priced over $100:\n", cursor3.Count())
	for cursor3.HasNext() {
		doc, err := cursor3.Next()
		if err != nil {
			log.Fatalf("Failed to get next document: %v", err)
		}
		name, _ := doc.Get("name")
		price, _ := doc.Get("price")
		category, _ := doc.Get("category")
		fmt.Printf("  - %s (%s) - $%d\n", name, category, price)
	}

	// Demo 4: Cursor with query options
	fmt.Println("\n5. Cursor with Query Options - Top 5 products by price")
	fmt.Println("------------------------------------------------------")
	queryOptions := &database.QueryOptions{
		Limit: 5,
		Projection: map[string]bool{
			"name":  true,
			"price": true,
		},
	}

	cursor4, err := coll.FindCursorWithOptions(map[string]interface{}{}, queryOptions, nil)
	if err != nil {
		log.Fatalf("Failed to create cursor: %v", err)
	}
	defer cursor4.Close()

	fmt.Printf("Top %d products (limited to 5):\n", cursor4.Count())
	position := 1
	for cursor4.HasNext() {
		doc, err := cursor4.Next()
		if err != nil {
			log.Fatalf("Failed to get next document: %v", err)
		}
		name, _ := doc.Get("name")
		price, _ := doc.Get("price")
		fmt.Printf("  %d. %s - $%d\n", position, name, price)
		position++
	}

	// Demo 5: Cursor Manager for server-side cursors
	fmt.Println("\n6. Cursor Manager - Managing multiple cursors")
	fmt.Println("--------------------------------------------")
	manager := db.CursorManager()

	// Create a cursor through the manager with a reasonable timeout
	cursor5, err := manager.CreateCursor(coll, nil, &database.CursorOptions{
		BatchSize: 2,
		Timeout:   database.DefaultCursorOptions().Timeout,
	})
	if err != nil {
		log.Fatalf("Failed to create cursor: %v", err)
	}

	fmt.Printf("Created cursor with ID: %s\n", cursor5.ID())
	fmt.Printf("Active cursors: %d\n", manager.ActiveCursors())

	// Fetch first batch
	batch, err := cursor5.NextBatch()
	if err != nil {
		log.Fatalf("Failed to get batch: %v", err)
	}
	fmt.Printf("Fetched first batch: %d documents\n", len(batch))
	fmt.Printf("Remaining: %d documents\n", cursor5.Remaining())

	// Retrieve cursor by ID
	retrieved, err := manager.GetCursor(cursor5.ID())
	if err != nil {
		log.Fatalf("Failed to retrieve cursor: %v", err)
	}
	fmt.Printf("Retrieved cursor: %s (position: %d/%d)\n",
		retrieved.ID(), retrieved.Position(), retrieved.Count())

	// Close cursor
	err = manager.CloseCursor(cursor5.ID())
	if err != nil {
		log.Fatalf("Failed to close cursor: %v", err)
	}
	fmt.Printf("Closed cursor. Active cursors: %d\n", manager.ActiveCursors())

	fmt.Println("\n✓ Cursor demo completed successfully!")
}
