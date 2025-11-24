package query

import (
	"runtime"
	"sync"

	"github.com/mnohosten/laura-db/pkg/document"
)

// ParallelConfig holds configuration for parallel query execution
type ParallelConfig struct {
	// MinDocsForParallel is the minimum number of documents to use parallel execution
	MinDocsForParallel int
	// MaxWorkers is the maximum number of parallel workers (0 = NumCPU)
	MaxWorkers int
	// ChunkSize is the number of documents per worker chunk (0 = auto-calculate)
	ChunkSize int
}

// DefaultParallelConfig returns a sensible default configuration
func DefaultParallelConfig() *ParallelConfig {
	return &ParallelConfig{
		MinDocsForParallel: 1000,     // Only use parallel for 1000+ documents
		MaxWorkers:         0,         // Use all available CPUs
		ChunkSize:          0,         // Auto-calculate based on document count
	}
}

// ExecuteParallel executes a query in parallel and returns matching documents
func (e *Executor) ExecuteParallel(query *Query, config *ParallelConfig) ([]*document.Document, error) {
	// Use default config if not provided
	if config == nil {
		config = DefaultParallelConfig()
	}

	// If document count is below threshold, use sequential execution
	if len(e.documents) < config.MinDocsForParallel {
		return e.Execute(query)
	}

	// Determine number of workers
	workers := config.MaxWorkers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	// Calculate chunk size
	chunkSize := config.ChunkSize
	if chunkSize <= 0 {
		chunkSize = (len(e.documents) + workers - 1) / workers
		// Ensure minimum chunk size to avoid too much overhead
		if chunkSize < 100 {
			chunkSize = 100
		}
	}

	// Filter documents in parallel
	results := e.parallelFilter(query, workers, chunkSize)

	// Sort results
	if len(query.GetSort()) > 0 {
		e.sortDocuments(results, query.GetSort())
	}

	// Apply skip
	if query.GetSkip() > 0 {
		if query.GetSkip() >= len(results) {
			results = []*document.Document{}
		} else {
			results = results[query.GetSkip():]
		}
	}

	// Apply limit
	if query.GetLimit() > 0 && query.GetLimit() < len(results) {
		results = results[:query.GetLimit()]
	}

	// Apply projection
	for i, doc := range results {
		results[i] = query.ApplyProjection(doc)
	}

	return results, nil
}

// ExecuteWithPlanParallel executes a query using a query plan in parallel
func (e *Executor) ExecuteWithPlanParallel(query *Query, plan *QueryPlan, config *ParallelConfig) ([]*document.Document, error) {
	// Use default config if not provided
	if config == nil {
		config = DefaultParallelConfig()
	}

	// Check if this is a covered query (can be satisfied entirely from index)
	if plan.IsCovered {
		// Covered queries are already optimized, use sequential execution
		return e.executeCoveredQuery(query, plan)
	}

	var candidates []*document.Document

	if plan.UseIndex && plan.Index != nil {
		// Use index to get candidate documents
		var err error
		candidates, err = e.executeIndexScan(plan)
		if err != nil {
			// Fall back to collection scan if index scan fails
			candidates = e.documents
		}
	} else {
		// Full collection scan
		candidates = e.documents
	}

	// If candidate count is below threshold, use sequential execution
	if len(candidates) < config.MinDocsForParallel {
		return e.executeWithCandidates(query, candidates)
	}

	// Determine number of workers
	workers := config.MaxWorkers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	// Calculate chunk size
	chunkSize := config.ChunkSize
	if chunkSize <= 0 {
		chunkSize = (len(candidates) + workers - 1) / workers
		if chunkSize < 100 {
			chunkSize = 100
		}
	}

	// Filter candidates in parallel
	results := e.parallelFilterDocs(query, candidates, workers, chunkSize)

	// Sort results
	if len(query.GetSort()) > 0 {
		e.sortDocuments(results, query.GetSort())
	}

	// Apply skip
	if query.GetSkip() > 0 {
		if query.GetSkip() >= len(results) {
			results = []*document.Document{}
		} else {
			results = results[query.GetSkip():]
		}
	}

	// Apply limit
	if query.GetLimit() > 0 && query.GetLimit() < len(results) {
		results = results[:query.GetLimit()]
	}

	// Apply projection
	for i, doc := range results {
		results[i] = query.ApplyProjection(doc)
	}

	return results, nil
}

// parallelFilter filters documents in parallel
func (e *Executor) parallelFilter(query *Query, workers int, chunkSize int) []*document.Document {
	return e.parallelFilterDocs(query, e.documents, workers, chunkSize)
}

// parallelFilterDocs filters a specific set of documents in parallel
func (e *Executor) parallelFilterDocs(query *Query, docs []*document.Document, workers int, chunkSize int) []*document.Document {
	// Create channels for work distribution
	type chunk struct {
		start int
		end   int
	}

	chunks := make(chan chunk, workers)
	resultChunks := make(chan []*document.Document, workers)
	errors := make(chan error, workers)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ch := range chunks {
				localResults := make([]*document.Document, 0)
				for j := ch.start; j < ch.end; j++ {
					doc := docs[j]
					matches, err := query.Matches(doc)
					if err != nil {
						errors <- err
						return
					}
					if matches {
						localResults = append(localResults, doc)
					}
				}
				resultChunks <- localResults
			}
		}()
	}

	// Distribute work
	go func() {
		for i := 0; i < len(docs); i += chunkSize {
			end := i + chunkSize
			if end > len(docs) {
				end = len(docs)
			}
			chunks <- chunk{start: i, end: end}
		}
		close(chunks)
	}()

	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(resultChunks)
		close(errors)
	}()

	// Check for errors
	select {
	case err := <-errors:
		if err != nil {
			// If there's an error, return empty results
			// In a production system, you might want to handle this differently
			return []*document.Document{}
		}
	default:
	}

	// Collect results
	results := make([]*document.Document, 0)
	for localResults := range resultChunks {
		results = append(results, localResults...)
	}

	return results
}

// executeWithCandidates is a helper for sequential execution with candidates
func (e *Executor) executeWithCandidates(query *Query, candidates []*document.Document) ([]*document.Document, error) {
	// Filter candidates
	results := make([]*document.Document, 0)
	for _, doc := range candidates {
		matches, err := query.Matches(doc)
		if err != nil {
			return nil, err
		}
		if matches {
			results = append(results, doc)
		}
	}

	// Sort results
	if len(query.GetSort()) > 0 {
		e.sortDocuments(results, query.GetSort())
	}

	// Apply skip
	if query.GetSkip() > 0 {
		if query.GetSkip() >= len(results) {
			results = []*document.Document{}
		} else {
			results = results[query.GetSkip():]
		}
	}

	// Apply limit
	if query.GetLimit() > 0 && query.GetLimit() < len(results) {
		results = results[:query.GetLimit()]
	}

	// Apply projection
	for i, doc := range results {
		results[i] = query.ApplyProjection(doc)
	}

	return results, nil
}
