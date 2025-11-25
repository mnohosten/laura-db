package regression

import (
	"strings"
	"testing"
	"time"
)

func TestParseBenchmarkResults(t *testing.T) {
	input := `goos: linux
goarch: amd64
pkg: github.com/mnohosten/laura-db/pkg/database
cpu: Intel(R) Core(TM) i7-9700K CPU @ 3.60GHz
BenchmarkInsertOne-8    	  100000	     15234 ns/op	    1024 B/op	      12 allocs/op
BenchmarkFind-8         	   50000	     25678 ns/op	    2048 B/op	      24 allocs/op
BenchmarkIndexLookup-8  	  200000	      8456 ns/op	     512 B/op	       8 allocs/op
PASS
ok  	github.com/mnohosten/laura-db/pkg/database	10.234s
`

	suite, err := ParseBenchmarkResults(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseBenchmarkResults failed: %v", err)
	}

	if len(suite.Results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(suite.Results))
	}

	// Check first result
	r := suite.Results[0]
	if r.Name != "BenchmarkInsertOne-8" {
		t.Errorf("Expected name 'BenchmarkInsertOne-8', got '%s'", r.Name)
	}
	if r.Iterations != 100000 {
		t.Errorf("Expected 100000 iterations, got %d", r.Iterations)
	}
	if r.NsPerOp != 15234 {
		t.Errorf("Expected 15234 ns/op, got %.0f", r.NsPerOp)
	}
	if r.BytesPerOp != 1024 {
		t.Errorf("Expected 1024 B/op, got %d", r.BytesPerOp)
	}
	if r.AllocsPerOp != 12 {
		t.Errorf("Expected 12 allocs/op, got %d", r.AllocsPerOp)
	}

	// Check metadata
	if suite.Metadata["goos"] != "linux" {
		t.Errorf("Expected goos=linux, got %s", suite.Metadata["goos"])
	}
	if suite.Metadata["goarch"] != "amd64" {
		t.Errorf("Expected goarch=amd64, got %s", suite.Metadata["goarch"])
	}

	// Check metadata was copied to results
	if r.OS != "linux" {
		t.Errorf("Expected OS=linux, got %s", r.OS)
	}
	if r.Architecture != "amd64" {
		t.Errorf("Expected Architecture=amd64, got %s", r.Architecture)
	}
}

func TestParseBenchmarkResults_Empty(t *testing.T) {
	input := `PASS
ok  	github.com/mnohosten/laura-db/pkg/database	0.001s
`

	suite, err := ParseBenchmarkResults(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseBenchmarkResults failed: %v", err)
	}

	if len(suite.Results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(suite.Results))
	}
}

func TestDetectRegressions_NoRegression(t *testing.T) {
	baseline := &BenchmarkSuite{
		Results: []*BenchmarkResult{
			{Name: "BenchmarkInsertOne-8", NsPerOp: 10000, BytesPerOp: 1000, AllocsPerOp: 10},
			{Name: "BenchmarkFind-8", NsPerOp: 20000, BytesPerOp: 2000, AllocsPerOp: 20},
		},
	}

	current := &BenchmarkSuite{
		Results: []*BenchmarkResult{
			{Name: "BenchmarkInsertOne-8", NsPerOp: 10500, BytesPerOp: 1050, AllocsPerOp: 11}, // 5% slower
			{Name: "BenchmarkFind-8", NsPerOp: 19000, BytesPerOp: 1900, AllocsPerOp: 19},     // 5% faster
		},
	}

	regressions := DetectRegressions(baseline, current, DefaultThresholds())

	if len(regressions) != 0 {
		t.Errorf("Expected no regressions, got %d", len(regressions))
	}
}

func TestDetectRegressions_Warning(t *testing.T) {
	baseline := &BenchmarkSuite{
		Results: []*BenchmarkResult{
			{Name: "BenchmarkInsertOne-8", NsPerOp: 10000, BytesPerOp: 1000, AllocsPerOp: 10},
		},
	}

	current := &BenchmarkSuite{
		Results: []*BenchmarkResult{
			{Name: "BenchmarkInsertOne-8", NsPerOp: 11500, BytesPerOp: 1200, AllocsPerOp: 12}, // 15% slower time, 20% more memory, 20% more allocs
		},
	}

	regressions := DetectRegressions(baseline, current, DefaultThresholds())

	if len(regressions) != 3 {
		t.Fatalf("Expected 3 regressions (ns/op, B/op, allocs/op), got %d", len(regressions))
	}

	// Check time regression
	if regressions[0].Severity != SeverityWarning {
		t.Errorf("Expected warning severity, got %v", regressions[0].Severity)
	}
	if regressions[0].Metric != "ns/op" {
		t.Errorf("Expected ns/op metric, got %s", regressions[0].Metric)
	}
}

func TestDetectRegressions_Critical(t *testing.T) {
	baseline := &BenchmarkSuite{
		Results: []*BenchmarkResult{
			{Name: "BenchmarkInsertOne-8", NsPerOp: 10000, BytesPerOp: 1000, AllocsPerOp: 10},
		},
	}

	current := &BenchmarkSuite{
		Results: []*BenchmarkResult{
			{Name: "BenchmarkInsertOne-8", NsPerOp: 13000, BytesPerOp: 1400, AllocsPerOp: 13}, // 30% slower
		},
	}

	regressions := DetectRegressions(baseline, current, DefaultThresholds())

	if len(regressions) != 3 {
		t.Fatalf("Expected 3 regressions, got %d", len(regressions))
	}

	// Check time regression is critical
	if regressions[0].Severity != SeverityCritical {
		t.Errorf("Expected critical severity, got %v", regressions[0].Severity)
	}
}

