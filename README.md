# LauraDB 📚

An educational implementation of a MongoDB-like document database in Go.

LauraDB is a fast, embedded document database with an interactive CLI tool and a complete HTTP server with web-based admin console.

## Overview

This project demonstrates how to build a production-grade document database from scratch, covering all major components including storage engines, indexing, query processing, and concurrency control.

## Architecture

### Core Components

#### 1. Document Format (`pkg/document`)
- BSON-like binary encoding for documents
- Support for various data types (string, number, boolean, array, nested documents)
- Efficient serialization/deserialization

#### 2. Storage Engine (`pkg/storage`)
- **Write-Ahead Log (WAL)**: Ensures durability by logging all operations before applying them
- **Page-based storage**: Data organized into fixed-size pages for efficient I/O
- **Buffer pool**: In-memory cache for frequently accessed pages
- **Checkpointing**: Periodic snapshots for faster recovery

#### 3. MVCC - Multi-Version Concurrency Control (`pkg/mvcc`)
- **Transaction isolation**: Multiple concurrent readers without blocking writers
- **Snapshot isolation**: Each transaction sees a consistent view of data
- **Version chains**: Multiple versions of documents maintained for concurrent access
- **Garbage collection**: Old versions cleaned up when no longer needed

#### 4. Indexing (`pkg/index`)
- **B+ tree implementation**: Self-balancing tree for fast lookups
- **Multi-key indexes**: Support for compound indexes on multiple fields
- **Index types**: Unique, non-unique, sparse indexes
- **Automatic index maintenance**: Updated during document modifications

#### 5. Query Engine (`pkg/query`)
- **Query parser**: Converts MongoDB-like queries into execution plans
- **Query operators**:
  - Comparison: `$eq`, `$ne`, `$gt`, `$gte`, `$lt`, `$lte`
  - Logical: `$and`, `$or`, `$not`
  - Array: `$in`, `$nin`, `$all`, `$elemMatch`, `$size`
  - Element: `$exists`, `$type`
  - Evaluation: `$regex` (pattern matching with full regex support)
- **Query optimizer**: Cost-based optimization using index statistics
  - Statistics tracking: Cardinality, selectivity, min/max values
  - Intelligent index selection: Chooses best index based on query cost estimation
  - Covered query detection: Query satisfied entirely from index
- **Parallel execution**: Multi-core query processing ✨ **NEW**
  - Up to 4.36x speedup on large datasets (50k+ documents)
  - Configurable worker count and chunk size
  - Automatic threshold-based activation
- **Projection**: Field selection and exclusion

#### 6. Query Cache (`pkg/cache`)
- **LRU eviction**: Least Recently Used policy with configurable capacity (1000 entries default)
- **TTL support**: Automatic expiration after 5 minutes
- **Performance**: 96x faster for cached queries (328µs → 3.4µs)
- **Thread-safe**: Concurrent access using sync.RWMutex
- **Smart invalidation**: Cache cleared on any write operation (Insert/Update/Delete)
- **Statistics**: Hit rate, misses, evictions tracking

#### 7. Aggregation Pipeline (`pkg/aggregation`)
- Stage-based processing: `$match`, `$group`, `$project`, `$sort`, `$limit`, `$skip`
- Pipeline optimization: Push down predicates, combine stages
- Aggregation operators: `$sum`, `$avg`, `$min`, `$max`, `$push`

#### 8. Database Interface (`pkg/database`)
- **Collections**: Logical grouping of documents
- **Transactions**: ACID guarantees for multi-document operations
- **Connection management**: Pool of connections for concurrent access

#### 9. Network Protocol (`pkg/protocol`)
- Custom binary protocol for client-server communication
- Request/response framing and multiplexing
- Authentication and authorization hooks

#### 10. HTTP Server (`pkg/server`)
- RESTful HTTP API similar to Elasticsearch
- JSON request/response format
- Comprehensive middleware stack (logging, CORS, recovery)
- Support for all database operations via HTTP endpoints

### Access Modes

1. **Embedded/Library Mode**: Import package directly in Go applications ✅ **Available**
2. **CLI Mode**: Interactive command-line interface (REPL) for database administration ✅ **Available**
3. **HTTP Server Mode**: RESTful API server with web-based admin console ✅ **Available**
4. **Client-Server Mode**: Standalone server with custom binary protocol ⚠️ **Not Yet Implemented**

## Project Structure

