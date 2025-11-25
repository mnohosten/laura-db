package regression

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BenchmarkDatabase stores historical benchmark results
type BenchmarkDatabase struct {
	basePath string
}

// NewBenchmarkDatabase creates a new benchmark database
func NewBenchmarkDatabase(basePath string) (*BenchmarkDatabase, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	return &BenchmarkDatabase{
		basePath: basePath,
	}, nil
}

// SaveBaseline stores a benchmark suite as the baseline
func (db *BenchmarkDatabase) SaveBaseline(suite *BenchmarkSuite) error {
	baselinePath := filepath.Join(db.basePath, "baseline.json")

	// Backup existing baseline
	if _, err := os.Stat(baselinePath); err == nil {
		backupPath := filepath.Join(db.basePath, fmt.Sprintf("baseline-backup-%s.json", time.Now().Format("20060102-150405")))
		if err := os.Rename(baselinePath, backupPath); err != nil {
			return fmt.Errorf("failed to backup existing baseline: %w", err)
		}
	}

	return db.saveSuite(baselinePath, suite)
}

// LoadBaseline loads the baseline benchmark suite
func (db *BenchmarkDatabase) LoadBaseline() (*BenchmarkSuite, error) {
	baselinePath := filepath.Join(db.basePath, "baseline.json")
	return db.loadSuite(baselinePath)
}

// SaveHistorical stores a benchmark suite in historical records
func (db *BenchmarkDatabase) SaveHistorical(suite *BenchmarkSuite, commitHash string) error {
	timestamp := suite.Timestamp.Format("20060102-150405")
	filename := fmt.Sprintf("bench-%s-%s.json", timestamp, commitHash)
	histPath := filepath.Join(db.basePath, filename)

	return db.saveSuite(histPath, suite)
}

// LoadHistorical loads a specific historical benchmark suite
func (db *BenchmarkDatabase) LoadHistorical(filename string) (*BenchmarkSuite, error) {
	histPath := filepath.Join(db.basePath, filename)
	return db.loadSuite(histPath)
}

// ListHistorical returns all historical benchmark filenames, sorted by date (newest first)
func (db *BenchmarkDatabase) ListHistorical() ([]string, error) {
	entries, err := os.ReadDir(db.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read database directory: %w", err)
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Include historical bench files but exclude baseline
		if filepath.Ext(name) == ".json" && name != "baseline.json" && !strings.HasPrefix(filepath.Base(name), "baseline") {
			files = append(files, name)
		}
	}

	// Sort by filename (which includes timestamp)
	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	return files, nil
}

// CleanOldResults removes historical results older than the specified duration
func (db *BenchmarkDatabase) CleanOldResults(olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)
	entries, err := os.ReadDir(db.basePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read database directory: %w", err)
	}

	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Only clean historical bench files
		if filepath.Ext(name) == ".json" && name != "baseline.json" && !strings.HasPrefix(filepath.Base(name), "baseline") {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			if info.ModTime().Before(cutoff) {
				filePath := filepath.Join(db.basePath, name)
				if err := os.Remove(filePath); err != nil {
					return removed, fmt.Errorf("failed to remove old result %s: %w", name, err)
				}
				removed++
			}
		}
	}

	return removed, nil
}

// GetTrend returns benchmark trends over time for a specific benchmark
func (db *BenchmarkDatabase) GetTrend(benchmarkName string, limit int) ([]*BenchmarkResult, error) {
	files, err := db.ListHistorical()
	if err != nil {
		return nil, err
	}

	results := make([]*BenchmarkResult, 0)

	// Load each historical file and extract the specific benchmark
	for i, filename := range files {
		if limit > 0 && i >= limit {
			break
		}

		suite, err := db.LoadHistorical(filename)
		if err != nil {
			continue // Skip files that can't be loaded
		}

		// Find the benchmark in this suite
		for _, result := range suite.Results {
			if result.Name == benchmarkName {
				results = append(results, result)
				break
			}
		}
	}

	return results, nil
}

