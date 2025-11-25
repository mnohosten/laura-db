# LauraDB Cluster gRPC Server

This package implements the gRPC server infrastructure for LauraDB's distributed cluster communication.

## Overview

The server provides a complete gRPC implementation for all cluster services defined in the protocol buffer schemas:

- **ClusterService**: Node registration, discovery, and health monitoring
- **ReplicationService**: Oplog streaming and leader election
- **RouterService**: Query routing and shard management
- **TransactionService**: Two-phase commit for distributed transactions

## Features

### Core Server
- ✅ gRPC server with configurable listen address
- ✅ Concurrent connection handling with limits
- ✅ Graceful shutdown with timeout
- ✅ Health check support
- ✅ TLS/mTLS support (configurable)

### Connection Management
- ✅ Keep-alive parameters for connection health
- ✅ Keep-alive enforcement policy
- ✅ Maximum concurrent streams limit
- ✅ Connection timeout configuration

### Service Implementations

#### ClusterService (Fully Functional)
- ✅ Node registration with duplicate detection
- ✅ Node deregistration with cleanup
- ✅ Heartbeat handling with topology updates
- ✅ Topology versioning and change detection
- ✅ Thread-safe node management

#### ReplicationService (Stub Implementation)
- 🚧 Oplog streaming (placeholder)
- 🚧 Initial sync (placeholder)
- 🚧 Vote handling (placeholder)
- 🚧 Write acknowledgments (placeholder)

#### RouterService (Stub Implementation)
- 🚧 Query routing (placeholder)
- 🚧 Insert/Update/Delete routing (placeholder)
- 🚧 Aggregation routing (placeholder)
- 🚧 Shard management (placeholder)

#### TransactionService (Stub Implementation)
- 🚧 Two-phase commit prepare (placeholder)
- 🚧 Commit/Abort handling (placeholder)
- 🚧 Transaction state tracking (basic)

## Usage

### Basic Server Setup

```go
package main

import (
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/mnohosten/laura-db/pkg/cluster/server"
)

func main() {
    // Create server with default config
    config := server.DefaultConfig()
    config.Host = "0.0.0.0"
    config.Port = 27018

    srv, err := server.NewServer(config)
    if err != nil {
        log.Fatalf("Failed to create server: %v", err)
    }

    // Start server
    if err := srv.Start(); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }

    fmt.Printf("Server listening on %s\n", srv.Addr())

    // Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    fmt.Println("Shutting down server...")
    if err := srv.Stop(); err != nil {
        log.Printf("Error stopping server: %v", err)
    }
}
```

### With TLS Configuration

```go
config := server.DefaultConfig()
config.TLSEnabled = true
config.CertFile = "/path/to/cert.pem"
config.KeyFile = "/path/to/key.pem"

srv, err := server.NewServer(config)
```

### Custom Configuration

```go
config := &server.Config{
    Host:               "127.0.0.1",
    Port:               27018,
    TLSEnabled:         false,
    MaxConnections:     5000,
    MaxConcurrentRPCs:  200,
    ConnectionTimeout:  15 * time.Second,
    RequestTimeout:     60 * time.Second,
    KeepAliveInterval:  60 * time.Second,
    KeepAliveTimeout:   20 * time.Second,
    EnableReflection:   true,
}

srv, err := server.NewServer(config)
```

## Client Connection

Clients can connect to the server using standard gRPC clients:

```go
package main

import (
    "context"
    "log"
    "time"

    pb "github.com/mnohosten/laura-db/pkg/cluster/proto"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

func main() {
    // Connect to server
    conn, err := grpc.Dial("localhost:27018",
        grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()

    // Create cluster service client
    client := pb.NewClusterServiceClient(conn)

    // Register a node
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    resp, err := client.RegisterNode(ctx, &pb.NodeRegistrationRequest{
        Node: &pb.Node{
            Id:       "node-1",
            Host:     "192.168.1.10",
            Port:     27017,
            Role:     pb.NodeRole_SECONDARY,
            Priority: 5,
        },
    })

    if err != nil {
        log.Fatalf("RegisterNode failed: %v", err)
    }

    log.Printf("Node registered: %v", resp.Success)
}
```

