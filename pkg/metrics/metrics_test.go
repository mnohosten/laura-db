package metrics

import (
	"testing"
	"time"
)

func TestMetricsCollector_RecordQuery(t *testing.T) {
	mc := NewMetricsCollector()

	// Record successful queries
	mc.RecordQuery(10*time.Millisecond, true)
	mc.RecordQuery(20*time.Millisecond, true)
	mc.RecordQuery(5*time.Millisecond, false) // Failed query

	metrics := mc.GetMetrics()
	queries := metrics["queries"].(map[string]interface{})

	if queries["total"].(uint64) != 3 {
		t.Errorf("Expected 3 total queries, got %v", queries["total"])
	}
	if queries["failed"].(uint64) != 1 {
		t.Errorf("Expected 1 failed query, got %v", queries["failed"])
	}

	successRate := queries["success_rate"].(float64)
	if successRate < 66.0 || successRate > 67.0 {
		t.Errorf("Expected success rate around 66.67%%, got %.2f%%", successRate)
	}
}

func TestMetricsCollector_RecordInsert(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordInsert(1*time.Millisecond, true)
	mc.RecordInsert(2*time.Millisecond, true)
	mc.RecordInsert(3*time.Millisecond, true)

	metrics := mc.GetMetrics()
	inserts := metrics["inserts"].(map[string]interface{})

	if inserts["total"].(uint64) != 3 {
		t.Errorf("Expected 3 total inserts, got %v", inserts["total"])
	}
	if inserts["failed"].(uint64) != 0 {
		t.Errorf("Expected 0 failed inserts, got %v", inserts["failed"])
	}

	successRate := inserts["success_rate"].(float64)
	if successRate != 100.0 {
		t.Errorf("Expected 100%% success rate, got %.2f%%", successRate)
	}
}

func TestMetricsCollector_RecordUpdate(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordUpdate(5*time.Millisecond, true)
	mc.RecordUpdate(10*time.Millisecond, false)

	metrics := mc.GetMetrics()
	updates := metrics["updates"].(map[string]interface{})

	if updates["total"].(uint64) != 2 {
		t.Errorf("Expected 2 total updates, got %v", updates["total"])
	}
	if updates["failed"].(uint64) != 1 {
		t.Errorf("Expected 1 failed update, got %v", updates["failed"])
	}
}

func TestMetricsCollector_RecordDelete(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordDelete(3*time.Millisecond, true)
	mc.RecordDelete(7*time.Millisecond, true)
	mc.RecordDelete(2*time.Millisecond, true)

	metrics := mc.GetMetrics()
	deletes := metrics["deletes"].(map[string]interface{})

	if deletes["total"].(uint64) != 3 {
		t.Errorf("Expected 3 total deletes, got %v", deletes["total"])
	}
	if deletes["failed"].(uint64) != 0 {
		t.Errorf("Expected 0 failed deletes, got %v", deletes["failed"])
	}
}

func TestMetricsCollector_Transactions(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordTransactionStart()
	mc.RecordTransactionStart()
	mc.RecordTransactionCommit()
	mc.RecordTransactionStart()
	mc.RecordTransactionAbort()
	mc.RecordTransactionCommit()

	metrics := mc.GetMetrics()
	txns := metrics["transactions"].(map[string]interface{})

	if txns["started"].(uint64) != 3 {
		t.Errorf("Expected 3 started transactions, got %v", txns["started"])
	}
	if txns["committed"].(uint64) != 2 {
		t.Errorf("Expected 2 committed transactions, got %v", txns["committed"])
	}
	if txns["aborted"].(uint64) != 1 {
		t.Errorf("Expected 1 aborted transaction, got %v", txns["aborted"])
	}
}

func TestMetricsCollector_Cache(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordCacheHit()
	mc.RecordCacheHit()
	mc.RecordCacheHit()
	mc.RecordCacheMiss()

	metrics := mc.GetMetrics()
	cache := metrics["cache"].(map[string]interface{})

	if cache["hits"].(uint64) != 3 {
		t.Errorf("Expected 3 cache hits, got %v", cache["hits"])
	}
	if cache["misses"].(uint64) != 1 {
		t.Errorf("Expected 1 cache miss, got %v", cache["misses"])
	}

	hitRate := cache["hit_rate"].(float64)
	if hitRate != 75.0 {
		t.Errorf("Expected 75%% hit rate, got %.2f%%", hitRate)
	}
}

