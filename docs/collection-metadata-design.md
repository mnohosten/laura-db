# Collection Metadata Persistence Design

## Overview

This document describes the design for persisting collection metadata to disk, enabling collections to survive server restarts. This is a critical component of the disk storage implementation (Priority 0, Phase 1).

## Goals

1. **Persistence**: Collection definitions, schemas, and indexes survive server restarts
2. **Consistency**: Metadata is kept in sync with data through atomic updates
3. **Performance**: Fast collection discovery on database startup
4. **Extensibility**: Support for future metadata types (sharding info, constraints, etc.)
5. **ACID Compliance**: Metadata changes are logged in WAL for crash recovery

## Architecture Overview

```
┌──────────────────────────────────────────────────────────────┐
│                    Database Layer                             │
│  (manages collection catalog, metadata cache)                 │
└──────────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────────┐
│              Collection Catalog                               │
│  (central registry of all collections)                        │
│  - CollectionID assignment                                    │
│  - Collection name → metadata mapping                         │
│  - Metadata page management                                   │
└──────────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────────┐
│              Metadata Pages                                   │
│  - Collection metadata pages                                  │
│  - Index metadata pages                                       │
│  - Statistics pages                                           │
└──────────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────────┐
│              Storage Engine                                   │
│  (buffer pool, WAL, disk manager)                             │
└──────────────────────────────────────────────────────────────┘
```

## Collection Catalog Structure

### Catalog Page (Special Page 0)

The first page in the database file is reserved for the collection catalog. This is a special system page that stores metadata about all collections.

```
Page 0: Collection Catalog
┌────────────────────────────────────────────────────────────┐
│          Page Header (16 bytes)                             │
│  - PageID = 0                                               │
│  - PageType = PageTypeCatalog                               │
├────────────────────────────────────────────────────────────┤
│          Catalog Header (32 bytes)                          │
│  - Magic Number (4 bytes): 0x4C415552 ("LAUR")             │
│  - Version (2 bytes): Schema version (1)                    │
│  - CollectionCount (4 bytes)                                │
│  - NextCollectionID (4 bytes)                               │
│  - FirstMetadataPageID (4 bytes)                            │
│  - FreeMetadataPageID (4 bytes)                             │
│  - LastCheckpointTxnID (8 bytes)                            │
│  - Reserved (2 bytes)                                       │
├────────────────────────────────────────────────────────────┤
│      Collection Directory (grows downward)                  │
│  Entry 0: [CollectionID][NameLen][Name...][MetadataPageID] │
│  Entry 1: [CollectionID][NameLen][Name...][MetadataPageID] │
│  ...                                                        │
│                                                             │
│              Free Space                                     │
│                                                             │
└────────────────────────────────────────────────────────────┘
```

### Catalog Header (32 bytes)

| Field | Size | Description |
|-------|------|-------------|
| Magic Number | 4 bytes | 0x4C415552 ("LAUR" in ASCII) for file format validation |
| Version | 2 bytes | Schema version for compatibility (current: 1) |
| CollectionCount | 4 bytes | Total number of collections in database |
| NextCollectionID | 4 bytes | Next available collection ID (monotonic counter) |
| FirstMetadataPageID | 4 bytes | First page in metadata page chain |
| FreeMetadataPageID | 4 bytes | First free metadata page (for reuse) |
| LastCheckpointTxnID | 8 bytes | Transaction ID of last checkpoint |
| Reserved | 2 bytes | Reserved for future use |

### Collection Directory Entry (variable length)

Each entry maps a collection name to its metadata:

```
┌────────────────────────────────────────────────────────────┐
│ CollectionID (4 bytes)                                      │
│ NameLength (2 bytes)                                        │
│ Name (variable, up to 255 chars)                            │
│ MetadataPageID (4 bytes)                                    │
│ Flags (2 bytes)                                             │
│   - Bit 0: IsActive (1 = active, 0 = dropped)              │
│   - Bit 1: IsSystem (1 = system collection)                │
│   - Bits 2-15: Reserved                                     │
└────────────────────────────────────────────────────────────┘
```

Total size: 12 + NameLength bytes

**Collection Naming Rules**:
- Maximum name length: 255 characters
- Valid characters: alphanumeric, underscore, dash
- Case-sensitive
- Cannot start with "system." (reserved for system collections)

## Collection Metadata Pages

### Collection Metadata Page Structure

