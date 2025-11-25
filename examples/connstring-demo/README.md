# Connection String Parsing Demo

This example demonstrates LauraDB's MongoDB-compatible connection string parser.

## Features Demonstrated

1. **Basic Connection String** - Simple local connection
2. **Authentication** - Username/password with URL encoding
3. **Multiple Hosts** - Replica set configuration
4. **Connection Options** - Timeout, pooling, TLS settings
5. **Complex Real-World** - Production-grade configuration

## Building

```bash
# From repository root
make build

# Or build just this example
cd examples/connstring-demo
go build -o ../../bin/connstring-demo main.go
```

## Running

```bash
# From repository root
./bin/connstring-demo
```

## Output

The demo parses various connection strings and displays:
- Scheme (laura:// or mongodb://)
- Host(s) and port(s)
- Database name
- Authentication details
- Connection options (timeout, pooling, TLS, etc.)
- Replica set configuration

## Example Connection Strings

### Basic
```
laura://localhost:8080/mydb
```

### With Authentication
```
mongodb://admin:P%40ssw0rd@localhost:27017/production
```

### Replica Set
```
laura://host1:8080,host2:8081,host3:8082/mydb?replicaSet=rs0
```

### With Options
```
laura://localhost:8080/mydb?timeout=10000&maxConnections=200&tls=true&appName=MyApp
```

### Production Configuration
```
mongodb://admin:SecretPass123@primary.example.com:27017,secondary1.example.com:27017,secondary2.example.com:27017/production?replicaSet=rs0&readPreference=secondaryPreferred&maxConnections=500&timeout=30000&tls=true&appName=ProductionApp&retryWrites=true
```

## API Usage

```go
import "github.com/mnohosten/laura-db/pkg/connstring"

// Parse a connection string
cs, err := connstring.Parse("laura://localhost:8080/mydb")
if err != nil {
    log.Fatal(err)
}

// Access components
fmt.Println("Host:", cs.Hosts[0].Host)
fmt.Println("Port:", cs.Hosts[0].Port)
fmt.Println("Database:", cs.Database)

// Access options
fmt.Println("Timeout:", cs.Options.Timeout)
fmt.Println("TLS:", cs.Options.TLS)
```

## See Also

- [Connection Strings Documentation](../../docs/connection-strings.md)
- [HTTP Server Documentation](../../docs/http-api.md)
- [TLS/SSL Documentation](../../docs/tls-ssl.md)
