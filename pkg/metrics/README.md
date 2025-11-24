# Metrics Package

Real-time performance metrics collection and slow query logging for LauraDB with low overhead and comprehensive insights.

## Features

### Real-Time Metrics
- **Lock-free atomic operations**: ~7ns per metric recording
- **Timing histograms**: Automatic bucketing (0-1ms, 1-10ms, 10-100ms, 100-1000ms, >1s)
- **Percentile tracking**: P50, P95, P99 latency percentiles
- **Thread-safe**: Safe for concurrent use across multiple goroutines
- **Comprehensive metrics**: Queries, inserts, updates, deletes, transactions, cache, scans, connections
- **Zero allocation recording**: No allocations during metric recording (except histogram)

### Slow Query Log
- **Configurable threshold**: Log queries exceeding a duration threshold (default: 100ms)
- **In-memory buffer**: Keep recent slow queries in memory (default: 1000 entries)
- **File persistence**: Optional JSON log file for permanent storage
- **Rich query context**: Collection, filter, execution plan, index usage, documents examined
- **Query analysis**: Statistics, top slowest queries, per-collection analysis
- **Export capabilities**: JSON export for external analysis

### Query Profiler
- **Detailed stage timing**: Track query execution through multiple stages (parse, optimize, execute, etc.)
- **Nested profiling**: Profile sub-operations within each stage
- **Metadata support**: Attach custom metadata to profiling sessions
- **Bottleneck identification**: Automatically identify slowest stages
- **Percentage breakdown**: See what percentage of time each stage consumed
- **Human-readable summaries**: Format profile results for easy analysis
- **Zero overhead when disabled**: No performance impact when profiling is off

### Resource Tracker
- **Memory tracking**: Heap, stack, allocations, object count monitoring
- **Goroutine tracking**: Monitor concurrent goroutine count
- **I/O tracking**: Track bytes read/written and operation counts
- **GC statistics**: Monitor garbage collection runs and pause times
- **Historical sampling**: Keep time-series data for trend analysis (default: 60 samples @ 1s interval)
- **Trend analysis**: Automatic calculation of growth rates and averages
- **Thread-safe**: Atomic counters for concurrent access
- **Configurable sampling**: Adjust sample interval and history depth

## Usage

### Basic Usage

```go
import "github.com/mnohosten/laura-db/pkg/metrics"

// Create a metrics collector
mc := metrics.NewMetricsCollector()

// Record query execution
start := time.Now()
// ... execute query ...
mc.RecordQuery(time.Since(start), true) // true = success

// Record cache operations
mc.RecordCacheHit()
mc.RecordCacheMiss()

// Get all metrics
snapshot := mc.GetMetrics()
fmt.Printf("Metrics: %+v\n", snapshot)
```

### Integration with Database

```go
type Database struct {
    metrics *metrics.MetricsCollector
    // ... other fields
}

func (db *Database) executeQuery(query string) error {
    start := time.Now()
    defer func() {
        success := true
        if r := recover(); r != nil {
            success = false
            panic(r)
        }
        db.metrics.RecordQuery(time.Since(start), success)
    }()

    // Execute query
    return nil
}
```

### Available Metrics

```go
metrics := mc.GetMetrics()

// Query metrics
queries := metrics["queries"].(map[string]interface{})
queries["total"]              // Total queries executed
queries["failed"]             // Failed queries
queries["success_rate"]       // Success rate (%)
queries["avg_duration_ms"]    // Average duration in ms
queries["timing_histogram"]   // Histogram buckets
queries["timing_percentiles"] // P50, P95, P99

// Insert/Update/Delete metrics (same structure as queries)
inserts := metrics["inserts"].(map[string]interface{})
updates := metrics["updates"].(map[string]interface{})
deletes := metrics["deletes"].(map[string]interface{})

// Transaction metrics
txns := metrics["transactions"].(map[string]interface{})
txns["started"]     // Transactions started
txns["committed"]   // Transactions committed
txns["aborted"]     // Transactions aborted
txns["commit_rate"] // Commit success rate (%)

// Cache metrics
cache := metrics["cache"].(map[string]interface{})
cache["hits"]      // Cache hits
cache["misses"]    // Cache misses
cache["hit_rate"]  // Hit rate (%)

// Scan metrics
scans := metrics["scans"].(map[string]interface{})
scans["index"]           // Index scans
scans["collection"]      // Collection scans
scans["index_usage_pct"] // Index usage percentage

// Connection metrics (for HTTP server)
conns := metrics["connections"].(map[string]interface{})
conns["active"] // Active connections
conns["total"]  // Total connections

// Uptime
uptime := metrics["uptime_seconds"].(float64)
```

## API Reference

### MetricsCollector

