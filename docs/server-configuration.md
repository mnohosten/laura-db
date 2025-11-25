# LauraDB Server Configuration Guide

This guide covers all configuration options for the LauraDB HTTP server.

## Table of Contents

1. [Command-Line Flags](#command-line-flags)
2. [Storage Configuration](#storage-configuration)
3. [Performance Tuning](#performance-tuning)
4. [Network Configuration](#network-configuration)
5. [Security Configuration](#security-configuration)
6. [Examples](#examples)

---

## Command-Line Flags

The LauraDB server supports the following command-line flags:

### Storage & Performance

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-data-dir` | string | `./data` | Data directory for database storage (persistent disk storage) |
| `-buffer-size` | int | `1000` | Buffer pool size in pages (1 page = 4KB, default = ~4MB) |
| `-doc-cache` | int | `1000` | Document cache size per collection |

### Network

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-host` | string | `localhost` | Server host address |
| `-port` | int | `8080` | Server port |
| `-cors-origin` | string | `*` | CORS allowed origin |

### Security (TLS/SSL)

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-tls` | bool | `false` | Enable TLS/SSL |
| `-tls-cert` | string | `""` | Path to TLS certificate file |
| `-tls-key` | string | `""` | Path to TLS private key file |

---

## Storage Configuration

### Data Directory (`-data-dir`)

The data directory is where LauraDB stores all persistent data. This includes:

- **`data.db`**: Main database file containing all documents
- **`wal.log`**: Write-ahead log for crash recovery
- **`collections/`**: Collection metadata and catalog

**Important Notes**:
- Data in this directory persists across server restarts
- Ensure the directory has proper read/write permissions
- For production, use a dedicated disk or volume
- Consider backup strategies for this directory

**Example**:
```bash
# Development
./bin/laura-server -data-dir ./dev-data

# Production (dedicated volume)
./bin/laura-server -data-dir /var/lib/lauradb
```

### Buffer Pool Size (`-buffer-size`)

The buffer pool caches frequently accessed pages in memory to reduce disk I/O.

**Configuration Guidelines**:

| Dataset Size | Recommended Buffer Size | Memory Usage |
|--------------|------------------------|--------------|
| <10K documents | 100-500 pages | 0.4-2 MB |
| 10K-100K documents | 1000-2000 pages | 4-8 MB |
| 100K-1M documents | 5000-10000 pages | 20-40 MB |
| >1M documents | 10000+ pages | 40+ MB |

**Formula**: Memory = Buffer Size × 4KB

**Trade-offs**:
- ✅ **Larger buffer pool**: Faster queries, less disk I/O
- ⚠️ **Larger buffer pool**: More memory usage
- ✅ **Smaller buffer pool**: Lower memory footprint
- ⚠️ **Smaller buffer pool**: More disk I/O, slower queries

**Example**:
```bash
# Small deployment (embedded devices)
./bin/laura-server -buffer-size 500

# Medium deployment (typical server)
./bin/laura-server -buffer-size 5000

# Large deployment (database server)
./bin/laura-server -buffer-size 20000
```

### Document Cache (`-doc-cache`)

Per-collection LRU cache for frequently accessed documents.

**Default**: 1000 documents per collection

**When to increase**:
- High read workload on same documents
- Sufficient memory available
- Query cache hit rate is low

**When to decrease**:
- Memory constrained environments
- Large documents (MB-sized)
- Write-heavy workloads

**Example**:
```bash
# Memory-constrained environment
./bin/laura-server -doc-cache 500

# High-read workload
./bin/laura-server -doc-cache 5000
```

---

## Performance Tuning

### Memory Budget Calculation

Total memory usage formula:
```
Total Memory = (BufferSize × 4KB) + (Collections × DocCache × AvgDocSize) + Overhead
```

**Example**:
- Buffer Size: 5000 pages
- Collections: 10
- Doc Cache: 1000 documents
- Avg Document Size: 1KB
- Overhead: ~50MB (Go runtime, connections, etc.)

```
Total = (5000 × 4KB) + (10 × 1000 × 1KB) + 50MB
      = 20MB + 10MB + 50MB
      = 80MB
```

### Optimization Strategies

#### 1. **Read-Heavy Workloads**
```bash
./bin/laura-server \
  -buffer-size 10000 \
  -doc-cache 5000 \
  -data-dir /fast-ssd/lauradb
```

#### 2. **Write-Heavy Workloads**
```bash
./bin/laura-server \
  -buffer-size 5000 \
  -doc-cache 1000 \
  -data-dir /fast-ssd/lauradb
```

#### 3. **Memory-Constrained**
```bash
./bin/laura-server \
  -buffer-size 500 \
  -doc-cache 500 \
  -data-dir ./data
```

#### 4. **Large Dataset (>1M documents)**
```bash
./bin/laura-server \
  -buffer-size 20000 \
  -doc-cache 2000 \
  -data-dir /dedicated-disk/lauradb
```

---

## Network Configuration

### Basic Setup

```bash
# Listen on all interfaces
./bin/laura-server -host 0.0.0.0 -port 8080

# Localhost only (development)
./bin/laura-server -host localhost -port 8080

# Custom port
./bin/laura-server -port 9090
```

### CORS Configuration

```bash
# Allow all origins (default)
./bin/laura-server -cors-origin "*"

# Allow specific origin
./bin/laura-server -cors-origin "https://myapp.com"

# Multiple origins (not directly supported via flag)
# Use environment variables or configuration file
```

---

## Security Configuration

### TLS/SSL Setup

#### 1. Generate Self-Signed Certificate (Development)

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
```

#### 2. Start Server with TLS

```bash
./bin/laura-server \
  -tls \
  -tls-cert ./cert.pem \
  -tls-key ./key.pem
```

#### 3. Production Setup (Let's Encrypt)

```bash
# Using Let's Encrypt certificates
./bin/laura-server \
  -tls \
  -tls-cert /etc/letsencrypt/live/example.com/fullchain.pem \
  -tls-key /etc/letsencrypt/live/example.com/privkey.pem \
  -host 0.0.0.0 \
  -port 443
```

### Security Best Practices

1. **Always use TLS in production**
   ```bash
   ./bin/laura-server -tls -tls-cert /path/to/cert.pem -tls-key /path/to/key.pem
   ```

2. **Restrict CORS origins**
   ```bash
   ./bin/laura-server -cors-origin "https://yourdomain.com"
   ```

3. **Use firewall rules**
   - Allow only necessary ports
   - Restrict access to trusted IPs

4. **Secure data directory**
   ```bash
   chmod 700 /var/lib/lauradb
   chown lauradb:lauradb /var/lib/lauradb
   ```

5. **Regular backups**
   - Backup the entire data directory
   - Test restore procedures

---

## Examples

### Development Setup

```bash
./bin/laura-server \
  -data-dir ./dev-data \
  -port 8080 \
  -buffer-size 1000
```

### Production Setup

```bash
./bin/laura-server \
  -data-dir /var/lib/lauradb \
  -host 0.0.0.0 \
  -port 443 \
  -buffer-size 10000 \
  -doc-cache 5000 \
  -tls \
  -tls-cert /etc/letsencrypt/live/example.com/fullchain.pem \
  -tls-key /etc/letsencrypt/live/example.com/privkey.pem \
  -cors-origin "https://yourdomain.com"
```

### Docker Deployment

```bash
docker run -d \
  --name lauradb \
  -p 8080:8080 \
  -v /data/lauradb:/data \
  lauradb/lauradb:latest \
  -data-dir /data \
  -buffer-size 5000 \
  -doc-cache 2000 \
  -host 0.0.0.0
```

### Embedded Device (Raspberry Pi)

```bash
./bin/laura-server \
  -data-dir /home/pi/lauradb \
  -port 8080 \
  -buffer-size 250 \
  -doc-cache 250
```

### High-Performance Server

```bash
./bin/laura-server \
  -data-dir /nvme/lauradb \
  -host 0.0.0.0 \
  -port 8080 \
  -buffer-size 50000 \
  -doc-cache 10000
```

---

## Monitoring and Diagnostics

### Check Server Status

```bash
# Health check
curl http://localhost:8080/api/v1/status

# Metrics (Prometheus format)
curl http://localhost:8080/metrics
```

### Performance Monitoring

Monitor these metrics:
- **Buffer pool hit rate**: Higher is better (>90% ideal)
- **Query cache hit rate**: Higher is better (>80% ideal)
- **Disk I/O**: Lower is better
- **Memory usage**: Should stay within budget
- **Query latency**: Monitor p50, p95, p99

### Logging

Check server logs for:
- Startup configuration
- Connection attempts
- Query errors
- Performance warnings

---

## Troubleshooting

### High Memory Usage

**Symptoms**: Server using more memory than expected

**Solutions**:
1. Reduce buffer size: `-buffer-size 2000`
2. Reduce document cache: `-doc-cache 500`
3. Check for memory leaks in application code

### Slow Queries

**Symptoms**: Queries taking longer than expected

**Solutions**:
1. Increase buffer size: `-buffer-size 10000`
2. Create indexes on frequently queried fields
3. Use query `Explain()` to verify index usage
4. Move data directory to faster storage (SSD/NVMe)

### Disk Space Issues

**Symptoms**: Running out of disk space

**Solutions**:
1. Check WAL log size
2. Implement data retention policies
3. Archive old data
4. Increase disk capacity

### Connection Refused

**Symptoms**: Cannot connect to server

**Solutions**:
1. Check server is running: `ps aux | grep laura-server`
2. Verify port: `netstat -an | grep 8080`
3. Check firewall rules
4. Verify host binding (`-host 0.0.0.0` for all interfaces)

---

## Additional Resources

- [API Reference](./api-reference.md) - Complete API documentation
- [Performance Tuning](./performance-tuning.md) - Detailed optimization guide
- [HTTP API](./http-api.md) - HTTP endpoint documentation
- [Storage Engine](./storage-engine.md) - Storage internals
- [Docker Deployment](./docker-compose.md) - Container deployment guide

---

## Version

This documentation is for LauraDB v0.1.0.

Last updated: 2025-01-15
