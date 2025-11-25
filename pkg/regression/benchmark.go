// Package regression provides performance regression testing capabilities for LauraDB.
//
// It parses benchmark results, stores historical data, detects regressions, and generates reports.
package regression

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// BenchmarkResult represents a single benchmark measurement
type BenchmarkResult struct {
	Name         string    // Benchmark name (e.g., "BenchmarkInsertOne-8")
	Iterations   int64     // Number of iterations
	NsPerOp      float64   // Nanoseconds per operation
	BytesPerOp   int64     // Bytes allocated per operation
	AllocsPerOp  int64     // Allocations per operation
	MBPerSec     float64   // MB/s throughput (if applicable)
	Timestamp    time.Time // When benchmark was run
	CommitHash   string    // Git commit hash
	BranchName   string    // Git branch name
	GoVersion    string    // Go version used
	OS           string    // Operating system
	Architecture string    // CPU architecture
}

// BenchmarkSuite represents a collection of benchmark results
type BenchmarkSuite struct {
	Results   []*BenchmarkResult
	Metadata  map[string]string
	Timestamp time.Time
}

// Regression represents a detected performance regression
type Regression struct {
	BenchmarkName string
	Baseline      *BenchmarkResult
	Current       *BenchmarkResult
	PercentChange float64
	Metric        string // "ns/op", "B/op", "allocs/op"
	Severity      Severity
}

