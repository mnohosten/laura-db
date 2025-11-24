package main

import (
	"fmt"
	"log"
	"os"

	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/repair"
)

func main() {
	fmt.Println("╔═══════════════════════════════════════════╗")
	fmt.Println("║   LauraDB Repair & Maintenance Demo      ║")
	fmt.Println("╚═══════════════════════════════════════════╝")
	fmt.Println()

	// Clean up any existing test database
	os.RemoveAll("./repair-demo-data")
	defer os.RemoveAll("./repair-demo-data")

	// Open database
	config := database.DefaultConfig("./repair-demo-data")
	db, err := database.Open(config)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	fmt.Println("Database opened successfully")
	fmt.Println()

	// Run all demos
	runDemo1_BasicValidation(db)
	runDemo2_RepairWithIssues(db)
	runDemo3_IndexRebuild(db)
	runDemo4_Defragmentation(db)
	runDemo5_CollectionSpecificRepair(db)

	fmt.Println("╔═══════════════════════════════════════════╗")
	fmt.Println("║  All Repair & Maintenance Demos Complete ║")
	fmt.Println("╚═══════════════════════════════════════════╝")
}

func runDemo1_BasicValidation(db *database.Database) {
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("Demo 1: Basic Database Validation")
	fmt.Println("═══════════════════════════════════════════")

	// Create a collection and insert some documents
	coll := db.Collection("users")

	// Insert normal documents
	for i := 1; i <= 5; i++ {
		doc := map[string]interface{}{
			"name":  fmt.Sprintf("User %d", i),
			"age":   int64(20 + i),
			"email": fmt.Sprintf("user%d@example.com", i),
		}

		if _, err := coll.InsertOne(doc); err != nil {
			log.Printf("Failed to insert document: %v", err)
		}
	}

	fmt.Printf("Inserted 5 test documents\n\n")

	// Create indexes
	coll.CreateIndex("email", true)
	coll.CreateIndex("age", false)

	fmt.Printf("Created indexes on 'email' (unique) and 'age'\n\n")

	// Validate the database
	validator := repair.NewValidator(db)
	report, err := validator.Validate()
	if err != nil {
		log.Printf("Validation failed: %v", err)
		return
	}

	// Print results
	printValidationReport(report)
	fmt.Println()
}

func runDemo2_RepairWithIssues(db *database.Database) {
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("Demo 2: Repair Operation (Dry Run)")
	fmt.Println("═══════════════════════════════════════════")

	// First validate to find any issues
	validator := repair.NewValidator(db)
	validationReport, err := validator.Validate()
	if err != nil {
		log.Printf("Validation failed: %v", err)
		return
	}

	fmt.Printf("Found %d issues to repair\n\n", len(validationReport.Issues))

	// Perform a dry-run repair
	repairer := repair.NewRepairer(db)
	options := &repair.RepairOptions{
		RebuildIndexes:           false,
		RemoveOrphans:            true,
		AddMissingEntries:        true,
		UniqueConflictResolution: "fail",
		DryRun:                   true, // Dry run - don't make changes
	}

	fmt.Println("Repair Options:")
	fmt.Printf("  Rebuild Indexes:     %v\n", options.RebuildIndexes)
	fmt.Printf("  Remove Orphans:      %v\n", options.RemoveOrphans)
	fmt.Printf("  Add Missing Entries: %v\n", options.AddMissingEntries)
	fmt.Printf("  Dry Run:             %v\n\n", options.DryRun)

	repairReport, err := repairer.Repair(options)
	if err != nil {
		log.Printf("Repair failed: %v", err)
		return
	}

	printRepairReport(repairReport)
	fmt.Println()
}

func runDemo3_IndexRebuild(db *database.Database) {
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("Demo 3: Index Rebuild")
	fmt.Println("═══════════════════════════════════════════")

	coll := db.Collection("products")

	// Insert documents
	products := []map[string]interface{}{
		{"name": "Laptop", "price": int64(1200), "category": "Electronics"},
		{"name": "Mouse", "price": int64(25), "category": "Electronics"},
		{"name": "Desk", "price": int64(350), "category": "Furniture"},
		{"name": "Chair", "price": int64(200), "category": "Furniture"},
		{"name": "Monitor", "price": int64(300), "category": "Electronics"},
	}

	for _, p := range products {
		coll.InsertOne(p)
	}

	fmt.Printf("Inserted %d products\n", len(products))

	// Create indexes
	coll.CreateIndex("category", false)
	coll.CreateIndex("price", false)

	fmt.Println("Created indexes on 'category' and 'price'")
	fmt.Println()

	// Rebuild indexes
	fmt.Println("Rebuilding indexes...")

	repairer := repair.NewRepairer(db)
	options := &repair.RepairOptions{
		RebuildIndexes: true,
		DryRun:         false,
	}

	repairReport, err := repairer.RepairCollection("products", options)
	if err != nil {
		log.Printf("Repair failed: %v", err)
		return
	}

	printRepairReport(repairReport)
	fmt.Println()

	// Verify indexes still work
	results, err := coll.Find(map[string]interface{}{
		"category": "Electronics",
	})
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}

	fmt.Printf("Query test: Found %d electronics after index rebuild\n", len(results))
	fmt.Println()
}

