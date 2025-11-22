package text

import (
	"testing"
)

func TestInvertedIndexBasic(t *testing.T) {
	idx := NewInvertedIndex()

	// Index some documents
	idx.Index("doc1", "The quick brown fox jumps over the lazy dog")
	idx.Index("doc2", "The lazy dog sleeps all day")
	idx.Index("doc3", "A quick brown fox is very fast")

	// Test search
	results := idx.Search("quick fox")

	// Both doc1 and doc3 should match
	if len(results) < 2 {
		t.Fatalf("Expected at least 2 results, got %d", len(results))
	}

	// doc1 and doc3 should be in results
	foundDoc1 := false
	foundDoc3 := false
	for _, result := range results {
		if result.DocID == "doc1" {
			foundDoc1 = true
		}
		if result.DocID == "doc3" {
			foundDoc3 = true
		}
	}

	if !foundDoc1 {
		t.Error("Expected to find doc1 in results")
	}
	if !foundDoc3 {
		t.Error("Expected to find doc3 in results")
	}

	// doc3 should have higher score (contains both "quick" and "fox", and has no irrelevant terms)
	if len(results) >= 2 {
		// First result should have highest score
		if results[0].Score <= 0 {
			t.Error("Expected positive scores")
		}
	}
}

func TestInvertedIndexRemove(t *testing.T) {
	idx := NewInvertedIndex()

	idx.Index("doc1", "The quick brown fox")
	idx.Index("doc2", "A lazy dog")

	// Search should find doc1
	results := idx.Search("quick fox")
	if len(results) != 1 || results[0].DocID != "doc1" {
		t.Error("Expected to find doc1")
	}

	// Remove doc1
	idx.Remove("doc1")

	// Search should find nothing
	results = idx.Search("quick fox")
	if len(results) != 0 {
		t.Errorf("Expected no results after removal, got %d", len(results))
	}

	// doc2 should still be searchable
	results = idx.Search("lazy dog")
	if len(results) != 1 || results[0].DocID != "doc2" {
		t.Error("Expected to still find doc2")
	}
}

func TestInvertedIndexStats(t *testing.T) {
	idx := NewInvertedIndex()

	idx.Index("doc1", "The quick brown fox")
	idx.Index("doc2", "A lazy dog")
	idx.Index("doc3", "Quick jumps")

	stats := idx.Stats()

	totalDocs := stats["total_documents"].(int)
	if totalDocs != 3 {
		t.Errorf("Expected 3 documents, got %d", totalDocs)
	}

	totalTerms := stats["total_terms"].(int)
	if totalTerms == 0 {
		t.Error("Expected some terms in index")
	}
}

func TestRelevanceScoring(t *testing.T) {
	idx := NewInvertedIndex()

	// doc1 contains "database" once
	idx.Index("doc1", "This is a database system")

	// doc2 contains "database" three times
	idx.Index("doc2", "Database database database")

	// doc3 doesn't contain "database"
	idx.Index("doc3", "A completely different document")

	results := idx.Search("database")

	// Should find doc1 and doc2, but not doc3
	if len(results) < 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// doc2 should score higher than doc1 (more term frequency)
	found1 := false
	found2 := false
	var score1, score2 float64

	for _, result := range results {
		if result.DocID == "doc1" {
			found1 = true
			score1 = result.Score
		}
		if result.DocID == "doc2" {
			found2 = true
			score2 = result.Score
		}
	}

	if !found1 || !found2 {
		t.Error("Expected to find both doc1 and doc2")
	}

	if score2 <= score1 {
		t.Errorf("Expected doc2 (score=%.2f) to score higher than doc1 (score=%.2f)", score2, score1)
	}
}

func TestEmptyQuery(t *testing.T) {
	idx := NewInvertedIndex()

	idx.Index("doc1", "Some text here")

	// Empty query
	results := idx.Search("")
	if len(results) != 0 {
		t.Error("Expected no results for empty query")
	}

	// Only stop words
	results = idx.Search("the and or")
	if len(results) != 0 {
		t.Error("Expected no results for stop words only")
	}
}

func TestStemmingInSearch(t *testing.T) {
	idx := NewInvertedIndex()

	// Index with "running"
	idx.Index("doc1", "The athlete is running fast")

	// Search for "run" (stem of "running")
	results := idx.Search("run")
	if len(results) != 1 {
		t.Fatalf("Expected to find doc1 with stemmed search, got %d results", len(results))
	}

	if results[0].DocID != "doc1" {
		t.Error("Expected to find doc1")
	}

	// Search for "runs" (also stems to "run")
	results = idx.Search("runs")
	if len(results) != 1 {
		t.Fatalf("Expected to find doc1 with different form, got %d results", len(results))
	}
}

func TestMultiWordQuery(t *testing.T) {
	idx := NewInvertedIndex()

	idx.Index("doc1", "MongoDB is a NoSQL database")
	idx.Index("doc2", "PostgreSQL is a SQL database")
	idx.Index("doc3", "Redis is a cache store")

	// Search for "database"
	results := idx.Search("database")

	// Should find doc1 and doc2
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// Search for "MongoDB database"
	results = idx.Search("MongoDB database")

	// doc1 should rank highest (contains both terms)
	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	if results[0].DocID != "doc1" {
		t.Errorf("Expected doc1 to rank highest, got %s", results[0].DocID)
	}
}

func TestCaseInsensitivity(t *testing.T) {
	idx := NewInvertedIndex()

	idx.Index("doc1", "MongoDB Database")

	// Search with different case
	results := idx.Search("mongodb")
	if len(results) != 1 {
		t.Fatal("Expected case-insensitive match")
	}

	results = idx.Search("DATABASE")
	if len(results) != 1 {
		t.Fatal("Expected case-insensitive match for uppercase")
	}
}