// Severity indicates how severe a regression is
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityCritical
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARNING"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// ParseBenchmarkResults parses Go benchmark output
//
// Example format:
// BenchmarkInsertOne-8    100000   15234 ns/op   1024 B/op   12 allocs/op
func ParseBenchmarkResults(r io.Reader) (*BenchmarkSuite, error) {
	suite := &BenchmarkSuite{
		Results:   make([]*BenchmarkResult, 0),
		Metadata:  make(map[string]string),
		Timestamp: time.Now(),
	}

	scanner := bufio.NewScanner(r)
	// Regex to match benchmark lines
	// Format: BenchmarkName-N   iterations   ns/op   [B/op   allocs/op]   [MB/s]
	benchRegex := regexp.MustCompile(`^(Benchmark\S+)\s+(\d+)\s+([\d.]+)\s+ns/op(?:\s+([\d.]+)\s+B/op)?(?:\s+([\d.]+)\s+allocs/op)?(?:\s+([\d.]+)\s+MB/s)?`)

	for scanner.Scan() {
		line := scanner.Text()

		// Try to match benchmark result
		matches := benchRegex.FindStringSubmatch(line)
		if matches != nil {
			result := &BenchmarkResult{
				Name:      matches[1],
				Timestamp: suite.Timestamp,
			}

			// Parse iterations
			if val, err := strconv.ParseInt(matches[2], 10, 64); err == nil {
				result.Iterations = val
			}

			// Parse ns/op
			if val, err := strconv.ParseFloat(matches[3], 64); err == nil {
				result.NsPerOp = val
			}

			// Parse B/op (optional)
			if matches[4] != "" {
				if val, err := strconv.ParseFloat(matches[4], 64); err == nil {
					result.BytesPerOp = int64(val)
				}
			}

			// Parse allocs/op (optional)
			if matches[5] != "" {
				if val, err := strconv.ParseFloat(matches[5], 64); err == nil {
					result.AllocsPerOp = int64(val)
				}
			}

			// Parse MB/s (optional)
			if matches[6] != "" {
				if val, err := strconv.ParseFloat(matches[6], 64); err == nil {
					result.MBPerSec = val
				}
			}

			suite.Results = append(suite.Results, result)
		} else {
			// Check for metadata lines (e.g., "goos: linux", "goarch: amd64")
			if strings.Contains(line, ":") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					suite.Metadata[key] = value
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading benchmark results: %w", err)
	}

	// Extract metadata into results
	for _, result := range suite.Results {
		if goVersion, ok := suite.Metadata["go"]; ok {
			result.GoVersion = goVersion
		}
		if os, ok := suite.Metadata["goos"]; ok {
			result.OS = os
		}
		if arch, ok := suite.Metadata["goarch"]; ok {
			result.Architecture = arch
		}
	}

	return suite, nil
}

// DetectRegressions compares two benchmark suites and identifies regressions
//
// Parameters:
//   - baseline: Historical baseline benchmark results
//   - current: Current benchmark results to compare
//   - thresholds: Thresholds for determining severity
func DetectRegressions(baseline, current *BenchmarkSuite, thresholds *Thresholds) []*Regression {
	if thresholds == nil {
		thresholds = DefaultThresholds()
	}

	regressions := make([]*Regression, 0)

	// Create a map of baseline results for quick lookup
	baselineMap := make(map[string]*BenchmarkResult)
	for _, result := range baseline.Results {
		baselineMap[result.Name] = result
	}

	// Compare each current result with baseline
	for _, curr := range current.Results {
		base, exists := baselineMap[curr.Name]
		if !exists {
			// New benchmark, skip (not a regression)
			continue
		}

		// Check ns/op regression
		if base.NsPerOp > 0 {
			percentChange := ((curr.NsPerOp - base.NsPerOp) / base.NsPerOp) * 100
			if percentChange > thresholds.TimeRegressionWarning {
				severity := SeverityWarning
				if percentChange > thresholds.TimeRegressionCritical {
					severity = SeverityCritical
				}

				regressions = append(regressions, &Regression{
					BenchmarkName: curr.Name,
					Baseline:      base,
					Current:       curr,
					PercentChange: percentChange,
					Metric:        "ns/op",
					Severity:      severity,
				})
			}
		}

		// Check memory regression (B/op)
		if base.BytesPerOp > 0 {
			percentChange := float64((curr.BytesPerOp-base.BytesPerOp)*100) / float64(base.BytesPerOp)
			if percentChange > thresholds.MemoryRegressionWarning {
				severity := SeverityWarning
				if percentChange > thresholds.MemoryRegressionCritical {
					severity = SeverityCritical
				}

				regressions = append(regressions, &Regression{
					BenchmarkName: curr.Name,
					Baseline:      base,
					Current:       curr,
					PercentChange: percentChange,
					Metric:        "B/op",
					Severity:      severity,
				})
			}
		}

		// Check allocation regression (allocs/op)
		if base.AllocsPerOp > 0 {
			percentChange := float64((curr.AllocsPerOp-base.AllocsPerOp)*100) / float64(base.AllocsPerOp)
			if percentChange > thresholds.AllocRegressionWarning {
				severity := SeverityWarning
				if percentChange > thresholds.AllocRegressionCritical {
					severity = SeverityCritical
				}

				regressions = append(regressions, &Regression{
					BenchmarkName: curr.Name,
					Baseline:      base,
					Current:       curr,
					PercentChange: percentChange,
					Metric:        "allocs/op",
					Severity:      severity,
				})
			}
		}
	}

	return regressions
}

// Thresholds defines the percentage thresholds for regression detection
type Thresholds struct {
	// Time (ns/op) thresholds
	TimeRegressionWarning  float64 // e.g., 10.0 means 10% slower is a warning
	TimeRegressionCritical float64 // e.g., 25.0 means 25% slower is critical

	// Memory (B/op) thresholds
	MemoryRegressionWarning  float64
	MemoryRegressionCritical float64

	// Allocations (allocs/op) thresholds
	AllocRegressionWarning  float64
	AllocRegressionCritical float64
}

// DefaultThresholds returns sensible default thresholds
func DefaultThresholds() *Thresholds {
	return &Thresholds{
		TimeRegressionWarning:    10.0,  // 10% slower
		TimeRegressionCritical:   25.0,  // 25% slower
		MemoryRegressionWarning:  15.0,  // 15% more memory
		MemoryRegressionCritical: 30.0,  // 30% more memory
		AllocRegressionWarning:   10.0,  // 10% more allocations
		AllocRegressionCritical:  25.0,  // 25% more allocations
	}
}

// StrictThresholds returns stricter thresholds for critical code paths
func StrictThresholds() *Thresholds {
	return &Thresholds{
		TimeRegressionWarning:    5.0,
		TimeRegressionCritical:   15.0,
		MemoryRegressionWarning:  5.0,
		MemoryRegressionCritical: 15.0,
		AllocRegressionWarning:   5.0,
		AllocRegressionCritical:  15.0,
	}
}

// RelaxedThresholds returns more lenient thresholds
func RelaxedThresholds() *Thresholds {
	return &Thresholds{
		TimeRegressionWarning:    20.0,
		TimeRegressionCritical:   50.0,
		MemoryRegressionWarning:  25.0,
		MemoryRegressionCritical: 50.0,
		AllocRegressionWarning:   20.0,
		AllocRegressionCritical:  50.0,
	}
}

// HasCriticalRegressions returns true if any critical regressions exist
func HasCriticalRegressions(regressions []*Regression) bool {
	for _, r := range regressions {
		if r.Severity == SeverityCritical {
			return true
		}
	}
	return false
}

// HasWarnings returns true if any warnings exist
func HasWarnings(regressions []*Regression) bool {
	for _, r := range regressions {
		if r.Severity == SeverityWarning || r.Severity == SeverityCritical {
			return true
		}
	}
	return false
}

// GroupBySeverity groups regressions by severity
func GroupBySeverity(regressions []*Regression) map[Severity][]*Regression {
	groups := map[Severity][]*Regression{
		SeverityInfo:     make([]*Regression, 0),
		SeverityWarning:  make([]*Regression, 0),
		SeverityCritical: make([]*Regression, 0),
	}

	for _, r := range regressions {
		groups[r.Severity] = append(groups[r.Severity], r)
	}

	return groups
}
