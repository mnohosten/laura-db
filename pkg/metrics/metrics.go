package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector collects real-time performance metrics for the database
type MetricsCollector struct {
	// Query metrics
	queriesExecuted   uint64
	queriesFailed     uint64
	totalQueryTime    uint64 // in nanoseconds

	// Insert metrics
	insertsExecuted   uint64
	insertsFailed     uint64
	totalInsertTime   uint64 // in nanoseconds

	// Update metrics
	updatesExecuted   uint64
	updatesFailed     uint64
	totalUpdateTime   uint64 // in nanoseconds

	// Delete metrics
	deletesExecuted   uint64
	deletesFailed     uint64
	totalDeleteTime   uint64 // in nanoseconds

	// Transaction metrics
	transactionsStarted   uint64
	transactionsCommitted uint64
	transactionsAborted   uint64

	// Cache metrics
	cacheHits         uint64
	cacheMisses       uint64

	// Index metrics
	indexScans        uint64
	collectionScans   uint64

	// Connection metrics (for HTTP server)
	activeConnections uint64
	totalConnections  uint64

	// Operation timing buckets (histogram)
	mu               sync.RWMutex
	queryTimings     *TimingHistogram
	insertTimings    *TimingHistogram
	updateTimings    *TimingHistogram
	deleteTimings    *TimingHistogram

	// Start time for uptime calculation
	startTime        time.Time
}

// TimingHistogram stores timing data in buckets for histogram generation
type TimingHistogram struct {
	// Buckets: <1ms, 1-10ms, 10-100ms, 100ms-1s, >1s
	bucket0_1ms      uint64 // 0-1ms
	bucket1_10ms     uint64 // 1-10ms
	bucket10_100ms   uint64 // 10-100ms
	bucket100_1000ms uint64 // 100-1000ms
	bucket1000ms     uint64 // >1s

	// P50, P95, P99 tracking
	mu               sync.Mutex
	recentTimings    []time.Duration // Keep last 1000 timings
	maxRecentTimings int
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		queryTimings:  NewTimingHistogram(1000),
		insertTimings: NewTimingHistogram(1000),
		updateTimings: NewTimingHistogram(1000),
		deleteTimings: NewTimingHistogram(1000),
		startTime:     time.Now(),
	}
}

// NewTimingHistogram creates a new timing histogram
func NewTimingHistogram(maxRecent int) *TimingHistogram {
	return &TimingHistogram{
		recentTimings:    make([]time.Duration, 0, maxRecent),
		maxRecentTimings: maxRecent,
	}
}

// RecordQuery records a query execution
func (mc *MetricsCollector) RecordQuery(duration time.Duration, success bool) {
	atomic.AddUint64(&mc.queriesExecuted, 1)
	if !success {
		atomic.AddUint64(&mc.queriesFailed, 1)
	}
	atomic.AddUint64(&mc.totalQueryTime, uint64(duration.Nanoseconds()))
	mc.queryTimings.Record(duration)
}

// RecordInsert records an insert operation
func (mc *MetricsCollector) RecordInsert(duration time.Duration, success bool) {
	atomic.AddUint64(&mc.insertsExecuted, 1)
	if !success {
		atomic.AddUint64(&mc.insertsFailed, 1)
	}
	atomic.AddUint64(&mc.totalInsertTime, uint64(duration.Nanoseconds()))
	mc.insertTimings.Record(duration)
}

// RecordUpdate records an update operation
func (mc *MetricsCollector) RecordUpdate(duration time.Duration, success bool) {
	atomic.AddUint64(&mc.updatesExecuted, 1)
	if !success {
		atomic.AddUint64(&mc.updatesFailed, 1)
	}
	atomic.AddUint64(&mc.totalUpdateTime, uint64(duration.Nanoseconds()))
	mc.updateTimings.Record(duration)
}