```
laura-db/
├── cmd/
│   ├── server/          # Database server executable
│   └── cli/             # Command-line client tool
├── pkg/
│   ├── document/        # Document format and BSON encoding
│   ├── storage/         # Storage engine with WAL
│   ├── mvcc/            # Multi-Version Concurrency Control
│   ├── index/           # B+ tree indexing
│   ├── query/           # Query parser and execution
│   ├── cache/           # LRU query cache with TTL
│   ├── aggregation/     # Aggregation pipeline
│   ├── database/        # Main database interface
│   ├── impex/           # Import/export utilities (JSON, CSV)
│   ├── server/          # HTTP server and handlers
│   └── protocol/        # Network protocol
├── client/              # Client library for remote access
├── examples/            # Example usage code
├── docs/                # Detailed documentation
│   ├── architecture.md  # System architecture
│   ├── storage.md       # Storage engine internals
│   ├── mvcc.md          # MVCC implementation details
│   ├── indexing.md      # Indexing algorithms
│   ├── query.md         # Query processing
│   └── http-api.md      # HTTP API reference
└── README.md
```

## Educational Goals

This project aims to teach:

1. **Storage Systems**: How databases persist data to disk efficiently
2. **Concurrency Control**: Managing concurrent access without data corruption
3. **Indexing Algorithms**: B+ trees and how they speed up queries
4. **Query Processing**: Parsing, optimizing, and executing queries
5. **Transaction Management**: ACID properties and implementation
6. **Network Protocols**: Building client-server systems
7. **System Design**: Architecture decisions and trade-offs

## Quick Start

### Embedded Mode

```go
package main

import (
    "github.com/mnohosten/laura-db/pkg/database"
)

func main() {
    // Open database
    db, err := database.Open("./data")
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // Get collection
    coll := db.Collection("users")

    // Insert document
    doc := map[string]interface{}{
        "name": "John Doe",
        "email": "john@example.com",
        "age": 30,
    }
    id, err := coll.InsertOne(doc)

    // Query documents
    results, err := coll.Find(map[string]interface{}{
        "age": map[string]interface{}{"$gte": 18},
    })

    // Create index
    err = coll.CreateIndex("email", true) // unique index
}
```

### HTTP Server Mode

Start the HTTP server with a RESTful API and web-based admin console:

```bash
# Build server
make server

# Start server
./bin/laura-server -port 8080 -data-dir ./data
```

**Access the Admin Console:**

Open your browser to http://localhost:8080/

The admin console provides a Kibana-like interface with:
- **Console**: Interactive query editor with syntax highlighting
- **Collections**: Browse and manage collections
- **Documents**: View, search, edit, and delete documents
- **Indexes**: Create and manage indexes
- **Statistics**: Real-time database statistics and metrics

**Using the API directly:**

```bash
# Insert a document
curl -X POST http://localhost:8080/users/_doc \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice", "age": 25, "city": "NYC"}'

# Query documents
curl -X POST http://localhost:8080/users/_search \
  -H "Content-Type: application/json" \
  -d '{"filter": {"age": {"$gte": 18}}}'

# Aggregate data
curl -X POST http://localhost:8080/users/_aggregate \
  -H "Content-Type: application/json" \
  -d '{
    "pipeline": [
      {"$group": {"_id": "$city", "avgAge": {"$avg": "$age"}}}
    ]
  }'

# Get database stats
curl http://localhost:8080/_stats
```

**Server Options:**
- `-host` - Server host (default: "localhost")
- `-port` - Server port (default: 8080)
- `-data-dir` - Data directory (default: "./data")
- `-buffer-size` - Buffer pool size (default: 1000)
- `-cors-origin` - CORS allowed origin (default: "*")

See [HTTP API documentation](docs/http-api.md) for complete API reference.

### CLI Mode (Interactive REPL)

Use the interactive command-line interface for database administration and development:

```bash
# Build CLI
make cli

# Run CLI with default data directory (./laura-data)
./bin/laura-cli

# Run with custom data directory
./bin/laura-cli /path/to/data
```

**Example CLI Session:**

```bash
laura> use users
Switched to collection 'users'

laura:users> insert {"name": "Alice", "age": 25}
Inserted document with _id: 507f1f77bcf86cd799439011

laura:users> find {"age": {"$gte": 20}}
Found 1 document(s):
[1] {
  "_id": "507f1f77bcf86cd799439011",
  "age": 25,
  "name": "Alice"
}

laura:users> createindex email {"unique": true}
Created index on field 'email' (unique=true)

laura:users> stats
Collection statistics for 'users': {...}
```

**Available Commands:**
- Collection operations: `insert`, `find`, `update`, `delete`, `count`
- Index management: `createindex`, `getindexes`
- Information: `stats`, `show collections`
- MongoDB-like syntax: `<collection>.find({query})`
- Help: `help` or `?`

