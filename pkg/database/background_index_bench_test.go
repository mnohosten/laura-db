package database

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func BenchmarkCreateIndexForeground(b *testing.B) {
	dir := "./bench_foreground"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("test")

	// Insert 10000 documents
	for i := 0; i < 10000; i++ {
		coll.InsertOne(map[string]interface{}{
			"id":    i,
			"value": i,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		indexName := fmt.Sprintf("value_%d_1", i)
		// Create foreground index (blocks until complete)
		coll.CreateIndexWithBackground("value_"+fmt.Sprint(i), false, false)
		// Drop index for next iteration
		if i < b.N-1 {
			coll.DropIndex(indexName)
		}
	}
}

func BenchmarkCreateIndexBackground(b *testing.B) {
	dir := "./bench_background"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("test")

	// Insert 10000 documents
	for i := 0; i < 10000; i++ {
		coll.InsertOne(map[string]interface{}{
			"id":    i,
			"value": i,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		indexName := fmt.Sprintf("value_%d_1", i)
		// Create background index (returns immediately)
		coll.CreateIndexWithBackground("value_"+fmt.Sprint(i), false, true)

		// Wait for build to complete before next iteration
		for {
			progress, _ := coll.GetIndexBuildProgress(indexName)
			if progress["state"].(string) == "ready" {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}

		// Drop index for next iteration
		if i < b.N-1 {
			coll.DropIndex(indexName)
		}
	}
}

func BenchmarkBackgroundIndexConcurrentInserts(b *testing.B) {
	dir := "./bench_concurrent_inserts"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("test")

	// Insert initial documents
	for i := 0; i < 5000; i++ {
		coll.InsertOne(map[string]interface{}{
			"id":    i,
			"value": i,
		})
	}

	// Create background index
	coll.CreateIndexWithBackground("value", false, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Insert documents while index is building
		coll.InsertOne(map[string]interface{}{
			"id":    5000 + i,
			"value": 5000 + i,
		})
	}
}

func BenchmarkCompoundIndexForeground(b *testing.B) {
	dir := "./bench_compound_foreground"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("test")

	// Insert documents
	for i := 0; i < 1000; i++ {
		coll.InsertOne(map[string]interface{}{
			"field1": i,
			"field2": i * 2,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.CreateCompoundIndexWithBackground([]string{"field1", "field2"}, false, false)
		if i < b.N-1 {
			coll.DropIndex("field1_field2_1")
		}
	}
}

func BenchmarkCompoundIndexBackground(b *testing.B) {
	dir := "./bench_compound_background"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("test")

	// Insert documents
	for i := 0; i < 1000; i++ {
		coll.InsertOne(map[string]interface{}{
			"field1": i,
			"field2": i * 2,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.CreateCompoundIndexWithBackground([]string{"field1", "field2"}, false, true)

		// Wait for completion
		for {
			progress, _ := coll.GetIndexBuildProgress("field1_field2_1")
			if progress["state"].(string) == "ready" {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}

		if i < b.N-1 {
			coll.DropIndex("field1_field2_1")
		}
	}
}
