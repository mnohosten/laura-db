# Performance Regression Testing

LauraDB includes a comprehensive performance regression testing system to detect and prevent performance degradations before they reach production. The system consists of:

1. **Benchmark Database** - Historical storage of benchmark results
2. **Regression Detection** - Automated comparison and analysis
3. **CI/CD Integration** - Automatic regression checks in pull requests
4. **CLI Tool** - Manual regression testing and trend analysis

## Table of Contents

- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [CLI Tool Usage](#cli-tool-usage)
- [CI/CD Integration](#cicd-integration)
- [Thresholds and Configuration](#thresholds-and-configuration)
- [Interpreting Results](#interpreting-results)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Quick Start

### Creating a Baseline

The baseline represents your "known good" performance that future benchmarks will be compared against:

```bash
# Run benchmarks and save results
make bench-all > bench.txt

# Create baseline
./bin/laura-regression baseline --file bench.txt --commit $(git rev-parse --short HEAD)

# Verify baseline was created
ls benchmarks/
# Output: baseline.json  bench-20250124-abc123.json
```

### Checking for Regressions

After making changes, check if there are any performance regressions:

```bash
# Run current benchmarks
make bench-all > current.txt

# Check against baseline
./bin/laura-regression check --file current.txt
```

**Example Output:**

```
═══════════════════════════════════════════════════════════════════════
                    PERFORMANCE REGRESSION REPORT
═══════════════════════════════════════════════════════════════════════

Generated: 2025-01-24 15:30:45
Total Regressions: 2
  Critical: 1
  Warning:  1

───────────────────────────────────────────────────────────────────────
CRITICAL REGRESSIONS (> 25% slower)
───────────────────────────────────────────────────────────────────────

Benchmark                                     Metric    Baseline   Current    Change
─────────                                     ──────    ────────   ───────    ──────
BenchmarkInsertOne-8                          ns/op     10000      13500      +35.0%

───────────────────────────────────────────────────────────────────────
WARNINGS (10-25% slower)
───────────────────────────────────────────────────────────────────────

Benchmark                                     Metric    Baseline   Current    Change
─────────                                     ──────    ────────   ───────    ──────
BenchmarkFind-8                               ns/op     25000      28000      +12.0%

═══════════════════════════════════════════════════════════════════════
```

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────────┐
│                  Performance Regression System               │
└─────────────────────────────────────────────────────────────┘
                              │
           ┌──────────────────┼──────────────────┐
           │                  │                  │
           ▼                  ▼                  ▼
    ┌──────────┐       ┌──────────┐      ┌──────────┐
    │ Benchmark│       │Regression│      │   CLI    │
    │ Database │◄──────┤ Detection│◄─────┤   Tool   │
    │          │       │  Engine  │      │          │
    └──────────┘       └──────────┘      └──────────┘
         │                    │                 │
         │                    │                 │
         ▼                    ▼                 ▼
    Historical           Threshold         Manual
     Storage             Analysis         Testing
```

### Benchmark Database

The benchmark database (`benchmarks/` directory) stores:

- **`baseline.json`** - Current baseline benchmark results
- **`bench-TIMESTAMP-COMMIT.json`** - Historical benchmark snapshots
- **`baseline-backup-*.json`** - Automatic backups when baseline is updated

**Storage Format:** JSON with full benchmark metadata including:
- Benchmark names and results (ns/op, B/op, allocs/op)
- Git commit hashes
- Timestamps
- Go version, OS, architecture

### Regression Detection

The detection engine compares benchmarks using statistical analysis:

1. **Parsing** - Extract benchmark metrics from Go test output
2. **Normalization** - Handle different benchmark formats and scales
3. **Comparison** - Calculate percentage changes for all metrics
4. **Classification** - Categorize regressions by severity:
   - **Info**: < 10% degradation
   - **Warning**: 10-25% degradation
   - **Critical**: > 25% degradation
5. **Reporting** - Generate human-readable and machine-parseable reports

## CLI Tool Usage

The `laura-regression` CLI tool provides comprehensive regression testing capabilities.

### Commands Overview

| Command | Description | Use Case |
|---------|-------------|----------|
| `baseline` | Create a new baseline | Initial setup, after major releases |
| `check` | Check for regressions | Development, pre-commit validation |
| `compare` | Compare two benchmark files | Historical analysis, A/B testing |
| `trend` | Show performance trends | Identifying gradual degradation |
| `clean` | Remove old results | Disk space management |

### 1. Baseline Management

#### Create Baseline

```bash
laura-regression baseline --file bench.txt [options]
```

**Options:**
- `--file <path>` - Benchmark results file (required)
- `--db <path>` - Database directory (default: `benchmarks`)
- `--commit <hash>` - Git commit hash (default: `unknown`)

**Example:**
```bash
# Create baseline with git commit
go test -bench=. -benchmem ./pkg/... > bench.txt
laura-regression baseline --file bench.txt --commit $(git rev-parse --short HEAD)
```

**Output:**
```
✓ Baseline created with 156 benchmarks
  Database: benchmarks
  Commit: abc1234
```

### 2. Regression Checking

#### Check Against Baseline

```bash
laura-regression check --file current.txt [options]
```

**Options:**
- `--file <path>` - Current benchmark results (required)
- `--db <path>` - Database directory (default: `benchmarks`)
- `--threshold <mode>` - Threshold mode: `default`, `strict`, `relaxed`
- `--format <fmt>` - Output format: `text`, `markdown`, `json`
- `--fail-on-critical` - Exit with error if critical regressions found (default: true)
- `--fail-on-warning` - Exit with error if any warnings found (default: false)

**Examples:**

```bash
# Basic regression check
laura-regression check --file current.txt

# Use strict thresholds (5%/15%)
laura-regression check --file current.txt --threshold strict

# Generate markdown report (for GitHub comments)
laura-regression check --file current.txt --format markdown

# Fail on any warning (for strict CI)
laura-regression check --file current.txt --fail-on-warning
```

### 3. Comparing Benchmarks

#### Compare Two Files

```bash
laura-regression compare --old old.txt --new new.txt [options]
```

**Options:**
- `--old <path>` - Old benchmark results (required)
- `--new <path>` - New benchmark results (required)
- `--threshold <mode>` - Threshold mode (default: `default`)
- `--format <fmt>` - Output format (default: `text`)

**Example:**
```bash
# Compare feature branch against main
git checkout main
make bench-all > main.txt
git checkout feature-branch
make bench-all > feature.txt
laura-regression compare --old main.txt --new feature.txt
```

### 4. Trend Analysis

#### View Performance Trends

```bash
laura-regression trend [options]
```

**Options:**
- `--db <path>` - Database directory (default: `benchmarks`)
- `--benchmark <name>` - Specific benchmark name (optional)
- `--limit <n>` - Number of historical results (default: 10)

**Examples:**

```bash
# Show trends for all benchmarks (last 10 runs)
laura-regression trend

# Show trend for specific benchmark
laura-regression trend --benchmark BenchmarkInsertOne-8 --limit 20

# Get detailed historical view
laura-regression trend --limit 50 > trends.txt
```

**Example Output:**
```
═══════════════════════════════════════════════════════════════════════
                    PERFORMANCE TREND REPORT
═══════════════════════════════════════════════════════════════════════

BenchmarkInsertOne-8
──────────────────────────────────────────────────────────────────────
Timestamp           ns/op     B/op    allocs/op
2025-01-24 15:00    10234     1024    12
2025-01-23 14:30    10156     1024    12
2025-01-22 13:15    10089     1024    12
2025-01-21 16:45    10123     1024    12
...
```

### 5. Cleanup

#### Remove Old Results

```bash
laura-regression clean [options]
```

**Options:**
- `--db <path>` - Database directory (default: `benchmarks`)
- `--older-than <days>` - Remove results older than N days (default: 30)

**Example:**
```bash
# Clean results older than 60 days
laura-regression clean --older-than 60
```

## CI/CD Integration

### GitHub Actions

LauraDB includes automatic regression detection in CI/CD via GitHub Actions.

#### Pull Request Flow

1. **Trigger**: When a PR is opened or updated
2. **Benchmark**: Run full benchmark suite (5 iterations)
3. **Download Baseline**: Fetch latest baseline from `main` branch artifacts
4. **Detect Regressions**: Compare PR benchmarks against baseline
5. **Comment**: Post regression report as PR comment
6. **Status**: Fail PR if critical regressions detected

#### Baseline Update Flow

1. **Trigger**: When code is merged to `main`
2. **Benchmark**: Run full benchmark suite (5 iterations for stability)
3. **Update Baseline**: Save as new baseline
4. **Store Historical**: Add to historical database
5. **Upload Artifacts**: Store baseline and database for future comparisons

### Configuration

The workflow is defined in `.github/workflows/benchmarks.yml`:

```yaml
on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM UTC
```

### Manual Trigger

You can manually trigger regression checks:

1. Go to **Actions** tab in GitHub
2. Select **Performance Benchmarks** workflow
3. Click **Run workflow**
4. Select branch and click **Run**

## Thresholds and Configuration

### Threshold Modes

LauraDB provides three pre-configured threshold modes:

| Mode | Warning Threshold | Critical Threshold | Use Case |
|------|-------------------|-------------------|----------|
| **default** | 10% | 25% | General development |
| **strict** | 5% | 15% | Performance-critical code |
| **relaxed** | 20% | 50% | Experimental features |

### Metrics Tracked

For each benchmark, the following metrics are monitored:

1. **Time (ns/op)** - Nanoseconds per operation
   - Most important metric
   - Directly impacts user experience

2. **Memory (B/op)** - Bytes allocated per operation
   - Impacts memory pressure and GC frequency
   - Can affect overall system performance

3. **Allocations (allocs/op)** - Number of allocations per operation
   - More allocations = more GC pressure
   - Micro-benchmark quality indicator

### Custom Thresholds

For advanced use cases, you can customize thresholds in code:

```go
thresholds := &regression.Thresholds{
    TimeRegressionWarning:    8.0,   // 8% time degradation
    TimeRegressionCritical:   20.0,  // 20% time degradation
    MemoryRegressionWarning:  10.0,  // 10% more memory
    MemoryRegressionCritical: 25.0,  // 25% more memory
    AllocRegressionWarning:   10.0,  // 10% more allocations
    AllocRegressionCritical:  25.0,  // 25% more allocations
}
```

## Interpreting Results

### Understanding Severity Levels

#### ✅ No Regressions

```
✓ No performance regressions detected
```

**Action**: None required. Performance is stable or improved.

#### ⚠️ Warning (10-25% degradation)

```
WARNINGS (10-25% slower)
BenchmarkFind-8   ns/op   25000   28000   +12.0%
```

**Action**:
- Review recent changes
- Consider if trade-off is justified (e.g., added features)
- Document performance impact in commit message
- Monitor in future benchmarks

#### ❌ Critical (>25% degradation)

```
CRITICAL REGRESSIONS (> 25% slower)
BenchmarkInsertOne-8   ns/op   10000   13500   +35.0%
```

**Action**:
- **Do not merge** until resolved
- Profile the code to identify bottleneck
- Investigate algorithm changes, unnecessary allocations
- Consider reverting changes if cause is unclear

### Common Regression Causes

1. **Algorithm Complexity**
   - Changed O(1) operation to O(n)
   - Added nested loops
   - **Fix**: Use more efficient algorithms

2. **Excessive Allocations**
   - Creating temporary objects in hot paths
   - String concatenation in loops
   - **Fix**: Reuse buffers, use `strings.Builder`

3. **Lock Contention**
   - Added locks in concurrent code
   - Holding locks too long
   - **Fix**: Reduce critical sections, use RWMutex

4. **I/O Operations**
   - Added synchronous I/O in hot path
   - Increased disk/network calls
   - **Fix**: Batch operations, use caching

5. **Reflection/Interface{}**
   - Using reflection instead of type-specific code
   - Type assertions in loops
   - **Fix**: Use generics or code generation

## Best Practices

### For Developers

1. **Run Benchmarks Locally**
   ```bash
   # Before committing changes
   make bench-all > before.txt
   # Make your changes
   make bench-all > after.txt
   laura-regression compare --old before.txt --new after.txt
   ```

2. **Baseline Updates**
   - Update baseline after major releases
   - Never update baseline to hide regressions
   - Document why baseline changed

3. **Profile Before Optimizing**
   ```bash
   # Generate CPU profile
   go test -bench=BenchmarkInsertOne -cpuprofile=cpu.prof ./pkg/database

   # Analyze profile
   go tool pprof cpu.prof
   ```

4. **Interpret with Context**
   - Small regressions (<5%) might be noise
   - Check if hardware changed (CI runners)
   - Look at trends over multiple runs

### For Teams

1. **Establish Baseline Policy**
   - Define when baseline updates are allowed
   - Require performance review for critical paths
   - Track baseline history

2. **Review Process**
   - Include performance in code review checklist
   - Require explanation for warnings
   - Block merges on critical regressions

3. **Monitoring**
   - Run daily scheduled benchmarks
   - Track long-term trends
   - Alert on gradual degradation

4. **Documentation**
   - Document expected performance characteristics
   - Note any trade-offs made
   - Keep performance requirements updated

## Troubleshooting

### Issue: Noisy Benchmarks

**Symptoms**: Benchmark results vary significantly between runs

**Solutions**:
```bash
# Increase benchmark time and iterations
go test -bench=. -benchtime=5s -count=10

# Use stricter CPU affinity (Linux)
taskset -c 0 go test -bench=.

# Disable CPU frequency scaling
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor
```

### Issue: False Positives

**Symptoms**: Regressions detected but code hasn't changed

**Solutions**:
- Use `relaxed` thresholds for initial development
- Increase benchmark stability (more iterations)
- Check for system changes (OS updates, other processes)
- Regenerate baseline on same hardware

### Issue: Baseline Not Found

**Symptoms**: `failed to load baseline: no such file`

**Solutions**:
```bash
# Create initial baseline
make bench-all > bench.txt
laura-regression baseline --file bench.txt

# Or in CI, check if baseline artifact exists
# First PR will create baseline automatically
```

### Issue: Out of Disk Space

**Symptoms**: Benchmark database growing too large

**Solutions**:
```bash
# Clean old results
laura-regression clean --older-than 30

# Or set automatic cleanup in CI
laura-regression clean --older-than 90  # Keep 90 days
```

### Issue: Benchmark Timeout

**Symptoms**: Benchmarks take too long in CI

**Solutions**:
- Reduce `-benchtime` (default: 3s)
- Reduce `-count` (default: 5)
- Run subset of benchmarks:
  ```bash
  go test -bench=BenchmarkInsert.* ./pkg/database
  ```

## Advanced Usage

### Integrating with Monitoring

Export metrics to monitoring systems:

```bash
# Generate JSON for processing
laura-regression check --file current.txt --format json > metrics.json

# Parse and send to monitoring
jq '.regressions[] | {name: .benchmark, change: .percent_change}' metrics.json | \
  send-to-datadog
```

### Automated Performance Gates

Add to pre-commit hook:

```bash
#!/bin/bash
# .git/hooks/pre-commit

if ! make bench-all > /tmp/bench.txt; then
    echo "Benchmarks failed!"
    exit 1
fi

if laura-regression check --file /tmp/bench.txt --threshold strict 2>&1 | grep -q "CRITICAL"; then
    echo "Critical performance regression detected!"
    echo "Run: laura-regression check --file /tmp/bench.txt"
    exit 1
fi
```

### Custom Reports

Generate custom reports using the Go API:

```go
package main

import (
    "os"
    "github.com/mnohosten/laura-db/pkg/regression"
)

func main() {
    // Parse benchmarks
    f, _ := os.Open("bench.txt")
    suite, _ := regression.ParseBenchmarkResults(f)

    // Load baseline
    db, _ := regression.NewBenchmarkDatabase("benchmarks")
    baseline, _ := db.LoadBaseline()

    // Detect regressions
    regressions := regression.DetectRegressions(baseline, suite, regression.StrictThresholds())

    // Custom processing
    for _, r := range regressions {
        if r.Severity == regression.SeverityCritical {
            // Send alert, update dashboard, etc.
        }
    }
}
```

## See Also

- [Benchmarking Guide](benchmarking.md) - How to write effective benchmarks
- [Performance Tuning](performance-tuning.md) - Optimizing LauraDB performance
- [GitHub Actions Documentation](.github/workflows/benchmarks.yml) - CI/CD workflow details
