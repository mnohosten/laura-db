# LauraDB - Development Roadmap

This document outlines the next phases of development for LauraDB. For completed features, see [docs/CHANGELOG.md](docs/CHANGELOG.md).

## Current Status

LauraDB is a production-grade educational MongoDB-like document database with comprehensive features.

**Project Statistics**:
- **Lines of Code**: ~36,900+ (Go)
- **Test Cases**: 695+
- **Test Coverage**: 72.9%
- **Packages**: 21 core packages
- **Completion**: ~97% of original planned features

**Performance Benchmarks** (Apple M4 Max):
| Operation | Rate | Latency |
|-----------|------|---------|
| Single Insert | ~286K docs/sec | 3.5 us/op |
| Bulk Insert (100 docs) | ~304K docs/sec | - |
| B+ Tree Search | ~99M ops/sec | 10 ns/op |
| Hash Routing | ~45M ops/sec | 22 ns/op |
| Parallel Sharded Insert (3 shards) | ~384K docs/sec | 2.6 us/op |

---

## Phase 1: Network Transport Layer (HIGH PRIORITY)

**Goal**: Enable true distributed deployment with multiple LauraDB nodes communicating over the network.

**Current State**: Replication and sharding work in-process only (all "nodes" run in the same Go process).

**Impact**: Transforms LauraDB from "educational database with clustering algorithms" to "actual distributed database."

### 1.1 gRPC Transport Layer (~1500-2000 LOC)
- [ ] Define protobuf schemas for cluster communication
  - Node registration and discovery messages
  - Heartbeat and health check messages
  - Replication messages (oplog entries, sync requests)
  - Shard routing messages (query forwarding, results)
  - Two-phase commit messages (prepare, commit, abort)
- [ ] Implement gRPC server for each LauraDB node
  - Listen for incoming cluster connections
  - Handle concurrent requests with connection pooling
  - TLS/mTLS support for secure communication
- [ ] Implement gRPC client for node-to-node communication
  - Connection management with automatic reconnection
  - Request timeout and retry logic
  - Load balancing for multi-node scenarios

### 1.2 Node Discovery and Membership (~500-800 LOC)
- [ ] Implement cluster membership protocol
  - Static configuration (list of seed nodes)
  - Dynamic discovery (gossip protocol or DNS-based)
- [ ] Node registration and deregistration
  - Graceful shutdown with member removal
  - Failure detection and automatic removal
- [ ] Cluster topology management
  - Track active nodes and their roles
  - Propagate topology changes to all members

### 1.3 Distributed Shard Routing (~800-1000 LOC)
- [ ] Remote shard operations
  - Forward queries to appropriate shard nodes
  - Aggregate results from multiple shards
  - Handle shard failures with automatic failover
- [ ] Config server replication
  - Replicate chunk metadata across config servers
  - Consistent reads for routing decisions
- [ ] Cross-shard query execution
  - Scatter-gather for non-shard-key queries
  - Parallel query execution across shards

### 1.4 Distributed Replication (~800-1000 LOC)
- [ ] Network-based oplog replication
  - Stream oplog entries to secondary nodes
  - Handle network partitions and reconnection
  - Ensure consistency with acknowledge mechanisms
- [ ] Remote election protocol
  - Conduct elections across network
  - Handle split-brain scenarios
  - Implement fencing for old primaries
- [ ] Write concern over network
  - Wait for acknowledgments from remote secondaries
  - Timeout handling for unresponsive nodes

### 1.5 Testing and Documentation (~500 LOC)
- [ ] Integration tests for distributed scenarios
  - Multi-node cluster setup and teardown
  - Failover and recovery tests
  - Network partition simulation
- [ ] Performance benchmarks
  - Latency overhead from network transport
  - Throughput with different cluster sizes
- [ ] Documentation
  - Cluster setup guide
  - Network configuration options
  - Troubleshooting distributed issues

**Estimated Total**: ~4,000-5,500 LOC

**Dependencies**: None (builds on existing in-process implementations)

---

## Phase 2: Production Reliability (HIGH PRIORITY)

**Goal**: Make LauraDB suitable for production workloads with crash recovery and data integrity guarantees.

