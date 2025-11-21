# LauraDB 📚

An educational implementation of a MongoDB-like document database in Go.

LauraDB is a fast, embedded document database with a REST API and web-based admin console.

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
- **Query optimizer**: Selects best execution strategy (index scan vs collection scan)
- **Projection**: Field selection and exclusion

#### 6. Aggregation Pipeline (`pkg/aggregation`)
- Stage-based processing: `$match`, `$group`, `$project`, `$sort`, `$limit`, `$skip`
- Pipeline optimization: Push down predicates, combine stages
- Aggregation operators: `$sum`, `$avg`, `$min`, `$max`, `$push`

#### 7. Database Interface (`pkg/database`)
- **Collections**: Logical grouping of documents
- **Transactions**: ACID guarantees for multi-document operations
- **Connection management**: Pool of connections for concurrent access

#### 8. Network Protocol (`pkg/protocol`)
- Custom binary protocol for client-server communication
- Request/response framing and multiplexing
- Authentication and authorization hooks

#### 9. HTTP Server (`pkg/server`)
- RESTful HTTP API similar to Elasticsearch
- JSON request/response format
- Comprehensive middleware stack (logging, CORS, recovery)
- Support for all database operations via HTTP endpoints

### Access Modes

1. **Embedded/Library Mode**: Import package directly in Go applications
2. **HTTP Server Mode**: RESTful API server for language-agnostic access
3. **Client-Server Mode**: Standalone server with custom binary protocol

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
│   ├── aggregation/     # Aggregation pipeline
│   ├── database/        # Main database interface
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
go build -o server ./cmd/server/main.go

# Start server
./server -port 8080 -data-dir ./data
```

**Access the Admin Console:**

Open your browser to http://localhost:8080/admin/

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

### Binary Protocol Server Mode

```bash
# Start server
./bin/server --data-dir ./data --port 27018

# Use CLI client
./bin/cli --host localhost:27018
> use mydb
> db.users.insertOne({"name": "Alice", "age": 25})
> db.users.find({"age": {"$gte": 18}})
```

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
