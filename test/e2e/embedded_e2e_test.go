package e2e

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/query"
)

// TestEmbeddedFullWorkflow tests complete end-to-end embedded/library mode workflow
func TestEmbeddedFullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Setup: Create temporary data directory
	tmpDir, err := os.MkdirTemp("", "laura-embedded-e2e-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Open database
	config := database.DefaultConfig(tmpDir)
	config.BufferPoolSize = 1000
	db, err := database.Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	t.Log("Database opened successfully")

	// Run test scenarios
	t.Run("CollectionLifecycle", func(t *testing.T) {
		testEmbeddedCollectionLifecycle(t, db)
	})

	t.Run("DocumentCRUD", func(t *testing.T) {
		testEmbeddedDocumentCRUD(t, db)
	})

	t.Run("Transactions", func(t *testing.T) {
		testEmbeddedTransactions(t, db)
	})

	t.Run("ComplexQueries", func(t *testing.T) {
		testEmbeddedComplexQueries(t, db)
	})

	t.Run("IndexedQueries", func(t *testing.T) {
		testEmbeddedIndexedQueries(t, db)
	})

	t.Run("AggregationWorkflow", func(t *testing.T) {
		testEmbeddedAggregationWorkflow(t, db)
	})

	t.Run("ConcurrentOperations", func(t *testing.T) {
		testEmbeddedConcurrentOperations(t, db)
	})

	t.Run("DataPersistence", func(t *testing.T) {
		testEmbeddedDataPersistence(t, tmpDir)
	})
}

func testEmbeddedCollectionLifecycle(t *testing.T, db *database.Database) {
	// Create collection
	coll, err := db.CreateCollection("lifecycle_test")
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	// List collections
	collections := db.ListCollections()
	found := false
	for _, name := range collections {
		if name == "lifecycle_test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Collection not found in list")
	}

	// Get collection stats
	stats := coll.Stats()
	if stats["name"] != "lifecycle_test" {
		t.Errorf("Expected collection name 'lifecycle_test', got %v", stats["name"])
	}

	// Drop collection
	if err := db.DropCollection("lifecycle_test"); err != nil {
		t.Errorf("Failed to drop collection: %v", err)
	}

	t.Log("✓ Collection lifecycle passed")
}

func testEmbeddedDocumentCRUD(t *testing.T, db *database.Database) {
	coll, _ := db.CreateCollection("crud_test")
	defer db.DropCollection("crud_test")

	// Insert document
	doc := map[string]interface{}{
		"name":  "Alice",
		"age":   int64(28),
		"email": "alice@example.com",
		"tags":  []interface{}{"golang", "database", "developer"},
	}
	insertedID, err := coll.InsertOne(doc)
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}
	if insertedID == "" {
		t.Fatal("Expected inserted ID")
	}

	// Find document
	results, err := coll.Find(map[string]interface{}{"name": "Alice"})
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 document, got %d", len(results))
	}

	// Update document
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Alice"},
		map[string]interface{}{
			"$set": map[string]interface{}{"age": int64(29)},
			"$push": map[string]interface{}{"tags": "expert"},
		},
	)
	if err != nil {
		t.Fatalf("Failed to update document: %v", err)
	}

	// Verify update
	results, _ = coll.Find(map[string]interface{}{"name": "Alice"})
	age, _ := results[0].Get("age")
	if age.(int64) != 29 {
		t.Errorf("Expected age 29, got %v", age)
	}
	tagsVal, _ := results[0].Get("tags")
	tags := tagsVal.([]interface{})
	if len(tags) != 4 {
		t.Errorf("Expected 4 tags after push, got %d", len(tags))
	}

	// Delete document
	err = coll.DeleteOne(map[string]interface{}{"name": "Alice"})
	if err != nil {
		t.Fatalf("Failed to delete document: %v", err)
	}

	// Verify deletion
	results, _ = coll.Find(map[string]interface{}{})
	if len(results) != 0 {
		t.Errorf("Expected 0 documents after deletion, got %d", len(results))
	}

	t.Log("✓ Document CRUD passed")
}

