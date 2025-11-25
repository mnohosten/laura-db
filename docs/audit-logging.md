# Audit Logging

LauraDB provides comprehensive audit logging capabilities to track database operations for security, compliance, and debugging purposes.

## Overview

Audit logging records all database operations including:
- Document operations (Insert, Update, Delete, Find)
- Collection management (Create, Drop)
- Index operations (Create, Drop)
- Query and aggregation operations
- Text search operations

Each audit event includes:
- Timestamp
- Operation type
- Collection and database name
- User (if authentication is enabled)
- Success/failure status
- Operation duration
- Error messages (if any)
- Query filters and update specifications (optional)
- Document counts

## Features

### Multiple Output Formats
- **JSON**: Structured logging for easy parsing and analysis
- **Text**: Human-readable format for console output

### Flexible Configuration
- Enable/disable logging at runtime
- Filter by operation type
- Filter by severity level (Info, Warning, Error)
- Control query data inclusion
- Limit field sizes to prevent log bloat

### Performance Optimized
- Minimal overhead on database operations
- Thread-safe concurrent logging
- Efficient field truncation for large queries

### Output Destinations
- Standard output (stdout)
- File-based logging with append mode
- Custom writers (any `io.Writer`)

## Configuration

### Basic Configuration

```go
import "github.com/mnohosten/laura-db/pkg/audit"

// Create default configuration (stdout, JSON format)
config := audit.DefaultConfig()

// Or customize configuration
config := &audit.Config{
    Enabled:          true,
    OutputWriter:     os.Stdout,
    Format:           "json",  // or "text"
    MinSeverity:      audit.SeverityInfo,
    IncludeQueryData: true,
    MaxFieldSize:     1000,  // 1KB limit
    Operations:       nil,   // Log all operations
}
```

### Configuration Options

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `Enabled` | bool | Enable/disable audit logging | true |
| `OutputWriter` | io.Writer | Destination for log output | stdout |
| `Format` | string | "json" or "text" | "json" |
| `MinSeverity` | Severity | Minimum severity to log | SeverityInfo |
| `IncludeQueryData` | bool | Include full query/update data | true |
| `MaxFieldSize` | int | Max size for query fields (0=unlimited) | 1000 |
| `Operations` | []OperationType | Operations to audit (empty=all) | nil |

### Severity Levels

- `SeverityInfo`: Successful operations
- `SeverityWarning`: Warnings (reserved for future use)
- `SeverityError`: Failed operations

## Usage

### Using Audit Logging with Database

```go
import (
    "github.com/mnohosten/laura-db/pkg/audit"
    "github.com/mnohosten/laura-db/pkg/database"
)

// Create audit configuration
auditConfig := &audit.Config{
    Enabled:          true,
    OutputWriter:     os.Stdout,
    Format:           "json",
    IncludeQueryData: true,
}

// Create database configuration with audit logging
dbConfig := &database.Config{
    DataDir:        "./data",
    BufferPoolSize: 1000,
    AuditConfig:    auditConfig,
}

// Open database (audit logging is now active)
db, err := database.Open(dbConfig)
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// All operations are now automatically audited
coll := db.Collection("users")
id, err := coll.InsertOne(map[string]interface{}{
    "name": "John Doe",
    "age":  int64(30),
})
// Audit log entry created automatically
```

### File-Based Logging

```go
// Create file-based audit logger
logger, err := audit.NewFileAuditLogger("./logs/audit.log", config)
if err != nil {
    log.Fatal(err)
}
defer logger.Close()

// Use with database config
dbConfig := &database.Config{
    DataDir:     "./data",
    AuditConfig: config,
}
```

### Filtering Operations

```go
// Only audit write operations
config := &audit.Config{
    Enabled: true,
    Operations: []audit.OperationType{
        audit.OperationInsert,
        audit.OperationUpdate,
        audit.OperationDelete,
    },
}
```

### Filtering by Severity

```go
// Only log errors
config := &audit.Config{
    Enabled:     true,
    MinSeverity: audit.SeverityError,
}
```

### Runtime Control

```go
// Disable logging at runtime
if db.auditLogger != nil {
    db.auditLogger.SetEnabled(false)
}

// Re-enable logging
if db.auditLogger != nil {
    db.auditLogger.SetEnabled(true)
}

// Check if enabled
enabled := db.auditLogger.IsEnabled()
```

## Audit Event Structure

### JSON Format

```json
{
  "timestamp": "2025-11-24T10:30:45Z",
  "operation": "insert",
  "collection": "users",
  "database": "default",
  "user": "admin",
  "remoteAddr": "192.168.1.100",
  "success": true,
  "duration": 1500000,
  "severity": "info",
  "documentCount": 1,
  "details": {
    "field1": "value1"
  }
}
```

### Text Format

```
[2025-11-24T10:30:45Z] [info] [SUCCESS] insert operation on default.users by user admin (took 1.5ms) - 1 documents
```

## Operation Types

| Operation Type | Description |
|---------------|-------------|
| `OperationInsert` | Insert single document |
| `OperationInsertMany` | Insert multiple documents |
| `OperationUpdate` | Update single document |
| `OperationUpdateMany` | Update multiple documents |
| `OperationDelete` | Delete single document |
| `OperationDeleteMany` | Delete multiple documents |
| `OperationFind` | Find documents |
| `OperationFindOne` | Find single document |
| `OperationAggregate` | Aggregation pipeline |
| `OperationCreateIndex` | Create index |
| `OperationDropIndex` | Drop index |
| `OperationCreateCollection` | Create collection |
| `OperationDropCollection` | Drop collection |
| `OperationTextSearch` | Text search |
| `OperationCount` | Count documents |

## Use Cases

