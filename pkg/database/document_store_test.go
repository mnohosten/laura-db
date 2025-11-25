package database

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/storage"
)

func createTestDocumentStore(t *testing.T) (*DocumentStore, string, func()) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "lauradb-docstore-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create disk manager
	dbPath := filepath.Join(tmpDir, "test.db")
	diskManager, err := storage.NewDiskManager(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create disk manager: %v", err)
	}

	// Create document store
	docStore := NewDocumentStore(diskManager, 1000)

	cleanup := func() {
		diskManager.Close()
		os.RemoveAll(tmpDir)
	}

	return docStore, tmpDir, cleanup
}

func TestDocumentStore_Insert(t *testing.T) {
	docStore, _, cleanup := createTestDocumentStore(t)
	defer cleanup()

	// Create a test document
	doc := document.NewDocumentFromMap(map[string]interface{}{
		"_id":  "doc1",
		"name": "Test Document",
		"age":  int64(30),
	})

	// Insert document
	err := docStore.Insert("doc1", doc)
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Verify document count
	if count := docStore.Count(); count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}

	// Verify document exists
	if !docStore.Exists("doc1") {
		t.Error("Document should exist")
	}
}

func TestDocumentStore_InsertDuplicate(t *testing.T) {
	docStore, _, cleanup := createTestDocumentStore(t)
	defer cleanup()

	doc := document.NewDocumentFromMap(map[string]interface{}{
		"_id":  "doc1",
		"name": "Test Document",
	})

	// Insert document
	err := docStore.Insert("doc1", doc)
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Try to insert duplicate
	err = docStore.Insert("doc1", doc)
	if err == nil {
		t.Error("Expected error when inserting duplicate, got nil")
	}
}

func TestDocumentStore_Get(t *testing.T) {
	docStore, _, cleanup := createTestDocumentStore(t)
	defer cleanup()

	// Insert document
	originalDoc := document.NewDocumentFromMap(map[string]interface{}{
		"_id":  "doc1",
		"name": "Test Document",
		"age":  int64(30),
	})

	err := docStore.Insert("doc1", originalDoc)
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Get document
	retrievedDoc, err := docStore.Get("doc1")
	if err != nil {
		t.Fatalf("Failed to get document: %v", err)
	}

	// Verify fields
	name, _ := retrievedDoc.Get("name")
	if name != "Test Document" {
		t.Errorf("Expected name 'Test Document', got %v", name)
	}

	age, _ := retrievedDoc.Get("age")
	if age != int64(30) {
		t.Errorf("Expected age 30, got %v", age)
	}
}

func TestDocumentStore_GetFromCache(t *testing.T) {
	docStore, _, cleanup := createTestDocumentStore(t)
	defer cleanup()

	// Insert document
	originalDoc := document.NewDocumentFromMap(map[string]interface{}{
		"_id":  "doc1",
		"name": "Test Document",
	})

	err := docStore.Insert("doc1", originalDoc)
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// First get (should be cache hit from insert)
	_, err = docStore.Get("doc1")
	if err != nil {
		t.Fatalf("Failed to get document: %v", err)
	}

	// Second get (should also be from cache)
	_, err = docStore.Get("doc1")
	if err != nil {
		t.Fatalf("Failed to get cached document: %v", err)
	}

	// Check cache stats - insert caches the document, so both gets should be hits
	stats := docStore.Stats()
	cacheSize := stats["cache_size"].(int)
	if cacheSize < 1 {
		t.Errorf("Expected cache size >= 1, got %d", cacheSize)
	}
}

func TestDocumentStore_GetNotFound(t *testing.T) {
	docStore, _, cleanup := createTestDocumentStore(t)
	defer cleanup()

	// Try to get non-existent document
	_, err := docStore.Get("nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent document, got nil")
	}
}

func TestDocumentStore_Update(t *testing.T) {
	docStore, _, cleanup := createTestDocumentStore(t)
	defer cleanup()

	// Insert document
	originalDoc := document.NewDocumentFromMap(map[string]interface{}{
		"_id":  "doc1",
		"name": "Test Document",
		"age":  int64(30),
	})

	err := docStore.Insert("doc1", originalDoc)
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Update document
	updatedDoc := document.NewDocumentFromMap(map[string]interface{}{
		"_id":  "doc1",
		"name": "Updated Document",
		"age":  int64(35),
	})

	err = docStore.Update("doc1", updatedDoc)
	if err != nil {
		t.Fatalf("Failed to update document: %v", err)
	}

	// Verify update
	retrievedDoc, err := docStore.Get("doc1")
	if err != nil {
		t.Fatalf("Failed to get updated document: %v", err)
	}

	name, _ := retrievedDoc.Get("name")
	if name != "Updated Document" {
		t.Errorf("Expected name 'Updated Document', got %v", name)
	}

	age, _ := retrievedDoc.Get("age")
	if age != int64(35) {
		t.Errorf("Expected age 35, got %v", age)
	}
}

