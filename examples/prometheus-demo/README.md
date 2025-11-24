# LauraDB Prometheus/Grafana Integration Demo

This example demonstrates how to monitor LauraDB using Prometheus and visualize metrics in Grafana.

## Overview

LauraDB exports comprehensive metrics in Prometheus format via the `/_metrics` HTTP endpoint. This example shows:

1. How to configure Prometheus to scrape LauraDB metrics
2. How to create Grafana dashboards for monitoring
3. A complete monitoring setup with realistic workload simulation

## Quick Start

### 1. Start LauraDB with Workload Simulator

```bash
go run main.go
```

The server will start on http://localhost:8080 and begin generating realistic database traffic.

### 2. Access Metrics Endpoint

View raw Prometheus metrics:
```bash
curl http://localhost:8080/_metrics
```

Or open in browser: http://localhost:8080/_metrics

### 3. Set Up Prometheus

Install Prometheus:
```bash
# macOS
brew install prometheus

# Linux
wget https://github.com/prometheus/prometheus/releases/download/v2.45.0/prometheus-2.45.0.linux-amd64.tar.gz
tar xvfz prometheus-*.tar.gz
cd prometheus-*
```

Start Prometheus with the provided config:
```bash
prometheus --config.file=prometheus.yml
```

Access Prometheus UI: http://localhost:9090

### 4. Set Up Grafana

Install Grafana:
```bash
# macOS
brew install grafana

# Linux
sudo apt-get install -y grafana
```

Start Grafana:
```bash
# macOS
brew services start grafana

# Linux
sudo systemctl start grafana-server
```

Access Grafana UI: http://localhost:3000 (default credentials: admin/admin)

Configure Grafana:
1. Add Prometheus data source:
   - Go to Configuration → Data Sources → Add data source
   - Select Prometheus
   - URL: http://localhost:9090
   - Click "Save & Test"

2. Import LauraDB dashboard:
   - Go to Dashboards → Import
   - Upload `grafana-dashboard.json`
   - Select your Prometheus data source
   - Click Import

## Available Metrics

### Operation Metrics
- `laura_db_queries_total` - Total number of queries
- `laura_db_inserts_total` - Total number of inserts
- `laura_db_updates_total` - Total number of updates
- `laura_db_deletes_total` - Total number of deletes
- `laura_db_*_failed_total` - Failed operation counts

### Latency Metrics
- `laura_db_query_duration_seconds` - Query latency histogram
- `laura_db_query_duration_seconds_p50` - P50 percentile
- `laura_db_query_duration_seconds_p95` - P95 percentile
- `laura_db_query_duration_seconds_p99` - P99 percentile

### Cache Metrics
- `laura_db_cache_hits_total` - Total cache hits
- `laura_db_cache_misses_total` - Total cache misses
- `laura_db_cache_hit_rate` - Cache hit rate (0-1)

### Transaction Metrics
- `laura_db_transactions_started_total` - Transactions started
- `laura_db_transactions_committed_total` - Transactions committed
- `laura_db_transactions_aborted_total` - Transactions aborted

### Scan Metrics
- `laura_db_index_scans_total` - Index scan operations
- `laura_db_collection_scans_total` - Collection scan operations
- `laura_db_index_usage_rate` - Index usage rate (0-1)

### Connection Metrics
- `laura_db_active_connections` - Current active connections
- `laura_db_connections_total` - Total connections

### Resource Metrics
- `laura_db_memory_heap_bytes` - Heap memory usage
- `laura_db_memory_stack_bytes` - Stack memory usage
- `laura_db_goroutines` - Number of goroutines
- `laura_db_io_bytes_read_total` - Total bytes read
- `laura_db_io_bytes_written_total` - Total bytes written
- `laura_db_gc_runs_total` - Garbage collection runs

## Useful PromQL Queries

### Query Throughput
```promql
rate(laura_db_queries_total[1m])
```

### Error Rate
```promql
rate(laura_db_queries_failed_total[1m]) / rate(laura_db_queries_total[1m]) * 100
```

### Average Latency (milliseconds)
```promql
rate(laura_db_query_duration_nanoseconds_total[1m]) / rate(laura_db_queries_total[1m]) / 1000000
```

### Cache Hit Rate Percentage
```promql
laura_db_cache_hit_rate * 100
```

### Index Usage Percentage
```promql
laura_db_index_usage_rate * 100
```

### Memory Growth Rate
```promql
deriv(laura_db_memory_heap_bytes[5m])
```

### Transaction Commit Rate
```promql
rate(laura_db_transactions_committed_total[1m]) / rate(laura_db_transactions_started_total[1m]) * 100
```

## Alerting Examples

Create alerting rules in Prometheus (`alerts.yml`):

```yaml
groups:
  - name: lauradb_alerts
    interval: 30s
    rules:
      - alert: HighErrorRate
        expr: rate(laura_db_queries_failed_total[5m]) / rate(laura_db_queries_total[5m]) > 0.05
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High query error rate"
          description: "Query error rate is {{ $value | humanizePercentage }}"

      - alert: LowCacheHitRate
        expr: laura_db_cache_hit_rate < 0.5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Low cache hit rate"
          description: "Cache hit rate is {{ $value | humanizePercentage }}"

      - alert: HighP99Latency
        expr: laura_db_query_duration_seconds_p99 > 1.0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High P99 query latency"
          description: "P99 latency is {{ $value }}s"

      - alert: HighMemoryUsage
        expr: laura_db_memory_heap_bytes > 1e9
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage"
          description: "Heap memory usage is {{ $value | humanize1024 }}B"
```

## Dashboard Panels

The included Grafana dashboard provides:

1. **Query Rate** - Real-time query throughput
2. **Operation Success Rate** - Success rates for all operation types
3. **Query Latency Percentiles** - P50, P95, P99 latencies
4. **Cache Performance** - Hit rate and throughput
5. **Memory Usage** - Heap and stack memory trends
6. **Active Connections** - Connection count over time
7. **Transaction Metrics** - Commit and abort rates
8. **Index Usage** - Index vs collection scan ratio
9. **I/O Throughput** - Read and write bandwidth
10. **Goroutines & GC** - Runtime metrics

## Workload Simulator

The example includes a realistic workload simulator that generates:

- **60%** insert operations
- **30%** query operations
- **7%** update operations
- **3%** delete operations
- Transaction events (10% of operations)
- Cache hits/misses (70% hit rate)
- Index vs collection scans (80% index usage)
- I/O operations

This provides realistic data for exploring Prometheus/Grafana integration.

## Production Deployment

For production use:

1. **Secure the metrics endpoint** - Add authentication/authorization
2. **Configure retention** - Set appropriate Prometheus retention period
3. **Set up alerting** - Define alerts for critical metrics
4. **Use remote storage** - Consider Prometheus remote write for long-term storage
5. **Monitor Prometheus/Grafana** - Ensure monitoring infrastructure is also monitored
6. **Fine-tune scrape intervals** - Balance between resolution and resource usage

## References

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [PromQL Guide](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Grafana Dashboards](https://grafana.com/grafana/dashboards/)
