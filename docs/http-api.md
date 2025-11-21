# HTTP API Reference

The document database provides a RESTful HTTP API similar to Elasticsearch for querying and managing documents, collections, and indexes.

## Table of Contents

- [Getting Started](#getting-started)
- [Response Format](#response-format)
- [Health & Admin Endpoints](#health--admin-endpoints)
- [Document Operations](#document-operations)
- [Query & Search](#query--search)
- [Aggregation](#aggregation)
- [Collection Management](#collection-management)
- [Index Management](#index-management)

## Getting Started

Start the HTTP server with:

```bash
go build -o server ./cmd/server/main.go
./server -port 8080 -data-dir ./data
```

Command-line flags:
- `-host` - Server host address (default: "localhost")
- `-port` - Server port (default: 8080)
- `-data-dir` - Data directory for database storage (default: "./data")
- `-buffer-size` - Buffer pool size in pages (default: 1000)
- `-cors-origin` - CORS allowed origin (default: "*")

## Response Format

### Success Response

```json
{
  "ok": true,
  "result": { ... },
  "count": 5  // optional, for multiple results
}
```

### Error Response

```json
{
  "ok": false,
  "error": "DocumentNotFound",
  "message": "document not found",
  "code": 404
}
```

## Health & Admin Endpoints

### Health Check

Check server health and uptime.

```bash
GET /_health
```

**Response:**
```json
{
  "ok": true,
  "result": {
    "status": "healthy",
    "uptime": "5m30s",
    "time": "2025-11-21T10:00:00Z"
  }
}
```

### Database Statistics

Get comprehensive database statistics.

```bash
GET /_stats
```

**Response:**
```json
{
  "ok": true,
  "result": {
    "name": "default",
    "collections": 1,
    "active_transactions": 0,
    "collection_stats": {
      "users": {
        "name": "users",
        "count": 3,
        "indexes": 2,
        "index_details": [...]
      }
    },
    "storage_stats": {
      "buffer_pool": {
        "capacity": 1000,
        "size": 0,
        "hits": 0,
        "misses": 0,
        "hit_rate": 0,
        "evictions": 0
      },
      "disk": {
        "total_reads": 0,
        "total_writes": 0,
        "next_page_id": 0,
        "free_pages": 0
      }
    }
  }
}
```

### List Collections

List all collections in the database.

```bash
GET /_collections
```

**Response:**
```json
{
  "ok": true,
  "result": {
    "collections": ["users", "products", "orders"]
  }
}
```

## Document Operations

### Insert Document

Insert a new document with auto-generated ID.

```bash
POST /{collection}/_doc
Content-Type: application/json

{
  "name": "Alice",
  "age": 30,
  "city": "New York"
}
```

**Response:**
```json
{
  "ok": true,
  "result": {
    "id": "6920293a55a5f4f005000001",
    "collection": "users"
  }
}
```

### Insert Document with Specific ID

Insert a document with a specific ID.

```bash
POST /{collection}/_doc/{id}
Content-Type: application/json

{
  "name": "Bob",
  "age": 35
}
```

**Response:**
```json
{
  "ok": true,
  "result": {
    "id": "custom-user-123",
    "collection": "users"
  }
}
```

### Get Document

Retrieve a document by ID.

```bash
GET /{collection}/_doc/{id}
```

**Response:**
```json
{
  "ok": true,
  "result": {
    "_id": "6920293a55a5f4f005000001",
    "name": "Alice",
    "age": 30,
    "city": "New York"
  }
}
```

### Update Document

Update a document using update operators.

```bash
PUT /{collection}/_doc/{id}
Content-Type: application/json

{
  "$set": {
    "age": 31,
    "email": "alice@example.com"
  }
}
```

**Supported operators:**
- `$set` - Set field values
- `$unset` - Remove fields
- `$inc` - Increment numeric fields
- `$push` - Add to array
- `$pull` - Remove from array

**Response:**
```json
{
  "ok": true,
  "result": {
    "id": "6920293a55a5f4f005000001",
    "collection": "users"
  }
}
```

### Delete Document

Delete a document by ID.

```bash
DELETE /{collection}/_doc/{id}
```

**Response:**
```json
{
  "ok": true,
  "result": {
    "id": "6920293a55a5f4f005000001",
    "collection": "users"
  }
}
```

### Bulk Insert

Insert multiple documents at once.

```bash
POST /{collection}/_bulk
Content-Type: application/json

[
  {"name": "Alice", "age": 25},
  {"name": "Bob", "age": 35},
  {"name": "Charlie", "age": 30}
]
```

**Response:**
```json
{
  "ok": true,
  "result": {
    "ids": [
      "6920293a55a5f4f005000002",
      "6920293a55a5f4f005000003",
      "6920293a55a5f4f005000004"
    ],
    "collection": "users"
  },
  "count": 3
}
```

## Query & Search

### Search Documents

Search documents with filters, projection, sorting, and pagination.

```bash
POST /{collection}/_search
Content-Type: application/json

{
  "filter": {
    "age": {"$gte": 30}
  },
  "projection": {
    "name": true,
    "age": true
  },
  "sort": [
    {"field": "age", "order": "asc"}
  ],
  "limit": 10,
  "skip": 0
}
```

**Filter operators:**
- `$eq` - Equal
- `$ne` - Not equal
- `$gt` - Greater than
- `$gte` - Greater than or equal
- `$lt` - Less than
- `$lte` - Less than or equal
- `$in` - In array
- `$nin` - Not in array
- `$and` - Logical AND
- `$or` - Logical OR
- `$not` - Logical NOT
- `$exists` - Field exists

**Response:**
```json
{
  "ok": true,
  "result": [
    {
      "_id": "6920293a55a5f4f005000003",
      "name": "Bob",
      "age": 35
    },
    {
      "_id": "6920293a55a5f4f005000004",
      "name": "Diana",
      "age": 30
    }
  ],
  "count": 2
}
```

**Examples:**

Filter by city:
```bash
curl -X POST http://localhost:8080/users/_search \
  -H "Content-Type: application/json" \
  -d '{"filter": {"city": "NYC"}}'
```

Filter with comparison:
```bash
curl -X POST http://localhost:8080/users/_search \
  -H "Content-Type: application/json" \
  -d '{"filter": {"age": {"$gte": 30}}}'
```

Complex filter with $or:
```bash
curl -X POST http://localhost:8080/users/_search \
  -H "Content-Type: application/json" \
  -d '{
    "filter": {
      "$or": [
        {"city": "NYC"},
        {"age": {"$gte": 35}}
      ]
    }
  }'
```

### Count Documents

Count documents matching a filter.

```bash
POST /{collection}/_count
Content-Type: application/json

{
  "filter": {
    "age": {"$gte": 30}
  }
}
```

**Response:**
```json
{
  "ok": true,
  "result": {
    "collection": "users",
    "count": 2
  }
}
```

You can also use GET for counting all documents:
```bash
GET /{collection}/_count
```

## Aggregation

### Execute Aggregation Pipeline

Run MongoDB-style aggregation pipelines.

```bash
POST /{collection}/_aggregate
Content-Type: application/json

{
  "pipeline": [
    {
      "$group": {
        "_id": "$city",
        "avgAge": {"$avg": "$age"},
        "count": {"$count": {}}
      }
    }
  ]
}
```

**Supported stages:**
- `$match` - Filter documents
- `$group` - Group by field
- `$sort` - Sort results
- `$limit` - Limit results
- `$skip` - Skip results
- `$project` - Select/transform fields

**Supported accumulators:**
- `$count` - Count documents
- `$sum` - Sum values
- `$avg` - Average values
- `$min` - Minimum value
- `$max` - Maximum value
- `$first` - First value
- `$last` - Last value
- `$push` - Collect values into array

**Response:**
```json
{
  "ok": true,
  "result": [
    {
      "_id": "NYC",
      "avgAge": 25,
      "count": 1
    },
    {
      "_id": "LA",
      "avgAge": 35,
      "count": 1
    },
    {
      "_id": "Chicago",
      "avgAge": 30,
      "count": 1
    }
  ],
  "count": 3
}
```

**Examples:**

Average age by city:
```bash
curl -X POST http://localhost:8080/users/_aggregate \
  -H "Content-Type: application/json" \
  -d '{
    "pipeline": [
      {
        "$group": {
          "_id": "$city",
          "avgAge": {"$avg": "$age"},
          "count": {"$count": {}}
        }
      }
    ]
  }'
```

Filter then aggregate:
```bash
curl -X POST http://localhost:8080/users/_aggregate \
  -H "Content-Type: application/json" \
  -d '{
    "pipeline": [
      {
        "$match": {"age": {"$gte": 25}}
      },
      {
        "$group": {
          "_id": "$city",
          "totalUsers": {"$count": {}}
        }
      }
    ]
  }'
```

## Collection Management

### Create Collection

Explicitly create a collection.

```bash
PUT /{collection}
```

**Response:**
```json
{
  "ok": true,
  "result": {
    "collection": "users"
  }
}
```

Note: Collections are also created implicitly when you insert the first document.

### Drop Collection

Delete a collection and all its documents.

```bash
DELETE /{collection}
```

**Response:**
```json
{
  "ok": true,
  "result": {
    "collection": "users"
  }
}
```

### Get Collection Statistics

Get statistics for a specific collection.

```bash
GET /{collection}/_stats
```

**Response:**
```json
{
  "ok": true,
  "result": {
    "name": "users",
    "count": 3,
    "indexes": 2,
    "index_details": [
      {
        "name": "_id_",
        "type": 0,
        "field_path": "_id",
        "unique": true,
        "size": 3,
        "height": 1
      },
      {
        "name": "email_1",
        "type": 0,
        "field_path": "email",
        "unique": true,
        "size": 1,
        "height": 1
      }
    ]
  }
}
```

## Index Management

### Create Index

Create an index on a field to improve query performance.

```bash
POST /{collection}/_index
Content-Type: application/json

{
  "field": "email",
  "unique": true
}
```

**Response:**
```json
{
  "ok": true,
  "result": {
    "collection": "users",
    "field": "email",
    "unique": true
  }
}
```

### List Indexes

List all indexes on a collection.

```bash
GET /{collection}/_index
```

**Response:**
```json
{
  "ok": true,
  "result": {
    "collection": "users",
    "indexes": [
      {
        "name": "_id_",
        "type": 0,
        "field_path": "_id",
        "unique": true,
        "size": 3,
        "height": 1
      },
      {
        "name": "email_1",
        "type": 0,
        "field_path": "email",
        "unique": true,
        "size": 1,
        "height": 1
      }
    ]
  }
}
```

### Drop Index

Delete an index by name.

```bash
DELETE /{collection}/_index/{name}
```

**Response:**
```json
{
  "ok": true,
  "result": {
    "collection": "users",
    "index": "email_1"
  }
}
```

## Error Codes

| HTTP Code | Error Type | Description |
|-----------|------------|-------------|
| 400 | BadRequest | Invalid request body or parameters |
| 404 | DocumentNotFound | Document not found |
| 404 | CollectionNotFound | Collection not found |
| 409 | DuplicateKey | Unique constraint violation |
| 500 | InternalError | Internal server error |

## Complete Example Session

Here's a complete example demonstrating common operations:

```bash
# Start server
./server -port 8080 -data-dir ./data

# Insert documents
curl -X POST http://localhost:8080/users/_bulk \
  -H "Content-Type: application/json" \
  -d '[
    {"name": "Alice", "age": 25, "city": "NYC", "email": "alice@example.com"},
    {"name": "Bob", "age": 35, "city": "LA", "email": "bob@example.com"},
    {"name": "Charlie", "age": 30, "city": "Chicago", "email": "charlie@example.com"}
  ]'

# Create unique index on email
curl -X POST http://localhost:8080/users/_index \
  -H "Content-Type: application/json" \
  -d '{"field": "email", "unique": true}'

# Search for users in NYC
curl -X POST http://localhost:8080/users/_search \
  -H "Content-Type: application/json" \
  -d '{"filter": {"city": "NYC"}}'

# Get average age by city
curl -X POST http://localhost:8080/users/_aggregate \
  -H "Content-Type: application/json" \
  -d '{
    "pipeline": [
      {
        "$group": {
          "_id": "$city",
          "avgAge": {"$avg": "$age"},
          "count": {"$count": {}}
        }
      }
    ]
  }'

# Update a document
curl -X PUT http://localhost:8080/users/_doc/{id} \
  -H "Content-Type: application/json" \
  -d '{"$set": {"age": 26}}'

# Get collection statistics
curl http://localhost:8080/users/_stats

# Get database statistics
curl http://localhost:8080/_stats
```

## Middleware & Features

### CORS Support

CORS is enabled by default with the origin specified by the `-cors-origin` flag.

### Request Logging

All requests are logged with method, path, status code, and duration.

### Request Size Limit

Requests are limited to 10MB by default to prevent memory issues.

### Compression

Response compression is enabled automatically for large responses.

### Graceful Shutdown

The server handles SIGINT and SIGTERM signals for graceful shutdown, allowing active requests to complete.

## Configuration

Server configuration can be customized via command-line flags or by modifying `pkg/server/config.go`:

```go
type Config struct {
    Host            string
    Port            int
    ReadTimeout     time.Duration
    WriteTimeout    time.Duration
    IdleTimeout     time.Duration
    MaxRequestSize  int64
    EnableCORS      bool
    AllowedOrigins  []string
    AllowedMethods  []string
    AllowedHeaders  []string
    EnableLogging   bool
    LogFormat       string
}
```
