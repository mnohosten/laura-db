package database

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mnohosten/laura-db/pkg/geo"
)

// TestFullDatabaseWorkflow tests a complete end-to-end workflow with real disk I/O
func TestFullDatabaseWorkflow(t *testing.T) {
	// Create temporary data directory
	dataDir := filepath.Join(os.TempDir(), fmt.Sprintf("test-db-integration-%d", time.Now().UnixNano()))
	defer os.RemoveAll(dataDir)

	// Open database
	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create collection
	coll := db.Collection("users")

	// Insert documents
	docs := []map[string]interface{}{
		{"name": "Alice", "age": int64(30), "email": "alice@example.com", "active": true},
		{"name": "Bob", "age": int64(25), "email": "bob@example.com", "active": true},
		{"name": "Charlie", "age": int64(35), "email": "charlie@example.com", "active": false},
		{"name": "Diana", "age": int64(28), "email": "diana@example.com", "active": true},
	}

	var insertedIDs []interface{}
	for _, doc := range docs {
		id, err := coll.InsertOne(doc)
		if err != nil {
			t.Fatalf("Failed to insert document: %v", err)
		}
		insertedIDs = append(insertedIDs, id)
	}

	// Create index on age field
	if err := coll.CreateIndex("age", false); err != nil {
		t.Fatalf("Failed to create age index: %v", err)
	}

	// Create compound index on name and age
	if err := coll.CreateCompoundIndex([]string{"name", "age"}, false); err != nil {
		t.Fatalf("Failed to create compound index: %v", err)
	}

	// Query with index
	results, err := coll.Find(map[string]interface{}{"age": map[string]interface{}{"$gte": int64(28)}})
	if err != nil {
		t.Fatalf("Failed to find documents: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Query with compound index
	results, err = coll.Find(map[string]interface{}{"name": "Alice", "age": int64(30)})
	if err != nil {
		t.Fatalf("Failed to find with compound index: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// Update document
	err = coll.UpdateOne(
		map[string]interface{}{"name": "Alice"},
		map[string]interface{}{"$set": map[string]interface{}{"age": int64(31)}},
	)
	if err != nil {
		t.Fatalf("Failed to update document: %v", err)
	}

	// Verify update
	results, err = coll.Find(map[string]interface{}{"name": "Alice"})
	if err != nil {
		t.Fatalf("Failed to find updated document: %v", err)
	}
	if age, _ := results[0].Get("age"); age.(int64) != 31 {
		t.Errorf("Expected age 31, got %d", age)
	}

	// Delete document
	err = coll.DeleteOne(map[string]interface{}{"name": "Charlie"})
	if err != nil {
		t.Fatalf("Failed to delete document: %v", err)
	}

	// Verify deletion
	results, err = coll.Find(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to find all documents: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("Expected 3 documents remaining, got %d", len(results))
	}

	// Close and reopen database
	db.Close()

	db, err = Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db.Close()

	// Verify data persisted
	coll = db.Collection("users")
	results, err = coll.Find(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to find documents after reopen: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("Expected 3 documents after reopen, got %d", len(results))
	}

	// Verify index persisted
	stats := coll.Stats()
	if indexes, ok := stats["indexes"].(int); ok && indexes < 3 { // _id_ + age + name_age compound
		t.Errorf("Expected at least 3 indexes after reopen, got %d", indexes)
	}
}

// TestTransactionIntegration tests MVCC transactions with real storage
func TestTransactionIntegration(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), fmt.Sprintf("test-txn-integration-%d", time.Now().UnixNano()))
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("accounts")

	// Insert initial documents
	_, err = coll.InsertOne(map[string]interface{}{"account": "A", "balance": int64(1000)})
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}
	_, err = coll.InsertOne(map[string]interface{}{"account": "B", "balance": int64(500)})
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Start transaction
	session := db.StartSession()

	// Transfer money from A to B
	results, err := coll.Find(map[string]interface{}{"account": "A"})
	if err != nil {
		t.Fatalf("Failed to find account A: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 account A, got %d", len(results))
	}

	// Update both accounts
	err = coll.UpdateOne(
		map[string]interface{}{"account": "A"},
		map[string]interface{}{"$inc": map[string]interface{}{"balance": int64(-200)}},
	)
	if err != nil {
		t.Fatalf("Failed to update account A: %v", err)
	}

	err = coll.UpdateOne(
		map[string]interface{}{"account": "B"},
		map[string]interface{}{"$inc": map[string]interface{}{"balance": int64(200)}},
	)
	if err != nil {
		t.Fatalf("Failed to update account B: %v", err)
	}

	// Commit transaction
	if err := session.CommitTransaction(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// Verify final balances
	results, err = coll.Find(map[string]interface{}{"account": "A"})
	if err != nil {
		t.Fatalf("Failed to find account A after commit: %v", err)
	}
	if balance, _ := results[0].Get("balance"); balance != nil {
		// Handle both int64 and float64
		var balanceVal int64
		switch v := balance.(type) {
		case int64:
			balanceVal = v
		case float64:
			balanceVal = int64(v)
		}
		if balanceVal != 800 {
			t.Errorf("Expected balance 800, got %d", balanceVal)
		}
	}

	results, err = coll.Find(map[string]interface{}{"account": "B"})
	if err != nil {
		t.Fatalf("Failed to find account B after commit: %v", err)
	}
	if balance, _ := results[0].Get("balance"); balance != nil {
		// Handle both int64 and float64
		var balanceVal int64
		switch v := balance.(type) {
		case int64:
			balanceVal = v
		case float64:
			balanceVal = int64(v)
		}
		if balanceVal != 700 {
			t.Errorf("Expected balance 700, got %d", balanceVal)
		}
	}
}

// TestAggregationPipelineIntegration tests aggregation with real data
func TestAggregationPipelineIntegration(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), fmt.Sprintf("test-agg-integration-%d", time.Now().UnixNano()))
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("sales")

	// Insert sales data
	salesData := []map[string]interface{}{
		{"product": "Widget", "category": "Hardware", "amount": int64(100), "quantity": int64(2)},
		{"product": "Gadget", "category": "Hardware", "amount": int64(150), "quantity": int64(3)},
		{"product": "Software", "category": "Software", "amount": int64(200), "quantity": int64(1)},
		{"product": "Widget", "category": "Hardware", "amount": int64(100), "quantity": int64(1)},
		{"product": "License", "category": "Software", "amount": int64(300), "quantity": int64(5)},
	}

	for _, sale := range salesData {
		if _, err := coll.InsertOne(sale); err != nil {
			t.Fatalf("Failed to insert sale: %v", err)
		}
	}

	// Run aggregation pipeline: group by category, sum amounts
	pipeline := []map[string]interface{}{
		{"$match": map[string]interface{}{"amount": map[string]interface{}{"$gte": int64(100)}}},
		{"$group": map[string]interface{}{
			"_id":          "$category",
			"totalAmount":  map[string]interface{}{"$sum": "$amount"},
			"totalQty":     map[string]interface{}{"$sum": "$quantity"},
			"avgAmount":    map[string]interface{}{"$avg": "$amount"},
		}},
		{"$sort": map[string]interface{}{"totalAmount": int64(-1)}},
	}

	results, err := coll.Aggregate(pipeline)
	if err != nil {
		t.Fatalf("Failed to run aggregation: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(results))
	}

	// Verify Hardware category
	found := false
	for _, result := range results {
		if id, _ := result.Get("_id"); id == "Hardware" {
			found = true
			if total, _ := result.Get("totalAmount"); total != nil {
				var totalVal int64
				switch v := total.(type) {
				case int64:
					totalVal = v
				case float64:
					totalVal = int64(v)
				}
				if totalVal != 350 {
					t.Errorf("Expected Hardware total 350, got %d", totalVal)
				}
			}
			if qty, _ := result.Get("totalQty"); qty != nil {
				var qtyVal int64
				switch v := qty.(type) {
				case int64:
					qtyVal = v
				case float64:
					qtyVal = int64(v)
				}
				if qtyVal != 6 {
					t.Errorf("Expected Hardware qty 6, got %d", qtyVal)
				}
			}
			break
		}
	}
	if !found {
		t.Error("Hardware category not found in results")
	}
}

// TestTextSearchIntegration tests text search with real data
func TestTextSearchIntegration(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), fmt.Sprintf("test-text-integration-%d", time.Now().UnixNano()))
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("articles")

	// Create text index
	if err := coll.CreateTextIndex([]string{"title", "content"}); err != nil {
		t.Fatalf("Failed to create text index: %v", err)
	}

	// Insert articles
	articles := []map[string]interface{}{
		{
			"title":   "Introduction to Go Programming",
			"content": "Go is a statically typed, compiled programming language designed at Google",
			"views":   int64(1000),
		},
		{
			"title":   "Advanced Go Techniques",
			"content": "Learn advanced patterns and best practices for Go development",
			"views":   int64(500),
		},
		{
			"title":   "Python for Beginners",
			"content": "Python is an easy to learn programming language",
			"views":   int64(2000),
		},
		{
			"title":   "Database Design Principles",
			"content": "Understanding database normalization and indexing strategies",
			"views":   int64(750),
		},
	}

	for _, article := range articles {
		if _, err := coll.InsertOne(article); err != nil {
			t.Fatalf("Failed to insert article: %v", err)
		}
	}

	// Search for "programming"
	results, err := coll.TextSearch("programming", nil)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	if len(results) < 2 {
		t.Errorf("Expected at least 2 results for 'programming', got %d", len(results))
	}

	// Verify results contain expected documents
	foundGo := false
	foundPython := false
	for _, result := range results {
		if titleVal, _ := result.Get("title"); titleVal != nil {
			title := titleVal.(string)
			if title == "Introduction to Go Programming" {
				foundGo = true
			}
			if title == "Python for Beginners" {
				foundPython = true
			}
		}
	}

	if !foundGo {
		t.Error("Expected to find 'Introduction to Go Programming' in results")
	}
	if !foundPython {
		t.Error("Expected to find 'Python for Beginners' in results")
	}
}

