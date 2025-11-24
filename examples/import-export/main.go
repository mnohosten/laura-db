package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/impex"
)

func main() {
	fmt.Println("LauraDB Import/Export Example")
	fmt.Println("==============================")

	// Open database
	db, err := database.Open(database.DefaultConfig("./data/impex-demo"))
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Get or create collection
	coll := db.Collection("users")

	// Insert sample documents
	fmt.Println("\n1. Inserting sample documents...")
	users := []map[string]interface{}{
		{
			"name":   "Alice Johnson",
			"age":    int64(30),
			"email":  "alice@example.com",
			"active": true,
			"tags":   []interface{}{"admin", "developer"},
		},
		{
			"name":   "Bob Smith",
			"age":    int64(25),
			"email":  "bob@example.com",
			"active": true,
			"tags":   []interface{}{"developer"},
		},
		{
			"name":   "Charlie Brown",
			"age":    int64(35),
			"email":  "charlie@example.com",
			"active": false,
			"tags":   []interface{}{"manager"},
		},
	}

	for _, user := range users {
		if _, err := coll.InsertOne(user); err != nil {
			log.Fatalf("Failed to insert document: %v", err)
		}
	}
	fmt.Printf("Inserted %d documents\n", len(users))

	// Find all documents
	docs, err := coll.Find(map[string]interface{}{})
	if err != nil {
		log.Fatalf("Failed to find documents: %v", err)
	}

	// Export to JSON (pretty)
	fmt.Println("\n2. Exporting to JSON (pretty)...")
	var jsonBuf bytes.Buffer
	if err := impex.Export(&jsonBuf, docs, impex.FormatJSON, map[string]interface{}{"pretty": true}); err != nil {
		log.Fatalf("Failed to export JSON: %v", err)
	}
	fmt.Println(jsonBuf.String())

	// Export to JSON file
	jsonFile, err := os.Create("./data/users-export.json")
	if err != nil {
		log.Fatalf("Failed to create JSON file: %v", err)
	}
	defer jsonFile.Close()

	if err := impex.Export(jsonFile, docs, impex.FormatJSON, map[string]interface{}{"pretty": true}); err != nil {
		log.Fatalf("Failed to export to JSON file: %v", err)
	}
	fmt.Println("Exported to ./data/users-export.json")

	// Export to CSV
	fmt.Println("\n3. Exporting to CSV...")
	var csvBuf bytes.Buffer
	fields := []string{"name", "age", "email", "active"}
	if err := impex.Export(&csvBuf, docs, impex.FormatCSV, map[string]interface{}{"fields": fields}); err != nil {
		log.Fatalf("Failed to export CSV: %v", err)
	}
	fmt.Println(csvBuf.String())

	// Export to CSV file
	csvFile, err := os.Create("./data/users-export.csv")
	if err != nil {
		log.Fatalf("Failed to create CSV file: %v", err)
	}
	defer csvFile.Close()

	if err := impex.Export(csvFile, docs, impex.FormatCSV, map[string]interface{}{"fields": fields}); err != nil {
		log.Fatalf("Failed to export to CSV file: %v", err)
	}
	fmt.Println("Exported to ./data/users-export.csv")

	// Import from JSON
	fmt.Println("\n4. Importing from JSON...")
	jsonFile, err = os.Open("./data/users-export.json")
	if err != nil {
		log.Fatalf("Failed to open JSON file: %v", err)
	}
	defer jsonFile.Close()

	importedDocs, err := impex.Import(jsonFile, impex.FormatJSON, nil)
	if err != nil {
		log.Fatalf("Failed to import JSON: %v", err)
	}
	fmt.Printf("Imported %d documents from JSON\n", len(importedDocs))

	// Import from CSV
	fmt.Println("\n5. Importing from CSV...")
	csvFile, err = os.Open("./data/users-export.csv")
	if err != nil {
		log.Fatalf("Failed to open CSV file: %v", err)
	}
	defer csvFile.Close()

	importedCSVDocs, err := impex.Import(csvFile, impex.FormatCSV, nil)
	if err != nil {
		log.Fatalf("Failed to import CSV: %v", err)
	}
	fmt.Printf("Imported %d documents from CSV\n", len(importedCSVDocs))

	// Insert imported documents into new collection
	fmt.Println("\n6. Inserting imported documents into new collection...")
	importColl := db.Collection("imported_users")

	for _, doc := range importedCSVDocs {
		docMap := doc.ToMap()
		// Remove _id to generate new ones
		delete(docMap, "_id")
		if _, err := importColl.InsertOne(docMap); err != nil {
			log.Fatalf("Failed to insert imported document: %v", err)
		}
	}

	count, err := importColl.Count(map[string]interface{}{})
	if err != nil {
		log.Fatalf("Failed to count documents: %v", err)
	}
	fmt.Printf("Imported collection now has %d documents\n", count)

	// Export with auto-detected fields
	fmt.Println("\n7. Exporting CSV with auto-detected fields...")
	var autoCSVBuf bytes.Buffer
	if err := impex.Export(&autoCSVBuf, docs, impex.FormatCSV, nil); err != nil {
		log.Fatalf("Failed to export CSV: %v", err)
	}
	fmt.Println(autoCSVBuf.String())

	fmt.Println("\n✓ Import/Export example completed successfully!")
}
