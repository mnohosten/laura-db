package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mnohosten/laura-db/pkg/audit"
	"github.com/mnohosten/laura-db/pkg/database"
)

func main() {
	fmt.Println("=== LauraDB Audit Logging Demo ===\n")

	// Clean up any existing data
	os.RemoveAll("./audit_demo_data")
	defer os.RemoveAll("./audit_demo_data")

	// Demo 1: Basic audit logging to stdout
	fmt.Println("Demo 1: Basic Audit Logging to stdout")
	fmt.Println("---------------------------------------")
	demo1BasicLogging()
	fmt.Println()

	// Demo 2: File-based audit logging
	fmt.Println("\nDemo 2: File-Based Audit Logging")
	fmt.Println("----------------------------------")
	demo2FileLogging()
	fmt.Println()

	// Demo 3: Filtering operations
	fmt.Println("\nDemo 3: Filtering Operations")
	fmt.Println("-----------------------------")
	demo3FilteringOperations()
	fmt.Println()

	// Demo 4: Severity filtering
	fmt.Println("\nDemo 4: Severity Filtering (Errors Only)")
	fmt.Println("-----------------------------------------")
	demo4SeverityFiltering()
	fmt.Println()

	// Demo 5: Text format logging
	fmt.Println("\nDemo 5: Human-Readable Text Format")
	fmt.Println("-----------------------------------")
	demo5TextFormat()
	fmt.Println()

	fmt.Println("\n=== Demo Complete ===")
}

