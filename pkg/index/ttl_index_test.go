package index

import (
	"strings"
	"testing"
	"time"
)

func TestNewTTLIndex(t *testing.T) {
	idx := NewTTLIndex("ttl_idx", "createdAt", 3600)

	if idx == nil {
		t.Fatal("NewTTLIndex returned nil")
	}

	if idx.Name() != "ttl_idx" {
		t.Errorf("Expected name 'ttl_idx', got '%s'", idx.Name())
	}

	if idx.FieldPath() != "createdAt" {
		t.Errorf("Expected field path 'createdAt', got '%s'", idx.FieldPath())
	}

	if idx.TTLSeconds() != 3600 {
		t.Errorf("Expected TTL 3600 seconds, got %d", idx.TTLSeconds())
	}

	if idx.Count() != 0 {
		t.Errorf("Expected count 0 for new index, got %d", idx.Count())
	}
}

func TestTTLIndex_Index(t *testing.T) {
	idx := NewTTLIndex("ttl_idx", "timestamp", 60) // 60 second TTL

	now := time.Now()
	err := idx.Index("doc1", now)
	if err != nil {
		t.Fatalf("Failed to index document: %v", err)
	}

	if idx.Count() != 1 {
		t.Errorf("Expected count 1 after indexing, got %d", idx.Count())
	}

	// Verify expiration time
	expTime, exists := idx.GetExpirationTime("doc1")
	if !exists {
		t.Fatal("Document not found in TTL index")
	}

	expectedExpTime := now.Add(60 * time.Second)
	if expTime.Unix() != expectedExpTime.Unix() {
		t.Errorf("Expected expiration time %v, got %v", expectedExpTime, expTime)
	}
}

func TestTTLIndex_Remove(t *testing.T) {
	idx := NewTTLIndex("ttl_idx", "timestamp", 60)

	now := time.Now()
	idx.Index("doc1", now)
	idx.Index("doc2", now)

	if idx.Count() != 2 {
		t.Fatalf("Expected count 2, got %d", idx.Count())
	}

	// Remove doc1
	idx.Remove("doc1")

	if idx.Count() != 1 {
		t.Errorf("Expected count 1 after removal, got %d", idx.Count())
	}

	// Verify doc1 removed
	_, exists := idx.GetExpirationTime("doc1")
	if exists {
		t.Error("doc1 should not exist after removal")
	}

	// Verify doc2 still exists
	_, exists = idx.GetExpirationTime("doc2")
	if !exists {
		t.Error("doc2 should still exist after removing doc1")
	}
}

func TestTTLIndex_GetExpiredDocuments(t *testing.T) {
	idx := NewTTLIndex("ttl_idx", "timestamp", 10) // 10 second TTL

	now := time.Now()

	// Add documents at different times
	past := now.Add(-20 * time.Second)  // Already expired
	recent := now.Add(-5 * time.Second) // Not yet expired
	future := now                       // Definitely not expired

	idx.Index("expired1", past)
	idx.Index("active1", recent)
	idx.Index("active2", future)

	// Check for expired documents
	expired := idx.GetExpiredDocuments(now)

	if len(expired) != 1 {
		t.Fatalf("Expected 1 expired document, got %d: %v", len(expired), expired)
	}

	if expired[0] != "expired1" {
		t.Errorf("Expected 'expired1' to be expired, got '%s'", expired[0])
	}
}

func TestTTLIndex_GetExpiredDocuments_Multiple(t *testing.T) {
	idx := NewTTLIndex("ttl_idx", "timestamp", 30) // 30 second TTL

	now := time.Now()

	// Add multiple expired documents
	past1 := now.Add(-60 * time.Second)
	past2 := now.Add(-45 * time.Second)
	recent := now.Add(-10 * time.Second)

	idx.Index("expired1", past1)
	idx.Index("expired2", past2)
	idx.Index("active", recent)

	expired := idx.GetExpiredDocuments(now)

	if len(expired) != 2 {
		t.Fatalf("Expected 2 expired documents, got %d", len(expired))
	}

	// Check both expired docs are in the result
	expiredMap := make(map[string]bool)
	for _, docID := range expired {
		expiredMap[docID] = true
	}

	if !expiredMap["expired1"] {
		t.Error("Expected 'expired1' to be in expired list")
	}
	if !expiredMap["expired2"] {
		t.Error("Expected 'expired2' to be in expired list")
	}
	if expiredMap["active"] {
		t.Error("'active' should not be in expired list")
	}
}

func TestTTLIndex_GetExpiredDocuments_None(t *testing.T) {
	idx := NewTTLIndex("ttl_idx", "timestamp", 60)

	now := time.Now()

	// Add recent documents
	idx.Index("doc1", now)
	idx.Index("doc2", now.Add(-10*time.Second))

	expired := idx.GetExpiredDocuments(now)

	if len(expired) != 0 {
		t.Errorf("Expected 0 expired documents, got %d", len(expired))
	}
}