Each collection has a dedicated metadata page containing its configuration and schema information.

```
Collection Metadata Page
┌────────────────────────────────────────────────────────────┐
│          Page Header (16 bytes)                             │
│  - PageID = assigned ID                                     │
│  - PageType = PageTypeCollectionMeta                        │
├────────────────────────────────────────────────────────────┤
│      Collection Metadata Header (48 bytes)                  │
│  - CollectionID (4 bytes)                                   │
│  - NameLength (2 bytes)                                     │
│  - Name (up to 64 bytes, null-padded)                       │
│  - CreatedTxnID (8 bytes)                                   │
│  - CreatedTimestamp (8 bytes, Unix nanoseconds)             │
│  - FirstDataPageID (4 bytes)                                │
│  - DocumentCount (8 bytes)                                  │
│  - DataSizeBytes (8 bytes)                                  │
│  - IndexCount (2 bytes)                                     │
│  - FirstIndexMetadataPageID (4 bytes)                       │
│  - StatisticsPageID (4 bytes)                               │
│  - Flags (2 bytes)                                          │
│  - Reserved (10 bytes)                                      │
├────────────────────────────────────────────────────────────┤
│              Schema Descriptor (variable)                   │
│  - SchemaVersion (2 bytes)                                  │
│  - SchemaLength (2 bytes)                                   │
│  - Schema (JSON-encoded, variable length)                   │
│    {                                                        │
│      "type": "object",                                      │
│      "properties": {...},                                   │
│      "required": [...],                                     │
│      "additionalProperties": true                           │
│    }                                                        │
├────────────────────────────────────────────────────────────┤
│          Collection Options (variable)                      │
│  - OptionsLength (2 bytes)                                  │
│  - Options (JSON-encoded)                                   │
│    {                                                        │
│      "capped": false,                                       │
│      "maxSize": 0,                                          │
│      "maxDocuments": 0,                                     │
│      "validationLevel": "strict",                           │
│      "validationAction": "error"                            │
│    }                                                        │
└────────────────────────────────────────────────────────────┘
```

### Collection Metadata Header (48 bytes)

| Field | Size | Description |
|-------|------|-------------|
| CollectionID | 4 bytes | Unique collection identifier |
| NameLength | 2 bytes | Length of collection name |
| Name | 64 bytes | Collection name (null-padded) |
| CreatedTxnID | 8 bytes | Transaction ID when collection was created |
| CreatedTimestamp | 8 bytes | Unix nanoseconds when collection was created |
| FirstDataPageID | 4 bytes | First page containing document data |
| DocumentCount | 8 bytes | Total number of documents (cached, may be stale) |
| DataSizeBytes | 8 bytes | Total size of all documents in bytes |
| IndexCount | 2 bytes | Number of indexes (including default _id index) |
| FirstIndexMetadataPageID | 4 bytes | First page in index metadata chain |
| StatisticsPageID | 4 bytes | Page containing collection statistics |
| Flags | 2 bytes | Collection-level flags |
| Reserved | 10 bytes | Reserved for future use |

### Schema Descriptor

Collections can optionally have a schema defined (JSON Schema format):

```json
{
  "type": "object",
  "properties": {
    "_id": {"type": "string"},
    "name": {"type": "string", "minLength": 1, "maxLength": 100},
    "age": {"type": "integer", "minimum": 0, "maximum": 150},
    "email": {"type": "string", "format": "email"},
    "address": {
      "type": "object",
      "properties": {
        "street": {"type": "string"},
        "city": {"type": "string"},
        "zip": {"type": "string", "pattern": "^[0-9]{5}$"}
      }
    }
  },
  "required": ["_id", "name"],
  "additionalProperties": true
}
```

**Schema Validation Levels**:
- `off`: No validation
- `moderate`: Validate on inserts and updates (skip invalid updates)
- `strict`: Validate on inserts and updates (reject invalid operations)

**Schema Validation Actions**:
- `error`: Reject invalid documents
- `warn`: Log warning but allow operation

## Index Metadata Pages

### Index Metadata Page Structure

Each index has metadata stored in a dedicated page. Indexes are chained together per collection.

