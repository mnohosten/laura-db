package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/replication"
)

func main() {
	fmt.Println("==============================================")
	fmt.Println("LauraDB - Write Concern Demonstration")
	fmt.Println("==============================================")

	// Create temporary directories for databases
	tmpDir := "tmp_write_concern_demo"

	// Demo 1: Unacknowledged writes (w:0)
	demo1UnacknowledgedWrites(tmpDir)

	// Demo 2: Primary-only acknowledgment (w:1, default)
	demo2PrimaryOnlyAck(tmpDir)

	// Demo 3: Two-node acknowledgment (w:2)
	demo3TwoNodeAck(tmpDir)

	// Demo 4: Majority acknowledgment
	demo4MajorityAck(tmpDir)

	// Demo 5: Write with timeout
	demo5WriteWithTimeout(tmpDir)

	// Demo 6: Journal sync (w:1, j:true)
	demo6JournalSync(tmpDir)

	fmt.Println("\n==============================================")
	fmt.Println("All demonstrations completed successfully!")
	fmt.Println("==============================================")
}

func demo1UnacknowledgedWrites(tmpDir string) {
	fmt.Println("\n--- Demo 1: Unacknowledged Writes (w:0) ---")
	fmt.Println("Fire-and-forget writes with no acknowledgment")

	// Create database and replica set
	db := createTestDatabase(filepath.Join(tmpDir, "demo1"))
	defer db.Close()

	rs := createReplicaSet("rs_demo1", "node1", db, filepath.Join(tmpDir, "demo1_oplog"))
	defer rs.Stop()

	// Become primary
	rs.Start()
	rs.BecomePrimary()

	// Add some members
	rs.AddMember("node2", 1, true)
	rs.AddMember("node3", 1, true)

	// Write with w:0 (unacknowledged)
	wc := replication.UnacknowledgedWriteConcern()
	entry := replication.CreateInsertEntry("testdb", "users", map[string]interface{}{
		"_id":  "user1",
		"name": "Alice",
		"age":  int64(30),
	})

	ctx := context.Background()
	result, err := rs.WriteWithConcern(ctx, entry, wc)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Write concern: %v\n", wc)
	fmt.Printf("Result: %v\n", result)
	fmt.Printf("✓ Write completed with no acknowledgment (fastest, least safe)\n")
}

func demo2PrimaryOnlyAck(tmpDir string) {
	fmt.Println("\n--- Demo 2: Primary-Only Acknowledgment (w:1, default) ---")
	fmt.Println("Wait for primary to acknowledge the write")

	db := createTestDatabase(filepath.Join(tmpDir, "demo2"))
	defer db.Close()

	rs := createReplicaSet("rs_demo2", "node1", db, filepath.Join(tmpDir, "demo2_oplog"))
	defer rs.Stop()

	rs.Start()
	rs.BecomePrimary()

	rs.AddMember("node2", 1, true)
	rs.AddMember("node3", 1, true)

	// Write with w:1 (default)
	wc := replication.W1WriteConcern()
	entry := replication.CreateInsertEntry("testdb", "users", map[string]interface{}{
		"_id":   "user2",
		"name":  "Bob",
		"email": "bob@example.com",
	})

	ctx := context.Background()
	result, err := rs.WriteWithConcern(ctx, entry, wc)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Write concern: %v\n", wc)
	fmt.Printf("Result: %v\n", result)
	fmt.Printf("✓ Write acknowledged by primary (default, good balance)\n")
}

func demo3TwoNodeAck(tmpDir string) {
	fmt.Println("\n--- Demo 3: Two-Node Acknowledgment (w:2) ---")
	fmt.Println("Wait for primary + 1 secondary to acknowledge")

	db := createTestDatabase(filepath.Join(tmpDir, "demo3"))
	defer db.Close()

	rs := createReplicaSet("rs_demo3", "node1", db, filepath.Join(tmpDir, "demo3_oplog"))
	defer rs.Stop()

	rs.Start()
	rs.BecomePrimary()

	rs.AddMember("node2", 1, true)
	rs.AddMember("node3", 1, true)

	// Write with w:2
	wc := replication.W2WriteConcern().WithTimeout(5 * time.Second)
	entry := replication.CreateInsertEntry("testdb", "users", map[string]interface{}{
		"_id":     "user3",
		"name":    "Charlie",
		"role":    "admin",
		"created": time.Now(),
	})

	ctx := context.Background()

	// Simulate node2 catching up
	go func() {
		time.Sleep(100 * time.Millisecond)
		rs.UpdateMemberHeartbeat("node2", replication.OpID(1))
		fmt.Println("  → node2 has replicated the write")
	}()

	result, err := rs.WriteWithConcern(ctx, entry, wc)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Write concern: %v\n", wc)
	fmt.Printf("Result: %v\n", result)
	fmt.Printf("✓ Write acknowledged by 2 nodes (more durable)\n")
}

