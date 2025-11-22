package text

import (
	"math"
	"sort"
	"sync"
)

// InvertedIndex maps tokens to document IDs with term frequencies
type InvertedIndex struct {
	mu sync.RWMutex

	// token -> document postings
	index map[string]*PostingsList

	// document ID -> document length (number of tokens)
	docLengths map[string]int

	// Total number of documents
	totalDocs int

	// Average document length
	avgDocLength float64

	analyzer *Analyzer
}

// PostingsList contains all documents that contain a specific term
type PostingsList struct {
	// document ID -> term frequency
	Postings map[string]int

	// Document frequency (how many documents contain this term)
	DocFreq int
}

// NewInvertedIndex creates a new inverted index
func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{
		index:      make(map[string]*PostingsList),
		docLengths: make(map[string]int),
		totalDocs:  0,
		analyzer:   NewAnalyzer(),
	}
}

// Index adds a document to the inverted index
func (idx *InvertedIndex) Index(docID string, text string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Analyze text into tokens
	tokens := idx.analyzer.Analyze(text)

	// Count term frequencies in this document
	termFreqs := make(map[string]int)
	for _, token := range tokens {
		termFreqs[token]++
	}

	// Update inverted index
	for token, freq := range termFreqs {
		if idx.index[token] == nil {
			idx.index[token] = &PostingsList{
				Postings: make(map[string]int),
				DocFreq:  0,
			}
		}

		postings := idx.index[token]

		// If this is a new document for this token, increment doc frequency
		if _, exists := postings.Postings[docID]; !exists {
			postings.DocFreq++
		}

		postings.Postings[docID] = freq
	}

	// Update document length
	idx.docLengths[docID] = len(tokens)
	idx.totalDocs++

	// Recalculate average document length
	totalLength := 0
	for _, length := range idx.docLengths {
		totalLength += length
	}
	if idx.totalDocs > 0 {
		idx.avgDocLength = float64(totalLength) / float64(idx.totalDocs)
	}
}

// Remove removes a document from the inverted index
func (idx *InvertedIndex) Remove(docID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Find all tokens for this document and update postings
	for token, postings := range idx.index {
		if _, exists := postings.Postings[docID]; exists {
			delete(postings.Postings, docID)
			postings.DocFreq--

			// Remove posting list if empty
			if postings.DocFreq == 0 {
				delete(idx.index, token)
			}
		}
	}

	// Remove document length
	delete(idx.docLengths, docID)
	idx.totalDocs--

	// Recalculate average document length
	if idx.totalDocs > 0 {
		totalLength := 0
		for _, length := range idx.docLengths {
			totalLength += length
		}
		idx.avgDocLength = float64(totalLength) / float64(idx.totalDocs)
	} else {
		idx.avgDocLength = 0
	}
}

// Search searches for documents matching the query
func (idx *InvertedIndex) Search(query string) []SearchResult {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Analyze query into tokens
	tokens := idx.analyzer.Analyze(query)

	if len(tokens) == 0 {
		return nil
	}

	// Find documents containing any of the query tokens
	docScores := make(map[string]float64)

	for _, token := range tokens {
		postings := idx.index[token]
		if postings == nil {
			continue
		}

		// Calculate scores for all documents containing this token
		for docID, termFreq := range postings.Postings {
			// BM25 scoring
			score := idx.calculateBM25(token, docID, termFreq, postings.DocFreq)
			docScores[docID] += score
		}
	}

	// Convert to sorted results
	results := make([]SearchResult, 0, len(docScores))
	for docID, score := range docScores {
		results = append(results, SearchResult{
			DocID: docID,
			Score: score,
		})
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// calculateBM25 calculates BM25 relevance score
// BM25 is an improved version of TF-IDF
func (idx *InvertedIndex) calculateBM25(token, docID string, termFreq, docFreq int) float64 {
	// BM25 parameters
	k1 := 1.5  // term frequency saturation parameter
	b := 0.75  // length normalization parameter

	// Document length normalization
	docLength := float64(idx.docLengths[docID])
	avgDocLength := idx.avgDocLength
	if avgDocLength == 0 {
		avgDocLength = 1
	}

	// IDF (Inverse Document Frequency)
	// log((N - df + 0.5) / (df + 0.5) + 1)
	N := float64(idx.totalDocs)
	df := float64(docFreq)
	idf := math.Log((N-df+0.5)/(df+0.5) + 1.0)

	// TF component with saturation
	tf := float64(termFreq)
	lengthNorm := 1.0 - b + b*(docLength/avgDocLength)
	tfComponent := (tf * (k1 + 1.0)) / (tf + k1*lengthNorm)

	return idf * tfComponent
}

// SearchResult represents a document with its relevance score
type SearchResult struct {
	DocID string
	Score float64
}

// Stats returns statistics about the inverted index
func (idx *InvertedIndex) Stats() map[string]interface{} {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return map[string]interface{}{
		"total_documents":    idx.totalDocs,
		"total_terms":        len(idx.index),
		"avg_document_length": idx.avgDocLength,
	}
}

// Size returns the number of unique terms in the index
func (idx *InvertedIndex) Size() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.index)
}
