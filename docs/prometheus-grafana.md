# Prometheus & Grafana Integration

LauraDB provides native Prometheus metrics integration for comprehensive database monitoring and observability.

## Overview

LauraDB exports metrics in Prometheus text format via the `/_metrics` HTTP endpoint when running in server mode. These metrics can be scraped by Prometheus and visualized in Grafana dashboards.

**Key Features:**
- ✅ 40+ metrics covering all database operations
- ✅ Histogram-based latency tracking with percentiles
- ✅ Resource monitoring (memory, CPU, I/O, GC)
- ✅ Zero external dependencies
- ✅ Prometheus text format (OpenMetrics compatible)
- ✅ Built-in workload simulator for testing

## Quick Start

### 1. Start LauraDB Server

```bash
./bin/laura-server -port 8080 -data-dir ./data
```

### 2. View Metrics

```bash
curl http://localhost:8080/_metrics
```

Example output:
```
# HELP laura_db_uptime_seconds Database uptime in seconds
# TYPE laura_db_uptime_seconds gauge
laura_db_uptime_seconds 42.5

# HELP laura_db_queries_total Total number of queries executed
# TYPE laura_db_queries_total counter
laura_db_queries_total 1234

# HELP laura_db_query_duration_seconds Query execution duration histogram
# TYPE laura_db_query_duration_seconds histogram
laura_db_query_duration_seconds_bucket{le="0.001"} 500
laura_db_query_duration_seconds_bucket{le="0.01"} 1100
laura_db_query_duration_seconds_bucket{le="0.1"} 1200
laura_db_query_duration_seconds_bucket{le="1.0"} 1230
laura_db_query_duration_seconds_bucket{le="+Inf"} 1234
laura_db_query_duration_seconds_count 1234
```

### 3. Configure Prometheus

Create `prometheus.yml`:

```yaml
global:
  scrape_interval: 5s

scrape_configs:
  - job_name: 'laura_db'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/_metrics'
```

Start Prometheus:
```bash
prometheus --config.file=prometheus.yml
```

### 4. Visualize in Grafana

1. Add Prometheus data source in Grafana
2. Import the dashboard from `examples/prometheus-demo/grafana-dashboard.json`
3. View real-time metrics

## Metric Reference

### Uptime

| Metric | Type | Description |
|--------|------|-------------|
| `laura_db_uptime_seconds` | gauge | Database uptime in seconds |

### Operation Counters

| Metric | Type | Description |
|--------|------|-------------|
| `laura_db_queries_total` | counter | Total queries executed |
| `laura_db_queries_failed_total` | counter | Total failed queries |
| `laura_db_inserts_total` | counter | Total insert operations |
| `laura_db_inserts_failed_total` | counter | Total failed inserts |
| `laura_db_updates_total` | counter | Total update operations |
| `laura_db_updates_failed_total` | counter | Total failed updates |
| `laura_db_deletes_total` | counter | Total delete operations |
| `laura_db_deletes_failed_total` | counter | Total failed deletes |

### Latency Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `laura_db_query_duration_seconds` | histogram | Query latency distribution |
| `laura_db_query_duration_seconds_p50` | gauge | 50th percentile query latency |
| `laura_db_query_duration_seconds_p95` | gauge | 95th percentile query latency |
| `laura_db_query_duration_seconds_p99` | gauge | 99th percentile query latency |
| `laura_db_insert_duration_seconds` | histogram | Insert latency distribution |
| `laura_db_insert_duration_seconds_p50` | gauge | 50th percentile insert latency |
| `laura_db_insert_duration_seconds_p95` | gauge | 95th percentile insert latency |
| `laura_db_insert_duration_seconds_p99` | gauge | 99th percentile insert latency |
| `laura_db_update_duration_seconds` | histogram | Update latency distribution |
| `laura_db_update_duration_seconds_p50` | gauge | 50th percentile update latency |
| `laura_db_update_duration_seconds_p95` | gauge | 95th percentile update latency |
| `laura_db_update_duration_seconds_p99` | gauge | 99th percentile update latency |
| `laura_db_delete_duration_seconds` | histogram | Delete latency distribution |
| `laura_db_delete_duration_seconds_p50` | gauge | 50th percentile delete latency |
| `laura_db_delete_duration_seconds_p95` | gauge | 95th percentile delete latency |
| `laura_db_delete_duration_seconds_p99` | gauge | 99th percentile delete latency |

