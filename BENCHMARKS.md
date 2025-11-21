# LauraDB Performance Benchmarks

Performance benchmarking guide and baseline results for LauraDB.

## Quick Start

```bash
# Run main benchmarks
make bench

# Run all benchmarks with detailed output
make bench-all

# Run specific benchmarks
make bench-insert
make bench-find
make bench-index
```

## Baseline Results

**Test Environment:**
- CPU: Intel Core i5-7200U @ 2.50GHz (4 cores)
- OS: Linux (kernel 6.17.8)
- Go: 1.25.4

### Database Operations

| Benchmark | Ops/sec | ns/op | B/op | allocs/op | Notes |
|-----------|---------|-------|------|-----------|-------|
| InsertOne | 93K | 10,742 | 834 | 19 | Single document insert |
| FindWithoutIndex | 3.9K | 254,328 | 8,826 | 8 | Collection scan (1000 docs) |
| **FindWithIndex** | **24.3K** | **41,207** | **8,462** | **4** | **6x faster with index!** |
| UpdateOne | 18K | 55,500 | 10,738 | 120 | Single document update |
| Aggregation | 1.4K | 691,288 | 51,021 | 115 | Group by with avg |
| BulkInsert (100) | 960 | 1,042,546 | 76,466 | 1,235 | Insert 100 documents |

**Key Findings:**
- ✅ **Index performance**: 6x improvement (254µs → 41µs)
- ✅ **Write throughput**: ~93K inserts/second
- ✅ **Update throughput**: ~18K updates/second
- ⚠️ **Aggregation**: Room for optimization

### Index Operations (B+ Tree)

| Benchmark | Ops/sec | ns/op | B/op | allocs/op | Notes |
|-----------|---------|-------|------|-----------|-------|
| BTreeInsert | 132K | 7,585 | 477 | 9 | Insert into B+ tree |
| BTreeSearch | **17.5M** | **57** | **0** | **0** | Point lookup - very fast! |
| BTreeRangeScan | 3.1M | 321 | 32 | 2 | Scan 1000 keys |
| BTreeDelete | 11M | 91 | 0 | 0 | Delete from tree |
| Mixed Operations | 3.7M | 270 | 17 | 1 | 25% each: insert/search/scan/delete |

**Key Findings:**
- 🚀 **Search performance**: 57 nanoseconds per lookup!
- ✅ **Zero allocations**: Search and delete ops
- ✅ **Scalability**: Handles millions of ops/second

## Performance Analysis

### Index Impact

Without index (collection scan of 1000 docs):
```
254,328 ns/op = 254µs
```

With B+ tree index (O(log n) lookup):
```
41,207 ns/op = 41µs
```

**Speedup**: 6.2x faster

**Why?**
- Collection scan: O(n) - checks every document
- Index lookup: O(log n) - binary search in B+ tree

### Memory Efficiency

**Insert Operation**:
- 834 bytes/op
- 19 allocations/op
- Includes: document creation, index updates, memory management

**Index Search**:
- 0 bytes/op (stack only!)
- 0 allocations/op
- Pure pointer traversal

## Running Benchmarks

### Basic Commands

```bash
# All benchmarks
go test -bench=. -benchmem ./pkg/...

# Specific package
go test -bench=. -benchmem ./pkg/database

# Specific benchmark
go test -bench=BenchmarkInsertOne -benchmem ./pkg/database

# Run longer for more stable results
go test -bench=. -benchmem -benchtime=10s ./pkg/database

# With CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./pkg/database
go tool pprof cpu.prof
```

### Comparing Performance

```bash
# Save baseline
go test -bench=. ./pkg/database > old.txt

# Make changes
# ...

# Compare
go test -bench=. ./pkg/database > new.txt
benchstat old.txt new.txt
```

## Benchmark Details

### Database Benchmarks

**BenchmarkInsertOne**
- Inserts single documents with auto-generated ObjectIDs
- Includes index maintenance
- Measures real-world insert performance

**BenchmarkFindWithoutIndex**
- 1000 documents in collection
- Collection scan (checks every document)
- Baseline for comparison with indexed queries

**BenchmarkFindWithIndex**
- 1000 documents with index on query field
- Uses query planner to select index scan
- Demonstrates index performance benefit

**BenchmarkUpdateOne**
- Updates document with $set operator
- Includes finding document and applying update
- Realistic update workload

**BenchmarkAggregation**
- Pipeline: $match + $group with $avg
- 1000 documents across 5 cities
- Complex operation involving filtering and aggregation

**BenchmarkBulkInsert**
- Inserts 100 documents at once
- Tests bulk operation efficiency
- Includes transaction overhead

### Index Benchmarks

**BenchmarkBTreeInsert**
- Pure B+ tree insert performance
- No disk I/O (in-memory)
- Order-32 tree

**BenchmarkBTreeSearch**
- Point lookups in populated tree
- 10,000 pre-existing keys
- Zero-allocation fast path

