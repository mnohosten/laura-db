package replication

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mnohosten/laura-db/pkg/database"
)

func TestReplicaSetBasic(t *testing.T) {
	tmpDir := t.TempDir()

	// Create database
	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "rs1")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create replica set
	config := DefaultReplicaSetConfig("rs0", "node1", db, filepath.Join(tmpDir, "oplog.bin"))
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}
	defer rs.Stop()

	// Start replica set
	if err := rs.Start(); err != nil {
		t.Fatalf("Failed to start replica set: %v", err)
	}

	// Check initial state
	if rs.GetRole() != RoleSecondary {
		t.Errorf("Expected role SECONDARY, got %s", rs.GetRole())
	}

	// Check stats
	stats := rs.Stats()
	if stats["node_id"] != "node1" {
		t.Errorf("Expected node_id node1, got %v", stats["node_id"])
	}
}

func TestReplicaSetAddRemoveMember(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "rs1")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := DefaultReplicaSetConfig("rs0", "node1", db, filepath.Join(tmpDir, "oplog.bin"))
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}
	defer rs.Stop()

	// Add member
	if err := rs.AddMember("node2", 1, true); err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	// Check member exists
	members := rs.GetMembers()
	if len(members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(members))
	}

	// Try to add duplicate
	if err := rs.AddMember("node2", 1, true); err == nil {
		t.Error("Expected error adding duplicate member")
	}

	// Remove member
	if err := rs.RemoveMember("node2"); err != nil {
		t.Fatalf("Failed to remove member: %v", err)
	}

	// Check member removed
	members = rs.GetMembers()
	if len(members) != 1 {
		t.Errorf("Expected 1 member, got %d", len(members))
	}

	// Try to remove self
	if err := rs.RemoveMember("node1"); err == nil {
		t.Error("Expected error removing self")
	}
}

func TestReplicaSetUpdateHeartbeat(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "rs1")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := DefaultReplicaSetConfig("rs0", "node1", db, filepath.Join(tmpDir, "oplog.bin"))
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}
	defer rs.Stop()

	// Add member
	if err := rs.AddMember("node2", 1, true); err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	// Update heartbeat
	if err := rs.UpdateMemberHeartbeat("node2", OpID(10)); err != nil {
		t.Fatalf("Failed to update heartbeat: %v", err)
	}

	// Check member state
	members := rs.GetMembers()
	var node2 *ReplicaSetMember
	for _, m := range members {
		if m.NodeID == "node2" {
			node2 = m
			break
		}
	}

	if node2 == nil {
		t.Fatal("node2 not found")
	}

	if node2.State != StateHealthy {
		t.Errorf("Expected state HEALTHY, got %s", node2.State)
	}

	if node2.LastOpID != 10 {
		t.Errorf("Expected LastOpID 10, got %d", node2.LastOpID)
	}
}

func TestReplicaSetBecomePrimary(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "rs1")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := DefaultReplicaSetConfig("rs0", "node1", db, filepath.Join(tmpDir, "oplog.bin"))
	config.Priority = 10 // High priority
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}
	defer rs.Stop()

	if err := rs.Start(); err != nil {
		t.Fatalf("Failed to start replica set: %v", err)
	}

	// Manually become primary
	if err := rs.BecomePrimary(); err != nil {
		t.Fatalf("Failed to become primary: %v", err)
	}

	// Check role
	if !rs.IsPrimary() {
		t.Error("Expected node to be primary")
	}

	// Check primary ID
	if rs.GetPrimary() != "node1" {
		t.Errorf("Expected primary node1, got %s", rs.GetPrimary())
	}

	// Try to log operation
	entry := CreateInsertEntry("testdb", "users", map[string]interface{}{
		"name": "Alice",
	})

	if err := rs.LogOperation(entry); err != nil {
		t.Fatalf("Failed to log operation: %v", err)
	}
}

func TestReplicaSetBecomeSecondary(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "rs1")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := DefaultReplicaSetConfig("rs0", "node1", db, filepath.Join(tmpDir, "oplog.bin"))
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}
	defer rs.Stop()

	// First become primary
	if err := rs.BecomePrimary(); err != nil {
		t.Fatalf("Failed to become primary: %v", err)
	}

	// Then become secondary
	if err := rs.becomeSecondary("node2"); err != nil {
		t.Fatalf("Failed to become secondary: %v", err)
	}

	// Check role
	if rs.IsPrimary() {
		t.Error("Expected node to be secondary")
	}

	if rs.GetPrimary() != "node2" {
		t.Errorf("Expected primary node2, got %s", rs.GetPrimary())
	}
}