// GetAllTrends returns trends for all benchmarks
func (db *BenchmarkDatabase) GetAllTrends(limit int) (map[string][]*BenchmarkResult, error) {
	files, err := db.ListHistorical()
	if err != nil {
		return nil, err
	}

	trends := make(map[string][]*BenchmarkResult)

	for i, filename := range files {
		if limit > 0 && i >= limit {
			break
		}

		suite, err := db.LoadHistorical(filename)
		if err != nil {
			continue
		}

		for _, result := range suite.Results {
			trends[result.Name] = append(trends[result.Name], result)
		}
	}

	return trends, nil
}

// saveSuite saves a benchmark suite to a file
func (db *BenchmarkDatabase) saveSuite(path string, suite *BenchmarkSuite) error {
	data, err := json.MarshalIndent(suite, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark suite: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write benchmark suite: %w", err)
	}

	return nil
}

// loadSuite loads a benchmark suite from a file
func (db *BenchmarkDatabase) loadSuite(path string) (*BenchmarkSuite, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read benchmark suite: %w", err)
	}

	suite := &BenchmarkSuite{}
	if err := json.Unmarshal(data, suite); err != nil {
		return nil, fmt.Errorf("failed to unmarshal benchmark suite: %w", err)
	}

	return suite, nil
}

// CompareWithHistory compares current results with recent history
func (db *BenchmarkDatabase) CompareWithHistory(current *BenchmarkSuite, historyCount int, thresholds *Thresholds) (map[string]*HistoricalComparison, error) {
	files, err := db.ListHistorical()
	if err != nil {
		return nil, err
	}

	if historyCount > len(files) {
		historyCount = len(files)
	}

	comparisons := make(map[string]*HistoricalComparison)

	// Initialize comparisons for each benchmark in current suite
	for _, result := range current.Results {
		comparisons[result.Name] = &HistoricalComparison{
			BenchmarkName: result.Name,
			Current:       result,
			History:       make([]*BenchmarkResult, 0),
		}
	}

	// Load historical results
	for i := 0; i < historyCount; i++ {
		suite, err := db.LoadHistorical(files[i])
		if err != nil {
			continue
		}

		for _, result := range suite.Results {
			if comp, exists := comparisons[result.Name]; exists {
				comp.History = append(comp.History, result)
			}
		}
	}

	// Calculate trends and detect regressions
	for _, comp := range comparisons {
		if len(comp.History) > 0 {
			comp.CalculateTrend()
			comp.DetectRegression(thresholds)
		}
	}

	return comparisons, nil
}

// HistoricalComparison represents a benchmark comparison with historical data
type HistoricalComparison struct {
	BenchmarkName   string
	Current         *BenchmarkResult
	History         []*BenchmarkResult
	AverageNsPerOp  float64
	TrendDirection  string // "improving", "stable", "degrading"
	IsRegression    bool
	RegressionScore float64 // How much worse than historical average (percentage)
}

// CalculateTrend analyzes the historical trend
func (hc *HistoricalComparison) CalculateTrend() {
	if len(hc.History) == 0 {
		return
	}

	// Calculate average from history
	sum := 0.0
	for _, h := range hc.History {
		sum += h.NsPerOp
	}
	hc.AverageNsPerOp = sum / float64(len(hc.History))

	// Determine trend
	percentDiff := ((hc.Current.NsPerOp - hc.AverageNsPerOp) / hc.AverageNsPerOp) * 100

	if percentDiff < -5.0 {
		hc.TrendDirection = "improving"
	} else if percentDiff > 5.0 {
		hc.TrendDirection = "degrading"
	} else {
		hc.TrendDirection = "stable"
	}

	hc.RegressionScore = percentDiff
}

// DetectRegression checks if current result is a regression compared to history
func (hc *HistoricalComparison) DetectRegression(thresholds *Thresholds) {
	if thresholds == nil {
		thresholds = DefaultThresholds()
	}

	if hc.RegressionScore > thresholds.TimeRegressionWarning {
		hc.IsRegression = true
	}
}
