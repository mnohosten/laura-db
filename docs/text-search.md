# Text Search

## Overview

Laura-DB provides full-text search capabilities through text indexes and the `TextSearch` API. The implementation uses an inverted index data structure with BM25 relevance scoring for efficient and accurate text retrieval.

## Features

- **Tokenization**: Automatic word extraction with punctuation handling
- **Stop Word Filtering**: Removes common words (a, the, and, etc.) that don't add search value
- **Stemming**: Porter stemmer reduces words to their root form (running → run, databases → databas)
- **BM25 Scoring**: Advanced relevance ranking algorithm (improvement over TF-IDF)
- **Multi-Field Indexing**: Index multiple text fields together
- **Automatic Maintenance**: Indexes updated on insert, update, and delete operations

## Creating Text Indexes

### Single Field

```go
coll := db.Collection("articles")

// Create text index on the "content" field
err := coll.CreateTextIndex([]string{"content"})
```

### Multiple Fields

```go
// Index multiple text fields together
err := coll.CreateTextIndex([]string{"title", "content", "tags"})
```

Text fields are combined during indexing, so a search will match terms from any of the indexed fields.

## Performing Text Searches

### Basic Search

```go
// Search for documents containing "database"
results, err := coll.TextSearch("database", nil)

for _, doc := range results {
	title, _ := doc.Get("title")
	score, _ := doc.Get("_textScore")
	fmt.Printf("Title: %s, Score: %.2f\n", title, score)
}
```

Results are automatically sorted by relevance score (highest first). Each result includes a `_textScore` field with the BM25 relevance score.

### Multi-Word Search

```go
// Search for documents containing "nosql database"
// Documents matching both terms will rank higher
results, err := coll.TextSearch("nosql database", nil)
```

### Search with Options

```go
// Search with projection, limit, and skip
results, err := coll.TextSearch("database", &QueryOptions{
	Projection: map[string]bool{
		"title":  true,
		"author": true,
	},
	Limit: 10,
	Skip:  0,
})
```

## How Text Search Works

### 1. Tokenization

Text is split into words, removing punctuation:

```
"Hello, world! How are you?" → ["Hello", "world", "How", "are", "you"]
```

### 2. Normalization

All tokens are converted to lowercase:

```
["Hello", "world"] → ["hello", "world"]
```

### 3. Stop Word Filtering

Common words that don't add meaning are removed:

```
["hello", "world", "how", "are", "you"] → ["hello", "world"]
// "how", "are", "you" are stop words
```

### 4. Stemming

Words are reduced to their root form using the Porter stemmer:

```
["running", "databases", "quickly"] → ["run", "databas", "quickli"]
```

This allows searches to match different word forms:
- Searching for "run" matches "running", "runs", "ran"
- Searching for "database" matches "databases", "database's"

### 5. Inverted Index Lookup

The inverted index maps each term to the documents containing it:

```
"databas" → {doc1: freq=3, doc2: freq=1, doc3: freq=2}
"nosql"   → {doc1: freq=1, doc3: freq=1}
```

### 6. BM25 Scoring

Each document receives a relevance score based on:
- **Term Frequency (TF)**: How often the term appears in the document
- **Inverse Document Frequency (IDF)**: How rare the term is across all documents
- **Document Length Normalization**: Favors shorter documents with the same term frequency

BM25 Formula:
```
score(D,Q) = Σ IDF(qi) × (f(qi,D) × (k1 + 1)) / (f(qi,D) + k1 × (1 - b + b × |D| / avgdl))
```

Where:
- `D` = document
- `Q` = query
- `qi` = query term i
- `f(qi,D)` = frequency of term qi in document D
- `|D|` = length of document D (number of terms)
- `avgdl` = average document length
- `k1` = 1.5 (term frequency saturation parameter)
- `b` = 0.75 (length normalization parameter)

## Examples

### Article Search

```go
coll := db.Collection("articles")

// Create text index
coll.CreateTextIndex([]string{"title", "content"})

// Insert articles
coll.InsertOne(map[string]interface{}{
	"title":   "Introduction to NoSQL Databases",
	"content": "NoSQL databases provide flexible schemas and horizontal scalability.",
	"author":  "Alice",
})

coll.InsertOne(map[string]interface{}{
	"title":   "SQL vs NoSQL",
	"content": "Understanding the differences between SQL and NoSQL databases.",
	"author":  "Bob",
})

coll.InsertOne(map[string]interface{}{
	"title":   "Machine Learning Basics",
	"content": "An introduction to machine learning concepts and algorithms.",
	"author":  "Charlie",
})

// Search for "nosql"
results, _ := coll.TextSearch("nosql", nil)

// Results sorted by relevance:
// 1. "Introduction to NoSQL Databases" (highest score)
// 2. "SQL vs NoSQL" (medium score)
// (Machine Learning article not returned)
```

### Product Search

```go
coll := db.Collection("products")

// Index product fields
coll.CreateTextIndex([]string{"name", "description", "category"})

// Insert products
coll.InsertOne(map[string]interface{}{
	"name":        "Wireless Mouse",
	"description": "Ergonomic wireless mouse with precision tracking",
	"category":    "Electronics",
	"price":       29.99,
})

// Search products
results, _ := coll.TextSearch("wireless mouse", &QueryOptions{
	Limit: 20,
})

for _, doc := range results {
	name, _ := doc.Get("name")
	score, _ := doc.Get("_textScore")
	fmt.Printf("%s (score: %.2f)\n", name, score)
}
```