## Configuration Options

### Network Configuration
- `Host`: Listen address (default: "0.0.0.0")
- `Port`: Listen port (default: 27018)

### TLS Configuration
- `TLSEnabled`: Enable TLS (default: false)
- `TLSConfig`: Custom TLS configuration
- `CertFile`: Path to certificate file
- `KeyFile`: Path to private key file

### Connection Settings
- `MaxConnections`: Maximum concurrent connections (default: 1000)
- `MaxConcurrentRPCs`: Maximum concurrent RPCs (default: 100)
- `ConnectionTimeout`: Connection timeout (default: 10s)
- `RequestTimeout`: Request timeout (default: 30s)
- `KeepAliveInterval`: Keep-alive ping interval (default: 30s)
- `KeepAliveTimeout`: Keep-alive timeout (default: 10s)

### Server Options
- `EnableReflection`: Enable gRPC reflection (default: true)

## Testing

Run the tests:

```bash
go test ./pkg/cluster/server -v
```

Run tests with race detection:

```bash
go test ./pkg/cluster/server -race
```

## Architecture

### Server Lifecycle

```
NewServer() -> Start() -> [Running] -> Stop() -> [Stopped]
                           |
                           v
                    WaitForShutdown()
```

### Service Registration

The server automatically registers all four services on creation:

1. ClusterService - Node management and topology
2. ReplicationService - Oplog and leader election
3. RouterService - Query routing and sharding
4. TransactionService - Distributed transactions

### Thread Safety

- Server state (started/stopped) is protected by RWMutex
- ClusterService node registry is thread-safe
- TransactionService transaction map is thread-safe
- All gRPC handlers can be called concurrently

## Development Status

### Completed (✅)
- gRPC server infrastructure
- Server lifecycle management
- Connection pooling and limits
- TLS/mTLS support
- ClusterService full implementation
- Health check support
- Comprehensive test suite

### In Progress (🚧)
- ReplicationService implementation
- RouterService implementation
- TransactionService implementation
- Integration with existing LauraDB components

### Planned
- Connection metrics and monitoring
- Request rate limiting
- Circuit breaker for failing nodes
- Load balancing across replicas
- Advanced TLS features (certificate rotation)

## Performance Considerations

### Connection Pooling
- Server reuses connections efficiently
- Keep-alive prevents connection churn
- Graceful shutdown ensures clean connection closure

### Concurrency
- Each RPC is handled in a separate goroutine
- MaxConcurrentRPCs prevents resource exhaustion
- Services use fine-grained locking for scalability

### Resource Limits
- MaxConnections prevents unbounded growth
- Connection timeouts prevent resource leaks
- Graceful shutdown has timeout to force stop if needed

## Security

### TLS/mTLS
When TLS is enabled:
- All data is encrypted in transit
- mTLS provides mutual authentication
- Certificate validation ensures node identity

### Best Practices
1. Always enable TLS in production
2. Use strong cipher suites
3. Rotate certificates regularly
4. Implement proper node authentication
5. Use firewall rules to restrict access

## Troubleshooting

### Server Won't Start
```
Error: address already in use
```
Solution: Change port or kill process using the port

### Connection Refused
```
Error: connection refused
```
Solution: Verify server is running and firewall allows connections

### TLS Handshake Failed
```
Error: tls: handshake failure
```
Solution: Verify certificate/key files and TLS configuration

### Deadline Exceeded
```
Error: context deadline exceeded
```
Solution: Increase request timeout or check network latency

## Next Steps

To complete the distributed cluster implementation:

1. Implement ReplicationService oplog streaming
2. Implement RouterService query routing
3. Implement TransactionService 2PC logic
4. Add comprehensive integration tests
5. Add performance benchmarks
6. Document deployment patterns

## References

- [gRPC Go Documentation](https://grpc.io/docs/languages/go/)
- [Protocol Buffer Schemas](../proto/README.md)
- [LauraDB Architecture](../../../docs/architecture.md)
