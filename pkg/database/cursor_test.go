package database

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/mnohosten/laura-db/pkg/mvcc"
	"github.com/mnohosten/laura-db/pkg/storage"
)

// Helper function to create test collection with data
func createTestCollectionWithData(t *testing.T, name string, count int) *Collection {
	// Create a temporary directory for test data
	tmpDir, err := os.MkdirTemp("", "cursor_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	// Create disk manager with a data file path
	dataFile := tmpDir + "/data.db"
	diskMgr, err := storage.NewDiskManager(dataFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	t.Cleanup(func() { diskMgr.Close() })

	// Create document store
	docStore := NewDocumentStore(diskMgr, 100)

	txnMgr := mvcc.NewTransactionManager()
	coll := NewCollection(name, txnMgr, docStore)

	// Insert test documents
	for i := 0; i < count; i++ {
		doc := map[string]interface{}{
			"name":  fmt.Sprintf("user_%d", i),
			"age":   int64(20 + i),
			"score": int64(100 + i),
		}
		_, err := coll.InsertOne(doc)
		if err != nil {
			t.Fatalf("Failed to insert document: %v", err)
		}
	}

	return coll
}

func TestCursorBasic(t *testing.T) {
	coll := createTestCollectionWithData(t, "users", 10)

	cursor, err := coll.FindCursor(map[string]interface{}{}, nil)
	if err != nil {
		t.Fatalf("Failed to create cursor: %v", err)
	}
	defer cursor.Close()

	// Verify cursor ID is generated
	if cursor.ID() == "" {
		t.Fatal("Cursor ID should not be empty")
	}

	// Verify count
	if cursor.Count() != 10 {
		t.Errorf("Expected count 10, got %d", cursor.Count())
	}

	// Verify position starts at 0
	if cursor.Position() != 0 {
		t.Errorf("Expected initial position 0, got %d", cursor.Position())
	}

	// Iterate through all documents
	count := 0
	for cursor.HasNext() {
		doc, err := cursor.Next()
		if err != nil {
			t.Fatalf("Failed to get next document: %v", err)
		}
		if doc == nil {
			t.Fatal("Expected non-nil document")
		}
		count++
	}

	if count != 10 {
		t.Errorf("Expected to iterate 10 documents, got %d", count)
	}

	// Verify cursor is exhausted
	if !cursor.IsExhausted() {
		t.Error("Cursor should be exhausted")
	}

	// Verify no more documents
	if cursor.HasNext() {
		t.Error("HasNext should return false after exhaustion")
	}
}

func TestCursorNextBatch(t *testing.T) {
	coll := createTestCollectionWithData(t, "users", 25)

	options := &CursorOptions{
		BatchSize: 10,
		Timeout:   5 * time.Minute,
	}

	cursor, err := coll.FindCursor(map[string]interface{}{}, options)
	if err != nil {
		t.Fatalf("Failed to create cursor: %v", err)
	}
	defer cursor.Close()

	// First batch - should get 10 documents
	batch1, err := cursor.NextBatch()
	if err != nil {
		t.Fatalf("Failed to get first batch: %v", err)
	}
	if len(batch1) != 10 {
		t.Errorf("Expected batch size 10, got %d", len(batch1))
	}
	if cursor.Position() != 10 {
		t.Errorf("Expected position 10, got %d", cursor.Position())
	}
	if cursor.Remaining() != 15 {
		t.Errorf("Expected 15 remaining, got %d", cursor.Remaining())
	}

	// Second batch - should get 10 documents
	batch2, err := cursor.NextBatch()
	if err != nil {
		t.Fatalf("Failed to get second batch: %v", err)
	}
	if len(batch2) != 10 {
		t.Errorf("Expected batch size 10, got %d", len(batch2))
	}
	if cursor.Position() != 20 {
		t.Errorf("Expected position 20, got %d", cursor.Position())
	}

	// Third batch - should get 5 documents (remainder)
	batch3, err := cursor.NextBatch()
	if err != nil {
		t.Fatalf("Failed to get third batch: %v", err)
	}
	if len(batch3) != 5 {
		t.Errorf("Expected batch size 5, got %d", len(batch3))
	}
	if cursor.Position() != 25 {
		t.Errorf("Expected position 25, got %d", cursor.Position())
	}

	// Fourth batch - should be empty
	batch4, err := cursor.NextBatch()
	if err != nil {
		t.Fatalf("Failed to get fourth batch: %v", err)
	}
	if len(batch4) != 0 {
		t.Errorf("Expected empty batch, got %d documents", len(batch4))
	}

	// Verify cursor is exhausted
	if !cursor.IsExhausted() {
		t.Error("Cursor should be exhausted")
	}
}

func TestCursorWithFilter(t *testing.T) {
	coll := createTestCollectionWithData(t, "users", 20)

	// Find users with age > 25
	filter := map[string]interface{}{
		"age": map[string]interface{}{
			"$gt": int64(25),
		},
	}

	cursor, err := coll.FindCursor(filter, nil)
	if err != nil {
		t.Fatalf("Failed to create cursor: %v", err)
	}
	defer cursor.Close()

	// Should find documents with age 26, 27, ..., 39 (14 documents)
	count := cursor.Count()
	if count != 14 {
		t.Errorf("Expected 14 documents with age > 25, got %d", count)
	}

	// Verify all documents match filter
	for cursor.HasNext() {
		doc, err := cursor.Next()
		if err != nil {
			t.Fatalf("Failed to get next document: %v", err)
		}

		age, _ := doc.Get("age")
		ageInt := age.(int64)
		if ageInt <= 25 {
			t.Errorf("Expected age > 25, got %d", ageInt)
		}
	}
}

func TestCursorWithQueryOptions(t *testing.T) {
	coll := createTestCollectionWithData(t, "users", 20)

	queryOptions := &QueryOptions{
		Skip:  5,
		Limit: 10,
		Projection: map[string]bool{
			"name": true,
		},
	}

	cursor, err := coll.FindCursorWithOptions(map[string]interface{}{}, queryOptions, nil)
	if err != nil {
		t.Fatalf("Failed to create cursor: %v", err)
	}
	defer cursor.Close()

	// Should have 10 documents (skip 5, limit 10)
	if cursor.Count() != 10 {
		t.Errorf("Expected 10 documents, got %d", cursor.Count())
	}

	// Verify projection is applied
	doc, err := cursor.Next()
	if err != nil {
		t.Fatalf("Failed to get next document: %v", err)
	}

	// Should have name field (projection was applied)
	if _, exists := doc.Get("name"); !exists {
		t.Error("Expected name field in projection")
	}
	// age and score should be excluded (depending on projection implementation)
	// Note: The actual projection behavior depends on query engine implementation
}

func TestCursorEmpty(t *testing.T) {
	coll := createTestCollectionWithData(t, "users", 10)

	// Query that matches no documents
	filter := map[string]interface{}{
		"age": map[string]interface{}{
			"$gt": int64(1000),
		},
	}

	cursor, err := coll.FindCursor(filter, nil)
	if err != nil {
		t.Fatalf("Failed to create cursor: %v", err)
	}
	defer cursor.Close()

	// Should have no documents
	if cursor.Count() != 0 {
		t.Errorf("Expected 0 documents, got %d", cursor.Count())
	}

	// HasNext should return false
	if cursor.HasNext() {
		t.Error("HasNext should return false for empty cursor")
	}

	// After trying to fetch, it should be exhausted
	batch, err := cursor.NextBatch()
	if err != nil {
		t.Fatalf("NextBatch should not error on empty cursor: %v", err)
	}
	if len(batch) != 0 {
		t.Errorf("Expected empty batch, got %d", len(batch))
	}
	if !cursor.IsExhausted() {
		t.Error("Empty cursor should be exhausted after NextBatch")
	}
}

func TestCursorManager(t *testing.T) {
	coll := createTestCollectionWithData(t, "users", 20)
	manager := NewCursorManager()

	// Create cursor through manager
	cursor, err := manager.CreateCursor(coll, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create cursor: %v", err)
	}

	cursorID := cursor.ID()

	// Verify cursor is registered
	if manager.ActiveCursors() != 1 {
		t.Errorf("Expected 1 active cursor, got %d", manager.ActiveCursors())
	}

	// Retrieve cursor
	retrieved, err := manager.GetCursor(cursorID)
	if err != nil {
		t.Fatalf("Failed to get cursor: %v", err)
	}
	if retrieved.ID() != cursorID {
		t.Error("Retrieved cursor ID does not match")
	}

	// Close cursor
	err = manager.CloseCursor(cursorID)
	if err != nil {
		t.Fatalf("Failed to close cursor: %v", err)
	}

	// Verify cursor is removed
	if manager.ActiveCursors() != 0 {
		t.Errorf("Expected 0 active cursors after close, got %d", manager.ActiveCursors())
	}

	// Try to get closed cursor
	_, err = manager.GetCursor(cursorID)
	if err == nil {
		t.Error("Expected error getting closed cursor")
	}
}

func TestCursorManagerTimeout(t *testing.T) {
	coll := createTestCollectionWithData(t, "users", 10)
	manager := NewCursorManager()

	// Create cursor with short timeout
	options := &CursorOptions{
		BatchSize: 5,
		Timeout:   100 * time.Millisecond,
	}

	cursor, err := manager.CreateCursor(coll, nil, options)
	if err != nil {
		t.Fatalf("Failed to create cursor: %v", err)
	}

	cursorID := cursor.ID()

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Try to get cursor - should fail due to timeout
	_, err = manager.GetCursor(cursorID)
	if err == nil {
		t.Error("Expected error getting timed out cursor")
	}

	// Verify cursor was removed
	if manager.ActiveCursors() != 0 {
		t.Errorf("Expected 0 active cursors after timeout, got %d", manager.ActiveCursors())
	}
}

func TestCursorManagerCleanup(t *testing.T) {
	coll := createTestCollectionWithData(t, "users", 10)
	manager := NewCursorManager()

	// Create multiple cursors with short timeout
	options := &CursorOptions{
		BatchSize: 5,
		Timeout:   100 * time.Millisecond,
	}

	for i := 0; i < 5; i++ {
		_, err := manager.CreateCursor(coll, nil, options)
		if err != nil {
			t.Fatalf("Failed to create cursor %d: %v", i, err)
		}
	}

	// Verify all cursors are active
	if manager.ActiveCursors() != 5 {
		t.Errorf("Expected 5 active cursors, got %d", manager.ActiveCursors())
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Run cleanup
	removed := manager.CleanupTimedOutCursors()
	if removed != 5 {
		t.Errorf("Expected 5 cursors removed, got %d", removed)
	}

	// Verify all cursors are removed
	if manager.ActiveCursors() != 0 {
		t.Errorf("Expected 0 active cursors after cleanup, got %d", manager.ActiveCursors())
	}
}

func TestCursorManagerConcurrent(t *testing.T) {
	coll := createTestCollectionWithData(t, "users", 100)
	manager := NewCursorManager()

	var wg sync.WaitGroup
	numGoroutines := 10

	// Create cursors concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			cursor, err := manager.CreateCursor(coll, nil, nil)
			if err != nil {
				t.Errorf("Failed to create cursor: %v", err)
				return
			}

			// Iterate through some documents
			for j := 0; j < 10 && cursor.HasNext(); j++ {
				_, err := cursor.Next()
				if err != nil {
					t.Errorf("Failed to get next document: %v", err)
					return
				}
			}

			// Close cursor
			manager.CloseCursor(cursor.ID())
		}()
	}

	wg.Wait()

	// All cursors should be closed
	if manager.ActiveCursors() != 0 {
		t.Errorf("Expected 0 active cursors after concurrent operations, got %d", manager.ActiveCursors())
	}
}

func TestCursorClose(t *testing.T) {
	coll := createTestCollectionWithData(t, "users", 10)

	cursor, err := coll.FindCursor(map[string]interface{}{}, nil)
	if err != nil {
		t.Fatalf("Failed to create cursor: %v", err)
	}

	// Close cursor
	cursor.Close()

	// Verify cursor is exhausted
	if !cursor.IsExhausted() {
		t.Error("Closed cursor should be exhausted")
	}

	// Verify HasNext returns false
	if cursor.HasNext() {
		t.Error("HasNext should return false for closed cursor")
	}

	// Verify Next returns error
	_, err = cursor.Next()
	if err == nil {
		t.Error("Expected error calling Next on closed cursor")
	}
}

func TestCursorDefaultOptions(t *testing.T) {
	coll := createTestCollectionWithData(t, "users", 10)

	// Create cursor with nil options (should use defaults)
	cursor, err := coll.FindCursor(map[string]interface{}{}, nil)
	if err != nil {
		t.Fatalf("Failed to create cursor: %v", err)
	}
	defer cursor.Close()

	// Verify default batch size (100)
	if cursor.batchSize != 100 {
		t.Errorf("Expected default batch size 100, got %d", cursor.batchSize)
	}

	// Verify default timeout (10 minutes)
	if cursor.timeout != 10*time.Minute {
		t.Errorf("Expected default timeout 10 minutes, got %v", cursor.timeout)
	}
}

func TestCursorExhaustedBehavior(t *testing.T) {
	coll := createTestCollectionWithData(t, "users", 3)

	cursor, err := coll.FindCursor(map[string]interface{}{}, nil)
	if err != nil {
		t.Fatalf("Failed to create cursor: %v", err)
	}
	defer cursor.Close()

	// Exhaust the cursor
	for cursor.HasNext() {
		cursor.Next()
	}

	// Try to get another batch
	batch, err := cursor.NextBatch()
	if err != nil {
		t.Fatalf("NextBatch should not error on exhausted cursor, got: %v", err)
	}
	if len(batch) != 0 {
		t.Errorf("Expected empty batch from exhausted cursor, got %d documents", len(batch))
	}

	// Try to get next document
	_, err = cursor.Next()
	if err == nil {
		t.Error("Expected error calling Next on exhausted cursor")
	}
}

func TestGenerateCursorID(t *testing.T) {
	// Generate multiple cursor IDs
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := generateCursorID()
		if err != nil {
			t.Fatalf("Failed to generate cursor ID: %v", err)
		}

		// Verify ID is not empty
		if id == "" {
			t.Fatal("Generated cursor ID is empty")
		}

		// Verify ID is unique
		if ids[id] {
			t.Fatalf("Duplicate cursor ID generated: %s", id)
		}
		ids[id] = true

		// Verify ID length (32 hex chars from 16 bytes)
		if len(id) != 32 {
			t.Errorf("Expected cursor ID length 32, got %d", len(id))
		}
	}
}
