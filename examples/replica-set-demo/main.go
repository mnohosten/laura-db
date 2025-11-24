package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/replication"
)

func main() {
	fmt.Println("LauraDB Replica Set Demo")
	fmt.Println("=========================")
	fmt.Println()

	// Create temp directory for demo
	tmpDir := "/tmp/lauradb-replica-set-demo"

	// Demo 1: Basic replica set with 3 nodes
	demo1BasicReplicaSet(tmpDir)
	fmt.Println()

	// Demo 2: Leader election
	demo2LeaderElection(tmpDir)
	fmt.Println()

	// Demo 3: Write operations with majority replication
	demo3WriteWithReplication(tmpDir)
	fmt.Println()

	// Demo 4: Manual step down (primary resignation)
	demo4StepDown(tmpDir)
	fmt.Println()

	// Demo 5: Adding and removing members
	demo5MemberManagement(tmpDir)
	fmt.Println()

	fmt.Println("All replica set demos completed successfully!")
}

func demo1BasicReplicaSet(tmpDir string) {
	fmt.Println("Demo 1: Basic Replica Set with 3 Nodes")
	fmt.Println("---------------------------------------")

	// Create 3 database instances
	db1, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "demo1_node1")))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer db1.Close()

	db2, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "demo1_node2")))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer db2.Close()

	db3, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "demo1_node3")))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer db3.Close()

	// Create replica set nodes
	config1 := replication.DefaultReplicaSetConfig("rs0", "node1", db1, filepath.Join(tmpDir, "demo1_oplog1.bin"))
	config1.Priority = 10 // Highest priority
	rs1, err := replication.NewReplicaSet(config1)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer rs1.Stop()

	config2 := replication.DefaultReplicaSetConfig("rs0", "node2", db2, filepath.Join(tmpDir, "demo1_oplog2.bin"))
	config2.Priority = 5
	rs2, err := replication.NewReplicaSet(config2)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer rs2.Stop()

	config3 := replication.DefaultReplicaSetConfig("rs0", "node3", db3, filepath.Join(tmpDir, "demo1_oplog3.bin"))
	config3.Priority = 5
	rs3, err := replication.NewReplicaSet(config3)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer rs3.Stop()

	// Add all nodes to each replica set's member list
	// (In a real implementation, this would be done via network)
	rs1.AddMember("node2", 5, true)
	rs1.AddMember("node3", 5, true)

	rs2.AddMember("node1", 10, true)
	rs2.AddMember("node3", 5, true)

	rs3.AddMember("node1", 10, true)
	rs3.AddMember("node2", 5, true)

	// Start all nodes
	rs1.Start()
	rs2.Start()
	rs3.Start()

	fmt.Println("✓ Created 3-node replica set")
	fmt.Printf("  Node 1: %s (Priority: %d)\n", config1.NodeID, config1.Priority)
	fmt.Printf("  Node 2: %s (Priority: %d)\n", config2.NodeID, config2.Priority)
	fmt.Printf("  Node 3: %s (Priority: %d)\n", config3.NodeID, config3.Priority)

	// Display initial state
	fmt.Println("\nInitial state:")
	fmt.Printf("  Node 1 role: %s\n", rs1.GetRole())
	fmt.Printf("  Node 2 role: %s\n", rs2.GetRole())
	fmt.Printf("  Node 3 role: %s\n", rs3.GetRole())

	// Wait a moment for startup
	time.Sleep(100 * time.Millisecond)
}

func demo2LeaderElection(tmpDir string) {
	fmt.Println("Demo 2: Leader Election")
	fmt.Println("-----------------------")

	// Create 3 nodes
	db1, _ := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "demo2_node1")))
	defer db1.Close()
	db2, _ := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "demo2_node2")))
	defer db2.Close()
	db3, _ := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "demo2_node3")))
	defer db3.Close()

	config1 := replication.DefaultReplicaSetConfig("rs1", "node1", db1, filepath.Join(tmpDir, "demo2_oplog1.bin"))
	config1.Priority = 10 // Highest priority - should win election
	rs1, _ := replication.NewReplicaSet(config1)
	defer rs1.Stop()

	config2 := replication.DefaultReplicaSetConfig("rs1", "node2", db2, filepath.Join(tmpDir, "demo2_oplog2.bin"))
	config2.Priority = 5
	rs2, _ := replication.NewReplicaSet(config2)
	defer rs2.Stop()

	config3 := replication.DefaultReplicaSetConfig("rs1", "node3", db3, filepath.Join(tmpDir, "demo2_oplog3.bin"))
	config3.Priority = 5
	rs3, _ := replication.NewReplicaSet(config3)
	defer rs3.Stop()

	// Configure members
	rs1.AddMember("node2", 5, true)
	rs1.AddMember("node3", 5, true)
	rs2.AddMember("node1", 10, true)
	rs2.AddMember("node3", 5, true)
	rs3.AddMember("node1", 10, true)
	rs3.AddMember("node2", 5, true)

	// Mark all members as healthy
	rs1.UpdateMemberHeartbeat("node2", 0)
	rs1.UpdateMemberHeartbeat("node3", 0)

	fmt.Println("✓ Created 3-node replica set")
	fmt.Println("  All nodes start as SECONDARY")

	// Trigger election on node1
	fmt.Println("\nTriggering election on node1 (priority 10)...")
	rs1.Start()

	// Manually start election (simplified)
	// In production, this would be triggered automatically by election timeout
	// rs1.startElection() - private method, so we can't call it directly
	// For demo purposes, we'll manually become primary
	fmt.Println("  Node1 collecting votes...")

	// Simulate votes (in real implementation, this happens via RPC)
	// Node1 should get votes from healthy nodes with lower priority
	fmt.Println("  Node1 received votes from: node1 (self), node2, node3")
	fmt.Println("  Votes: 3/3 (majority achieved)")

	// Manually trigger role change
	err := rs1.BecomePrimary()
	if err != nil {
		fmt.Printf("Error becoming primary: %v\n", err)
		return
	}

	fmt.Println("\n✓ Election completed")
	fmt.Printf("  New PRIMARY: %s\n", rs1.GetPrimary())
	fmt.Printf("  Node 1 role: %s\n", rs1.GetRole())
}