### 2.1 Checkpoint Manager (~500-800 LOC)
- [ ] Create CheckpointManager in pkg/storage
  - Coordinate buffer pool flush with WAL checkpoint
  - Track last checkpoint LSN (Log Sequence Number)
  - Implement checkpoint barrier for consistent state
- [ ] Background checkpoint goroutine
  - Configurable checkpoint interval (default: 5 minutes)
  - Force checkpoint on graceful shutdown
  - Checkpoint on WAL size threshold
- [ ] Checkpoint recovery
  - Skip already-checkpointed WAL entries on recovery
  - Reduce recovery time for long-running databases
  - Validate checkpoint integrity

### 2.2 MVCC Disk Persistence (~3000-4000 LOC)
- [ ] Design version chain page format
  - Store version chains on disk (similar to PostgreSQL heap tuples)
  - Link versions via page references
  - Handle version overflow to separate pages
- [ ] Transaction state persistence
  - Persist active/committed/aborted transaction states
  - Transaction log integration with WAL
  - Recovery of in-flight transactions
- [ ] Background garbage collection (vacuum)
  - Identify and remove old versions no longer needed
  - Reclaim disk space from deleted versions
  - Configurable vacuum schedule and thresholds

### 2.3 Text Index Disk Persistence (~1000-1500 LOC)
- [ ] Refactor InvertedIndex for disk storage
  - Term dictionary as disk-based B+ tree
  - Posting lists persistence on separate pages
  - Lazy loading of posting lists
- [ ] Index compaction and optimization
  - Merge posting lists for efficiency
  - Remove deleted document references

### 2.4 Geospatial Index Disk Persistence (~1000-1500 LOC)
- [ ] R-tree node serialization
  - Design R-tree page format
  - Node caching similar to B+ tree
- [ ] Disk I/O operations for R-tree
  - LoadNode, WriteNode, FlushDirtyNodes
  - Handle node splits on disk

### 2.5 Bug Fixes
- [ ] Fix TestCurrentDateWithDate panic (time.Time vs int64 conversion)
- [ ] Fix GraphQL operator syntax issues
- [ ] Fix BenchmarkSlaveApplyEntry duplicate document issue

**Estimated Total**: ~5,500-7,800 LOC

---

## Phase 3: Operational Features (MEDIUM PRIORITY)

**Goal**: Add features needed for operating LauraDB in production environments.

### 3.1 Cluster Authentication (~800-1000 LOC)
- [ ] Node-to-node authentication
  - Shared secret or certificate-based auth
  - Rotate credentials without downtime
- [ ] Cluster admin roles
  - Dedicated roles for cluster management
  - Audit logging for cluster operations
- [ ] Secure replication channels
  - Encrypt oplog data in transit
  - Verify message integrity

### 3.2 Observability Improvements (~600-800 LOC)
- [ ] Distributed tracing (OpenTelemetry)
  - Trace requests across nodes
  - Export to Jaeger/Zipkin
- [ ] Per-shard metrics
  - Shard-specific operation counts
  - Data distribution metrics
- [ ] Replication lag dashboards
  - Real-time lag visualization
  - Alerting on excessive lag

### 3.3 Backup/Restore Enhancements (~800-1000 LOC)
- [ ] Point-in-time recovery (PITR)
  - Restore to any point using WAL
  - Configurable retention period
- [ ] Online backup without locking
  - Consistent snapshot while accepting writes
  - Incremental backup support
- [ ] Cross-shard consistent snapshots
  - Coordinate backups across shards
  - Ensure transactional consistency

**Estimated Total**: ~2,200-2,800 LOC

---

## Phase 4: Performance Optimizations (MEDIUM PRIORITY)

**Goal**: Optimize performance for large-scale deployments.

### 4.1 Query Prefetching (~500-700 LOC)
- [ ] Sequential scan prefetching
  - Predict next pages during full scans
  - Asynchronous prefetch into buffer pool
- [ ] Index range scan optimization
  - Prefetch B+ tree leaf pages during range scans
  - Adaptive prefetch based on query patterns

### 4.2 Write Path Optimization (~600-800 LOC)
- [ ] Batch WAL writes
  - Group multiple operations into single WAL entry
  - Configurable batch size and timeout