func demo1BasicLogging() {
	// Create audit configuration with JSON format to stdout
	auditConfig := &audit.Config{
		Enabled:          true,
		OutputWriter:     os.Stdout,
		Format:           "json",
		MinSeverity:      audit.SeverityInfo,
		IncludeQueryData: true,
		MaxFieldSize:     500,
	}

	// Create database with audit logging
	dbConfig := &database.Config{
		DataDir:        "./audit_demo_data/demo1",
		BufferPoolSize: 100,
		AuditConfig:    auditConfig,
	}

	db, err := database.Open(dbConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create collection
	coll, err := db.CreateCollection("users")
	if err != nil {
		log.Fatal(err)
	}

	// Insert document
	id, err := coll.InsertOne(map[string]interface{}{
		"name":  "Alice Johnson",
		"email": "alice@example.com",
		"age":   int64(28),
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Inserted document with ID: %s\n", id)

	// Find documents
	docs, err := coll.Find(map[string]interface{}{
		"age": map[string]interface{}{"$gte": int64(25)},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d documents\n", len(docs))

	// Update document
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Alice Johnson"},
		map[string]interface{}{"$set": map[string]interface{}{"age": int64(29)}},
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Updated document")
}

func demo2FileLogging() {
	// Create file-based audit logger
	auditLogFile := "./audit_demo_data/audit.log"
	os.MkdirAll("./audit_demo_data", 0755)

	auditConfig := &audit.Config{
		Enabled:          true,
		Format:           "json",
		MinSeverity:      audit.SeverityInfo,
		IncludeQueryData: true,
	}

	// Open file for audit logging
	file, err := os.OpenFile(auditLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	auditConfig.OutputWriter = file

	// Create database with file-based audit logging
	dbConfig := &database.Config{
		DataDir:        "./audit_demo_data/demo2",
		BufferPoolSize: 100,
		AuditConfig:    auditConfig,
	}

	db, err := database.Open(dbConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	coll := db.Collection("products")

	// Perform operations
	products := []map[string]interface{}{
		{"name": "Laptop", "price": int64(1200), "stock": int64(50)},
		{"name": "Mouse", "price": int64(25), "stock": int64(200)},
		{"name": "Keyboard", "price": int64(80), "stock": int64(150)},
	}

	for _, product := range products {
		_, err := coll.InsertOne(product)
		if err != nil {
			log.Printf("Error inserting product: %v\n", err)
		}
	}

	fmt.Printf("Inserted %d products\n", len(products))
	fmt.Printf("Audit log written to: %s\n", auditLogFile)

	// Read and display audit log
	data, err := os.ReadFile(auditLogFile)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nAudit log contents:")
	fmt.Println(string(data))
}

func demo3FilteringOperations() {
	// Only audit write operations (insert, update, delete)
	auditConfig := &audit.Config{
		Enabled:     true,
		OutputWriter: os.Stdout,
		Format:      "json",
		Operations: []audit.OperationType{
			audit.OperationInsert,
			audit.OperationUpdate,
			audit.OperationDelete,
		},
	}

	dbConfig := &database.Config{
		DataDir:        "./audit_demo_data/demo3",
		BufferPoolSize: 100,
		AuditConfig:    auditConfig,
	}

	db, err := database.Open(dbConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	coll := db.Collection("orders")

	// Insert (should be logged)
	id, _ := coll.InsertOne(map[string]interface{}{
		"order_id": "ORD-001",
		"total":    int64(150),
	})
	fmt.Printf("Inserted order: %s\n", id)

	// Find (should NOT be logged due to filter)
	docs, _ := coll.Find(map[string]interface{}{})
	fmt.Printf("Found %d orders (not logged due to filter)\n", len(docs))

	// Update (should be logged)
	err = coll.UpdateOne(
		map[string]interface{}{"order_id": "ORD-001"},
		map[string]interface{}{"$set": map[string]interface{}{"status": "shipped"}},
	)
	if err == nil {
		fmt.Println("Updated order status")
	}

	// Delete (should be logged)
	err = coll.DeleteOne(map[string]interface{}{"order_id": "ORD-001"})
	if err == nil {
		fmt.Println("Deleted order")
	}
}

func demo4SeverityFiltering() {
	// Only log errors
	auditConfig := &audit.Config{
		Enabled:          true,
		OutputWriter:     os.Stdout,
		Format:           "json",
		MinSeverity:      audit.SeverityError,
		IncludeQueryData: true,
	}

	dbConfig := &database.Config{
		DataDir:        "./audit_demo_data/demo4",
		BufferPoolSize: 100,
		AuditConfig:    auditConfig,
	}

	db, err := database.Open(dbConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	coll := db.Collection("test")

	// Successful insert (not logged - severity is Info)
	id, err := coll.InsertOne(map[string]interface{}{
		"test": "data",
	})
	if err == nil {
		fmt.Printf("Inserted document %s (not logged - success is Info level)\n", id)
	}

	// Try to insert duplicate (error - will be logged)
	_, err = coll.InsertOne(map[string]interface{}{
		"_id":  id,
		"test": "duplicate",
	})
	if err != nil {
		fmt.Printf("Failed to insert duplicate (logged as Error): %v\n", err)
	}

	// Try to update non-existent document (error - will be logged)
	err = coll.UpdateOne(
		map[string]interface{}{"_id": "nonexistent"},
		map[string]interface{}{"$set": map[string]interface{}{"test": "updated"}},
	)
	if err != nil {
		fmt.Printf("Failed to update (logged as Error): %v\n", err)
	}
}

func demo5TextFormat() {
	// Use human-readable text format
	auditConfig := &audit.Config{
		Enabled:      true,
		OutputWriter: os.Stdout,
		Format:       "text",
		MinSeverity:  audit.SeverityInfo,
	}

	dbConfig := &database.Config{
		DataDir:        "./audit_demo_data/demo5",
		BufferPoolSize: 100,
		AuditConfig:    auditConfig,
	}

	db, err := database.Open(dbConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	coll := db.Collection("analytics")

	// Create index
	err = coll.CreateIndex("timestamp", false)
	if err == nil {
		fmt.Println("Created index")
	}

	// Insert document
	id, err := coll.InsertOne(map[string]interface{}{
		"event":     "page_view",
		"timestamp": time.Now().Unix(),
		"user_id":   "user123",
	})
	if err == nil {
		fmt.Printf("Logged event: %s\n", id)
	}

	// Aggregate
	pipeline := []map[string]interface{}{
		{"$match": map[string]interface{}{"event": "page_view"}},
		{"$group": map[string]interface{}{
			"_id":   "$user_id",
			"count": map[string]interface{}{"$sum": int64(1)},
		}},
	}
	results, err := coll.Aggregate(pipeline)
	if err == nil {
		fmt.Printf("Aggregation returned %d results\n", len(results))
	}

	// Drop index
	err = coll.DropIndex("timestamp_1")
	if err == nil {
		fmt.Println("Dropped index")
	}
}