```
Index Metadata Page
┌────────────────────────────────────────────────────────────┐
│          Page Header (16 bytes)                             │
│  - PageID = assigned ID                                     │
│  - PageType = PageTypeIndexMeta                             │
├────────────────────────────────────────────────────────────┤
│        Index Metadata Header (64 bytes)                     │
│  - IndexID (4 bytes)                                        │
│  - CollectionID (4 bytes)                                   │
│  - NameLength (2 bytes)                                     │
│  - Name (up to 32 bytes, null-padded)                       │
│  - IndexType (1 byte)                                       │
│    0 = B+ tree, 1 = Hash, 2 = Text,                        │
│    3 = 2d (planar), 4 = 2dsphere (spherical)               │
│  - IndexVersion (1 byte)                                    │
│  - FieldCount (2 bytes): Number of fields (1 for simple,   │
│                          >1 for compound)                   │
│  - IsUnique (1 byte): 1 = unique, 0 = non-unique           │
│  - IsSparse (1 byte): 1 = sparse, 0 = dense                │
│  - IsPartial (1 byte): 1 = partial, 0 = full               │
│  - Reserved (1 byte)                                        │
│  - CreatedTxnID (8 bytes)                                   │
│  - CreatedTimestamp (8 bytes)                               │
│  - RootPageID (4 bytes): Root page of B+ tree              │
│  - EntryCount (8 bytes): Number of index entries           │
│  - Depth (2 bytes): Tree depth (for B+ tree)               │
│  - Order (2 bytes): B+ tree order                          │
│  - NextIndexMetadataPageID (4 bytes): Next index in chain  │
│  - StatisticsPageID (4 bytes): Page with index statistics  │
│  - Reserved (8 bytes)                                       │
├────────────────────────────────────────────────────────────┤
│          Field Definitions (variable)                       │
│  For each field (up to 32 fields for compound indexes):    │
│    - FieldNameLength (1 byte)                              │
│    - FieldName (up to 127 bytes)                           │
│    - SortOrder (1 byte): 1 = ascending, -1 = descending    │
│  Total per field: 2 + FieldNameLength bytes                │
├────────────────────────────────────────────────────────────┤
│          Partial Index Filter (variable, optional)         │
│  - FilterLength (2 bytes)                                   │
│  - Filter (JSON-encoded query expression)                   │
│    {"age": {"$gte": 18}, "status": "active"}               │
├────────────────────────────────────────────────────────────┤
│          Index-Specific Options (variable)                  │
│  - OptionsLength (2 bytes)                                  │
│  - Options (JSON-encoded)                                   │
│                                                             │
│  For Text Indexes:                                          │
│    {"defaultLanguage": "english",                           │
│     "weights": {"title": 10, "body": 1}}                   │
│                                                             │
│  For Geo Indexes:                                           │
│    {"min": -180, "max": 180,                               │
│     "distanceMultiplier": 1.0}                             │
│                                                             │
│  For TTL Indexes:                                           │
│    {"expireAfterSeconds": 86400}                           │
└────────────────────────────────────────────────────────────┘
```

### Index Metadata Header (64 bytes)

| Field | Size | Description |
|-------|------|-------------|
| IndexID | 4 bytes | Unique index identifier (within collection) |
| CollectionID | 4 bytes | Parent collection ID |
| NameLength | 2 bytes | Length of index name |
| Name | 32 bytes | Index name (null-padded) |
| IndexType | 1 byte | Type of index (BTree, Hash, Text, etc.) |
| IndexVersion | 1 byte | Index format version |
| FieldCount | 2 bytes | Number of fields (1 = simple, >1 = compound) |
| IsUnique | 1 byte | Uniqueness constraint flag |
| IsSparse | 1 byte | Sparse index flag (skip docs without field) |
| IsPartial | 1 byte | Partial index flag (filtered index) |
| Reserved | 1 byte | Reserved |
| CreatedTxnID | 8 bytes | Transaction ID when index was created |
| CreatedTimestamp | 8 bytes | Unix nanoseconds when index was created |
| RootPageID | 4 bytes | Root page of B+ tree (0 if not B+ tree) |
| EntryCount | 8 bytes | Number of entries in index |
| Depth | 2 bytes | Tree depth (for B+ tree) |
| Order | 2 bytes | B+ tree order (typically 32) |
| NextIndexMetadataPageID | 4 bytes | Next index metadata page (linked list) |
| StatisticsPageID | 4 bytes | Page containing index statistics |
| Reserved | 8 bytes | Reserved for future use |

### Index Type Encoding

