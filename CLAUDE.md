# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## ✅ HTTP Server Status

**The HTTP server is NOW FULLY IMPLEMENTED!** All core REST API endpoints from `docs/http-api.md` are functional:
- ✅ `pkg/server` package - FULLY IMPLEMENTED
- ✅ `cmd/server/main.go` - FULLY IMPLEMENTED
- ✅ All API handlers - FULLY IMPLEMENTED
- ✅ Admin console UI - FULLY IMPLEMENTED ✨
- ✅ Integration tests - FULLY IMPLEMENTED

**All three modes are now functional:**
- ✅ Embedded/Library Mode (`pkg/database` - direct Go import)
- ✅ CLI Mode (`cmd/laura-cli` - interactive REPL)
- ✅ HTTP Server Mode (`cmd/server` - REST API server with web admin console) ✨

The HTTP server is production-ready with a complete web-based admin interface accessible at http://localhost:8080/

## Project Overview

LauraDB is an educational MongoDB-like document database written in Go. It demonstrates production-grade database internals including storage engines, MVCC transactions, B+ tree indexing, query processing, and crash recovery. The project is fully tested (695+ tests passing) with comprehensive features.

**Current Status**: Documents are persisted to disk using a slotted page structure. The database supports datasets larger than memory with LRU caching for performance. All data survives server restarts through the Write-Ahead Log (WAL) and disk-based storage.

## Development Commands

### Build
```bash
make build              # Build server, CLI and examples
make server            # Build HTTP server (./bin/laura-server)
make cli               # Build CLI only (./bin/laura-cli)
make examples          # Build all example programs
```

### Testing
```bash
make test              # Run all tests
make test-coverage     # Run tests with coverage summary
make coverage          # Generate detailed coverage report
make coverage-html     # Generate and open HTML coverage report

# Run specific package tests
go test ./pkg/database -v
go test ./pkg/query -v

# Run a single test
go test ./pkg/database -run TestInsertOne -v

# Run HTTP server integration tests
go test ./pkg/server -v
```

### Benchmarking
```bash
make bench             # Run benchmarks for database and index packages
make bench-all         # Run all benchmarks with detailed output
make bench-insert      # Run insert benchmarks
make bench-find        # Run find benchmarks
make bench-index       # Run index benchmarks
```

### Running
```bash
# HTTP Server Mode (now available!):
./bin/laura-server                              # Start server on localhost:8080
./bin/laura-server -port 9090                   # Start server on custom port
./bin/laura-server -port 8080 -data-dir ./data  # Full configuration

# CLI Mode:
./bin/laura-cli                    # Start CLI REPL with default data dir
./bin/laura-cli /path/to/data      # Start CLI with custom data dir
```

### Cleanup
```bash
make clean             # Remove all build artifacts, data directories, and coverage files
```

## Architecture Overview

LauraDB uses a layered architecture with clear separation of concerns:

### Layer 1: Storage Engine (`pkg/storage`)
- **Page-based storage**: Fixed 4KB pages managed by DiskManager
- **Buffer pool**: LRU cache for frequently accessed pages (default: 1000 pages)
- **Write-Ahead Log (WAL)**: Sequential log ensuring durability before applying changes
- **Crash recovery**: Automatic WAL replay on startup to restore consistency

Key files:
- `storage.go`: Main StorageEngine coordinating disk, buffer, and WAL
- `disk.go`: DiskManager for low-level page I/O
- `buffer.go`: BufferPool with LRU eviction
- `wal.go`: Write-ahead log implementation

### Layer 2: MVCC Transaction Manager (`pkg/mvcc`)
- **Snapshot isolation**: Each transaction sees a consistent view at a point in time
- **Version chains**: Multiple document versions coexist for concurrent readers
- **Garbage collection**: Old versions cleaned when no longer needed by any transaction
- **Non-blocking reads**: Readers never block writers, writers never block readers

Key concepts:
- Each transaction gets a unique `TxnID` and `ReadVersion`
- `VersionedValue` tracks created/deleted transaction IDs
- `VersionStore` maintains version chains per document

Key files:
- `transaction.go`: Transaction and TransactionManager
- `version.go`: VersionStore managing version chains

### Layer 3: Document Format (`pkg/document`)
- **BSON-like encoding**: Efficient binary serialization
- **ObjectID generation**: MongoDB-compatible 12-byte IDs (timestamp + machine + process + counter)
- **Type system**: String, Number, Boolean, Array, Document, ObjectID, Null
- **Nested documents**: Full support for complex hierarchical structures

