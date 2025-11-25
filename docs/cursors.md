# Cursor API

LauraDB provides cursor support for efficiently iterating over large result sets without loading all documents into memory at once. Cursors are particularly useful for:

- **Large result sets**: Processing thousands of documents without memory exhaustion
- **Pagination**: Implementing REST API pagination with server-side state
- **Streaming data**: Processing query results in chunks
- **Resource efficiency**: Fetching only the data you need when you need it

## Table of Contents

- [Basic Usage](#basic-usage)
- [Cursor Options](#cursor-options)
- [Cursor Methods](#cursor-methods)
- [Cursor Manager](#cursor-manager)
- [Best Practices](#best-practices)
- [Examples](#examples)
- [HTTP API Integration](#http-api-integration)

## Basic Usage

### Creating a Cursor

```go
// Simple cursor with default options
cursor, err := collection.FindCursor(filter, nil)
if err != nil {
    log.Fatal(err)
}
defer cursor.Close()

// Iterate through results
for cursor.HasNext() {
    doc, err := cursor.Next()
    if err != nil {
        log.Fatal(err)
    }
    // Process document
    fmt.Println(doc)
}
```

### With Custom Options

```go
options := &database.CursorOptions{
    BatchSize: 50,                // Fetch 50 documents per batch
    Timeout:   5 * time.Minute,   // Cursor expires after 5 minutes of inactivity
}

cursor, err := collection.FindCursor(filter, options)
if err != nil {
    log.Fatal(err)
}
defer cursor.Close()
```

### With Query Options

```go
queryOptions := &database.QueryOptions{
    Projection: map[string]bool{"name": true, "email": true},
    Sort:       []query.SortField{{Field: "name", Ascending: true}},
    Limit:      100,
    Skip:       0,
}

cursorOptions := &database.CursorOptions{
    BatchSize: 25,
    Timeout:   10 * time.Minute,
}

cursor, err := collection.FindCursorWithOptions(filter, queryOptions, cursorOptions)
if err != nil {
    log.Fatal(err)
}
defer cursor.Close()
```

## Cursor Options

### CursorOptions

```go
type CursorOptions struct {
    BatchSize int           // Documents per batch (default: 100)
    Timeout   time.Duration // Idle timeout (default: 10 minutes)
}
```

**BatchSize**: Number of documents to fetch in each batch. Larger batch sizes reduce the number of iterations but increase memory usage per batch.

**Timeout**: Duration of inactivity before the cursor is automatically closed and removed from the cursor manager. The timeout is reset on each cursor access.

### Default Options

```go
defaultOptions := database.DefaultCursorOptions()
// Returns: &CursorOptions{BatchSize: 100, Timeout: 10 * time.Minute}
```

## Cursor Methods

### ID() string
Returns the cursor's unique identifier (32-character hex string).

```go
cursorID := cursor.ID()
fmt.Printf("Cursor ID: %s\n", cursorID)
```

### HasNext() bool
Returns true if there are more documents to fetch.

```go
for cursor.HasNext() {
    doc, err := cursor.Next()
    // ...
}
```

### Next() (*document.Document, error)
Returns the next document in the result set. Returns an error if the cursor is exhausted or closed.

```go
doc, err := cursor.Next()
if err != nil {
    log.Fatal(err)
}
```

### NextBatch() ([]*document.Document, error)
Returns the next batch of documents (up to `BatchSize` documents). Returns an empty slice when no more documents are available.

```go
for !cursor.IsExhausted() {
    batch, err := cursor.NextBatch()
    if err != nil {
        log.Fatal(err)
    }
    if len(batch) == 0 {
        break // No more documents
    }

    // Process batch
    for _, doc := range batch {
        fmt.Println(doc)
    }
}
```

### Count() int
Returns the total number of documents in the result set.

```go
total := cursor.Count()
fmt.Printf("Total documents: %d\n", total)
```

### Position() int
Returns the current position (number of documents already fetched).

```go
position := cursor.Position()
fmt.Printf("Fetched %d of %d documents\n", position, cursor.Count())
```

### Remaining() int
Returns the number of documents remaining to be fetched.

```go
remaining := cursor.Remaining()
fmt.Printf("%d documents remaining\n", remaining)
```

### IsExhausted() bool
Returns true if all documents have been fetched or the cursor is closed.

```go
if cursor.IsExhausted() {
    fmt.Println("Cursor exhausted")
}
```

### Close()
Closes the cursor and releases resources. Always defer cursor.Close() after creating a cursor.

```go
cursor.Close()
```

### IsTimedOut() bool
Returns true if the cursor has exceeded its idle timeout. Typically used internally by the cursor manager.

```go
if cursor.IsTimedOut() {
    fmt.Println("Cursor has timed out")
}
```

## Cursor Manager

The `CursorManager` maintains server-side cursors, enabling cursor persistence across requests (useful for HTTP APIs).

### Accessing the Cursor Manager

```go
db, _ := database.Open(config)
manager := db.CursorManager()
```

### Creating a Managed Cursor

```go
cursor, err := manager.CreateCursor(collection, query, options)
if err != nil {
    log.Fatal(err)
}

cursorID := cursor.ID()
fmt.Printf("Created cursor: %s\n", cursorID)
```

### Retrieving a Cursor by ID

```go
cursor, err := manager.GetCursor(cursorID)
if err != nil {
    log.Fatal(err) // Cursor not found or timed out
}
```

### Closing a Managed Cursor

```go
err := manager.CloseCursor(cursorID)
if err != nil {
    log.Fatal(err)
}
```

### Cleanup Operations

```go
// Get active cursor count
active := manager.ActiveCursors()
fmt.Printf("Active cursors: %d\n", active)

// Manual cleanup of timed-out cursors
removed := manager.CleanupTimedOutCursors()
fmt.Printf("Removed %d timed-out cursors\n", removed)
```

**Note**: LauraDB automatically runs cursor cleanup every 60 seconds to remove timed-out and exhausted cursors.

## Best Practices

### 1. Always Close Cursors

Use `defer cursor.Close()` immediately after creating a cursor to ensure resources are released:

```go
cursor, err := collection.FindCursor(filter, options)
if err != nil {
    return err
}
defer cursor.Close() // Always defer close
```

### 2. Choose Appropriate Batch Sizes

- **Small batches (10-50)**: Better for real-time streaming, lower memory usage
- **Medium batches (100-500)**: Balanced performance, good for most use cases
- **Large batches (1000+)**: Faster processing, higher memory usage

```go
// For streaming to web clients
options := &database.CursorOptions{BatchSize: 25}

// For batch processing
options := &database.CursorOptions{BatchSize: 500}
```

### 3. Set Appropriate Timeouts

- **Short timeouts (1-5 min)**: For interactive web applications
- **Long timeouts (10-30 min)**: For long-running batch processes

```go
// Web API cursor
options := &database.CursorOptions{
    BatchSize: 50,
    Timeout:   2 * time.Minute,
}

// Background job cursor
options := &database.CursorOptions{
    BatchSize: 1000,
    Timeout:   30 * time.Minute,
}
```

### 4. Monitor Cursor Usage

In production environments, monitor active cursor counts:

```go
manager := db.CursorManager()
active := manager.ActiveCursors()

if active > 1000 {
    log.Printf("Warning: High cursor count: %d\n", active)
}
```

### 5. Handle Exhausted Cursors Gracefully

```go
for cursor.HasNext() {
    doc, err := cursor.Next()
    if err != nil {
        if cursor.IsExhausted() {
            break // Normal completion
        }
        return err // Actual error
    }
    // Process doc
}
```

## Examples

### Example 1: Basic Iteration

```go
cursor, _ := collection.FindCursor(map[string]interface{}{}, nil)
defer cursor.Close()

count := 0
for cursor.HasNext() {
    doc, _ := cursor.Next()
    fmt.Printf("%d. %v\n", count+1, doc)
    count++
}
fmt.Printf("Processed %d documents\n", count)
```

### Example 2: Batch Processing

```go
options := &database.CursorOptions{BatchSize: 100}
cursor, _ := collection.FindCursor(filter, options)
defer cursor.Close()

for !cursor.IsExhausted() {
    batch, _ := cursor.NextBatch()
    if len(batch) == 0 {
        break
    }

    // Process batch in parallel
    var wg sync.WaitGroup
    for _, doc := range batch {
        wg.Add(1)
        go func(d *document.Document) {
            defer wg.Done()
            processDocument(d)
        }(doc)
    }
    wg.Wait()
}
```

### Example 3: Pagination API

```go
func paginateResults(w http.ResponseWriter, r *http.Request) {
    cursorID := r.URL.Query().Get("cursor")

    var cursor *database.Cursor
    var err error

    if cursorID == "" {
        // Create new cursor
        cursor, err = manager.CreateCursor(collection, nil, &database.CursorOptions{
            BatchSize: 20,
            Timeout:   5 * time.Minute,
        })
    } else {
        // Retrieve existing cursor
        cursor, err = manager.GetCursor(cursorID)
    }

    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Fetch next page
    batch, _ := cursor.NextBatch()

    response := map[string]interface{}{
        "data":     batch,
        "cursor":   cursor.ID(),
        "hasMore":  cursor.HasNext(),
        "position": cursor.Position(),
        "total":    cursor.Count(),
    }

    json.NewEncoder(w).Encode(response)
}
```

### Example 4: Filtered Cursor with Progress

```go
filter := map[string]interface{}{
    "status": "pending",
}

cursor, _ := collection.FindCursor(filter, nil)
defer cursor.Close()

fmt.Printf("Processing %d pending items\n", cursor.Count())

for cursor.HasNext() {
    doc, _ := cursor.Next()

    // Show progress
    progress := float64(cursor.Position()) / float64(cursor.Count()) * 100
    fmt.Printf("Progress: %.1f%% (%d/%d)\n",
        progress, cursor.Position(), cursor.Count())

    // Process document
    processDocument(doc)
}
```

### Example 5: Resumable Processing

```go
func processWithResume() error {
    manager := db.CursorManager()

    // Try to resume from saved cursor
    cursorID := loadCheckpoint()

    var cursor *database.Cursor
    var err error

    if cursorID != "" {
        cursor, err = manager.GetCursor(cursorID)
        if err != nil {
            // Cursor expired, create new one
            cursor, err = manager.CreateCursor(collection, nil, nil)
        }
    } else {
        cursor, err = manager.CreateCursor(collection, nil, nil)
    }

    if err != nil {
        return err
    }

    // Process with periodic checkpoints
    for cursor.HasNext() {
        doc, _ := cursor.Next()
        processDocument(doc)

        // Save checkpoint every 100 documents
        if cursor.Position()%100 == 0 {
            saveCheckpoint(cursor.ID())
        }
    }

    manager.CloseCursor(cursor.ID())
    clearCheckpoint()
    return nil
}
```

## HTTP API Integration

The cursor API is fully integrated with the HTTP server. See [HTTP API Documentation](http-api.md#cursor-api) for complete details.

### Cursor Creation Endpoint

```http
POST /api/cursors
Content-Type: application/json

{
  "collection": "users",
  "filter": {"age": {"$gte": 18}},
  "batchSize": 50,
  "timeout": "5m"
}

Response:
{
  "cursorId": "8e6d04d2a499dbe294893621d5d2c04c",
  "count": 1523,
  "batchSize": 50
}
```

### Fetch Batch Endpoint

```http
GET /api/cursors/8e6d04d2a499dbe294893621d5d2c04c/batch

Response:
{
  "documents": [...],
  "position": 50,
  "remaining": 1473,
  "hasMore": true
}
```

### Close Cursor Endpoint

```http
DELETE /api/cursors/8e6d04d2a499dbe294893621d5d2c04c

Response:
{
  "ok": true
}
```

## Performance Considerations

### Memory Usage

- Each cursor stores the full result set in memory
- Memory usage = `document_count × average_document_size`
- For very large result sets (>100K documents), consider using pagination with skip/limit instead

### Timeout Tuning

- Shorter timeouts reduce memory usage but may interrupt long-running operations
- Longer timeouts improve reliability but increase memory footprint
- Default 10 minutes is suitable for most interactive use cases

### Batch Size Optimization

Benchmark results for different batch sizes:

| Batch Size | Iteration Time | Memory Usage | Best For |
|------------|----------------|--------------|----------|
| 10         | Slower         | Low          | Real-time streaming |
| 100        | Good           | Medium       | General use |
| 500        | Better         | Higher       | Batch processing |
| 1000       | Best           | Highest      | Bulk operations |

### Cleanup Frequency

- Automatic cleanup runs every 60 seconds
- Manual cleanup with `CleanupTimedOutCursors()` for immediate effect
- Monitor with `ActiveCursors()` to detect leaks

## Limitations

1. **In-Memory Results**: Current implementation loads all query results into memory at cursor creation. Future versions may support lazy evaluation.

2. **No Cursor Persistence**: Cursors are lost on database restart. Server-side cursors exist only during the database lifetime.

3. **Read-Only**: Cursors provide read-only access. Modifications to the collection while a cursor is active are not reflected in the cursor's results.

4. **Single-Threaded Iteration**: Each cursor is designed for sequential access by a single goroutine. Use locks if accessing from multiple goroutines.

## See Also

- [Query Engine](query-engine.md) - Building queries
- [API Reference](api-reference.md) - Complete API documentation
- [HTTP API](http-api.md) - REST API endpoints
- [Performance Tuning](performance-tuning.md) - Optimization tips