func runDemo4_Defragmentation(db *database.Database) {
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("Demo 4: Database Defragmentation")
	fmt.Println("═══════════════════════════════════════════")

	// Create a collection with many documents
	coll := db.Collection("logs")

	fmt.Println("Inserting 100 log documents...")
	for i := 1; i <= 100; i++ {
		doc := map[string]interface{}{
			"timestamp": int64(1700000000 + i),
			"level":     []string{"INFO", "WARN", "ERROR"}[i%3],
			"message":   fmt.Sprintf("Log message %d with some additional text to make it larger", i),
			"source":    fmt.Sprintf("service-%d", i%5),
		}

		coll.InsertOne(doc)
	}

	// Create indexes
	coll.CreateIndex("level", false)
	coll.CreateIndex("source", false)

	fmt.Println("Created indexes on 'level' and 'source'")
	fmt.Println()

	// Get initial stats
	stats := coll.Stats()
	fmt.Printf("Collection stats before defragmentation:\n")
	fmt.Printf("  Documents: %v\n", stats["document_count"])
	fmt.Printf("  Indexes:   %v\n", stats["index_count"])
	fmt.Println()

	// Defragment
	fmt.Println("Running defragmentation...")
	defragmenter := repair.NewDefragmenter(db)
	defragReport, err := defragmenter.Defragment()
	if err != nil {
		log.Printf("Defragmentation failed: %v", err)
		return
	}

	printDefragmentationReport(defragReport)
	fmt.Println()

	// Verify data integrity
	results, err := coll.Find(map[string]interface{}{})
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}

	fmt.Printf("Data integrity check: %d documents found (expected 100)\n", len(results))
	fmt.Println()
}

func runDemo5_CollectionSpecificRepair(db *database.Database) {
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("Demo 5: Collection-Specific Operations")
	fmt.Println("═══════════════════════════════════════════")

	coll := db.Collection("orders")

	// Insert orders
	for i := 1; i <= 10; i++ {
		doc := map[string]interface{}{
			"order_id": fmt.Sprintf("ORD-%04d", i),
			"customer": fmt.Sprintf("Customer %d", i),
			"total":    int64(100 * i),
			"status":   []string{"pending", "shipped", "delivered"}[i%3],
		}

		coll.InsertOne(doc)
	}

	fmt.Printf("Inserted 10 orders\n")

	// Create index
	coll.CreateIndex("order_id", true)
	fmt.Println("Created unique index on 'order_id'")
	fmt.Println()

	// Validate specific collection
	fmt.Println("Validating 'orders' collection...")
	validator := repair.NewValidator(db)
	report, err := validator.ValidateCollection("orders")
	if err != nil {
		log.Printf("Validation failed: %v", err)
		return
	}

	printValidationReport(report)
	fmt.Println()

	// Defragment specific collection
	fmt.Println("Defragmenting 'orders' collection...")
	defragmenter := repair.NewDefragmenter(db)
	defragReport, err := defragmenter.DefragmentCollection("orders")
	if err != nil {
		log.Printf("Defragmentation failed: %v", err)
		return
	}

	printDefragmentationReport(defragReport)
	fmt.Println()
}

// Helper functions to print reports

func printValidationReport(report *repair.ValidationReport) {
	fmt.Println("─────────────────────────────────────────")
	fmt.Println("Validation Report")
	fmt.Println("─────────────────────────────────────────")
	fmt.Printf("Duration:     %v\n", report.EndTime.Sub(report.StartTime))
	fmt.Printf("Collections:  %d\n", len(report.Collections))
	fmt.Printf("Documents:    %d\n", report.DocumentCount)
	fmt.Printf("Indexes:      %d\n", report.IndexCount)
	fmt.Printf("Health:       %s\n", healthStatus(report.IsHealthy))
	fmt.Printf("Issues:       %d\n", len(report.Issues))
	fmt.Println("─────────────────────────────────────────")
	fmt.Println(report.Summary())

	if len(report.Issues) > 0 {
		fmt.Println("\nIssues:")
		for i, issue := range report.Issues {
			fmt.Printf("  [%d] %s - %s (Collection: %s)\n",
				i+1, issue.Severity, issue.Description, issue.Collection)
		}
	}
}

func printRepairReport(report *repair.RepairReport) {
	fmt.Println("─────────────────────────────────────────")
	fmt.Println("Repair Report")
	fmt.Println("─────────────────────────────────────────")
	fmt.Printf("Duration:     %v\n", report.EndTime.Sub(report.StartTime))
	fmt.Printf("Issues Found: %d\n", len(report.Issues))
	fmt.Printf("Fixed:        %d\n", report.Fixed)
	fmt.Printf("Failed:       %d\n", report.Failed)
	fmt.Println("─────────────────────────────────────────")
	fmt.Println(report.Summary())
}

func printDefragmentationReport(report *repair.DefragmentationReport) {
	percentSaved := 0.0
	if report.InitialFileSize > 0 {
		percentSaved = float64(report.SpaceSaved) / float64(report.InitialFileSize) * 100.0
	}

	fmt.Println("─────────────────────────────────────────")
	fmt.Println("Defragmentation Report")
	fmt.Println("─────────────────────────────────────────")
	fmt.Printf("Duration:          %v\n", report.EndTime.Sub(report.StartTime))
	fmt.Printf("Pages Compacted:   %d\n", report.PagesCompacted)
	fmt.Printf("Initial Size:      %s\n", formatBytes(report.InitialFileSize))
	fmt.Printf("Final Size:        %s\n", formatBytes(report.FinalFileSize))
	fmt.Printf("Space Saved:       %s (%.2f%%)\n", formatBytes(report.SpaceSaved), percentSaved)
	fmt.Printf("Fragmentation:     %.2f%%\n", report.FragmentationRatio*100.0)
	fmt.Println("─────────────────────────────────────────")
	fmt.Println(report.Summary())
}

func healthStatus(isHealthy bool) string {
	if isHealthy {
		return "✓ Healthy"
	}
	return "✗ Issues Detected"
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.2f %s", float64(bytes)/float64(div), units[exp])
}
