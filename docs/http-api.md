# HTTP API Reference

The document database provides a RESTful HTTP API similar to Elasticsearch for querying and managing documents, collections, and indexes.

## Table of Contents

- [Getting Started](#getting-started)
- [Response Format](#response-format)
- [Health & Admin Endpoints](#health--admin-endpoints)
- [Document Operations](#document-operations)
- [Query & Search](#query--search)
- [Cursor API](#cursor-api)
- [Aggregation](#aggregation)
- [Collection Management](#collection-management)
- [Index Management](#index-management)
- [WebSocket API](#websocket-api)

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

### Bulk Write

Perform multiple insert, update, and delete operations in a single request. This is more efficient than executing operations individually and allows for complex multi-operation workflows.

```bash
POST /{collection}/_bulkWrite
Content-Type: application/json

{
  "operations": [
    {
      "type": "insert",
      "document": {
        "name": "Alice",
        "age": 25
      }
    },
    {
      "type": "update",
      "filter": {"name": "Bob"},
      "update": {"$set": {"age": 36}}
    },
    {
      "type": "delete",
      "filter": {"name": "Charlie"}
    }
  ]
}
```

**Query Parameters:**
- `ordered` (optional): If set to "false", continues executing remaining operations even if one fails. Default is "true" (stops on first error).

**Response (Success):**
```json
{
  "ok": true,
  "result": {
    "insertedCount": 1,
    "modifiedCount": 1,
    "deletedCount": 1,
    "insertedIds": ["6920293a55a5f4f005000005"],
    "errors": []
  }
}
```

**Response (With Errors):**
```json
{
  "ok": false,
  "error": "BulkWriteError",
  "message": "bulk write failed at operation 1: ...",
  "code": 207,
  "insertedCount": 1,
  "modifiedCount": 0,
  "deletedCount": 0,
  "insertedIds": ["6920293a55a5f4f005000005"],
  "errors": [
    "operation 1: update requires filter and update"
  ]
}
```

**Operation Types:**
- `insert`: Inserts a new document
  - Required fields: `document`
- `update`: Updates documents matching a filter
  - Required fields: `filter`, `update`
- `delete`: Deletes documents matching a filter
  - Required fields: `filter`

**Ordered vs. Unordered:**
- **Ordered** (default): Operations are executed sequentially. If an operation fails, remaining operations are not executed.
- **Unordered** (`?ordered=false`): All operations are attempted regardless of failures. Useful for maximizing throughput when individual operation failures are acceptable.

**Example: Unordered Bulk Write**
```bash
POST /users/_bulkWrite?ordered=false
Content-Type: application/json

{
  "operations": [
    {"type": "insert", "document": {"name": "User1"}},
    {"type": "insert", "document": {"name": "User2"}},
    {"type": "insert", "document": {"_id": "duplicate"}},
    {"type": "insert", "document": {"name": "User3"}}
  ]
}
```

In unordered mode, even if the third operation fails (duplicate key), operations 1, 2, and 4 will still be executed.

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

## Cursor API

Cursors provide efficient iteration over large result sets without loading all documents into memory at once. They are ideal for pagination and processing large datasets.

### Create Cursor

Creates a new server-side cursor for iterating over query results in batches.

```bash
POST /_cursors
Content-Type: application/json

{
  "collection": "users",
  "filter": {"age": {"$gte": 18}},
  "projection": {"name": true, "email": true},
  "sort": [{"field": "name", "order": "asc"}],
  "limit": 1000,
  "skip": 0,
  "batchSize": 50,
  "timeout": "5m"
}
```

**Request Fields:**
- `collection` (required): Name of the collection to query
- `filter` (optional): Query filter (default: `{}`)
- `projection` (optional): Fields to include/exclude
- `sort` (optional): Array of sort specifications
- `limit` (optional): Maximum number of documents to return
- `skip` (optional): Number of documents to skip
- `batchSize` (optional): Documents per batch (default: 100)
- `timeout` (optional): Cursor idle timeout (default: "10m")

**Response:**
```json
{
  "ok": true,
  "result": {
    "cursorId": "8e6d04d2a499dbe294893621d5d2c04c",
    "count": 523,
    "batchSize": 50
  }
}
```

### Fetch Batch

Retrieves the next batch of documents from a cursor.

```bash
GET /_cursors/{cursorId}/batch
```

**Response:**
```json
{
  "ok": true,
  "result": {
    "documents": [
      {"_id": "...", "name": "Alice", "email": "alice@example.com"},
      {"_id": "...", "name": "Bob", "email": "bob@example.com"}
    ],
    "position": 50,
    "remaining": 473,
    "hasMore": true
  }
}
```

**Response Fields:**
- `documents`: Array of documents in this batch
- `position`: Current position in the result set
- `remaining`: Number of documents remaining
- `hasMore`: Whether there are more documents to fetch

### Close Cursor

Closes and removes a cursor to free up server resources.

```bash
DELETE /_cursors/{cursorId}
```

**Response:**
```json
{
  "ok": true,
  "result": {
    "ok": true
  }
}
```

### Example: Paginating Through Results

```bash
# Create cursor
curl -X POST http://localhost:8080/_cursors \
  -H "Content-Type: application/json" \
  -d '{
    "collection": "users",
    "filter": {"status": "active"},
    "batchSize": 100
  }'

# Response: {"ok": true, "result": {"cursorId": "abc123...", ...}}

# Fetch first batch
curl http://localhost:8080/_cursors/abc123.../batch

# Fetch second batch
curl http://localhost:8080/_cursors/abc123.../batch

# Close cursor when done
curl -X DELETE http://localhost:8080/_cursors/abc123...
```

### Cursor Lifecycle

- **Creation**: Cursor executes the query and stores results in memory
- **Idle Timeout**: Cursors expire after the specified timeout period of inactivity
- **Auto-cleanup**: Expired cursors are automatically removed every 60 seconds
- **Manual Close**: Always close cursors when finished to free resources immediately

### Best Practices

1. **Choose appropriate batch sizes**: Larger batches reduce network overhead but increase memory usage
2. **Close cursors explicitly**: Don't rely on timeout for cleanup
3. **Monitor active cursors**: Use `GET /_stats` to track cursor usage
4. **Use shorter timeouts for interactive queries**: Prevent resource leaks from abandoned cursors
5. **Consider skip/limit for simple pagination**: Cursors are best for large result sets

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

## WebSocket API

LauraDB provides a WebSocket API for receiving real-time change notifications from the database.

### Change Streams Endpoint

**WebSocket Endpoint**: `GET /_ws/watch`

Establish a WebSocket connection to receive real-time change events from the database.

#### Connection Request

After establishing the WebSocket connection, send a JSON subscription request:

```json
{
  "database": "mydb",
  "collection": "users",
  "filter": {
    "operationType": "insert"
  },
  "pipeline": [],
  "resumeToken": null
}
```

**Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `database` | string | Yes | Database name (empty = all databases) |
| `collection` | string | No | Collection name (empty = all collections) |
| `filter` | object | No | Query filter for events |
| `pipeline` | array | No | Aggregation pipeline for transforming events |
| `resumeToken` | object | No | Resume token to continue from previous position |

#### Response Messages

The server sends JSON messages with the following structure:

```json
{
  "type": "event|connected|heartbeat|error",
  "event": { /* change event */ },
  "error": "error message",
  "message": "status message"
}
```

**Message Types:**

- **`connected`**: Subscription established successfully
- **`event`**: Contains a change event
- **`heartbeat`**: Keepalive message (sent every 30 seconds)
- **`error`**: Error occurred (connection will be closed)

#### Example: JavaScript Client

```javascript
const ws = new WebSocket('ws://localhost:8080/_ws/watch');

ws.onopen = () => {
  // Subscribe to changes
  ws.send(JSON.stringify({
    database: 'mydb',
    collection: 'users',
    filter: {
      operationType: 'insert'
    }
  }));
};

ws.onmessage = (event) => {
  const response = JSON.parse(event.data);

  switch (response.type) {
    case 'connected':
      console.log('Subscribed successfully');
      break;
    case 'event':
      console.log('Change event:', response.event);
      break;
    case 'heartbeat':
      // Connection is alive
      break;
    case 'error':
      console.error('Error:', response.error);
      break;
  }
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('Connection closed');
};
```

#### Change Event Format

Change events follow the MongoDB change stream format:

**Insert Event:**
```json
{
  "_id": { "opId": 1 },
  "operationType": "insert",
  "clusterTime": "2025-01-24T10:30:00Z",
  "db": "mydb",
  "coll": "users",
  "documentKey": { "_id": "user1" },
  "fullDocument": {
    "_id": "user1",
    "name": "Alice",
    "age": 30
  }
}
```

**Update Event:**
```json
{
  "_id": { "opId": 2 },
  "operationType": "update",
  "clusterTime": "2025-01-24T10:30:01Z",
  "db": "mydb",
  "coll": "users",
  "documentKey": { "_id": "user1" },
  "updateDescription": {
    "updatedFields": { "age": 31 },
    "removedFields": ["email"]
  }
}
```

**Delete Event:**
```json
{
  "_id": { "opId": 3 },
  "operationType": "delete",
  "clusterTime": "2025-01-24T10:30:02Z",
  "db": "mydb",
  "coll": "users",
  "documentKey": { "_id": "user1" }
}
```

#### Filtering Events

Filter events by operation type:

```json
{
  "database": "mydb",
  "collection": "users",
  "filter": {
    "operationType": { "$in": ["insert", "update"] }
  }
}
```

Filter by document fields:

```json
{
  "database": "mydb",
  "collection": "users",
  "filter": {
    "fullDocument.age": { "$gte": 18 }
  }
}
```

#### Resume Tokens

Resume tokens allow you to continue watching from a specific point after reconnecting:

```json
{
  "database": "mydb",
  "collection": "users",
  "resumeToken": {
    "opId": 12345
  }
}
```

Extract the resume token from each event's `_id` field:

```javascript
ws.onmessage = (event) => {
  const response = JSON.parse(event.data);

  if (response.type === 'event') {
    const resumeToken = response.event._id;
    // Save for reconnection
    localStorage.setItem('resumeToken', JSON.stringify(resumeToken));
  }
};
```

### Use Cases

1. **Real-time Dashboards**: Update UI immediately when data changes
2. **Cache Invalidation**: Clear cache entries when documents are modified
3. **Data Synchronization**: Keep multiple systems in sync
4. **Notifications**: Send alerts when specific events occur
5. **Audit Logging**: Track all database operations in real-time

### Complete Documentation

For detailed WebSocket API documentation, including:
- Advanced filtering and pipeline transformations
- Error handling and reconnection strategies
- Performance tuning and best practices
- Complete code examples in multiple languages

See: [WebSocket API Documentation](./websocket-api.md)

### Example Program

A complete working example is available at [examples/websocket-demo](../examples/websocket-demo/)
