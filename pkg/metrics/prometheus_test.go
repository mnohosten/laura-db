package metrics

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestPrometheusExporter_BasicMetrics(t *testing.T) {
	collector := NewMetricsCollector()
	exporter := NewPrometheusExporter(collector, nil)

	// Record some operations
	collector.RecordQuery(100*time.Millisecond, true)
	collector.RecordInsert(10*time.Millisecond, true)
	collector.RecordUpdate(50*time.Millisecond, false)
	collector.RecordDelete(5*time.Millisecond, true)

	var buf bytes.Buffer
	err := exporter.WriteMetrics(&buf)
	if err != nil {
		t.Fatalf("Failed to write metrics: %v", err)
	}

	output := buf.String()

	// Check for essential metric types
	if !strings.Contains(output, "# TYPE laura_db_queries_total counter") {
		t.Error("Missing queries_total counter type")
	}
	if !strings.Contains(output, "# TYPE laura_db_inserts_total counter") {
		t.Error("Missing inserts_total counter type")
	}
	if !strings.Contains(output, "# TYPE laura_db_updates_total counter") {
		t.Error("Missing updates_total counter type")
	}
	if !strings.Contains(output, "# TYPE laura_db_deletes_total counter") {
		t.Error("Missing deletes_total counter type")
	}

	// Check for metric values
	if !strings.Contains(output, "laura_db_queries_total 1") {
		t.Error("Expected queries_total to be 1")
	}
	if !strings.Contains(output, "laura_db_inserts_total 1") {
		t.Error("Expected inserts_total to be 1")
	}
	if !strings.Contains(output, "laura_db_updates_total 1") {
		t.Error("Expected updates_total to be 1")
	}
	if !strings.Contains(output, "laura_db_updates_failed_total 1") {
		t.Error("Expected updates_failed_total to be 1")
	}
	if !strings.Contains(output, "laura_db_deletes_total 1") {
		t.Error("Expected deletes_total to be 1")
	}
}

func TestPrometheusExporter_Histograms(t *testing.T) {
	collector := NewMetricsCollector()
	exporter := NewPrometheusExporter(collector, nil)

	// Record operations with different timings to populate histogram buckets
	collector.RecordQuery(500*time.Microsecond, true) // 0-1ms
	collector.RecordQuery(5*time.Millisecond, true)   // 1-10ms
	collector.RecordQuery(50*time.Millisecond, true)  // 10-100ms
	collector.RecordQuery(500*time.Millisecond, true) // 100-1000ms
	collector.RecordQuery(2*time.Second, true)        // >1000ms

	var buf bytes.Buffer
	err := exporter.WriteMetrics(&buf)
	if err != nil {
		t.Fatalf("Failed to write metrics: %v", err)
	}

	output := buf.String()

	// Check for histogram type
	if !strings.Contains(output, "# TYPE laura_db_query_duration_seconds histogram") {
		t.Error("Missing query_duration_seconds histogram type")
	}

	// Check for histogram buckets (cumulative counts)
	if !strings.Contains(output, "laura_db_query_duration_seconds_bucket{le=\"0.001\"} 1") {
		t.Error("Expected 1 operation in 0-1ms bucket")
	}
	if !strings.Contains(output, "laura_db_query_duration_seconds_bucket{le=\"0.01\"} 2") {
		t.Error("Expected cumulative 2 operations in 1-10ms bucket")
	}
	if !strings.Contains(output, "laura_db_query_duration_seconds_bucket{le=\"0.1\"} 3") {
		t.Error("Expected cumulative 3 operations in 10-100ms bucket")
	}
	if !strings.Contains(output, "laura_db_query_duration_seconds_bucket{le=\"1.0\"} 4") {
		t.Error("Expected cumulative 4 operations in 100-1000ms bucket")
	}
	if !strings.Contains(output, "laura_db_query_duration_seconds_bucket{le=\"+Inf\"} 5") {
		t.Error("Expected cumulative 5 operations in +Inf bucket")
	}

	// Check for count
	if !strings.Contains(output, "laura_db_query_duration_seconds_count 5") {
		t.Error("Expected histogram count to be 5")
	}
}

