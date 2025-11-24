# Parallel Query Execution

LauraDB supports parallel query execution to improve performance on large datasets by utilizing multiple CPU cores.

## Overview

Parallel query execution divides the document filtering workload across multiple worker goroutines, allowing the database to process queries significantly faster on multi-core systems. This feature is particularly beneficial for:

- Large collections (1000+ documents)
- Complex filter conditions
- CPU-bound query operations
- Systems with multiple cores

## Performance

Benchmark results on Apple M4 Max (14 cores):

| Dataset Size | Sequential | Parallel | Speedup |
|--------------|-----------|----------|---------|
| 1,000 docs   | 64.4µs    | 51.9µs   | 1.24x   |
| 5,000 docs   | 328µs     | 169µs    | 1.94x   |
| 10,000 docs  | 655µs     | 217µs    | 3.02x   |
| 20,000 docs  | 1337µs    | 376µs    | **3.56x** |
| 50,000 docs  | 3358µs    | 770µs    | **4.36x** |

**Key Findings:**
- Speedup increases with dataset size
- Up to **4.36x faster** on large datasets (50k+ documents)
- Automatic threshold-based activation (default: 1000 documents)
- Minimal overhead for small datasets

## Usage

### Basic Parallel Execution

```go
// Create query
q := query.NewQuery(map[string]interface{}{
    "age": map[string]interface{}{
        "$gte": int64(30),
    },
})

// Execute in parallel with default configuration
executor := query.NewExecutor(documents)
results, err := executor.ExecuteParallel(q, nil)
```

### Custom Configuration

```go
// Configure parallel execution
config := &query.ParallelConfig{
    MinDocsForParallel: 500,          // Use parallel for 500+ documents
    MaxWorkers:         8,             // Use 8 worker goroutines
    ChunkSize:          1000,          // Process 1000 documents per chunk
}

results, err := executor.ExecuteParallel(q, config)
```

### With Query Plan

Parallel execution works seamlessly with the query optimizer:

```go
// Create planner
planner := query.NewQueryPlanner(indexes)
plan := planner.Plan(q)

// Execute with plan in parallel
results, err := executor.ExecuteWithPlanParallel(q, plan, config)
```

## Configuration

### ParallelConfig

```go
type ParallelConfig struct {
    // MinDocsForParallel: minimum number of documents to use parallel execution
    // Default: 1000
    // Recommendation: Set based on your query complexity and hardware
    MinDocsForParallel int

    // MaxWorkers: maximum number of parallel workers
    // Default: 0 (uses runtime.NumCPU())
    // Recommendation: Leave at 0 for automatic CPU detection
    MaxWorkers int

    // ChunkSize: number of documents per worker chunk
    // Default: 0 (auto-calculated: totalDocs / workers)
    // Recommendation: Leave at 0 unless you have specific performance needs
    ChunkSize int
}
```

### Default Configuration

```go
config := query.DefaultParallelConfig()
// MinDocsForParallel: 1000
// MaxWorkers: 0 (uses all CPUs)
// ChunkSize: 0 (auto-calculated)
```

## How It Works

### Execution Flow

1. **Threshold Check**: If document count < `MinDocsForParallel`, falls back to sequential execution
2. **Worker Allocation**: Determines worker count (uses `MaxWorkers` or `runtime.NumCPU()`)
3. **Chunking**: Divides documents into chunks (auto-calculated or uses `ChunkSize`)
4. **Parallel Filtering**: Each worker processes its chunks concurrently
5. **Result Collection**: Results are merged from all workers
6. **Post-Processing**: Applies sort, skip, limit, and projection sequentially

### Architecture

```
┌─────────────────────────────────────────────────────┐
│                 Query Executor                      │
└─────────────────────────────────────────────────────┘
                       │
                       ├──> Threshold Check
                       │    (docs < MinDocsForParallel?)
                       │
        ┌──────────────┴──────────────┐
        │                             │
    Sequential                    Parallel
     Execution                    Execution
                                      │
                    ┌─────────────────┼─────────────────┐
                    │                 │                 │
               Worker 1           Worker 2         Worker N
            (chunk 0-999)      (chunk 1000-1999)  (chunk ...)
                    │                 │                 │
                    └─────────────────┼─────────────────┘
                                      │
                              Merge Results
                                      │
                          Sort → Skip → Limit → Project
```

### Worker Distribution

Workers process documents in chunks to minimize synchronization overhead:

```go
// Example with 10,000 documents and 4 workers
// Auto-calculated chunk size: 10000 / 4 = 2500

Worker 1: docs[0:2500]
Worker 2: docs[2500:5000]
Worker 3: docs[5000:7500]
Worker 4: docs[7500:10000]
```