| IndexType Value | Description |
|----------------|-------------|
| 0 | B+ Tree (default for most indexes) |
| 1 | Hash (future, not currently implemented) |
| 2 | Text (inverted index with stemming) |
| 3 | 2d (planar geospatial, Euclidean distance) |
| 4 | 2dsphere (spherical geospatial, Haversine distance) |

## Statistics Persistence

### Collection Statistics Page

```
Collection Statistics Page
┌────────────────────────────────────────────────────────────┐
│          Page Header (16 bytes)                             │
│  - PageID = assigned ID                                     │
│  - PageType = PageTypeStats                                 │
├────────────────────────────────────────────────────────────┤
│      Collection Statistics (64 bytes)                       │
│  - CollectionID (4 bytes)                                   │
│  - LastUpdated (8 bytes): Unix nanoseconds                  │
│  - DocumentCount (8 bytes)                                  │
│  - AvgDocumentSize (4 bytes): In bytes                      │
│  - DataSize (8 bytes): Total data size in bytes            │
│  - StorageSize (8 bytes): Total storage (including overhead)│
│  - TotalIndexSize (8 bytes): Sum of all index sizes        │
│  - PaddingFactor (4 bytes): Float32, space for growth      │
│  - DeletedCount (8 bytes): Number of deleted documents     │
│  - Reserved (8 bytes)                                       │
├────────────────────────────────────────────────────────────┤
│          Field Cardinality Statistics (variable)           │
│  Array of:                                                  │
│    - FieldNameLength (1 byte)                              │
│    - FieldName (up to 127 bytes)                           │
│    - Cardinality (8 bytes): Number of distinct values      │
│    - NullCount (8 bytes): Number of null/missing values    │
│  Useful for query planning                                  │
└────────────────────────────────────────────────────────────┘
```

### Index Statistics Page

```
Index Statistics Page
┌────────────────────────────────────────────────────────────┐
│          Page Header (16 bytes)                             │
│  - PageID = assigned ID                                     │
│  - PageType = PageTypeStats                                 │
├────────────────────────────────────────────────────────────┤
│          Index Statistics (64 bytes)                        │
│  - IndexID (4 bytes)                                        │
│  - CollectionID (4 bytes)                                   │
│  - LastUpdated (8 bytes): Unix nanoseconds                  │
│  - TotalEntries (8 bytes): Total number of entries         │
│  - UniqueKeys (8 bytes): Number of distinct keys           │
│  - MinValueLength (2 bytes)                                 │
│  - MinValue (up to 64 bytes): Smallest key                 │
│  - MaxValueLength (2 bytes)                                 │
│  - MaxValue (up to 64 bytes): Largest key                  │
│  - AvgKeySize (4 bytes): Average key size in bytes         │
│  - Depth (2 bytes): B+ tree depth                          │
│  - IsStale (1 byte): 1 = needs recalculation               │
│  - Reserved (1 byte)                                        │
├────────────────────────────────────────────────────────────┤
│          Histogram Buckets (variable)                       │
│  - NumBuckets (2 bytes): Number of histogram buckets       │
│  For each bucket:                                           │
│    - LowerBoundLength (2 bytes)                            │
│    - LowerBound (variable, up to 64 bytes)                 │
│    - UpperBoundLength (2 bytes)                            │
│    - UpperBound (variable, up to 64 bytes)                 │
│    - Count (4 bytes): Number of values in bucket           │
│    - Frequency (4 bytes): Float32, normalized frequency    │
└────────────────────────────────────────────────────────────┘
```

**Statistics Update Policy**:
- Mark as stale on every write operation
- Recalculate asynchronously when:
  - Query optimizer needs stats and IsStale = true
  - Background stats refresh job runs (every 5 minutes)
  - Manual ANALYZE command issued
- Small updates (< 1% of data) → keep existing stats
- Large updates (> 10% of data) → force immediate recalculation

## Metadata Update Operations

### Creating a Collection

```
Transaction: CreateCollection("users")

1. Begin transaction (get TxnID)
2. Allocate new CollectionID (atomic increment of NextCollectionID)
3. Allocate metadata page for collection
4. Write collection metadata:
   - CollectionID
   - Name = "users"
   - CreatedTxnID = current txn
   - CreatedTimestamp = now
   - Default values for counts
5. Create default _id index:
   - Allocate index metadata page
   - Write index metadata
   - Link to collection metadata
6. Update catalog page:
   - Increment CollectionCount
   - Add collection directory entry
   - Update NextCollectionID
7. Log all changes to WAL:
   - WAL: AllocatePage(metadataPageID)
   - WAL: WriteCollectionMetadata(...)
   - WAL: WriteIndexMetadata(...)
   - WAL: UpdateCatalog(...)
8. Commit transaction
9. Flush WAL to disk
```