func TestPrometheusExporter_Percentiles(t *testing.T) {
	collector := NewMetricsCollector()
	exporter := NewPrometheusExporter(collector, nil)

	// Record 100 operations with varying timings
	for i := 0; i < 100; i++ {
		duration := time.Duration(i) * time.Millisecond
		collector.RecordQuery(duration, true)
	}

	var buf bytes.Buffer
	err := exporter.WriteMetrics(&buf)
	if err != nil {
		t.Fatalf("Failed to write metrics: %v", err)
	}

	output := buf.String()

	// Check for percentile metrics
	if !strings.Contains(output, "# TYPE laura_db_query_duration_seconds_p50 gauge") {
		t.Error("Missing P50 percentile metric")
	}
	if !strings.Contains(output, "# TYPE laura_db_query_duration_seconds_p95 gauge") {
		t.Error("Missing P95 percentile metric")
	}
	if !strings.Contains(output, "# TYPE laura_db_query_duration_seconds_p99 gauge") {
		t.Error("Missing P99 percentile metric")
	}

	// Check that percentile values are present (values will vary)
	if !strings.Contains(output, "laura_db_query_duration_seconds_p50") {
		t.Error("Missing P50 percentile value")
	}
	if !strings.Contains(output, "laura_db_query_duration_seconds_p95") {
		t.Error("Missing P95 percentile value")
	}
	if !strings.Contains(output, "laura_db_query_duration_seconds_p99") {
		t.Error("Missing P99 percentile value")
	}
}

func TestPrometheusExporter_TransactionMetrics(t *testing.T) {
	collector := NewMetricsCollector()
	exporter := NewPrometheusExporter(collector, nil)

	// Record transaction events
	collector.RecordTransactionStart()
	collector.RecordTransactionStart()
	collector.RecordTransactionCommit()
	collector.RecordTransactionAbort()

	var buf bytes.Buffer
	err := exporter.WriteMetrics(&buf)
	if err != nil {
		t.Fatalf("Failed to write metrics: %v", err)
	}

	output := buf.String()

	// Check transaction metrics
	if !strings.Contains(output, "laura_db_transactions_started_total 2") {
		t.Error("Expected transactions_started_total to be 2")
	}
	if !strings.Contains(output, "laura_db_transactions_committed_total 1") {
		t.Error("Expected transactions_committed_total to be 1")
	}
	if !strings.Contains(output, "laura_db_transactions_aborted_total 1") {
		t.Error("Expected transactions_aborted_total to be 1")
	}
}

func TestPrometheusExporter_CacheMetrics(t *testing.T) {
	collector := NewMetricsCollector()
	exporter := NewPrometheusExporter(collector, nil)

	// Record cache operations
	for i := 0; i < 7; i++ {
		collector.RecordCacheHit()
	}
	for i := 0; i < 3; i++ {
		collector.RecordCacheMiss()
	}

	var buf bytes.Buffer
	err := exporter.WriteMetrics(&buf)
	if err != nil {
		t.Fatalf("Failed to write metrics: %v", err)
	}

	output := buf.String()

	// Check cache metrics
	if !strings.Contains(output, "laura_db_cache_hits_total 7") {
		t.Error("Expected cache_hits_total to be 7")
	}
	if !strings.Contains(output, "laura_db_cache_misses_total 3") {
		t.Error("Expected cache_misses_total to be 3")
	}

	// Cache hit rate should be 0.7 (7/10)
	if !strings.Contains(output, "laura_db_cache_hit_rate 0.7") {
		t.Error("Expected cache_hit_rate to be 0.7")
	}
}

