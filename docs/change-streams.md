# Change Streams

Change Streams allow applications to access real-time data changes without the complexity of tailing an oplog. Applications can use change streams to subscribe to all data changes on a collection, database, or entire deployment, and immediately react to them.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Architecture](#architecture)
- [Usage](#usage)
- [Change Events](#change-events)
- [Resume Tokens](#resume-tokens)
- [Filtering](#filtering)
- [Pipeline Transformations](#pipeline-transformations)
- [Performance](#performance)
- [Examples](#examples)
- [Best Practices](#best-practices)

## Overview

Change Streams provide a unified interface for watching changes across collections. They are built on top of LauraDB's operation log (oplog) and provide:

- **Real-time notifications**: Receive events as changes occur
- **Resume capability**: Resume watching from a specific point in time
- **Filtering**: Watch only specific types of changes
- **Pipeline support**: Transform events using aggregation-like pipelines

## Features

### Core Functionality

1. **Watch Operations**
   - Insert operations
   - Update operations
   - Delete operations
   - Collection operations (create, drop)
   - Index operations (create, drop)

2. **Scope Control**
   - Watch entire database
   - Watch specific collection
   - Watch all collections

3. **Resume Tokens**
   - Opaque tokens for resuming streams
   - Survive process restarts
   - Guarantee ordering

4. **Filtering**
   - Filter by operation type
   - Filter by document fields
   - Custom query filters

5. **Pipeline Transformations**
   - $match stage for filtering
   - Extensible for future stages

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Application                          │
└────────────┬───────────────────────────────────────────┘
             │ Subscribe to changes
             ▼
┌─────────────────────────────────────────────────────────┐
│                  Change Stream                          │
│  ┌───────────┐  ┌─────────────┐  ┌─────────────────┐  │
│  │  Watcher  │─▶│   Filters   │─▶│  Event Channel  │  │
│  └───────────┘  └─────────────┘  └─────────────────┘  │
└────────────┬───────────────────────────────────────────┘
             │ Poll for new entries
             ▼
┌─────────────────────────────────────────────────────────┐
│                      Oplog                              │
│  ┌────────────────────────────────────────────────────┐│
│  │  OpID │ Timestamp │ Type │ Database │ Collection  ││
│  ├────────────────────────────────────────────────────┤│
│  │   1   │  10:00:00 │ ins  │   testdb │   users     ││
│  │   2   │  10:00:01 │ upd  │   testdb │   users     ││
│  │   3   │  10:00:02 │ del  │   testdb │   products  ││
│  └────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────┘
```

### Components

1. **ChangeStream**: Main structure managing the stream lifecycle
2. **Watcher Loop**: Background goroutine polling oplog for new entries
3. **Filters**: Query-based filtering of events
4. **Event Channel**: Buffered channel for delivering events to consumers
5. **Resume Token**: OpID-based token for resuming streams

## Usage

### Basic Change Stream

```go
import (
    "context"
    "github.com/mnohosten/laura-db/pkg/changestream"
    "github.com/mnohosten/laura-db/pkg/replication"
)

// Create oplog
oplog, _ := replication.NewOplog("/path/to/oplog.bin")

// Create change stream watching all collections
cs := changestream.NewChangeStream(oplog, "mydb", "", nil)

// Start watching
cs.Start()
defer cs.Close()

// Consume events
ctx := context.Background()
for {
    event, err := cs.Next(ctx)
    if err != nil {
        break
    }

    fmt.Printf("Operation: %s, Collection: %s\n",
        event.OperationType, event.Collection)
}
```

### Watch Specific Collection

```go
// Watch only "users" collection
cs := changestream.NewChangeStream(oplog, "mydb", "users", nil)
cs.Start()
defer cs.Close()

ctx := context.Background()
event, _ := cs.Next(ctx)
fmt.Printf("User changed: %v\n", event.DocumentKey)
```

### Non-blocking Consumption

```go
cs := changestream.NewChangeStream(oplog, "mydb", "users", nil)
cs.Start()
defer cs.Close()

// Non-blocking check
event, err := cs.TryNext()
if event != nil {
    fmt.Printf("Event received: %v\n", event.OperationType)
} else {
    fmt.Println("No event available")
}
```

### Using Event Channels

```go
cs := changestream.NewChangeStream(oplog, "mydb", "users", nil)
cs.Start()
defer cs.Close()

// Direct channel access
for event := range cs.Events() {
    fmt.Printf("Event: %v\n", event.OperationType)
}
```

## Change Events

### Event Structure

```go
type ChangeEvent struct {
    // Resume token for this event
    ID ResumeToken

    // Type of operation
    OperationType OperationType

    // When the operation occurred
    Timestamp time.Time

    // Namespace
    Database   string
    Collection string

    // Document identifier
    DocumentKey map[string]interface{}

    // Full document (for inserts)
    FullDocument map[string]interface{}

    // Update details (for updates)
    UpdateDescription *UpdateDescription

    // Index details (for index operations)
    IndexDefinition map[string]interface{}
}
```

### Operation Types

- `insert`: Document inserted
- `update`: Document updated
- `delete`: Document deleted
- `replace`: Document replaced
- `drop`: Collection dropped
- `dropDatabase`: Database dropped
- `rename`: Collection renamed
- `createIndex`: Index created
- `dropIndex`: Index dropped
- `createCollection`: Collection created

### Insert Event

```json
{
    "_id": {"opId": 1},
    "operationType": "insert",
    "clusterTime": "2025-01-15T10:00:00Z",
    "db": "testdb",
    "coll": "users",
    "documentKey": {"_id": "user1"},
    "fullDocument": {
        "_id": "user1",
        "name": "Alice",
        "age": 30
    }
}
```

### Update Event

```json
{
    "_id": {"opId": 2},
    "operationType": "update",
    "clusterTime": "2025-01-15T10:00:01Z",
    "db": "testdb",
    "coll": "users",
    "documentKey": {"_id": "user1"},
    "updateDescription": {
        "updatedFields": {"age": 31},
        "removedFields": ["email"]
    }
}
```

### Delete Event

```json
{
    "_id": {"opId": 3},
    "operationType": "delete",
    "clusterTime": "2025-01-15T10:00:02Z",
    "db": "testdb",
    "coll": "users",
    "documentKey": {"_id": "user1"}
}
```

## Resume Tokens

Resume tokens allow you to resume a change stream from a specific point in time. This is useful for:

- Recovering from crashes
- Processing events in batches
- Implementing at-least-once delivery semantics

### Basic Resume

```go
// First stream
cs1 := changestream.NewChangeStream(oplog, "mydb", "users", nil)
cs1.Start()

event, _ := cs1.Next(context.Background())
resumeToken := event.ID
cs1.Close()

// Resume from token
options := changestream.DefaultChangeStreamOptions()
options.ResumeAfter = &resumeToken

cs2 := changestream.NewChangeStream(oplog, "mydb", "users", options)
cs2.Start()

// Will receive events after the saved token
nextEvent, _ := cs2.Next(context.Background())
```

### Getting Current Resume Token

```go
cs := changestream.NewChangeStream(oplog, "mydb", "users", nil)
cs.Start()

event, _ := cs.Next(context.Background())

// Get current position
token := cs.ResumeToken()
fmt.Printf("Current position: OpID=%d\n", token.OpID)
```

## Filtering

### Operation Type Filter

```go
cs := changestream.NewChangeStream(oplog, "mydb", "users", nil)

// Only watch insert operations
filter := map[string]interface{}{
    "operationType": "insert",
}
cs.SetFilter(filter)
cs.Start()
```

### Document Field Filter

```go
// Watch changes to specific document
filter := map[string]interface{}{
    "documentKey._id": "user1",
}
cs.SetFilter(filter)
```

### Complex Filters

```go
// Multiple conditions
filter := map[string]interface{}{
    "$and": []interface{}{
        map[string]interface{}{"operationType": "update"},
        map[string]interface{}{"collection": "users"},
    },
}
cs.SetFilter(filter)
```

## Pipeline Transformations

Change streams support aggregation-style pipelines for transforming events.

### $match Stage

```go
options := changestream.DefaultChangeStreamOptions()
options.Pipeline = []map[string]interface{}{
    {
        "$match": map[string]interface{}{
            "operationType": map[string]interface{}{
                "$in": []string{"insert", "update"},
            },
        },
    },
}

cs := changestream.NewChangeStream(oplog, "mydb", "users", options)
```

### Multiple Stages

```go
options := changestream.DefaultChangeStreamOptions()
options.Pipeline = []map[string]interface{}{
    {
        "$match": map[string]interface{}{
            "operationType": "insert",
        },
    },
    {
        "$match": map[string]interface{}{
            "database": "testdb",
        },
    },
}
```

## Performance

### Configuration Options

```go
options := &changestream.ChangeStreamOptions{
    // Full document inclusion strategy
    FullDocument: changestream.FullDocumentDefault,

    // Resume from specific point
    ResumeAfter: nil,

    // Maximum wait time for new changes (polling interval)
    MaxAwaitTime: 1 * time.Second,

    // Event buffer size
    BatchSize: 100,

    // Transformation pipeline
    Pipeline: nil,
}
```

### Tuning Parameters

1. **MaxAwaitTime**: Polling interval
   - Lower = more responsive, higher CPU
   - Higher = less CPU, less responsive
   - Default: 1 second

2. **BatchSize**: Event buffer size
   - Larger = better throughput
   - Smaller = lower latency
   - Default: 100 events

### Performance Characteristics

- **Latency**: ~1-2 seconds (depends on MaxAwaitTime)
- **Throughput**: Thousands of events per second
- **Memory**: ~10KB per change stream + buffer
- **CPU**: Minimal (polling-based)

## Examples

### Example 1: Audit Log

```go
// Watch all operations for auditing
cs := changestream.NewChangeStream(oplog, "", "", nil)
cs.Start()
defer cs.Close()

logFile, _ := os.Create("audit.log")
defer logFile.Close()

ctx := context.Background()
for {
    event, err := cs.Next(ctx)
    if err != nil {
        break
    }

    logEntry := fmt.Sprintf("[%s] %s on %s.%s\n",
        event.Timestamp, event.OperationType,
        event.Database, event.Collection)
    logFile.WriteString(logEntry)
}
```

### Example 2: Cache Invalidation

```go
// Watch user collection and invalidate cache on changes
cs := changestream.NewChangeStream(oplog, "mydb", "users", nil)
cs.Start()
defer cs.Close()

cache := make(map[string]interface{})

go func() {
    ctx := context.Background()
    for {
        event, err := cs.Next(ctx)
        if err != nil {
            return
        }

        // Invalidate cache entry
        if docID, ok := event.DocumentKey["_id"].(string); ok {
            delete(cache, docID)
            fmt.Printf("Cache invalidated for user: %s\n", docID)
        }
    }
}()
```

### Example 3: Real-time Sync

```go
// Sync changes to remote system
cs := changestream.NewChangeStream(oplog, "mydb", "users", nil)
cs.Start()
defer cs.Close()

ctx := context.Background()
for {
    event, err := cs.Next(ctx)
    if err != nil {
        break
    }

    switch event.OperationType {
    case changestream.OperationTypeInsert:
        remoteDB.Insert(event.FullDocument)
    case changestream.OperationTypeUpdate:
        remoteDB.Update(event.DocumentKey, event.UpdateDescription)
    case changestream.OperationTypeDelete:
        remoteDB.Delete(event.DocumentKey)
    }
}
```

## Best Practices

### 1. Always Use Resume Tokens

```go
// Periodically save resume tokens
ticker := time.NewTicker(10 * time.Second)
defer ticker.Stop()

go func() {
    for range ticker.C {
        token := cs.ResumeToken()
        saveTokenToDisk(token)
    }
}()
```

### 2. Handle Errors Gracefully

```go
for {
    event, err := cs.Next(ctx)
    if err != nil {
        if errors.Is(err, context.Canceled) {
            // Normal shutdown
            break
        }

        log.Printf("Change stream error: %v", err)
        time.Sleep(5 * time.Second)
        continue
    }

    // Process event
}
```

### 3. Use Filters Early

```go
// Filter at change stream level (efficient)
filter := map[string]interface{}{
    "operationType": "insert",
}
cs.SetFilter(filter)

// Instead of filtering in application code (inefficient)
```

### 4. Close Streams Properly

```go
// Use defer for cleanup
cs := changestream.NewChangeStream(oplog, "mydb", "users", nil)
cs.Start()
defer cs.Close()

// Or use context for cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go func() {
    event, _ := cs.Next(ctx)
    // Process event
}()
```

### 5. Buffer Size Selection

```go
options := changestream.DefaultChangeStreamOptions()

// High-throughput scenario
options.BatchSize = 1000

// Low-latency scenario
options.BatchSize = 10

cs := changestream.NewChangeStream(oplog, "mydb", "users", options)
```

### 6. Monitor for Stale Streams

```go
// Detect if stream hasn't received events for too long
lastEventTime := time.Now()

go func() {
    for {
        event, err := cs.Next(ctx)
        if err != nil {
            return
        }

        lastEventTime = time.Now()
        // Process event
    }
}()

// Monitor goroutine
go func() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        if time.Since(lastEventTime) > 5*time.Minute {
            log.Println("Warning: No events received in 5 minutes")
        }
    }
}()
```

## Limitations

1. **Ordering**: Events are ordered by OpID within a single change stream
2. **Buffering**: Events may be dropped if buffer is full and consumer is slow
3. **TTL**: Oplog entries may be trimmed, making old resume tokens invalid
4. **Pipeline**: Currently only supports $match stage
5. **Cluster-wide**: Change streams are local to a single database instance

## Future Enhancements

- Full aggregation pipeline support ($project, $group, etc.)
- Cluster-wide change streams
- Configurable oplog retention
- Pre-image and post-image support
- Distributed resume tokens
- Change stream cursors

## References

- [MongoDB Change Streams Documentation](https://docs.mongodb.com/manual/changeStreams/)
- [LauraDB Oplog Implementation](./replication.md)
- [Query Engine](./query-engine.md)
- [Examples](../examples/changestream-demo/)