### Creating an Index

```
Transaction: CreateIndex(collectionID, indexConfig)

1. Begin transaction
2. Allocate index metadata page
3. Write index metadata:
   - IndexID = collectionID + indexCount
   - Name, Type, Fields, Options
   - CreatedTxnID = current txn
   - RootPageID = 0 (empty tree)
4. Update collection metadata:
   - Increment IndexCount
   - Link index metadata page to chain
5. Log to WAL:
   - WAL: WriteIndexMetadata(...)
   - WAL: UpdateCollectionMetadata(...)
6. Commit transaction
7. Build index in background (if background=true)
```

### Updating Statistics

```
Transaction: UpdateIndexStats(indexID, stats)

1. Begin transaction
2. Read or allocate statistics page
3. Write updated statistics:
   - TotalEntries
   - UniqueKeys
   - MinValue, MaxValue
   - Histogram buckets
   - LastUpdated = now
   - IsStale = false
4. Log to WAL:
   - WAL: WriteIndexStats(...)
5. Commit transaction
```

### Dropping a Collection

```
Transaction: DropCollection(collectionID)

1. Begin transaction
2. Mark collection as inactive in catalog:
   - Set Flags.IsActive = 0
3. Log to WAL:
   - WAL: UpdateCatalog(...)
4. Commit transaction
5. Background cleanup (async):
   - Deallocate all data pages
   - Deallocate all index pages
   - Deallocate metadata pages
   - Remove from catalog
```

## Metadata Caching Strategy

### In-Memory Metadata Cache

To avoid reading metadata pages on every operation, maintain an in-memory cache:

```go
type MetadataCache struct {
    collections   map[uint32]*CollectionMetadata    // CollectionID → metadata
    collectionsByName map[string]uint32              // Name → CollectionID
    indexes       map[uint32]map[uint32]*IndexMetadata // CollectionID → IndexID → metadata
    statistics    map[uint32]*IndexStats             // IndexID → stats
    mu            sync.RWMutex
}
```

**Cache Update Policy**:
- Cache populated on database open (read catalog + all metadata pages)
- Cache updated on metadata changes (collection creation, index creation)
- Cache invalidated on collection drop
- Statistics cached separately with TTL (5 minutes)

**Cache Consistency**:
- All metadata changes logged to WAL before updating cache
- On crash recovery, rebuild cache from persisted metadata
- No stale reads: cache updated transactionally

## Database Startup Sequence

### Opening an Existing Database

```
Open(dataDir)

1. Open storage engine (disk manager, buffer pool, WAL)
2. Check if catalog page exists:
   - Read Page 0
   - Verify magic number (0x4C415552)
   - Verify schema version
3. If catalog exists (existing database):
   a. Read catalog header
   b. For each collection in directory:
      - Read collection metadata page
      - Load into metadata cache
      - Read all index metadata pages
      - Load index metadata into cache
   c. Build name → CollectionID map
   d. Restore statistics (lazy load on first use)
4. If catalog doesn't exist (new database):
   a. Initialize catalog page
   b. Write magic number and version
   c. Set CollectionCount = 0, NextCollectionID = 1
   d. Flush catalog to disk
5. Create transaction manager
6. Start TTL cleanup goroutine
7. Return Database instance
```

### Crash Recovery

On database open after a crash:

```
Recovery Sequence

1. Open storage engine
2. Replay WAL:
   - For each WAL entry:
     - If AllocatePage → mark page as allocated
     - If WriteCollectionMetadata → restore metadata
     - If WriteIndexMetadata → restore index metadata
     - If UpdateCatalog → restore catalog
     - If WritePage → restore page data
3. After WAL replay:
   - Rebuild metadata cache from recovered pages
   - Verify catalog integrity (checksum)
   - Mark statistics as stale (force recalculation)
4. Truncate WAL (all changes applied)
5. Continue normal startup sequence
```

## Page Type Registry

Add new page types to storage engine:

