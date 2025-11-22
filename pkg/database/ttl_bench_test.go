package database

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func BenchmarkCreateTTLIndex(b *testing.B) {
	dir := "./bench_ttl_create"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("bench_sessions")

	// Insert 1000 documents
	now := time.Now()
	for i := 0; i < 1000; i++ {
		coll.InsertOne(map[string]interface{}{
			"user":      fmt.Sprintf("user%d", i),
			"createdAt": now.Add(time.Duration(i) * time.Second),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.CreateTTLIndex("createdAt", 3600)
		if i < b.N-1 {
			coll.DropIndex("createdAt_ttl")
		}
	}
}

func BenchmarkTTLInsert(b *testing.B) {
	dir := "./bench_ttl_insert"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("bench_logs")

	// Create TTL index first
	coll.CreateTTLIndex("timestamp", 300)

	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.InsertOne(map[string]interface{}{
			"message":   fmt.Sprintf("log%d", i),
			"timestamp": now,
		})
	}
}

func BenchmarkTTLUpdate(b *testing.B) {
	dir := "./bench_ttl_update"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("bench_events")
	coll.CreateTTLIndex("updatedAt", 600)

	// Insert initial documents
	now := time.Now()
	for i := 0; i < 1000; i++ {
		coll.InsertOne(map[string]interface{}{
			"event":     fmt.Sprintf("event%d", i),
			"updatedAt": now,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := i % 1000
		coll.UpdateOne(
			map[string]interface{}{"event": fmt.Sprintf("event%d", idx)},
			map[string]interface{}{
				"$set": map[string]interface{}{
					"updatedAt": now.Add(time.Duration(i) * time.Second),
				},
			},
		)
	}
}

func BenchmarkTTLCleanup(b *testing.B) {
	dir := "./bench_ttl_cleanup"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("bench_temp")
	coll.CreateTTLIndex("expiresAt", 1)

	// Insert mix of expired and non-expired documents
	pastTime := time.Now().Add(-10 * time.Second)
	futureTime := time.Now().Add(100 * time.Second)

	for i := 0; i < 500; i++ {
		coll.InsertOne(map[string]interface{}{
			"data":      fmt.Sprintf("expired%d", i),
			"expiresAt": pastTime,
		})
	}

	for i := 0; i < 500; i++ {
		coll.InsertOne(map[string]interface{}{
			"data":      fmt.Sprintf("valid%d", i),
			"expiresAt": futureTime,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.CleanupExpiredDocuments()

		// Re-insert expired docs for next iteration
		if i < b.N-1 {
			for j := 0; j < 500; j++ {
				coll.InsertOne(map[string]interface{}{
					"data":      fmt.Sprintf("expired%d", j),
					"expiresAt": pastTime,
				})
			}
		}
	}
}

func BenchmarkTTLCleanupLargeDataset(b *testing.B) {
	dir := "./bench_ttl_large"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("bench_large")
	coll.CreateTTLIndex("timestamp", 60)

	// Insert 10,000 documents
	now := time.Now()
	expiredTime := now.Add(-120 * time.Second)

	for i := 0; i < 9000; i++ {
		coll.InsertOne(map[string]interface{}{
			"id":        i,
			"timestamp": now,
		})
	}

	for i := 9000; i < 10000; i++ {
		coll.InsertOne(map[string]interface{}{
			"id":        i,
			"timestamp": expiredTime,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.CleanupExpiredDocuments()

		// Re-insert expired docs
		if i < b.N-1 {
			for j := 9000; j < 10000; j++ {
				coll.InsertOne(map[string]interface{}{
					"id":        j,
					"timestamp": expiredTime,
				})
			}
		}
	}
}

func BenchmarkTTLGetExpiredDocuments(b *testing.B) {
	dir := "./bench_ttl_get_expired"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("bench_check")
	coll.CreateTTLIndex("createdAt", 30)

	// Insert documents with various timestamps
	now := time.Now()
	for i := 0; i < 5000; i++ {
		timestamp := now.Add(time.Duration(i-2500) * time.Second)
		coll.InsertOne(map[string]interface{}{
			"id":        i,
			"createdAt": timestamp,
		})
	}

	ttlIdx := coll.ttlIndexes["createdAt_ttl"]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ttlIdx.GetExpiredDocuments(now)
	}
}

func BenchmarkTTLMultipleIndexes(b *testing.B) {
	dir := "./bench_ttl_multiple"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("bench_multi")

	// Create multiple TTL indexes
	coll.CreateTTLIndex("createdAt", 300)
	coll.CreateTTLIndex("updatedAt", 600)
	coll.CreateTTLIndex("expiresAt", 900)

	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.InsertOne(map[string]interface{}{
			"id":        i,
			"createdAt": now,
			"updatedAt": now,
			"expiresAt": now.Add(100 * time.Second),
		})
	}
}

func BenchmarkTTLDelete(b *testing.B) {
	dir := "./bench_ttl_delete"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("bench_delete")
	coll.CreateTTLIndex("timestamp", 60)

	// Pre-populate for each iteration
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Insert document
		coll.InsertOne(map[string]interface{}{
			"name":      fmt.Sprintf("doc%d", i),
			"timestamp": now,
		})

		// Delete it
		b.StopTimer()
		coll.DeleteOne(map[string]interface{}{"name": fmt.Sprintf("doc%d", i)})
		b.StartTimer()
	}
}

func BenchmarkTTLIndexComparisonWithout(b *testing.B) {
	dir := "./bench_ttl_comparison"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	collWithTTL := db.Collection("with_ttl")
	collWithoutTTL := db.Collection("without_ttl")

	// Create TTL index on one collection
	collWithTTL.CreateTTLIndex("timestamp", 300)

	now := time.Now()

	b.Run("With TTL Index", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			collWithTTL.InsertOne(map[string]interface{}{
				"id":        i,
				"timestamp": now,
			})
		}
	})

	b.Run("Without TTL Index", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			collWithoutTTL.InsertOne(map[string]interface{}{
				"id":        i,
				"timestamp": now,
			})
		}
	})
}