See [CLI documentation](cmd/laura-cli/README.md) for complete command reference.

### Import/Export Utilities

LauraDB provides utilities to import and export data in JSON and CSV formats:

```go
import (
    "github.com/mnohosten/laura-db/pkg/database"
    "github.com/mnohosten/laura-db/pkg/impex"
    "os"
)

// Export to JSON
db := database.Open(database.DefaultConfig("./data"))
coll := db.Collection("users")
docs, _ := coll.Find(map[string]interface{}{})

jsonFile, _ := os.Create("users.json")
defer jsonFile.Close()
impex.Export(jsonFile, docs, impex.FormatJSON, map[string]interface{}{
    "pretty": true,  // Enable pretty-printing
})

// Export to CSV (specific fields)
csvFile, _ := os.Create("users.csv")
defer csvFile.Close()
impex.Export(csvFile, docs, impex.FormatCSV, map[string]interface{}{
    "fields": []string{"name", "age", "email"},
})

// Import from JSON
jsonFile, _ = os.Open("users.json")
defer jsonFile.Close()
importedDocs, _ := impex.Import(jsonFile, impex.FormatJSON, nil)

// Import from CSV
csvFile, _ = os.Open("users.csv")
defer csvFile.Close()
importedDocs, _ = impex.Import(csvFile, impex.FormatCSV, nil)
```

**Features:**
- **JSON Export**: Pretty-printing support, preserves all data types
- **CSV Export**: Field selection or auto-detection, handles complex types
- **JSON Import**: Smart type parsing (ObjectID, time.Time, nested structures)
- **CSV Import**: Automatic type detection (int64, float64, bool, string)
- **Round-trip Support**: Data integrity maintained through export/import cycles

See [examples/import-export](examples/import-export) for a complete working example.

### Binary Protocol Server Mode

> **⚠️ NOTE: The binary protocol server is currently NOT IMPLEMENTED.**
>
> The custom binary protocol (`pkg/protocol`) and client-server mode have not been built yet. See TODO.md for implementation status.

```bash
# Start server (NOT YET AVAILABLE)
./bin/server --data-dir ./data --port 27018

# Use CLI client (NOT YET AVAILABLE)
./bin/cli --host localhost:27018
> use mydb
> db.users.insertOne({"name": "Alice", "age": 25})
> db.users.find({"age": {"$gte": 18}})
```

## Testing and Benchmarking

LauraDB includes comprehensive testing and performance benchmarking systems.

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Generate HTML coverage report
make coverage-html

# Run with race detector
go test -race ./pkg/...
```

**Test Coverage:** 72.9% overall with 76+ passing tests across all packages.

### Performance Benchmarking

LauraDB includes automated performance benchmarking to track performance and detect regressions.

```bash
# Run all benchmarks
make bench-all

# Create performance baseline
make bench-baseline

# Check for regressions against baseline
make bench-check

# Compare two benchmark results
make bench-compare OLD=old.txt NEW=new.txt
```

**Key Features:**
- 📊 Automated CI/CD benchmarking on every PR and push
- 📈 Historical performance tracking on main branch
- 🔍 Statistical comparison with `benchstat`
- 📝 Automatic benchmark result comments on PRs
- 🎯 Daily scheduled benchmark runs
- 📦 90-365 day artifact retention

**Benchmark Coverage:**
- Collection operations (insert, find, update, delete)
- B+ tree index operations
- Query execution and optimization
- Query cache performance
- Parallel query execution
- Text search and geospatial queries
- Storage engine operations
- MVCC transaction processing

See [docs/benchmarking.md](docs/benchmarking.md) for comprehensive benchmarking documentation.

## Implementation Roadmap

- [x] Project setup
- [ ] Document format and BSON encoding
- [ ] Storage engine with WAL
- [ ] MVCC transaction system
- [ ] B+ tree indexing
- [ ] Query engine
- [ ] Aggregation pipeline
- [ ] Library API
- [ ] Network protocol and server
- [ ] Client library
- [ ] Documentation and examples

## Key Concepts Explained

### Write-Ahead Logging (WAL)
Every modification is first written to a sequential log file before being applied to the database. This ensures durability - if the system crashes, we can replay the log to recover.

### MVCC (Multi-Version Concurrency Control)
Instead of locking, we keep multiple versions of each document. Readers see a snapshot at a point in time, while writers create new versions. This allows high concurrency without blocking.

### B+ Tree Indexes
A self-balancing tree where data is only in leaf nodes, connected as a linked list. This provides O(log n) lookups and efficient range scans.

## License

MIT License - Educational purposes

## Contributing

This is an educational project. Feel free to fork and experiment!