```go
// Create new collector
mc := NewMetricsCollector()

// Record operations
mc.RecordQuery(duration, success)
mc.RecordInsert(duration, success)
mc.RecordUpdate(duration, success)
mc.RecordDelete(duration, success)

// Record transactions
mc.RecordTransactionStart()
mc.RecordTransactionCommit()
mc.RecordTransactionAbort()

// Record cache operations
mc.RecordCacheHit()
mc.RecordCacheMiss()

// Record scans
mc.RecordIndexScan()
mc.RecordCollectionScan()

// Record connections
mc.RecordConnectionStart()
mc.RecordConnectionEnd()

// Get metrics snapshot
metrics := mc.GetMetrics()

// Reset all metrics
mc.Reset()
```

### TimingHistogram

```go
th := NewTimingHistogram(1000) // Keep last 1000 timings

// Record timing
th.Record(10 * time.Millisecond)

// Get histogram buckets
buckets := th.GetBuckets()
// Returns: {"0-1ms": 5, "1-10ms": 12, "10-100ms": 8, ...}

// Get percentiles
percentiles := th.GetPercentiles()
// Returns: {"p50": 10ms, "p95": 95ms, "p99": 120ms}
```

### SlowQueryLog

```go
// Create slow query log
config := &SlowQueryLogConfig{
    Threshold:   100 * time.Millisecond,  // Log queries > 100ms
    MaxEntries:  1000,                     // Keep 1000 in memory
    LogFilePath: "/var/log/slow_query.log", // Optional file logging
    Enabled:     true,
}
sql, err := NewSlowQueryLog(config)

// Log a slow query
sql.LogQuery(SlowQueryEntry{
    Duration:     150 * time.Millisecond,
    Operation:    "query",
    Collection:   "users",
    Filter:       map[string]interface{}{"age": int64(25)},
    DocsExamined: 1000,
    DocsReturned: 50,
    IndexUsed:    "age_idx",
})

// Get all entries
entries := sql.GetEntries()

// Get recent entries
recent := sql.GetRecentEntries(10)

// Get top slowest queries
slowest := sql.GetTopSlowest(5)

// Filter by collection
userQueries := sql.GetEntriesByCollection("users")

// Filter by operation
queries := sql.GetEntriesByOperation("query")

// Get statistics
stats := sql.GetStatistics()
// Returns: total_entries, avg/min/max duration, breakdown by operation/collection

// Export to JSON
sql.ExportToJSON(os.Stdout)

// Update threshold
sql.SetThreshold(200 * time.Millisecond)

// Enable/disable
sql.Disable()
sql.Enable()

// Clear all entries
sql.Clear()

// Close (flushes file if using file logging)
sql.Close()
```

### QueryProfiler

```go
// Create profiler
profiler := NewQueryProfiler(true) // enabled

// Start profiling session
session := profiler.StartProfile()
if session != nil {
    session.AddMetadata("collection", "users")
    session.AddMetadata("operation", "find")

    // Profile stages
    session.StartStage("parse_query")
    // ... parse query ...
    session.AddStageDetail("complexity", "simple")
    session.EndStage()

    session.StartStage("optimize")
    session.AddStageDetail("indexes_available", 3)
    session.AddStageDetail("index_selected", "age_idx")
    // ... optimize query ...
    session.EndStage()

    session.StartStage("execute")
    session.AddStageDetail("docs_scanned", 1000)
    session.AddStageDetail("docs_returned", 50)
    // ... execute query ...
    session.EndStage()

    // Get results
    result := session.Finish()

    // Print summary
    fmt.Println(result.GetSummary())

    // Get bottleneck
    bottleneck := result.GetBottleneck()
    fmt.Printf("Slowest stage: %s (%.2fms)\n", bottleneck.Name, bottleneck.DurationMS)

    // Get stage percentages
    percentages := result.GetStagePercentages()
    for stage, pct := range percentages {
        fmt.Printf("%s: %.1f%%\n", stage, pct)
    }

    // Get slow stages (> 10ms)
    slowStages := result.GetSlowStages(10 * time.Millisecond)
}

// Using defer for automatic timing
func executeQuery() {
    session := profiler.StartProfile()
    if session != nil {
        defer session.EndStage()
        session.StartStage("query_execution")
    }
    // ... query execution ...
}

// Helper for convenient stage timing
defer TimeStage(session, "stage_name")()

// Enable/disable profiling
profiler.Disable()
profiler.Enable()
```

### ResourceTracker