## Best Practices

### When to Use Parallel Execution

✅ **Use parallel execution when:**
- Working with large collections (1000+ documents)
- Running complex filter conditions
- System has multiple CPU cores
- Query performance is critical

❌ **Don't use parallel execution when:**
- Dataset is small (< 1000 documents)
- Single-core system
- Index can satisfy query (covered queries)
- Memory is constrained

### Configuration Tuning

1. **MinDocsForParallel**
   - Start with default (1000)
   - Increase if queries are simple (2000-5000)
   - Decrease if queries are complex (500-1000)

2. **MaxWorkers**
   - Leave at 0 (auto-detect) for most cases
   - Reduce if you need to limit CPU usage
   - Increase only if you have many idle cores

3. **ChunkSize**
   - Leave at 0 (auto-calculate) for most cases
   - Decrease (100-500) for very complex queries
   - Increase (5000-10000) for simple queries

### Integration with Collection

While the Collection API doesn't expose parallel execution directly, you can use it by:

```go
// Get documents from collection
docs := collection.GetAllDocuments()

// Create executor
executor := query.NewExecutor(docs)

// Execute in parallel
results, err := executor.ExecuteParallel(query, config)
```

## Limitations

1. **Memory Usage**: Parallel execution uses more memory due to goroutine overhead
2. **Sort/Limit/Skip**: Applied sequentially after parallel filtering
3. **Covered Queries**: Falls back to sequential for covered queries (already optimized)
4. **Small Datasets**: No benefit for datasets below threshold

## Future Enhancements

Potential improvements for parallel execution:

- [ ] Parallel sorting for large result sets
- [ ] Adaptive worker count based on query complexity
- [ ] Per-collection parallel execution settings
- [ ] Integration with Collection Find API
- [ ] Parallel aggregation pipeline stages
- [ ] NUMA-aware worker allocation
- [ ] GPU-accelerated filtering (experimental)

## Benchmarks

Run benchmarks to measure performance on your hardware:

```bash
# Run all parallel benchmarks
go test ./pkg/query -bench=Parallel -benchmem

# Compare sequential vs parallel
go test ./pkg/query -bench=BenchmarkExecuteParallel_vs_Sequential -benchmem

# Test worker scaling
go test ./pkg/query -bench=BenchmarkExecuteParallel_WorkerScaling -benchmem

# Test chunk size impact
go test ./pkg/query -bench=BenchmarkExecuteParallel_ChunkSize -benchmem
```

## Examples

### Example 1: Large Dataset Query

```go
package main

import (
    "fmt"
    "github.com/mnohosten/laura-db/pkg/document"
    "github.com/mnohosten/laura-db/pkg/query"
)

func main() {
    // Create large dataset
    docs := make([]*document.Document, 50000)
    for i := 0; i < 50000; i++ {
        doc := document.NewDocument()
        doc.Set("_id", i)
        doc.Set("age", int64(20 + i%60))
        doc.Set("score", int64(50 + i%100))
        docs[i] = doc
    }

    // Create executor
    executor := query.NewExecutor(docs)

    // Complex query
    q := query.NewQuery(map[string]interface{}{
        "$and": []interface{}{
            map[string]interface{}{
                "age": map[string]interface{}{"$gte": int64(40)},
            },
            map[string]interface{}{
                "score": map[string]interface{}{"$gt": int64(75)},
            },
        },
    })

    // Execute in parallel
    results, err := executor.ExecuteParallel(q, nil)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Found %d matching documents\n", len(results))
}
```

### Example 2: Custom Worker Configuration

```go
// Configure for CPU-intensive queries with limited cores
config := &query.ParallelConfig{
    MinDocsForParallel: 500,
    MaxWorkers:         4,  // Limit to 4 cores
    ChunkSize:          250, // Smaller chunks for better load balancing
}

results, err := executor.ExecuteParallel(query, config)
```

### Example 3: Adaptive Execution

```go
// Use parallel only for large datasets
var results []*document.Document
var err error

if len(documents) >= 5000 {
    // Use parallel for large datasets
    config := &query.ParallelConfig{
        MinDocsForParallel: 1000,
        MaxWorkers:         0,
        ChunkSize:          0,
    }
    results, err = executor.ExecuteParallel(q, config)
} else {
    // Use sequential for small datasets
    results, err = executor.Execute(q)
}
```

## See Also

- [Query Engine Documentation](query-engine.md)
- [Statistics-Based Optimization](statistics-optimization.md)
- [Covered Queries](indexing.md#covered-queries)
- [Benchmarking Guide](../BENCHMARKS.md)