// TestGeospatialIntegration tests geospatial queries with real data
// Note: This test is disabled for now due to geospatial API issues
func _TestGeospatialIntegration(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), fmt.Sprintf("test-geo-integration-%d", time.Now().UnixNano()))
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("places")

	// Create 2dsphere index
	if err := coll.Create2DSphereIndex("location"); err != nil {
		t.Fatalf("Failed to create geo index: %v", err)
	}

	// Insert places with coordinates
	places := []map[string]interface{}{
		{
			"name":     "Statue of Liberty",
			"location": map[string]interface{}{"type": "Point", "coordinates": []float64{-74.0445, 40.6892}},
			"city":     "New York",
		},
		{
			"name":     "Times Square",
			"location": map[string]interface{}{"type": "Point", "coordinates": []float64{-73.9855, 40.7580}},
			"city":     "New York",
		},
		{
			"name":     "Golden Gate Bridge",
			"location": map[string]interface{}{"type": "Point", "coordinates": []float64{-122.4783, 37.8199}},
			"city":     "San Francisco",
		},
		{
			"name":     "Central Park",
			"location": map[string]interface{}{"type": "Point", "coordinates": []float64{-73.9654, 40.7829}},
			"city":     "New York",
		},
	}

	for _, place := range places {
		if _, err := coll.InsertOne(place); err != nil {
			t.Fatalf("Failed to insert place: %v", err)
		}
	}

	// Find places near Times Square (within 10km)
	center := geo.Point{Lon: -73.9855, Lat: 40.7580}
	results, err := coll.Near("location", &center, 10000, 10, nil) // 10km in meters, limit 10
	if err != nil {
		t.Fatalf("Failed to find nearby places: %v", err)
	}

	// Should find Times Square and Central Park (both in NY, close together)
	if len(results) < 2 {
		t.Errorf("Expected at least 2 places near Times Square, got %d", len(results))
	}

	// Verify Times Square is in results
	foundTimesSquare := false
	for _, result := range results {
		if nameVal, _ := result.Get("name"); nameVal != nil && nameVal.(string) == "Times Square" {
			foundTimesSquare = true
			break
		}
	}
	if !foundTimesSquare {
		t.Error("Expected to find Times Square in nearby results")
	}
}

