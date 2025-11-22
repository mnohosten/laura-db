package database

import (
	"os"
	"testing"
)

func TestTextIndexCreation(t *testing.T) {
	dir := "./test_text_idx"
	defer os.RemoveAll(dir)

	db, err := Open(DefaultConfig(dir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("articles")

	// Create text index on title and content
	err = coll.CreateTextIndex([]string{"title", "content"})
	if err != nil {
		t.Fatalf("Failed to create text index: %v", err)
	}

	// Insert some documents
	_, err = coll.InsertOne(map[string]interface{}{
		"title":   "Introduction to Databases",
		"content": "Databases are systems for storing and retrieving data efficiently.",
		"author":  "Alice",
	})
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	_, err = coll.InsertOne(map[string]interface{}{
		"title":   "NoSQL Databases Explained",
		"content": "NoSQL databases provide flexible schema and horizontal scalability.",
		"author":  "Bob",
	})
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	_, err = coll.InsertOne(map[string]interface{}{
		"title":   "Machine Learning Basics",
		"content": "Machine learning is a subset of artificial intelligence.",
		"author":  "Charlie",
	})
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Test text search
	results, err := coll.TextSearch("databases", nil)
	if err != nil {
		t.Fatalf("Text search failed: %v", err)
	}

	// Should find both database-related articles
	if len(results) < 2 {
		t.Errorf("Expected at least 2 results for 'databases', got %d", len(results))
	}

	// First result should have the highest relevance score
	if len(results) > 0 {
		score1, _ := results[0].Get("_textScore")
		if score1 == nil {
			t.Error("Expected _textScore field in results")
		}

		// Verify results are sorted by score (descending)
		if len(results) > 1 {
			score2, _ := results[1].Get("_textScore")
			if score2 != nil && score1.(float64) < score2.(float64) {
				t.Error("Results should be sorted by relevance score (descending)")
			}
		}
	}
}

func TestTextSearchWithMultipleTerms(t *testing.T) {
	dir := "./test_text_multi"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("docs")

	// Create text index
	coll.CreateTextIndex([]string{"text"})

	// Insert documents
	coll.InsertOne(map[string]interface{}{
		"text": "The quick brown fox jumps over the lazy dog",
	})

	coll.InsertOne(map[string]interface{}{
		"text": "A quick brown animal",
	})

	coll.InsertOne(map[string]interface{}{
		"text": "The lazy cat sleeps",
	})

	// Search for multiple terms
	results, _ := coll.TextSearch("quick brown", nil)

	// Both documents with "quick" and "brown" should rank higher
	if len(results) == 0 {
		t.Fatal("Expected some results")
	}

	// Document with both terms should rank highest
	firstDoc := results[0]
	text, _ := firstDoc.Get("text")
	// Should be one of the first two documents
	if text != "The quick brown fox jumps over the lazy dog" && text != "A quick brown animal" {
		t.Error("Expected document with both search terms to rank high")
	}
}

func TestTextSearchWithStopWords(t *testing.T) {
	dir := "./test_text_stop"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("docs")
	coll.CreateTextIndex([]string{"text"})

	coll.InsertOne(map[string]interface{}{
		"text": "MongoDB is a NoSQL database",
	})

	coll.InsertOne(map[string]interface{}{
		"text": "PostgreSQL is a SQL database",
	})

	// Search with stop words - they should be filtered out
	results, _ := coll.TextSearch("the database", nil)

	// Should find both documents (stop word "the" filtered, "database" matches)
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestTextSearchWithStemming(t *testing.T) {
	dir := "./test_text_stem"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("docs")
	coll.CreateTextIndex([]string{"text"})

	coll.InsertOne(map[string]interface{}{
		"text": "The developer is developing software",
	})

	// Search for "develop" - should match "developer" and "developing" (stemmed)
	results, _ := coll.TextSearch("develop", nil)

	if len(results) != 1 {
		t.Errorf("Expected 1 result with stemming, got %d", len(results))
	}
}

func TestTextSearchWithProjection(t *testing.T) {
	dir := "./test_text_proj"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("articles")
	coll.CreateTextIndex([]string{"content"})

	coll.InsertOne(map[string]interface{}{
		"title":   "Article 1",
		"content": "database systems",
		"author":  "Alice",
	})

	// Search with projection
	results, _ := coll.TextSearch("database", &QueryOptions{
		Projection: map[string]bool{
			"title": true,
		},
	})

	if len(results) == 0 {
		t.Fatal("Expected results")
	}

	doc := results[0]

	// Should have title and _id
	if _, exists := doc.Get("title"); !exists {
		t.Error("Expected title field in projection")
	}

	if _, exists := doc.Get("_id"); !exists {
		t.Error("Expected _id field (included by default)")
	}

	// Should NOT have author (not in projection)
	if _, exists := doc.Get("author"); exists {
		t.Error("Did not expect author field in projection")
	}
}

func TestTextSearchWithSkipAndLimit(t *testing.T) {
	dir := "./test_text_skip"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("docs")
	coll.CreateTextIndex([]string{"text"})

	// Insert 10 documents with the word "test"
	for i := 0; i < 10; i++ {
		coll.InsertOne(map[string]interface{}{
			"text": "This is a test document",
			"num":  i,
		})
	}

	// Test limit
	results, _ := coll.TextSearch("test", &QueryOptions{Limit: 5})
	if len(results) != 5 {
		t.Errorf("Expected 5 results with limit, got %d", len(results))
	}

	// Test skip
	results, _ = coll.TextSearch("test", &QueryOptions{Skip: 8})
	if len(results) != 2 {
		t.Errorf("Expected 2 results with skip=8, got %d", len(results))
	}

	// Test skip + limit
	results, _ = coll.TextSearch("test", &QueryOptions{Skip: 3, Limit: 4})
	if len(results) != 4 {
		t.Errorf("Expected 4 results with skip=3, limit=4, got %d", len(results))
	}
}

func TestTextIndexUpdate(t *testing.T) {
	dir := "./test_text_update"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("docs")
	coll.CreateTextIndex([]string{"text"})

	// Insert document
	coll.InsertOne(map[string]interface{}{
		"name": "doc1",
		"text": "original content",
	})

	// Search for original content
	results, _ := coll.TextSearch("original", nil)
	if len(results) != 1 {
		t.Fatalf("Expected 1 result for original content")
	}

	// Update the text
	coll.UpdateOne(
		map[string]interface{}{"name": "doc1"},
		map[string]interface{}{"$set": map[string]interface{}{"text": "updated content"}},
	)

	// Search for original content - should find nothing
	results, _ = coll.TextSearch("original", nil)
	if len(results) != 0 {
		t.Errorf("Expected 0 results for old content after update, got %d", len(results))
	}

	// Search for updated content - should find it
	results, _ = coll.TextSearch("updated", nil)
	if len(results) != 1 {
		t.Errorf("Expected 1 result for updated content, got %d", len(results))
	}
}

func TestTextIndexDelete(t *testing.T) {
	dir := "./test_text_delete"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("docs")
	coll.CreateTextIndex([]string{"text"})

	// Insert documents
	coll.InsertOne(map[string]interface{}{
		"name": "doc1",
		"text": "deletable content",
	})

	coll.InsertOne(map[string]interface{}{
		"name": "doc2",
		"text": "persistent content",
	})

	// Delete one document
	coll.DeleteOne(map[string]interface{}{"name": "doc1"})

	// Search for deleted content
	results, _ := coll.TextSearch("deletable", nil)
	if len(results) != 0 {
		t.Errorf("Expected 0 results for deleted content, got %d", len(results))
	}

	// Search for remaining content
	results, _ = coll.TextSearch("persistent", nil)
	if len(results) != 1 {
		t.Errorf("Expected 1 result for remaining content, got %d", len(results))
	}
}

func TestTextSearchNoIndex(t *testing.T) {
	dir := "./test_text_noindex"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("docs")

	// Try to search without creating a text index
	_, err := coll.TextSearch("test", nil)
	if err == nil {
		t.Error("Expected error when searching without text index")
	}
}

func TestMultipleTextFields(t *testing.T) {
	dir := "./test_text_multi_field"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("articles")
	coll.CreateTextIndex([]string{"title", "content", "tags"})

	// Insert document with multiple text fields
	coll.InsertOne(map[string]interface{}{
		"title":   "Database Tutorial",
		"content": "Learn about NoSQL databases",
		"tags":    "mongodb database nosql",
	})

	// Search should match across all indexed fields
	results, _ := coll.TextSearch("database", nil)
	if len(results) != 1 {
		t.Errorf("Expected 1 result matching across multiple fields, got %d", len(results))
	}

	// Search for term in tags
	results, _ = coll.TextSearch("mongodb", nil)
	if len(results) != 1 {
		t.Errorf("Expected 1 result matching tags field, got %d", len(results))
	}
}