Important: ObjectID generation uses process-unique bytes initialized once at startup to prevent duplicates during rapid inserts.

### Layer 4: Indexing (`pkg/index`)
Multiple index types supported:

#### B+ Tree Index (`btree.go`)
- Self-balancing tree with data only in leaf nodes
- Leaf nodes linked for efficient range scans
- Default order: 32 (max 32 keys per node)
- Supports unique and non-unique indexes
- **Critical**: `lastSplitKey` field temporarily stores separator keys during node splits

#### Compound Indexes
- Multi-field indexes (e.g., `{city: 1, age: 1}`)
- Uses composite keys for B+ tree storage
- Query optimizer understands index prefix matching

#### Text Indexes (`text_index.go`)
- Inverted index with tokenization and stemming (Porter stemmer)
- BM25 relevance scoring
- Stop word filtering
- Multi-field text search

#### Geospatial Indexes (`geo_index.go`)
- **2d**: Planar coordinates (Euclidean distance)
- **2dsphere**: Geographic coordinates (Haversine distance for Earth surface)
- R-tree structure for spatial queries
- Proximity, polygon containment, and bounding box queries

#### TTL Indexes
- Automatic document expiration based on timestamp fields
- Background cleanup goroutine checks every 60 seconds
- Documents deleted when `field + duration < now()`

#### Partial Indexes
- Index only documents matching a filter expression
- Saves space and improves performance for selective queries

### Layer 5: Query Engine (`pkg/query`)

#### Query Parser (`query.go`)
Converts MongoDB-like query syntax to executable operators:
- Comparison: `$eq`, `$ne`, `$gt`, `$gte`, `$lt`, `$lte`
- Logical: `$and`, `$or`, `$not`
- Array: `$in`, `$nin`, `$all`, `$elemMatch`, `$size`
- Element: `$exists`, `$type`
- Evaluation: `$regex` (full regex support)

#### Query Optimizer (`optimizer.go`)
- **Statistics-based**: Tracks cardinality, selectivity, min/max values per index
- **Cost estimation**: Calculates scan cost for each available index
- **Index selection**: Chooses lowest-cost index or full collection scan
- **Covered queries**: Detects when query can be satisfied entirely from index (no document fetch)

#### Query Executor (`executor.go`)
- Executes optimized query plans
- Handles index lookups vs. collection scans
- Applies sort, skip, limit operations
- Projection support (field inclusion/exclusion)

### Layer 6: Query Cache (`pkg/cache`)
- LRU eviction policy (default: 1000 entries)
- TTL-based expiration (default: 5 minutes)
- Thread-safe with RWMutex
- Invalidated on any write operation (Insert/Update/Delete)
- Performance: 96x faster for cached queries

### Layer 7: Aggregation Pipeline (`pkg/aggregation`)
Stage-based document processing:
- `$match`: Filter documents (can push down to query engine)
- `$group`: Group by key with aggregation operators
- `$project`: Field selection and transformation
- `$sort`: Order results
- `$limit`: Limit result count
- `$skip`: Skip initial results

Aggregation operators: `$sum`, `$avg`, `$min`, `$max`, `$count`, `$push`

### Layer 8: Database & Collections (`pkg/database`)

#### Database (`database.go`)
- Manages collections and shared resources (storage, transaction manager)
- TTL cleanup goroutine lifecycle
- Collection creation and lifecycle

#### Collection (`collection.go`)
- **Document storage**:
  - **Disk-based storage**: Documents persisted using slotted page structure (`DocumentStore`)
  - **Location tracking**: Document IDs mapped to disk locations (PageID + SlotID)
  - **Caching**: LRU document cache for frequently accessed documents
  - **Persistence**: All documents survive server restarts via WAL and disk pages
- Index management (B+ tree, text, geo, TTL)
- CRUD operations: InsertOne, InsertMany, Find, FindOne, UpdateOne, UpdateMany, DeleteOne, DeleteMany
- Query cache per collection
- Automatic index maintenance on all write operations

**Critical implementation notes:**
- All numeric values in Go should use `int64` for consistency (BSON numeric type)
- Partial indexes require filter matching check before insertion
- Compound indexes extract composite keys from multiple fields
- Default `_id_` index is automatically created for every collection