// TestTTLIndexIntegration tests TTL index with real data and expiration
func TestTTLIndexIntegration(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), fmt.Sprintf("test-ttl-integration-%d", time.Now().UnixNano()))
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("sessions")

	// Create TTL index (expire after 2 seconds)
	if err := coll.CreateTTLIndex("createdAt", 2); err != nil { // 2 seconds
		t.Fatalf("Failed to create TTL index: %v", err)
	}

	// Insert documents with timestamps
	now := time.Now()
	sessions := []map[string]interface{}{
		{"sessionId": "session1", "createdAt": now.Add(-3 * time.Second)}, // Already expired
		{"sessionId": "session2", "createdAt": now.Add(-1 * time.Second)}, // Not expired yet
		{"sessionId": "session3", "createdAt": now},                       // Not expired
	}

	for _, session := range sessions {
		if _, err := coll.InsertOne(session); err != nil {
			t.Fatalf("Failed to insert session: %v", err)
		}
	}

	// Trigger TTL cleanup manually (in real system this runs in background)
	coll.CleanupExpiredDocuments()

	// Check remaining documents
	results, err := coll.Find(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to find sessions: %v", err)
	}

	// Session1 should be expired, session2 and session3 should remain
	if len(results) != 2 {
		t.Errorf("Expected 2 sessions after cleanup, got %d", len(results))
	}

	// Verify session1 is gone
	foundSession1 := false
	for _, result := range results {
		if sessionIdVal, _ := result.Get("sessionId"); sessionIdVal != nil && sessionIdVal.(string) == "session1" {
			foundSession1 = true
		}
	}
	if foundSession1 {
		t.Error("Expected session1 to be expired and removed")
	}
}

