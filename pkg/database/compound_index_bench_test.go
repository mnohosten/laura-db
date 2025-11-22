package database

import (
	"fmt"
	"os"
	"testing"
)

func BenchmarkCompoundIndex(b *testing.B) {
	dir := "./bench_compound_idx"
	defer os.RemoveAll(dir)

	db, err := Open(DefaultConfig(dir))
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("users")

	// Create compound index on city and age
	err = coll.CreateCompoundIndex([]string{"city", "age"}, false)
	if err != nil {
		b.Fatalf("Failed to create compound index: %v", err)
	}

	// Insert 1,000 documents with unique (city, age) combinations
	cities := []string{"NYC", "Boston", "Seattle", "SF", "LA", "Chicago", "Austin", "Denver"}
	for i := 0; i < 1000; i++ {
		city := cities[i%len(cities)]
		// Ensure unique (city, age) combinations by using unique age per city
		age := int64(20 + i)
		_, err := coll.InsertOne(map[string]interface{}{
			"name":   fmt.Sprintf("User%d", i),
			"city":   city,
			"age":    age,
			"salary": int64(50000 + (i * 1000)),
		})
		if err != nil {
			b.Fatalf("Failed to insert document: %v", err)
		}
	}

	b.Run("Full compound match", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, _ := coll.Find(map[string]interface{}{
				"city": "NYC",
				"age":  int64(30),
			})
			_ = results
		}
	})

	b.Run("Prefix match - first field only", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, _ := coll.Find(map[string]interface{}{
				"city": "NYC",
			})
			_ = results
		}
	})

	b.Run("No index - second field only", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, _ := coll.Find(map[string]interface{}{
				"age": int64(30),
			})
			_ = results
		}
	})
}

func BenchmarkCompoundVsSingleIndex(b *testing.B) {
	// Test compound index (city, age)
	b.Run("Compound index", func(b *testing.B) {
		dir := "./bench_compound"
		defer os.RemoveAll(dir)

		db, _ := Open(DefaultConfig(dir))
		defer db.Close()

		coll := db.Collection("users")
		coll.CreateCompoundIndex([]string{"city", "age"}, false)

		// Insert 1,000 documents with unique combinations
		cities := []string{"NYC", "Boston", "Seattle", "SF"}
		for i := 0; i < 1000; i++ {
			coll.InsertOne(map[string]interface{}{
				"city": cities[i%len(cities)],
				"age":  int64(20 + i),
				"name": fmt.Sprintf("User%d", i),
			})
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, _ := coll.Find(map[string]interface{}{
				"city": "NYC",
				"age":  int64(30),
			})
			_ = results
		}
	})

	// Test single index on city only
	b.Run("Single index on city", func(b *testing.B) {
		dir := "./bench_single"
		defer os.RemoveAll(dir)

		db, _ := Open(DefaultConfig(dir))
		defer db.Close()

		coll := db.Collection("users")
		coll.CreateIndex("city", false)

		// Insert 1,000 documents with unique combinations
		cities := []string{"NYC", "Boston", "Seattle", "SF"}
		for i := 0; i < 1000; i++ {
			coll.InsertOne(map[string]interface{}{
				"city": cities[i%len(cities)],
				"age":  int64(20 + i),
				"name": fmt.Sprintf("User%d", i),
			})
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, _ := coll.Find(map[string]interface{}{
				"city": "NYC",
				"age":  int64(30),
			})
			_ = results
		}
	})

	// Test no index
	b.Run("No index", func(b *testing.B) {
		dir := "./bench_noindex"
		defer os.RemoveAll(dir)

		db, _ := Open(DefaultConfig(dir))
		defer db.Close()

		coll := db.Collection("users")

		// Insert 1,000 documents with unique combinations
		cities := []string{"NYC", "Boston", "Seattle", "SF"}
		for i := 0; i < 1000; i++ {
			coll.InsertOne(map[string]interface{}{
				"city": cities[i%len(cities)],
				"age":  int64(20 + i),
				"name": fmt.Sprintf("User%d", i),
			})
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, _ := coll.Find(map[string]interface{}{
				"city": "NYC",
				"age":  int64(30),
			})
			_ = results
		}
	})
}

func BenchmarkCompoundIndexUpdates(b *testing.B) {
	dir := "./bench_compound_update"
	defer os.RemoveAll(dir)

	db, err := Open(DefaultConfig(dir))
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("users")

	// Create compound index
	err = coll.CreateCompoundIndex([]string{"city", "age"}, false)
	if err != nil {
		b.Fatalf("Failed to create compound index: %v", err)
	}

	// Insert 1,000 documents
	for i := 0; i < 1000; i++ {
		coll.InsertOne(map[string]interface{}{
			"name": fmt.Sprintf("User%d", i),
			"city": "NYC",
			"age":  int64(25),
		})
	}

	b.Run("Update indexed field", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			idx := i % 1000
			coll.UpdateOne(
				map[string]interface{}{"name": fmt.Sprintf("User%d", idx)},
				map[string]interface{}{"$set": map[string]interface{}{"age": int64(26 + (i % 10))}},
			)
		}
	})
}

func BenchmarkCompoundIndexThreeFields(b *testing.B) {
	dir := "./bench_compound_3field"
	defer os.RemoveAll(dir)

	db, err := Open(DefaultConfig(dir))
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("employees")

	// Create 3-field compound index
	err = coll.CreateCompoundIndex([]string{"department", "level", "salary"}, false)
	if err != nil {
		b.Fatalf("Failed to create compound index: %v", err)
	}

	// Insert 1,000 documents with unique (department, level, salary) combinations
	departments := []string{"Engineering", "Sales", "Marketing", "HR"}
	levels := []string{"Junior", "Mid", "Senior", "Lead"}
	for i := 0; i < 1000; i++ {
		coll.InsertOne(map[string]interface{}{
			"name":       fmt.Sprintf("Employee%d", i),
			"department": departments[i%len(departments)],
			"level":      levels[i%len(levels)],
			"salary":     int64(50000 + (i * 1000)),
		})
	}

	b.Run("Full 3-field match", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, _ := coll.Find(map[string]interface{}{
				"department": "Engineering",
				"level":      "Senior",
				"salary":     int64(100000),
			})
			_ = results
		}
	})

	b.Run("2-field prefix", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, _ := coll.Find(map[string]interface{}{
				"department": "Engineering",
				"level":      "Senior",
			})
			_ = results
		}
	})

	b.Run("1-field prefix", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, _ := coll.Find(map[string]interface{}{
				"department": "Engineering",
			})
			_ = results
		}
	})
}