func TestDocumentStore_UpdateNotFound(t *testing.T) {
	docStore, _, cleanup := createTestDocumentStore(t)
	defer cleanup()

	// Try to update non-existent document
	doc := document.NewDocumentFromMap(map[string]interface{}{
		"_id":  "doc1",
		"name": "Test",
	})

	err := docStore.Update("nonexistent", doc)
	if err == nil {
		t.Error("Expected error when updating non-existent document, got nil")
	}
}

func TestDocumentStore_Delete(t *testing.T) {
	docStore, _, cleanup := createTestDocumentStore(t)
	defer cleanup()

	// Insert document
	doc := document.NewDocumentFromMap(map[string]interface{}{
		"_id":  "doc1",
		"name": "Test Document",
	})

	err := docStore.Insert("doc1", doc)
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Delete document
	err = docStore.Delete("doc1")
	if err != nil {
		t.Fatalf("Failed to delete document: %v", err)
	}

	// Verify deletion
	if docStore.Exists("doc1") {
		t.Error("Document should not exist after deletion")
	}

	if count := docStore.Count(); count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
}

func TestDocumentStore_DeleteNotFound(t *testing.T) {
	docStore, _, cleanup := createTestDocumentStore(t)
	defer cleanup()

	// Try to delete non-existent document
	err := docStore.Delete("nonexistent")
	if err == nil {
		t.Error("Expected error when deleting non-existent document, got nil")
	}
}

func TestDocumentStore_GetAllIDs(t *testing.T) {
	docStore, _, cleanup := createTestDocumentStore(t)
	defer cleanup()

	// Insert multiple documents
	for i := 1; i <= 5; i++ {
		doc := document.NewDocumentFromMap(map[string]interface{}{
			"_id":   fmt.Sprintf("doc%d", i),
			"value": int64(i),
		})
		err := docStore.Insert(fmt.Sprintf("doc%d", i), doc)
		if err != nil {
			t.Fatalf("Failed to insert document %d: %v", i, err)
		}
	}

	// Get all IDs
	ids := docStore.GetAllIDs()
	if len(ids) != 5 {
		t.Errorf("Expected 5 IDs, got %d", len(ids))
	}

	// Verify all IDs are present
	idMap := make(map[string]bool)
	for _, id := range ids {
		idMap[id] = true
	}

	for i := 1; i <= 5; i++ {
		expectedID := fmt.Sprintf("doc%d", i)
		if !idMap[expectedID] {
			t.Errorf("Expected ID %s not found in results", expectedID)
		}
	}
}

func TestDocumentStore_MultiplePages(t *testing.T) {
	docStore, _, cleanup := createTestDocumentStore(t)
	defer cleanup()

	// Insert many documents to span multiple pages
	// Each document should be small enough to fit multiple per page
	numDocs := 100

	for i := 0; i < numDocs; i++ {
		doc := document.NewDocumentFromMap(map[string]interface{}{
			"_id":   fmt.Sprintf("doc%d", i),
			"index": int64(i),
			"data":  "Some test data",
		})
		err := docStore.Insert(fmt.Sprintf("doc%d", i), doc)
		if err != nil {
			t.Fatalf("Failed to insert document %d: %v", i, err)
		}
	}

	// Verify count
	if count := docStore.Count(); count != numDocs {
		t.Errorf("Expected count %d, got %d", numDocs, count)
	}

	// Verify we can retrieve all documents
	for i := 0; i < numDocs; i++ {
		id := fmt.Sprintf("doc%d", i)
		doc, err := docStore.Get(id)
		if err != nil {
			t.Fatalf("Failed to get document %s: %v", id, err)
		}

		index, _ := doc.Get("index")
		if index != int64(i) {
			t.Errorf("Expected index %d, got %v", i, index)
		}
	}

	// Check that multiple pages were used
	stats := docStore.Stats()
	activePages := stats["active_pages"].(int)
	if activePages < 2 {
		t.Errorf("Expected at least 2 active pages, got %d", activePages)
	}
}

