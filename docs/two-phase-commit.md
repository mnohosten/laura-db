# Two-Phase Commit (2PC) for Distributed Transactions

## Overview

LauraDB implements the classic Two-Phase Commit (2PC) protocol to enable atomic transactions across multiple database instances or participants. This allows you to coordinate distributed transactions that must either all commit or all abort together, maintaining ACID properties across distributed systems.

## What is Two-Phase Commit?

Two-Phase Commit is a distributed transaction protocol that ensures atomicity across multiple participants. It works in two phases:

1. **Prepare Phase**: The coordinator asks all participants if they're ready to commit
2. **Commit/Abort Phase**: If all participants vote YES, the coordinator tells everyone to commit; otherwise, it tells everyone to abort

## Architecture

### Components

#### Coordinator
The coordinator manages the 2PC protocol:
- Tracks participant states
- Sends prepare requests
- Collects votes
- Sends commit or abort decisions
- Handles timeouts and failures

#### Participants
Participants are resources that can join a distributed transaction:
- Receive prepare requests
- Vote YES or NO based on readiness to commit
- Execute commit or abort on coordinator's decision
- Examples: databases, message queues, external services

#### DatabaseParticipant
LauraDB provides `DatabaseParticipant` which wraps a Database instance to participate in 2PC:
- Manages transaction sessions
- Implements the Participant interface
- Handles prepare/commit/abort for database operations

## Usage

### Basic Example

```go
package main

import (
    "context"
    "github.com/mnohosten/laura-db/pkg/database"
    "github.com/mnohosten/laura-db/pkg/distributed"
    "github.com/mnohosten/laura-db/pkg/mvcc"
)

func main() {
    // Open databases
    db1, _ := database.Open("./data/db1")
    defer db1.Close()

    db2, _ := database.Open("./data/db2")
    defer db2.Close()

    // Create participants
    p1 := distributed.NewDatabaseParticipant("db1", db1)
    p2 := distributed.NewDatabaseParticipant("db2", db2)

    // Create coordinator
    txnID := mvcc.TxnID(1)
    coordinator := distributed.NewCoordinator(txnID, 0)

    // Add participants
    coordinator.AddParticipant(p1)
    coordinator.AddParticipant(p2)

    // Start sessions on each participant
    session1 := p1.StartTransaction(txnID)
    session2 := p2.StartTransaction(txnID)

    // Perform operations
    session1.InsertOne("users", map[string]interface{}{
        "name": "Alice",
        "age": int64(30),
    })

    session2.InsertOne("orders", map[string]interface{}{
        "user": "Alice",
        "amount": int64(100),
    })

    // Execute 2PC protocol
    ctx := context.Background()
    if err := coordinator.Execute(ctx); err != nil {
        // Transaction aborted
        log.Printf("Transaction failed: %v", err)
    } else {
        // Transaction committed across all databases
        log.Println("Transaction committed successfully")
    }
}
```

### Bank Transfer Example

```go
// Transfer money between accounts in different banks
func transferBetweenBanks(amount int64) error {
    // Set up databases for each bank
    bank1DB, _ := database.Open("./data/bank1")
    defer bank1DB.Close()

    bank2DB, _ := database.Open("./data/bank2")
    defer bank2DB.Close()

    clearingDB, _ := database.Open("./data/clearing")
    defer clearingDB.Close()

    // Create participants
    bank1P := distributed.NewDatabaseParticipant("bank1", bank1DB)
    bank2P := distributed.NewDatabaseParticipant("bank2", bank2DB)
    clearingP := distributed.NewDatabaseParticipant("clearing", clearingDB)

    // Create coordinator
    txnID := mvcc.TxnID(time.Now().Unix())
    coordinator := distributed.NewCoordinator(txnID, 30*time.Second)

    coordinator.AddParticipant(bank1P)
    coordinator.AddParticipant(bank2P)
    coordinator.AddParticipant(clearingP)

    // Start sessions
    bank1Session := bank1P.StartTransaction(txnID)
    bank2Session := bank2P.StartTransaction(txnID)
    clearingSession := clearingP.StartTransaction(txnID)

    // Debit sender (Bank 1)
    err := bank1Session.UpdateOne("accounts",
        map[string]interface{}{"account_id": "ACC-001"},
        map[string]interface{}{"$inc": map[string]interface{}{"balance": -amount}},
    )
    if err != nil {
        return err
    }

    // Credit receiver (Bank 2)
    err = bank2Session.UpdateOne("accounts",
        map[string]interface{}{"account_id": "ACC-002"},
        map[string]interface{}{"$inc": map[string]interface{}{"balance": amount}},
    )
    if err != nil {
        return err
    }

    // Record in clearing house
    _, err = clearingSession.InsertOne("transfers", map[string]interface{}{
        "from": "ACC-001",
        "to": "ACC-002",
        "amount": amount,
        "timestamp": time.Now(),
    })
    if err != nil {
        return err
    }

    // Execute 2PC
    ctx := context.Background()
    return coordinator.Execute(ctx)
}
```

## API Reference

### Coordinator

#### Creating a Coordinator

