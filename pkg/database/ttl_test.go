package database

import (
	"os"
	"testing"
	"time"
)

func TestCreateTTLIndex(t *testing.T) {
	dir := "./test_ttl_create"
	defer os.RemoveAll(dir)

	db, err := Open(DefaultConfig(dir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("sessions")

	// Insert a document with a timestamp
	createdAt := time.Now()
	_, err = coll.InsertOne(map[string]interface{}{
		"user":      "alice",
		"createdAt": createdAt,
	})
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Create TTL index with 60 second expiration
	err = coll.CreateTTLIndex("createdAt", 60)
	if err != nil {
		t.Fatalf("Failed to create TTL index: %v", err)
	}

	// Verify index exists
	indexes := coll.ListIndexes()
	found := false
	for _, idx := range indexes {
		if idx["name"] == "createdAt_ttl" {
			found = true
			if idx["type"] != "ttl" {
				t.Errorf("Expected type ttl, got %v", idx["type"])
			}
			if idx["ttlSeconds"] != int64(60) {
				t.Errorf("Expected ttlSeconds 60, got %v", idx["ttlSeconds"])
			}
			break
		}
	}

	if !found {
		t.Error("TTL index not found in index list")
	}
}

func TestTTLIndexDuplicateError(t *testing.T) {
	dir := "./test_ttl_duplicate"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("data")

	// Create first TTL index
	err := coll.CreateTTLIndex("timestamp", 300)
	if err != nil {
		t.Fatalf("Failed to create first TTL index: %v", err)
	}

	// Try to create duplicate
	err = coll.CreateTTLIndex("timestamp", 300)
	if err == nil {
		t.Error("Expected error when creating duplicate TTL index")
	}
}

func TestTTLIndexMaintenance(t *testing.T) {
	dir := "./test_ttl_maintenance"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("logs")

	// Create TTL index first
	err := coll.CreateTTLIndex("timestamp", 5)
	if err != nil {
		t.Fatalf("Failed to create TTL index: %v", err)
	}

	// Insert document
	now := time.Now()
	_, err = coll.InsertOne(map[string]interface{}{
		"message":   "test log",
		"timestamp": now,
	})
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Verify TTL index has the document
	if len(coll.ttlIndexes) != 1 {
		t.Fatalf("Expected 1 TTL index, got %d", len(coll.ttlIndexes))
	}

	ttlIdx := coll.ttlIndexes["timestamp_ttl"]
	if ttlIdx.Count() != 1 {
		t.Errorf("Expected 1 document in TTL index, got %d", ttlIdx.Count())
	}

	// Update the timestamp
	newTime := now.Add(10 * time.Second)
	err = coll.UpdateOne(
		map[string]interface{}{"message": "test log"},
		map[string]interface{}{
			"$set": map[string]interface{}{
				"timestamp": newTime,
			},
		},
	)
	if err != nil {
		t.Fatalf("Failed to update document: %v", err)
	}

	// Verify TTL index still has 1 document
	if ttlIdx.Count() != 1 {
		t.Errorf("Expected 1 document in TTL index after update, got %d", ttlIdx.Count())
	}

	// Delete the document
	err = coll.DeleteOne(map[string]interface{}{"message": "test log"})
	if err != nil {
		t.Fatalf("Failed to delete document: %v", err)
	}

	// Verify TTL index is empty
	if ttlIdx.Count() != 0 {
		t.Errorf("Expected 0 documents in TTL index after delete, got %d", ttlIdx.Count())
	}
}

func TestTTLExpirationDetection(t *testing.T) {
	dir := "./test_ttl_expiration"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("temp")

	// Create TTL index with 2 second expiration
	err := coll.CreateTTLIndex("expiresAt", 2)
	if err != nil {
		t.Fatalf("Failed to create TTL index: %v", err)
	}

	// Insert documents with different timestamps
	pastTime := time.Now().Add(-5 * time.Second)    // Already expired
	currentTime := time.Now()                       // Will expire in 2 seconds
	futureTime := time.Now().Add(10 * time.Second)  // Won't expire for a while

	coll.InsertOne(map[string]interface{}{
		"name":      "expired",
		"expiresAt": pastTime,
	})

	coll.InsertOne(map[string]interface{}{
		"name":      "current",
		"expiresAt": currentTime,
	})

	coll.InsertOne(map[string]interface{}{
		"name":      "future",
		"expiresAt": futureTime,
	})

	// Get TTL index
	ttlIdx := coll.ttlIndexes["expiresAt_ttl"]
	if ttlIdx.Count() != 3 {
		t.Fatalf("Expected 3 documents in TTL index, got %d", ttlIdx.Count())
	}

	// Check for expired documents
	expiredDocs := ttlIdx.GetExpiredDocuments(time.Now())

	// Should find at least the one that's already expired
	if len(expiredDocs) < 1 {
		t.Errorf("Expected at least 1 expired document, got %d", len(expiredDocs))
	}
}

func TestCleanupExpiredDocuments(t *testing.T) {
	dir := "./test_ttl_cleanup"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("sessions")

	// Create TTL index with 1 second expiration
	err := coll.CreateTTLIndex("createdAt", 1)
	if err != nil {
		t.Fatalf("Failed to create TTL index: %v", err)
	}

	// Insert expired document
	oldTime := time.Now().Add(-5 * time.Second)
	coll.InsertOne(map[string]interface{}{
		"user":      "alice",
		"createdAt": oldTime,
	})

	// Insert non-expired document
	newTime := time.Now()
	coll.InsertOne(map[string]interface{}{
		"user":      "bob",
		"createdAt": newTime,
	})

	// Verify both documents exist
	allDocs, _ := coll.Find(map[string]interface{}{})
	if len(allDocs) != 2 {
		t.Fatalf("Expected 2 documents before cleanup, got %d", len(allDocs))
	}

	// Run cleanup
	deletedCount := coll.CleanupExpiredDocuments()

	// Should delete the expired document
	if deletedCount != 1 {
		t.Errorf("Expected 1 document deleted, got %d", deletedCount)
	}

	// Verify only 1 document remains
	remainingDocs, _ := coll.Find(map[string]interface{}{})
	if len(remainingDocs) != 1 {
		t.Errorf("Expected 1 document after cleanup, got %d", len(remainingDocs))
	}

	// Verify it's the non-expired one
	if len(remainingDocs) > 0 {
		user, _ := remainingDocs[0].Get("user")
		if user != "bob" {
			t.Errorf("Expected remaining user to be bob, got %v", user)
		}
	}
}

func TestMultipleTTLIndexes(t *testing.T) {
	dir := "./test_ttl_multiple"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("events")

	// Create multiple TTL indexes on different fields
	err := coll.CreateTTLIndex("createdAt", 60)
	if err != nil {
		t.Fatalf("Failed to create first TTL index: %v", err)
	}

	err = coll.CreateTTLIndex("expiresAt", 120)
	if err != nil {
		t.Fatalf("Failed to create second TTL index: %v", err)
	}

	// Verify both indexes exist
	if len(coll.ttlIndexes) != 2 {
		t.Errorf("Expected 2 TTL indexes, got %d", len(coll.ttlIndexes))
	}

	// Insert document with both timestamps
	now := time.Now()
	_, err = coll.InsertOne(map[string]interface{}{
		"event":     "test",
		"createdAt": now,
		"expiresAt": now.Add(30 * time.Second),
	})
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Both indexes should have the document
	for name, ttlIdx := range coll.ttlIndexes {
		if ttlIdx.Count() != 1 {
			t.Errorf("Expected 1 document in %s, got %d", name, ttlIdx.Count())
		}
	}
}

func TestTTLWithDifferentTimeFormats(t *testing.T) {
	dir := "./test_ttl_formats"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("mixed")

	// Create TTL index
	err := coll.CreateTTLIndex("timestamp", 300)
	if err != nil {
		t.Fatalf("Failed to create TTL index: %v", err)
	}

	// Insert documents with different time formats
	now := time.Now()

	// time.Time format
	coll.InsertOne(map[string]interface{}{
		"name":      "time.Time",
		"timestamp": now,
	})

	// RFC3339 string format
	coll.InsertOne(map[string]interface{}{
		"name":      "RFC3339",
		"timestamp": now.Format(time.RFC3339),
	})

	// Unix timestamp (int64)
	coll.InsertOne(map[string]interface{}{
		"name":      "Unix",
		"timestamp": now.Unix(),
	})

	// Verify all three were indexed
	ttlIdx := coll.ttlIndexes["timestamp_ttl"]
	if ttlIdx.Count() != 3 {
		t.Errorf("Expected 3 documents in TTL index, got %d", ttlIdx.Count())
	}
}

func TestDropTTLIndex(t *testing.T) {
	dir := "./test_ttl_drop"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("test")

	// Create TTL index
	err := coll.CreateTTLIndex("expires", 60)
	if err != nil {
		t.Fatalf("Failed to create TTL index: %v", err)
	}

	// Verify it exists
	if len(coll.ttlIndexes) != 1 {
		t.Fatal("TTL index not created")
	}

	// Drop the index
	err = coll.DropIndex("expires_ttl")
	if err != nil {
		t.Fatalf("Failed to drop TTL index: %v", err)
	}

	// Verify it's gone
	if len(coll.ttlIndexes) != 0 {
		t.Error("TTL index still exists after drop")
	}
}

func TestTTLCleanupNoExpiredDocs(t *testing.T) {
	dir := "./test_ttl_no_expired"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("fresh")

	// Create TTL index with long expiration
	err := coll.CreateTTLIndex("createdAt", 3600)
	if err != nil {
		t.Fatalf("Failed to create TTL index: %v", err)
	}

	// Insert fresh documents
	now := time.Now()
	coll.InsertOne(map[string]interface{}{
		"data":      "fresh1",
		"createdAt": now,
	})
	coll.InsertOne(map[string]interface{}{
		"data":      "fresh2",
		"createdAt": now,
	})

	// Run cleanup
	deletedCount := coll.CleanupExpiredDocuments()

	// Should delete nothing
	if deletedCount != 0 {
		t.Errorf("Expected 0 documents deleted, got %d", deletedCount)
	}

	// Verify both documents still exist
	docs, _ := coll.Find(map[string]interface{}{})
	if len(docs) != 2 {
		t.Errorf("Expected 2 documents after cleanup, got %d", len(docs))
	}
}

func TestTTLCleanupNoIndexes(t *testing.T) {
	dir := "./test_ttl_no_indexes"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("regular")

	// Insert documents without TTL index
	coll.InsertOne(map[string]interface{}{
		"data": "test1",
	})
	coll.InsertOne(map[string]interface{}{
		"data": "test2",
	})

	// Run cleanup
	deletedCount := coll.CleanupExpiredDocuments()

	// Should delete nothing (no TTL indexes)
	if deletedCount != 0 {
		t.Errorf("Expected 0 documents deleted, got %d", deletedCount)
	}

	// Verify both documents still exist
	docs, _ := coll.Find(map[string]interface{}{})
	if len(docs) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(docs))
	}
}