### Transaction Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `laura_db_transactions_started_total` | counter | Total transactions started |
| `laura_db_transactions_committed_total` | counter | Total transactions committed |
| `laura_db_transactions_aborted_total` | counter | Total transactions aborted |

### Cache Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `laura_db_cache_hits_total` | counter | Total cache hits |
| `laura_db_cache_misses_total` | counter | Total cache misses |
| `laura_db_cache_hit_rate` | gauge | Cache hit rate (0-1) |

### Scan Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `laura_db_index_scans_total` | counter | Total index scan operations |
| `laura_db_collection_scans_total` | counter | Total collection scan operations |
| `laura_db_index_usage_rate` | gauge | Index usage rate (0-1) |

### Connection Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `laura_db_active_connections` | gauge | Current active connections |
| `laura_db_connections_total` | counter | Total connections |

### Resource Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `laura_db_memory_heap_bytes` | gauge | Heap memory usage in bytes |
| `laura_db_memory_stack_bytes` | gauge | Stack memory usage in bytes |
| `laura_db_memory_allocations_total` | counter | Total memory allocations |
| `laura_db_memory_objects` | gauge | Number of allocated objects |
| `laura_db_goroutines` | gauge | Number of goroutines |
| `laura_db_io_bytes_read_total` | counter | Total bytes read |
| `laura_db_io_bytes_written_total` | counter | Total bytes written |
| `laura_db_io_read_operations_total` | counter | Total read operations |
| `laura_db_io_write_operations_total` | counter | Total write operations |
| `laura_db_gc_runs_total` | counter | Total garbage collection runs |
| `laura_db_gc_pause_nanoseconds` | gauge | Last GC pause time in nanoseconds |
| `laura_db_cpu_count` | gauge | Number of CPUs |

## Useful PromQL Queries

### Query Throughput (queries per second)
```promql
rate(laura_db_queries_total[1m])
```

### Error Rate (percentage)
```promql
rate(laura_db_queries_failed_total[1m]) / rate(laura_db_queries_total[1m]) * 100
```

### Average Query Latency (milliseconds)
```promql
rate(laura_db_query_duration_nanoseconds_total[1m]) / rate(laura_db_queries_total[1m]) / 1000000
```

### Cache Hit Rate (percentage)
```promql
laura_db_cache_hit_rate * 100
```

### Index Usage (percentage)
```promql
laura_db_index_usage_rate * 100
```

### Transaction Commit Rate (percentage)
```promql
rate(laura_db_transactions_committed_total[1m]) / rate(laura_db_transactions_started_total[1m]) * 100
```

### Memory Growth Rate (bytes/second)
```promql
deriv(laura_db_memory_heap_bytes[5m])
```

### I/O Throughput (KB/s)
```promql
rate(laura_db_io_bytes_read_total[1m]) / 1024  # Read
rate(laura_db_io_bytes_written_total[1m]) / 1024  # Write
```

### P95 Query Latency by Operation
```promql
laura_db_query_duration_seconds_p95{operation="query"}
laura_db_insert_duration_seconds_p95{operation="insert"}
```

### Success Rate by Operation
```promql
(rate(laura_db_queries_total[5m]) - rate(laura_db_queries_failed_total[5m])) / rate(laura_db_queries_total[5m]) * 100
```

## Grafana Dashboard Setup

### Manual Setup

1. **Add Data Source:**
   - Configuration → Data Sources → Add data source
   - Select "Prometheus"
   - URL: `http://localhost:9090`
   - Click "Save & Test"