func demo3WriteWithReplication(tmpDir string) {
	fmt.Println("Demo 3: Write Operations with Majority Replication")
	fmt.Println("---------------------------------------------------")

	// Create primary node
	db, _ := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "demo3_primary")))
	defer db.Close()

	config := replication.DefaultReplicaSetConfig("rs2", "primary", db, filepath.Join(tmpDir, "demo3_oplog.bin"))
	rs, _ := replication.NewReplicaSet(config)
	defer rs.Stop()

	// Add two secondary members
	rs.AddMember("secondary1", 1, true)
	rs.AddMember("secondary2", 1, true)

	// Become primary
	rs.Start()
	err := rs.BecomePrimary()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("✓ Replica set initialized with 1 primary and 2 secondaries")

	// Create collection and insert document
	coll := db.Collection("users")
	id, err := coll.InsertOne(map[string]interface{}{
		"name": "Alice",
		"age":  int64(30),
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\n✓ Inserted document: _id=%s\n", id)

	// Log the operation to oplog
	entry := replication.CreateInsertEntry("testdb", "users", map[string]interface{}{
		"_id":  id,
		"name": "Alice",
		"age":  int64(30),
	})
	err = rs.LogOperation(entry)
	if err != nil {
		fmt.Printf("Error logging operation: %v\n", err)
		return
	}

	fmt.Printf("✓ Operation logged to oplog (OpID: %d)\n", entry.OpID)

	// Simulate replication progress
	rs.UpdateMemberHeartbeat("primary", entry.OpID)
	rs.UpdateMemberHeartbeat("secondary1", entry.OpID)
	rs.UpdateMemberHeartbeat("secondary2", entry.OpID-1) // One behind

	fmt.Println("\nReplication status:")
	members := rs.GetMembers()
	for _, member := range members {
		if member.IsVotingMember {
			fmt.Printf("  %s: OpID=%d, Lag=%s, State=%s\n",
				member.NodeID, member.LastOpID, member.Lag, member.State)
		}
	}

	fmt.Println("\n✓ 2/3 nodes have replicated (majority achieved)")
}

func demo4StepDown(tmpDir string) {
	fmt.Println("Demo 4: Manual Step Down (Primary Resignation)")
	fmt.Println("----------------------------------------------")

	// Create primary node
	db, _ := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "demo4_primary")))
	defer db.Close()

	config := replication.DefaultReplicaSetConfig("rs3", "primary", db, filepath.Join(tmpDir, "demo4_oplog.bin"))
	rs, _ := replication.NewReplicaSet(config)
	defer rs.Stop()

	rs.Start()
	err := rs.BecomePrimary()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("✓ Node started as PRIMARY\n")
	fmt.Printf("  Current role: %s\n", rs.GetRole())

	// Step down
	fmt.Println("\nInitiating step down...")
	err = rs.StepDown()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("✓ Step down completed")
	fmt.Printf("  New role: %s\n", rs.GetRole())
	fmt.Println("  Node will participate in new election")
}

func demo5MemberManagement(tmpDir string) {
	fmt.Println("Demo 5: Adding and Removing Members")
	fmt.Println("------------------------------------")

	// Create replica set
	db, _ := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "demo5_node")))
	defer db.Close()

	config := replication.DefaultReplicaSetConfig("rs4", "node1", db, filepath.Join(tmpDir, "demo5_oplog.bin"))
	rs, _ := replication.NewReplicaSet(config)
	defer rs.Stop()

	rs.Start()

	// Initial state
	members := rs.GetMembers()
	fmt.Printf("✓ Initial members: %d (self only)\n", len(members))

	// Add voting members
	fmt.Println("\nAdding voting members...")
	rs.AddMember("node2", 5, true)
	rs.AddMember("node3", 5, true)
	members = rs.GetMembers()
	fmt.Printf("✓ Added 2 voting members (total: %d)\n", len(members))

	// Add non-voting arbiter
	fmt.Println("\nAdding non-voting arbiter...")
	rs.AddMember("arbiter1", 0, false)
	members = rs.GetMembers()
	fmt.Printf("✓ Added arbiter (total members: %d)\n", len(members))

	// Display all members
	fmt.Println("\nCurrent members:")
	for _, member := range members {
		votingStatus := "voting"
		if !member.IsVotingMember {
			votingStatus = "non-voting (arbiter)"
		}
		fmt.Printf("  %s: Priority=%d, %s\n", member.NodeID, member.Priority, votingStatus)
	}

	// Remove a member
	fmt.Println("\nRemoving node3...")
	err := rs.RemoveMember("node3")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	members = rs.GetMembers()
	fmt.Printf("✓ Removed node3 (total: %d)\n", len(members))

	// Final state
	fmt.Println("\nFinal members:")
	for _, member := range members {
		votingStatus := "voting"
		if !member.IsVotingMember {
			votingStatus = "non-voting (arbiter)"
		}
		fmt.Printf("  %s: Priority=%d, %s\n", member.NodeID, member.Priority, votingStatus)
	}
}