func TestReplicaSetStepDown(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "rs1")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := DefaultReplicaSetConfig("rs0", "node1", db, filepath.Join(tmpDir, "oplog.bin"))
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}
	defer rs.Stop()

	// Try to step down as secondary (should fail)
	if err := rs.StepDown(); err == nil {
		t.Error("Expected error stepping down as secondary")
	}

	// Become primary
	if err := rs.BecomePrimary(); err != nil {
		t.Fatalf("Failed to become primary: %v", err)
	}

	// Step down
	if err := rs.StepDown(); err != nil {
		t.Fatalf("Failed to step down: %v", err)
	}

	// Check role
	if rs.IsPrimary() {
		t.Error("Expected node to be secondary after step down")
	}
}

func TestReplicaSetElection(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "rs1")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := DefaultReplicaSetConfig("rs0", "node1", db, filepath.Join(tmpDir, "oplog.bin"))
	config.Priority = 10 // High priority
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}
	defer rs.Stop()

	// Add voting members with lower priority
	if err := rs.AddMember("node2", 1, true); err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}
	if err := rs.AddMember("node3", 1, true); err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	// Mark members as healthy
	if err := rs.UpdateMemberHeartbeat("node2", 0); err != nil {
		t.Fatalf("Failed to update heartbeat: %v", err)
	}
	if err := rs.UpdateMemberHeartbeat("node3", 0); err != nil {
		t.Fatalf("Failed to update heartbeat: %v", err)
	}

	// Start election
	rs.startElection()

	// Should win with 3 votes (self + 2 lower priority members)
	if !rs.IsPrimary() {
		t.Error("Expected node to become primary after winning election")
	}

	// Check term incremented
	if rs.currentTerm != 1 {
		t.Errorf("Expected term 1, got %d", rs.currentTerm)
	}
}

func TestReplicaSetElectionFailure(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "rs1")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := DefaultReplicaSetConfig("rs0", "node1", db, filepath.Join(tmpDir, "oplog.bin"))
	config.Priority = 1 // Low priority
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}
	defer rs.Stop()

	// Add voting members with higher priority
	if err := rs.AddMember("node2", 10, true); err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}
	if err := rs.AddMember("node3", 10, true); err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	// Mark members as unhealthy (won't vote)
	if err := rs.SimulateFailure("node2"); err != nil {
		t.Fatalf("Failed to simulate failure: %v", err)
	}
	if err := rs.SimulateFailure("node3"); err != nil {
		t.Fatalf("Failed to simulate failure: %v", err)
	}

	// Start election
	rs.startElection()

	// Should not win (only 1 vote, need 2)
	if rs.IsPrimary() {
		t.Error("Expected node to remain secondary after losing election")
	}
}

func TestReplicaSetAutomaticFailover(t *testing.T) {
	t.Skip("Skipping: 2-node automatic failover conflicts with standard Raft majority requirements. " +
		"In production, use 3+ nodes or an arbiter for automatic failover.")

	tmpDir := t.TempDir()

	// Create node1 (will be initial primary)
	db1, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "rs1")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db1.Close()

	config1 := DefaultReplicaSetConfig("rs0", "node1", db1, filepath.Join(tmpDir, "oplog1.bin"))
	config1.Priority = 10
	config1.ElectionTimeout = 2 * time.Second // Short timeout for testing
	rs1, err := NewReplicaSet(config1)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}
	defer rs1.Stop()

	// Create node2 (will be secondary, then failover to primary)
	db2, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "rs2")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db2.Close()

	config2 := DefaultReplicaSetConfig("rs0", "node2", db2, filepath.Join(tmpDir, "oplog2.bin"))
	config2.Priority = 9
	config2.ElectionTimeout = 2 * time.Second
	rs2, err := NewReplicaSet(config2)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}
	defer rs2.Stop()

	// Add node2 to node1's member list
	if err := rs1.AddMember("node2", 9, true); err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	// Add node1 to node2's member list
	if err := rs2.AddMember("node1", 10, true); err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	// Start both nodes
	if err := rs1.Start(); err != nil {
		t.Fatalf("Failed to start rs1: %v", err)
	}
	if err := rs2.Start(); err != nil {
		t.Fatalf("Failed to start rs2: %v", err)
	}

	// Node1 becomes primary
	if err := rs1.BecomePrimary(); err != nil {
		t.Fatalf("Failed for node1 to become primary: %v", err)
	}

	// Node2 becomes secondary
	if err := rs2.becomeSecondary("node1"); err != nil {
		t.Fatalf("Failed for node2 to become secondary: %v", err)
	}

	// Verify initial state
	if !rs1.IsPrimary() {
		t.Error("Expected node1 to be primary")
	}
	if rs2.IsPrimary() {
		t.Error("Expected node2 to be secondary")
	}

	// Simulate node1 failure
	if err := rs2.SimulateFailure("node1"); err != nil {
		t.Fatalf("Failed to simulate failure: %v", err)
	}

	// Wait for election timeout
	time.Sleep(3 * time.Second)

	// Node2 should have started election and become primary
	// (In simplified implementation, needs manual trigger)
	rs2.startElection()

	// Check if node2 became primary
	if !rs2.IsPrimary() {
		t.Error("Expected node2 to become primary after failover")
	}
}

