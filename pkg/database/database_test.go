package database

import (
	"fmt"
	"os"
	"testing"

	"github.com/mnohosten/laura-db/pkg/query"
)

func TestDatabaseOpen(t *testing.T) {
	dir := "./test_db"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	db, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if db == nil {
		t.Fatal("Expected non-nil database")
	}
}

func TestCollectionOperations(t *testing.T) {
	dir := "./test_db_coll"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	// Get collection
	users := db.Collection("users")
	if users == nil {
		t.Fatal("Expected non-nil collection")
	}

	if users.Name() != "users" {
		t.Errorf("Expected collection name 'users', got %s", users.Name())
	}
}

func TestInsertOne(t *testing.T) {
	dir := "./test_db_insert"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	users := db.Collection("users")

	id, err := users.InsertOne(map[string]interface{}{
		"name":  "Alice",
		"age":   int64(30),
		"email": "alice@example.com",
	})

	if err != nil {
		t.Fatalf("InsertOne failed: %v", err)
	}

	if id == "" {
		t.Error("Expected non-empty ID")
	}
}

func TestInsertMany(t *testing.T) {
	dir := "./test_db_insert_many"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	users := db.Collection("users")

	docs := []map[string]interface{}{
		{"name": "Alice", "age": int64(30)},
		{"name": "Bob", "age": int64(25)},
		{"name": "Charlie", "age": int64(35)},
	}

	ids, err := users.InsertMany(docs)
	if err != nil {
		t.Fatalf("InsertMany failed: %v", err)
	}

	if len(ids) != 3 {
		t.Errorf("Expected 3 IDs, got %d", len(ids))
	}
}

func TestFind(t *testing.T) {
	dir := "./test_db_find"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	users := db.Collection("users")

	// Insert test data
	users.InsertOne(map[string]interface{}{"name": "Alice", "age": int64(30)})
	users.InsertOne(map[string]interface{}{"name": "Bob", "age": int64(25)})
	users.InsertOne(map[string]interface{}{"name": "Charlie", "age": int64(35)})

	// Find all
	all, err := users.Find(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("Expected 3 documents, got %d", len(all))
	}

	// Find with filter
	results, _ := users.Find(map[string]interface{}{
		"age": map[string]interface{}{"$gte": int64(30)},
	})

	if len(results) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(results))
	}
}

func TestFindOne(t *testing.T) {
	dir := "./test_db_find_one"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	users := db.Collection("users")
	users.InsertOne(map[string]interface{}{"name": "Alice", "age": int64(30)})

	doc, err := users.FindOne(map[string]interface{}{"name": "Alice"})
	if err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	if doc == nil {
		t.Fatal("Expected non-nil document")
	}

	name, _ := doc.Get("name")
	if name.(string) != "Alice" {
		t.Errorf("Expected 'Alice', got %v", name)
	}
}

func TestFindWithOptions(t *testing.T) {
	dir := "./test_db_find_options"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	users := db.Collection("users")

	users.InsertOne(map[string]interface{}{"name": "Alice", "age": int64(30), "email": "alice@example.com"})
	users.InsertOne(map[string]interface{}{"name": "Bob", "age": int64(25), "email": "bob@example.com"})
	users.InsertOne(map[string]interface{}{"name": "Charlie", "age": int64(35), "email": "charlie@example.com"})

	// With projection
	results, _ := users.FindWithOptions(
		map[string]interface{}{},
		&QueryOptions{
			Projection: map[string]bool{
				"name": true,
				"age":  true,
			},
			Sort:  []query.SortField{{Field: "age", Ascending: false}},
			Limit: 2,
		},
	)

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Verify projection
	if results[0].Has("email") {
		t.Error("Expected email to be excluded")
	}

	// Verify sort (descending age)
	age1, _ := results[0].Get("age")
	age2, _ := results[1].Get("age")
	if age1.(int64) < age2.(int64) {
		t.Error("Results not sorted correctly")
	}
}

func TestUpdateOne(t *testing.T) {
	dir := "./test_db_update"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	users := db.Collection("users")
	users.InsertOne(map[string]interface{}{"name": "Alice", "age": int64(30)})

	err := users.UpdateOne(
		map[string]interface{}{"name": "Alice"},
		map[string]interface{}{
			"$set": map[string]interface{}{
				"age": int64(31),
			},
		},
	)

	if err != nil {
		t.Fatalf("UpdateOne failed: %v", err)
	}

	// Verify update
	doc, _ := users.FindOne(map[string]interface{}{"name": "Alice"})
	age, _ := doc.Get("age")
	if age.(int64) != 31 {
		t.Errorf("Expected age 31, got %v", age)
	}
}

func TestUpdateMany(t *testing.T) {
	dir := "./test_db_update_many"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	users := db.Collection("users")
	users.InsertOne(map[string]interface{}{"status": "pending", "value": int64(10)})
	users.InsertOne(map[string]interface{}{"status": "pending", "value": int64(20)})
	users.InsertOne(map[string]interface{}{"status": "active", "value": int64(30)})

	count, err := users.UpdateMany(
		map[string]interface{}{"status": "pending"},
		map[string]interface{}{
			"$set": map[string]interface{}{
				"status": "active",
			},
		},
	)

	if err != nil {
		t.Fatalf("UpdateMany failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 updates, got %d", count)
	}

	// Verify all are active now
	active, _ := users.Find(map[string]interface{}{"status": "active"})
	if len(active) != 3 {
		t.Errorf("Expected 3 active documents, got %d", len(active))
	}
}