### Blog Search with Pagination

```go
coll := db.Collection("posts")
coll.CreateTextIndex([]string{"title", "body"})

// Page 1 (results 1-10)
page1, _ := coll.TextSearch("database", &QueryOptions{
	Limit: 10,
	Skip:  0,
})

// Page 2 (results 11-20)
page2, _ := coll.TextSearch("database", &QueryOptions{
	Limit: 10,
	Skip:  10,
})
```

## Index Maintenance

Text indexes are automatically maintained:

### Insert

```go
// Document is automatically indexed
coll.InsertOne(map[string]interface{}{
	"title": "New Article About Databases",
})
```

### Update

```go
// Old index entries removed, new ones added
coll.UpdateOne(
	map[string]interface{}{"_id": docID},
	map[string]interface{}{
		"$set": map[string]interface{}{
			"title": "Updated Article Title",
		},
	},
)
```

### Delete

```go
// Document removed from text index
coll.DeleteOne(map[string]interface{}{"_id": docID})
```

## Performance Characteristics

### Time Complexity

| Operation | Complexity | Notes |
|-----------|------------|-------|
| Index Creation | O(N × M) | N = documents, M = avg terms per document |
| Insert | O(M) | M = terms in document |
| Search | O(T + K) | T = unique query terms, K = results |
| Update | O(M) | Remove old + add new |
| Delete | O(M) | M = terms in document |

### Benchmarks

Based on 10,000 documents:

```
BenchmarkTextSearch/Single_term              22    73.7ms/op    6.6 MB/op
BenchmarkTextSearch/Multiple_terms           37    29.4ms/op    6.6 MB/op
BenchmarkTextSearchVsCollectionScan:
  With_text_index                           295     4.1ms/op    993 KB/op
  Collection_scan_with_regex                156     7.9ms/op   2145 KB/op
```

**Text Index vs Collection Scan**:
- **1.9x faster** query execution
- **54% less memory** usage

## Best Practices

### 1. Choose the Right Fields

Index fields that users will search:
```go
// Good: User-facing search fields
CreateTextIndex([]string{"title", "description", "tags"})

// Poor: Internal fields not used in search
CreateTextIndex([]string{"internal_id", "created_at"})
```

### 2. Limit Number of Indexed Fields

More fields = larger index and slower updates:
```go
// Good: 2-3 main content fields
CreateTextIndex([]string{"title", "content"})

// Poor: Too many fields
CreateTextIndex([]string{"title", "content", "meta1", "meta2", "meta3", ...})
```

### 3. Use Projection for Large Documents

Return only needed fields:
```go
results, _ := coll.TextSearch("query", &QueryOptions{
	Projection: map[string]bool{
		"title":  true,
		"author": true,
		// Exclude large "content" field
	},
})
```

### 4. Implement Pagination

Don't return all results at once:
```go
// Good: Paginated results
coll.TextSearch("query", &QueryOptions{
	Limit: 20,
	Skip:  page * 20,
})

// Poor: Unbounded results
coll.TextSearch("query", nil)
```

### 5. Analyze Indexes Periodically

Keep statistics fresh for optimization:
```go
// Recalculate index statistics
coll.Analyze()
```

## Limitations

### Current Limitations

1. **Language**: English stop words and stemming only
2. **Phrase Matching**: No support for exact phrase searches ("word1 word2")
3. **Wildcards**: No wildcard support (prefix*, *suffix)
4. **Fuzzy Matching**: No fuzzy/approximate matching
5. **Field Weighting**: All indexed fields weighted equally
6. **Minimum Word Length**: Words with < 2 characters are filtered out

### Future Enhancements

- Multi-language support
- Phrase queries
- Proximity searches
- Fuzzy matching (Levenshtein distance)
- Field-specific weighting
- Query suggestions
- Highlighting of matched terms

## Comparison with Other Databases

### MongoDB

```go
// MongoDB
db.collection.createIndex({ content: "text" })
db.collection.find({ $text: { $search: "database" } })

// Laura-DB
coll.CreateTextIndex([]string{"content"})
coll.TextSearch("database", nil)
```

### Elasticsearch

Laura-DB's text search is simpler than Elasticsearch but covers common use cases:
- ✅ Full-text search
- ✅ Relevance scoring
- ✅ Stop words and stemming
- ❌ Advanced query DSL
- ❌ Faceted search
- ❌ Geographic search

## Troubleshooting

### No Results Found

Check if text index exists:
```go
indexes := coll.ListIndexes()
for _, idx := range indexes {
	if idx["type"] == "text" {
		fmt.Printf("Text index: %s\n", idx["name"])
	}
}
```

### Poor Relevance Ranking

- Ensure documents have substantive content (not just titles)
- Try multi-term queries for better ranking
- Consider the impact of stop words (very common words are filtered)

### Slow Searches

- Add pagination (limit results)
- Use projection to exclude large fields
- Check index statistics with `coll.Analyze()`
- Consider if you need all indexed fields

## Summary

Laura-DB's text search provides:
- Easy-to-use API for full-text search
- Automatic tokenization, stemming, and normalization
- BM25 relevance scoring for accurate ranking
- Efficient inverted index implementation
- Automatic index maintenance
- 1.9x performance improvement over regex-based search

Perfect for applications needing search functionality without external search engines!
