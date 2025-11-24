# Lock-Free Data Structures

LauraDB includes a collection of high-performance concurrent data structures that minimize lock contention and improve scalability under concurrent workloads.

## Overview

Lock-free data structures use atomic operations (CAS - Compare-And-Swap) instead of traditional mutex locks to achieve thread-safe concurrent access. This approach provides:

- **Better scalability**: No lock contention as threads don't block each other
- **Lower latency**: No waiting for lock acquisition
- **Progress guarantees**: At least one thread makes progress on each operation
- **Cache-friendly**: Reduced false sharing and better CPU cache utilization

## Implemented Data Structures

### 1. Lock-Free Counter (`concurrent.Counter`)

A thread-safe counter using atomic operations for all mutations.

**Features:**
- `Inc()` / `Dec()` - Increment/decrement by 1
- `Add(delta)` / `Sub(delta)` - Add/subtract arbitrary values
- `Load()` / `Store(value)` - Read/write current value
- `CompareAndSwap(old, new)` - Atomic CAS operation
- `Swap(new)` - Atomic swap returning old value
- `Reset()` - Reset to 0 and return old value

**Performance:**
```
BenchmarkCounter_Inc-14              698M ops/sec (1.6 ns/op)
BenchmarkCounter_IncParallel-14      27.7M ops/sec (48.7 ns/op)
BenchmarkCounter_Load-14             1000M ops/sec (0.24 ns/op)
BenchmarkCounter_LoadParallel-14     1000M ops/sec (0.09 ns/op)
```

**Comparison to Mutex-based Counter:**
- Sequential: ~1.45x faster (1.6ns vs 2.3ns)
- Parallel: ~2.15x faster (48.7ns vs 104.7ns)

**Use Cases:**
- Hit/miss counters in cache implementations
- Request counters in HTTP servers
- Statistics tracking
- Transaction ID generation

**Example:**
```go
import "github.com/mnohosten/laura-db/pkg/concurrent"

counter := concurrent.NewCounter()

// Multiple goroutines can safely increment
go func() {
    for i := 0; i < 1000; i++ {
        counter.Inc()
    }
}()

value := counter.Load()  // Read current value
```

---

### 2. Lock-Free Stack (`concurrent.LockFreeStack`)

A thread-safe stack implementation based on Treiber's algorithm using atomic pointer operations.

**Features:**
- `Push(value)` - Add element to top (LIFO)
- `Pop()` - Remove and return element from top
- `Peek()` - View top element without removing
- `IsEmpty()` - Check if stack is empty
- `Size()` - Get approximate element count
- `Clear()` - Remove all elements atomically

**Performance:**
```
BenchmarkStack_Push-14               36.9M ops/sec (30.9 ns/op)
BenchmarkStack_PushParallel-14       2.8M ops/sec (429 ns/op)
BenchmarkStack_Pop-14                248M ops/sec (6.1 ns/op)
BenchmarkStack_PopParallel-14        1000M ops/sec (0.58 ns/op)
BenchmarkStack_PushPop-14            52.7M ops/sec (22.1 ns/op)
```

**Algorithm Details:**
- Uses Treiber's lock-free stack algorithm (1986)
- Atomic CAS operations on head pointer
- ABA problem mitigated by Go's garbage collector
- Retry loop on CAS failure (optimistic concurrency)

**Use Cases:**
- Work-stealing queues
- Temporary object pools
- Undo/redo stacks
- Call stacks in async processing

**Example:**
```go
stack := concurrent.NewLockFreeStack()

// Push items
stack.Push("task1")
stack.Push("task2")
stack.Push("task3")

// Pop items (LIFO order)
if value, ok := stack.Pop(); ok {
    fmt.Println(value)  // "task3"
}

// Peek without removing
if value, ok := stack.Peek(); ok {
    fmt.Println(value)  // "task2"
}
```

---

### 3. Sharded LRU Cache (`concurrent.ShardedLRUCache`)

A high-performance LRU cache that reduces lock contention by partitioning data across multiple shards.

**Features:**
- Configurable shard count (auto-rounded to power of 2)
- Per-shard LRU eviction
- TTL-based expiration
- Thread-safe Get/Put operations
- Aggregated statistics across all shards
- Periodic expired entry cleanup

**Architecture:**
```
ShardedLRUCache
├── Shard 0 (RWMutex)
│   ├── HashMap (items)
│   └── Linked List (LRU order)
├── Shard 1 (RWMutex)
│   ├── HashMap
│   └── Linked List
├── ...
└── Shard N (RWMutex)
    ├── HashMap
    └── Linked List
```

**Shard Selection:**
- Uses FNV-1 hash function (fast, non-cryptographic)
- Bitwise AND modulo for power-of-2 shard counts
- Uniform distribution across shards

**Performance:**
```
BenchmarkShardedLRU_Put-14              4.8M ops/sec (235 ns/op)
BenchmarkShardedLRU_PutParallel-14      8.0M ops/sec (159 ns/op)
BenchmarkShardedLRU_Get-14              13.5M ops/sec (88 ns/op)
BenchmarkShardedLRU_GetParallel-14      9.3M ops/sec (130 ns/op)
BenchmarkShardedLRU_Mixed-14            12.5M ops/sec (96 ns/op)
```

