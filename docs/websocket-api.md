# WebSocket API

LauraDB provides a WebSocket API for receiving real-time change notifications from the database. This allows applications to build reactive features that immediately respond to data changes.

## Table of Contents

- [Overview](#overview)
- [Connection](#connection)
- [Request Format](#request-format)
- [Response Format](#response-format)
- [Message Types](#message-types)
- [Change Events](#change-events)
- [Filtering](#filtering)
- [Pipeline Transformations](#pipeline-transformations)
- [Resume Tokens](#resume-tokens)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)
- [Examples](#examples)

## Overview

The WebSocket API builds on LauraDB's [Change Streams](./change-streams.md) functionality to provide real-time event delivery over WebSocket connections. Key features include:

- **Real-time updates**: Receive notifications as soon as changes occur
- **Low latency**: Sub-second notification delivery
- **Bi-directional**: Both client and server can send messages
- **Efficient**: Long-lived connections reduce overhead
- **Scalable**: Support for many concurrent connections

## Connection

### Endpoint

```
ws://host:port/_ws/watch
```

For TLS-enabled servers:

```
wss://host:port/_ws/watch
```

### Connection Flow

1. **Establish WebSocket connection**: Client connects to the WebSocket endpoint
2. **Send subscription request**: Client sends initial JSON request specifying what to watch
3. **Receive acknowledgment**: Server confirms subscription with `connected` message
4. **Receive events**: Server streams change events as they occur
5. **Heartbeats**: Server sends periodic keepalive messages
6. **Close**: Either party can close the connection

### Example Connection (JavaScript)

```javascript
const ws = new WebSocket('ws://localhost:8080/_ws/watch');

ws.onopen = () => {
  // Send subscription request
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
  console.log('Received:', response);
};
```

### Example Connection (Go)

```go
import "github.com/gorilla/websocket"

u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/_ws/watch"}
conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

// Send subscription request
req := map[string]interface{}{
    "database":   "mydb",
    "collection": "users",
}
conn.WriteJSON(req)

// Read messages
var response map[string]interface{}
for {
    err := conn.ReadJSON(&response)
    if err != nil {
        break
    }
    // Handle response
}
```

## Request Format

After establishing the WebSocket connection, send a JSON request to specify what changes to watch:

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

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `database` | string | Yes | Database name to watch (empty string = all databases) |
| `collection` | string | No | Collection name to watch (empty string = all collections) |
| `filter` | object | No | Query filter for events (see [Filtering](#filtering)) |
| `pipeline` | array | No | Aggregation pipeline for transforming events |
| `resumeToken` | object | No | Resume token to continue from previous position |

## Response Format

All responses from the server are JSON objects with this structure:

```json
{
  "type": "event",
  "event": { /* change event */ },
  "error": null,
  "message": null
}
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Message type: `connected`, `event`, `heartbeat`, or `error` |
| `event` | object | Change event details (present when `type` is `event`) |
| `error` | string | Error message (present when `type` is `error`) |
| `message` | string | Status message (present for `connected` and `heartbeat`) |

## Message Types

### Connected

Sent immediately after subscription is established:

```json
{
  "type": "connected",
  "message": "Change stream connected successfully"
}
```

### Event

Contains a change event:

```json
{
  "type": "event",
  "event": {
    "_id": {
      "opId": 1
    },
    "operationType": "insert",
    "clusterTime": "2025-01-24T10:30:00Z",
    "db": "mydb",
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
}
```

### Heartbeat

Sent every 30 seconds to keep the connection alive:

```json
{
  "type": "heartbeat",
  "message": "keepalive"
}
```

### Error

Indicates an error occurred (connection will be closed):

```json
{
  "type": "error",
  "error": "Failed to process change stream: ..."
}
```

## Change Events

Change events follow the same structure as [Change Streams](./change-streams.md#change-events).

### Insert Event

```json
{
  "_id": {
    "opId": 1
  },
  "operationType": "insert",
  "clusterTime": "2025-01-24T10:30:00Z",
  "db": "mydb",
  "coll": "users",
  "documentKey": {
    "_id": "user1"
  },
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
  "_id": {
    "opId": 2
  },
  "operationType": "update",
  "clusterTime": "2025-01-24T10:30:01Z",
  "db": "mydb",
  "coll": "users",
  "documentKey": {
    "_id": "user1"
  },
  "updateDescription": {
    "updatedFields": {
      "age": 31
    },
    "removedFields": ["email"]
  }
}
```

### Delete Event

```json
{
  "_id": {
    "opId": 3
  },
  "operationType": "delete",
  "clusterTime": "2025-01-24T10:30:02Z",
  "db": "mydb",
  "coll": "users",
  "documentKey": {
    "_id": "user1"
  }
}
```

## Filtering

Filter events using query operators:

### Filter by Operation Type

Watch only insert operations:

```json
{
  "database": "mydb",
  "collection": "users",
  "filter": {
    "operationType": "insert"
  }
}
```

### Filter by Multiple Operation Types

Watch inserts and updates:

```json
{
  "database": "mydb",
  "collection": "users",
  "filter": {
    "operationType": {
      "$in": ["insert", "update"]
    }
  }
}
```

### Filter by Document Fields

Watch changes to specific documents:

```json
{
  "database": "mydb",
  "collection": "users",
  "filter": {
    "documentKey._id": "user1"
  }
}
```

### Complex Filters

Combine multiple conditions:

```json
{
  "database": "mydb",
  "collection": "users",
  "filter": {
    "$and": [
      {
        "operationType": "update"
      },
      {
        "fullDocument.age": {
          "$gte": 18
        }
      }
    ]
  }
}
```

## Pipeline Transformations

Apply aggregation-style pipelines to filter and transform events:

### $match Stage

Filter events using aggregation pipeline:

```json
{
  "database": "mydb",
  "collection": "users",
  "pipeline": [
    {
      "$match": {
        "operationType": {
          "$in": ["insert", "update"]
        }
      }
    }
  ]
}
```

### Multiple Stages

Chain multiple pipeline stages:

```json
{
  "database": "mydb",
  "collection": "users",
  "pipeline": [
    {
      "$match": {
        "operationType": "insert"
      }
    },
    {
      "$match": {
        "fullDocument.country": "USA"
      }
    }
  ]
}
```

## Resume Tokens

Resume tokens allow you to continue watching from a specific point after reconnecting.

### Saving Resume Tokens

Extract the resume token from each event:

```javascript
ws.onmessage = (event) => {
  const response = JSON.parse(event.data);

  if (response.type === 'event') {
    // Save the resume token
    const resumeToken = response.event._id;
    localStorage.setItem('resumeToken', JSON.stringify(resumeToken));
  }
};
```

### Resuming from Token

Include the resume token in the subscription request:

```json
{
  "database": "mydb",
  "collection": "users",
  "resumeToken": {
    "opId": 12345
  }
}
```

### Example: Auto-Resume on Reconnect

```javascript
function connect() {
  const ws = new WebSocket('ws://localhost:8080/_ws/watch');

  ws.onopen = () => {
    // Load saved resume token
    const savedToken = localStorage.getItem('resumeToken');
    const resumeToken = savedToken ? JSON.parse(savedToken) : null;

    ws.send(JSON.stringify({
      database: 'mydb',
      collection: 'users',
      resumeToken: resumeToken
    }));
  };

  ws.onclose = () => {
    // Reconnect after 5 seconds
    setTimeout(connect, 5000);
  };

  ws.onmessage = (event) => {
    const response = JSON.parse(event.data);

    if (response.type === 'event') {
      // Save resume token for each event
      localStorage.setItem('resumeToken', JSON.stringify(response.event._id));

      // Process event
      handleEvent(response.event);
    }
  };
}

connect();
```

## Error Handling

### Connection Errors

Handle connection failures and retries:

```javascript
let reconnectAttempts = 0;
const maxReconnectAttempts = 5;

function connect() {
  const ws = new WebSocket('ws://localhost:8080/_ws/watch');

  ws.onerror = (error) => {
    console.error('WebSocket error:', error);
  };

  ws.onclose = (event) => {
    if (event.code === 1000) {
      // Normal closure
      console.log('Connection closed normally');
    } else if (reconnectAttempts < maxReconnectAttempts) {
      // Unexpected closure, reconnect
      reconnectAttempts++;
      const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
      console.log(`Reconnecting in ${delay}ms (attempt ${reconnectAttempts})`);
      setTimeout(connect, delay);
    } else {
      console.error('Max reconnection attempts reached');
    }
  };

  ws.onopen = () => {
    // Reset reconnection counter on successful connect
    reconnectAttempts = 0;
  };
}
```

### Stream Errors

Handle errors from the change stream:

```javascript
ws.onmessage = (event) => {
  const response = JSON.parse(event.data);

  if (response.type === 'error') {
    console.error('Stream error:', response.error);

    // Connection will be closed by server
    // Implement reconnection logic
  }
};
```

### Timeout Detection

Detect connection issues using heartbeats:

```javascript
let lastHeartbeat = Date.now();
const heartbeatTimeout = 60000; // 60 seconds

ws.onmessage = (event) => {
  const response = JSON.parse(event.data);

  if (response.type === 'heartbeat') {
    lastHeartbeat = Date.now();
  }
};

// Check for stale connection every 30 seconds
setInterval(() => {
  const timeSinceHeartbeat = Date.now() - lastHeartbeat;

  if (timeSinceHeartbeat > heartbeatTimeout) {
    console.warn('No heartbeat received, reconnecting...');
    ws.close();
  }
}, 30000);
```

## Best Practices

### 1. Always Handle All Message Types

```javascript
ws.onmessage = (event) => {
  const response = JSON.parse(event.data);

  switch (response.type) {
    case 'connected':
      console.log('Connected successfully');
      break;
    case 'event':
      handleEvent(response.event);
      break;
    case 'heartbeat':
      // Update last activity time
      break;
    case 'error':
      console.error('Error:', response.error);
      break;
    default:
      console.warn('Unknown message type:', response.type);
  }
};
```

### 2. Implement Reconnection Logic

Always implement automatic reconnection with exponential backoff:

```javascript
function connectWithRetry() {
  let retryDelay = 1000;

  function tryConnect() {
    const ws = new WebSocket('ws://localhost:8080/_ws/watch');

    ws.onclose = () => {
      console.log(`Reconnecting in ${retryDelay}ms...`);
      setTimeout(tryConnect, retryDelay);
      retryDelay = Math.min(retryDelay * 2, 30000);
    };

    ws.onopen = () => {
      retryDelay = 1000; // Reset on successful connect
    };
  }

  tryConnect();
}
```

### 3. Save Resume Tokens Periodically

Persist resume tokens to recover from crashes:

```javascript
let lastSavedToken = null;

ws.onmessage = (event) => {
  const response = JSON.parse(event.data);

  if (response.type === 'event') {
    const token = response.event._id;

    // Save token every 10 events to reduce I/O
    if (!lastSavedToken || token.opId % 10 === 0) {
      saveTokenToDisk(token);
      lastSavedToken = token;
    }
  }
};
```

### 4. Use Filters Early

Apply filters at the server level rather than in client code:

```javascript
// Good: Server-side filtering
ws.send(JSON.stringify({
  database: 'mydb',
  collection: 'users',
  filter: {
    operationType: 'insert',
    'fullDocument.age': { $gte: 18 }
  }
}));

// Bad: Client-side filtering
ws.onmessage = (event) => {
  const response = JSON.parse(event.data);
  if (response.event.fullDocument.age >= 18) {
    // Process event
  }
};
```

### 5. Monitor Connection Health

Track connection metrics:

```javascript
const metrics = {
  eventsReceived: 0,
  lastEventTime: Date.now(),
  reconnections: 0,
  errors: 0
};

ws.onmessage = (event) => {
  const response = JSON.parse(event.data);

  if (response.type === 'event') {
    metrics.eventsReceived++;
    metrics.lastEventTime = Date.now();
  } else if (response.type === 'error') {
    metrics.errors++;
  }
};

ws.onclose = () => {
  metrics.reconnections++;
};

// Log metrics every minute
setInterval(() => {
  console.log('Metrics:', metrics);
}, 60000);
```

### 6. Clean Up Resources

Always clean up when done:

```javascript
window.addEventListener('beforeunload', () => {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.close(1000, 'Page unloading');
  }
});
```

## Examples

### Example 1: Real-time Dashboard

```javascript
const ws = new WebSocket('ws://localhost:8080/_ws/watch');

ws.onopen = () => {
  ws.send(JSON.stringify({
    database: 'analytics',
    collection: 'events',
    filter: {
      operationType: 'insert',
      'fullDocument.type': 'pageview'
    }
  }));
};

ws.onmessage = (event) => {
  const response = JSON.parse(event.data);

  if (response.type === 'event') {
    // Update dashboard with new pageview
    updateDashboard(response.event.fullDocument);
  }
};

function updateDashboard(pageview) {
  // Update metrics
  incrementCounter('pageviews');
  updateChart(pageview.page);
}
```

### Example 2: Cache Invalidation

```javascript
const cache = new Map();

const ws = new WebSocket('ws://localhost:8080/_ws/watch');

ws.onopen = () => {
  ws.send(JSON.stringify({
    database: 'mydb',
    collection: 'users'
  }));
};

ws.onmessage = (event) => {
  const response = JSON.parse(event.data);

  if (response.type === 'event') {
    const docId = response.event.documentKey._id;

    switch (response.event.operationType) {
      case 'insert':
      case 'update':
        // Invalidate cache entry
        cache.delete(docId);
        console.log(`Cache invalidated for ${docId}`);
        break;
      case 'delete':
        cache.delete(docId);
        console.log(`Cache entry removed for ${docId}`);
        break;
    }
  }
};
```

### Example 3: Data Synchronization

```javascript
const ws = new WebSocket('ws://localhost:8080/_ws/watch');

ws.onopen = () => {
  ws.send(JSON.stringify({
    database: 'mydb',
    collection: 'documents'
  }));
};

ws.onmessage = (event) => {
  const response = JSON.parse(event.data);

  if (response.type === 'event') {
    const event = response.event;

    switch (event.operationType) {
      case 'insert':
        // Sync new document to local storage
        syncToLocal('insert', event.fullDocument);
        break;
      case 'update':
        // Sync updates to local storage
        syncToLocal('update', event.documentKey, event.updateDescription);
        break;
      case 'delete':
        // Remove from local storage
        syncToLocal('delete', event.documentKey);
        break;
    }
  }
};

function syncToLocal(operation, ...args) {
  // Implement local synchronization logic
  console.log(`Syncing ${operation} to local storage`);
}
```

## Performance Considerations

### Scalability

- Each WebSocket connection consumes server resources
- The server can handle hundreds of concurrent connections
- Use filters to reduce unnecessary event delivery
- Consider connection pooling for high-scale applications

### Latency

- Events are delivered with ~1-2 second latency
- Latency depends on the change stream polling interval (default: 1s)
- Network latency adds to total delivery time

### Throughput

- The server can handle thousands of events per second
- Throughput is limited by network bandwidth and client processing speed
- Use batching for high-throughput scenarios

## Security Considerations

### Authentication

Currently, WebSocket connections do not require authentication. In production environments:

- Implement authentication using connection headers or initial handshake
- Use TLS/SSL (wss://) for encrypted connections
- Validate client permissions before subscribing to changes

### Rate Limiting

Consider implementing rate limiting to prevent abuse:

- Limit number of connections per IP
- Limit message frequency per connection
- Implement backpressure for slow clients

## Limitations

1. **No Authentication**: WebSocket connections currently don't support authentication
2. **Single Database**: Each connection watches changes from a single database instance
3. **No Backpressure**: Slow clients may miss events if buffer fills up
4. **Polling-Based**: Events are delivered via polling (not push-based)

## Learn More

- [Change Streams Documentation](./change-streams.md)
- [HTTP API Documentation](./http-api.md)
- [WebSocket Example](../examples/websocket-demo/)
- [Query Engine](./query-engine.md)

## Troubleshooting

### Connection Refused

- Verify server is running
- Check firewall rules
- Confirm correct host and port

### No Events Received

- Verify database and collection names are correct
- Check if filter is too restrictive
- Generate test events to confirm operation

### High Latency

- Check network connection
- Verify server load
- Consider reducing polling interval (requires server configuration)

### Disconnections

- Implement reconnection logic with exponential backoff
- Check server logs for errors
- Monitor network stability