func TestDeleteOne(t *testing.T) {
	dir := "./test_db_delete"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	users := db.Collection("users")
	users.InsertOne(map[string]interface{}{"name": "Alice"})
	users.InsertOne(map[string]interface{}{"name": "Bob"})

	err := users.DeleteOne(map[string]interface{}{"name": "Alice"})
	if err != nil {
		t.Fatalf("DeleteOne failed: %v", err)
	}

	// Verify deletion
	count, _ := users.Count(map[string]interface{}{})
	if count != 1 {
		t.Errorf("Expected 1 document remaining, got %d", count)
	}

	_, err = users.FindOne(map[string]interface{}{"name": "Alice"})
	if err != ErrDocumentNotFound {
		t.Error("Expected document to be deleted")
	}
}

func TestDeleteMany(t *testing.T) {
	dir := "./test_db_delete_many"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	users := db.Collection("users")
	users.InsertOne(map[string]interface{}{"age": int64(20)})
	users.InsertOne(map[string]interface{}{"age": int64(25)})
	users.InsertOne(map[string]interface{}{"age": int64(30)})

	count, err := users.DeleteMany(map[string]interface{}{
		"age": map[string]interface{}{"$lt": int64(28)},
	})

	if err != nil {
		t.Fatalf("DeleteMany failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 deletions, got %d", count)
	}

	remaining, _ := users.Count(map[string]interface{}{})
	if remaining != 1 {
		t.Errorf("Expected 1 document remaining, got %d", remaining)
	}
}

func TestCreateIndex(t *testing.T) {
	dir := "./test_db_index"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	users := db.Collection("users")
	users.InsertOne(map[string]interface{}{"email": "alice@example.com", "name": "Alice"})

	err := users.CreateIndex("email", true)
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}

	indexes := users.ListIndexes()
	if len(indexes) < 2 { // _id index + email index
		t.Errorf("Expected at least 2 indexes, got %d", len(indexes))
	}
}

func TestIndexUniqueness(t *testing.T) {
	dir := "./test_db_index_unique"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	users := db.Collection("users")
	users.CreateIndex("email", true)

	users.InsertOne(map[string]interface{}{"email": "alice@example.com", "name": "Alice"})

	// Try to insert duplicate
	_, err := users.InsertOne(map[string]interface{}{"email": "alice@example.com", "name": "Alice2"})
	if err == nil {
		t.Error("Expected error for duplicate email in unique index")
	}
}