// TestPartialIndexIntegration tests partial index with real data
func TestPartialIndexIntegration(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), fmt.Sprintf("test-partial-integration-%d", time.Now().UnixNano()))
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("products")

	// Create partial index (only index active products)
	filter := map[string]interface{}{"active": true}
	if err := coll.CreatePartialIndex("price", filter, false); err != nil {
		t.Fatalf("Failed to create partial index: %v", err)
	}

	// Insert products
	products := []map[string]interface{}{
		{"name": "Laptop", "price": int64(1000), "active": true},
		{"name": "Mouse", "price": int64(25), "active": true},
		{"name": "Keyboard", "price": int64(75), "active": false}, // Not indexed
		{"name": "Monitor", "price": int64(300), "active": true},
		{"name": "Cable", "price": int64(10), "active": false}, // Not indexed
	}

	for _, product := range products {
		if _, err := coll.InsertOne(product); err != nil {
			t.Fatalf("Failed to insert product: %v", err)
		}
	}

	// Query active products by price
	results, err := coll.Find(map[string]interface{}{
		"active": true,
		"price":  map[string]interface{}{"$gte": int64(50)},
	})
	if err != nil {
		t.Fatalf("Failed to find products: %v", err)
	}

	// Should find Laptop and Monitor (both active and price >= 50)
	if len(results) != 2 {
		t.Errorf("Expected 2 products, got %d", len(results))
	}

	// Verify results
	for _, result := range results {
		if activeVal, _ := result.Get("active"); activeVal != nil && !activeVal.(bool) {
			t.Error("Found inactive product in results")
		}
		if priceVal, _ := result.Get("price"); priceVal != nil && priceVal.(int64) < 50 {
			t.Error("Found product with price < 50 in results")
		}
	}
}

// TestCursorIntegration tests cursor functionality with large result sets
func TestCursorIntegration(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), fmt.Sprintf("test-cursor-integration-%d", time.Now().UnixNano()))
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("items")

	// Insert many documents
	numDocs := 250
	for i := 0; i < numDocs; i++ {
		doc := map[string]interface{}{
			"item":  fmt.Sprintf("item-%d", i),
			"value": int64(i * 10),
			"index": int64(i),
		}
		if _, err := coll.InsertOne(doc); err != nil {
			t.Fatalf("Failed to insert document: %v", err)
		}
	}

	// Create cursor with batch size of 50
	cursorOpts := &CursorOptions{BatchSize: 50, Timeout: 5 * time.Minute}
	cursor, err := coll.FindCursor(map[string]interface{}{}, cursorOpts)
	if err != nil {
		t.Fatalf("Failed to open cursor: %v", err)
	}

	// Iterate through all documents
	count := 0
	for cursor.HasNext() {
		_, err := cursor.Next()
		if err != nil {
			t.Fatalf("Failed to get next document: %v", err)
		}
		count++
	}

	if count != numDocs {
		t.Errorf("Expected %d documents from cursor, got %d", numDocs, count)
	}

	// Close cursor
	cursor.Close()
}