2. **Create Dashboard:**
   - Dashboards → New Dashboard → Add new panel
   - Enter PromQL query
   - Configure visualization (Graph, Gauge, Stat, etc.)
   - Save panel and dashboard

3. **Suggested Panels:**

   **Query Rate Panel:**
   - Type: Graph
   - Query: `rate(laura_db_queries_total[1m])`
   - Title: "Queries per Second"

   **Latency Panel:**
   - Type: Graph
   - Queries:
     - `laura_db_query_duration_seconds_p50` (P50)
     - `laura_db_query_duration_seconds_p95` (P95)
     - `laura_db_query_duration_seconds_p99` (P99)
   - Title: "Query Latency Percentiles"

   **Cache Hit Rate Panel:**
   - Type: Stat
   - Query: `laura_db_cache_hit_rate * 100`
   - Unit: Percent (0-100)
   - Title: "Cache Hit Rate"

   **Memory Usage Panel:**
   - Type: Graph
   - Query: `laura_db_memory_heap_bytes / 1024 / 1024`
   - Unit: MiB
   - Title: "Heap Memory Usage"

### Import Pre-built Dashboard

Use the included dashboard template:

```bash
# In Grafana UI:
# Dashboards → Import → Upload JSON file
# Select: examples/prometheus-demo/grafana-dashboard.json
```

The pre-built dashboard includes 10 panels covering all key metrics.

## Alerting

### Prometheus Alert Rules

Create `alerts.yml`:

```yaml
groups:
  - name: lauradb_alerts
    interval: 30s
    rules:
      # High error rate alert
      - alert: HighQueryErrorRate
        expr: rate(laura_db_queries_failed_total[5m]) / rate(laura_db_queries_total[5m]) > 0.05
        for: 2m
        labels:
          severity: warning
          component: database
        annotations:
          summary: "High query error rate detected"
          description: "Query error rate is {{ $value | humanizePercentage }} (threshold: 5%)"

      # Low cache hit rate
      - alert: LowCacheHitRate
        expr: laura_db_cache_hit_rate < 0.5
        for: 5m
        labels:
          severity: warning
          component: cache
        annotations:
          summary: "Low cache hit rate"
          description: "Cache hit rate is {{ $value | humanizePercentage }} (threshold: 50%)"

      # High P99 latency
      - alert: HighQueryLatencyP99
        expr: laura_db_query_duration_seconds_p99 > 1.0
        for: 5m
        labels:
          severity: critical
          component: performance
        annotations:
          summary: "High P99 query latency"
          description: "P99 latency is {{ $value }}s (threshold: 1s)"

      # High memory usage
      - alert: HighMemoryUsage
        expr: laura_db_memory_heap_bytes > 1e9
        for: 10m
        labels:
          severity: warning
          component: resources
        annotations:
          summary: "High memory usage"
          description: "Heap memory is {{ $value | humanize1024 }}B (threshold: 1GB)"

      # Low index usage
      - alert: LowIndexUsage
        expr: laura_db_index_usage_rate < 0.7
        for: 10m
        labels:
          severity: info
          component: optimization
        annotations:
          summary: "Low index usage detected"
          description: "Index usage is {{ $value | humanizePercentage }} (threshold: 70%)"

      # High transaction abort rate
      - alert: HighTransactionAbortRate
        expr: rate(laura_db_transactions_aborted_total[5m]) / rate(laura_db_transactions_started_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
          component: transactions
        annotations:
          summary: "High transaction abort rate"
          description: "{{ $value | humanizePercentage }} of transactions are aborting"
```

Load alerts in Prometheus config:

```yaml
rule_files:
  - "alerts.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets: ['localhost:9093']  # Alertmanager address
```

## Custom Namespace

Change the metric namespace from default `laura_db` to a custom name:

```go
import "github.com/mnohosten/laura-db/pkg/metrics"

exporter := metrics.NewPrometheusExporter(collector, tracker)
exporter.SetNamespace("my_database")
```