```go
func NewCoordinator(txnID mvcc.TxnID, timeout time.Duration) *Coordinator
```

Creates a new 2PC coordinator. If timeout is 0, defaults to 30 seconds.

#### Adding Participants

```go
func (c *Coordinator) AddParticipant(participant Participant) error
```

Adds a participant to the transaction. Must be called before starting the protocol.

#### Executing 2PC

```go
func (c *Coordinator) Execute(ctx context.Context) error
```

Runs the full 2PC protocol: prepare phase, then commit or abort based on votes.

#### Manual Control

```go
func (c *Coordinator) Prepare(ctx context.Context) (bool, error)
func (c *Coordinator) Commit(ctx context.Context) error
func (c *Coordinator) Abort(ctx context.Context) error
```

For advanced use cases, you can manually control each phase.

#### State Inspection

```go
func (c *Coordinator) GetState() CoordinatorState
func (c *Coordinator) GetParticipantState(id ParticipantID) (ParticipantState, error)
func (c *Coordinator) GetParticipantCount() int
```

### DatabaseParticipant

#### Creating a Participant

```go
func NewDatabaseParticipant(id string, db *database.Database) *DatabaseParticipant
```

Wraps a LauraDB database to participate in 2PC.

#### Starting a Transaction

```go
func (dp *DatabaseParticipant) StartTransaction(txnID mvcc.TxnID) *database.Session
```

Creates a new session for the transaction ID. Returns a Session that can be used to perform operations.

#### Getting a Session

```go
func (dp *DatabaseParticipant) GetSession(txnID mvcc.TxnID) (*database.Session, error)
```

Retrieves an existing session for a transaction ID.

### Participant Interface

To create custom participants (e.g., for message queues or external services):

```go
type Participant interface {
    // Prepare asks if ready to commit (Phase 1)
    Prepare(ctx context.Context, txnID mvcc.TxnID) (bool, error)

    // Commit tells participant to commit (Phase 2)
    Commit(ctx context.Context, txnID mvcc.TxnID) error

    // Abort tells participant to abort (Phase 2)
    Abort(ctx context.Context, txnID mvcc.TxnID) error

    // ID returns unique participant identifier
    ID() ParticipantID
}
```

## Protocol Details

### Successful Commit Flow

1. Coordinator: Send PREPARE to all participants
2. Participants: Validate and vote YES/NO
3. Coordinator: If all vote YES, send COMMIT to all
4. Participants: Commit and acknowledge
5. Coordinator: Transaction complete

### Abort Flow

1. Coordinator: Send PREPARE to all participants
2. Participants: One or more vote NO, or error occurs
3. Coordinator: Send ABORT to all participants
4. Participants: Rollback and acknowledge
5. Coordinator: Transaction aborted

### State Diagram

```
Coordinator States:
Init -> Preparing -> Committing -> Committed
  |         |            |
  |         +----------> Aborting -> Aborted
  |                        ^
  +-----------------------+

Participant States:
Init -> Prepared -> Committed
  |        |
  |        +-----> Aborted
  |
  +-------------> Aborted
```

## Error Handling

### Prepare Phase Failures

If any participant:
- Votes NO
- Returns an error
- Times out

The coordinator automatically aborts the transaction.

### Commit Phase Failures

If commit fails on any participant after all voted YES:
- This is a **critical situation**
- Some participants may have committed while others failed
- In production systems, this requires:
  - Retry logic
  - Transaction log
  - Manual intervention or recovery protocols

LauraDB logs these errors but doesn't automatically retry. For production use, implement:
- Persistent transaction log
- Automatic retry with exponential backoff
- Alerting and monitoring

### Timeout Handling

Each phase has a timeout (default 30 seconds):
- Context cancellation is respected
- Timed-out participants are treated as voting NO
- Coordinator aborts on timeout

## Conflict Detection

LauraDB's MVCC system provides automatic write conflict detection:

```go
// Transaction 1
session1.FindOne("accounts", filter)  // Read version 10
// ... time passes ...
session1.UpdateOne("accounts", filter, update)  // Try to write

// Transaction 2 (concurrent)
coll.UpdateOne("accounts", filter, update)  // Commits, creates version 11

// When Transaction 1 tries to commit via 2PC:
// Prepare succeeds, but Commit detects conflict
// Transaction 1 aborts automatically
```

## Performance Considerations

### Latency

2PC adds latency due to:
- Network round trips (prepare + commit)
- Participant preparation time
- Consensus overhead

Expected latency: `2 * (network_latency + participant_time)`

### Throughput

2PC reduces throughput because:
- Locks are held during both phases
- All participants must complete before any can finish
- Blocking protocol (not lock-free)

### Scalability

2PC scalability limits:
- O(n) prepare messages (n = participants)
- O(n) commit/abort messages
- Coordinator is a single point of coordination
- More participants = longer critical section

Best practices:
- Minimize number of participants
- Use timeout to prevent indefinite blocking
- Consider alternatives (Saga pattern, eventual consistency) for high-scale systems

## Limitations

### Blocking Protocol

2PC can block if:
- Coordinator crashes after prepare but before commit/abort decision
- Participants must wait for coordinator recovery
- Use persistent coordinator state for production