// RecordDelete records a delete operation
func (mc *MetricsCollector) RecordDelete(duration time.Duration, success bool) {
	atomic.AddUint64(&mc.deletesExecuted, 1)
	if !success {
		atomic.AddUint64(&mc.deletesFailed, 1)
	}
	atomic.AddUint64(&mc.totalDeleteTime, uint64(duration.Nanoseconds()))
	mc.deleteTimings.Record(duration)
}

// RecordTransaction records transaction events
func (mc *MetricsCollector) RecordTransactionStart() {
	atomic.AddUint64(&mc.transactionsStarted, 1)
}

func (mc *MetricsCollector) RecordTransactionCommit() {
	atomic.AddUint64(&mc.transactionsCommitted, 1)
}

func (mc *MetricsCollector) RecordTransactionAbort() {
	atomic.AddUint64(&mc.transactionsAborted, 1)
}

// RecordCacheHit records a cache hit
func (mc *MetricsCollector) RecordCacheHit() {
	atomic.AddUint64(&mc.cacheHits, 1)
}

// RecordCacheMiss records a cache miss
func (mc *MetricsCollector) RecordCacheMiss() {
	atomic.AddUint64(&mc.cacheMisses, 1)
}

// RecordIndexScan records an index scan
func (mc *MetricsCollector) RecordIndexScan() {
	atomic.AddUint64(&mc.indexScans, 1)
}

// RecordCollectionScan records a collection scan
func (mc *MetricsCollector) RecordCollectionScan() {
	atomic.AddUint64(&mc.collectionScans, 1)
}

// RecordConnection records connection metrics
func (mc *MetricsCollector) RecordConnectionStart() {
	atomic.AddUint64(&mc.totalConnections, 1)
	atomic.AddUint64(&mc.activeConnections, 1)
}

func (mc *MetricsCollector) RecordConnectionEnd() {
	atomic.AddUint64(&mc.activeConnections, ^uint64(0)) // Decrement using two's complement
}

// Record adds a timing to the histogram
func (th *TimingHistogram) Record(duration time.Duration) {
	// Update buckets atomically
	ms := duration.Milliseconds()
	if ms < 1 {
		atomic.AddUint64(&th.bucket0_1ms, 1)
	} else if ms < 10 {
		atomic.AddUint64(&th.bucket1_10ms, 1)
	} else if ms < 100 {
		atomic.AddUint64(&th.bucket10_100ms, 1)
	} else if ms < 1000 {
		atomic.AddUint64(&th.bucket100_1000ms, 1)
	} else {
		atomic.AddUint64(&th.bucket1000ms, 1)
	}

	// Add to recent timings for percentile calculation
	th.mu.Lock()
	defer th.mu.Unlock()

	if len(th.recentTimings) >= th.maxRecentTimings {
		// Shift array to remove oldest
		th.recentTimings = th.recentTimings[1:]
	}
	th.recentTimings = append(th.recentTimings, duration)
}

// GetBuckets returns the histogram bucket counts
func (th *TimingHistogram) GetBuckets() map[string]uint64 {
	return map[string]uint64{
		"0-1ms":       atomic.LoadUint64(&th.bucket0_1ms),
		"1-10ms":      atomic.LoadUint64(&th.bucket1_10ms),
		"10-100ms":    atomic.LoadUint64(&th.bucket10_100ms),
		"100-1000ms":  atomic.LoadUint64(&th.bucket100_1000ms),
		">1000ms":     atomic.LoadUint64(&th.bucket1000ms),
	}
}

