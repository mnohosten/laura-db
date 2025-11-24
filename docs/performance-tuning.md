# LauraDB Performance Tuning Guide

This guide covers best practices and techniques for optimizing LauraDB performance across different workloads.

## Table of Contents

1. [Configuration Tuning](#configuration-tuning)
2. [Index Optimization](#index-optimization)
3. [Query Optimization](#query-optimization)
4. [Storage Performance](#storage-performance)
5. [Concurrency and Parallelism](#concurrency-and-parallelism)
6. [Memory Management](#memory-management)
7. [Caching Strategies](#caching-strategies)
8. [Workload-Specific Tuning](#workload-specific-tuning)
9. [Monitoring and Profiling](#monitoring-and-profiling)
10. [Performance Benchmarks](#performance-benchmarks)

---

## Configuration Tuning

### Buffer Pool Size

The buffer pool caches frequently accessed pages in memory, reducing disk I/O.

**Default:** 1000 pages (4MB)

**Recommendation:**
- **Small datasets (<100MB):** 1000-2000 pages
- **Medium datasets (100MB-1GB):** 5000-10000 pages
- **Large datasets (>1GB):** 20000+ pages
- **Rule of thumb:** Allocate 10-20% of available RAM

```go
config := database.DefaultConfig("./data")
config.BufferPoolSize = 10000  // 40MB cache

db, err := database.Open(config)
```

**Impact:**
- More pages in cache = fewer disk reads
- Diminishing returns after working set fits in memory
- Monitor hit rate with `db.Stats()`

---

### HTTP Server Configuration

Tune server timeouts and limits for your workload:

```go
config := server.DefaultConfig()

// High-throughput API server
config.ReadTimeout = 10 * time.Second   // Quick reads
config.WriteTimeout = 10 * time.Second  // Quick writes
config.MaxRequestSize = 5 * 1024 * 1024 // 5MB limit

// Long-running analytics queries
config.ReadTimeout = 60 * time.Second
config.WriteTimeout = 120 * time.Second
config.MaxRequestSize = 50 * 1024 * 1024 // 50MB for bulk ops

srv, err := server.New(config)
```

---

## Index Optimization

### Index Selection

Choose the right index type for your access patterns:

| Access Pattern | Recommended Index | Reason |
|---------------|-------------------|---------|
| Exact matches | B+ tree | O(log n) lookups |
| Range queries | B+ tree | Efficient scans |
| Text search | Text index | BM25 relevance scoring |
| Geospatial queries | 2d/2dsphere | Spatial indexing |
| Multi-field queries | Compound index | Single index scan |
| Subset of data | Partial index | Smaller index size |
| Time-based expiration | TTL index | Automatic cleanup |

---

### Compound Indexes

**Problem:** Multiple indexes = multiple scans + merge

**Solution:** Compound indexes combine multiple fields

```go
// Bad: Separate indexes
users.CreateIndex("city", false)
users.CreateIndex("age", false)

// Query uses index intersection (slower)
users.Find(map[string]interface{}{
    "city": "NYC",
    "age": map[string]interface{}{"$gte": int64(25)},
})

// Good: Compound index
users.CreateCompoundIndex([]string{"city", "age"}, false)

// Query uses single index scan (faster)
```

**Field Order Matters:**

```go
// Index: ["city", "age"]
// ✅ Uses index: {"city": "NYC"}
// ✅ Uses index: {"city": "NYC", "age": 30}
// ❌ Doesn't use index: {"age": 30}  // Missing prefix

// Guideline: Order by selectivity (high to low)
// 1. Equality matches first
// 2. Range queries last
```

---

### Partial Indexes

**Problem:** Indexing inactive/deleted data wastes space

**Solution:** Index only relevant documents

```go
// Bad: Index all users (including deleted)
users.CreateIndex("email", true)  // 100k entries

// Good: Index only active users
users.CreatePartialIndex(
    "email",
    map[string]interface{}{"status": "active"},
    true,
)  // 80k entries (20% savings)
```

**Benefits:**
- Smaller index = faster scans
- Less memory usage
- Faster index maintenance

**Best for:**
- Large collections with logical subsets
- Queries with common filters
- Space-constrained environments

---

### Background Index Building

**Problem:** Index creation blocks writes

**Solution:** Build indexes in background

```go
// Bad: Blocks all operations (for large collections)
users.CreateIndex("username", true)

// Good: Non-blocking (for large collections)
users.CreateIndexWithBackground("username", true, true)

// Monitor progress
progress, _ := users.GetIndexBuildProgress("username_1")
fmt.Printf("Progress: %.1f%%\n", progress["percentComplete"])
```

**Trade-off:**
- Background: Slower build, no blocking
- Foreground: Faster build, blocks writes

**Recommendation:** Use background for collections >10,000 documents

---

### Index Statistics

**Keep statistics fresh for optimal query planning:**

```go
// After bulk operations
users.InsertMany(largeDataset)
users.Analyze()  // Recalculate index statistics

// Periodically
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        for _, coll := range db.ListCollections() {
            db.Collection(coll).Analyze()
        }
    }
}()
```

**Impact:** Query optimizer uses statistics to choose best index

---

## Query Optimization

### Use Indexes Effectively

**Check query plans:**

```go
plan := users.Explain(filter)
fmt.Printf("Index: %v\n", plan["indexName"])
fmt.Printf("Scan type: %v\n", plan["scanType"])
fmt.Printf("Estimated cost: %v\n", plan["estimatedCost"])

// If indexName is nil, query does full collection scan
if plan["indexName"] == nil {
    log.Println("WARNING: No index used, consider creating one")
}
```

**Common Issues:**

```go
// ❌ No index on 'age' field
filter := map[string]interface{}{
    "age": map[string]interface{}{"$gte": int64(25)},
}

// ✅ Create index
users.CreateIndex("age", false)

// ❌ Index exists but query doesn't use it
filter := map[string]interface{}{
    "$or": []interface{}{
        map[string]interface{}{"age": int64(25)},
        map[string]interface{}{"status": "active"},
    },
}
// Fix: Create indexes on both fields or use $and if possible

// ✅ Index-friendly query
filter := map[string]interface{}{
    "age": int64(25),
    "status": "active",
}
```

---

### Covered Queries

**Problem:** Index scan + document fetch = 2 operations

**Solution:** Query entirely from index (2.2x faster)

```go
// Create compound index
users.CreateCompoundIndex([]string{"city", "age", "status"}, false)

// Query with projection covering indexed fields
options := &database.QueryOptions{
    Projection: map[string]bool{
        "city": true,
        "age": true,
        "status": true,
    },
}

// ✅ Covered query: No document fetch needed
docs, _ := users.FindWithOptions(
    map[string]interface{}{"city": "NYC"},
    options,
)

// ❌ Not covered: Fetches documents for 'email' field
options.Projection["email"] = true
```

**Benefits:**
- 2.2x faster query execution
- Lower memory usage
- Reduced I/O

**Best for:**
- Dashboards showing summary data
- API endpoints returning limited fields
- High-throughput queries

---

### Query Cache

**Automatic caching of frequently executed queries:**

**Configuration:**
- LRU eviction (1000 entries per collection)
- 5-minute TTL
- Thread-safe
- Invalidated on writes

**Performance:** 96x faster for cached queries (328µs → 3.4µs)

**Best practices:**

```go
// ✅ Cache-friendly: Identical filters
for i := 0; i < 100; i++ {
    users.Find(map[string]interface{}{"status": "active"})
}  // Cached after first execution

// ❌ Cache-unfriendly: Different filters
for i := 0; i < 100; i++ {
    users.Find(map[string]interface{}{"age": int64(i)})
}  // Each query misses cache

// Cache is automatically invalidated on writes
users.InsertOne(newUser)  // Clears entire cache for collection
```

**When to disable cache:**
- Rarely repeated queries
- Extremely tight memory constraints
- Real-time data requirements

---

### Parallel Query Execution

**Automatic parallelization for large result sets:**

**Configuration:**
- Activates for 1000+ documents (configurable)
- Uses multiple CPU cores
- Configurable worker count

**Performance:** Up to 4.36x speedup on large datasets

```go
// Automatic parallelization (no code changes needed)
// For collections with 50k+ documents, queries automatically parallelize

// Performance characteristics:
// - 1k docs: 1.0x (sequential faster due to overhead)
// - 10k docs: 2.1x speedup
// - 50k docs: 4.36x speedup
```

**Manual configuration (if needed):**

```go
// Default thresholds work well for most cases
// Only tune if profiling shows benefit

// See docs/parallel-query-execution.md for advanced tuning
```

**Best for:**
- Large collections (50k+ documents)
- Complex filters with many matches
- Multi-core servers

---

### Projection Optimization

**Only retrieve fields you need:**

```go
// ❌ Bad: Retrieves entire document
docs, _ := users.Find(filter)
for _, doc := range docs {
    name := doc.Get("name")
    // Only using 'name' field
}

// ✅ Good: Retrieves only needed fields
options := &database.QueryOptions{
    Projection: map[string]bool{
        "name": true,
        "email": true,
    },
}
docs, _ := users.FindWithOptions(filter, options)

// Benefits:
// - Lower memory usage
// - Faster network transfer (HTTP mode)
// - Better cache utilization
```

---

### Pagination

**Use limit + skip for pagination:**

```go
// Page 1 (items 0-19)
options := &database.QueryOptions{
    Limit: 20,
    Skip: 0,
}

// Page 2 (items 20-39)
options.Skip = 20

// Page 3 (items 40-59)
options.Skip = 40
```

**Performance considerations:**
- `Skip` requires scanning skipped documents
- For large skip values, consider cursor-based pagination (future feature)
- Current performance: ~1µs per skipped document

---

### Aggregation Pipeline Optimization

**1. Put $match first:**

```go
// ❌ Bad: Processes all documents
pipeline := []map[string]interface{}{
    {"$group": map[string]interface{}{ /* ... */ }},
    {"$match": map[string]interface{}{"total": map[string]interface{}{"$gt": int64(100)}}},
}

// ✅ Good: Filters early
pipeline := []map[string]interface{}{
    {"$match": map[string]interface{}{"status": "completed"}},  // Reduce dataset
    {"$group": map[string]interface{}{ /* ... */ }},
}
```

**2. Use index-friendly $match:**

```go
// ✅ Can use index
{"$match": map[string]interface{}{"userId": "abc123"}}

// ❌ Cannot use index
{"$match": map[string]interface{}{
    "$expr": map[string]interface{}{ /* complex expression */ },
}}
```

**3. Limit early:**

```go
pipeline := []map[string]interface{}{
    {"$match": map[string]interface{}{"status": "active"}},
    {"$sort": map[string]interface{}{"createdAt": int64(-1)}},
    {"$limit": int64(10)},  // ✅ Reduces data for subsequent stages
    {"$group": map[string]interface{}{ /* ... */ }},
}
```

---

## Storage Performance

### Memory-Mapped Files

**For read-heavy workloads, use mmap storage:**

**Performance:**
- 1.44x faster reads (748ns vs 1078ns per page)
- 1.61x faster mixed workloads (70% read / 30% write)
- Up to 5.36x faster writes (platform-specific)

```go
import "github.com/mnohosten/laura-db/pkg/storage"

// Standard disk manager
dm := storage.NewDiskManager(dataFile)

// Memory-mapped disk manager
mmapDM := storage.NewMmapDiskManager(dataFile)

// Configure hints
mmapDM.MadviseRandom()      // Random access pattern
mmapDM.MadviseSequential()  // Sequential access pattern
mmapDM.MadviseWillNeed()    // Preload into memory
```

**Trade-offs:**
- ✅ Faster reads
- ✅ Kernel manages page cache
- ❌ Higher memory usage
- ❌ Platform-specific (Unix/Linux/macOS only)

**Best for:**
- Read-heavy workloads (80%+ reads)
- Large datasets that don't fit in buffer pool
- Systems with available RAM

**See:** `docs/mmap-storage.md`

---

### LSM Tree Storage

**For write-heavy workloads, use LSM tree:**

**Performance:**
- High write throughput (sequential I/O)
- Excellent for time-series data
- 2-3x write amplification

```go
import "github.com/mnohosten/laura-db/pkg/lsm"

// Create LSM tree
tree := lsm.NewLSMTree("/path/to/data")

// Write-optimized operations
tree.Put("key", "value")  // Fast: writes to memtable
tree.Get("key")          // Read: checks memtable + SSTables
```

**Architecture:**
- MemTable (in-memory, skip list)
- SSTables (immutable on-disk files)
- Bloom filters (fast non-existent key detection)
- Background compaction

**Best for:**
- Time-series data
- Logging systems
- Metrics collection
- High write throughput requirements

**See:** `docs/lsm-tree.md`

---

### Compression

**Reduce storage size and I/O:**

**Algorithms:**
- **Snappy:** Fast, moderate compression (20-40% space savings)
- **Zstd:** Balanced, good compression (30-60% savings)
- **Gzip:** Slower, high compression (50-70% savings)
- **Zlib:** Similar to Gzip

```go
import "github.com/mnohosten/laura-db/pkg/compression"

// Document compression
compressor := compression.NewSnappyCompressor()
compressed := compressor.Compress(documentBytes)
decompressed := compressor.Decompress(compressed)

// Page compression (for storage engine)
compressor := compression.NewZstdCompressor(3)  // Level 3
compressed := compressor.Compress(pageData)
```

**Performance vs. Compression:**

| Algorithm | Speed | Ratio | Best For |
|-----------|-------|-------|----------|
| Snappy | Fastest | 20-40% | Hot data, low latency |
| Zstd (level 3) | Fast | 30-60% | General purpose |
| Gzip | Slow | 50-70% | Cold data, archival |

**Recommendation:**
- **OLTP workloads:** Snappy (low latency)
- **General purpose:** Zstd level 3
- **Archival/cold storage:** Gzip or Zlib

**See:** `docs/compression.md`

---

### Defragmentation

**Reclaim space from deleted/updated documents:**

```go
import "github.com/mnohosten/laura-db/pkg/repair"

// Collection-level defragmentation
validator := repair.NewValidator(db)
repairer := repair.NewRepairer(db, validator)
defragmenter := repair.NewDefragmenter(db, repairer)

report, err := defragmenter.DefragmentCollection("users")
fmt.Printf("Pages compacted: %d\n", report.PagesCompacted)
fmt.Printf("Space saved: %d bytes\n", report.SpaceSaved)
fmt.Printf("Fragmentation ratio: %.2f%%\n", report.FragmentationRatio*100)

// Database-level defragmentation
report, err := defragmenter.DefragmentDatabase()
```

**When to defragment:**
- After bulk deletes
- After many updates
- When fragmentation ratio > 20%
- During maintenance windows

**See:** `docs/repair-tools.md`

---

## Concurrency and Parallelism

### MVCC Transactions

**LauraDB uses MVCC for non-blocking reads:**

**Benefits:**
- Readers never block writers
- Writers never block readers
- Snapshot isolation

**Best practices:**

```go
// ✅ Good: Short transactions
err := db.WithTransaction(func(s *database.Session) error {
    // Quick operations
    s.InsertOne("logs", logEntry)
    s.UpdateOne("counters", filter, update)
    return nil
})

// ❌ Bad: Long-running transactions
err := db.WithTransaction(func(s *database.Session) error {
    // Don't do this: blocks garbage collection
    time.Sleep(10 * time.Minute)
    return nil
})

// ✅ Good: Retry on conflicts
for retries := 0; retries < 3; retries++ {
    err := db.WithTransaction(func(s *database.Session) error {
        // Transaction logic
        return nil
    })
    if err != mvcc.ErrConflict {
        break
    }
    time.Sleep(time.Millisecond * 10 * time.Duration(retries+1))  // Exponential backoff
}
```

**Transaction overhead:** ~1.3µs per transaction

---

### Connection Pooling

**Session pooling for reuse:**

**Performance:** 1.27x faster than allocating new sessions (106ns vs 135ns)

```go
// Automatic session pooling (built-in)
// No configuration needed - uses sync.Pool internally

// Sessions are automatically recycled
session := db.StartSession()
defer session.AbortTransaction()  // Returns to pool

// Session state is reset on return to pool
```

**Worker Pool:**

**Performance:** 1.62x faster than raw goroutines (83ns vs 134ns)

```go
import "github.com/mnohosten/laura-db/pkg/database"

// Create worker pool
pool := database.NewWorkerPool(runtime.NumCPU())

// Submit tasks
pool.Submit(func() {
    // Background work
})

// Shutdown
pool.Shutdown()
pool.Wait()
```

**Benefits:**
- Controlled concurrency
- Zero allocation task submission
- Graceful shutdown

**See:** `docs/connection-pooling.md`

---

### Read-Write Lock Optimization

**Buffer pool uses optimized locking:**

**Pattern:**
1. Acquire read lock
2. Check condition
3. If modification needed:
   - Release read lock
   - Acquire write lock
   - Double-check condition (race protection)
   - Modify
   - Release write lock

**Performance:** 3-5x improvement in concurrent read throughput

**Implementation:** Automatic in buffer pool, no configuration needed

**See:** `docs/rwlock-optimization.md`

---

### Lock-Free Data Structures

**Available lock-free components:**

```go
import "github.com/mnohosten/laura-db/pkg/concurrent"

// Lock-free counter (1.6ns/op sequential, 48.7ns/op parallel)
counter := concurrent.NewCounter()
counter.Inc()
counter.Add(10)
val := counter.Get()

// Lock-free stack (30.9ns push, 6.1ns pop)
stack := concurrent.NewStack()
stack.Push(item)
item, ok := stack.Pop()

// Sharded LRU cache (3.5x faster with 32 shards)
cache := concurrent.NewShardedLRU(1000, 32)  // capacity, shards
cache.Put("key", "value")
value, ok := cache.Get("key")
```

**Use cases:**
- High-contention counters (statistics, metrics)
- Query result cache (sharded LRU)
- Temporary data structures

**See:** `docs/lock-free-data-structures.md`

---

## Memory Management

### Buffer Pool Tuning

**Monitor hit rate:**

```go
stats := db.Stats()
storageStats := stats["storage"].(map[string]interface{})

hitRate := storageStats["hitRate"].(float64)
fmt.Printf("Buffer pool hit rate: %.2f%%\n", hitRate*100)

// Target: 90%+ hit rate
// If <90%, increase buffer pool size
```

---

### Query Cache Management

**Automatic LRU eviction:**

**Configuration:**
- Per-collection cache (1000 entries)
- 5-minute TTL
- Auto-invalidation on writes

**Memory usage:** ~1KB per cached query

**Total memory:** ~1MB per collection (at max capacity)

**To reduce memory:**
- Smaller collections naturally use less cache
- Cache is cleared on any write operation

---

### Collection Size Monitoring

```go
stats := collection.Stats()
fmt.Printf("Documents: %v\n", stats["count"])
fmt.Printf("Estimated size: %v bytes\n", stats["estimatedSize"])

// Monitor growth
if stats["count"].(int) > 1000000 {
    log.Println("Consider archiving old data")
}
```

---

## Caching Strategies

### Application-Level Caching

**Complement database features with application cache:**

```go
// In-memory cache for hot data
var userCache sync.Map

func GetUser(id string) (*User, error) {
    // Check cache
    if cached, ok := userCache.Load(id); ok {
        return cached.(*User), nil
    }

    // Query database
    doc, err := users.FindOne(map[string]interface{}{"_id": id})
    if err != nil {
        return nil, err
    }

    user := docToUser(doc)
    userCache.Store(id, user)
    return user, nil
}

func UpdateUser(id string, updates map[string]interface{}) error {
    err := users.UpdateOne(map[string]interface{}{"_id": id}, updates)
    if err == nil {
        userCache.Delete(id)  // Invalidate cache
    }
    return err
}
```

---

### TTL Indexes for Automatic Cleanup

**Automatic expiration of old data:**

```go
// Sessions expire after 1 hour
sessions.CreateTTLIndex("createdAt", 3600)

// Insert with timestamp
sessions.InsertOne(map[string]interface{}{
    "userId": "user123",
    "token": "...",
    "createdAt": time.Now(),
})

// Automatic cleanup every 60 seconds
// No manual cleanup needed
```

**Benefits:**
- Automatic memory reclamation
- No manual cleanup logic
- Minimal overhead (~7% on inserts)

---

## Workload-Specific Tuning

### OLTP (Online Transaction Processing)

**Characteristics:** Many small, fast transactions

**Optimizations:**
1. **Smaller buffer pool** (1000-5000 pages)
2. **Indexes on all query fields**
3. **Short transactions** (<100ms)
4. **Connection pooling** for concurrent clients
5. **Snappy compression** for low latency

```go
config := database.DefaultConfig("./data")
config.BufferPoolSize = 2000

// Create indexes on common queries
users.CreateIndex("email", true)
orders.CreateCompoundIndex([]string{"userId", "status"}, false)

// Use transactions for consistency
err := db.WithTransaction(func(s *database.Session) error {
    s.InsertOne("orders", order)
    s.UpdateOne("inventory", filter, update)
    return nil
})
```

---

### OLAP (Online Analytical Processing)

**Characteristics:** Complex queries over large datasets

**Optimizations:**
1. **Large buffer pool** (20000+ pages)
2. **Compound indexes** for multi-field queries
3. **Parallel query execution** (automatic for 1000+ docs)
4. **Aggregation pipelines** for summarization
5. **Covered queries** where possible
6. **Zstd compression** for space savings

```go
config := database.DefaultConfig("./data")
config.BufferPoolSize = 50000  // 200MB cache

// Compound indexes for analytical queries
events.CreateCompoundIndex([]string{"eventType", "timestamp"}, false)
events.CreateCompoundIndex([]string{"userId", "eventType", "timestamp"}, false)

// Use aggregation for summaries
pipeline := []map[string]interface{}{
    {"$match": map[string]interface{}{
        "timestamp": map[string]interface{}{
            "$gte": startDate,
            "$lt": endDate,
        },
    }},
    {"$group": map[string]interface{}{
        "_id": "$eventType",
        "count": map[string]interface{}{"$sum": int64(1)},
    }},
}
results, _ := events.Aggregate(pipeline)
```

---

### Time-Series Data

**Characteristics:** High write rate, time-based queries

**Optimizations:**
1. **LSM tree storage** for high write throughput
2. **TTL indexes** for automatic expiration
3. **Compound index** on timestamp + other fields
4. **Compression** (Zstd or Gzip for older data)
5. **Periodic defragmentation**

```go
// Use LSM tree (see docs/lsm-tree.md)

// TTL for old data
metrics.CreateTTLIndex("timestamp", 86400*30)  // 30 days

// Compound index for queries
metrics.CreateCompoundIndex([]string{"timestamp", "metricName"}, false)

// Regular defragmentation
defragmenter.DefragmentCollection("metrics")
```

---

### Read-Heavy Workloads

**Optimizations:**
1. **Memory-mapped files**
2. **Large buffer pool**
3. **Query cache** (automatic)
4. **Covered queries**
5. **Read replicas** (when implemented)

```go
// Use mmap storage
mmapDM := storage.NewMmapDiskManager(dataFile)

// Large buffer pool
config.BufferPoolSize = 100000  // 400MB

// Covered queries
users.CreateCompoundIndex([]string{"status", "email", "name"}, false)
options := &database.QueryOptions{
    Projection: map[string]bool{"status": true, "email": true, "name": true},
}
```

---

### Write-Heavy Workloads

**Optimizations:**
1. **LSM tree storage**
2. **Smaller buffer pool** (less cache to invalidate)
3. **Background index building**
4. **Partial indexes** (reduce index maintenance)
5. **Batch writes** with InsertMany

```go
// LSM tree for write optimization

// Batch inserts
docs := make([]map[string]interface{}, 1000)
for i := 0; i < 1000; i++ {
    docs[i] = generateDocument()
}
logs.InsertMany(docs)  // Single operation

// Background indexes
logs.CreateIndexWithBackground("timestamp", false, true)
```

---

## Monitoring and Profiling

### Database Statistics

```go
stats := db.Stats()
fmt.Printf("Collections: %v\n", stats["collections"])
fmt.Printf("Active transactions: %v\n", stats["transactions"]["active"])
fmt.Printf("Buffer pool size: %v\n", stats["storage"]["bufferPoolSize"])
```

---

### Collection Statistics

```go
stats := collection.Stats()
fmt.Printf("Document count: %v\n", stats["count"])
fmt.Printf("Index count: %v\n", len(stats["indexes"].([]map[string]interface{})))
```

---

### Index Statistics

```go
indexes := collection.ListIndexes()
for _, idx := range indexes {
    name := idx["name"].(string)
    stats := idx["stats"].(map[string]interface{})

    fmt.Printf("Index: %s\n", name)
    fmt.Printf("  Cardinality: %v\n", stats["cardinality"])
    fmt.Printf("  Selectivity: %.4f\n", stats["selectivity"])
    fmt.Printf("  Min: %v\n", stats["min"])
    fmt.Printf("  Max: %v\n", stats["max"])
}
```

---

### Query Profiling

```go
// Explain query plan
plan := collection.Explain(filter)
fmt.Printf("Index: %v\n", plan["indexName"])
fmt.Printf("Scan type: %v\n", plan["scanType"])
fmt.Printf("Estimated cost: %v\n", plan["estimatedCost"])
fmt.Printf("Covered: %v\n", plan["isCovered"])

// Time queries
start := time.Now()
docs, _ := collection.Find(filter)
elapsed := time.Since(start)
fmt.Printf("Query time: %v\n", elapsed)
```

---

### Benchmarking

**Use built-in benchmarks:**

```bash
# Run all benchmarks
make bench

# Specific benchmarks
go test -bench=BenchmarkInsert -benchmem ./pkg/database
go test -bench=BenchmarkFind -benchmem ./pkg/database
go test -bench=BenchmarkIndex -benchmem ./pkg/index

# With CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./pkg/database
go tool pprof cpu.prof

# With memory profiling
go test -bench=. -memprofile=mem.prof ./pkg/database
go tool pprof mem.prof
```

**See:** `docs/benchmarking.md`

---

## Performance Benchmarks

### Insert Performance

| Operation | Documents | Time | Throughput |
|-----------|-----------|------|------------|
| InsertOne | 1 | ~50µs | 20,000/sec |
| InsertMany | 100 | ~3ms | 33,000/sec |
| InsertMany | 1000 | ~25ms | 40,000/sec |

---

### Query Performance

| Operation | Result Size | Index | Time |
|-----------|-------------|-------|------|
| FindOne | 1 | Yes | ~10µs |
| FindOne | 1 | No | ~500µs |
| Find | 100 | Yes | ~100µs |
| Find | 100 | No | ~5ms |
| Find (covered) | 100 | Yes | ~50µs |
| Find (cached) | 100 | Yes | ~3.4µs |

---

### Index Performance

| Operation | Index Type | Time |
|-----------|------------|------|
| B+ tree insert | Single field | ~1.5µs |
| B+ tree search | Single field | ~800ns |
| Compound insert | 3 fields | ~2µs |
| Text search | Multi-field | ~100µs |
| Geo search (2dsphere) | Near query | ~150µs |

---

### Transaction Performance

| Operation | Time |
|-----------|------|
| Begin transaction | ~500ns |
| Commit (1 write) | ~2µs |
| Commit (10 writes) | ~15µs |
| Abort | ~1µs |

---

### Storage Performance

| Operation | Standard | Mmap | LSM |
|-----------|----------|------|-----|
| Page read | 1078ns | 748ns | N/A |
| Page write | 2500ns | 1550ns | N/A |
| Random insert | 50µs | 45µs | 10µs |
| Sequential write | 35µs | 30µs | 5µs |

---

## Quick Reference: Performance Checklist

### Before Going to Production

- [ ] Buffer pool sized appropriately (10-20% of RAM)
- [ ] Indexes created on all queried fields
- [ ] Compound indexes for multi-field queries
- [ ] Partial indexes for filtered queries
- [ ] Query plans verified with Explain()
- [ ] Covered queries where possible
- [ ] Projections used to limit data transfer
- [ ] Appropriate storage engine (standard/mmap/LSM)
- [ ] Compression enabled if storage-constrained
- [ ] TTL indexes for expiring data
- [ ] Transactions kept short (<100ms)
- [ ] Connection pooling configured
- [ ] Statistics updated regularly (Analyze())
- [ ] Monitoring in place (stats, profiling)

### Common Performance Issues

| Symptom | Likely Cause | Solution |
|---------|-------------|----------|
| Slow queries | No index | Create index on queried fields |
| High memory | Large buffer pool | Reduce buffer pool size |
| Slow writes | Too many indexes | Use partial indexes |
| Cache misses | Unique queries | Optimize for reuse |
| Lock contention | Long transactions | Keep transactions short |
| High CPU | Full scans | Create appropriate indexes |
| Disk I/O | Small buffer pool | Increase buffer pool or use mmap |

---

## Conclusion

Performance tuning is an iterative process:

1. **Measure** current performance (benchmarks, profiling)
2. **Identify** bottlenecks (query plans, statistics)
3. **Optimize** based on workload (indexes, configuration, storage)
4. **Verify** improvements (re-benchmark)
5. **Monitor** in production (statistics, logging)

For workload-specific guidance, refer to the relevant sections above.

---

## Additional Resources

- [API Reference](api-reference.md)
- [Index Documentation](indexing.md)
- [Query Optimization](statistics-optimization.md)
- [Storage Engine](storage-engine.md)
- [MVCC Transactions](mvcc.md)
- [Benchmarking Guide](benchmarking.md)

---

**Version:** LauraDB v0.1.0
**Last Updated:** 2024-01-15