func TestTTLIndex_GetExpirationTime(t *testing.T) {
	idx := NewTTLIndex("ttl_idx", "timestamp", 120)

	now := time.Now()
	idx.Index("doc1", now)

	// Get expiration time
	expTime, exists := idx.GetExpirationTime("doc1")
	if !exists {
		t.Fatal("Document should exist in TTL index")
	}

	expectedExpTime := now.Add(120 * time.Second)
	if expTime.Unix() != expectedExpTime.Unix() {
		t.Errorf("Expected expiration time %v, got %v", expectedExpTime, expTime)
	}

	// Get non-existent document
	_, exists = idx.GetExpirationTime("nonexistent")
	if exists {
		t.Error("Non-existent document should not exist in TTL index")
	}
}

func TestTTLIndex_String(t *testing.T) {
	idx := NewTTLIndex("test_ttl", "createdAt", 3600)

	idx.Index("doc1", time.Now())
	idx.Index("doc2", time.Now())

	str := idx.String()

	// Verify string contains expected information
	if !strings.Contains(str, "test_ttl") {
		t.Error("String representation should contain index name")
	}
	if !strings.Contains(str, "createdAt") {
		t.Error("String representation should contain field path")
	}
	if !strings.Contains(str, "3600") {
		t.Error("String representation should contain TTL seconds")
	}
	if !strings.Contains(str, "2") {
		t.Error("String representation should contain document count")
	}
}

func TestTTLIndex_UpdateExistingDocument(t *testing.T) {
	idx := NewTTLIndex("ttl_idx", "timestamp", 60)

	now := time.Now()

	// Index document
	idx.Index("doc1", now)

	// Get initial expiration time
	exp1, _ := idx.GetExpirationTime("doc1")

	// Re-index same document with new timestamp
	laterTime := now.Add(30 * time.Second)
	idx.Index("doc1", laterTime)

	// Get updated expiration time
	exp2, exists := idx.GetExpirationTime("doc1")
	if !exists {
		t.Fatal("Document should still exist after re-indexing")
	}

	// Expiration time should be updated
	if exp2.Unix() <= exp1.Unix() {
		t.Error("Expiration time should be updated when re-indexing")
	}

	// Count should still be 1 (not 2)
	if idx.Count() != 1 {
		t.Errorf("Expected count 1 after re-indexing same doc, got %d", idx.Count())
	}
}

func TestTTLIndex_Concurrent(t *testing.T) {
	idx := NewTTLIndex("ttl_idx", "timestamp", 60)

	now := time.Now()

	// Concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			docID := "doc" + string(rune('0'+id))
			idx.Index(docID, now)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all documents indexed
	if idx.Count() != 10 {
		t.Errorf("Expected 10 documents after concurrent writes, got %d", idx.Count())
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(id int) {
			docID := "doc" + string(rune('0'+id))
			_, exists := idx.GetExpirationTime(docID)
			if !exists {
				t.Errorf("Document %s should exist", docID)
			}
			done <- true
		}(i)
	}

	// Wait for all reads
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent GetExpiredDocuments
	for i := 0; i < 5; i++ {
		go func() {
			expired := idx.GetExpiredDocuments(now)
			// Should have no expired documents
			if len(expired) != 0 {
				t.Errorf("Expected 0 expired documents, got %d", len(expired))
			}
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestTTLIndex_ZeroTTL(t *testing.T) {
	// TTL of 0 means documents expire immediately
	idx := NewTTLIndex("ttl_idx", "timestamp", 0)

	now := time.Now()
	idx.Index("doc1", now)

	// Document should be expired after 0 seconds (or at the exact same time)
	// Need to check slightly after to account for time precision
	checkTime := now.Add(1 * time.Nanosecond)
	expired := idx.GetExpiredDocuments(checkTime)
	if len(expired) == 0 {
		t.Error("Expected document to be immediately expired with 0 TTL")
	}
}

func TestTTLIndex_LargeTTL(t *testing.T) {
	// Very large TTL (1 year)
	oneYear := int64(365 * 24 * 3600)
	idx := NewTTLIndex("ttl_idx", "timestamp", oneYear)

	now := time.Now()
	idx.Index("doc1", now)

	// Document should not be expired
	expired := idx.GetExpiredDocuments(now)
	if len(expired) != 0 {
		t.Error("Document should not be expired with large TTL")
	}

	// Check expiration is set correctly
	expTime, _ := idx.GetExpirationTime("doc1")
	expectedExpTime := now.Add(time.Duration(oneYear) * time.Second)

	// Allow 1 second tolerance due to time precision
	diff := expTime.Unix() - expectedExpTime.Unix()
	if diff < -1 || diff > 1 {
		t.Errorf("Expiration time mismatch: expected %v, got %v", expectedExpTime, expTime)
	}
}