**Scalability with Shard Count:**
```
Shards   Parallel Mixed Ops/sec   Speedup vs 1 shard
------   ----------------------   ------------------
1        4.0M ops/sec (251 ns)    1.00x
2        4.3M ops/sec (234 ns)    1.07x
4        5.0M ops/sec (202 ns)    1.24x
8        7.5M ops/sec (133 ns)    1.88x
16       10.7M ops/sec (93 ns)    2.68x
32       14.0M ops/sec (71 ns)    3.50x
```

**Optimal Shard Count:**
- Rule of thumb: `shardCount = numCPUs * 2` to `numCPUs * 4`
- More shards = less contention but more overhead
- Diminishing returns beyond 32 shards for most workloads
- Default: 8 shards works well for most use cases

**Use Cases:**
- Query result caching (LauraDB query cache)
- HTTP response caching
- Computed value memoization
- Session storage

**Example:**
```go
// Create cache with 1000 capacity, 5min TTL, 8 shards
cache := concurrent.NewShardedLRUCache(1000, 5*time.Minute, 8)

// Put items (distributes across shards)
cache.Put("user:123", userData)
cache.Put("session:abc", sessionData)

// Get items
if value, ok := cache.Get("user:123"); ok {
    user := value.(*UserData)
    // Use cached data
}

// View statistics
stats := cache.Stats()
fmt.Printf("Hit rate: %v\n", stats["hit_rate"])
fmt.Printf("Shards: %v\n", stats["shard_count"])

// Periodic cleanup of expired entries
removed := cache.CleanupExpired()
```

---

## Design Principles

### 1. Atomic Operations
All lock-free structures use `sync/atomic` package operations:
- `atomic.LoadUint64()` / `atomic.StoreUint64()`
- `atomic.AddUint64()`
- `atomic.CompareAndSwapUint64()`
- `atomic.CompareAndSwapPointer()`

### 2. Memory Ordering
Go's memory model guarantees:
- Atomic operations have sequential consistency
- Happens-before relationships are preserved
- No need for explicit memory barriers

### 3. ABA Problem Mitigation
The ABA problem (value changes from A→B→A between CAS operations) is mitigated by:
- Go's garbage collector prevents premature reclamation
- No explicit memory management required
- Version counters not needed for most cases

### 4. Progress Guarantees
- **Lock-free**: At least one thread makes progress in finite steps
- **Wait-free**: Not achieved (retry loops in CAS can theoretically starve)
- **Obstruction-free**: Single thread in isolation makes progress

### 5. Cache-Line Awareness
- Counters and atomic pointers aligned to cache lines
- False sharing minimized through padding (when needed)
- Sharding reduces cross-core cache line bouncing

---

## Performance Comparison

### Counter: Lock-Free vs Mutex

| Operation | Lock-Free | Mutex-based | Speedup |
|-----------|-----------|-------------|---------|
| Sequential Inc | 1.6 ns | 2.3 ns | 1.45x |
| Parallel Inc | 48.7 ns | 104.7 ns | 2.15x |
| Sequential Load | 0.24 ns | N/A | - |
| Parallel Load | 0.09 ns | N/A | - |

### Cache: Sharded vs Single Lock

| Shard Count | Mixed Workload | Improvement |
|-------------|----------------|-------------|
| 1 (baseline) | 251 ns/op | 1.00x |
| 8 shards | 133 ns/op | 1.88x |
| 16 shards | 93 ns/op | 2.68x |
| 32 shards | 71 ns/op | 3.50x |

---

## Testing

### Test Coverage
- **33 unit tests** covering correctness and concurrency
- **25 benchmark tests** measuring performance
- All tests passing (33/33)

### Concurrent Testing
Tests include:
- Race condition detection (`go test -race`)
- High-contention scenarios (10+ goroutines)
- Mixed operations (reads + writes)
- Stress tests (1000+ operations per goroutine)

### Benchmark Methodology
- `benchtime=1s` for stable results
- Parallel benchmarks use `b.RunParallel()`
- CPU: Apple M4 Max (14 cores)
- Go version: 1.25.4

---

## Integration with LauraDB

### Potential Applications

1. **Query Cache** (`pkg/cache`)
   - Replace `sync.RWMutex` with `ShardedLRUCache`
   - Expected: 2-3x improvement under high concurrency
   - Current hit rate stats can use `Counter` instead of uint64

2. **Statistics Counters** (`pkg/index/stats.go`)
   - Use `Counter` for cardinality, insert count, delete count
   - Reduces lock contention during stat updates
   - Maintains accuracy with atomic operations

3. **Buffer Pool** (`pkg/storage/buffer_pool.go`)
   - Hit/miss counters → `Counter`
   - Eviction counter → `Counter`
   - Core page management still needs mutex (map access)

