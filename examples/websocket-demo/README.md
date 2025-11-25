# WebSocket Change Streams Demo

This example demonstrates how to use LauraDB's WebSocket API to receive real-time change notifications from the database.

## Overview

WebSocket change streams allow you to:
- Subscribe to real-time data changes across collections
- Filter events by operation type (insert, update, delete)
- Receive immediate notifications when documents are modified
- Build reactive applications that respond to database changes

## Prerequisites

1. LauraDB server must be running:
   ```bash
   # Build the server
   make server

   # Start the server
   ./bin/laura-server
   ```

2. The server will start on `http://localhost:8080` by default

## Running the Demo

```bash
# Build and run the demo
cd examples/websocket-demo
go run main.go

# Or specify a custom server URL
go run main.go localhost:9090
```

## What This Demo Does

1. **Connects to WebSocket endpoint**: Establishes a WebSocket connection to `ws://localhost:8080/_ws/watch`
2. **Subscribes to changes**: Requests change notifications for the `testdb.users` collection
3. **Displays events**: Shows all change events (insert, update, delete) in real-time
4. **Handles heartbeats**: Automatically processes keepalive messages from the server

## Testing the Demo

### Terminal 1: Start the WebSocket client
```bash
go run main.go
```

### Terminal 2: Generate change events

**Insert a document:**
```bash
curl -X POST http://localhost:8080/users/_doc \
  -H 'Content-Type: application/json' \
  -d '{"name":"Alice","age":30}'
```

**Update a document:**
```bash
curl -X PUT http://localhost:8080/users/_doc/USER_ID \
  -H 'Content-Type: application/json' \
  -d '{"$set":{"age":31}}'
```

**Delete a document:**
```bash
curl -X DELETE http://localhost:8080/users/_doc/USER_ID
```

You should see change events appear in Terminal 1 in real-time!

## Example Output

```
=== LauraDB WebSocket Change Streams Demo ===

Connecting to ws://localhost:8080/_ws/watch
✅ Connected to WebSocket server

📡 Subscribing to changes:
   Database: testdb
   Collection: users

🔍 Watching for changes... (Press Ctrl+C to exit)

✅ Change stream connected successfully

📨 Change Event Received:
─────────────────────────────────────
   {
     "_id": {
       "opId": 1
     },
     "operationType": "insert",
     "clusterTime": "2025-01-24T10:30:45Z",
     "db": "testdb",
     "coll": "users",
     "documentKey": {
       "_id": "67891234567890abcdef0001"
     },
     "fullDocument": {
       "_id": "67891234567890abcdef0001",
       "name": "Alice",
       "age": 30
     }
   }
─────────────────────────────────────
```

## Filtering Events

You can filter events by operation type by modifying the `Filter` field in the request:

```go
req := ChangeStreamRequest{
    Database:   "testdb",
    Collection: "users",
    Filter: map[string]interface{}{
        "operationType": "insert",  // Only receive insert events
    },
}
```

## Advanced Usage

### Watch All Collections

Set `Collection` to an empty string to watch all collections in a database:

```go
req := ChangeStreamRequest{
    Database:   "testdb",
    Collection: "",  // Watch all collections
}
```

### Watch All Databases

Set both `Database` and `Collection` to empty strings:

```go
req := ChangeStreamRequest{
    Database:   "",  // Watch all databases
    Collection: "",  // Watch all collections
}
```

### Use Pipeline Transformations

Apply aggregation-style pipelines to filter and transform events:

```go
req := ChangeStreamRequest{
    Database:   "testdb",
    Collection: "users",
    Pipeline: []map[string]interface{}{
        {
            "$match": map[string]interface{}{
                "operationType": map[string]interface{}{
                    "$in": []string{"insert", "update"},
                },
            },
        },
    },
}
```

## Message Types

The WebSocket server sends different types of messages:

- **`connected`**: Sent immediately after subscription is successful
- **`event`**: Contains a change event with operation details
- **`heartbeat`**: Keepalive message sent every 30 seconds
- **`error`**: Indicates an error occurred (connection will be closed)

## Error Handling

The demo includes proper error handling:
- Connection failures are logged
- Unexpected disconnections are handled gracefully
- Interrupt signals (Ctrl+C) trigger clean shutdown

## Learn More

- [Change Streams Documentation](../../docs/change-streams.md)
- [WebSocket API Documentation](../../docs/websocket-api.md)
- [HTTP API Documentation](../../docs/http-api.md)

## Notes

- WebSocket connections are long-lived; keep them open to receive continuous updates
- The server automatically closes inactive connections after a timeout
- Use heartbeat messages to detect connection issues
- Always handle the `connected` acknowledgment before processing events
