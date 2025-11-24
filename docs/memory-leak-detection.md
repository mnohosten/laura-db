# Memory Leak Detection in LauraDB

This document describes the memory leak detection and profiling capabilities available in LauraDB.

## Overview

LauraDB provides comprehensive memory leak detection tools to ensure the database doesn't leak memory during normal operations. This is critical for long-running server processes and prevents memory growth issues in production.

## Components

### 1. Memory Leak Detection Tests (`pkg/metrics/memory_leak_test.go`)

Automated tests that verify memory stability across various operations:

- **ResourceTracker Lifecycle**: Verifies that creating and destroying ResourceTracker instances doesn't leak memory
- **Sampling Operations**: Ensures continuous sampling doesn't accumulate memory
- **I/O Recording**: Validates that recording thousands of I/O operations doesn't leak
- **Multiple Concurrent Trackers**: Tests that concurrent tracker usage is leak-free
- **Sample Clearing**: Verifies that clearing samples properly releases memory
- **Enable/Disable Cycling**: Ensures toggling tracking on/off doesn't leak
- **Trend Calculations**: Validates that repeated trend analysis doesn't grow memory

### 2. Memory Leak Detector Utility

The `MemoryLeakDetector` provides a reusable testing utility:

```go
detector := NewMemoryLeakDetector(
    0.2,  // 20% threshold for heap growth
    2,    // Allow 2 goroutine variance
)

// Set baseline before operations
detector.SetBaseline()

// ... perform operations ...

// Check for leaks
if detector.CheckLeak(t, "operation name") {
    t.Error("Memory leak detected")
}
```

**Features:**
- Automatic garbage collection before measurements
- Heap allocation tracking
- Goroutine leak detection
- Configurable thresholds
- Detailed logging

### 3. Memory Profiling Script (`scripts/memory-profile.sh`)

A comprehensive shell script for memory profiling and leak detection:

```bash
# Run memory leak detection tests
./scripts/memory-profile.sh test

# Generate memory profile for a package
./scripts/memory-profile.sh profile ./pkg/database

# Generate heap profile
./scripts/memory-profile.sh heap ./pkg/storage

# Analyze a profile
./scripts/memory-profile.sh analyze profiles/mem.prof

# Compare two profiles
./scripts/memory-profile.sh compare old.prof new.prof

# Run benchmarks with profiling
./scripts/memory-profile.sh benchmark

# Run all checks
./scripts/memory-profile.sh check

# Clean up profiles
./scripts/memory-profile.sh clean
```

### 4. Makefile Integration

Memory profiling is fully integrated into the build system:

```bash
# Run leak detection tests
make memory-leak

# Profile a specific package
make memory-profile PKG=./pkg/database

# Generate heap profile
make memory-heap PKG=./pkg/storage

# Run all memory checks
make memory-check

# Analyze a profile
make memory-analyze PROFILE=profiles/mem.prof

# Clean profiles
make memory-clean
```

## Usage Guide

### Running Leak Detection Tests

The simplest way to check for memory leaks:

```bash
make memory-leak
```

This runs all automated leak detection tests. Tests pass if:
- Heap growth is below threshold (typically 20%)
- Goroutine count doesn't increase significantly
- Memory is properly released after operations

**Example output:**
```
PASS ResourceTracker lifecycle: Heap: 0.23 MB (0.5% growth), Goroutines: 2->2
PASS ResourceTracker sampling: Heap: 0.23 MB (1.2% growth), Goroutines: 2->3
PASS ResourceTracker I/O recording: Heap: 0.23 MB (0.0% growth), Goroutines: 3->3
```

### Profiling Memory Usage

To generate a detailed memory profile:

```bash
make memory-profile PKG=./pkg/database
```

This creates a profile file in `profiles/mem_<timestamp>.prof` that can be analyzed with `go tool pprof`.

**Analyzing the profile:**
```bash
# Interactive text mode
go tool pprof profiles/mem_20250124_120000.prof

# Web interface (recommended)
go tool pprof -http=:8080 profiles/mem_20250124_120000.prof
```

### Heap Profiling

For detailed heap allocation analysis:

```bash
make memory-heap PKG=./pkg/storage
```

**Analyzing heap profiles:**
```bash
# Show allocated space
go tool pprof -alloc_space profiles/heap_20250124_120000.prof

# Show in-use space
go tool pprof -inuse_space profiles/heap_20250124_120000.prof
```

### Comparing Profiles

To detect memory growth over time:

```bash
# Generate baseline profile
make memory-profile PKG=./pkg/database
mv profiles/mem_*.prof profiles/baseline.prof

# Make code changes...

# Generate new profile
make memory-profile PKG=./pkg/database
mv profiles/mem_*.prof profiles/current.prof

# Compare
./scripts/memory-profile.sh compare profiles/baseline.prof profiles/current.prof
```

### Writing Custom Leak Detection Tests

You can add leak detection to your own tests:

```go
import "github.com/mnohosten/laura-db/pkg/metrics"

func TestMyFeature_NoMemoryLeak(t *testing.T) {
    detector := metrics.NewMemoryLeakDetector(0.15, 2) // 15% threshold
    detector.SetBaseline()

    // Perform operations that should not leak
    for i := 0; i < 1000; i++ {
        // ... your operations ...
    }

    if detector.CheckLeak(t, "MyFeature operations") {
        t.Error("Memory leak detected")
    }
}
```

## Thresholds and Tuning

### Heap Growth Threshold

The heap growth threshold determines acceptable memory increase:

- **Conservative (5-10%)**: For critical, long-running operations
- **Normal (15-20%)**: For typical operations with some allocation
- **Permissive (30%+)**: For operations that intentionally allocate memory