4. **MVCC Transaction IDs** (`pkg/mvcc/transaction.go`)
   - Already uses `atomic.AddUint64` for `nextTxnID`
   - Can formalize with `Counter` for better API

5. **Object ID Generation** (`pkg/document/objectid.go`)
   - Already uses `atomic.AddUint32`
   - Can use `Counter` for consistency

### Migration Strategy

**Phase 1: Statistics Counters** (Low Risk)
- Replace uint64 counters with `concurrent.Counter`
- No behavior changes, only performance improvement
- Easy to roll back if issues arise

**Phase 2: Query Cache** (Medium Risk)
- Create new `ShardedQueryCache` wrapper
- A/B test against existing `LRUCache`
- Gradual rollout with feature flag

**Phase 3: Version Store** (High Risk - Future)
- Research lock-free hash map implementations
- Prototype concurrent version chain access
- Extensive testing before production use

---

## Limitations and Caveats

### 1. Go Runtime Overhead
- Go's runtime adds overhead to goroutine scheduling
- CAS retry loops can cause temporary spinning
- Not as fast as C/C++ lock-free implementations

### 2. Memory Model
- Go's memory model is simpler than C++
- Less control over memory ordering
- Relies on runtime guarantees

### 3. Garbage Collection
- GC pauses can affect lock-free performance
- No manual memory management
- ABA problem mostly solved but not eliminated

### 4. API Constraints
- Generic `interface{}` requires type assertions
- No generics in Go 1.25 (though available in newer versions)
- Runtime type checking overhead

### 5. Debugging
- Lock-free bugs are harder to reproduce
- Race detector helps but doesn't catch all issues
- Requires careful reasoning about memory ordering

---

## Best Practices

### 1. When to Use Lock-Free
✅ **Good use cases:**
- Simple counters and statistics
- Read-heavy workloads with occasional writes
- Hot paths with high contention
- Fixed-size data structures

❌ **Avoid lock-free for:**
- Complex state management
- Algorithms requiring multiple atomic operations
- Dynamic resizing (use sharding instead)
- Code where correctness is hard to verify

### 2. Sharding Strategy
```go
// Good: Power of 2 shards for fast modulo
cache := NewShardedLRUCache(1000, ttl, 8)

// Good: Adjust based on CPU count
shards := runtime.NumCPU() * 2
cache := NewShardedLRUCache(capacity, ttl, uint32(shards))

// Avoid: Non-power-of-2 (auto-rounded up)
cache := NewShardedLRUCache(1000, ttl, 7)  // Becomes 8
```

### 3. Testing
```go
// Always run tests with race detector
go test -race ./pkg/concurrent

// Stress test with high goroutine count
go test -run=TestConcurrent -count=100

// Benchmark with parallelism
go test -bench=Parallel -benchtime=5s
```

### 4. Error Handling
```go
// Lock-free stack Pop returns (value, ok)
if value, ok := stack.Pop(); ok {
    // Process value
} else {
    // Stack was empty
}

// Don't assume Pop always succeeds
value := stack.Pop()  // Wrong! Returns (interface{}, bool)
```

---

## References

### Academic Papers
- Treiber, R.K. (1986). "Systems Programming: Coping with Parallelism"
  - Lock-free stack algorithm
- Herlihy, M. & Shavit, N. (2008). "The Art of Multiprocessor Programming"
  - Comprehensive lock-free algorithms reference

### Go Memory Model
- https://go.dev/ref/mem
- https://research.swtch.com/gomm

### Related Work
- `github.com/puzpuzpuz/xsync` - Concurrent maps for Go
- `github.com/tidwall/hashmap` - Fast concurrent hash map
- `github.com/cornelk/hashmap` - Lock-free hash map

---

## Future Work

### Planned Enhancements
1. **Lock-Free Queue** (MPMC)
   - Michael-Scott queue algorithm
   - Use case: Task distribution, work stealing

2. **Lock-Free Hash Map** (Experimental)
   - Split-ordered lists or hopscotch hashing
   - Replace version store maps

3. **Generics Support** (Go 1.18+)
   - Type-safe lock-free structures
   - Eliminate interface{} overhead

4. **Memory Pooling**
   - Object pools for stack nodes
   - Reduce GC pressure

5. **Performance Tuning**
   - Cache line padding
   - NUMA awareness
   - Backoff strategies for CAS retry loops

---

## Summary

The `pkg/concurrent` package provides production-ready lock-free data structures that improve performance under concurrent workloads:

- **Counter**: 2.15x faster than mutex-based in parallel workloads
- **Stack**: Lock-free with retry-based CAS operations
- **Sharded LRU**: 3.5x speedup with 32 shards vs single lock

These structures are designed for integration into LauraDB's hot paths (query cache, statistics, buffer pool) to reduce lock contention and improve scalability on multi-core systems.

**Key Takeaway**: Lock-free data structures are not a silver bullet. Use them judiciously for simple, high-contention operations where correctness is easy to verify. For complex state management, traditional locking with careful design is often simpler and safer.