```go
const (
    PageTypeData     PageType = 0  // Document data pages (existing)
    PageTypeFreeList PageType = 1  // Free page list (existing)
    PageTypeOverflow PageType = 2  // Overflow pages (existing)
    PageTypeCatalog  PageType = 3  // Collection catalog (NEW)
    PageTypeCollectionMeta PageType = 4  // Collection metadata (NEW)
    PageTypeIndexMeta PageType = 5  // Index metadata (NEW)
    PageTypeStats    PageType = 6  // Statistics pages (NEW)
    PageTypeBTreeNode PageType = 7  // B+ tree nodes (future)
    PageTypeBTreeLeaf PageType = 8  // B+ tree leaf nodes (future)
)
```

## Serialization Format

### Encoding

All metadata uses **little-endian** byte order for consistency across platforms.

**Primitive Types**:
- `uint8`, `int8`: 1 byte
- `uint16`, `int16`: 2 bytes
- `uint32`, `int32`: 4 bytes
- `uint64`, `int64`: 8 bytes
- `float32`: 4 bytes (IEEE 754)
- `float64`: 8 bytes (IEEE 754)

**Variable-Length Strings**:
```
[Length (2 bytes)][Data (variable)]
```

**Variable-Length Objects** (Schema, Options, Filters):
```
[Length (2 bytes)][JSON Data (variable)]
```

**Arrays**:
```
[Count (2 bytes)][Element 0][Element 1]...[Element N-1]
```

### Checksums (Future Enhancement)

For data integrity, add checksums to metadata pages:

```
Page Footer (8 bytes):
  - Checksum (4 bytes): CRC32 of page contents
  - ChecksumType (1 byte): 0 = none, 1 = CRC32, 2 = xxHash
  - Reserved (3 bytes)
```

Verify checksum on page read, reject if mismatch.

## Space Requirements

### Catalog Page (Page 0)

- Fixed overhead: 48 bytes (page header + catalog header)
- Per collection: ~20 bytes (directory entry with typical name length)
- **Capacity**: ~200 collections per catalog page

If more collections needed, use overflow pages linked from catalog.

### Collection Metadata Page

- Fixed overhead: 64 bytes (page header + metadata header)
- Schema: variable (typically 200-1000 bytes)
- Options: variable (typically 50-200 bytes)
- **Capacity**: 1 collection per page (4 KB is sufficient for most schemas)

### Index Metadata Page

- Fixed overhead: 80 bytes (page header + metadata header)
- Field definitions: ~20 bytes per field
- Partial filter: variable (typically 50-500 bytes)
- Options: variable (typically 50-200 bytes)
- **Capacity**: 1 index per page

### Statistics Page

- Fixed overhead: 80 bytes (page header + statistics header)
- Histogram: ~20 bytes per bucket × 10 buckets = 200 bytes
- **Capacity**: 1 index statistics per page

### Total Overhead per Collection

Assuming 3 indexes (default _id + 2 user indexes):
- Catalog entry: 20 bytes
- Collection metadata: 1 page (4 KB)
- Index metadata: 3 pages (12 KB)
- Statistics: 4 pages (16 KB, collection + 3 indexes)
- **Total**: ~33 KB per collection

For 100 collections: ~3.3 MB of metadata (negligible overhead).

## API Changes

### Database API

```go
// Internal metadata management (not exposed to users)
type Database struct {
    catalog       *CollectionCatalog
    metadataCache *MetadataCache
    // ... existing fields
}

// Load all collections on startup
func (db *Database) loadCatalog() error

// Persist collection metadata
func (db *Database) persistCollection(coll *Collection) error

// Persist index metadata
func (db *Database) persistIndex(idx *Index, collID uint32) error

// Update statistics on disk
func (db *Database) persistStatistics(indexID uint32, stats *IndexStats) error
```

### Collection API

```go
// Add metadata reference
type Collection struct {
    id           uint32            // CollectionID (NEW)
    metadataPage PageID            // Page containing metadata (NEW)
    schema       *CollectionSchema // Validation schema (NEW)
    options      *CollectionOptions // Collection options (NEW)
    // ... existing fields
}

// Schema validation
func (c *Collection) validateDocument(doc *Document) error

// Persist changes to disk
func (c *Collection) syncMetadata() error
```

### Index API

```go
// Add metadata reference
type Index struct {
    id           uint32  // IndexID (NEW)
    collectionID uint32  // Parent collection (NEW)
    metadataPage PageID  // Page containing metadata (NEW)
    rootPage     PageID  // Root page of B+ tree (NEW, for disk indexes)
    // ... existing fields
}

// Persist index metadata
func (idx *Index) syncMetadata() error
```