func TestPrometheusExporter_ScanMetrics(t *testing.T) {
	collector := NewMetricsCollector()
	exporter := NewPrometheusExporter(collector, nil)

	// Record scan operations
	for i := 0; i < 8; i++ {
		collector.RecordIndexScan()
	}
	for i := 0; i < 2; i++ {
		collector.RecordCollectionScan()
	}

	var buf bytes.Buffer
	err := exporter.WriteMetrics(&buf)
	if err != nil {
		t.Fatalf("Failed to write metrics: %v", err)
	}

	output := buf.String()

	// Check scan metrics
	if !strings.Contains(output, "laura_db_index_scans_total 8") {
		t.Error("Expected index_scans_total to be 8")
	}
	if !strings.Contains(output, "laura_db_collection_scans_total 2") {
		t.Error("Expected collection_scans_total to be 2")
	}

	// Index usage rate should be 0.8 (8/10)
	if !strings.Contains(output, "laura_db_index_usage_rate 0.8") {
		t.Error("Expected index_usage_rate to be 0.8")
	}
}

func TestPrometheusExporter_ConnectionMetrics(t *testing.T) {
	collector := NewMetricsCollector()
	exporter := NewPrometheusExporter(collector, nil)

	// Simulate connections
	collector.RecordConnectionStart()
	collector.RecordConnectionStart()
	collector.RecordConnectionStart()
	collector.RecordConnectionEnd()

	var buf bytes.Buffer
	err := exporter.WriteMetrics(&buf)
	if err != nil {
		t.Fatalf("Failed to write metrics: %v", err)
	}

	output := buf.String()

	// Check connection metrics
	if !strings.Contains(output, "laura_db_active_connections 2") {
		t.Error("Expected active_connections to be 2")
	}
	if !strings.Contains(output, "laura_db_connections_total 3") {
		t.Error("Expected connections_total to be 3")
	}
}

func TestPrometheusExporter_ResourceTrackerIntegration(t *testing.T) {
	collector := NewMetricsCollector()
	tracker := NewResourceTracker(nil) // Use default config
	defer tracker.Disable()

	exporter := NewPrometheusExporter(collector, tracker)

	// Give tracker time to collect some data
	time.Sleep(100 * time.Millisecond)

	// Record some I/O operations
	tracker.RecordRead(1024)
	tracker.RecordWrite(2048)

	var buf bytes.Buffer
	err := exporter.WriteMetrics(&buf)
	if err != nil {
		t.Fatalf("Failed to write metrics: %v", err)
	}

	output := buf.String()

	// Check for resource metrics
	if !strings.Contains(output, "# TYPE laura_db_memory_heap_bytes gauge") {
		t.Error("Missing memory_heap_bytes metric")
	}
	if !strings.Contains(output, "# TYPE laura_db_goroutines gauge") {
		t.Error("Missing goroutines metric")
	}
	if !strings.Contains(output, "# TYPE laura_db_io_bytes_read_total counter") {
		t.Error("Missing io_bytes_read_total metric")
	}
	if !strings.Contains(output, "# TYPE laura_db_io_bytes_written_total counter") {
		t.Error("Missing io_bytes_written_total metric")
	}
	if !strings.Contains(output, "# TYPE laura_db_cpu_count gauge") {
		t.Error("Missing cpu_count metric")
	}

	// Check I/O values
	if !strings.Contains(output, "laura_db_io_bytes_read_total 1024") {
		t.Error("Expected io_bytes_read_total to be 1024")
	}
	if !strings.Contains(output, "laura_db_io_bytes_written_total 2048") {
		t.Error("Expected io_bytes_written_total to be 2048")
	}
}

func TestPrometheusExporter_CustomNamespace(t *testing.T) {
	collector := NewMetricsCollector()
	exporter := NewPrometheusExporter(collector, nil)
	exporter.SetNamespace("custom_db")

	collector.RecordQuery(10*time.Millisecond, true)

	var buf bytes.Buffer
	err := exporter.WriteMetrics(&buf)
	if err != nil {
		t.Fatalf("Failed to write metrics: %v", err)
	}

	output := buf.String()

	// Check for custom namespace
	if !strings.Contains(output, "custom_db_queries_total 1") {
		t.Error("Expected custom namespace 'custom_db' in metric name")
	}
	if strings.Contains(output, "laura_db_queries_total") {
		t.Error("Should not contain default namespace 'laura_db'")
	}
}

