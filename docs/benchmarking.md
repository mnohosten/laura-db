# Performance Benchmarking

LauraDB includes a comprehensive automated performance benchmarking system to track performance over time and detect regressions.

## Overview

The benchmarking system consists of:

1. **Extensive benchmark suite** - Performance tests across all major components
2. **Automated CI/CD benchmarks** - GitHub Actions workflow for continuous benchmarking
3. **Local benchmark tools** - Scripts for running and comparing benchmarks locally
4. **Performance tracking** - Historical benchmark data and regression detection

## Running Benchmarks

### Quick Start

```bash
# Run all benchmarks
make bench-all

# Run specific package benchmarks
make bench           # database and index packages
make bench-insert    # insertion benchmarks
make bench-find      # query benchmarks
make bench-index     # index benchmarks
```

### Benchmark Packages

LauraDB includes benchmarks for:

- **pkg/database/** - Collection operations (insert, find, update, delete)
- **pkg/index/** - B+ tree operations
- **pkg/query/** - Query execution and optimization
- **pkg/storage/** - Storage engine operations
- **pkg/mvcc/** - Transaction processing

Key benchmark files:
- `collection_bench_test.go` - Core CRUD operations
- `cache_bench_test.go` - Query cache performance
- `btree_bench_test.go` - B+ tree index performance
- `covered_bench_test.go` - Covered query optimization
- `parallel_bench_test.go` - Parallel query execution
- `text_index_bench_test.go` - Text search performance
- `geo_bench_test.go` - Geospatial query performance

### Advanced Benchmark Options

```bash
# Run with custom benchmark time
go test -bench=. -benchmem -benchtime=10s ./pkg/database

# Run specific benchmark function
go test -bench=BenchmarkInsertOne -benchmem ./pkg/database

# Run with multiple iterations for statistical accuracy
go test -bench=. -benchmem -count=10 ./pkg/database

# Save results to file
go test -bench=. -benchmem ./pkg/... > bench-results.txt
```

## Performance Tracking

### Creating a Baseline

Before making changes, create a baseline to compare against:

```bash
# Create baseline benchmark
make bench-baseline
```

This runs all benchmarks 5 times and stores the results in `benchmarks/baseline.txt`.

### Checking for Regressions

After making changes, check for performance regressions:

```bash
# Run benchmarks and compare with baseline
make bench-check
```

This will:
1. Run all benchmarks with current code
2. Compare results with the baseline
3. Show performance differences

### Manual Comparison

Compare two specific benchmark results:

```bash
# Compare two benchmark files
make bench-compare OLD=benchmarks/old.txt NEW=benchmarks/new.txt
```

### Using benchstat

For detailed statistical comparison, install `benchstat`:

```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

Then run comparisons:

```bash
benchstat benchmarks/baseline.txt benchmarks/current.txt
```

Output example:
```
name                  old time/op    new time/op    delta
InsertOne-8             45.2µs ± 2%    42.8µs ± 1%   -5.31%
FindOne-8               12.3µs ± 1%    11.9µs ± 2%   -3.25%
UpdateOne-8             38.7µs ± 3%    37.1µs ± 2%   -4.13%
```

## Automated Benchmarking (CI/CD)

### GitHub Actions Workflow

The project includes automated benchmarking via GitHub Actions (`.github/workflows/benchmarks.yml`).

**Triggers:**
- Every push to `main` or `develop`
- Every pull request
- Daily at 2 AM UTC (scheduled)
- Manual dispatch

**What it does:**
1. Runs full benchmark suite
2. Uploads results as artifacts
3. Comments benchmark summary on PRs
4. Stores historical data for `main` branch

### Viewing CI Benchmark Results

**On Pull Requests:**
- Benchmark results are automatically posted as a comment
- Download full results from workflow artifacts

**On Main Branch:**
- Benchmark history stored as artifacts (365 days retention)
- Access via Actions tab → Workflow runs → Artifacts

### Continuous Benchmarking

The `continuous-benchmarking` job (runs only on `main` branch):
- Runs benchmarks 5 times for statistical accuracy
- Creates timestamped results with commit SHA
- Stores results for historical tracking
- Generates performance reports

## Benchmark Script

The `scripts/benchmark.sh` script provides convenient benchmark management:

```bash
# Show help
./scripts/benchmark.sh help

# Run benchmarks and save with timestamp
./scripts/benchmark.sh run

# Create performance baseline
./scripts/benchmark.sh baseline

# Run and compare with baseline
./scripts/benchmark.sh check

# Compare two result files
./scripts/benchmark.sh compare old.txt new.txt

# Clean old benchmark results (>30 days)
./scripts/benchmark.sh clean
```

## Interpreting Results

### Benchmark Output Format

```
BenchmarkInsertOne-8    26453    45234 ns/op    8192 B/op    12 allocs/op
```

- `BenchmarkInsertOne-8` - Benchmark name (-8 = 8 CPU cores)
- `26453` - Number of iterations
- `45234 ns/op` - Nanoseconds per operation
- `8192 B/op` - Bytes allocated per operation
- `12 allocs/op` - Number of allocations per operation

### Performance Metrics to Watch

**Speed (ns/op):**
- Lower is better
- Look for regressions > 10%

**Memory (B/op):**
- Lower is better
- Unexpected increases may indicate leaks

**Allocations (allocs/op):**
- Lower is better
- Fewer allocations = less GC pressure

### Statistical Significance

For reliable comparisons:
- Run benchmarks multiple times (`-count=5` or more)
- Use `benchstat` for statistical analysis
- Look for consistent patterns across iterations
- Consider variance (±%) in results

## Best Practices

### When to Run Benchmarks

**Always benchmark:**
- Before and after performance optimizations
- When changing core data structures
- When modifying hot code paths
- Before merging performance-critical PRs

**Create baselines:**
- Before starting optimization work
- At version milestones
- When switching branches for comparison

### Writing Effective Benchmarks

See existing benchmarks for examples:

```go
func BenchmarkInsertOne(b *testing.B) {
    db := setupBenchDB(b)
    defer db.Close()

    coll := db.Collection("bench")

    b.ResetTimer() // Reset timer after setup

    for i := 0; i < b.N; i++ {
        doc := map[string]interface{}{
            "name": "user" + strconv.Itoa(i),
            "age": int64(i % 100),
        }
        _, err := coll.InsertOne(doc)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

**Tips:**
- Use `b.ResetTimer()` after setup code
- Use `b.StopTimer()` / `b.StartTimer()` to exclude cleanup
- Report custom metrics with `b.ReportMetric()`
- Use realistic data sizes
- Test both hot and cold cache scenarios

### Continuous Monitoring

1. **Review benchmark comments on PRs** - Check for unexpected regressions
2. **Monitor daily benchmark runs** - Track long-term performance trends
3. **Compare across versions** - Use historical artifacts
4. **Set performance budgets** - Define acceptable regression thresholds

## Troubleshooting

### Unstable Results

**Problem:** Benchmark results vary widely between runs

**Solutions:**
- Increase `-benchtime` (e.g., `-benchtime=10s`)
- Run more iterations (`-count=10`)
- Ensure machine is idle during benchmarking
- Disable CPU frequency scaling
- Use a dedicated benchmark machine

### Memory Profiling

To investigate memory issues:

```bash
# Run with memory profiling
go test -bench=BenchmarkInsertOne -benchmem -memprofile=mem.prof ./pkg/database

# Analyze profile
go tool pprof mem.prof
```

### CPU Profiling

To investigate CPU bottlenecks:

```bash
# Run with CPU profiling
go test -bench=BenchmarkInsertOne -cpuprofile=cpu.prof ./pkg/database

# Analyze profile
go tool pprof cpu.prof
```

## Performance Goals

Current performance targets (as of the latest benchmarks):

- **Insert**: < 50µs per document
- **Find**: < 15µs per document (with index)
- **Update**: < 40µs per document
- **Query cache hit**: < 5µs
- **B+ tree search**: < 10µs
- **Covered query**: < 50µs (no document fetch)

## Integration with Development Workflow

### Pre-commit

Before committing performance changes:
```bash
make bench-baseline  # Create baseline
# Make your changes
make bench-check     # Verify no regressions
```

### Pull Request Review

1. CI automatically runs benchmarks
2. Review benchmark comment on PR
3. Compare with expectations
4. Investigate any significant regressions
5. Document expected performance changes

### Release Process

Before each release:
1. Run full benchmark suite
2. Compare with previous release
3. Document performance improvements/changes
4. Update performance goals if needed

## Future Enhancements

Planned improvements:
- Automated regression detection with thresholds
- Performance trend visualization
- Benchmark result dashboard
- Integration with monitoring systems (Prometheus/Grafana)
- A/B testing framework for performance comparisons
- Benchmark result database for long-term tracking

## Resources

- [Go Benchmark Guide](https://pkg.go.dev/testing#hdr-Benchmarks)
- [benchstat Documentation](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
- [Profiling Go Programs](https://go.dev/blog/pprof)
- [How to Write Benchmarks in Go](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)

## See Also

- [Testing Documentation](../README.md#testing)
- [Performance Tuning](../CLAUDE.md#performance-optimization)
- [CI/CD Workflows](../.github/workflows/)