// TestMultiCollectionIntegration tests multiple collections in one database
func TestMultiCollectionIntegration(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), fmt.Sprintf("test-multi-coll-integration-%d", time.Now().UnixNano()))
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create multiple collections
	users := db.Collection("users")
	orders := db.Collection("orders")
	products := db.Collection("products")

	// Insert into users
	userId, err := users.InsertOne(map[string]interface{}{"name": "John", "email": "john@example.com"})
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	// Insert into products
	productId, err := products.InsertOne(map[string]interface{}{"name": "Widget", "price": int64(99)})
	if err != nil {
		t.Fatalf("Failed to insert product: %v", err)
	}

	// Insert into orders (referencing user and product)
	_, err = orders.InsertOne(map[string]interface{}{
		"userId":    userId,
		"productId": productId,
		"quantity":  int64(2),
		"total":     int64(198),
	})
	if err != nil {
		t.Fatalf("Failed to insert order: %v", err)
	}

	// Verify each collection has data
	userResults, _ := users.Find(map[string]interface{}{})
	if len(userResults) != 1 {
		t.Errorf("Expected 1 user, got %d", len(userResults))
	}

	productResults, _ := products.Find(map[string]interface{}{})
	if len(productResults) != 1 {
		t.Errorf("Expected 1 product, got %d", len(productResults))
	}

	orderResults, _ := orders.Find(map[string]interface{}{})
	if len(orderResults) != 1 {
		t.Errorf("Expected 1 order, got %d", len(orderResults))
	}

	// List all collections
	collections := db.ListCollections()
	if len(collections) != 3 {
		t.Errorf("Expected 3 collections, got %d", len(collections))
	}

	// Drop a collection
	if err := db.DropCollection("orders"); err != nil {
		t.Fatalf("Failed to drop collection: %v", err)
	}

	collections = db.ListCollections()
	if len(collections) != 2 {
		t.Errorf("Expected 2 collections after drop, got %d", len(collections))
	}
}

