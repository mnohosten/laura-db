# Disk Persistence Examples

This directory contains examples demonstrating LauraDB's disk-based storage capabilities. These examples show how data persists across database restarts, how to handle large datasets, and how to optimize performance.

## Examples

### 1. Basic Persistence (`basic_persistence.go`)

**Purpose**: Demonstrates that data survives database restarts.

**What it shows**:
- Creating a database and inserting documents
- Closing the database (flushing to disk)
- Reopening the database and verifying all data is intact
- Performing queries on persisted data
- Modifying data and ensuring changes persist

**Run**:
```bash
go run basic_persistence.go
```

**Key Takeaways**:
- All data is automatically persisted to disk
- Data survives server crashes and restarts
- The Write-Ahead Log (WAL) ensures durability
- CRUD operations work seamlessly with disk storage

**Output**: Creates `./persistence_demo_data/` directory containing:
- `data.db` - Main database file with all documents
- `wal.log` - Write-ahead log for crash recovery
- `collections/` - Collection metadata

---

### 2. Large Dataset (`large_dataset.go`)

**Purpose**: Shows how LauraDB handles datasets larger than the buffer pool.

**What it shows**:
- Inserting 10,000 documents (exceeds buffer pool capacity)
- Creating indexes for efficient queries
- Query performance with caching
- Memory efficiency through automatic page eviction
- Aggregation on large datasets
- Persistence of large datasets

**Run**:
```bash
go run large_dataset.go
```

**Key Takeaways**:
- Dataset can be much larger than available memory
- Buffer pool (LRU cache) keeps hot data in memory
- Cold data is transparently loaded from disk
- Indexes are essential for query performance on large datasets
- Query cache provides significant speedup (10-100x)
- All data persists correctly regardless of size

**Configuration**: Uses a small buffer pool (500 pages = ~2MB) to demonstrate disk I/O behavior.

---

### 3. Performance Tuning (`performance_tuning.go`)

**Purpose**: Demonstrates performance optimization techniques.

**What it shows**:
- Impact of buffer pool sizing (100 vs 500 vs 1000 vs 5000 pages)
- Index usage vs full table scans (10-100x speedup)
- Batch operations vs individual operations (5-10x speedup)
- Query optimization with `Explain()`
- Query cache benefits
- Configuration recommendations

**Run**:
```bash
go run performance_tuning.go
```

**Key Takeaways**:
- Larger buffer pool = better performance but more memory
- Indexes provide 10-100x speedup for lookups
- Batch operations (InsertMany) are significantly faster
- Use `Explain()` to verify queries use indexes
- Query cache provides major speedup for repeated queries

**Performance Guidelines**:
| Dataset Size | Buffer Pool Size | Memory Usage |
|--------------|------------------|--------------|
| <10K docs    | 100-500 pages    | ~0.4-2 MB    |
| 10K-100K     | 1000-2000 pages  | ~4-8 MB      |
| 100K-1M      | 5000-10000 pages | ~20-40 MB    |
| >1M docs     | 10000+ pages     | ~40+ MB      |

---

## Architecture Overview

LauraDB's disk storage uses multiple layers of caching for optimal performance:

```
┌─────────────────────────────────────────┐
│     Application (CRUD operations)       │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────┼───────────────────────┐
│     Query Cache (LRU, 5-min TTL)        │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────┼───────────────────────┐
│   Document Cache (per-collection LRU)   │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────┼───────────────────────┐
│  Buffer Pool (page cache, configurable) │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────┼───────────────────────┐
│            Disk Files                   │
│  - data.db (slotted pages)              │
│  - wal.log (write-ahead log)            │
│  - collections/ (metadata)              │
└─────────────────────────────────────────┘
```

## Configuration

### Basic Configuration

```go
config := database.DefaultConfig("./data")
db, err := database.Open(config)
```

### Optimized Configuration

```go
config := database.DefaultConfig("./data")
config.BufferPoolSize = 5000  // 20MB buffer pool for larger datasets
db, err := database.Open(config)
```

## Best Practices

### 1. Buffer Pool Sizing
- Start with default (1000 pages = ~4MB)
- Increase if working with large datasets
- Monitor query performance and adjust

### 2. Index Strategy
- Create indexes on frequently queried fields
- Use `Explain()` to verify index usage
- Consider compound indexes for multi-field queries

### 3. Batch Operations
- Use `InsertMany()` for bulk inserts
- Use `UpdateMany()` for bulk updates
- Reduces overhead and improves throughput

### 4. Query Optimization
- Use projections to fetch only needed fields
- Add `Limit` when you don't need all results
- Leverage query cache for repeated queries

### 5. Performance Monitoring
```go
// Check if query uses index
plan := collection.Explain(filter)
if plan["indexName"] == nil {
    log.Println("Warning: Query not using index!")
}
```

## Expected Performance

Performance with disk storage (SSD):

| Operation         | Cached      | Disk (Cold) |
|-------------------|-------------|-------------|
| InsertOne         | ~100-200µs  | ~500-1000µs |
| FindOne by _id    | ~50-100µs   | ~500-1000µs |
| Find (indexed)    | ~100-200µs  | ~1-2ms      |
| Find (full scan)  | ~200-500µs  | ~2-5ms      |
| UpdateOne         | ~150-300µs  | ~1-2ms      |
| DeleteOne         | ~100-200µs  | ~1-2ms      |

**Note**: "Cached" means data is in buffer pool or document cache.

## Cleanup

After running the examples, clean up the test data:

```bash
# Clean up all test data directories
rm -rf ./persistence_demo_data
rm -rf ./large_dataset_data
rm -rf ./perf_*
```

## Additional Resources

- [API Reference](../../docs/api-reference.md) - Complete API documentation
- [Storage Engine](../../docs/storage-engine.md) - Storage engine internals
- [Performance Tuning](../../docs/performance-tuning.md) - Detailed tuning guide
- [Disk Storage Design](../../docs/disk-storage-design.md) - Architecture details

## Learn More

For more examples:
- `../basic/` - Basic document and storage operations
- `../full_demo/` - Complete database demonstration
- `../aggregation_demo/` - Aggregation pipeline examples