## Testing Strategy

### Unit Tests

- Catalog page serialization/deserialization
- Collection metadata encoding/decoding
- Index metadata encoding/decoding
- Statistics serialization
- Metadata cache operations

### Integration Tests

1. **Create collection → restart → verify collection exists**
   - Create collection with schema
   - Create indexes
   - Restart database
   - Verify all metadata restored

2. **Update statistics → restart → verify statistics persisted**
   - Insert data
   - Update statistics
   - Restart database
   - Verify statistics correct

3. **Drop collection → restart → verify collection gone**
   - Create and populate collection
   - Drop collection
   - Restart database
   - Verify collection not in catalog

4. **Crash during metadata update → verify recovery**
   - Start metadata update
   - Kill process mid-update
   - Restart database
   - Verify WAL recovery restored consistency

### Performance Tests

- Collection creation throughput
- Index creation latency
- Database startup time (with N collections)
- Metadata cache hit rate
- Concurrent metadata updates

## Migration Plan

### Phase 1: Add Metadata Structures

- Define page types and structures
- Implement serialization/deserialization
- Add unit tests

### Phase 2: Implement Collection Catalog

- Initialize catalog on database open
- Read/write catalog page
- Collection registration/discovery

### Phase 3: Persist Collection Metadata

- Write collection metadata on creation
- Load collection metadata on startup
- Update metadata on schema changes

### Phase 4: Persist Index Metadata

- Write index metadata on creation
- Load index metadata on startup
- Link indexes to B+ tree root pages (future)

### Phase 5: Persist Statistics

- Write statistics to disk
- Load statistics on demand
- Background statistics refresh

### Phase 6: Metadata Cache

- Implement in-memory cache
- Cache invalidation on updates
- Cache reconstruction on startup

### Phase 7: WAL Integration

- Log metadata changes to WAL
- Replay metadata changes on recovery
- Checkpoint metadata pages

## Configuration Options

```go
type MetadataConfig struct {
    // Enable schema validation
    EnableSchemaValidation bool // default: false

    // Cache size
    MetadataCacheSize int // default: 1000 collections

    // Statistics refresh interval
    StatsRefreshInterval time.Duration // default: 5 minutes

    // Auto-create indexes on schema fields
    AutoIndexSchemaFields bool // default: false
}
```

## Future Enhancements

### 1. Schema Evolution

Support schema versioning and migration:
```go
type SchemaVersion struct {
    Version    int
    Schema     *CollectionSchema
    MigrationFn func(doc *Document) error
}
```

### 2. Collection Templates

Predefined collection templates for common use cases:
- User management collection
- Audit log collection
- Time-series collection
- Event stream collection

### 3. Metadata Replication

For distributed setups, replicate metadata across nodes:
- Metadata changelog
- Metadata sync protocol
- Conflict resolution

### 4. Metadata Backup

Periodic metadata snapshots for disaster recovery:
```bash
laura-backup --metadata-only --output=metadata.snapshot
```

### 5. Metadata Indexes

For fast collection discovery by attributes:
- Index collections by creation date
- Index collections by size
- Index collections by tag

## Summary

This design provides:

✅ **Persistence**: Collections and indexes survive restarts
✅ **Performance**: In-memory cache with lazy loading
✅ **Consistency**: WAL-logged metadata changes
✅ **Extensibility**: Room for future metadata types
✅ **Efficiency**: Minimal overhead (~33 KB per collection)

**Key Design Decisions**:
1. **Dedicated Catalog Page**: Page 0 is reserved for collection catalog
2. **One Page per Collection**: Sufficient for most schemas (4 KB)
3. **Metadata Chaining**: Indexes linked in a chain per collection
4. **In-Memory Cache**: Fast access, rebuild from disk on startup
5. **JSON for Complex Data**: Schemas, options, and filters use JSON encoding
6. **WAL Integration**: All metadata changes logged for crash recovery
7. **Lazy Statistics Loading**: Load statistics on demand to speed up startup

**Next Steps**:
- Proceed to implement catalog page structure (Phase 2 of TODO.md)
- Add metadata serialization/deserialization utilities
- Integrate with existing storage engine

This design complements the document storage design (`disk-storage-design.md`) and provides the foundation for persistent collections.