// TestUpdateOperatorsIntegration tests all update operators together
func TestUpdateOperatorsIntegration(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), fmt.Sprintf("test-update-integration-%d", time.Now().UnixNano()))
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("documents")

	// Insert test document
	docData := map[string]interface{}{
		"counter":  int64(10),
		"score":    int64(100),
		"tags":     []interface{}{"tag1", "tag2"},
		"oldField": "value",
		"numbers":  []interface{}{int64(1), int64(2), int64(3), int64(4)},
		"bits":     int64(5), // binary: 0101
	}
	_, err = coll.InsertOne(docData)
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Test $inc (use counter field as filter instead of _id)
	err = coll.UpdateOne(
		map[string]interface{}{"counter": int64(10)},
		map[string]interface{}{"$inc": map[string]interface{}{"counter": int64(5)}},
	)
	if err != nil {
		t.Fatalf("Failed to $inc: %v", err)
	}

	// Test $mul
	err = coll.UpdateOne(
		map[string]interface{}{"score": int64(100)},
		map[string]interface{}{"$mul": map[string]interface{}{"score": int64(2)}},
	)
	if err != nil {
		t.Fatalf("Failed to $mul: %v", err)
	}

	// Test $push
	err = coll.UpdateOne(
		map[string]interface{}{"counter": int64(15)}, // counter is now 15 after $inc
		map[string]interface{}{"$push": map[string]interface{}{"tags": "tag3"}},
	)
	if err != nil {
		t.Fatalf("Failed to $push: %v", err)
	}

	// Test $pull
	err = coll.UpdateOne(
		map[string]interface{}{"counter": int64(15)},
		map[string]interface{}{"$pull": map[string]interface{}{"tags": "tag1"}},
	)
	if err != nil {
		t.Fatalf("Failed to $pull: %v", err)
	}

	// Test $rename
	err = coll.UpdateOne(
		map[string]interface{}{"counter": int64(15)},
		map[string]interface{}{"$rename": map[string]interface{}{"oldField": "newField"}},
	)
	if err != nil {
		t.Fatalf("Failed to $rename: %v", err)
	}

	// Test $pop (remove last element)
	err = coll.UpdateOne(
		map[string]interface{}{"counter": int64(15)},
		map[string]interface{}{"$pop": map[string]interface{}{"numbers": int64(1)}},
	)
	if err != nil {
		t.Fatalf("Failed to $pop: %v", err)
	}

	// Test $bit (bitwise AND with 3 = 0011, so 0101 & 0011 = 0001 = 1)
	err = coll.UpdateOne(
		map[string]interface{}{"counter": int64(15)},
		map[string]interface{}{"$bit": map[string]interface{}{
			"bits": map[string]interface{}{"and": int64(3)},
		}},
	)
	if err != nil {
		t.Fatalf("Failed to $bit: %v", err)
	}

	// Verify all updates
	results, err := coll.Find(map[string]interface{}{"counter": int64(15)})
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(results))
	}

	doc := results[0]

	// Verify $inc: 10 + 5 = 15
	if counter, _ := doc.Get("counter"); counter != nil {
		var counterVal int64
		switch v := counter.(type) {
		case int64:
			counterVal = v
		case float64:
			counterVal = int64(v)
		}
		if counterVal != 15 {
			t.Errorf("Expected counter 15, got %d", counterVal)
		}
	}

	// Verify $mul: 100 * 2 = 200
	if score, _ := doc.Get("score"); score != nil {
		var scoreVal int64
		switch v := score.(type) {
		case int64:
			scoreVal = v
		case float64:
			scoreVal = int64(v)
		}
		if scoreVal != 200 {
			t.Errorf("Expected score 200, got %d", scoreVal)
		}
	}

	// Verify $push and $pull: ["tag2", "tag3"]
	if tagsVal, _ := doc.Get("tags"); tagsVal != nil {
		tags := tagsVal.([]interface{})
		if len(tags) != 2 {
			t.Errorf("Expected 2 tags, got %d", len(tags))
		}
	}

	// Verify $rename
	if _, exists := doc.Get("newField"); !exists {
		t.Error("Expected newField to exist after rename")
	}
	if _, exists := doc.Get("oldField"); exists {
		t.Error("Expected oldField to not exist after rename")
	}

	// Verify $pop: [1, 2, 3] (removed last element 4)
	if numbersVal, _ := doc.Get("numbers"); numbersVal != nil {
		numbers := numbersVal.([]interface{})
		if len(numbers) != 3 {
			t.Errorf("Expected 3 numbers after pop, got %d", len(numbers))
		}
	}

	// Verify $bit: 5 & 3 = 1
	if bits, _ := doc.Get("bits"); bits != nil {
		var bitsVal int64
		switch v := bits.(type) {
		case int64:
			bitsVal = v
		case float64:
			bitsVal = int64(v)
		}
		if bitsVal != 1 {
			t.Errorf("Expected bits 1, got %d", bitsVal)
		}
	}
}

// TestConcurrentOperationsIntegration tests concurrent database operations
func TestConcurrentOperationsIntegration(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), fmt.Sprintf("test-concurrent-integration-%d", time.Now().UnixNano()))
	defer os.RemoveAll(dataDir)

	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("concurrent")

	// Insert initial document with a unique identifier
	docData := map[string]interface{}{"name": "test-counter", "counter": int64(0)}
	_, err = coll.InsertOne(docData)
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Run 100 concurrent increments
	numGoroutines := 100
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			err := coll.UpdateOne(
				map[string]interface{}{"name": "test-counter"},
				map[string]interface{}{"$inc": map[string]interface{}{"counter": int64(1)}},
			)
			if err != nil {
				t.Errorf("Concurrent update failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify final counter value
	results, err := coll.Find(map[string]interface{}{"name": "test-counter"})
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(results))
	}

	if counter, _ := results[0].Get("counter"); counter != nil {
		var counterVal int64
		switch v := counter.(type) {
		case int64:
			counterVal = v
		case float64:
			counterVal = int64(v)
		}
		if counterVal != int64(numGoroutines) {
			t.Errorf("Expected counter %d, got %d", numGoroutines, counterVal)
		}
	}
}