### Synchronous

All participants must respond before transaction completes:
- Not suitable for high-latency participants
- Consider asynchronous patterns for long-running operations

### No Partition Tolerance

2PC doesn't handle network partitions well:
- Split-brain scenarios can occur
- Use 3PC (Three-Phase Commit) or Paxos for partition tolerance
- LauraDB 2PC is best for controlled environments (same datacenter)

## Use Cases

### Good Use Cases

1. **Multi-database transactions**
   - Updating related data across databases
   - Maintaining referential integrity across services
   - Examples: bank transfers, order processing

2. **Microservices coordination**
   - Atomic operations across multiple services
   - Saga alternative when strong consistency required
   - Example: e-commerce order (inventory + payment + shipping)

3. **Data migration**
   - Ensuring consistency during data transfer
   - Atomic cutover between systems

### Poor Use Cases

1. **High-latency networks**
   - Use eventual consistency instead
   - Consider Saga pattern

2. **Many participants (>10)**
   - Coordination overhead too high
   - Consider batch operations or eventual consistency

3. **Long-running transactions**
   - Blocks resources for extended periods
   - Use asynchronous workflows instead

## Testing

### Unit Tests

```bash
# Test coordinator
go test ./pkg/distributed -run TestCoordinator

# Test database participant
go test ./pkg/distributed -run TestDatabaseParticipant

# All tests
go test ./pkg/distributed -v
```

### Integration Tests

```bash
# Test distributed transactions
go test ./pkg/distributed -run TestDistributedTransaction

# Bank transfer scenario
go test ./pkg/distributed -run TestMultiDatabaseBankTransfer
```

### Example Program

```bash
# Build and run demo
make build
./bin/distributed-2pc-demo
```

## Comparison with Alternatives

### 2PC vs Saga Pattern

| Aspect | 2PC | Saga |
|--------|-----|------|
| Consistency | Strong (ACID) | Eventual |
| Latency | Higher | Lower |
| Complexity | Lower | Higher |
| Scalability | Limited | High |
| Failure handling | Blocking | Non-blocking |

### 2PC vs Three-Phase Commit (3PC)

| Aspect | 2PC | 3PC |
|--------|-----|-----|
| Phases | 2 | 3 |
| Blocking | Yes (on coordinator failure) | No |
| Complexity | Lower | Higher |
| Network partitions | Poor | Better |
| Latency | Lower | Higher |

### 2PC vs Eventual Consistency

| Aspect | 2PC | Eventual Consistency |
|--------|-----|----------------------|
| Consistency | Immediate | Delayed |
| Availability | Lower | Higher |
| Partition tolerance | Poor | Good |
| Programming model | Simpler | More complex |
| Use case | Financial | Social media |

## Best Practices

1. **Keep transactions short**
   - Minimize time between prepare and commit
   - Release locks quickly

2. **Use timeouts**
   - Prevent indefinite blocking
   - Set appropriate timeout for your use case

3. **Monitor coordinator**
   - Log all 2PC operations
   - Alert on frequent aborts
   - Track latency metrics

4. **Handle failures gracefully**
   - Implement retry logic for transient failures
   - Log failures for manual intervention
   - Consider circuit breaker pattern

5. **Test failure scenarios**
   - Simulate participant failures
   - Test timeout handling
   - Verify abort behavior

6. **Document participants**
   - Clearly define each participant's role
   - Document expected behavior
   - Maintain participant inventory

## Future Enhancements

Potential improvements for production use:

1. **Persistent coordinator state**
   - Write transaction log to disk
   - Enable coordinator recovery after crashes

2. **Retry logic**
   - Automatic retry for transient failures
   - Exponential backoff

3. **Three-Phase Commit**
   - Add pre-commit phase
   - Better partition tolerance

4. **Distributed coordinator**
   - Use consensus (Raft/Paxos)
   - Eliminate single point of failure

5. **Performance optimizations**
   - Parallel prepare requests (already implemented)
   - Batch commit messages
   - Pipelining

## References

- [Two-Phase Commit Protocol (Wikipedia)](https://en.wikipedia.org/wiki/Two-phase_commit_protocol)
- [Distributed Transactions: The Icebergs of Microservices](https://www.grahamlea.com/2016/08/distributed-transactions-microservices-icebergs/)
- [Life Beyond Distributed Transactions (Pat Helland)](https://queue.acm.org/detail.cfm?id=3025012)
- LauraDB Session API: `pkg/database/session.go`
- LauraDB MVCC: `docs/mvcc.md`

## Summary

Two-Phase Commit in LauraDB provides:
- ✅ Atomic distributed transactions
- ✅ Strong consistency guarantees
- ✅ Automatic conflict detection
- ✅ Simple API
- ✅ Database participant support
- ⚠️ Blocking protocol (coordinator dependency)
- ⚠️ Limited partition tolerance
- ⚠️ Not suitable for high-scale distributed systems

Use 2PC when you need strong consistency guarantees and can tolerate the coordination overhead. For high-scale or partition-tolerant systems, consider eventual consistency patterns like Saga.
