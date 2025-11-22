package index

import (
	"sync"

	"github.com/mnohosten/laura-db/pkg/text"
)

// TextIndex represents a text search index using an inverted index
type TextIndex struct {
	name         string
	fieldPaths   []string
	invertedIdx  *text.InvertedIndex
	stats        *IndexStats
	mu           sync.RWMutex
}

// NewTextIndex creates a new text index
func NewTextIndex(name string, fieldPaths []string) *TextIndex {
	return &TextIndex{
		name:        name,
		fieldPaths:  fieldPaths,
		invertedIdx: text.NewInvertedIndex(),
		stats:       NewIndexStats(),
	}
}

// Index adds a document to the text index
func (ti *TextIndex) Index(docID string, texts []string) {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	// Combine all text fields into one string for indexing
	combinedText := ""
	for i, text := range texts {
		if i > 0 {
			combinedText += " "
		}
		combinedText += text
	}

	ti.invertedIdx.Index(docID, combinedText)

	// Mark stats as stale
	ti.stats.Update()
}

// Remove removes a document from the text index
func (ti *TextIndex) Remove(docID string) {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	ti.invertedIdx.Remove(docID)

	// Mark stats as stale
	ti.stats.Update()
}

// Search performs a text search and returns matching document IDs with scores
func (ti *TextIndex) Search(query string) []text.SearchResult {
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	return ti.invertedIdx.Search(query)
}

// Name returns the index name
func (ti *TextIndex) Name() string {
	ti.mu.RLock()
	defer ti.mu.RUnlock()
	return ti.name
}

// FieldPaths returns the indexed field paths
func (ti *TextIndex) FieldPaths() []string {
	ti.mu.RLock()
	defer ti.mu.RUnlock()
	return ti.fieldPaths
}

// IsCompound returns false for text indexes (they are not compound indexes in the B+ tree sense)
func (ti *TextIndex) IsCompound() bool {
	return len(ti.fieldPaths) > 1
}

// Stats returns statistics about the text index
func (ti *TextIndex) Stats() map[string]interface{} {
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	stats := ti.invertedIdx.Stats()

	return map[string]interface{}{
		"name":        ti.name,
		"field_paths": ti.fieldPaths,
		"type":        "text",
		"is_compound": len(ti.fieldPaths) > 1,
		"total_documents": stats["total_documents"],
		"total_terms": stats["total_terms"],
		"avg_document_length": stats["avg_document_length"],
	}
}

// Analyze updates the index statistics
func (ti *TextIndex) Analyze() {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	stats := ti.invertedIdx.Stats()

	// Update index statistics
	totalDocs := stats["total_documents"].(int)
	totalTerms := stats["total_terms"].(int)

	ti.stats.SetStats(totalDocs, totalTerms, nil, nil)
}

// GetStatistics returns the index statistics object
func (ti *TextIndex) GetStatistics() *IndexStats {
	return ti.stats
}

// Size returns the number of unique terms in the index
func (ti *TextIndex) Size() int {
	ti.mu.RLock()
	defer ti.mu.RUnlock()
	return ti.invertedIdx.Size()
}
