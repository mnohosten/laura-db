package index

import (
	"testing"
)

func TestNewTextIndex(t *testing.T) {
	// Test single field
	ti := NewTextIndex("test_idx", []string{"title"})
	if ti == nil {
		t.Fatal("NewTextIndex returned nil")
	}
	if ti.Name() != "test_idx" {
		t.Errorf("Expected name 'test_idx', got '%s'", ti.Name())
	}
	if len(ti.FieldPaths()) != 1 || ti.FieldPaths()[0] != "title" {
		t.Errorf("Expected field paths ['title'], got %v", ti.FieldPaths())
	}
	if ti.IsCompound() {
		t.Error("Single field index should not be compound")
	}

	// Test multiple fields (compound)
	ti2 := NewTextIndex("multi_idx", []string{"title", "description"})
	if !ti2.IsCompound() {
		t.Error("Multi-field index should be compound")
	}
}

func TestTextIndex_IndexAndSearch(t *testing.T) {
	ti := NewTextIndex("test_idx", []string{"content"})

	// Index some documents
	ti.Index("doc1", []string{"The quick brown fox jumps over the lazy dog"})
	ti.Index("doc2", []string{"A quick brown fox is a fast animal"})
	ti.Index("doc3", []string{"The lazy cat sleeps all day"})

	// Search for "quick"
	results := ti.Search("quick")
	if len(results) == 0 {
		t.Fatal("Expected to find documents matching 'quick'")
	}

	// Verify results contain expected documents
	foundDoc1, foundDoc2 := false, false
	for _, result := range results {
		if result.DocID == "doc1" {
			foundDoc1 = true
		}
		if result.DocID == "doc2" {
			foundDoc2 = true
		}
	}
	if !foundDoc1 || !foundDoc2 {
		t.Error("Expected to find both doc1 and doc2 for query 'quick'")
	}

	// Search for "lazy" - should find doc1 and doc3
	results = ti.Search("lazy")
	if len(results) == 0 {
		t.Fatal("Expected to find documents matching 'lazy'")
	}
	foundDoc1, foundDoc3 := false, false
	for _, result := range results {
		if result.DocID == "doc1" {
			foundDoc1 = true
		}
		if result.DocID == "doc3" {
			foundDoc3 = true
		}
	}
	if !foundDoc1 || !foundDoc3 {
		t.Error("Expected to find both doc1 and doc3 for query 'lazy'")
	}
}

func TestTextIndex_MultipleFields(t *testing.T) {
	ti := NewTextIndex("multi_idx", []string{"title", "body"})

	// Index document with multiple fields
	ti.Index("doc1", []string{"Database Systems", "This article discusses database internals"})
	ti.Index("doc2", []string{"Web Development", "Building modern web applications"})

	// Search should work across all fields
	results := ti.Search("database")
	if len(results) == 0 {
		t.Fatal("Expected to find documents matching 'database'")
	}
	if results[0].DocID != "doc1" {
		t.Errorf("Expected doc1, got %s", results[0].DocID)
	}

	results = ti.Search("web")
	if len(results) == 0 {
		t.Fatal("Expected to find documents matching 'web'")
	}
	if results[0].DocID != "doc2" {
		t.Errorf("Expected doc2, got %s", results[0].DocID)
	}
}

func TestTextIndex_Remove(t *testing.T) {
	ti := NewTextIndex("test_idx", []string{"content"})

	// Index documents
	ti.Index("doc1", []string{"hello world"})
	ti.Index("doc2", []string{"hello universe"})

	// Verify both are searchable
	results := ti.Search("hello")
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Remove doc1
	ti.Remove("doc1")

	// Search again - should only find doc2
	results = ti.Search("hello")
	if len(results) != 1 {
		t.Errorf("Expected 1 result after removal, got %d", len(results))
	}
	if results[0].DocID != "doc2" {
		t.Errorf("Expected doc2, got %s", results[0].DocID)
	}

	// Search for "world" should return no results
	results = ti.Search("world")
	if len(results) != 0 {
		t.Errorf("Expected 0 results for removed document, got %d", len(results))
	}
}

func TestTextIndex_EmptySearch(t *testing.T) {
	ti := NewTextIndex("test_idx", []string{"content"})

	ti.Index("doc1", []string{"some content"})

	// Search for non-existent term
	results := ti.Search("nonexistent")
	if len(results) != 0 {
		t.Errorf("Expected 0 results for non-existent term, got %d", len(results))
	}

	// Empty search query
	results = ti.Search("")
	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty query, got %d", len(results))
	}
}

