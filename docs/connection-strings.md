# Connection Strings

LauraDB supports MongoDB-style connection strings for configuring database connections. This provides a convenient and standardized way to specify connection parameters.

## Table of Contents

- [Overview](#overview)
- [URI Format](#uri-format)
- [Schemes](#schemes)
- [Components](#components)
- [Options](#options)
- [Examples](#examples)
- [Go API](#go-api)
- [Best Practices](#best-practices)

## Overview

Connection strings allow you to specify all connection parameters in a single URI, including:
- Host(s) and port(s)
- Authentication credentials
- Database name
- Connection options (timeouts, TLS, pooling, etc.)

The connection string format is compatible with MongoDB connection strings, making it easy for developers familiar with MongoDB to use LauraDB.

## URI Format

```
scheme://[username:password@]host[:port][,host2[:port2],...][/database][?options]
```

### Basic Structure

- **scheme**: `laura://` or `mongodb://`
- **username:password** (optional): Authentication credentials
- **host:port**: Server host and port (default port: 8080)
- **database** (optional): Database name
- **options** (optional): Query parameters for connection options

## Schemes

LauraDB supports two URI schemes:

### `laura://`
Native LauraDB scheme. Recommended for clarity.

```
laura://localhost:8080/mydb
```

### `mongodb://`
MongoDB-compatible scheme alias. Use this for easy migration from MongoDB.

```
mongodb://localhost:27017/mydb
```

Both schemes are functionally identical.

## Components

### Hosts

Specify one or more hosts for connection. Multiple hosts are comma-separated.

**Single host:**
```
laura://localhost:8080
```

**Multiple hosts (replica set):**
```
laura://host1:8080,host2:8081,host3:8082
```

**Default port:**
If no port is specified, the default port is 8080:
```
laura://localhost  # equivalent to laura://localhost:8080
```

### Authentication

Specify username and password using `username:password@` before the host.

**Basic authentication:**
```
laura://admin:password@localhost:8080
```

**URL-encoded password:**
Special characters in passwords must be URL-encoded:
```
laura://admin:P%40ssw0rd%21@localhost:8080  # password is "P@ssw0rd!"
```

Common URL encodings:
- `@` → `%40`
- `!` → `%21`
- `#` → `%23`
- `$` → `%24`
- `%` → `%25`
- `&` → `%26`
- `/` → `%2F`
- `:` → `%3A`
- `=` → `%3D`
- `?` → `%3F`

### Database

Specify the database name after the host(s):

```
laura://localhost:8080/production
```

If no database is specified, it's up to the client implementation to handle the default.

## Options

Options are specified as query parameters after a `?` character.

### Connection Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `timeout` / `timeoutMs` | int (ms) | 30000 | General timeout for operations |
| `maxConnections` / `maxPoolSize` | int | 100 | Maximum connections in pool |
| `minConnections` / `minPoolSize` | int | 1 | Minimum connections in pool |
| `maxIdleTime` / `maxIdleTimeMs` | int (ms) | 600000 | Max idle time for connections |
| `connectTimeout` / `connectTimeoutMs` | int (ms) | 10000 | Timeout for initial connection |
| `socketTimeout` / `socketTimeoutMs` | int (ms) | 30000 | Timeout for socket operations |

### TLS/SSL Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `tls` / `ssl` | bool | false | Enable TLS/SSL |
| `tlsInsecure` / `tlsInsecureSkipVerify` | bool | false | Skip certificate verification (insecure) |
| `tlsCertFile` / `tlsCertificateKeyFile` | string | - | Path to TLS certificate file |
| `tlsKeyFile` | string | - | Path to TLS key file |
| `tlsCAFile` | string | - | Path to CA certificate file |

### Read/Write Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `readPreference` | string | "primary" | Read preference (primary, secondary, etc.) |
| `writeConcern` / `w` | string | "majority" | Write concern level |

### Authentication Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `authSource` | string | - | Authentication database |
| `authMechanism` | string | - | Authentication mechanism |

### Replication Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `replicaSet` | string | - | Replica set name |
| `retryWrites` | bool | true | Retry failed write operations |
| `retryReads` | bool | true | Retry failed read operations |

### Application Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `appName` | string | - | Application name for logging |

### Boolean Values

Boolean options accept: `true`, `false`, `1`, `0`, `yes`, `no` (case-insensitive)

## Examples

### Example 1: Basic Local Connection

```
laura://localhost:8080/myapp
```

Connects to LauraDB on localhost port 8080, database "myapp".

### Example 2: With Authentication

```
laura://admin:SecretPassword@localhost:8080/production
```

Connects with username "admin" and password "SecretPassword".

### Example 3: With TLS

```
laura://localhost:8080/myapp?tls=true&tlsCertFile=/path/to/cert.pem&tlsKeyFile=/path/to/key.pem
```

Connects with TLS enabled and custom certificates.

### Example 4: Connection Pool Configuration

```
laura://localhost:8080/myapp?maxConnections=200&minConnections=10&timeout=5000
```

Configures connection pool with 10-200 connections and 5-second timeout.

### Example 5: Replica Set

```
laura://host1:8080,host2:8080,host3:8080/myapp?replicaSet=rs0&readPreference=secondaryPreferred
```

Connects to a replica set with secondary read preference.

### Example 6: Production Configuration

```
mongodb://app_user:C0mpl3x%40P%40ss%21@primary.example.com:27017,secondary1.example.com:27017,secondary2.example.com:27017/production?replicaSet=prod-rs&readPreference=secondaryPreferred&maxConnections=500&timeout=10000&tls=true&appName=MyProductionApp&retryWrites=true
```

Full production configuration with:
- Authentication with URL-encoded password
- Three-node replica set
- TLS enabled
- Connection pooling (500 max connections)
- 10-second timeout
- Secondary read preference
- Application name for logging
- Write retry enabled

## Go API

### Parsing Connection Strings

```go
import "github.com/mnohosten/laura-db/pkg/connstring"

// Parse a connection string
cs, err := connstring.Parse("laura://localhost:8080/mydb")
if err != nil {
    log.Fatalf("Parse error: %v", err)
}

// Access components
fmt.Println("Scheme:", cs.Scheme)
fmt.Println("Host:", cs.Hosts[0].Host)
fmt.Println("Port:", cs.Hosts[0].Port)
fmt.Println("Database:", cs.Database)
```

### Accessing Options

```go
cs, _ := connstring.Parse("laura://localhost:8080?timeout=5000&tls=true")

fmt.Println("Timeout:", cs.Options.Timeout)        // 5s
fmt.Println("TLS:", cs.Options.TLS)                // true
fmt.Println("Max Connections:", cs.Options.MaxConnections) // 100 (default)
```

### Helper Methods

```go
// Get first host
host := cs.GetFirstHost()
fmt.Printf("%s:%d\n", host.Host, host.Port)

// Check authentication
if cs.HasAuthentication() {
    fmt.Println("Username:", cs.Options.Username)
}

// Convert back to string
uri := cs.String()
fmt.Println("URI:", uri)
```

### ConnString Structure

```go
type ConnString struct {
    Scheme   string    // "laura" or "mongodb"
    Hosts    []Host    // List of host:port pairs
    Database string    // Database name (optional)
    Options  Options   // Connection options
}

type Host struct {
    Host string
    Port int
}

type Options struct {
    // Connection
    Timeout         time.Duration
    MaxConnections  int
    MinConnections  int
    MaxIdleTime     time.Duration
    ConnectTimeout  time.Duration
    SocketTimeout   time.Duration

    // TLS
    TLS         bool
    TLSInsecure bool
    TLSCertFile string
    TLSKeyFile  string
    TLSCAFile   string

    // Read/Write
    ReadPreference string
    WriteConcern   string

    // Auth
    Username string
    Password string
    AuthDB   string
    AuthMech string

    // Misc
    AppName     string
    ReplicaSet  string
    RetryWrites bool
    RetryReads  bool
}
```

## Best Practices

### Security

1. **URL-encode passwords**: Always URL-encode special characters in passwords
   ```go
   import "net/url"
   password := url.QueryEscape("P@ssw0rd!")
   uri := fmt.Sprintf("laura://user:%s@localhost:8080", password)
   ```

2. **Use environment variables**: Don't hardcode credentials
   ```go
   uri := fmt.Sprintf("laura://%s:%s@%s/%s",
       os.Getenv("DB_USER"),
       url.QueryEscape(os.Getenv("DB_PASSWORD")),
       os.Getenv("DB_HOST"),
       os.Getenv("DB_NAME"))
   ```

3. **Enable TLS in production**: Always use TLS for production deployments
   ```
   laura://localhost:8080/mydb?tls=true
   ```

4. **Validate certificates**: Don't use `tlsInsecure=true` in production
   ```
   laura://localhost:8080?tls=true&tlsCAFile=/path/to/ca.pem
   ```

### Performance

1. **Configure connection pools**: Set appropriate pool sizes
   ```
   laura://localhost:8080?maxConnections=100&minConnections=10
   ```

2. **Use appropriate timeouts**: Balance responsiveness and reliability
   ```
   laura://localhost:8080?timeout=30000&connectTimeout=5000
   ```

3. **Use replica sets**: Distribute read load
   ```
   laura://host1,host2,host3?replicaSet=rs0&readPreference=secondaryPreferred
   ```

### Reliability

1. **Enable retry**: Use retry for transient failures
   ```
   laura://localhost:8080?retryWrites=true&retryReads=true
   ```

2. **Use replica sets**: Automatic failover
   ```
   laura://primary,secondary1,secondary2?replicaSet=rs0
   ```

3. **Set appropriate write concern**: Balance durability and performance
   ```
   laura://localhost:8080?writeConcern=majority
   ```

### Debugging

1. **Use appName**: Identify clients in logs
   ```
   laura://localhost:8080?appName=MyApp
   ```

2. **Start with verbose logging**: During development
   ```go
   cs, err := connstring.Parse(uri)
   if err != nil {
       log.Fatalf("Connection string error: %v", err)
   }
   log.Printf("Connecting to: %s:%d", cs.Hosts[0].Host, cs.Hosts[0].Port)
   ```

## Error Handling

The parser returns specific errors for common issues:

```go
cs, err := connstring.Parse(uri)
if err != nil {
    switch {
    case errors.Is(err, connstring.ErrInvalidScheme):
        log.Fatal("Invalid scheme: must be 'laura://' or 'mongodb://'")
    case errors.Is(err, connstring.ErrNoHosts):
        log.Fatal("No hosts specified")
    case errors.Is(err, connstring.ErrInvalidConnString):
        log.Fatal("Invalid connection string format")
    default:
        log.Fatalf("Parse error: %v", err)
    }
}
```

## See Also

- [HTTP Server Documentation](http-api.md)
- [TLS/SSL Configuration](tls-ssl.md)
- [Replication Guide](replication.md)
- [Client Libraries](../clients/README.md)