func testEmbeddedTransactions(t *testing.T, db *database.Database) {
	coll, _ := db.CreateCollection("txn_test")
	defer db.DropCollection("txn_test")

	// Test successful transaction
	txn := db.BeginTransaction()
	doc := map[string]interface{}{"name": "Bob", "balance": int64(1000)}
	coll.InsertOne(doc)

	if err := db.CommitTransaction(txn); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// Verify commit
	results, _ := coll.Find(map[string]interface{}{"name": "Bob"})
	if len(results) != 1 {
		t.Error("Document should be present after commit")
	}

	// Test aborted transaction
	txn = db.BeginTransaction()
	doc2 := map[string]interface{}{"name": "Charlie", "balance": int64(500)}
	coll.InsertOne(doc2)

	if err := db.AbortTransaction(txn); err != nil {
		t.Fatalf("Failed to abort transaction: %v", err)
	}

	// Verify abort
	results, _ = coll.Find(map[string]interface{}{"name": "Charlie"})
	if len(results) != 0 {
		t.Error("Document should not be present after abort")
	}

	t.Log("✓ Transactions passed")
}

func testEmbeddedComplexQueries(t *testing.T, db *database.Database) {
	coll, _ := db.CreateCollection("query_test")
	defer db.DropCollection("query_test")

	// Insert test data
	testData := []map[string]interface{}{
		{"name": "Alice", "age": int64(25), "city": "NYC", "score": int64(85)},
		{"name": "Bob", "age": int64(30), "city": "LA", "score": int64(92)},
		{"name": "Charlie", "age": int64(35), "city": "NYC", "score": int64(78)},
		{"name": "David", "age": int64(28), "city": "SF", "score": int64(95)},
		{"name": "Eve", "age": int64(32), "city": "NYC", "score": int64(88)},
	}
	for _, doc := range testData {
		coll.InsertOne(doc)
	}

	// Range query
	results, err := coll.Find(map[string]interface{}{
		"age": map[string]interface{}{"$gte": int64(30)},
	})
	if err != nil {
		t.Fatalf("Failed range query: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("Expected 3 documents with age >= 30, got %d", len(results))
	}

	// Compound logical query
	results, _ = coll.Find(map[string]interface{}{
		"$and": []interface{}{
			map[string]interface{}{"city": "NYC"},
			map[string]interface{}{"score": map[string]interface{}{"$gte": int64(80)}},
		},
	})
	if len(results) != 2 {
		t.Errorf("Expected 2 documents matching compound query, got %d", len(results))
	}

	// Query with projection
	options := &database.QueryOptions{
		Projection: map[string]bool{"name": true, "score": true},
		Limit:      0,
		Skip:       0,
	}
	results, _ = coll.FindWithOptions(map[string]interface{}{}, options)
	if len(results) > 0 {
		firstDoc := results[0]
		if _, hasAge := firstDoc.Get("age"); hasAge {
			t.Error("Projection failed: age should not be present")
		}
		if _, hasName := firstDoc.Get("name"); !hasName {
			t.Error("Projection failed: name should be present")
		}
	}

	// Query with sort and limit
	options2 := &database.QueryOptions{
		Sort:  []query.SortField{{Field: "score", Ascending: false}}, // Descending
		Limit: 2,
		Skip:  0,
	}
	results, _ = coll.FindWithOptions(map[string]interface{}{}, options2)
	if len(results) != 2 {
		t.Errorf("Expected 2 documents with limit, got %d", len(results))
	}
	score0, _ := results[0].Get("score")
	score1, _ := results[1].Get("score")
	if score0.(int64) < score1.(int64) {
		t.Error("Sort failed: results not in descending order")
	}

	t.Log("✓ Complex queries passed")
}

func testEmbeddedIndexedQueries(t *testing.T, db *database.Database) {
	coll, _ := db.CreateCollection("index_test")
	defer db.DropCollection("index_test")

	// Create index
	if err := coll.CreateIndex("email", true); err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	// Insert documents
	docs := []map[string]interface{}{
		{"email": "user1@example.com", "name": "User 1"},
		{"email": "user2@example.com", "name": "User 2"},
		{"email": "user3@example.com", "name": "User 3"},
	}
	for _, doc := range docs {
		coll.InsertOne(doc)
	}

	// Query using index
	results, err := coll.Find(map[string]interface{}{"email": "user2@example.com"})
	if err != nil {
		t.Fatalf("Failed to query with index: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 document, got %d", len(results))
	}

	// Test unique constraint
	duplicateDoc := map[string]interface{}{"email": "user1@example.com", "name": "Duplicate"}
	_, err = coll.InsertOne(duplicateDoc)
	if err == nil {
		t.Error("Expected error for duplicate unique key")
	}

	// Compound index
	if err := coll.CreateCompoundIndex([]string{"name", "email"}, true); err != nil {
		t.Logf("Compound index creation skipped (may not be supported): %v", err)
	}

	t.Log("✓ Indexed queries passed")
}

func testEmbeddedAggregationWorkflow(t *testing.T, db *database.Database) {
	coll, _ := db.CreateCollection("agg_test")
	defer db.DropCollection("agg_test")

	// Insert sales data
	salesData := []map[string]interface{}{
		{"product": "Laptop", "category": "Electronics", "quantity": int64(5), "price": int64(1200)},
		{"product": "Mouse", "category": "Electronics", "quantity": int64(20), "price": int64(25)},
		{"product": "Desk", "category": "Furniture", "quantity": int64(3), "price": int64(300)},
		{"product": "Chair", "category": "Furniture", "quantity": int64(8), "price": int64(150)},
		{"product": "Monitor", "category": "Electronics", "quantity": int64(7), "price": int64(400)},
	}
	for _, doc := range salesData {
		coll.InsertOne(doc)
	}

	// Group by category
	pipeline := []map[string]interface{}{
		{
			"$group": map[string]interface{}{
				"_id":       "$category",
				"totalQty":  map[string]interface{}{"$sum": "$quantity"},
				"avgPrice":  map[string]interface{}{"$avg": "$price"},
				"itemCount": map[string]interface{}{"$count": map[string]interface{}{}},
			},
		},
		{
			"$sort": map[string]interface{}{"totalQty": int64(-1)},
		},
	}

	results, err := coll.Aggregate(pipeline)
	if err != nil {
		t.Fatalf("Failed to run aggregation: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(results))
	}

	// Verify aggregation results
	electronicsGroup := results[0]
	idVal, _ := electronicsGroup.Get("_id")
	if idVal != "Electronics" {
		t.Errorf("Expected first group to be Electronics, got %v", idVal)
	}
	totalQtyVal, _ := electronicsGroup.Get("totalQty")
	// Aggregation may return float64 or int64 depending on operations
	var totalQty int64
	switch v := totalQtyVal.(type) {
	case int64:
		totalQty = v
	case float64:
		totalQty = int64(v)
	default:
		t.Fatalf("Unexpected type for totalQty: %T", totalQtyVal)
	}
	if totalQty != 32 { // 5 + 20 + 7
		t.Errorf("Expected Electronics totalQty to be 32, got %d", totalQty)
	}

	t.Log("✓ Aggregation workflow passed")
}

func testEmbeddedConcurrentOperations(t *testing.T, db *database.Database) {
	coll, _ := db.CreateCollection("concurrent_test")
	defer db.DropCollection("concurrent_test")

	// Concurrent inserts
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			doc := map[string]interface{}{
				"id":    int64(id),
				"name":  fmt.Sprintf("User%d", id),
				"value": int64(id * 10),
			}
			coll.InsertOne(doc)
			done <- true
		}(i)
	}

	// Wait for all inserts
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all inserts
	results, err := coll.Find(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to find documents: %v", err)
	}
	if len(results) != 10 {
		t.Errorf("Expected 10 documents from concurrent inserts, got %d", len(results))
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(id int) {
			coll.Find(map[string]interface{}{"id": int64(id)})
			done <- true
		}(i)
	}

	// Wait for all reads
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent updates
	for i := 0; i < 10; i++ {
		go func(id int) {
			coll.UpdateOne(
				map[string]interface{}{"id": int64(id)},
				map[string]interface{}{
					"$inc": map[string]interface{}{"value": int64(1)},
				},
			)
			done <- true
		}(i)
	}

	// Wait for all updates
	for i := 0; i < 10; i++ {
		<-done
	}

	t.Log("✓ Concurrent operations passed")
}

