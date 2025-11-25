# LauraDB Architecture Deep-Dive

## Table of Contents

1. [Overview](#overview)
2. [System Architecture](#system-architecture)
3. [Layered Architecture](#layered-architecture)
4. [Data Flow](#data-flow)
5. [Component Interactions](#component-interactions)
6. [Concurrency Model](#concurrency-model)
7. [Performance Characteristics](#performance-characteristics)
8. [Design Decisions](#design-decisions)
9. [Comparison with Other Databases](#comparison-with-other-databases)

## Overview

LauraDB is an educational MongoDB-like document database written in Go, designed to demonstrate production-grade database internals. It implements a complete database system from the ground up, including:

- **Storage Layer**: Page-based storage with WAL and buffer pool
- **Transaction Layer**: MVCC with snapshot isolation
- **Indexing Layer**: Multiple index types (B+ tree, text, geospatial, TTL)
- **Query Layer**: MongoDB-compatible query language with optimization
- **API Layer**: Embedded library, CLI, and HTTP server modes

**Design Philosophy**: Educational clarity without sacrificing production-grade correctness. Every component is implemented to demonstrate real database concepts, fully tested, and documented.

## System Architecture

### High-Level Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         Application Layer                        │
│  ┌───────────┐  ┌──────────────┐  ┌────────────────────────┐   │
│  │ Go Client │  │  HTTP Server │  │  CLI (laura-cli)       │   │
│  │ (Embedded)│  │  (REST API)  │  │  (Interactive REPL)    │   │
│  └─────┬─────┘  └──────┬───────┘  └──────────┬─────────────┘   │
└────────┼────────────────┼──────────────────────┼─────────────────┘
         │                │                      │
         └────────────────┴──────────────────────┘
                          │
┌─────────────────────────┼─────────────────────────────────────────┐
│                         ▼                                          │
│                   Database API (pkg/database)                      │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │ Database, Collection, Session, Index Management          │    │
│  └────────────────────┬──────────────────────────────────────┘   │
└───────────────────────┼──────────────────────────────────────────┘
                        │
         ┌──────────────┼──────────────┐
         │              │               │
         ▼              ▼               ▼
┌─────────────┐  ┌────────────┐  ┌──────────────┐
│   Query     │  │ Aggregation│  │  Cache       │
│   Engine    │  │ Pipeline   │  │  (LRU)       │
│ (Optimizer) │  │ (Stages)   │  │              │
└──────┬──────┘  └─────┬──────┘  └──────────────┘
       │               │
       └───────┬───────┘
               │
┌──────────────┼────────────────────────────────────────────────────┐
│              ▼                                                     │
│        Index Layer (pkg/index)                                     │
│  ┌────────────┐  ┌──────────┐  ┌─────────┐  ┌────────────────┐  │
│  │  B+ Tree   │  │   Text   │  │   Geo   │  │  Compound/TTL  │  │
│  │   Index    │  │  Search  │  │  Index  │  │  Partial Index │  │
│  └────────────┘  └──────────┘  └─────────┘  └────────────────┘  │
└────────────────────────────────────────────────────────────────────┘
               │
┌──────────────┼────────────────────────────────────────────────────┐
│              ▼                                                     │
│        MVCC Transaction Manager (pkg/mvcc)                         │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │ Transaction Lifecycle, Version Store, Snapshot Isolation │    │
│  │ Optimistic Concurrency Control, Garbage Collection       │    │
│  └────────────────────────────────────────────────────────────┘   │
└────────────────────────────────────────────────────────────────────┘
               │
┌──────────────┼────────────────────────────────────────────────────┐
│              ▼                                                     │
│        Storage Engine (pkg/storage)                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │ Buffer Pool  │  │     WAL      │  │   Disk Manager       │   │
│  │ (LRU Cache)  │  │ (Durability) │  │ (Page I/O, Mmap)     │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
└────────────────────────────────────────────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────────────────────────┐
│                      File System                                  │
│  ┌──────────┐  ┌─────────┐  ┌──────────────────────────────┐   │
│  │ data.db  │  │ wal.log │  │ indexes/*.idx, collections/* │   │
│  └──────────┘  └─────────┘  └──────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
```

## Layered Architecture

LauraDB is organized into distinct layers, each with clear responsibilities:

### Layer 1: Storage Engine (Foundation)

**Location**: `pkg/storage`

**Responsibility**: Reliable, efficient data persistence

**Components**:
- **DiskManager**: Manages file I/O operations for pages
  - Standard implementation: `syscall.Read/Write`
  - Memory-mapped implementation: `MmapDiskManager` (1.44x faster reads)
- **Page**: Fixed 4KB blocks (matching OS page size for optimal I/O)
- **BufferPool**: LRU cache of pages in memory (default: 1000 pages = 4MB)
  - Two-phase locking: read lock → upgrade to write lock when needed
  - Eviction policy: Least Recently Used (LRU)
- **Write-Ahead Log (WAL)**: Sequential append-only log for durability
  - Log format: `[RecordSize][PageID][PageData]`
  - Crash recovery: Replay WAL on startup

**Key Characteristics**:
- **Durability**: WAL ensures no data loss even on crash
- **Performance**: Buffer pool reduces disk I/O by ~96% (cached page fetch: 239ns)
- **Correctness**: Page-level atomicity guarantees

**Reference Documentation**: [storage-engine.md](storage-engine.md)

### Layer 2: MVCC Transaction Manager

**Location**: `pkg/mvcc`

**Responsibility**: Multi-version concurrency control with snapshot isolation

**Components**:
- **TransactionManager**: Coordinates transaction lifecycle
  - Assigns unique transaction IDs (monotonically increasing)
  - Manages active transaction registry
- **Transaction**: Represents single unit of work
  - `ReadVersion`: Snapshot timestamp
  - `WriteSet`: Modified documents (key → VersionedValue)
  - `ReadSet`: Read keys for conflict detection
- **VersionStore**: Maintains version chains per document
  - Each key has linked list of versions (most recent first)
  - Version metadata: `CreatedBy`, `DeletedBy` transaction IDs

**Concurrency Model**:
- **Snapshot Isolation**: Each transaction sees consistent database snapshot
- **Non-blocking reads**: Readers never block writers (and vice versa)
- **First-Committer-Wins**: Write-write conflicts detected at commit time
- **Garbage Collection**: Old versions pruned when no longer visible

**Example**:
```
Timeline:
T1 (ReadVersion=100): Start → Read X → Commit
T2 (ReadVersion=101): Start → Write X → Commit
T3 (ReadVersion=102): Start → Read X (sees T2's write) → Commit

Version chain for key X:
[V2: value=20, CreatedBy=T2] → [V1: value=10, CreatedBy=T1] → null
```

**Reference Documentation**: [mvcc.md](mvcc.md)

### Layer 3: Document Format

**Location**: `pkg/document`

**Responsibility**: Document encoding, ObjectID generation, type system

**Components**:
- **Document**: `map[string]interface{}` with BSON-like encoding
- **ObjectID**: 12-byte globally unique identifier
  - Structure: `[4-byte timestamp][5-byte random][3-byte counter]`
  - Process-unique initialization prevents duplicates
- **Type System**: String, Number (int64), Boolean, Array, Document, ObjectID, Null, Time

**Encoding**:
```
Document: { "name": "Alice", "age": 30 }
           ↓
BSON: [type:String][len:5][name][type:Number][value:30][age]...
```

**Reference Documentation**: [document-format.md](document-format.md)

### Layer 4: Indexing

**Location**: `pkg/index`

**Responsibility**: Fast document lookup without full collection scans

**Index Types**:

#### 1. B+ Tree Index (Primary)
- **Structure**: Self-balancing tree with data in leaf nodes
- **Order**: 32 (max 32 keys per node)
- **Operations**: O(log n) insert/delete/search
- **Features**: Range scans (leaf nodes linked), unique constraints
- **Use Case**: Primary `_id` index, single-field indexes

#### 2. Compound Index
- **Structure**: B+ tree with composite keys
- **Key Format**: Lexicographic ordering `[field1][field2][field3]...`
- **Query Optimization**: Supports prefix matching
- **Example**: Index on `{city: 1, age: 1}` accelerates queries on `city` or `city + age`

#### 3. Text Index
- **Structure**: Inverted index (term → [doc1, doc2, ...])
- **Features**: Tokenization, stop word filtering, Porter stemming
- **Scoring**: BM25 relevance algorithm
- **Performance**: 1.9x faster than regex-based search

#### 4. Geospatial Index
- **2d**: Planar coordinates (Euclidean distance)
- **2dsphere**: Spherical coordinates (Haversine distance)
- **Structure**: R-tree for spatial partitioning
- **Queries**: `$near` (proximity), `$geoWithin` (containment), `$geoIntersects` (bounding box)

#### 5. TTL Index
- **Purpose**: Automatic document expiration
- **Mechanism**: Background goroutine checks every 60 seconds
- **Condition**: Delete documents where `field + duration < now()`

#### 6. Partial Index
- **Purpose**: Index only documents matching filter expression
- **Benefit**: Reduced memory/disk usage for selective indexing
- **Example**: Index on `{status: "active"}` only indexes active documents

**Index Statistics** (for query optimization):
- Cardinality: Number of unique values
- Selectivity: Cardinality / Total documents
- Min/Max: Range boundaries
- Histogram: Value distribution (10-100 buckets)

**Reference Documentation**: [indexing.md](indexing.md), [text-search.md](text-search.md), [geospatial.md](geospatial.md), [ttl-indexes.md](ttl-indexes.md)

### Layer 5: Query Engine

**Location**: `pkg/query`

**Responsibility**: Parse, optimize, and execute queries

**Components**:

#### Query Parser
Converts MongoDB-like queries to executable operators:
```go
// Input
filter := map[string]interface{}{
    "age": map[string]interface{}{"$gte": 18},
    "city": "New York",
}

// Parsed to
AND(
    GTE(age, 18),
    EQ(city, "New York")
)
```

**Supported Operators**:
- **Comparison**: `$eq`, `$ne`, `$gt`, `$gte`, `$lt`, `$lte`
- **Logical**: `$and`, `$or`, `$not`
- **Array**: `$in`, `$nin`, `$all`, `$elemMatch`, `$size`
- **Element**: `$exists`, `$type`
- **Evaluation**: `$regex`

#### Query Optimizer
**Goal**: Choose lowest-cost execution plan

**Process**:
1. **Index Enumeration**: List all indexes applicable to query
2. **Cost Estimation**: Calculate scan cost for each index
   - Formula: `Cost = NumKeys * IndexLookupCost + NumDocs * FetchCost`
   - Covered queries: Zero fetch cost (no document retrieval)
3. **Selectivity Estimation**:
   - **Equality**: `1 / cardinality`
   - **Range**: Histogram-based estimation (166ns per estimate)
   - **IN operator**: Sum of selectivities for each value
4. **Plan Selection**: Choose index with minimum cost

**Optimizations**:
- **Covered Queries**: Query satisfied entirely from index (2.2x faster)
- **Index Intersection**: Use multiple indexes simultaneously
- **Parallel Execution**: Multi-core query processing (4.36x speedup on 50k+ docs)

#### Query Executor
Executes optimized plan:
1. **Index Scan**: Retrieve matching document IDs from index
2. **Document Fetch**: Load full documents from storage (skip if covered)
3. **Filter**: Apply remaining predicates not handled by index
4. **Sort**: Order results (if not index-ordered)
5. **Projection**: Select requested fields
6. **Pagination**: Apply skip/limit

**Reference Documentation**: [query-engine.md](query-engine.md), [statistics-optimization.md](statistics-optimization.md)

### Layer 6: Query Cache

**Location**: `pkg/cache`

**Responsibility**: Cache frequently executed queries

**Implementation**:
- **Eviction**: LRU (Least Recently Used) with 1000 entry capacity
- **TTL**: 5-minute expiration
- **Invalidation**: Cleared on any write operation (Insert/Update/Delete)
- **Thread Safety**: RWMutex for concurrent access

**Performance**: 96x faster for cached queries (328µs → 3.4µs)

**Cache Key**: Hash of filter, projection, sort, skip, limit

### Layer 7: Aggregation Pipeline

**Location**: `pkg/aggregation`

**Responsibility**: Complex document transformations and analytics

**Architecture**:
```
Input Documents
      ↓
   $match (filter)
      ↓
   $project (transform)
      ↓
   $group (aggregate)
      ↓
   $sort (order)
      ↓
   $limit (pagination)
      ↓
Output Documents
```

**Stages**:
- **$match**: Filter documents (can push down to query engine for index use)
- **$group**: Group by expression with accumulators (`$sum`, `$avg`, `$min`, `$max`, `$push`)
- **$project**: Field selection, computed fields, renaming
- **$sort**: Order by fields
- **$limit / $skip**: Pagination

**Optimization**: `$match` stage pushed to beginning to reduce pipeline input

**Reference Documentation**: [aggregation.md](aggregation.md)

### Layer 8: Database & Collections

**Location**: `pkg/database`

**Responsibility**: High-level API, collection management, index coordination

**Components**:

#### Database
- Manages storage engine and transaction manager
- Creates and drops collections
- Coordinates TTL cleanup (background goroutine)
- Provides statistics

#### Collection
- **Storage**: In-memory map `_id` → Document
- **Indexes**: Map of index name → Index
- **Cache**: Query cache instance
- **Operations**:
  - **Insert**: InsertOne, InsertMany (bulk insert with rollback on error)
  - **Find**: Find, FindOne, FindWithOptions (projection, sort, pagination)
  - **Update**: UpdateOne, UpdateMany (with update operators)
  - **Delete**: DeleteOne, DeleteMany
  - **Specialized**: TextSearch, Near (geo), GeoWithin

**Index Maintenance**:
- All write operations automatically update all indexes
- Partial indexes: Check filter before insertion
- Compound indexes: Extract composite keys from multiple fields

**Reference Documentation**: Database and collection APIs in [api-reference.md](api-reference.md)

### Layer 9: Client APIs

**Locations**: `cmd/laura-cli`, `pkg/server`, `clients/*`

**Modes**:

#### 1. Embedded Mode (Go Library)
```go
db, _ := database.Open("/path/to/data")
defer db.Close()
collection := db.Collection("users")
collection.InsertOne(doc)
```

#### 2. CLI Mode (Interactive REPL)
```bash
./bin/laura-cli
> use mydb
> db.users.insertOne({name: "Alice", age: 30})
> db.users.find({age: {$gte: 18}})
```

#### 3. HTTP Server Mode (REST API)
```bash
./bin/laura-server -port 8080

# API Calls
POST   /api/v1/collections/:name/documents
GET    /api/v1/collections/:name/documents
PUT    /api/v1/collections/:name/documents/:id
DELETE /api/v1/collections/:name/documents/:id
```

Admin console: http://localhost:8080/

#### 4. Client Libraries
- **Go Client**: `pkg/client` (connection pooling, retries, circuit breaker)
- **Python Client**: `clients/python` (requests-based, type hints)
- **Node.js Client**: `clients/nodejs` (axios-based, promises/async)
- **Java Client**: `clients/java` (CompletableFuture, builder pattern)

**Reference Documentation**: [http-api.md](http-api.md), [go-client.md](go-client.md), [python-client.md](python-client.md), [nodejs-client.md](nodejs-client.md), [java-client.md](java-client.md)

## Data Flow

### Write Operation Flow (InsertOne)

```
1. Application calls collection.InsertOne(doc)
   ↓
2. Generate ObjectID if missing (_id field)
   ↓
3. Start MVCC transaction
   txn := txnManager.Begin()
   ↓
4. Add to transaction's write set
   txn.WriteSet[docID] = VersionedValue{Value: doc, CreatedBy: txn.ID}
   ↓
5. Update all indexes
   for each index:
     - Extract key from document
     - Check partial filter (if partial index)
     - Insert into B+ tree: index.Insert(key, docID)
     - Update index statistics
   ↓
6. Write to WAL (durability)
   wal.Append(PageID, PageData)
   ↓
7. Write to buffer pool
   bufferPool.WritePage(pageID, pageData)
   ↓
8. Commit transaction
   txn.Commit() → versionStore.Put(docID, version)
   ↓
9. Invalidate query cache
   collection.cache.Clear()
   ↓
10. Return inserted document ID
```

**Atomicity**: If any step fails, transaction aborts and rolls back

**Durability**: WAL written before data pages

**Consistency**: All indexes updated or none

### Read Operation Flow (Find)

```
1. Application calls collection.Find(filter, options)
   ↓
2. Check query cache
   cacheKey := hash(filter, projection, sort, skip, limit)
   if cached := cache.Get(cacheKey); cached != nil {
       return cached  // 96x faster!
   }
   ↓
3. Parse query filter
   parser.Parse(filter) → QueryOperators
   ↓
4. Start MVCC transaction (for consistent snapshot)
   txn := txnManager.Begin()
   ↓
5. Query optimizer selects execution plan
   a. Enumerate applicable indexes
   b. Estimate selectivity for each index
   c. Calculate cost (index scan + document fetch)
   d. Choose lowest-cost plan
   ↓
6. Execute query
   IF index selected:
     - Index scan: docIDs := index.Search(key)
     - Document fetch: for docID in docIDs:
         doc := versionStore.Get(docID, txn.ReadVersion)
         if doc visible and matches filter:
             results.append(doc)
   ELSE:
     - Full collection scan: for doc in collection:
         if matches filter:
             results.append(doc)
   ↓
7. Apply sort (if not index-ordered)
   ↓
8. Apply projection (field selection)
   ↓
9. Apply skip/limit (pagination)
   ↓
10. Cache results
    cache.Put(cacheKey, results, ttl=5min)
    ↓
11. Commit transaction (releases snapshot)
    txn.Commit()
    ↓
12. Return results
```

**Snapshot Isolation**: All reads see consistent snapshot at txn.ReadVersion

**Performance**: Covered queries skip step 6b (no document fetch)

### Update Operation Flow (UpdateOne)

```
1. Application calls collection.UpdateOne(filter, update)
   ↓
2. Start MVCC transaction
   ↓
3. Find matching document (uses Find internally)
   ↓
4. Parse update operators ($set, $inc, $push, etc.)
   ↓
5. Apply updates to document copy
   newDoc := applyUpdates(oldDoc, update)
   ↓
6. Update indexes
   for each index:
     - Delete old entry: index.Delete(oldKey, docID)
     - Insert new entry: index.Insert(newKey, docID)
     - Update statistics
   ↓
7. Create new version in version store
   txn.WriteSet[docID] = VersionedValue{
       Value: newDoc,
       CreatedBy: txn.ID
   }
   ↓
8. Mark old version as deleted
   oldVersion.DeletedBy = txn.ID
   ↓
9. Write to WAL and buffer pool
   ↓
10. Commit transaction
    ↓
11. Invalidate query cache
    ↓
12. Return update count
```

**Version Chain After Update**:
```
Key "user:123"
    ↓
[V2: {age:31}, Created=T2] → [V1: {age:30}, Created=T1, Deleted=T2] → null
```

## Component Interactions

### Scenario: Concurrent Transactions

**Setup**: Two transactions updating different documents simultaneously

```go
// Transaction 1: Update user:alice
go func() {
    txn1 := txnMgr.Begin()  // ReadVersion=100
    collection.UpdateOne({"_id": "alice"}, {"$set": {"age": 31}})
    txn1.Commit()
}()

// Transaction 2: Update user:bob
go func() {
    txn2 := txnMgr.Begin()  // ReadVersion=101
    collection.UpdateOne({"_id": "bob"}, {"$set": {"age": 25}})
    txn2.Commit()
}()
```

**Interaction Flow**:

1. **Concurrent Execution**: Both transactions run in parallel
2. **Non-blocking**: T1 doesn't block T2 (different documents)
3. **Snapshot Isolation**: Each transaction sees database at its ReadVersion
4. **Write Sets**: T1 writes to WriteSet["alice"], T2 writes to WriteSet["bob"]
5. **Commit**: Both commits succeed (no conflicts)
6. **Version Store**: Both versions added to respective chains

**Result**: Both updates succeed without blocking

### Scenario: Write-Write Conflict

**Setup**: Two transactions updating the same document

```go
// Transaction 1
txn1 := txnMgr.Begin()  // ReadVersion=100
doc1 := collection.FindOne({"_id": "alice"})  // Adds to ReadSet
collection.UpdateOne({"_id": "alice"}, {"$set": {"age": 31}})
// Delay...
txn1.Commit()  // Conflict detected!

// Transaction 2
txn2 := txnMgr.Begin()  // ReadVersion=101
doc2 := collection.FindOne({"_id": "alice"})  // Adds to ReadSet
collection.UpdateOne({"_id": "alice"}, {"$set": {"age": 25}})
txn2.Commit()  // Succeeds (first to commit)
```

**Conflict Detection**:
1. T2 commits first → WriteSet["alice"] = version2
2. T1 tries to commit → Checks ReadSet["alice"]
3. Version store shows alice was modified after T1's ReadVersion
4. **Conflict!** T1.Commit() returns `ErrConflict`
5. Application must retry T1

**First-Committer-Wins**: T2 wins, T1 must abort

### Scenario: Index-Accelerated Query

**Setup**: Query with index on age field

```go
// Create index
collection.CreateIndex("age_idx", []string{"age"}, &IndexConfig{Unique: false})

// Query
results := collection.Find({"age": {"$gte": 18}})
```

**Component Interaction**:

1. **Query Parser** (pkg/query):
   - Parses filter: `GTE(age, 18)`
   - Identifies indexable predicate: `age >= 18`

2. **Query Optimizer** (pkg/query):
   - Finds `age_idx` index applicable
   - Gets index statistics: cardinality=50, total=1000
   - Estimates selectivity: 0.5 (50% of documents match)
   - Cost with index: `0.5 * 1000 * 10 = 5000`
   - Cost without index: `1000 * 100 = 100000`
   - **Decision**: Use age_idx

3. **Index Scan** (pkg/index):
   - B+ tree search: Find first leaf node with key >= 18
   - Scan right: Collect all docIDs with age >= 18
   - Returns: `[docID1, docID2, ...]`

4. **Document Fetch** (pkg/mvcc):
   - For each docID, fetch from version store
   - Apply snapshot isolation: Get version visible to transaction
   - Returns full documents

5. **Result Processing**:
   - Apply remaining filters (if any)
   - Apply projection, sort, skip, limit
   - Cache results

**Performance**: Index scan reduces search space from 1000 docs to 500 docs

## Concurrency Model

### MVCC Snapshot Isolation

**Key Principle**: Readers never block writers, writers never block readers

**Mechanism**:
- Each transaction gets unique monotonically increasing ID
- `ReadVersion` determines which versions are visible
- `WriteSet` holds uncommitted changes
- `ReadSet` tracks read keys for conflict detection

**Visibility Rule**:
```go
func IsVisible(version *VersionedValue, readVersion uint64) bool {
    return version.CreatedBy <= readVersion &&
           (version.DeletedBy == 0 || version.DeletedBy > readVersion)
}
```

**Version Chain Traversal**:
```go
func Get(key string, readVersion uint64) interface{} {
    versions := versionStore.GetVersions(key)
    for _, version := range versions {
        if IsVisible(version, readVersion) {
            return version.Value
        }
    }
    return nil  // Key not visible
}
```

**Garbage Collection**:
```go
// Clean versions no longer visible to any active transaction
minActiveVersion := txnMgr.MinActiveReadVersion()
for key, versions := range versionStore {
    for i, version := range versions {
        if version.DeletedBy != 0 && version.DeletedBy < minActiveVersion {
            // Safe to remove
            versions = append(versions[:i], versions[i+1:]...)
        }
    }
}
```

### Lock-Free Structures

**Where Used**:
- **Query Cache**: Sharded LRU with 32 shards (3.5x faster)
- **Statistics Counters**: Atomic operations (1.6ns/op)
- **Buffer Pool**: Read-write lock upgrade pattern (3-5x faster)

**Benefits**:
- Reduced contention under high concurrency
- Better CPU cache utilization
- Scalability on multi-core systems

### Connection Pooling (HTTP Server)

**Session Pool**: Reuses transaction session objects
```go
pool := &sync.Pool{
    New: func() interface{} {
        return &Session{...}
    },
}

session := pool.Get().(*Session)
defer pool.Put(session)
```

**Benefits**:
- Reduced allocation overhead
- Faster transaction startup
- Lower GC pressure

## Performance Characteristics

### Time Complexity

| Operation | Without Index | With B+ Tree Index | With Hash Index |
|-----------|---------------|--------------------|--------------------|
| Insert    | O(1)          | O(log n)           | O(1)               |
| Find (equality) | O(n)    | O(log n)           | O(1)               |
| Find (range) | O(n)       | O(log n + k)       | O(n)               |
| Update    | O(n)          | O(log n)           | O(1)               |
| Delete    | O(n)          | O(log n)           | O(1)               |

*k = number of matching documents*

### Space Complexity

| Component | Size | Notes |
|-----------|------|-------|
| Document | ~100-1000 bytes | JSON-like structure |
| ObjectID | 12 bytes | Timestamp + random + counter |
| B+ Tree Node | ~4KB | Order=32, fits one page |
| Version | ~150 bytes | Document + metadata |
| Buffer Pool | 4MB (default) | 1000 pages * 4KB |
| Query Cache | ~100KB | 1000 entries * ~100 bytes |

### Benchmarks

**Insert Performance**:
```
BenchmarkInsertOne-8           50000    25400 ns/op    (39,370 inserts/sec)
BenchmarkInsertMany-8         100000    13200 ns/op    (75,757 inserts/sec)
```

**Query Performance**:
```
BenchmarkFindWithIndex-8      100000    10900 ns/op    (91,743 queries/sec)
BenchmarkFindCached-8        3000000      342 ns/op    (2,923,976 queries/sec)
BenchmarkCoveredQuery-8       200000     4900 ns/op    (204,081 queries/sec)
```

**Index Performance**:
```
BenchmarkBTreeInsert-8        500000     3480 ns/op
BenchmarkBTreeSearch-8       1000000     1250 ns/op
BenchmarkBTreeScan-8          100000    12300 ns/op
```

**Transaction Performance**:
```
BenchmarkTransactionCommit-8  200000     6850 ns/op
BenchmarkMVCCRead-8          1000000     1120 ns/op
```

**Reference Documentation**: [benchmarking.md](benchmarking.md), [performance-tuning.md](performance-tuning.md)

## Design Decisions

### 1. Disk-Based Document Storage with Slotted Pages

**Decision**: Store documents on disk using slotted page structure

**Rationale**:
- **Production-grade persistence**: Documents survive server restarts
- **Scalability**: Supports datasets larger than available memory
- **Educational value**: Demonstrates real database storage engine design
- **Efficient variable-length storage**: Slotted pages handle varying document sizes
- **Performance**: LRU caching provides fast access to frequently used documents

**Implementation**:
- **DocumentStore**: Manages disk-based document persistence (pkg/database/document_store.go)
- **SlottedPage**: Variable-length document storage with compaction support
- **Location tracking**: Document IDs mapped to (PageID, SlotID) for direct access
- **Caching**: Per-collection LRU document cache reduces disk I/O
- **WAL integration**: Write-ahead logging ensures durability

**Trade-offs**:
- **Pro**: Data persists across restarts
- **Pro**: Can handle datasets larger than RAM
- **Pro**: More realistic database behavior for learning
- **Con**: Slightly slower than pure in-memory (mitigated by caching)
- **Con**: More complex implementation (valuable for educational purposes)

### 2. B+ Tree for Indexes

**Decision**: Use B+ trees as primary index structure

**Rationale**:
- Industry standard (PostgreSQL, MySQL, SQLite)
- Self-balancing (guaranteed O(log n))
- Efficient range scans (linked leaf nodes)
- Cache-friendly (high fanout)

**Trade-offs**:
- **Pro**: Excellent read performance
- **Pro**: Good write performance
- **Con**: Write amplification on splits
- **Con**: Fragmentation over time

**Alternative**: LSM trees for write-heavy workloads (implemented in `pkg/lsm`)

### 3. MVCC with Snapshot Isolation

**Decision**: Multi-version concurrency control with snapshot isolation

**Rationale**:
- Non-blocking reads (readers don't wait for writers)
- Consistent snapshots (repeatable reads)
- Standard in modern databases (PostgreSQL, Oracle, MongoDB)

**Trade-offs**:
- **Pro**: High concurrency
- **Pro**: No read locks
- **Con**: Version chain overhead
- **Con**: Garbage collection complexity

**Alternative**: Two-phase locking (simpler but lower concurrency)

### 4. Page Size = 4KB

**Decision**: Fixed 4KB pages for storage engine

**Rationale**:
- Matches typical OS page size
- Optimal I/O alignment (no torn writes)
- Efficient memory mapping

**Trade-offs**:
- **Pro**: No partial page writes
- **Pro**: Efficient mmap operations
- **Con**: Small documents waste space
- **Con**: Large documents span multiple pages

### 5. LRU Eviction Policy

**Decision**: Least Recently Used (LRU) for buffer pool and cache

**Rationale**:
- Simple to implement
- Effective for most workloads
- Predictable behavior

**Trade-offs**:
- **Pro**: Easy to reason about
- **Pro**: Good for temporal locality
- **Con**: Vulnerable to sequential scans
- **Con**: Doesn't consider access frequency

**Alternative**: LRU-K, 2Q, or ARC (Adaptive Replacement Cache)

### 6. Statistics-Based Optimizer

**Decision**: Use index statistics for query optimization

**Rationale**:
- More accurate than heuristics
- Adapts to data distribution
- Standard approach in databases

**Trade-offs**:
- **Pro**: Better plan selection
- **Pro**: Handles skewed data well
- **Con**: Statistics maintenance overhead
- **Con**: Histogram memory usage

### 7. MongoDB-Like API

**Decision**: MongoDB-compatible query language and operators

**Rationale**:
- Familiar to developers
- Rich operator set ($gte, $in, $regex, etc.)
- Well-documented standard

**Trade-offs**:
- **Pro**: Low learning curve
- **Pro**: Comprehensive operator coverage
- **Con**: Complex parsing
- **Con**: Some operators inefficient

### 8. HTTP Server with Admin Console

**Decision**: Provide REST API with web-based admin interface

**Rationale**:
- Language-agnostic API (not just Go)
- Easy testing and debugging
- Visual database management

**Trade-offs**:
- **Pro**: Multi-language support
- **Pro**: Human-friendly UI
- **Con**: HTTP overhead vs. embedded
- **Con**: Serialization cost (JSON)

## Comparison with Other Databases

### LauraDB vs MongoDB

| Feature | LauraDB | MongoDB |
|---------|---------|---------|
| **Written in** | Go | C++ |
| **Storage** | In-memory + WAL | WiredTiger (on-disk) |
| **Concurrency** | MVCC (snapshot isolation) | MVCC (snapshot isolation) |
| **Indexes** | B+ tree, text, geo, TTL | B-tree, text, geo, TTL |
| **Transactions** | Single-node ACID | Multi-document ACID, sharding |
| **Replication** | None | Replica sets |
| **Sharding** | None | Automatic sharding |
| **Query Language** | MongoDB-like subset | Full MongoDB query language |
| **Size** | ~20,000 LOC | Millions of LOC |
| **Use Case** | Embedded, educational | Production, distributed |

**Similarities**:
- Document model with flexible schema
- Rich query operators
- Multiple index types
- Aggregation pipeline
- ObjectID format

**Differences**:
- LauraDB: Single-node, in-memory, educational
- MongoDB: Distributed, production-scale, enterprise features

### LauraDB vs SQLite

| Feature | LauraDB | SQLite |
|---------|---------|--------|
| **Data Model** | Document (schema-less) | Relational (schema-based) |
| **Query Language** | MongoDB-like | SQL |
| **Storage** | In-memory + WAL | B-tree on disk |
| **Concurrency** | MVCC (high concurrency) | Readers-writer locks |
| **Transactions** | ACID (snapshot isolation) | ACID (serializable) |
| **Indexes** | B+ tree, text, geo | B-tree, R-tree |
| **Footprint** | ~20 MB binary | ~1 MB library |
| **Maturity** | Educational | Production (20+ years) |

**When to use LauraDB**:
- Schema-less document storage
- MongoDB-like API preferred
- Learning database internals

**When to use SQLite**:
- Relational data model
- SQL queries
- Maximum portability

### LauraDB vs PostgreSQL

| Feature | LauraDB | PostgreSQL |
|---------|---------|------------|
| **Data Model** | Document | Relational + JSONB |
| **Query Language** | MongoDB-like | SQL |
| **Storage** | In-memory + WAL | MVCC on disk |
| **Concurrency** | MVCC (snapshot) | MVCC (serializable) |
| **Indexes** | B+ tree, text, geo | B-tree, GiST, GIN, BRIN |
| **Full-Text Search** | BM25 | tsvector/tsquery |
| **Replication** | None | Streaming replication |
| **Extensions** | None | Rich ecosystem |

**When to use LauraDB**:
- Document-centric applications
- Embedded database needs
- Learning MVCC and indexing

**When to use PostgreSQL**:
- Complex relational queries
- ACID guarantees at scale
- Production workloads

## Conclusion

LauraDB demonstrates production-grade database implementation with:
- **Clean Architecture**: Layered design with clear separation of concerns
- **Educational Value**: Every component explained and documented
- **Production Concepts**: MVCC, WAL, B+ trees, query optimization
- **Complete Implementation**: Storage → Transactions → Indexes → Query → API
- **Comprehensive Testing**: 76+ tests with high coverage

**Learning Path**:
1. Start with [getting-started.md](getting-started.md) for basic usage
2. Understand [storage-engine.md](storage-engine.md) for persistence layer
3. Study [mvcc.md](mvcc.md) for concurrency control
4. Explore [indexing.md](indexing.md) for query acceleration
5. Learn [query-engine.md](query-engine.md) for optimization
6. Read [performance-tuning.md](performance-tuning.md) for best practices

**Use Cases**:
- **Learning**: Study database internals with runnable code
- **Embedded**: Lightweight document database for Go applications
- **Prototyping**: Quick document storage without MongoDB dependency
- **Testing**: In-memory database for unit tests

**Not Recommended For**:
- Production distributed systems (no replication/sharding)
- Large datasets (in-memory limitation)
- Multi-node deployments (single-node only)

**Future Enhancements**:
- On-disk document storage (remove memory limitation)
- Replication (leader-follower)
- Query optimization improvements (join algorithms)
- More index types (hash indexes, full-text variants)

---

**Additional Resources**:
- [API Reference](api-reference.md) - Complete API documentation
- [Performance Tuning](performance-tuning.md) - Optimization guide
- [Benchmarking](benchmarking.md) - Performance testing
- [Client Libraries](go-client.md) - Multi-language client usage

For questions or contributions, see the main [README.md](../README.md).