func TestReplicaSetWaitForReplication(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "rs1")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := DefaultReplicaSetConfig("rs0", "node1", db, filepath.Join(tmpDir, "oplog.bin"))
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}
	defer rs.Stop()

	// Add members
	if err := rs.AddMember("node2", 1, true); err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}
	if err := rs.AddMember("node3", 1, true); err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	// Update heartbeats to show replication progress
	if err := rs.UpdateMemberHeartbeat("node1", OpID(10)); err != nil {
		t.Fatalf("Failed to update heartbeat: %v", err)
	}
	if err := rs.UpdateMemberHeartbeat("node2", OpID(10)); err != nil {
		t.Fatalf("Failed to update heartbeat: %v", err)
	}

	// Wait for replication (should succeed immediately since 2/3 have opID 10)
	ctx := context.Background()
	if err := rs.WaitForReplication(ctx, OpID(10), 1*time.Second); err != nil {
		t.Errorf("WaitForReplication failed: %v", err)
	}
}

func TestReplicaSetWaitForReplicationTimeout(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "rs1")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := DefaultReplicaSetConfig("rs0", "node1", db, filepath.Join(tmpDir, "oplog.bin"))
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}
	defer rs.Stop()

	// Add members
	if err := rs.AddMember("node2", 1, true); err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}
	if err := rs.AddMember("node3", 1, true); err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	// Only 1 member has replicated (not a majority)
	if err := rs.UpdateMemberHeartbeat("node1", OpID(10)); err != nil {
		t.Fatalf("Failed to update heartbeat: %v", err)
	}

	// Wait for replication (should timeout)
	ctx := context.Background()
	if err := rs.WaitForReplication(ctx, OpID(10), 500*time.Millisecond); err == nil {
		t.Error("Expected timeout error")
	}
}

func TestReplicaSetSimulateFailure(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "rs1")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := DefaultReplicaSetConfig("rs0", "node1", db, filepath.Join(tmpDir, "oplog.bin"))
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}
	defer rs.Stop()

	// Add member
	if err := rs.AddMember("node2", 1, true); err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	// Simulate failure
	if err := rs.SimulateFailure("node2"); err != nil {
		t.Fatalf("Failed to simulate failure: %v", err)
	}

	// Check member state
	members := rs.GetMembers()
	var node2 *ReplicaSetMember
	for _, m := range members {
		if m.NodeID == "node2" {
			node2 = m
			break
		}
	}

	if node2 == nil {
		t.Fatal("node2 not found")
	}

	if node2.State != StateUnreachable {
		t.Errorf("Expected state UNREACHABLE, got %s", node2.State)
	}
}

func TestReplicaSetVotingMemberCount(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "rs1")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := DefaultReplicaSetConfig("rs0", "node1", db, filepath.Join(tmpDir, "oplog.bin"))
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}
	defer rs.Stop()

	// Add voting and non-voting members
	if err := rs.AddMember("node2", 1, true); err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}
	if err := rs.AddMember("node3", 1, false); err != nil { // Non-voting
		t.Fatalf("Failed to add member: %v", err)
	}

	// Count voting members
	count := rs.countVotingMembers()
	if count != 2 {
		t.Errorf("Expected 2 voting members, got %d", count)
	}
}