func TestMetricsCollector_Scans(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordIndexScan()
	mc.RecordIndexScan()
	mc.RecordIndexScan()
	mc.RecordCollectionScan()

	metrics := mc.GetMetrics()
	scans := metrics["scans"].(map[string]interface{})

	if scans["index"].(uint64) != 3 {
		t.Errorf("Expected 3 index scans, got %v", scans["index"])
	}
	if scans["collection"].(uint64) != 1 {
		t.Errorf("Expected 1 collection scan, got %v", scans["collection"])
	}

	indexUsage := scans["index_usage_pct"].(float64)
	if indexUsage != 75.0 {
		t.Errorf("Expected 75%% index usage, got %.2f%%", indexUsage)
	}
}

func TestMetricsCollector_Connections(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordConnectionStart()
	mc.RecordConnectionStart()
	mc.RecordConnectionStart()
	mc.RecordConnectionEnd()

	metrics := mc.GetMetrics()
	conns := metrics["connections"].(map[string]interface{})

	if conns["active"].(uint64) != 2 {
		t.Errorf("Expected 2 active connections, got %v", conns["active"])
	}
	if conns["total"].(uint64) != 3 {
		t.Errorf("Expected 3 total connections, got %v", conns["total"])
	}
}

func TestTimingHistogram_Buckets(t *testing.T) {
	th := NewTimingHistogram(100)

	// Record timings in different buckets
	th.Record(500 * time.Microsecond)  // <1ms
	th.Record(5 * time.Millisecond)    // 1-10ms
	th.Record(50 * time.Millisecond)   // 10-100ms
	th.Record(500 * time.Millisecond)  // 100-1000ms
	th.Record(1500 * time.Millisecond) // >1s

	buckets := th.GetBuckets()

	if buckets["0-1ms"] != 1 {
		t.Errorf("Expected 1 in 0-1ms bucket, got %v", buckets["0-1ms"])
	}
	if buckets["1-10ms"] != 1 {
		t.Errorf("Expected 1 in 1-10ms bucket, got %v", buckets["1-10ms"])
	}
	if buckets["10-100ms"] != 1 {
		t.Errorf("Expected 1 in 10-100ms bucket, got %v", buckets["10-100ms"])
	}
	if buckets["100-1000ms"] != 1 {
		t.Errorf("Expected 1 in 100-1000ms bucket, got %v", buckets["100-1000ms"])
	}
	if buckets[">1000ms"] != 1 {
		t.Errorf("Expected 1 in >1000ms bucket, got %v", buckets[">1000ms"])
	}
}

func TestTimingHistogram_Percentiles(t *testing.T) {
	th := NewTimingHistogram(100)

	// Record 100 timings
	for i := 1; i <= 100; i++ {
		th.Record(time.Duration(i) * time.Millisecond)
	}

	percentiles := th.GetPercentiles()

	p50 := percentiles["p50"]
	if p50 < 40*time.Millisecond || p50 > 60*time.Millisecond {
		t.Errorf("Expected p50 around 50ms, got %v", p50)
	}

	p95 := percentiles["p95"]
	if p95 < 90*time.Millisecond || p95 > 100*time.Millisecond {
		t.Errorf("Expected p95 around 95ms, got %v", p95)
	}

	p99 := percentiles["p99"]
	if p99 < 95*time.Millisecond || p99 > 100*time.Millisecond {
		t.Errorf("Expected p99 around 99ms, got %v", p99)
	}
}

func TestTimingHistogram_EmptyPercentiles(t *testing.T) {
	th := NewTimingHistogram(100)

	percentiles := th.GetPercentiles()

	if percentiles["p50"] != 0 {
		t.Errorf("Expected p50 to be 0 for empty histogram, got %v", percentiles["p50"])
	}
	if percentiles["p95"] != 0 {
		t.Errorf("Expected p95 to be 0 for empty histogram, got %v", percentiles["p95"])
	}
	if percentiles["p99"] != 0 {
		t.Errorf("Expected p99 to be 0 for empty histogram, got %v", percentiles["p99"])
	}
}