func TestTextIndex_Stats(t *testing.T) {
	ti := NewTextIndex("test_idx", []string{"content"})

	// Index some documents
	ti.Index("doc1", []string{"the quick brown fox"})
	ti.Index("doc2", []string{"jumps over the lazy dog"})

	stats := ti.Stats()

	// Check stats structure
	if stats["name"] != "test_idx" {
		t.Errorf("Expected name 'test_idx', got %v", stats["name"])
	}
	if stats["type"] != "text" {
		t.Errorf("Expected type 'text', got %v", stats["type"])
	}

	fieldPaths, ok := stats["field_paths"].([]string)
	if !ok || len(fieldPaths) != 1 || fieldPaths[0] != "content" {
		t.Errorf("Expected field_paths ['content'], got %v", stats["field_paths"])
	}

	// Check that we have indexed documents
	totalDocs, ok := stats["total_documents"].(int)
	if !ok || totalDocs != 2 {
		t.Errorf("Expected total_documents 2, got %v", stats["total_documents"])
	}

	// Check that we have terms
	totalTerms, ok := stats["total_terms"].(int)
	if !ok || totalTerms == 0 {
		t.Errorf("Expected total_terms > 0, got %v", stats["total_terms"])
	}
}

func TestTextIndex_Analyze(t *testing.T) {
	ti := NewTextIndex("test_idx", []string{"content"})

	// Index documents
	ti.Index("doc1", []string{"hello world"})
	ti.Index("doc2", []string{"hello universe"})

	// Analyze should update statistics
	ti.Analyze()

	// Get statistics object
	stats := ti.GetStatistics()
	if stats == nil {
		t.Fatal("GetStatistics returned nil")
	}

	// Stats should be fresh after analyze
	if stats.IsStale {
		t.Error("Stats should not be stale after Analyze()")
	}
}

func TestTextIndex_Size(t *testing.T) {
	ti := NewTextIndex("test_idx", []string{"content"})

	// Initially empty
	if ti.Size() != 0 {
		t.Errorf("Expected size 0 for empty index, got %d", ti.Size())
	}

	// Index documents
	ti.Index("doc1", []string{"alpha beta gamma"})
	ti.Index("doc2", []string{"delta epsilon"})

	// Size should reflect number of unique terms
	size := ti.Size()
	if size == 0 {
		t.Error("Expected size > 0 after indexing")
	}
}

func TestTextIndex_Concurrent(t *testing.T) {
	ti := NewTextIndex("test_idx", []string{"content"})

	// Index initial document
	ti.Index("doc1", []string{"concurrent access test"})

	// Test concurrent reads
	done := make(chan bool, 3)
	for i := 0; i < 3; i++ {
		go func() {
			results := ti.Search("concurrent")
			if len(results) == 0 {
				t.Error("Expected to find documents in concurrent search")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Test concurrent writes
	for i := 0; i < 3; i++ {
		go func(id int) {
			ti.Index("doc"+string(rune('2'+id)), []string{"concurrent write test"})
			done <- true
		}(i)
	}

	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify all documents are indexed
	results := ti.Search("concurrent")
	if len(results) < 1 {
		t.Error("Expected to find documents after concurrent writes")
	}
}

func TestTextIndex_RelevanceScoring(t *testing.T) {
	ti := NewTextIndex("test_idx", []string{"content"})

	// Index documents with different relevance
	ti.Index("doc1", []string{"database database database"}) // High frequency
	ti.Index("doc2", []string{"database systems"})            // Lower frequency
	ti.Index("doc3", []string{"web development"})             // No match

	results := ti.Search("database")
	if len(results) < 2 {
		t.Fatalf("Expected at least 2 results, got %d", len(results))
	}

	// Results should be sorted by score (highest first)
	for i := 0; i < len(results)-1; i++ {
		if results[i].Score < results[i+1].Score {
			t.Error("Results should be sorted by score in descending order")
		}
	}

	// Document with more occurrences should have higher score
	if results[0].DocID != "doc1" {
		t.Errorf("Expected doc1 (with highest frequency) to be first, got %s", results[0].DocID)
	}
}