func TestAggregate(t *testing.T) {
	dir := "./test_db_aggregate"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	sales := db.Collection("sales")

	sales.InsertOne(map[string]interface{}{"category": "A", "price": 10.0})
	sales.InsertOne(map[string]interface{}{"category": "A", "price": 20.0})
	sales.InsertOne(map[string]interface{}{"category": "B", "price": 30.0})

	results, err := sales.Aggregate([]map[string]interface{}{
		{
			"$group": map[string]interface{}{
				"_id": "$category",
				"total": map[string]interface{}{
					"$sum": "$price",
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("Aggregate failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(results))
	}
}

func TestCount(t *testing.T) {
	dir := "./test_db_count"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	users := db.Collection("users")

	users.InsertOne(map[string]interface{}{"age": int64(25)})
	users.InsertOne(map[string]interface{}{"age": int64(30)})
	users.InsertOne(map[string]interface{}{"age": int64(35)})

	total, _ := users.Count(map[string]interface{}{})
	if total != 3 {
		t.Errorf("Expected count 3, got %d", total)
	}

	filtered, _ := users.Count(map[string]interface{}{
		"age": map[string]interface{}{"$gte": int64(30)},
	})
	if filtered != 2 {
		t.Errorf("Expected count 2, got %d", filtered)
	}
}

func TestListCollections(t *testing.T) {
	dir := "./test_db_list_coll"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	db.Collection("users")
	db.Collection("products")
	db.Collection("orders")

	collections := db.ListCollections()
	if len(collections) != 3 {
		t.Errorf("Expected 3 collections, got %d", len(collections))
	}
}

func TestDropCollection(t *testing.T) {
	dir := "./test_db_drop_coll"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	users := db.Collection("users")
	users.InsertOne(map[string]interface{}{"name": "Alice"})

	err := db.DropCollection("users")
	if err != nil {
		t.Fatalf("DropCollection failed: %v", err)
	}

	collections := db.ListCollections()
	if len(collections) != 0 {
		t.Errorf("Expected 0 collections after drop, got %d", len(collections))
	}
}

func TestCreateCollection(t *testing.T) {
	dir := "./test_db_create_coll"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	// Create a new collection
	coll, err := db.CreateCollection("users")
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}

	if coll == nil {
		t.Fatal("Expected non-nil collection")
	}

	if coll.Name() != "users" {
		t.Errorf("Expected collection name 'users', got %s", coll.Name())
	}

	// Try to create duplicate collection
	_, err = db.CreateCollection("users")
	if err == nil {
		t.Error("Expected error when creating duplicate collection")
	}
}

func TestRenameCollection(t *testing.T) {
	dir := "./test_db_rename_coll"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	// Create and populate collection
	users := db.Collection("users")
	users.InsertOne(map[string]interface{}{"name": "Alice"})

	// Rename collection
	err := db.RenameCollection("users", "people")
	if err != nil {
		t.Fatalf("RenameCollection failed: %v", err)
	}

	// Verify old name doesn't exist
	collections := db.ListCollections()
	for _, name := range collections {
		if name == "users" {
			t.Error("Old collection name still exists")
		}
	}

	// Verify new name exists and has data
	people := db.Collection("people")
	count, _ := people.Count(map[string]interface{}{})
	if count != 1 {
		t.Errorf("Expected 1 document in renamed collection, got %d", count)
	}

	// Test renaming non-existent collection
	err = db.RenameCollection("nonexistent", "other")
	if err == nil {
		t.Error("Expected error when renaming non-existent collection")
	}

	// Test renaming to existing name
	db.Collection("existing")
	err = db.RenameCollection("people", "existing")
	if err == nil {
		t.Error("Expected error when renaming to existing collection name")
	}
}

func TestTransactionMethods(t *testing.T) {
	dir := "./test_db_transactions"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	// Test BeginTransaction
	txn := db.BeginTransaction()
	if txn == nil {
		t.Fatal("Expected non-nil transaction")
	}

	// Test CommitTransaction
	err := db.CommitTransaction(txn)
	if err != nil {
		t.Fatalf("CommitTransaction failed: %v", err)
	}

	// Test AbortTransaction
	txn2 := db.BeginTransaction()
	err = db.AbortTransaction(txn2)
	if err != nil {
		t.Fatalf("AbortTransaction failed: %v", err)
	}
}

func TestDatabaseStats(t *testing.T) {
	dir := "./test_db_stats"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	// Create collections and add data
	users := db.Collection("users")
	users.InsertOne(map[string]interface{}{"name": "Alice"})
	users.InsertOne(map[string]interface{}{"name": "Bob"})

	products := db.Collection("products")
	products.InsertOne(map[string]interface{}{"name": "Widget"})

	// Get stats
	stats := db.Stats()
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	// Verify stats structure
	if stats["name"] != "default" {
		t.Errorf("Expected database name 'default', got %v", stats["name"])
	}

	collections, ok := stats["collections"].(int)
	if !ok || collections != 2 {
		t.Errorf("Expected 2 collections, got %v", stats["collections"])
	}

	if stats["collection_stats"] == nil {
		t.Error("Expected collection_stats in stats")
	}

	if stats["active_transactions"] == nil {
		t.Error("Expected active_transactions in stats")
	}

	if stats["storage_stats"] == nil {
		t.Error("Expected storage_stats in stats")
	}
}

func TestAnalyze(t *testing.T) {
	dir := "./test_db_analyze"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	users := db.Collection("users")

	// Insert test data
	for i := 0; i < 100; i++ {
		users.InsertOne(map[string]interface{}{
			"name": fmt.Sprintf("User%d", i),
			"age":  int64(20 + (i % 50)),
		})
	}

	// Create index
	users.CreateIndex("age", false)

	// Run analyze (no return value)
	users.Analyze()

	// Verify index statistics were updated
	indexes := users.ListIndexes()
	if len(indexes) == 0 {
		t.Fatal("Expected at least one index")
	}

	// Find the age index in the list
	var ageIndexFound bool
	for _, idx := range indexes {
		if field, ok := idx["field_path"].(string); ok && field == "age" {
			ageIndexFound = true
			// Check that size is present (btree size)
			if _, ok := idx["size"]; !ok {
				t.Error("Expected size in index stats")
			}
			break
		}
	}

	if !ageIndexFound {
		t.Error("Expected to find age index in list")
	}
}

func TestExplain(t *testing.T) {
	dir := "./test_db_explain"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	users := db.Collection("users")

	// Insert test data
	users.InsertOne(map[string]interface{}{"name": "Alice", "age": int64(30)})
	users.InsertOne(map[string]interface{}{"name": "Bob", "age": int64(25)})

	// Create index
	users.CreateIndex("age", false)

	// Explain query
	explanation := users.Explain(map[string]interface{}{
		"age": map[string]interface{}{"$gte": int64(25)},
	})

	if explanation == nil {
		t.Fatal("Expected non-nil explanation")
	}

	// Verify explanation structure (matches QueryPlan.Explain output)
	if explanation["estimatedCost"] == nil {
		t.Error("Expected estimatedCost in explanation")
	}

	if explanation["useIndex"] == nil {
		t.Error("Expected useIndex in explanation")
	}

	if explanation["collection"] == nil {
		t.Error("Expected collection in explanation")
	}

	if explanation["totalDocuments"] == nil {
		t.Error("Expected totalDocuments in explanation")
	}
}
