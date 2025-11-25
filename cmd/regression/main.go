// LauraDB Performance Regression Testing Tool
//
// This tool provides command-line utilities for performance regression testing,
// including baseline management, regression detection, and historical trend analysis.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mnohosten/laura-db/pkg/regression"
)

const version = "0.1.0"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Define subcommands
	if len(os.Args) < 2 {
		printUsage()
		return nil
	}

	command := os.Args[1]

	switch command {
	case "baseline":
		return runBaseline(os.Args[2:])
	case "check":
		return runCheck(os.Args[2:])
	case "compare":
		return runCompare(os.Args[2:])
	case "trend":
		return runTrend(os.Args[2:])
	case "clean":
		return runClean(os.Args[2:])
	case "version":
		fmt.Printf("LauraDB Regression Tool v%s\n", version)
		return nil
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func runBaseline(args []string) error {
	fs := flag.NewFlagSet("baseline", flag.ExitOnError)
	benchFile := fs.String("file", "", "Benchmark results file (required)")
	dbPath := fs.String("db", "benchmarks", "Database directory")
	commitHash := fs.String("commit", "unknown", "Git commit hash")

	fs.Parse(args)

	if *benchFile == "" {
		return fmt.Errorf("--file is required")
	}

	// Open benchmark file
	f, err := os.Open(*benchFile)
	if err != nil {
		return fmt.Errorf("failed to open benchmark file: %w", err)
	}
	defer f.Close()

	// Parse benchmark results
	suite, err := regression.ParseBenchmarkResults(f)
	if err != nil {
		return fmt.Errorf("failed to parse benchmark results: %w", err)
	}

	// Set commit hash
	for _, result := range suite.Results {
		result.CommitHash = *commitHash
	}

	// Open database
	db, err := regression.NewBenchmarkDatabase(*dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Save as baseline
	if err := db.SaveBaseline(suite); err != nil {
		return fmt.Errorf("failed to save baseline: %w", err)
	}

	// Also save to historical records
	if err := db.SaveHistorical(suite, *commitHash); err != nil {
		return fmt.Errorf("failed to save historical record: %w", err)
	}

	fmt.Printf("✓ Baseline created with %d benchmarks\n", len(suite.Results))
	fmt.Printf("  Database: %s\n", *dbPath)
	fmt.Printf("  Commit: %s\n", *commitHash)

	return nil
}

func runCheck(args []string) error {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	benchFile := fs.String("file", "", "Current benchmark results file (required)")
	dbPath := fs.String("db", "benchmarks", "Database directory")
	thresholdMode := fs.String("threshold", "default", "Threshold mode: default, strict, relaxed")
	format := fs.String("format", "text", "Output format: text, markdown, json")
	failOnCritical := fs.Bool("fail-on-critical", true, "Exit with error code if critical regressions found")
	failOnWarning := fs.Bool("fail-on-warning", false, "Exit with error code if any warnings found")

	fs.Parse(args)

	if *benchFile == "" {
		return fmt.Errorf("--file is required")
	}

	// Open benchmark file
	f, err := os.Open(*benchFile)
	if err != nil {
		return fmt.Errorf("failed to open benchmark file: %w", err)
	}
	defer f.Close()

	// Parse current results
	current, err := regression.ParseBenchmarkResults(f)
	if err != nil {
		return fmt.Errorf("failed to parse benchmark results: %w", err)
	}

	// Open database and load baseline
	db, err := regression.NewBenchmarkDatabase(*dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	baseline, err := db.LoadBaseline()
	if err != nil {
		return fmt.Errorf("failed to load baseline: %w", err)
	}

	// Select thresholds
	var thresholds *regression.Thresholds
	switch strings.ToLower(*thresholdMode) {
	case "default":
		thresholds = regression.DefaultThresholds()
	case "strict":
		thresholds = regression.StrictThresholds()
	case "relaxed":
		thresholds = regression.RelaxedThresholds()
	default:
		return fmt.Errorf("unknown threshold mode: %s", *thresholdMode)
	}

	// Detect regressions
	regressions := regression.DetectRegressions(baseline, current, thresholds)

	// Generate report
	var reportFormat regression.ReportFormat
	switch strings.ToLower(*format) {
	case "text":
		reportFormat = regression.FormatText
	case "markdown":
		reportFormat = regression.FormatMarkdown
	case "json":
		reportFormat = regression.FormatJSON
	default:
		return fmt.Errorf("unknown format: %s", *format)
	}

	if err := regression.GenerateReport(os.Stdout, regressions, reportFormat); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Check exit conditions
	if *failOnCritical && regression.HasCriticalRegressions(regressions) {
		return fmt.Errorf("critical performance regressions detected")
	}

	if *failOnWarning && regression.HasWarnings(regressions) {
		return fmt.Errorf("performance warnings detected")
	}

	return nil
}

func runCompare(args []string) error {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	oldFile := fs.String("old", "", "Old benchmark results file (required)")
	newFile := fs.String("new", "", "New benchmark results file (required)")
	thresholdMode := fs.String("threshold", "default", "Threshold mode: default, strict, relaxed")
	format := fs.String("format", "text", "Output format: text, markdown, json")

	fs.Parse(args)

	if *oldFile == "" || *newFile == "" {
		return fmt.Errorf("both --old and --new are required")
	}

	// Parse old results
	oldF, err := os.Open(*oldFile)
	if err != nil {
		return fmt.Errorf("failed to open old file: %w", err)
	}
	defer oldF.Close()

	oldSuite, err := regression.ParseBenchmarkResults(oldF)
	if err != nil {
		return fmt.Errorf("failed to parse old results: %w", err)
	}

	// Parse new results
	newF, err := os.Open(*newFile)
	if err != nil {
		return fmt.Errorf("failed to open new file: %w", err)
	}
	defer newF.Close()

	newSuite, err := regression.ParseBenchmarkResults(newF)
	if err != nil {
		return fmt.Errorf("failed to parse new results: %w", err)
	}

	// Select thresholds
	var thresholds *regression.Thresholds
	switch strings.ToLower(*thresholdMode) {
	case "default":
		thresholds = regression.DefaultThresholds()
	case "strict":
		thresholds = regression.StrictThresholds()
	case "relaxed":
		thresholds = regression.RelaxedThresholds()
	default:
		return fmt.Errorf("unknown threshold mode: %s", *thresholdMode)
	}

	// Detect regressions
	regressions := regression.DetectRegressions(oldSuite, newSuite, thresholds)

	// Generate report
	var reportFormat regression.ReportFormat
	switch strings.ToLower(*format) {
	case "text":
		reportFormat = regression.FormatText
	case "markdown":
		reportFormat = regression.FormatMarkdown
	case "json":
		reportFormat = regression.FormatJSON
	default:
		return fmt.Errorf("unknown format: %s", *format)
	}

	return regression.GenerateReport(os.Stdout, regressions, reportFormat)
}

func runTrend(args []string) error {
	fs := flag.NewFlagSet("trend", flag.ExitOnError)
	dbPath := fs.String("db", "benchmarks", "Database directory")
	benchmarkName := fs.String("benchmark", "", "Specific benchmark to show trend for (optional)")
	limit := fs.Int("limit", 10, "Number of historical results to include")

	fs.Parse(args)

	// Open database
	db, err := regression.NewBenchmarkDatabase(*dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	var trends map[string][]*regression.BenchmarkResult

	if *benchmarkName != "" {
		// Get trend for specific benchmark
		results, err := db.GetTrend(*benchmarkName, *limit)
		if err != nil {
			return fmt.Errorf("failed to get trend: %w", err)
		}
		trends = map[string][]*regression.BenchmarkResult{
			*benchmarkName: results,
		}
	} else {
		// Get trends for all benchmarks
		var err error
		trends, err = db.GetAllTrends(*limit)
		if err != nil {
			return fmt.Errorf("failed to get trends: %w", err)
		}
	}

	return regression.GenerateTrendReport(os.Stdout, trends)
}

func runClean(args []string) error {
	fs := flag.NewFlagSet("clean", flag.ExitOnError)
	dbPath := fs.String("db", "benchmarks", "Database directory")
	olderThanDays := fs.Int("older-than", 30, "Remove results older than N days")

	fs.Parse(args)

	// Open database
	db, err := regression.NewBenchmarkDatabase(*dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Clean old results
	olderThan := time.Duration(*olderThanDays) * 24 * time.Hour
	count, err := db.CleanOldResults(olderThan)
	if err != nil {
		return fmt.Errorf("failed to clean old results: %w", err)
	}

	fmt.Printf("✓ Cleaned %d old benchmark result(s)\n", count)

	return nil
}

func printUsage() {
	fmt.Println(`LauraDB Performance Regression Testing Tool

USAGE:
    laura-regression <command> [options]

COMMANDS:
    baseline    Create a new performance baseline
    check       Check current results against baseline
    compare     Compare two benchmark result files
    trend       Show performance trends over time
    clean       Clean old benchmark results
    version     Show version information
    help        Show this help message

BASELINE - Create a performance baseline:
    laura-regression baseline --file <benchmark-file> [options]

    Options:
      --file <path>     Benchmark results file (required)
      --db <path>       Database directory (default: benchmarks)
      --commit <hash>   Git commit hash (default: unknown)

    Example:
      laura-regression baseline --file bench.txt --commit abc123

CHECK - Check for performance regressions:
    laura-regression check --file <benchmark-file> [options]

    Options:
      --file <path>           Current benchmark results (required)
      --db <path>             Database directory (default: benchmarks)
      --threshold <mode>      Threshold mode: default, strict, relaxed (default: default)
      --format <fmt>          Output format: text, markdown, json (default: text)
      --fail-on-critical      Exit with error if critical regressions found (default: true)
      --fail-on-warning       Exit with error if any warnings found (default: false)

    Example:
      laura-regression check --file current.txt --format markdown

COMPARE - Compare two benchmark files:
    laura-regression compare --old <file> --new <file> [options]

    Options:
      --old <path>            Old benchmark results (required)
      --new <path>            New benchmark results (required)
      --threshold <mode>      Threshold mode: default, strict, relaxed (default: default)
      --format <fmt>          Output format: text, markdown, json (default: text)

    Example:
      laura-regression compare --old baseline.txt --new current.txt

TREND - Show performance trends:
    laura-regression trend [options]

    Options:
      --db <path>             Database directory (default: benchmarks)
      --benchmark <name>      Specific benchmark name (optional)
      --limit <n>             Number of historical results (default: 10)

    Example:
      laura-regression trend --benchmark BenchmarkInsertOne-8 --limit 20

CLEAN - Clean old results:
    laura-regression clean [options]

    Options:
      --db <path>             Database directory (default: benchmarks)
      --older-than <days>     Remove results older than N days (default: 30)

    Example:
      laura-regression clean --older-than 60

THRESHOLDS:
    default:  10% warning, 25% critical
    strict:   5% warning, 15% critical
    relaxed:  20% warning, 50% critical

EXAMPLES:

    # Create a baseline from current benchmarks
    go test -bench=. -benchmem ./pkg/... > baseline.txt
    laura-regression baseline --file baseline.txt

    # Check current results against baseline
    go test -bench=. -benchmem ./pkg/... > current.txt
    laura-regression check --file current.txt

    # Compare two specific files
    laura-regression compare --old old.txt --new new.txt --format markdown

    # Show trend for a specific benchmark
    laura-regression trend --benchmark BenchmarkInsertOne-8

    # Clean results older than 60 days
    laura-regression clean --older-than 60

For more information, visit: https://github.com/mnohosten/laura-db
`)
}
