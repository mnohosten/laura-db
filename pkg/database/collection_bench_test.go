package database

import (
	"fmt"
	"os"
	"testing"
)

// BenchmarkInsertOne benchmarks single document insertion
func BenchmarkInsertOne(b *testing.B) {
	testDir := "./bench_insert"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc := map[string]interface{}{
			"name":  fmt.Sprintf("User%d", i),
			"age":   25 + (i % 50),
			"email": fmt.Sprintf("user%d@example.com", i),
		}
		_, err := coll.InsertOne(doc)
		if err != nil {
			b.Fatalf("Insert failed: %v", err)
		}
	}
}

// BenchmarkFind benchmarks finding documents without index
func BenchmarkFindWithoutIndex(b *testing.B) {
	testDir := "./bench_find"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("bench")

	// Insert test data
	for i := 0; i < 1000; i++ {
		doc := map[string]interface{}{
			"name": fmt.Sprintf("User%d", i),
			"age":  25 + (i % 50),
		}
		coll.InsertOne(doc)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		age := 25 + (i % 50)
		filter := map[string]interface{}{
			"age": age,
		}
		_, err := coll.Find(filter)
		if err != nil {
			b.Fatalf("Find failed: %v", err)
		}
	}
}

// BenchmarkFindWithIndex benchmarks finding documents with index
func BenchmarkFindWithIndex(b *testing.B) {
	testDir := "./bench_find_index"
	os.RemoveAll(testDir) // Clean up first
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("bench")

	// Insert test data with unique values for all fields
	for i := 0; i < 1000; i++ {
		doc := map[string]interface{}{
			"user_id": i,  // Unique field
			"name":    fmt.Sprintf("User%d", i),
			"age":     i,  // Unique age values to avoid index duplicate issue
		}
		_, err := coll.InsertOne(doc)
		if err != nil {
			b.Fatalf("Failed to insert: %v", err)
		}
	}

	// Create index after data insertion to measure indexed query performance
	err = coll.CreateIndex("age", false)
	if err != nil {
		b.Fatalf("Failed to create index: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		age := i % 1000  // Query existing ages
		filter := map[string]interface{}{
			"age": age,
		}
		_, err := coll.Find(filter)
		if err != nil {
			b.Fatalf("Find failed: %v", err)
		}
	}
}

// BenchmarkUpdateOne benchmarks single document update
func BenchmarkUpdateOne(b *testing.B) {
	testDir := "./bench_update"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("bench")

	// Insert test data
	for i := 0; i < 100; i++ {
		doc := map[string]interface{}{
			"name": fmt.Sprintf("User%d", i),
			"age":  25,
		}
		coll.InsertOne(doc)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter := map[string]interface{}{
			"name": fmt.Sprintf("User%d", i%100),
		}
		update := map[string]interface{}{
			"$set": map[string]interface{}{
				"age": 30 + (i % 40),
			},
		}
		err := coll.UpdateOne(filter, update)
		if err != nil {
			b.Fatalf("Update failed: %v", err)
		}
	}
}

// BenchmarkAggregation benchmarks aggregation pipeline
func BenchmarkAggregation(b *testing.B) {
	testDir := "./bench_agg"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("bench")

	// Insert test data
	cities := []string{"NYC", "LA", "Chicago", "Boston", "Seattle"}
	for i := 0; i < 1000; i++ {
		doc := map[string]interface{}{
			"name":   fmt.Sprintf("User%d", i),
			"age":    25 + (i % 50),
			"city":   cities[i%len(cities)],
			"salary": 50000 + (i * 100),
		}
		coll.InsertOne(doc)
	}

	pipeline := []map[string]interface{}{
		{
			"$match": map[string]interface{}{
				"age": map[string]interface{}{"$gte": 30},
			},
		},
		{
			"$group": map[string]interface{}{
				"_id":       "$city",
				"avgSalary": map[string]interface{}{"$avg": "$salary"},
				"count":     map[string]interface{}{"$count": map[string]interface{}{}},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := coll.Aggregate(pipeline)
		if err != nil {
			b.Fatalf("Aggregation failed: %v", err)
		}
	}
}

// BenchmarkBulkInsert benchmarks bulk insertion
func BenchmarkBulkInsert(b *testing.B) {
	testDir := "./bench_bulk"
	defer os.RemoveAll(testDir)

	config := DefaultConfig(testDir)
	db, err := Open(config)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("bench")

	// Prepare docs
	docs := make([]map[string]interface{}, 100)
	for i := range docs {
		docs[i] = map[string]interface{}{
			"name": fmt.Sprintf("User%d", i),
			"age":  25 + (i % 50),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := coll.InsertMany(docs)
		if err != nil {
			b.Fatalf("Bulk insert failed: %v", err)
		}
	}
}
