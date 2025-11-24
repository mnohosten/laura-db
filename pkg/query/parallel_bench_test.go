package query

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/mnohosten/laura-db/pkg/document"
)

// BenchmarkExecute_Sequential benchmarks sequential execution
func BenchmarkExecute_Sequential(b *testing.B) {
	sizes := []int{1000, 5000, 10000, 50000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("docs_%d", size), func(b *testing.B) {
			// Create test documents
			docs := make([]*document.Document, size)
			for i := 0; i < size; i++ {
				doc := document.NewDocument()
				doc.Set("_id", i)
				doc.Set("age", int64(20+i%60))
				doc.Set("score", int64(50+i%50))
				docs[i] = doc
			}

			executor := NewExecutor(docs)

			q := NewQuery(map[string]interface{}{
				"age": map[string]interface{}{
					"$gte": int64(40),
				},
			})

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := executor.Execute(q)
				if err != nil {
					b.Fatalf("Execute failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkExecuteParallel benchmarks parallel execution
func BenchmarkExecuteParallel(b *testing.B) {
	sizes := []int{1000, 5000, 10000, 50000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("docs_%d", size), func(b *testing.B) {
			// Create test documents
			docs := make([]*document.Document, size)
			for i := 0; i < size; i++ {
				doc := document.NewDocument()
				doc.Set("_id", i)
				doc.Set("age", int64(20+i%60))
				doc.Set("score", int64(50+i%50))
				docs[i] = doc
			}

			executor := NewExecutor(docs)

			q := NewQuery(map[string]interface{}{
				"age": map[string]interface{}{
					"$gte": int64(40),
				},
			})

			config := &ParallelConfig{
				MinDocsForParallel: 500,
				MaxWorkers:         runtime.NumCPU(),
				ChunkSize:          0, // auto
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := executor.ExecuteParallel(q, config)
				if err != nil {
					b.Fatalf("ExecuteParallel failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkExecuteParallel_vs_Sequential compares parallel vs sequential
func BenchmarkExecuteParallel_vs_Sequential(b *testing.B) {
	// Use a large dataset where parallelization should help
	size := 20000
	docs := make([]*document.Document, size)
	for i := 0; i < size; i++ {
		doc := document.NewDocument()
		doc.Set("_id", i)
		doc.Set("value", int64(i))
		docs[i] = doc
	}

	executor := NewExecutor(docs)

	q := NewQuery(map[string]interface{}{
		"value": map[string]interface{}{
			"$gte": int64(5000),
		},
	})

	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := executor.Execute(q)
			if err != nil {
				b.Fatalf("Execute failed: %v", err)
			}
		}
	})

	b.Run("Parallel", func(b *testing.B) {
		config := &ParallelConfig{
			MinDocsForParallel: 1000,
			MaxWorkers:         runtime.NumCPU(),
			ChunkSize:          0,
		}

		for i := 0; i < b.N; i++ {
			_, err := executor.ExecuteParallel(q, config)
			if err != nil {
				b.Fatalf("ExecuteParallel failed: %v", err)
			}
		}
	})
}

// BenchmarkExecuteParallel_ComplexQuery benchmarks complex queries in parallel
func BenchmarkExecuteParallel_ComplexQuery(b *testing.B) {
	size := 10000
	docs := make([]*document.Document, size)
	for i := 0; i < size; i++ {
		doc := document.NewDocument()
		doc.Set("_id", i)
		doc.Set("age", int64(20+i%60))
		doc.Set("score", int64(50+i%50))
		doc.Set("active", i%3 == 0)
		docs[i] = doc
	}

	executor := NewExecutor(docs)

	// Complex query
	q := NewQuery(map[string]interface{}{
		"$and": []interface{}{
			map[string]interface{}{
				"age": map[string]interface{}{
					"$gte": int64(30),
				},
			},
			map[string]interface{}{
				"$or": []interface{}{
					map[string]interface{}{"score": map[string]interface{}{"$gt": int64(80)}},
					map[string]interface{}{"active": true},
				},
			},
		},
	})

	config := &ParallelConfig{
		MinDocsForParallel: 1000,
		MaxWorkers:         runtime.NumCPU(),
		ChunkSize:          0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.ExecuteParallel(q, config)
		if err != nil {
			b.Fatalf("ExecuteParallel failed: %v", err)
		}
	}
}

// BenchmarkExecuteParallel_WithSort benchmarks parallel execution with sorting
func BenchmarkExecuteParallel_WithSort(b *testing.B) {
	size := 10000
	docs := make([]*document.Document, size)
	for i := 0; i < size; i++ {
		doc := document.NewDocument()
		doc.Set("_id", i)
		doc.Set("score", int64(i))
		docs[i] = doc
	}

	executor := NewExecutor(docs)

	q := NewQuery(map[string]interface{}{
		"score": map[string]interface{}{
			"$gte": int64(2000),
		},
	})
	q.WithSort([]SortField{{Field: "score", Ascending: false}})

	config := &ParallelConfig{
		MinDocsForParallel: 1000,
		MaxWorkers:         runtime.NumCPU(),
		ChunkSize:          0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.ExecuteParallel(q, config)
		if err != nil {
			b.Fatalf("ExecuteParallel failed: %v", err)
		}
	}
}

// BenchmarkExecuteParallel_WorkerScaling benchmarks scaling with different worker counts
func BenchmarkExecuteParallel_WorkerScaling(b *testing.B) {
	size := 20000
	docs := make([]*document.Document, size)
	for i := 0; i < size; i++ {
		doc := document.NewDocument()
		doc.Set("_id", i)
		doc.Set("value", int64(i))
		docs[i] = doc
	}

	executor := NewExecutor(docs)

	q := NewQuery(map[string]interface{}{
		"value": map[string]interface{}{
			"$gte": int64(5000),
		},
	})

	workers := []int{1, 2, 4, 8, runtime.NumCPU()}

	for _, w := range workers {
		if w > runtime.NumCPU() {
			continue
		}

		b.Run(fmt.Sprintf("workers_%d", w), func(b *testing.B) {
			config := &ParallelConfig{
				MinDocsForParallel: 1000,
				MaxWorkers:         w,
				ChunkSize:          0,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := executor.ExecuteParallel(q, config)
				if err != nil {
					b.Fatalf("ExecuteParallel failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkExecuteParallel_ChunkSize benchmarks different chunk sizes
func BenchmarkExecuteParallel_ChunkSize(b *testing.B) {
	size := 10000
	docs := make([]*document.Document, size)
	for i := 0; i < size; i++ {
		doc := document.NewDocument()
		doc.Set("_id", i)
		doc.Set("value", int64(i))
		docs[i] = doc
	}

	executor := NewExecutor(docs)

	q := NewQuery(map[string]interface{}{
		"value": map[string]interface{}{
			"$gte": int64(2000),
		},
	})

	chunkSizes := []int{100, 500, 1000, 2500, 5000}

	for _, cs := range chunkSizes {
		b.Run(fmt.Sprintf("chunk_%d", cs), func(b *testing.B) {
			config := &ParallelConfig{
				MinDocsForParallel: 1000,
				MaxWorkers:         runtime.NumCPU(),
				ChunkSize:          cs,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := executor.ExecuteParallel(q, config)
				if err != nil {
					b.Fatalf("ExecuteParallel failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkExecuteWithPlanParallel benchmarks parallel execution with query plan
func BenchmarkExecuteWithPlanParallel(b *testing.B) {
	size := 10000
	docs := make([]*document.Document, size)
	for i := 0; i < size; i++ {
		doc := document.NewDocument()
		doc.Set("_id", i)
		doc.Set("age", int64(20+i%60))
		docs[i] = doc
	}

	executor := NewExecutor(docs)

	q := NewQuery(map[string]interface{}{
		"age": map[string]interface{}{
			"$gte": int64(40),
		},
	})

	plan := &QueryPlan{
		UseIndex: false,
		ScanType: ScanTypeCollection,
	}

	config := &ParallelConfig{
		MinDocsForParallel: 1000,
		MaxWorkers:         runtime.NumCPU(),
		ChunkSize:          0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.ExecuteWithPlanParallel(q, plan, config)
		if err != nil {
			b.Fatalf("ExecuteWithPlanParallel failed: %v", err)
		}
	}
}