func TestDetectRegressions_NewBenchmark(t *testing.T) {
	baseline := &BenchmarkSuite{
		Results: []*BenchmarkResult{
			{Name: "BenchmarkInsertOne-8", NsPerOp: 10000},
		},
	}

	current := &BenchmarkSuite{
		Results: []*BenchmarkResult{
			{Name: "BenchmarkInsertOne-8", NsPerOp: 10000},
			{Name: "BenchmarkNewFeature-8", NsPerOp: 50000}, // New benchmark
		},
	}

	regressions := DetectRegressions(baseline, current, DefaultThresholds())

	// New benchmarks should not be reported as regressions
	if len(regressions) != 0 {
		t.Errorf("Expected no regressions for new benchmarks, got %d", len(regressions))
	}
}

func TestThresholds(t *testing.T) {
	defaults := DefaultThresholds()
	if defaults.TimeRegressionWarning != 10.0 {
		t.Errorf("Expected default time warning threshold of 10.0, got %.1f", defaults.TimeRegressionWarning)
	}

	strict := StrictThresholds()
	if strict.TimeRegressionWarning != 5.0 {
		t.Errorf("Expected strict time warning threshold of 5.0, got %.1f", strict.TimeRegressionWarning)
	}

	relaxed := RelaxedThresholds()
	if relaxed.TimeRegressionWarning != 20.0 {
		t.Errorf("Expected relaxed time warning threshold of 20.0, got %.1f", relaxed.TimeRegressionWarning)
	}
}

func TestHasCriticalRegressions(t *testing.T) {
	regressions := []*Regression{
		{Severity: SeverityWarning},
		{Severity: SeverityCritical},
		{Severity: SeverityWarning},
	}

	if !HasCriticalRegressions(regressions) {
		t.Error("Expected to find critical regressions")
	}

	regressions = []*Regression{
		{Severity: SeverityWarning},
		{Severity: SeverityWarning},
	}

	if HasCriticalRegressions(regressions) {
		t.Error("Expected no critical regressions")
	}
}

func TestHasWarnings(t *testing.T) {
	regressions := []*Regression{
		{Severity: SeverityWarning},
	}

	if !HasWarnings(regressions) {
		t.Error("Expected to find warnings")
	}

	regressions = []*Regression{}
	if HasWarnings(regressions) {
		t.Error("Expected no warnings")
	}
}

func TestGroupBySeverity(t *testing.T) {
	regressions := []*Regression{
		{Severity: SeverityWarning, BenchmarkName: "Test1"},
		{Severity: SeverityCritical, BenchmarkName: "Test2"},
		{Severity: SeverityWarning, BenchmarkName: "Test3"},
		{Severity: SeverityCritical, BenchmarkName: "Test4"},
		{Severity: SeverityInfo, BenchmarkName: "Test5"},
	}

	groups := GroupBySeverity(regressions)

	if len(groups[SeverityWarning]) != 2 {
		t.Errorf("Expected 2 warnings, got %d", len(groups[SeverityWarning]))
	}
	if len(groups[SeverityCritical]) != 2 {
		t.Errorf("Expected 2 critical, got %d", len(groups[SeverityCritical]))
	}
	if len(groups[SeverityInfo]) != 1 {
		t.Errorf("Expected 1 info, got %d", len(groups[SeverityInfo]))
	}
}

func TestParseBenchmarkResults_WithMBPerSec(t *testing.T) {
	input := `BenchmarkCompression-8    	  10000	    150000 ns/op	  125.50 MB/s	    2048 B/op	      15 allocs/op
`

	suite, err := ParseBenchmarkResults(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseBenchmarkResults failed: %v", err)
	}

	if len(suite.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(suite.Results))
	}

	r := suite.Results[0]
	if r.MBPerSec != 125.50 {
		t.Errorf("Expected 125.50 MB/s, got %.2f", r.MBPerSec)
	}
}

func BenchmarkParseBenchmarkResults(b *testing.B) {
	input := `goos: linux
goarch: amd64
BenchmarkInsertOne-8    	  100000	     15234 ns/op	    1024 B/op	      12 allocs/op
BenchmarkFind-8         	   50000	     25678 ns/op	    2048 B/op	      24 allocs/op
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseBenchmarkResults(strings.NewReader(input))
	}
}

func BenchmarkDetectRegressions(b *testing.B) {
	baseline := &BenchmarkSuite{
		Results: make([]*BenchmarkResult, 100),
	}
	current := &BenchmarkSuite{
		Results: make([]*BenchmarkResult, 100),
	}

	for i := 0; i < 100; i++ {
		baseline.Results[i] = &BenchmarkResult{
			Name:        "BenchmarkTest-8",
			NsPerOp:     float64(10000 + i*100),
			BytesPerOp:  1000,
			AllocsPerOp: 10,
			Timestamp:   time.Now(),
		}
		current.Results[i] = &BenchmarkResult{
			Name:        "BenchmarkTest-8",
			NsPerOp:     float64(11000 + i*100), // 10% slower
			BytesPerOp:  1000,
			AllocsPerOp: 10,
			Timestamp:   time.Now(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DetectRegressions(baseline, current, DefaultThresholds())
	}
}