func TestPrometheusExporter_UptimeMetric(t *testing.T) {
	collector := NewMetricsCollector()
	exporter := NewPrometheusExporter(collector, nil)

	// Wait a bit for uptime
	time.Sleep(100 * time.Millisecond)

	var buf bytes.Buffer
	err := exporter.WriteMetrics(&buf)
	if err != nil {
		t.Fatalf("Failed to write metrics: %v", err)
	}

	output := buf.String()

	// Check for uptime metric
	if !strings.Contains(output, "# TYPE laura_db_uptime_seconds gauge") {
		t.Error("Missing uptime_seconds metric")
	}
	if !strings.Contains(output, "laura_db_uptime_seconds") {
		t.Error("Missing uptime_seconds value")
	}
}

func TestPrometheusExporter_AllOperationTypes(t *testing.T) {
	collector := NewMetricsCollector()
	exporter := NewPrometheusExporter(collector, nil)

	// Record all types of operations
	collector.RecordQuery(10*time.Millisecond, true)
	collector.RecordInsert(20*time.Millisecond, true)
	collector.RecordUpdate(30*time.Millisecond, true)
	collector.RecordDelete(40*time.Millisecond, true)

	var buf bytes.Buffer
	err := exporter.WriteMetrics(&buf)
	if err != nil {
		t.Fatalf("Failed to write metrics: %v", err)
	}

	output := buf.String()

	// Check for all operation histograms
	operations := []string{"query", "insert", "update", "delete"}
	for _, op := range operations {
		metricName := "laura_db_" + op + "_duration_seconds"
		if !strings.Contains(output, "# TYPE "+metricName+" histogram") {
			t.Errorf("Missing histogram for %s", op)
		}
		if !strings.Contains(output, metricName+"_bucket{le=\"0.001\"}") {
			t.Errorf("Missing histogram buckets for %s", op)
		}
		if !strings.Contains(output, metricName+"_p50") {
			t.Errorf("Missing P50 percentile for %s", op)
		}
		if !strings.Contains(output, metricName+"_p95") {
			t.Errorf("Missing P95 percentile for %s", op)
		}
		if !strings.Contains(output, metricName+"_p99") {
			t.Errorf("Missing P99 percentile for %s", op)
		}
	}
}

func TestPrometheusExporter_EmptyMetrics(t *testing.T) {
	collector := NewMetricsCollector()
	exporter := NewPrometheusExporter(collector, nil)

	var buf bytes.Buffer
	err := exporter.WriteMetrics(&buf)
	if err != nil {
		t.Fatalf("Failed to write metrics: %v", err)
	}

	output := buf.String()

	// Should still have metric definitions even with zero values
	if !strings.Contains(output, "laura_db_queries_total 0") {
		t.Error("Expected queries_total to be 0 when no operations recorded")
	}
	if !strings.Contains(output, "laura_db_cache_hit_rate 0") {
		t.Error("Expected cache_hit_rate to be 0 when no cache operations")
	}
}

func TestPrometheusExporter_LargeMetricValues(t *testing.T) {
	collector := NewMetricsCollector()
	exporter := NewPrometheusExporter(collector, nil)

	// Record many operations
	for i := 0; i < 1000; i++ {
		collector.RecordQuery(time.Duration(i)*time.Microsecond, true)
	}

	var buf bytes.Buffer
	err := exporter.WriteMetrics(&buf)
	if err != nil {
		t.Fatalf("Failed to write metrics: %v", err)
	}

	output := buf.String()

	// Check that large values are formatted correctly
	if !strings.Contains(output, "laura_db_queries_total 1000") {
		t.Error("Expected queries_total to be 1000")
	}
}
