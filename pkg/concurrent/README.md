# Lock-Free Concurrent Data Structures

This package provides high-performance lock-free and low-contention concurrent data structures for use in LauraDB.

## Data Structures

### Counter
Lock-free counter using atomic operations.

```go
counter := concurrent.NewCounter()
counter.Inc()                    // Returns new value
counter.Add(10)                  // Add arbitrary value
value := counter.Load()          // Read current value
counter.CompareAndSwap(old, new) // Atomic CAS operation
```

**Performance**: 2.15x faster than mutex-based counter in parallel workloads.

### LockFreeStack
Thread-safe stack using Treiber's algorithm.

```go
stack := concurrent.NewLockFreeStack()
stack.Push("item1")
stack.Push("item2")

if value, ok := stack.Pop(); ok {
    fmt.Println(value) // "item2" (LIFO)
}
```

**Performance**: 30.9ns push, 6.1ns pop operations.

### ShardedLRUCache
LRU cache with reduced lock contention through sharding.

```go
// Create cache: capacity=1000, TTL=5min, 8 shards
cache := concurrent.NewShardedLRUCache(1000, 5*time.Minute, 8)

cache.Put("key", value)
if value, ok := cache.Get("key"); ok {
    // Use cached value
}

stats := cache.Stats()
fmt.Printf("Hit rate: %v\n", stats["hit_rate"])
```

**Performance**: 3.5x faster with 32 shards vs single lock (71ns vs 251ns).

## Documentation

See [docs/lock-free-data-structures.md](../../docs/lock-free-data-structures.md) for:
- Detailed API documentation
- Performance benchmarks
- Design principles
- Integration guide
- Best practices

## Testing

```bash
# Run tests
go test ./pkg/concurrent -v

# Run with race detector
go test ./pkg/concurrent -v -race

# Run benchmarks
go test ./pkg/concurrent -bench=. -benchtime=1s
```

All 33 tests pass with race detector enabled.

## Benchmarks

| Data Structure | Operation | Sequential | Parallel |
|----------------|-----------|------------|----------|
| Counter | Inc | 1.6 ns/op | 48.7 ns/op |
| Counter | Load | 0.24 ns/op | 0.09 ns/op |
| Stack | Push | 30.9 ns/op | 429 ns/op |
| Stack | Pop | 6.1 ns/op | 0.58 ns/op |
| Sharded Cache (8) | Get | 88 ns/op | 130 ns/op |
| Sharded Cache (32) | Mixed | - | 71 ns/op |

Platform: Apple M4 Max (14 cores), Go 1.25.4