### Compliance and Regulatory Requirements

Track all data access and modifications for:
- GDPR compliance (data access audit trails)
- HIPAA compliance (healthcare data access)
- SOX compliance (financial data tracking)
- PCI-DSS compliance (payment card data)

### Security Monitoring

- Detect unauthorized access attempts
- Track administrative operations
- Monitor suspicious query patterns
- Identify data exfiltration attempts

### Debugging and Troubleshooting

- Trace operation failures
- Analyze slow query patterns
- Debug application issues
- Performance monitoring

### Analytics

- Operation frequency analysis
- User activity patterns
- Query performance metrics
- Resource usage tracking

## Performance Considerations

### Overhead

Audit logging adds minimal overhead:
- JSON serialization: ~10-50µs per operation
- File I/O: ~100-500µs per operation (buffered)
- Memory: ~1KB per cached event

### Optimization Tips

1. **Disable query data for high-volume operations**
   ```go
   config.IncludeQueryData = false
   ```

2. **Use field size limits**
   ```go
   config.MaxFieldSize = 500  // Truncate large queries
   ```

3. **Filter by operation type**
   ```go
   config.Operations = []audit.OperationType{
       audit.OperationInsert,
       audit.OperationUpdate,
       audit.OperationDelete,
   }
   ```

4. **Use buffered file writers**
   ```go
   file, _ := os.OpenFile("audit.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
   buffered := bufio.NewWriter(file)
   config.OutputWriter = buffered
   ```

5. **Filter by severity for production**
   ```go
   config.MinSeverity = audit.SeverityError  // Only log errors
   ```

## Integration with Authentication

When using LauraDB's authentication system, audit logs automatically include user information:

```go
// User information is automatically captured from context
// when authentication is enabled
{
  "timestamp": "2025-11-24T10:30:45Z",
  "operation": "delete",
  "collection": "sensitive_data",
  "user": "john.doe@example.com",
  "remoteAddr": "192.168.1.50",
  "success": true,
  "deletedCount": 1
}
```

## Log Analysis

### Parsing JSON Logs

```bash
# Count operations by type
cat audit.log | jq -r '.operation' | sort | uniq -c

# Find failed operations
cat audit.log | jq 'select(.success == false)'

# Calculate average operation duration
cat audit.log | jq '.duration' | awk '{sum+=$1; count++} END {print sum/count}'

# Track operations by user
cat audit.log | jq -r '.user' | sort | uniq -c
```

### Log Rotation

```bash
# Use logrotate for automatic log rotation
# /etc/logrotate.d/laura-db
/var/log/laura-db/audit.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0644 laura-db laura-db
}
```

## Best Practices

1. **Enable audit logging in production**
   - Always enable for compliance and security monitoring
   - Use appropriate severity filtering

2. **Secure audit logs**
   - Store logs in a secure location
   - Restrict file permissions (0600 or 0644)
   - Consider write-once storage for tamper-proofing

3. **Regular log review**
   - Monitor for suspicious patterns
   - Analyze failed operations
   - Track unauthorized access attempts

4. **Log retention policies**
   - Define retention periods based on compliance requirements
   - Implement automated archival
   - Use log rotation to manage disk space

5. **Performance monitoring**
   - Monitor audit log size and growth rate
   - Adjust field size limits as needed
   - Consider async logging for high-throughput systems

6. **Testing**
   - Test audit logging in development
   - Verify log formats and content
   - Ensure proper error handling

## Troubleshooting

### Logs not appearing

```go
// Verify audit logging is enabled
if db.auditLogger != nil && db.auditLogger.IsEnabled() {
    fmt.Println("Audit logging is active")
}

// Check configuration
config := audit.DefaultConfig()
logger, err := audit.NewAuditLogger(config)
if err != nil {
    log.Fatal(err)
}
```

### Large log files

```go
// Reduce field sizes
config.MaxFieldSize = 200

// Disable query data
config.IncludeQueryData = false

// Filter operations
config.Operations = []audit.OperationType{
    audit.OperationUpdate,
    audit.OperationDelete,
}
```

### Performance impact

```go
// Only log errors in production
config.MinSeverity = audit.SeverityError

// Use async logging (custom implementation)
asyncWriter := NewAsyncWriter(file)
config.OutputWriter = asyncWriter
```

## Future Enhancements

Potential future improvements:
- Async logging for zero-blocking performance
- Remote logging (syslog, Elasticsearch, etc.)
- Log streaming via webhooks
- Built-in log aggregation
- Real-time alerting
- Automatic log compression
- Structured search queries for logs

## Example Output

### Successful Insert
```json
{
  "timestamp": "2025-11-24T10:15:30.123Z",
  "operation": "insert",
  "collection": "users",
  "database": "default",
  "success": true,
  "duration": 1234567,
  "severity": "info",
  "documentCount": 1
}
```

### Failed Update
```json
{
  "timestamp": "2025-11-24T10:16:45.456Z",
  "operation": "update",
  "collection": "users",
  "database": "default",
  "success": false,
  "errorMessage": "document not found",
  "duration": 987654,
  "severity": "error",
  "queryFilter": {"_id": "507f1f77bcf86cd799439011"},
  "updateSpec": {"$set": {"status": "active"}}
}
```

### Index Creation
```json
{
  "timestamp": "2025-11-24T10:17:00.789Z",
  "operation": "createIndex",
  "collection": "products",
  "database": "default",
  "user": "admin",
  "success": true,
  "duration": 5432100,
  "severity": "info",
  "indexName": "price_1"
}
```

## Conclusion

Audit logging in LauraDB provides comprehensive tracking of database operations with minimal performance overhead. It's essential for compliance, security monitoring, and debugging, making it a critical component of production deployments.