Metrics will be prefixed with `my_database_` instead of `laura_db_`.

## Performance Considerations

### Scrape Interval

- **Production:** 15-30 seconds (balances resolution and overhead)
- **Development:** 5-10 seconds (faster feedback)
- **High-frequency:** 1-5 seconds (detailed analysis, higher load)

```yaml
scrape_configs:
  - job_name: 'laura_db'
    scrape_interval: 15s  # Adjust based on needs
```

### Metric Overhead

The Prometheus exporter has minimal performance impact:
- Metrics collection: ~10 nanoseconds per operation (atomic counters)
- Metrics export: ~1-2 milliseconds per scrape (minimal)
- Memory overhead: ~50KB for collector + resource tracker

### Retention

Configure Prometheus retention:

```bash
prometheus --storage.tsdb.retention.time=30d --storage.tsdb.retention.size=10GB
```

## Advanced Configuration

### Recording Rules

Pre-compute expensive queries with recording rules:

```yaml
groups:
  - name: lauradb_recording
    interval: 10s
    rules:
      # Pre-compute query throughput
      - record: laura_db:query_rate:1m
        expr: rate(laura_db_queries_total[1m])

      # Pre-compute error rate
      - record: laura_db:error_rate:1m
        expr: rate(laura_db_queries_failed_total[1m]) / rate(laura_db_queries_total[1m])

      # Pre-compute average latency
      - record: laura_db:query_latency_avg:1m
        expr: rate(laura_db_query_duration_nanoseconds_total[1m]) / rate(laura_db_queries_total[1m]) / 1000000
```

### Federation

Aggregate metrics from multiple LauraDB instances:

```yaml
# Central Prometheus server
scrape_configs:
  - job_name: 'federate'
    scrape_interval: 15s
    honor_labels: true
    metrics_path: '/federate'
    params:
      'match[]':
        - '{job="laura_db"}'
    static_configs:
      - targets:
        - 'prometheus-region1:9090'
        - 'prometheus-region2:9090'
```

### Long-term Storage

Use Prometheus remote write for long-term storage:

```yaml
remote_write:
  - url: "https://your-remote-storage/api/v1/write"
    queue_config:
      capacity: 10000
      max_samples_per_send: 1000
```

Compatible with:
- Thanos
- Cortex
- VictoriaMetrics
- InfluxDB
- Grafana Cloud

## Example: Complete Monitoring Stack

See `examples/prometheus-demo/` for a complete example with:
- LauraDB server with metrics
- Prometheus configuration
- Grafana dashboard
- Workload simulator
- Alert rules

Run the demo:

```bash
cd examples/prometheus-demo
go run main.go
```

Access endpoints:
- Metrics: http://localhost:8080/_metrics
- Admin console: http://localhost:8080/
- Prometheus: http://localhost:9090 (after starting Prometheus)
- Grafana: http://localhost:3000 (after starting Grafana)

## Troubleshooting

### Metrics Endpoint Returns Error

Check that the server is running:
```bash
curl http://localhost:8080/_health
```

### Prometheus Can't Scrape

1. Verify target is up in Prometheus UI (Status → Targets)
2. Check firewall rules
3. Verify metrics_path is correct (`/_metrics`)
4. Check Prometheus logs: `prometheus --log.level=debug`

### Missing Metrics

1. Generate some database activity first
2. Check if metrics are being recorded:
   ```bash
   curl http://localhost:8080/_metrics | grep laura_db_queries_total
   ```
3. Verify Prometheus is scraping (check `/federate` endpoint)

### High Memory Usage

1. Reduce scrape frequency
2. Limit retention time
3. Use recording rules for expensive queries
4. Consider remote storage for long-term data

## References

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [PromQL Basics](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [OpenMetrics Specification](https://openmetrics.io/)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/)

## See Also

- [Performance Tuning Guide](performance-tuning.md)
- [API Reference](api-reference.md)
- [HTTP Server Documentation](http-api.md)