func TestMetricsCollector_Reset(t *testing.T) {
	mc := NewMetricsCollector()

	// Record some metrics
	mc.RecordQuery(10*time.Millisecond, true)
	mc.RecordInsert(5*time.Millisecond, true)
	mc.RecordCacheHit()

	// Verify metrics are recorded
	metrics := mc.GetMetrics()
	if metrics["queries"].(map[string]interface{})["total"].(uint64) != 1 {
		t.Error("Expected 1 query before reset")
	}

	// Reset metrics
	mc.Reset()

	// Verify all metrics are reset
	metrics = mc.GetMetrics()
	queries := metrics["queries"].(map[string]interface{})
	inserts := metrics["inserts"].(map[string]interface{})
	cache := metrics["cache"].(map[string]interface{})

	if queries["total"].(uint64) != 0 {
		t.Errorf("Expected 0 queries after reset, got %v", queries["total"])
	}
	if inserts["total"].(uint64) != 0 {
		t.Errorf("Expected 0 inserts after reset, got %v", inserts["total"])
	}
	if cache["hits"].(uint64) != 0 {
		t.Errorf("Expected 0 cache hits after reset, got %v", cache["hits"])
	}
}

func TestMetricsCollector_AverageTiming(t *testing.T) {
	mc := NewMetricsCollector()

	// Record queries with known durations
	mc.RecordQuery(10*time.Millisecond, true)
	mc.RecordQuery(20*time.Millisecond, true)
	mc.RecordQuery(30*time.Millisecond, true)

	metrics := mc.GetMetrics()
	queries := metrics["queries"].(map[string]interface{})
	avgDuration := queries["avg_duration_ms"].(float64)

	// Average should be 20ms
	if avgDuration < 19.0 || avgDuration > 21.0 {
		t.Errorf("Expected average duration around 20ms, got %.2fms", avgDuration)
	}
}

func TestMetricsCollector_Uptime(t *testing.T) {
	mc := NewMetricsCollector()

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	metrics := mc.GetMetrics()
	uptime := metrics["uptime_seconds"].(float64)

	if uptime < 0.1 {
		t.Errorf("Expected uptime >= 0.1 seconds, got %.3f", uptime)
	}
}

func TestMetricsCollector_ZeroDivision(t *testing.T) {
	mc := NewMetricsCollector()

	// Get metrics without recording anything
	metrics := mc.GetMetrics()
	queries := metrics["queries"].(map[string]interface{})

	// Should not panic and should return 0 for averages
	if queries["avg_duration_ms"].(float64) != 0 {
		t.Errorf("Expected 0 average duration with no queries, got %v", queries["avg_duration_ms"])
	}

	cache := metrics["cache"].(map[string]interface{})
	if cache["hit_rate"].(float64) != 0 {
		t.Errorf("Expected 0 hit rate with no cache operations, got %v", cache["hit_rate"])
	}
}

func TestTimingHistogram_CircularBuffer(t *testing.T) {
	th := NewTimingHistogram(5) // Small buffer

	// Add more than max capacity
	for i := 1; i <= 10; i++ {
		th.Record(time.Duration(i) * time.Millisecond)
	}

	// Should only keep last 5
	th.mu.Lock()
	count := len(th.recentTimings)
	th.mu.Unlock()

	if count != 5 {
		t.Errorf("Expected 5 recent timings, got %d", count)
	}

	// Percentiles should be calculated from last 5 (6-10)
	percentiles := th.GetPercentiles()
	p50 := percentiles["p50"]

	// P50 of [6,7,8,9,10] should be 8
	if p50 < 7*time.Millisecond || p50 > 9*time.Millisecond {
		t.Errorf("Expected p50 around 8ms, got %v", p50)
	}
}

func TestMetricsCollector_Concurrent(t *testing.T) {
	mc := NewMetricsCollector()

	// Run concurrent operations
	done := make(chan bool, 4)

	go func() {
		for i := 0; i < 100; i++ {
			mc.RecordQuery(1*time.Millisecond, true)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			mc.RecordInsert(1*time.Millisecond, true)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			mc.RecordCacheHit()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = mc.GetMetrics()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}

	metrics := mc.GetMetrics()
	queries := metrics["queries"].(map[string]interface{})
	inserts := metrics["inserts"].(map[string]interface{})
	cache := metrics["cache"].(map[string]interface{})

	if queries["total"].(uint64) != 100 {
		t.Errorf("Expected 100 queries, got %v", queries["total"])
	}
	if inserts["total"].(uint64) != 100 {
		t.Errorf("Expected 100 inserts, got %v", inserts["total"])
	}
	if cache["hits"].(uint64) != 100 {
		t.Errorf("Expected 100 cache hits, got %v", cache["hits"])
	}
}