**Disk storage configuration:**
- **Buffer pool size**: Default 1000 pages (4MB), configurable via `StorageEngine`
- **Document cache size**: Default per-collection LRU cache, size configurable in `NewDocumentStore`
- **Page size**: Fixed 4KB (4096 bytes) matching typical OS page size
- **WAL**: Automatic write-ahead logging for durability, synced on commit
- **Data directory**: All data files stored in configured directory (`.laura-db` by default)

## Code Patterns and Conventions

### Type Handling
Always use `int64` for numeric values in documents and tests to match BSON encoding:
```go
// Correct
doc := map[string]interface{}{
    "age": int64(30),
    "count": int64(100),
}

// Incorrect - will cause type assertion panics
doc := map[string]interface{}{
    "age": 30,  // untyped int
}
```

### ObjectID Generation
ObjectID uses process-unique bytes initialized at startup. Never regenerate these during runtime:
```go
// Correct - ObjectID handles this internally
id := document.NewObjectID()

// When testing rapid inserts
for i := 0; i < 1000; i++ {
    id := document.NewObjectID()  // Safe, no duplicates
}
```

### Index Operations
When working with indexes:
- Check partial filter match before inserting into partial indexes
- For compound indexes, extract composite keys from all fields
- Handle missing fields appropriately (skip for non-unique, error for unique)

### MVCC Transactions
Transactions provide snapshot isolation:
```go
txn := txnMgr.Begin()
// Reads see snapshot at txn.ReadVersion
// Writes go to txn.WriteSet
txn.Commit()  // or txn.Abort()
```

### Query Optimizer Integration
When modifying Collection operations:
- Update index statistics after writes (UpdateStats)
- Clear query cache on any write operation
- Let optimizer choose execution plan (don't hardcode index selection)

## Testing Philosophy

- **Unit tests**: Test individual components in isolation
- **Integration tests**: Test component interactions (especially in `pkg/database`)
- **Coverage**: Aim for 100% (currently at 76/76 tests passing)
- **Benchmarks**: Performance-critical paths (inserts, finds, index operations)

When adding features:
1. Write failing tests first
2. Implement minimal code to pass tests
3. Add benchmarks for performance-sensitive code
4. Update relevant documentation in `docs/`

## Common Development Tasks

### Adding a New Query Operator
1. Add operator constant in `pkg/query/query.go`
2. Implement evaluation logic in operator evaluation switch
3. Add test cases in `pkg/query/query_test.go`
4. Update `docs/query-engine.md`

### Adding a New Index Type
1. Define index structure in `pkg/index/`
2. Implement Insert, Delete, Search methods
3. Add index type to Collection index maps
4. Add creation method to Collection
5. Update index maintenance in InsertOne, UpdateOne, DeleteOne
6. Add comprehensive tests
7. Document in `docs/indexing.md`

### Adding a New Aggregation Stage
1. Define stage parser in `pkg/aggregation/pipeline.go`
2. Implement stage executor
3. Add to pipeline execution logic
4. Add tests in `pkg/aggregation/aggregation_test.go`
5. Update `docs/aggregation.md`

## Important Files and Entry Points

- `pkg/database/database.go`: Main database entry point (Open function)
- `pkg/database/collection.go`: Core CRUD operations
- `pkg/query/optimizer.go`: Query optimization logic
- `pkg/index/btree.go`: B+ tree implementation (complex split logic)
- `pkg/storage/storage.go`: Storage engine coordination
- `pkg/mvcc/transaction.go`: Transaction lifecycle

## Known Limitations

### Architectural Limitations
- **Single database**: Only one database instance per data directory
- **No authentication in embedded mode**: Auth available in HTTP server mode only
- **Single-node disk storage**: Replication/sharding implemented at application level, not storage level
- **MVCC on disk**: MVCC version chains currently in-memory; disk persistence planned for future enhancement

## Module Information

- Module path: `github.com/mnohosten/laura-db`
- Go version: 1.25.4
- Dependencies: `github.com/go-chi/chi/v5` v5.2.3 (for future HTTP server)

## References to External Documentation

When implementing features, consult these docs for detailed specifications:
- `docs/storage-engine.md`: WAL, buffer pool, crash recovery details
- `docs/mvcc.md`: MVCC algorithm, version chains, garbage collection
- `docs/indexing.md`: B+ tree algorithms, index types
- `docs/query-engine.md`: Query operators, optimizer, execution
- `docs/text-search.md`: Text indexing, BM25 scoring
- `docs/geospatial.md`: Geo index types, distance calculations
- `docs/ttl-indexes.md`: TTL cleanup mechanics
- `docs/statistics-optimization.md`: Optimizer statistics