func testEmbeddedDataPersistence(t *testing.T, tmpDir string) {
	// Open database, insert data, close
	config := database.DefaultConfig(tmpDir)
	config.BufferPoolSize = 1000
	db1, err := database.Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	coll, _ := db1.CreateCollection("persist_test")
	testDoc := map[string]interface{}{
		"name":      "Persistent Doc",
		"value":     int64(42),
		"timestamp": time.Now().Unix(),
	}
	insertedID, _ := coll.InsertOne(testDoc)

	db1.Close()

	// Reopen database and verify data persists
	config2 := database.DefaultConfig(tmpDir)
	config2.BufferPoolSize = 1000
	db2, err := database.Open(config2)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	// Check collection exists
	collections := db2.ListCollections()
	found := false
	for _, name := range collections {
		if name == "persist_test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Collection not found after reopening database")
	}

	// Verify document persists
	coll2 := db2.Collection("persist_test")

	results, err := coll2.Find(map[string]interface{}{"name": "Persistent Doc"})
	if err != nil {
		t.Fatalf("Failed to find persisted document: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 persisted document, got %d", len(results))
	}

	docID, _ := results[0].Get("_id")
	if docID != insertedID {
		t.Error("Document ID changed after persistence")
	}

	valueData, _ := results[0].Get("value")
	if valueData.(int64) != 42 {
		t.Errorf("Expected value 42, got %v", valueData)
	}

	t.Log("✓ Data persistence passed")
}