**BenchmarkBTreeRangeScan**
- Scans 1000 consecutive keys
- Uses leaf node linked list
- Efficient sequential access

**BenchmarkBTreeDelete**
- Deletion from pre-populated tree
- Includes tree rebalancing
- Still very fast

**BenchmarkBTreeMixedOperations**
- Realistic mixed workload
- 25% insert, 25% search, 25% scan, 25% delete
- Simulates production usage

## Performance Goals

### Current (v0.1.0)
- ✅ 93K inserts/sec
- ✅ 24K indexed queries/sec
- ✅ 18K updates/sec
- ⚠️ 1.4K aggregations/sec

### Target (v0.2.0)
- 🎯 150K inserts/sec (1.6x)
- 🎯 50K indexed queries/sec (2x)
- 🎯 30K updates/sec (1.7x)
- 🎯 5K aggregations/sec (3.5x)

### Target (v1.0.0)
- 🎯 250K inserts/sec
- 🎯 100K indexed queries/sec
- 🎯 50K updates/sec
- 🎯 10K aggregations/sec

## Optimization Opportunities

### High Priority

**1. Aggregation Pipeline (3.5x target)**
- Current: 691µs per aggregation
- Opportunity: Pre-compute common aggregations
- Opportunity: Parallel stage execution
- Opportunity: Push-down predicates

**2. Bulk Operations (2x target)**
- Current: 1ms for 100 inserts (10µs each)
- Opportunity: Batch index updates
- Opportunity: Reduce per-document overhead

**3. Update Performance (1.7x target)**
- Current: 55µs per update
- Opportunity: In-place updates when possible
- Opportunity: Optimize document lookup

### Medium Priority

**4. Index Insert (1.5x target)**
- Current: 7.6µs per insert
- Opportunity: Bulk insertion optimization
- Opportunity: Lazy tree rebalancing

**5. Range Scans (2x target)**
- Current: 321ns for 1000 keys
- Opportunity: Prefetching
- Opportunity: SIMD operations

## Profiling

### CPU Profiling

```bash
go test -bench=BenchmarkInsertOne -cpuprofile=cpu.prof ./pkg/database
go tool pprof cpu.prof

# In pprof:
(pprof) top10
(pprof) list InsertOne
(pprof) web
```

### Memory Profiling

```bash
go test -bench=BenchmarkInsertOne -memprofile=mem.prof ./pkg/database
go tool pprof mem.prof

# In pprof:
(pprof) top10
(pprof) list InsertOne
```

### Trace Analysis

```bash
go test -bench=BenchmarkInsertOne -trace=trace.out ./pkg/database
go tool trace trace.out
```

## Continuous Benchmarking

### GitHub Actions (Planned)

```yaml
name: Benchmarks
on: [push, pull_request]
jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: go test -bench=. -benchmem ./pkg/... > bench.txt
      - uses: benchmark-action/github-action-benchmark@v1
        with:
          tool: 'go'
          output-file-path: bench.txt
```

## Best Practices

### Writing Benchmarks

1. **Reset timer after setup**
   ```go
   func BenchmarkOperation(b *testing.B) {
       // Setup
       setupStuff()

       b.ResetTimer()  // Don't count setup time
       for i := 0; i < b.N; i++ {
           operation()
       }
   }
   ```

2. **Use unique data**
   ```go
   for i := 0; i < b.N; i++ {
       key := fmt.Sprintf("key_%d", i)  // Unique per iteration
   }
   ```

3. **Prevent optimization**
   ```go
   var result interface{}
   for i := 0; i < b.N; i++ {
       result = operation()  // Prevent dead code elimination
   }
   _ = result  // Use result
   ```

4. **Clean up**
   ```go
   defer os.RemoveAll(testDir)
   defer db.Close()
   ```

### Interpreting Results

- **ns/op**: Nanoseconds per operation (lower is better)
- **B/op**: Bytes allocated per operation (lower is better)
- **allocs/op**: Number of allocations per operation (lower is better)

**Good performance indicators:**
- Sub-microsecond operations (<1,000 ns)
- Zero allocations for hot paths
- Linear scaling with data size

## Troubleshooting

### Benchmark Fails

```bash
# Clean up test data
rm -rf bench_* ./test_data*

# Run single benchmark
go test -bench=BenchmarkInsertOne -v ./pkg/database
```

### Inconsistent Results

```bash
# Run longer
go test -bench=. -benchtime=10s ./pkg/database

# Run multiple times
go test -bench=. -count=5 ./pkg/database
```

### High Memory Usage

```bash
# Check for leaks
go test -bench=. -memprofile=mem.prof ./pkg/database
go tool pprof -alloc_space mem.prof
```

## Related Documents

- [TESTING.md](./TESTING.md) - Test coverage and testing guide
- [TODO.md](./TODO.md) - Performance optimization tasks
- [ROADMAP.md](./ROADMAP.md) - Performance milestones

---

**Last Updated**: Initial benchmark suite completed
**Version**: v0.1.0
**Status**: ✅ All benchmarks passing with baseline metrics established
