# LauraDB Cluster Protocol Buffers

This directory contains the Protocol Buffer definitions for LauraDB's distributed cluster communication.

## Overview

LauraDB uses gRPC for all inter-node communication in a distributed cluster. The protocol is divided into four main areas:

### 1. Cluster Management (`cluster.proto`)

Handles node discovery, membership, and health monitoring:

- **ClusterService**: Node registration, heartbeats, topology management
- **Messages**: Node, ClusterTopology, HeartbeatRequest/Response
- **Use Cases**:
  - Node joins/leaves cluster
  - Periodic health checks
  - Topology change propagation

### 2. Replication (`replication.proto`)

Handles data replication between primary and secondary nodes:

- **ReplicationService**: Oplog streaming, initial sync, leader election
- **Messages**: OplogEntry, VoteRequest/Response, WriteAcknowledgment
- **Use Cases**:
  - Stream operations from primary to secondaries
  - Full data sync for new replicas
  - Raft-style leader election
  - Write concern acknowledgments

### 3. Routing (`routing.proto`)

Handles query routing to appropriate shards:

- **RouterService**: Query, Insert, Update, Delete, Aggregate operations
- **ConfigService**: Shard management, chunk splitting/migration
- **Messages**: QueryRequest/Response, ShardMapRequest/Response
- **Use Cases**:
  - Route queries to correct shard(s)
  - Scatter-gather for multi-shard queries
  - Shard rebalancing
  - Chunk management

### 4. Distributed Transactions (`transaction.proto`)

Handles two-phase commit (2PC) for distributed transactions:

- **TransactionService**: Prepare, Commit, Abort phases
- **Messages**: PrepareRequest/Response, CommitRequest/Response
- **Use Cases**:
  - Atomic commits across multiple shards
  - Transaction recovery after coordinator failure
  - Distributed ACID guarantees

## Generating Go Code

To generate Go code from these protobuf definitions:

```bash
# Install protoc compiler if not already installed
# macOS:
brew install protobuf

# Linux:
apt-get install -y protobuf-compiler

# Install Go protobuf plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate Go code
make proto
```

This will generate `*.pb.go` and `*_grpc.pb.go` files in this directory.

## Protocol Design Principles

### 1. Binary Serialization
All data is serialized using Protocol Buffers' binary format for efficiency. Complex structures like BSON documents are pre-serialized and transmitted as `bytes` fields.

### 2. Versioning
The `ClusterTopology` and `GetShardMapResponse` messages include version numbers to enable efficient polling and change detection.

### 3. Streaming
Long-running operations like oplog replication use gRPC streaming (`stream` keyword) to avoid repeatedly establishing connections.

### 4. Idempotency
All RPC methods are designed to be idempotent where possible. Request IDs enable deduplication.

### 5. Error Handling
All response messages include explicit success/error fields rather than relying solely on gRPC status codes.

## Security Considerations

All gRPC communication should use TLS/mTLS:

- **TLS**: Encrypts data in transit
- **mTLS**: Mutual authentication ensures only authorized nodes can join
- **Certificate rotation**: Support hot reload of certificates

See the server configuration documentation for TLS setup details.

## Example Usage

### Node Registration

```go
client := pb.NewClusterServiceClient(conn)
req := &pb.NodeRegistrationRequest{
    Node: &pb.Node{
        Id:       "node-1",
        Host:     "192.168.1.10",
        Port:     27017,
        Role:     pb.NodeRole_SECONDARY,
        Priority: 5,
    },
    SeedNodes: []string{"node-0:27017"},
}
resp, err := client.RegisterNode(ctx, req)
```

### Oplog Streaming

```go
client := pb.NewReplicationServiceClient(conn)
req := &pb.ReplicationStreamRequest{
    ReplicaId:     "replica-1",
    StartOplogId:  1000,
    InitialSync:   false,
}
stream, err := client.StreamOplog(ctx, req)
for {
    resp, err := stream.Recv()
    // Process oplog entries
}
```

### Query Routing

```go
client := pb.NewRouterServiceClient(conn)
req := &pb.QueryRequest{
    RequestId:  uuid.New().String(),
    Database:   "mydb",
    Collection: "users",
    Query:      bsonQuery, // Pre-serialized BSON
}
resp, err := client.Query(ctx, req)
```

## Future Enhancements

Potential additions to the protocol:

- **Compression**: Add optional compression for large messages
- **Batching**: Batch multiple operations in single RPC
- **Priority**: Add priority levels for different request types
- **Metrics**: Embed performance metrics in responses
- **Tracing**: Add distributed tracing support (OpenTelemetry)

## References

- [gRPC Documentation](https://grpc.io/docs/)
- [Protocol Buffers Guide](https://developers.google.com/protocol-buffers)
- [MongoDB Wire Protocol](https://docs.mongodb.com/manual/reference/mongodb-wire-protocol/)
- [Raft Consensus Algorithm](https://raft.github.io/)