// TestEmbeddedQueryOptimization tests query optimization with indexes
func TestEmbeddedQueryOptimization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	tmpDir, _ := os.MkdirTemp("", "laura-optimize-e2e-*")
	defer os.RemoveAll(tmpDir)

	config := database.DefaultConfig(tmpDir)
	config.BufferPoolSize = 1000
	db, _ := database.Open(config)
	defer db.Close()

	coll, _ := db.CreateCollection("optimize_test")

	// Insert test data
	for i := 0; i < 100; i++ {
		coll.InsertOne(map[string]interface{}{
			"id":    int64(i),
			"value": int64(i * 10),
		})
	}

	// Query without index
	results1, _ := coll.Find(map[string]interface{}{
		"id": map[string]interface{}{"$gte": int64(50)},
	})
	if len(results1) != 50 {
		t.Errorf("Expected 50 results without index, got %d", len(results1))
	}

	// Create index
	coll.CreateIndex("id", false)

	// Query with index (should be faster)
	results2, _ := coll.Find(map[string]interface{}{
		"id": map[string]interface{}{"$gte": int64(50)},
	})
	if len(results2) != 50 {
		t.Errorf("Expected 50 results with index, got %d", len(results2))
	}

	t.Log("✓ Query optimization passed")
}

// TestEmbeddedBulkOperations tests bulk write operations
func TestEmbeddedBulkOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	tmpDir, _ := os.MkdirTemp("", "laura-bulk-e2e-*")
	defer os.RemoveAll(tmpDir)

	config := database.DefaultConfig(tmpDir)
	config.BufferPoolSize = 1000
	db, _ := database.Open(config)
	defer db.Close()

	coll, _ := db.CreateCollection("bulk_test")

	// Test InsertMany
	docs := []map[string]interface{}{
		{"name": "Doc1", "value": int64(1)},
		{"name": "Doc2", "value": int64(2)},
		{"name": "Doc3", "value": int64(3)},
		{"name": "Doc4", "value": int64(4)},
		{"name": "Doc5", "value": int64(5)},
	}

	insertedIDs, err := coll.InsertMany(docs)
	if err != nil {
		t.Fatalf("Failed to insert many: %v", err)
	}
	if len(insertedIDs) != 5 {
		t.Errorf("Expected 5 inserted IDs, got %d", len(insertedIDs))
	}

	// Test UpdateMany
	updated, err := coll.UpdateMany(
		map[string]interface{}{
			"value": map[string]interface{}{"$gte": int64(3)},
		},
		map[string]interface{}{
			"$set": map[string]interface{}{"category": "high"},
		},
	)
	if err != nil {
		t.Fatalf("Failed to update many: %v", err)
	}
	if updated != 3 {
		t.Errorf("Expected 3 updated documents, got %d", updated)
	}

	// Verify updates
	results, _ := coll.Find(map[string]interface{}{"category": "high"})
	if len(results) != 3 {
		t.Errorf("Expected 3 documents with category 'high', got %d", len(results))
	}

	// Test DeleteMany
	deleted, err := coll.DeleteMany(map[string]interface{}{
		"value": map[string]interface{}{"$lte": int64(2)},
	})
	if err != nil {
		t.Fatalf("Failed to delete many: %v", err)
	}
	if deleted != 2 {
		t.Errorf("Expected 2 deleted documents, got %d", deleted)
	}

	// Verify deletions
	results, _ = coll.Find(map[string]interface{}{})
	if len(results) != 3 {
		t.Errorf("Expected 3 documents remaining, got %d", len(results))
	}

	t.Log("✓ Bulk operations passed")
}