```go
detector := NewMemoryLeakDetector(0.10, 2) // 10% threshold - conservative
```

### Goroutine Buffer

The goroutine buffer allows some variance in goroutine count:

- **Strict (0-1)**: No goroutine leaks allowed
- **Normal (2-3)**: Allow minor runtime variance
- **Permissive (5+)**: For tests with concurrent operations

```go
detector := NewMemoryLeakDetector(0.15, 0) // No goroutine leaks
```

## Best Practices

### 1. Run Tests Regularly

Include memory leak tests in your CI/CD pipeline:

```bash
go test ./pkg/metrics -run TestMemoryLeak -v
```

### 2. Profile Before Optimization

Always profile before optimizing:

```bash
make memory-profile PKG=./pkg/database
```

Use the web UI (`-http=:8080`) to visualize allocations.

### 3. Test Realistic Workloads

Create tests that simulate production usage:

```go
func TestDatabaseWorkload_NoLeak(t *testing.T) {
    detector := NewMemoryLeakDetector(0.20, 3)
    db := database.Open(...)
    defer db.Close()

    detector.SetBaseline()

    // Simulate realistic workload
    for i := 0; i < 10000; i++ {
        db.InsertOne("users", doc)
        db.FindOne("users", filter)
        db.UpdateOne("users", filter, update)
    }

    if detector.CheckLeak(t, "Database workload") {
        t.Error("Leak detected under workload")
    }
}
```

### 4. Clean Up Resources

Always clean up resources in tests:

```go
rt := NewResourceTracker(config)
defer rt.Close() // Ensures cleanup even if test fails
```

### 5. Force GC for Accurate Measurements

The detector automatically runs GC, but you can also manually trigger it:

```go
runtime.GC()
runtime.GC() // Run twice for thorough cleanup
```

### 6. Monitor Long-Running Processes

For servers, use ResourceTracker to monitor memory over time:

```go
tracker := metrics.NewResourceTracker(metrics.DefaultResourceTrackerConfig())
defer tracker.Close()

// Later, check trends
trends := tracker.GetTrends()
if trends["heap_growth_mb"].(float64) > 100 {
    log.Warn("High memory growth detected")
}
```

## Common Memory Leak Patterns

### 1. Goroutine Leaks

**Problem:** Goroutines that never exit
```go
// BAD: No way to stop goroutine
go func() {
    for {
        // work
        time.Sleep(1 * time.Second)
    }
}()
```

**Solution:** Use context or stop channel
```go
// GOOD: Goroutine can be stopped
stopChan := make(chan struct{})
go func() {
    for {
        select {
        case <-stopChan:
            return
        default:
            // work
            time.Sleep(1 * time.Second)
        }
    }
}()
defer close(stopChan)
```

### 2. Slice Growth

**Problem:** Unbounded slice growth
```go
// BAD: Slice grows without limit
var samples []Sample
for {
    samples = append(samples, takeSample())
}
```

**Solution:** Implement size limit
```go
// GOOD: Bounded slice with LRU eviction
if len(samples) >= maxSamples {
    samples = samples[1:] // Remove oldest
}
samples = append(samples, takeSample())
```

### 3. Map Growth

**Problem:** Maps that never shrink
```go
// BAD: Cache grows indefinitely
cache := make(map[string][]byte)
for {
    cache[key] = value
}
```

**Solution:** Implement eviction policy
```go
// GOOD: LRU cache with size limit
cache := NewLRUCache(1000) // Max 1000 entries
cache.Put(key, value) // Evicts oldest if full
```

### 4. Unclosed Resources

**Problem:** Resources not properly closed
```go
// BAD: ResourceTracker not closed
rt := NewResourceTracker(config)
// ... use rt ...
// Never closed - goroutine keeps running
```

**Solution:** Always use defer
```go
// GOOD: Ensures cleanup
rt := NewResourceTracker(config)
defer rt.Close()
```

## Performance Impact

Memory leak detection has minimal overhead:

- **MemorySnapshot**: ~100-200µs per snapshot (includes 2x GC)
- **LeakDetector.CheckLeak**: ~100-200µs per check
- **Tests**: Add ~100-500ms per test (due to GC and sleep)

The overhead is acceptable for testing but should not be used in production hot paths.

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Memory Tests

on: [push, pull_request]

jobs:
  memory-leak-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run memory leak tests
        run: make memory-leak
      - name: Upload profiles
        if: failure()
        uses: actions/upload-artifact@v3
        with:
          name: memory-profiles
          path: profiles/
```

### Pre-commit Hook

Add to `.git/hooks/pre-commit`:

```bash
#!/bin/bash
make memory-leak || {
    echo "Memory leak tests failed!"
    exit 1
}
```

## Troubleshooting

### Tests Fail with "LEAK DETECTED"

1. Check the heap growth percentage
2. If reasonable (<50%), increase threshold
3. If high (>50%), investigate with profiler:
   ```bash
   make memory-profile PKG=./pkg/problematic
   go tool pprof -http=:8080 profiles/mem_*.prof
   ```

### Goroutine Leaks

1. Check goroutine count difference
2. Use `-test.v` to see which test leaked
3. Add `defer` for cleanup functions
4. Use `runtime.NumGoroutine()` to debug

### Inconsistent Results

Memory tests can be flaky due to GC timing:

1. Increase threshold slightly (5-10%)
2. Run tests multiple times
3. Use `-count=5` to detect intermittent issues
4. Profile to understand real memory usage

## See Also

- [Performance Tuning Guide](performance-tuning.md)
- [Benchmarking Guide](benchmarking.md)
- [Resource Tracker Documentation](../pkg/metrics/resource_tracker.go)
- [Go pprof Documentation](https://pkg.go.dev/net/http/pprof)