```go
// Create resource tracker
config := &ResourceTrackerConfig{
    Enabled:        true,
    SampleInterval: 1 * time.Second,  // Sample every second
    MaxSamples:     60,                // Keep 60 samples (1 minute of history)
}
rt := NewResourceTracker(config)
defer rt.Close()

// Record I/O operations
rt.RecordRead(4096)  // Read 4KB
rt.RecordWrite(2048) // Wrote 2KB

// Get current statistics
stats := rt.GetStats()
fmt.Printf("Heap in use: %.2f MB\n", stats.HeapInUseMB)
fmt.Printf("Goroutines: %d\n", stats.NumGoroutines)
fmt.Printf("Bytes read: %d\n", stats.BytesRead)
fmt.Printf("Bytes written: %d\n", stats.BytesWritten)
fmt.Printf("GC runs: %d\n", stats.GCRuns)
fmt.Printf("GC pause: %.2f ms\n", stats.GCPauseTotalMs)

// Get sample history
samples := rt.GetSamples()
for _, sample := range samples {
    fmt.Printf("%s: Heap=%dMB, Goroutines=%d\n",
        sample.Timestamp.Format(time.RFC3339),
        sample.HeapInUse/1024/1024,
        sample.NumGoroutines)
}

// Get trend analysis
trends := rt.GetTrends()
fmt.Printf("Heap growth: %.2f MB\n", trends["heap_growth_mb"])
fmt.Printf("Goroutine growth: %d\n", trends["goroutine_growth"])
fmt.Printf("Average heap: %.2f MB\n", trends["avg_heap_mb"])
fmt.Printf("Average goroutines: %.0f\n", trends["avg_goroutines"])

// Clear sample history
rt.ClearSamples()

// Enable/disable tracking
rt.Disable()
rt.Enable()
```

## Performance

Benchmarks on Apple M4 Max:

```
BenchmarkMetricsCollector_RecordQuery-14         163M    7.0 ns/op    22 B/op    0 allocs/op
BenchmarkMetricsCollector_RecordInsert-14        166M    7.0 ns/op    22 B/op    0 allocs/op
BenchmarkMetricsCollector_GetMetrics-14          312K    3.9 µs/op    21KB/op    47 allocs/op
BenchmarkTimingHistogram_Record-14               191M    6.4 ns/op    22 B/op    0 allocs/op
BenchmarkTimingHistogram_GetPercentiles-14       899K    1.3 µs/op    8.4KB/op   3 allocs/op
BenchmarkMetricsCollector_Parallel-14            8.5M    144 ns/op    22 B/op    0 allocs/op
BenchmarkMetricsCollector_MixedOperations-14     38M     29 ns/op     91 B/op    0 allocs/op
```

- **Recording overhead**: ~7 nanoseconds per operation
- **Concurrent safe**: Lock-free atomic operations
- **GetMetrics**: ~4 microseconds to generate full snapshot
- **Percentile calculation**: ~1.3 microseconds for P50/P95/P99

## Architecture

### Atomic Counters

All counters use `sync/atomic` for lock-free operations:
- Queries, inserts, updates, deletes (total and failed)
- Transactions (started, committed, aborted)
- Cache (hits, misses)
- Scans (index, collection)
- Connections (active, total)

### Timing Histograms

Each operation type (query, insert, update, delete) maintains:
- **Histogram buckets**: Atomic counters for 5 time ranges
- **Recent timings**: Circular buffer of last N timings for percentiles
- **Mutex protection**: Only for percentile calculations (not hot path)

### Memory Usage

- Base collector: ~200 bytes
- Per histogram: ~8KB (1000 recent timings)
- Total footprint: ~35KB
- GetMetrics snapshot: ~22KB per call

## Thread Safety

All operations are thread-safe:
- Atomic operations for counters (no locks)
- Mutex protection for histogram percentile calculations
- Safe for concurrent reads and writes

## Use Cases

1. **Real-time monitoring**: Track database performance in production
2. **Performance analysis**: Identify slow queries and bottlenecks
3. **Capacity planning**: Monitor resource usage trends
4. **SLA tracking**: Ensure P95/P99 latencies meet targets
5. **HTTP server metrics**: Track connection and request patterns

## Integration Examples

### HTTP Server Middleware

```go
func MetricsMiddleware(mc *metrics.MetricsCollector) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            mc.RecordConnectionStart()
            defer mc.RecordConnectionEnd()

            start := time.Now()
            next.ServeHTTP(w, r)
            duration := time.Since(start)

            // Record based on endpoint
            if strings.Contains(r.URL.Path, "/query") {
                mc.RecordQuery(duration, true)
            }
        })
    }
}
```

### Collection Operations

```go
func (c *Collection) Find(filter map[string]interface{}) ([]*Document, error) {
    start := time.Now()
    success := false
    defer func() {
        c.metrics.RecordQuery(time.Since(start), success)
    }()

    // Execute query
    docs, err := c.executeQuery(filter)
    if err != nil {
        return nil, err
    }

    success = true
    return docs, nil
}
```

## Testing

Run tests:
```bash
go test ./pkg/metrics -v
```

Run benchmarks:
```bash
go test ./pkg/metrics -bench=. -benchmem
```

## Future Enhancements

- [ ] Prometheus exporter
- [ ] Grafana dashboard templates
- [ ] Custom percentiles (P90, P999)
- [ ] Time-series retention policies
- [ ] Metric aggregation by collection/operation