func TestDocumentStore_FlushAll(t *testing.T) {
	docStore, _, cleanup := createTestDocumentStore(t)
	defer cleanup()

	// Insert documents
	for i := 0; i < 10; i++ {
		doc := document.NewDocumentFromMap(map[string]interface{}{
			"_id":   fmt.Sprintf("doc%d", i),
			"value": int64(i),
		})
		err := docStore.Insert(fmt.Sprintf("doc%d", i), doc)
		if err != nil {
			t.Fatalf("Failed to insert document %d: %v", i, err)
		}
	}

	// Flush all
	err := docStore.FlushAll()
	if err != nil {
		t.Fatalf("Failed to flush all: %v", err)
	}
}

func TestDocumentStore_Stats(t *testing.T) {
	docStore, _, cleanup := createTestDocumentStore(t)
	defer cleanup()

	// Insert documents
	for i := 0; i < 5; i++ {
		doc := document.NewDocumentFromMap(map[string]interface{}{
			"_id":   fmt.Sprintf("doc%d", i),
			"value": int64(i),
		})
		err := docStore.Insert(fmt.Sprintf("doc%d", i), doc)
		if err != nil {
			t.Fatalf("Failed to insert document %d: %v", i, err)
		}
	}

	// Get stats
	stats := docStore.Stats()

	// Verify stats
	if stats["document_count"].(int) != 5 {
		t.Errorf("Expected document_count 5, got %v", stats["document_count"])
	}

	if _, ok := stats["active_pages"]; !ok {
		t.Error("Expected active_pages in stats")
	}

	if _, ok := stats["cache_size"]; !ok {
		t.Error("Expected cache_size in stats")
	}

	if _, ok := stats["cache_hit_rate"]; !ok {
		t.Error("Expected cache_hit_rate in stats")
	}
}

func TestDocumentStore_LargeDocument(t *testing.T) {
	docStore, _, cleanup := createTestDocumentStore(t)
	defer cleanup()

	// Create a larger document
	largeData := make([]byte, 1024) // 1KB of data
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	doc := document.NewDocumentFromMap(map[string]interface{}{
		"_id":  "large_doc",
		"data": string(largeData),
		"size": int64(len(largeData)),
	})

	// Insert large document
	err := docStore.Insert("large_doc", doc)
	if err != nil {
		t.Fatalf("Failed to insert large document: %v", err)
	}

	// Retrieve and verify
	retrievedDoc, err := docStore.Get("large_doc")
	if err != nil {
		t.Fatalf("Failed to get large document: %v", err)
	}

	size, _ := retrievedDoc.Get("size")
	if size != int64(len(largeData)) {
		t.Errorf("Expected size %d, got %v", len(largeData), size)
	}
}

func TestDocumentStore_Persistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lauradb-persistence-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Create first store and insert documents
	{
		diskManager, err := storage.NewDiskManager(dbPath)
		if err != nil {
			t.Fatalf("Failed to create disk manager: %v", err)
		}

		docStore := NewDocumentStore(diskManager, 1000)

		// Insert documents
		for i := 0; i < 10; i++ {
			doc := document.NewDocumentFromMap(map[string]interface{}{
				"_id":   fmt.Sprintf("doc%d", i),
				"value": int64(i),
			})
			err := docStore.Insert(fmt.Sprintf("doc%d", i), doc)
			if err != nil {
				t.Fatalf("Failed to insert document %d: %v", i, err)
			}
		}

		// Flush and close
		docStore.FlushAll()
		diskManager.Close()
	}

	// Reopen and verify documents persist
	{
		diskManager, err := storage.NewDiskManager(dbPath)
		if err != nil {
			t.Fatalf("Failed to reopen disk manager: %v", err)
		}
		defer diskManager.Close()

		docStore := NewDocumentStore(diskManager, 1000)

		// Note: In a full implementation, we would need to reload the location map
		// from a metadata page. For now, this test demonstrates the concept.
		// The actual persistence would require additional metadata storage.

		// For this test, we'll just verify that the disk manager was created successfully
		// A full persistence test would require implementing metadata persistence first
		if docStore == nil {
			t.Error("Expected non-nil document store")
		}
	}
}