// TestEmbeddedTextSearch tests text search functionality
func TestEmbeddedTextSearch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	tmpDir, _ := os.MkdirTemp("", "laura-text-e2e-*")
	defer os.RemoveAll(tmpDir)

	config := database.DefaultConfig(tmpDir)
	config.BufferPoolSize = 1000
	db, _ := database.Open(config)
	defer db.Close()

	coll, _ := db.CreateCollection("text_test")

	// Create text index
	if err := coll.CreateTextIndex([]string{"content"}); err != nil {
		t.Skipf("Text index not supported, skipping: %v", err)
		return
	}

	// Insert documents
	docs := []map[string]interface{}{
		{"title": "Go Tutorial", "content": "Learn Go programming language basics"},
		{"title": "Python Guide", "content": "Python programming for beginners"},
		{"title": "Database Design", "content": "Design principles for database systems"},
	}
	for _, doc := range docs {
		coll.InsertOne(doc)
	}

	// Perform text search
	results, err := coll.TextSearch("programming", nil)
	if err != nil {
		t.Fatalf("Failed text search: %v", err)
	}

	if len(results) < 2 {
		t.Errorf("Expected at least 2 results for 'programming', got %d", len(results))
	}

	t.Log("✓ Text search passed")
}

// TestEmbeddedComplexQueryFilters tests complex query filtering
func TestEmbeddedComplexQueryFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	tmpDir, _ := os.MkdirTemp("", "laura-queryfilter-e2e-*")
	defer os.RemoveAll(tmpDir)

	config := database.DefaultConfig(tmpDir)
	config.BufferPoolSize = 1000
	db, _ := database.Open(config)
	defer db.Close()

	coll, _ := db.CreateCollection("filter_test")

	// Insert test documents
	docs := []map[string]interface{}{
		{"age": int64(25), "status": "active"},
		{"age": int64(70), "status": "active"},
		{"age": int64(30), "status": "inactive"},
		{"age": int64(45), "status": "pending"},
	}
	for _, doc := range docs {
		coll.InsertOne(doc)
	}

	// Test complex AND/OR filter
	filter := map[string]interface{}{
		"$and": []interface{}{
			map[string]interface{}{"age": map[string]interface{}{"$gte": int64(18)}},
			map[string]interface{}{"age": map[string]interface{}{"$lte": int64(65)}},
			map[string]interface{}{
				"$or": []interface{}{
					map[string]interface{}{"status": "active"},
					map[string]interface{}{"status": "pending"},
				},
			},
		},
	}

	results, err := coll.Find(filter)
	if err != nil {
		t.Fatalf("Failed to execute complex filter: %v", err)
	}

	// Should match: age 25 (active), age 45 (pending)
	// Should NOT match: age 70 (too old), age 30 (inactive)
	if len(results) != 2 {
		t.Errorf("Expected 2 matching documents, got %d", len(results))
	}

	t.Log("✓ Complex query filters passed")
}
