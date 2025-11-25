# Chaos Engineering for LauraDB

LauraDB includes a comprehensive chaos engineering framework for testing system resilience under various failure conditions. The chaos testing infrastructure enables systematic fault injection to verify recovery mechanisms and data integrity.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Fault Types](#fault-types)
- [Fault Injector API](#fault-injector-api)
- [Scenario-Based Testing](#scenario-based-testing)
- [Best Practices](#best-practices)
- [Examples](#examples)

## Overview

Chaos engineering is the practice of intentionally introducing faults into a system to test its resilience and uncover weaknesses before they cause production outages. LauraDB's chaos framework provides:

- **Fault Injection**: Programmable failure injection for disk I/O, network, and process failures
- **Scenario Framework**: Structured test scenarios with steps and assertions
- **Statistical Control**: Probability-based fault injection for realistic testing
- **Event Tracking**: Comprehensive logging of chaos events
- **Recovery Verification**: Built-in assertions to verify system recovery

### Supported Fault Types

| Fault Type | Description | Use Case |
|------------|-------------|----------|
| **DiskRead** | Simulates disk read failures | Test read error handling |
| **DiskWrite** | Simulates disk write failures | Test write error handling and retry logic |
| **DiskFull** | Simulates out-of-disk-space | Test disk space exhaustion handling |
| **DiskCorruption** | Simulates data corruption | Test corruption detection and recovery |
| **NetworkPartition** | Simulates network disconnection | Test distributed system behavior |
| **ProcessCrash** | Simulates abrupt termination | Test crash recovery and durability |
| **SlowIO** | Simulates slow disk/network I/O | Test timeout handling and performance degradation |
| **MemoryPressure** | Simulates low memory conditions | Test memory management under pressure |

## Quick Start

### Basic Fault Injection

```go
package main

import (
    "github.com/mnohosten/laura-db/pkg/chaos"
    "github.com/mnohosten/laura-db/pkg/database"
)

func main() {
    // Create fault injector
    injector := chaos.NewFaultInjector(42)  // Seed for reproducibility
    injector.Enable()

    // Enable disk write failures (30% probability)
    injector.EnableFault(chaos.FaultTypeDiskWrite, 0.3)

    // Open database (with potential failures)
    db, _ := database.Open(database.DefaultConfig("./data"))
    defer db.Close()

    coll := db.Collection("test")

    // Insert operations will randomly fail
    for i := 0; i < 100; i++ {
        doc := map[string]interface{}{"id": int64(i)}
        if _, err := coll.InsertOne(doc); err != nil {
            // Handle failure (retry, log, etc.)
            println("Insert failed:", err.Error())
        }
    }

    // Disable fault injection
    injector.Disable()

    // Check how many failures occurred
    count := injector.GetTriggerCount(chaos.FaultTypeDiskWrite)
    println("Disk write faults triggered:", count)
}
```

### Testing Database Recovery

```go
func TestDatabaseRecovery(t *testing.T) {
    injector := chaos.NewFaultInjector(42)
    injector.Enable()

    // Insert data
    db, _ := database.Open(database.DefaultConfig("./test-data"))
    coll := db.Collection("recovery_test")
    for i := 0; i < 50; i++ {
        coll.InsertOne(map[string]interface{}{"id": int64(i)})
    }

    // Simulate crash (close without proper shutdown)
    db.Close()

    // Reopen and verify data
    db2, _ := database.Open(database.DefaultConfig("./test-data"))
    coll2 := db2.Collection("recovery_test")
    docs, _ := coll2.Find(map[string]interface{}{})

    if len(docs) != 50 {
        t.Errorf("Data loss detected: expected 50 docs, found %d", len(docs))
    }

    db2.Close()
}
```

## Fault Types

### DiskRead

Simulates failures when reading from disk.

**Configuration:**
```go
injector.ConfigureFault(&chaos.FaultConfig{
    Type:        chaos.FaultTypeDiskRead,
    Enabled:     true,
    Probability: 0.2,  // 20% failure rate
    ErrorMsg:    "disk read error: I/O error",
})
```

**Use Cases:**
- Test read retry logic
- Verify error propagation
- Test fallback to replicas

### DiskWrite

Simulates failures when writing to disk.

**Configuration:**
```go
injector.ConfigureFault(&chaos.FaultConfig{
    Type:        chaos.FaultTypeDiskWrite,
    Enabled:     true,
    Probability: 0.1,  // 10% failure rate
    ErrorMsg:    "disk write error: device error",
})
```

**Use Cases:**
- Test transaction rollback
- Test WAL write failures
- Test dirty page handling

### DiskFull

Simulates out-of-disk-space conditions.

**Configuration:**
```go
injector.EnableFault(chaos.FaultTypeDiskFull, 1.0)  // Always fail
```

**Use Cases:**
- Test space exhaustion handling
- Test compaction triggers
- Test error messages to users

### SlowIO

Adds delays to I/O operations without failing.

**Configuration:**
```go
injector.ConfigureFault(&chaos.FaultConfig{
    Type:        chaos.FaultTypeSlowIO,
    Enabled:     true,
    Probability: 1.0,
    Delay:       100 * time.Millisecond,  // Add 100ms delay
})
```

**Use Cases:**
- Test timeout handling
- Test performance under slow disk
- Test client retry logic

### ProcessCrash

Simulates abrupt process termination.

**Configuration:**
```go
injector.EnableFault(chaos.FaultTypeProcessCrash, 0.01)  // 1% crash rate
```

**Use Cases:**
- Test WAL recovery
- Test data durability
- Test lock cleanup

## Fault Injector API

### Creating an Injector

```go
// Create with random seed
injector := chaos.NewFaultInjector(0)

// Create with specific seed (reproducible tests)
injector := chaos.NewFaultInjector(42)
```

### Enabling/Disabling

```go
// Enable fault injection globally
injector.Enable()

// Check if enabled
if injector.IsEnabled() {
    // ...
}

// Disable all fault injection
injector.Disable()
```

### Configuring Faults

```go
// Simple: Enable fault with probability
injector.EnableFault(chaos.FaultTypeDiskWrite, 0.3)

// Advanced: Full configuration
injector.ConfigureFault(&chaos.FaultConfig{
    Type:        chaos.FaultTypeDiskWrite,
    Enabled:     true,
    Probability: 0.3,
    Delay:       50 * time.Millisecond,
    ErrorMsg:    "custom error message",
    Callback:    func() {
        // Called when fault is triggered
        log.Println("Fault triggered!")
    },
})

// Disable specific fault
injector.DisableFault(chaos.FaultTypeDiskWrite)
```

### Checking Fault Triggers

```go
// Check if fault should be injected
shouldInject, err := injector.ShouldInjectFault(chaos.FaultTypeDiskWrite)
if shouldInject {
    return err  // Return injected error
}

// Force fault injection (ignore probability)
if err := injector.InjectFault(chaos.FaultTypeDiskWrite); err != nil {
    return err
}
```

### Monitoring

```go
// Get trigger count
count := injector.GetTriggerCount(chaos.FaultTypeDiskWrite)
fmt.Printf("Fault triggered %d times\n", count)

// Reset counters
injector.Reset()

// Add event callback
injector.AddEventCallback(func(event chaos.ChaosEvent) {
    fmt.Printf("[%s] %s: %s (triggered: %v)\n",
        event.Timestamp.Format("15:04:05"),
        event.Type,
        event.Message,
        event.Triggered,
    )
})
```

## Scenario-Based Testing

### Creating Scenarios

Scenarios provide structured multi-step chaos tests with assertions.

```go
scenario := &chaos.Scenario{
    Name:        "Disk Failure Recovery",
    Description: "Tests database recovery from disk failures",
    Duration:    30 * time.Second,
}
```

### Adding Steps

```go
scenario.Steps = []chaos.ScenarioStep{
    {
        Name: "Normal operations",
        Action: func(ctx context.Context) error {
            // Normal database operations
            return nil
        },
    },
    {
        Name:  "Enable disk failures",
        Delay: 1 * time.Second,
        FaultConfig: &chaos.FaultConfig{
            Type:        chaos.FaultTypeDiskWrite,
            Enabled:     true,
            Probability: 0.5,
        },
        Duration: 10 * time.Second,
    },
    {
        Name: "Verify recovery",
        Action: func(ctx context.Context) error {
            // Verify database recovered
            return nil
        },
    },
}
```

### Adding Assertions

```go
scenario.Assertions = []chaos.Assertion{
    {
        Name: "Database is functional",
        Check: func() error {
            // Check database state
            if db == nil {
                return fmt.Errorf("database is nil")
            }
            return nil
        },
        Critical: true,  // Fail scenario if this fails
    },
    {
        Name: "No data loss",
        Check: func() error {
            docs, _ := coll.Find(map[string]interface{}{})
            if len(docs) < expectedCount {
                return fmt.Errorf("data loss detected")
            }
            return nil
        },
        Critical: true,
    },
}
```

### Running Scenarios

```go
runner := chaos.NewScenarioRunner(injector)
result, err := runner.Run(context.Background(), scenario)

if err != nil {
    t.Errorf("Scenario failed: %v", err)
}

// Print results
fmt.Println(runner.PrintResult(result))

// Check individual assertions
for _, assertion := range result.AssertionResults {
    if !assertion.Success {
        fmt.Printf("Assertion failed: %s: %v\n",
            assertion.Assertion.Name,
            assertion.Error,
        )
    }
}
```

### Predefined Scenario Builders

```go
// Disk failure scenario
scenario := chaos.DiskFailureScenario("Test disk failures",
    func(ctx context.Context) error {
        // Your test logic
        return nil
    },
)

// Slow disk scenario
scenario := chaos.SlowDiskScenario("Test slow I/O",
    100*time.Millisecond,  // Add 100ms delay
    func(ctx context.Context) error {
        // Your test logic
        return nil
    },
)

// Process crash scenario
scenario := chaos.ProcessCrashScenario("Test crash recovery",
    func(ctx context.Context) error {
        // Operations before crash
        return nil
    },
    func(ctx context.Context) error {
        // Recovery action 1
        return nil
    },
    func(ctx context.Context) error {
        // Recovery action 2
        return nil
    },
)
```

## Best Practices

### 1. Use Deterministic Seeds for Reproducibility

```go
// Good: Reproducible tests
injector := chaos.NewFaultInjector(42)

// Bad: Random behavior on each run
injector := chaos.NewFaultInjector(0)
```

### 2. Start with Low Probability

```go
// Good: Start conservative
injector.EnableFault(chaos.FaultTypeDiskWrite, 0.1)  // 10%

// Bad: Too aggressive for initial testing
injector.EnableFault(chaos.FaultTypeDiskWrite, 0.9)  // 90%
```

### 3. Test One Fault Type at a Time

```go
// Good: Isolate fault types
injector.EnableFault(chaos.FaultTypeDiskWrite, 0.3)
// Run tests
injector.DisableFault(chaos.FaultTypeDiskWrite)

injector.EnableFault(chaos.FaultTypeDiskRead, 0.3)
// Run tests

// Bad: Multiple faults simultaneously (unless testing interactions)
injector.EnableFault(chaos.FaultTypeDiskWrite, 0.3)
injector.EnableFault(chaos.FaultTypeDiskRead, 0.3)
injector.EnableFault(chaos.FaultTypeSlowIO, 0.5)
```

### 4. Always Verify Recovery

```go
// Good: Test recovery
injector.Enable()
// ... cause failures ...
injector.Disable()

// Verify system recovered
docs, err := coll.Find(map[string]interface{}{})
assert.NoError(t, err)
assert.Equal(t, expectedCount, len(docs))

// Bad: Only test failures, not recovery
injector.Enable()
// ... cause failures ...
// (no verification)
```

### 5. Use Scenarios for Complex Tests

```go
// Good: Structured scenario
scenario := &chaos.Scenario{
    Name: "Multi-step failure test",
    Steps: []chaos.ScenarioStep{...},
    Assertions: []chaos.Assertion{...},
}
runner.Run(context.Background(), scenario)

// Bad: Ad-hoc chaos testing
injector.Enable()
// ... random operations ...
injector.Disable()
// (no structure or assertions)
```

### 6. Monitor Fault Injection

```go
// Good: Track what happened
injector.AddEventCallback(func(event chaos.ChaosEvent) {
    log.Printf("Chaos event: %s", event.Message)
})

count := injector.GetTriggerCount(chaos.FaultTypeDiskWrite)
t.Logf("Disk write failures: %d", count)

// Bad: No visibility into chaos behavior
```

## Examples

### Example 1: Testing Write Durability

```go
func TestWriteDurability(t *testing.T) {
    injector := chaos.NewFaultInjector(42)
    injector.Enable()

    dataDir := filepath.Join(os.TempDir(), "chaos-test-durability")
    defer os.RemoveAll(dataDir)

    // Phase 1: Write data with potential failures
    db, _ := database.Open(database.DefaultConfig(dataDir))
    coll := db.Collection("test")

    injector.EnableFault(chaos.FaultTypeDiskWrite, 0.2)  // 20% failures

    insertedCount := 0
    for i := 0; i < 100; i++ {
        doc := map[string]interface{}{"id": int64(i), "value": i * 100}
        if _, err := coll.InsertOne(doc); err == nil {
            insertedCount++
        }
    }

    t.Logf("Successfully inserted %d/%d documents", insertedCount, 100)

    // Close database properly
    db.Close()

    // Phase 2: Reopen and verify
    injector.Disable()
    db2, _ := database.Open(database.DefaultConfig(dataDir))
    coll2 := db2.Collection("test")

    docs, _ := coll2.Find(map[string]interface{}{})
    if len(docs) != insertedCount {
        t.Errorf("Data loss: expected %d, found %d", insertedCount, len(docs))
    }

    db2.Close()
}
```

### Example 2: Testing Concurrent Operations with Chaos

```go
func TestConcurrentChaos(t *testing.T) {
    injector := chaos.NewFaultInjector(42)
    injector.Enable()
    injector.EnableFault(chaos.FaultTypeDiskWrite, 0.1)  // 10% failures
    injector.EnableFault(chaos.FaultTypeSlowIO, 0.2)     // 20% slow I/O

    db, _ := database.Open(database.DefaultConfig("./test-concurrent"))
    defer db.Close()

    coll := db.Collection("concurrent")

    var wg sync.WaitGroup
    successCount := int32(0)
    failCount := int32(0)

    // Start 5 concurrent writers
    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()

            for j := 0; j < 100; j++ {
                doc := map[string]interface{}{
                    "worker": int64(workerID),
                    "seq":    int64(j),
                }

                if _, err := coll.InsertOne(doc); err != nil {
                    atomic.AddInt32(&failCount, 1)
                } else {
                    atomic.AddInt32(&successCount, 1)
                }
            }
        }(i)
    }

    wg.Wait()

    t.Logf("Concurrent chaos: %d successes, %d failures", successCount, failCount)

    // Verify database consistency
    docs, _ := coll.Find(map[string]interface{}{})
    if len(docs) != int(successCount) {
        t.Errorf("Inconsistency: expected %d docs, found %d", successCount, len(docs))
    }
}
```

### Example 3: Network Partition Simulation

```go
func TestNetworkPartition(t *testing.T) {
    injector := chaos.NewFaultInjector(42)
    injector.Enable()

    scenario := chaos.NetworkPartitionScenario(
        "Network Partition Test",
        5*time.Second,  // Partition lasts 5 seconds
        func(ctx context.Context) error {
            // Operations during partition
            // These should fail or timeout
            return nil
        },
    )

    runner := chaos.NewScenarioRunner(injector)
    result, _ := runner.Run(context.Background(), scenario)

    t.Log(runner.PrintResult(result))
}
```

## Integration with Testing

### Skip Chaos Tests in Short Mode

```go
func TestChaosScenario(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping chaos test in short mode")
    }

    // Chaos test code...
}
```

### Run Chaos Tests

```bash
# Run all tests (including chaos)
go test ./pkg/...

# Skip chaos tests (fast)
go test -short ./pkg/...

# Run only chaos tests
go test ./pkg/chaos -v
```

## Troubleshooting

### Issue: Tests are Flaky

**Solution**: Use deterministic seeds and increase fault probability to ensure consistent behavior.

```go
// Deterministic seed
injector := chaos.NewFaultInjector(42)

// Higher probability for consistent testing
injector.EnableFault(chaos.FaultTypeDiskWrite, 0.5)  // 50%
```

### Issue: No Faults Triggered

**Solution**: Verify fault injection is enabled and fault type is configured.

```go
if !injector.IsEnabled() {
    panic("Fault injection not enabled!")
}

count := injector.GetTriggerCount(chaos.FaultTypeDiskWrite)
if count == 0 {
    t.Log("Warning: No faults triggered")
}
```

### Issue: Tests Take Too Long

**Solution**: Reduce scenario duration and use timeouts.

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := runner.Run(ctx, scenario)
```

## See Also

- [Testing Guide](testing.md) - General testing documentation
- [Performance Testing](regression-testing.md) - Performance regression testing
- [Integration Tests](../pkg/database/integration_test.go) - Example integration tests
