package database

import (
	"fmt"
	"os"
	"testing"
)

func BenchmarkCreatePartialIndex(b *testing.B) {
	dir := "./bench_partial_create"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("users")

	// Insert 1000 documents - 30% match filter
	for i := 0; i < 1000; i++ {
		coll.InsertOne(map[string]interface{}{
			"name":   fmt.Sprintf("user%d", i),
			"age":    20 + (i % 50),
			"active": i%10 < 3, // 30% active
		})
	}

	filter := map[string]interface{}{"active": true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.CreatePartialIndex("name", filter, false)
		if i < b.N-1 {
			coll.DropIndex("name_partial")
		}
	}
}

func BenchmarkPartialIndexInsert(b *testing.B) {
	dir := "./bench_partial_insert"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("products")

	// Create partial index
	filter := map[string]interface{}{"inStock": true}
	coll.CreatePartialIndex("sku", filter, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.InsertOne(map[string]interface{}{
			"sku":     fmt.Sprintf("SKU%d", i),
			"inStock": i%2 == 0, // 50% in stock
		})
	}
}

func BenchmarkPartialIndexInsertMatchingDocs(b *testing.B) {
	dir := "./bench_partial_insert_match"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("items")

	filter := map[string]interface{}{"active": true}
	coll.CreatePartialIndex("code", filter, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.InsertOne(map[string]interface{}{
			"code":   fmt.Sprintf("CODE%d", i),
			"active": true, // All match filter
		})
	}
}

func BenchmarkPartialIndexInsertNonMatchingDocs(b *testing.B) {
	dir := "./bench_partial_insert_nomatch"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("items")

	filter := map[string]interface{}{"active": true}
	coll.CreatePartialIndex("code", filter, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.InsertOne(map[string]interface{}{
			"code":   fmt.Sprintf("CODE%d", i),
			"active": false, // None match filter
		})
	}
}

func BenchmarkPartialIndexUpdate(b *testing.B) {
	dir := "./bench_partial_update"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("tasks")

	filter := map[string]interface{}{"completed": false}
	coll.CreatePartialIndex("priority", filter, false)

	// Pre-populate
	for i := 0; i < 1000; i++ {
		coll.InsertOne(map[string]interface{}{
			"task":      fmt.Sprintf("task%d", i),
			"priority":  i % 5,
			"completed": false,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := i % 1000
		coll.UpdateOne(
			map[string]interface{}{"task": fmt.Sprintf("task%d", idx)},
			map[string]interface{}{
				"$set": map[string]interface{}{
					"priority": (idx + 1) % 5,
				},
			},
		)
	}
}

func BenchmarkPartialIndexUpdateChangingFilter(b *testing.B) {
	dir := "./bench_partial_update_filter"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("docs")

	filter := map[string]interface{}{"published": true}
	coll.CreatePartialIndex("title", filter, false)

	// Pre-populate
	for i := 0; i < 1000; i++ {
		coll.InsertOne(map[string]interface{}{
			"doc":       fmt.Sprintf("doc%d", i),
			"title":     fmt.Sprintf("Title %d", i),
			"published": i%2 == 0,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := i % 1000
		// Toggle published status (moves docs in/out of index)
		coll.UpdateOne(
			map[string]interface{}{"doc": fmt.Sprintf("doc%d", idx)},
			map[string]interface{}{
				"$set": map[string]interface{}{
					"published": idx%2 != 0,
				},
			},
		)
	}
}

func BenchmarkPartialIndexDelete(b *testing.B) {
	dir := "./bench_partial_delete"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("logs")

	filter := map[string]interface{}{"level": "ERROR"}
	coll.CreatePartialIndex("timestamp", filter, false)

	// Pre-populate with more docs than we'll delete
	for i := 0; i < b.N+1000; i++ {
		coll.InsertOne(map[string]interface{}{
			"id":        i,
			"level":     "ERROR",
			"timestamp": i,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.DeleteOne(map[string]interface{}{"id": i})
	}
}

func BenchmarkPartialIndexComplexFilter(b *testing.B) {
	dir := "./bench_partial_complex"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("orders")

	// Complex filter with multiple conditions
	filter := map[string]interface{}{
		"status": "pending",
		"amount": map[string]interface{}{
			"$gt": 100,
		},
	}
	coll.CreatePartialIndex("orderNumber", filter, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.InsertOne(map[string]interface{}{
			"orderNumber": fmt.Sprintf("ORD%d", i),
			"status":      "pending",
			"amount":      50 + (i % 200), // Some match, some don't
		})
	}
}

func BenchmarkPartialIndexComparison(b *testing.B) {
	dir := "./bench_partial_comparison"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	// Collection with partial index
	collPartial := db.Collection("partial")
	filter := map[string]interface{}{"active": true}
	collPartial.CreatePartialIndex("email", filter, false)

	// Collection with full index
	collFull := db.Collection("full")
	collFull.CreateIndex("email", false)

	b.Run("Partial index (30% docs)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			collPartial.InsertOne(map[string]interface{}{
				"email":  fmt.Sprintf("user%d@example.com", i),
				"active": i%10 < 3, // 30% active
			})
		}
	})

	b.Run("Full index (100% docs)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			collFull.InsertOne(map[string]interface{}{
				"email":  fmt.Sprintf("user%d@example.com", i),
				"active": i%10 < 3,
			})
		}
	})
}

func BenchmarkPartialIndexMemoryUsage(b *testing.B) {
	dir := "./bench_partial_memory"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("large")

	// Create partial index that only indexes 10% of documents
	filter := map[string]interface{}{
		"priority": map[string]interface{}{
			"$gte": 9,
		},
	}
	coll.CreatePartialIndex("id", filter, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.InsertOne(map[string]interface{}{
			"id":       i,
			"priority": i % 10,
			"data":     fmt.Sprintf("Large data payload %d", i),
		})
	}
}