func demo4MajorityAck(tmpDir string) {
	fmt.Println("\n--- Demo 4: Majority Acknowledgment (w:majority) ---")
	fmt.Println("Wait for majority of voting members to acknowledge")

	db := createTestDatabase(filepath.Join(tmpDir, "demo4"))
	defer db.Close()

	rs := createReplicaSet("rs_demo4", "node1", db, filepath.Join(tmpDir, "demo4_oplog"))
	defer rs.Stop()

	rs.Start()
	rs.BecomePrimary()

	// Add 2 secondaries (total 3, majority = 2)
	rs.AddMember("node2", 1, true)
	rs.AddMember("node3", 1, true)

	// Write with w:majority
	wc := replication.MajorityWriteConcern().WithTimeout(5 * time.Second)
	entry := replication.CreateInsertEntry("testdb", "orders", map[string]interface{}{
		"_id":        "order123",
		"customer":   "Alice",
		"total":      int64(29999),
		"items":      []interface{}{"laptop", "mouse"},
		"created_at": time.Now(),
	})

	ctx := context.Background()

	// Simulate node2 catching up (to achieve majority)
	go func() {
		time.Sleep(150 * time.Millisecond)
		rs.UpdateMemberHeartbeat("node2", replication.OpID(1))
		fmt.Println("  → node2 has replicated (majority achieved: 2/3)")
	}()

	result, err := rs.WriteWithConcern(ctx, entry, wc)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Write concern: %v\n", wc)
	fmt.Printf("Result: %v\n", result)
	fmt.Printf("✓ Write acknowledged by majority (recommended for production)\n")
}

func demo5WriteWithTimeout(tmpDir string) {
	fmt.Println("\n--- Demo 5: Write with Timeout ---")
	fmt.Println("Demonstrate timeout when replication is slow")

	db := createTestDatabase(filepath.Join(tmpDir, "demo5"))
	defer db.Close()

	rs := createReplicaSet("rs_demo5", "node1", db, filepath.Join(tmpDir, "demo5_oplog"))
	defer rs.Stop()

	rs.Start()
	rs.BecomePrimary()

	// Add secondaries that won't replicate in time
	rs.AddMember("node2", 1, true)
	rs.AddMember("node3", 1, true)

	// Write with w:3 and short timeout
	wc := replication.W3WriteConcern().WithTimeout(200 * time.Millisecond)
	entry := replication.CreateInsertEntry("testdb", "logs", map[string]interface{}{
		"_id":       "log001",
		"message":   "Testing timeout behavior",
		"timestamp": time.Now(),
	})

	ctx := context.Background()
	fmt.Println("  Waiting for 3 nodes with 200ms timeout...")

	result, err := rs.WriteWithConcern(ctx, entry, wc)

	// This should timeout because secondaries won't replicate
	if err != nil {
		fmt.Printf("Expected timeout error: %v\n", err)
		if result != nil {
			fmt.Printf("Partial result: %v\n", result)
			fmt.Printf("✓ Write logged on primary, but not replicated to required nodes\n")
		}
	} else {
		fmt.Printf("Unexpected: write succeeded without timeout\n")
	}
}

func demo6JournalSync(tmpDir string) {
	fmt.Println("\n--- Demo 6: Journal Sync (w:1, j:true) ---")
	fmt.Println("Wait for write to be persisted to journal/oplog")

	db := createTestDatabase(filepath.Join(tmpDir, "demo6"))
	defer db.Close()

	rs := createReplicaSet("rs_demo6", "node1", db, filepath.Join(tmpDir, "demo6_oplog"))
	defer rs.Stop()

	rs.Start()
	rs.BecomePrimary()

	// Write with journal sync
	wc := replication.W1WriteConcern().WithJournal(true)
	entry := replication.CreateInsertEntry("testdb", "transactions", map[string]interface{}{
		"_id":    "txn001",
		"from":   "account123",
		"to":     "account456",
		"amount": int64(50000),
		"status": "completed",
	})

	ctx := context.Background()
	result, err := rs.WriteWithConcern(ctx, entry, wc)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Write concern: %v\n", wc)
	fmt.Printf("Result: %v\n", result)
	if result.JournalSynced {
		fmt.Printf("✓ Write persisted to journal (most durable for single node)\n")
	}
}

// Helper functions

func createTestDatabase(path string) *database.Database {
	db, err := database.Open(database.DefaultConfig(path))
	if err != nil {
		panic(fmt.Sprintf("Failed to open database: %v", err))
	}
	return db
}

func createReplicaSet(name, nodeID string, db *database.Database, oplogPath string) *replication.ReplicaSet {
	config := replication.DefaultReplicaSetConfig(name, nodeID, db, oplogPath)
	rs, err := replication.NewReplicaSet(config)
	if err != nil {
		panic(fmt.Sprintf("Failed to create replica set: %v", err))
	}
	return rs
}