// GetPercentiles calculates P50, P95, P99 from recent timings
func (th *TimingHistogram) GetPercentiles() map[string]time.Duration {
	th.mu.Lock()
	defer th.mu.Unlock()

	if len(th.recentTimings) == 0 {
		return map[string]time.Duration{
			"p50": 0,
			"p95": 0,
			"p99": 0,
		}
	}

	// Create sorted copy
	sorted := make([]time.Duration, len(th.recentTimings))
	copy(sorted, th.recentTimings)

	// Simple insertion sort (fine for 1000 elements)
	for i := 1; i < len(sorted); i++ {
		key := sorted[i]
		j := i - 1
		for j >= 0 && sorted[j] > key {
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = key
	}

	// Calculate percentiles
	p50idx := len(sorted) * 50 / 100
	p95idx := len(sorted) * 95 / 100
	p99idx := len(sorted) * 99 / 100

	return map[string]time.Duration{
		"p50": sorted[p50idx],
		"p95": sorted[p95idx],
		"p99": sorted[p99idx],
	}
}

// GetMetrics returns a snapshot of all metrics
func (mc *MetricsCollector) GetMetrics() map[string]interface{} {
	// Load all atomic counters
	queriesExecuted := atomic.LoadUint64(&mc.queriesExecuted)
	queriesFailed := atomic.LoadUint64(&mc.queriesFailed)
	totalQueryTime := atomic.LoadUint64(&mc.totalQueryTime)

	insertsExecuted := atomic.LoadUint64(&mc.insertsExecuted)
	insertsFailed := atomic.LoadUint64(&mc.insertsFailed)
	totalInsertTime := atomic.LoadUint64(&mc.totalInsertTime)

	updatesExecuted := atomic.LoadUint64(&mc.updatesExecuted)
	updatesFailed := atomic.LoadUint64(&mc.updatesFailed)
	totalUpdateTime := atomic.LoadUint64(&mc.totalUpdateTime)

	deletesExecuted := atomic.LoadUint64(&mc.deletesExecuted)
	deletesFailed := atomic.LoadUint64(&mc.deletesFailed)
	totalDeleteTime := atomic.LoadUint64(&mc.totalDeleteTime)

	transactionsStarted := atomic.LoadUint64(&mc.transactionsStarted)
	transactionsCommitted := atomic.LoadUint64(&mc.transactionsCommitted)
	transactionsAborted := atomic.LoadUint64(&mc.transactionsAborted)

	cacheHits := atomic.LoadUint64(&mc.cacheHits)
	cacheMisses := atomic.LoadUint64(&mc.cacheMisses)

	indexScans := atomic.LoadUint64(&mc.indexScans)
	collectionScans := atomic.LoadUint64(&mc.collectionScans)

	activeConnections := atomic.LoadUint64(&mc.activeConnections)
	totalConnections := atomic.LoadUint64(&mc.totalConnections)

	// Calculate averages (prevent division by zero)
	var avgQueryTime, avgInsertTime, avgUpdateTime, avgDeleteTime float64
	if queriesExecuted > 0 {
		avgQueryTime = float64(totalQueryTime) / float64(queriesExecuted) / 1e6 // Convert to ms
	}
	if insertsExecuted > 0 {
		avgInsertTime = float64(totalInsertTime) / float64(insertsExecuted) / 1e6
	}
	if updatesExecuted > 0 {
		avgUpdateTime = float64(totalUpdateTime) / float64(updatesExecuted) / 1e6
	}
	if deletesExecuted > 0 {
		avgDeleteTime = float64(totalDeleteTime) / float64(deletesExecuted) / 1e6
	}

	// Calculate cache hit rate
	var cacheHitRate float64
	totalCacheOps := cacheHits + cacheMisses
	if totalCacheOps > 0 {
		cacheHitRate = float64(cacheHits) / float64(totalCacheOps) * 100
	}

	// Calculate uptime
	uptime := time.Since(mc.startTime)

	return map[string]interface{}{
		"uptime_seconds": uptime.Seconds(),

		"queries": map[string]interface{}{
			"total":              queriesExecuted,
			"failed":             queriesFailed,
			"success_rate":       calculateSuccessRate(queriesExecuted, queriesFailed),
			"avg_duration_ms":    avgQueryTime,
			"timing_histogram":   mc.queryTimings.GetBuckets(),
			"timing_percentiles": mc.queryTimings.GetPercentiles(),
		},

		"inserts": map[string]interface{}{
			"total":              insertsExecuted,
			"failed":             insertsFailed,
			"success_rate":       calculateSuccessRate(insertsExecuted, insertsFailed),
			"avg_duration_ms":    avgInsertTime,
			"timing_histogram":   mc.insertTimings.GetBuckets(),
			"timing_percentiles": mc.insertTimings.GetPercentiles(),
		},

		"updates": map[string]interface{}{
			"total":              updatesExecuted,
			"failed":             updatesFailed,
			"success_rate":       calculateSuccessRate(updatesExecuted, updatesFailed),
			"avg_duration_ms":    avgUpdateTime,
			"timing_histogram":   mc.updateTimings.GetBuckets(),
			"timing_percentiles": mc.updateTimings.GetPercentiles(),
		},

		"deletes": map[string]interface{}{
			"total":              deletesExecuted,
			"failed":             deletesFailed,
			"success_rate":       calculateSuccessRate(deletesExecuted, deletesFailed),
			"avg_duration_ms":    avgDeleteTime,
			"timing_histogram":   mc.deleteTimings.GetBuckets(),
			"timing_percentiles": mc.deleteTimings.GetPercentiles(),
		},

		"transactions": map[string]interface{}{
			"started":       transactionsStarted,
			"committed":     transactionsCommitted,
			"aborted":       transactionsAborted,
			"commit_rate":   calculateSuccessRate(transactionsStarted, transactionsAborted),
		},

		"cache": map[string]interface{}{
			"hits":        cacheHits,
			"misses":      cacheMisses,
			"hit_rate":    cacheHitRate,
		},

		"scans": map[string]interface{}{
			"index":           indexScans,
			"collection":      collectionScans,
			"index_usage_pct": calculateIndexUsageRate(indexScans, collectionScans),
		},

		"connections": map[string]interface{}{
			"active": activeConnections,
			"total":  totalConnections,
		},
	}
}

// Reset resets all metrics to zero
func (mc *MetricsCollector) Reset() {
	atomic.StoreUint64(&mc.queriesExecuted, 0)
	atomic.StoreUint64(&mc.queriesFailed, 0)
	atomic.StoreUint64(&mc.totalQueryTime, 0)

	atomic.StoreUint64(&mc.insertsExecuted, 0)
	atomic.StoreUint64(&mc.insertsFailed, 0)
	atomic.StoreUint64(&mc.totalInsertTime, 0)

	atomic.StoreUint64(&mc.updatesExecuted, 0)
	atomic.StoreUint64(&mc.updatesFailed, 0)
	atomic.StoreUint64(&mc.totalUpdateTime, 0)

	atomic.StoreUint64(&mc.deletesExecuted, 0)
	atomic.StoreUint64(&mc.deletesFailed, 0)
	atomic.StoreUint64(&mc.totalDeleteTime, 0)

	atomic.StoreUint64(&mc.transactionsStarted, 0)
	atomic.StoreUint64(&mc.transactionsCommitted, 0)
	atomic.StoreUint64(&mc.transactionsAborted, 0)

	atomic.StoreUint64(&mc.cacheHits, 0)
	atomic.StoreUint64(&mc.cacheMisses, 0)

	atomic.StoreUint64(&mc.indexScans, 0)
	atomic.StoreUint64(&mc.collectionScans, 0)

	atomic.StoreUint64(&mc.totalConnections, 0)
	// Don't reset activeConnections as it represents current state

	// Reset histograms
	mc.mu.Lock()
	mc.queryTimings = NewTimingHistogram(1000)
	mc.insertTimings = NewTimingHistogram(1000)
	mc.updateTimings = NewTimingHistogram(1000)
	mc.deleteTimings = NewTimingHistogram(1000)
	mc.mu.Unlock()

	// Reset start time
	mc.startTime = time.Now()
}

// Helper functions

func calculateSuccessRate(total, failed uint64) float64 {
	if total == 0 {
		return 0
	}
	succeeded := total - failed
	return float64(succeeded) / float64(total) * 100
}

func calculateIndexUsageRate(indexScans, collectionScans uint64) float64 {
	total := indexScans + collectionScans
	if total == 0 {
		return 0
	}
	return float64(indexScans) / float64(total) * 100
}