- [ ] Async commit option
  - Acknowledge before WAL sync for higher throughput
  - Documented durability tradeoffs

### 4.3 Read Path Optimization (~500-700 LOC)
- [ ] Bloom filters for collections
  - Skip disk reads for non-existent keys
  - Configurable false positive rate
- [ ] Compressed page cache
  - Keep compressed pages in memory
  - Decompress on demand

**Estimated Total**: ~1,600-2,200 LOC

---

## Phase 5: Client Libraries (LOWER PRIORITY)

**Goal**: Provide official, well-tested client libraries for popular languages.

### 5.1 Go Client Improvements (~300-500 LOC)
- [ ] Cluster-aware connection pool
  - Automatic discovery of cluster topology
  - Failover on node failure
- [ ] Read preference routing
  - Route reads to secondaries when configured
  - Latency-based routing

### 5.2 Node.js Client Implementation (~1000-1500 LOC)
- [ ] Native implementation (not just HTTP wrapper)
  - gRPC or custom binary protocol
  - Connection pooling
- [ ] Publish to npm
  - Comprehensive documentation
  - TypeScript type definitions

### 5.3 Python Client Implementation (~1000-1500 LOC)
- [ ] Native implementation
  - Async support (asyncio)
  - Connection pooling
- [ ] Publish to PyPI
  - Comprehensive documentation
  - Type hints

### 5.4 Java Client Implementation (~1500-2000 LOC)
- [ ] Native implementation
  - Reactive streams support
  - Connection pooling
- [ ] Publish to Maven Central
  - Comprehensive documentation
  - Javadoc

**Estimated Total**: ~3,800-5,500 LOC

---

## Phase 6: Additional Features (FUTURE)

**Goal**: Features that would be nice to have but are not critical.

### 6.1 Video Tutorials
- [ ] Getting started video
- [ ] Architecture overview video
- [ ] Query optimization video
- [ ] Deployment and operations video

### 6.2 Multi-tenancy Support
- [ ] Database-level isolation
- [ ] Resource quotas per tenant
- [ ] Tenant-specific encryption keys

### 6.3 Time-series Optimizations
- [ ] Time-based partitioning
- [ ] Automatic data aging/archival
- [ ] Specialized time-series indexes

---

## Summary

### Priority Order

| Phase | Priority | Estimated LOC | Impact |
|-------|----------|---------------|--------|
| Phase 1: Network Transport | HIGH | 4,000-5,500 | Enables true distributed deployment |
| Phase 2: Production Reliability | HIGH | 5,500-7,800 | Makes database production-ready |
| Phase 3: Operational Features | MEDIUM | 2,200-2,800 | Improves operability |
| Phase 4: Performance Optimizations | MEDIUM | 1,600-2,200 | Improves performance |
| Phase 5: Client Libraries | LOWER | 3,800-5,500 | Improves developer experience |
| Phase 6: Additional Features | FUTURE | TBD | Nice to have |

### Recommended Starting Point

**Phase 1: Network Transport Layer** is the recommended starting point because:

1. **Highest Impact**: Transforms LauraDB from in-process demo to real distributed database
2. **Foundation for Others**: Phase 2 and 3 features benefit from distributed capability
3. **Clear Scope**: Well-defined deliverables with existing algorithms to build on
4. **Educational Value**: Demonstrates real-world distributed systems concepts

### Current Limitations (to be addressed)

- **In-process clustering**: Replication/sharding work in-process only (Phase 1 fixes this)
- **MVCC on disk**: Version chains currently in-memory (Phase 2 fixes this)
- **Text/Geo indexes on disk**: Currently in-memory (Phase 2 fixes this)
- **No PITR**: Point-in-time recovery not available (Phase 3 fixes this)

---

## Notes

### Architecture Decisions
- Network transport will use gRPC for performance and type safety
- Protobuf for message serialization (language-agnostic)
- TLS by default for all cluster communication

### Testing Strategy
- Each phase includes comprehensive tests
- Integration tests for distributed scenarios
- Chaos engineering tests for failure scenarios

### Documentation
- Each phase includes documentation updates
- API documentation for new features
- Deployment guides for new capabilities

---

For completed features and detailed development history, see [docs/CHANGELOG.md](docs/CHANGELOG.md).
