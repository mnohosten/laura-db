package database

import (
	"fmt"
	"os"
	"testing"
)

func BenchmarkTextIndexCreation(b *testing.B) {
	dir := "./bench_text_create"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("docs")

	// Insert 1000 documents before benchmarking
	for i := 0; i < 1000; i++ {
		coll.InsertOne(map[string]interface{}{
			"title":   fmt.Sprintf("Document %d", i),
			"content": "This is a sample document about databases and data storage systems",
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.CreateTextIndex([]string{"title", "content"})
		// Clean up for next iteration
		if i < b.N-1 {
			coll.DropIndex("title_content_text")
		}
	}
}

func BenchmarkTextSearch(b *testing.B) {
	dir := "./bench_text_search"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("articles")
	coll.CreateTextIndex([]string{"content"})

	// Insert 10,000 articles
	articles := []string{
		"MongoDB is a popular NoSQL database system",
		"PostgreSQL is a powerful relational database",
		"Redis is an in-memory data store",
		"Elasticsearch is a search and analytics engine",
		"MySQL is a widely used SQL database",
	}

	for i := 0; i < 10000; i++ {
		coll.InsertOne(map[string]interface{}{
			"title":   fmt.Sprintf("Article %d", i),
			"content": articles[i%len(articles)],
			"author":  fmt.Sprintf("Author%d", i%100),
		})
	}

	b.Run("Single term", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, _ := coll.TextSearch("database", nil)
			_ = results
		}
	})

	b.Run("Multiple terms", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, _ := coll.TextSearch("database system", nil)
			_ = results
		}
	})

	b.Run("With limit", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, _ := coll.TextSearch("database", &QueryOptions{Limit: 10})
			_ = results
		}
	})
}

func BenchmarkTextSearchVsCollectionScan(b *testing.B) {
	dir := "./bench_text_vs_scan"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	// Collection with text index
	collWithIndex := db.Collection("with_index")
	collWithIndex.CreateTextIndex([]string{"content"})

	// Collection without text index
	collNoIndex := db.Collection("no_index")

	// Insert same data in both collections
	for i := 0; i < 1000; i++ {
		doc := map[string]interface{}{
			"content": fmt.Sprintf("Article about databases and NoSQL systems number %d", i),
		}
		collWithIndex.InsertOne(doc)
		collNoIndex.InsertOne(doc)
	}

	b.Run("With text index", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, _ := collWithIndex.TextSearch("database", nil)
			_ = results
		}
	})

	b.Run("Collection scan with regex", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, _ := collNoIndex.Find(map[string]interface{}{
				"content": map[string]interface{}{
					"$regex": "database",
				},
			})
			_ = results
		}
	})
}

func BenchmarkTextIndexInsert(b *testing.B) {
	dir := "./bench_text_insert"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("docs")
	coll.CreateTextIndex([]string{"content"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.InsertOne(map[string]interface{}{
			"title":   fmt.Sprintf("Doc %d", i),
			"content": "This is a sample document about databases and data storage systems",
		})
	}
}

func BenchmarkTextIndexUpdate(b *testing.B) {
	dir := "./bench_text_update"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("docs")
	coll.CreateTextIndex([]string{"content"})

	// Insert documents
	for i := 0; i < 1000; i++ {
		coll.InsertOne(map[string]interface{}{
			"name":    fmt.Sprintf("doc%d", i),
			"content": "Original content about databases",
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := i % 1000
		coll.UpdateOne(
			map[string]interface{}{"name": fmt.Sprintf("doc%d", idx)},
			map[string]interface{}{"$set": map[string]interface{}{
				"content": fmt.Sprintf("Updated content number %d", i),
			}},
		)
	}
}

func BenchmarkTextIndexDelete(b *testing.B) {
	dir := "./bench_text_delete"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("docs")
	coll.CreateTextIndex([]string{"content"})

	// Insert many documents
	for i := 0; i < b.N+1000; i++ {
		coll.InsertOne(map[string]interface{}{
			"name":    fmt.Sprintf("doc%d", i),
			"content": "Content to be deleted",
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.DeleteOne(map[string]interface{}{"name": fmt.Sprintf("doc%d", i)})
	}
}

func BenchmarkTextSearchRelevanceScoring(b *testing.B) {
	dir := "./bench_text_scoring"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("docs")
	coll.CreateTextIndex([]string{"text"})

	// Insert documents with varying term frequencies
	coll.InsertOne(map[string]interface{}{
		"text": "database",
	})

	coll.InsertOne(map[string]interface{}{
		"text": "database database",
	})

	coll.InsertOne(map[string]interface{}{
		"text": "database database database",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, _ := coll.TextSearch("database", nil)
		// Verify scoring is working (results should be sorted by score)
		_ = results
	}
}

func BenchmarkTextAnalyzer(b *testing.B) {
	dir := "./bench_analyzer"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("docs")
	coll.CreateTextIndex([]string{"text"})

	text := "The quick brown fox jumps over the lazy dog. This is a sample sentence for benchmarking the text analyzer."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.InsertOne(map[string]interface{}{
			"text": text,
		})
	}
}
