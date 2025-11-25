package replication

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mnohosten/laura-db/pkg/database"
)

func TestMasterBasic(t *testing.T) {
	tmpDir := t.TempDir()

	// Create database
	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "master")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create master
	config := DefaultMasterConfig(db, filepath.Join(tmpDir, "oplog.bin"))
	master, err := NewMaster(config)
	if err != nil {
		t.Fatalf("Failed to create master: %v", err)
	}
	defer master.Stop()

	// Start master
	if err := master.Start(); err != nil {
		t.Fatalf("Failed to start master: %v", err)
	}

	// Log an operation
	entry := CreateInsertEntry("testdb", "users", map[string]interface{}{
		"name": "Alice",
		"age":  int64(30),
	})

	if err := master.LogOperation(entry); err != nil {
		t.Fatalf("Failed to log operation: %v", err)
	}

	// Check current OpID
	if master.GetCurrentOpID() != 1 {
		t.Errorf("Expected OpID 1, got %d", master.GetCurrentOpID())
	}
}

func TestMasterSlaveRegistration(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "master")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := DefaultMasterConfig(db, filepath.Join(tmpDir, "oplog.bin"))
	master, err := NewMaster(config)
	if err != nil {
		t.Fatalf("Failed to create master: %v", err)
	}
	defer master.Stop()

	if err := master.Start(); err != nil {
		t.Fatalf("Failed to start master: %v", err)
	}

	// Register slave
	if err := master.RegisterSlave("slave1"); err != nil {
		t.Fatalf("Failed to register slave: %v", err)
	}

	// Check slave info
	info, err := master.GetSlaveInfo("slave1")
	if err != nil {
		t.Fatalf("Failed to get slave info: %v", err)
	}

	if info.ID != "slave1" {
		t.Errorf("Expected slave ID slave1, got %s", info.ID)
	}

	// Register duplicate should fail
	if err := master.RegisterSlave("slave1"); err == nil {
		t.Error("Expected error registering duplicate slave")
	}

	// Unregister slave
	if err := master.UnregisterSlave("slave1"); err != nil {
		t.Fatalf("Failed to unregister slave: %v", err)
	}

	// Get slave info should fail
	if _, err := master.GetSlaveInfo("slave1"); err == nil {
		t.Error("Expected error getting unregistered slave")
	}
}

func TestMasterHeartbeat(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "master")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := DefaultMasterConfig(db, filepath.Join(tmpDir, "oplog.bin"))
	config.HeartbeatTimeout = 500 * time.Millisecond // Short timeout for testing
	master, err := NewMaster(config)
	if err != nil {
		t.Fatalf("Failed to create master: %v", err)
	}
	defer master.Stop()

	// Manually set up heartbeat with faster ticker for testing
	master.mu.Lock()
	master.isRunning = true
	master.heartbeatTicker = time.NewTicker(200 * time.Millisecond) // Check every 200ms
	go master.monitorHeartbeats()
	master.mu.Unlock()

	// Register slave
	if err := master.RegisterSlave("slave1"); err != nil {
		t.Fatalf("Failed to register slave: %v", err)
	}

	// Update heartbeat
	if err := master.UpdateSlaveHeartbeat("slave1", 100); err != nil {
		t.Fatalf("Failed to update heartbeat: %v", err)
	}

	// Check slave info
	info, err := master.GetSlaveInfo("slave1")
	if err != nil {
		t.Fatalf("Failed to get slave info: %v", err)
	}

	if info.LastOpID != 100 {
		t.Errorf("Expected LastOpID 100, got %d", info.LastOpID)
	}

	// Wait for heartbeat timeout (timeout is 500ms, check every 200ms, so wait 1s to be safe)
	time.Sleep(1 * time.Second)

	// Slave should be removed
	if _, err := master.GetSlaveInfo("slave1"); err == nil {
		t.Error("Expected slave to be removed after timeout")
	}
}

func TestMasterGetOplogEntries(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "master")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := DefaultMasterConfig(db, filepath.Join(tmpDir, "oplog.bin"))
	master, err := NewMaster(config)
	if err != nil {
		t.Fatalf("Failed to create master: %v", err)
	}
	defer master.Stop()

	if err := master.Start(); err != nil {
		t.Fatalf("Failed to start master: %v", err)
	}

	// Log multiple operations
	for i := 0; i < 10; i++ {
		entry := CreateInsertEntry("testdb", "users", map[string]interface{}{
			"index": int64(i),
		})
		if err := master.LogOperation(entry); err != nil {
			t.Fatalf("Failed to log operation: %v", err)
		}
	}

	// Get entries since OpID 5
	entries, err := master.GetOplogEntries(5)
	if err != nil {
		t.Fatalf("Failed to get entries: %v", err)
	}

	if len(entries) != 5 {
		t.Errorf("Expected 5 entries, got %d", len(entries))
	}

	for i, entry := range entries {
		expectedID := OpID(6 + i)
		if entry.OpID != expectedID {
			t.Errorf("Expected OpID %d, got %d", expectedID, entry.OpID)
		}
	}
}

func TestReplicationPairBasic(t *testing.T) {
	tmpDir := t.TempDir()

	// Create master database
	masterDB, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "master")))
	if err != nil {
		t.Fatalf("Failed to open master database: %v", err)
	}
	defer masterDB.Close()

	// Create slave database
	slaveDB, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "slave")))
	if err != nil {
		t.Fatalf("Failed to open slave database: %v", err)
	}
	defer slaveDB.Close()

	// Create replication pair
	masterConfig := DefaultMasterConfig(masterDB, filepath.Join(tmpDir, "oplog.bin"))
	slaveConfig := DefaultSlaveConfig("slave1", slaveDB, nil)

	pair, err := NewReplicationPair(masterConfig, slaveConfig)
	if err != nil {
		t.Fatalf("Failed to create replication pair: %v", err)
	}
	defer pair.Stop()

	// Start replication
	if err := pair.Start(); err != nil {
		t.Fatalf("Failed to start replication: %v", err)
	}

	// Create document with ID
	doc := map[string]interface{}{
		"_id":  "user1",
		"name": "Alice",
		"age":  int64(30),
	}

	// Insert document on master
	coll := masterDB.Collection("users")
	id, err := coll.InsertOne(doc)
	if err != nil {
		t.Fatalf("Failed to insert on master: %v", err)
	}

	// Log to oplog (use same doc to preserve _id)
	entry := CreateInsertEntry("default", "users", doc)
	if err := pair.Master.LogOperation(entry); err != nil {
		t.Fatalf("Failed to log operation: %v", err)
	}

	// Wait for replication
	time.Sleep(2 * time.Second)

	// Check slave has the document
	slaveColl := slaveDB.Collection("users")
	result, err := slaveColl.FindOne(nil)
	if err != nil {
		t.Fatalf("Failed to find on slave: %v", err)
	}

	resultMap := result.ToMap()
	if resultMap["_id"] != id {
		t.Errorf("Document ID mismatch: expected %v, got %v", id, resultMap["_id"])
	}

	if resultMap["name"] != "Alice" {
		t.Errorf("Name mismatch: expected Alice, got %v", resultMap["name"])
	}
}

func TestReplicationInsertUpdateDelete(t *testing.T) {
	tmpDir := t.TempDir()

	// Create databases
	masterDB, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "master")))
	if err != nil {
		t.Fatalf("Failed to open master database: %v", err)
	}
	defer masterDB.Close()

	slaveDB, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "slave")))
	if err != nil {
		t.Fatalf("Failed to open slave database: %v", err)
	}
	defer slaveDB.Close()

	// Create replication pair
	masterConfig := DefaultMasterConfig(masterDB, filepath.Join(tmpDir, "oplog.bin"))
	slaveConfig := DefaultSlaveConfig("slave1", slaveDB, nil)
	slaveConfig.PollInterval = 100 * time.Millisecond

	pair, err := NewReplicationPair(masterConfig, slaveConfig)
	if err != nil {
		t.Fatalf("Failed to create replication pair: %v", err)
	}
	defer pair.Stop()

	if err := pair.Start(); err != nil {
		t.Fatalf("Failed to start replication: %v", err)
	}

	// Insert
	doc := map[string]interface{}{
		"_id":  "user1",
		"name": "Alice",
		"age":  int64(30),
	}
	masterDB.Collection("users").InsertOne(doc)
	pair.Master.LogOperation(CreateInsertEntry("default", "users", doc))

	// Wait for replication
	time.Sleep(500 * time.Millisecond)

	// Verify insert on slave
	result, err := slaveDB.Collection("users").FindOne(nil)
	if err != nil {
		t.Fatalf("Failed to find on slave after insert: %v", err)
	}
	if result.ToMap()["name"] != "Alice" {
		t.Error("Insert not replicated correctly")
	}

	// Update
	filter := map[string]interface{}{"_id": "user1"}
	update := map[string]interface{}{"$set": map[string]interface{}{"age": int64(31)}}
	masterDB.Collection("users").UpdateOne(filter, update)
	pair.Master.LogOperation(CreateUpdateEntry("default", "users", filter, update))

	// Wait for replication
	time.Sleep(500 * time.Millisecond)

	// Verify update on slave
	result, err = slaveDB.Collection("users").FindOne(nil)
	if err != nil {
		t.Fatalf("Failed to find on slave after update: %v", err)
	}
	if result.ToMap()["age"] != int64(31) {
		t.Error("Update not replicated correctly")
	}

	// Delete
	masterDB.Collection("users").DeleteOne(filter)
	pair.Master.LogOperation(CreateDeleteEntry("default", "users", filter))

	// Wait for replication
	time.Sleep(500 * time.Millisecond)

	// Verify delete on slave
	count, err := slaveDB.Collection("users").Count(nil)
	if err != nil {
		t.Fatalf("Failed to count on slave: %v", err)
	}
	if count != 0 {
		t.Error("Delete not replicated correctly")
	}
}

func TestSlaveInitialSync(t *testing.T) {
	tmpDir := t.TempDir()

	// Create master with existing data
	masterDB, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "master")))
	if err != nil {
		t.Fatalf("Failed to open master database: %v", err)
	}
	defer masterDB.Close()

	masterConfig := DefaultMasterConfig(masterDB, filepath.Join(tmpDir, "oplog.bin"))
	master, err := NewMaster(masterConfig)
	if err != nil {
		t.Fatalf("Failed to create master: %v", err)
	}
	defer master.Stop()

	master.Start()

	// Insert data and log to oplog
	for i := 0; i < 10; i++ {
		doc := map[string]interface{}{
			"index": int64(i),
			"name":  "User " + string(rune('A'+i)),
		}
		masterDB.Collection("users").InsertOne(doc)
		master.LogOperation(CreateInsertEntry("default", "users", doc))
	}

	// Create slave
	slaveDB, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "slave")))
	if err != nil {
		t.Fatalf("Failed to open slave database: %v", err)
	}
	defer slaveDB.Close()

	client := NewLocalMasterClient(master)
	slaveConfig := DefaultSlaveConfig("slave1", slaveDB, client)
	slave, err := NewSlave(slaveConfig)
	if err != nil {
		t.Fatalf("Failed to create slave: %v", err)
	}
	defer slave.Stop()

	// Perform initial sync
	ctx := context.Background()
	if err := slave.InitialSync(ctx); err != nil {
		t.Fatalf("Initial sync failed: %v", err)
	}

	// Verify all documents are on slave
	count, err := slaveDB.Collection("users").Count(nil)
	if err != nil {
		t.Fatalf("Failed to count on slave: %v", err)
	}
	if count != 10 {
		t.Errorf("Expected 10 documents on slave, got %d", count)
	}

	// Verify last applied OpID
	if slave.GetLastAppliedOpID() != 10 {
		t.Errorf("Expected last applied OpID 10, got %d", slave.GetLastAppliedOpID())
	}
}

func TestSlaveLagCalculation(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "slave")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create mock client
	client := &mockMasterClient{
		entries: make([]*OplogEntry, 0),
	}

	config := DefaultSlaveConfig("slave1", db, client)
	slave, err := NewSlave(config)
	if err != nil {
		t.Fatalf("Failed to create slave: %v", err)
	}

	// Set last applied OpID
	slave.mu.Lock()
	slave.lastAppliedOpID = 100
	slave.mu.Unlock()

	// Calculate lag
	lag := slave.GetLag(150)
	expectedLag := 50 * time.Millisecond

	if lag != expectedLag {
		t.Errorf("Expected lag %v, got %v", expectedLag, lag)
	}

	// No lag
	lag = slave.GetLag(100)
	if lag != 0 {
		t.Errorf("Expected no lag, got %v", lag)
	}
}

// mockMasterClient for testing
type mockMasterClient struct {
	entries []*OplogEntry
	mu      sync.Mutex
}

func (m *mockMasterClient) GetOplogEntries(ctx context.Context, sinceID OpID) ([]*OplogEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]*OplogEntry, 0)
	for _, entry := range m.entries {
		if entry.OpID > sinceID {
			result = append(result, entry)
		}
	}
	return result, nil
}

func (m *mockMasterClient) SendHeartbeat(ctx context.Context, slaveID string, lastOpID OpID) error {
	return nil
}

func (m *mockMasterClient) Register(ctx context.Context, slaveID string) error {
	return nil
}

func (m *mockMasterClient) Unregister(ctx context.Context, slaveID string) error {
	return nil
}

func BenchmarkMasterLogOperation(b *testing.B) {
	tmpDir := b.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "master")))
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := DefaultMasterConfig(db, filepath.Join(tmpDir, "oplog.bin"))
	master, err := NewMaster(config)
	if err != nil {
		b.Fatalf("Failed to create master: %v", err)
	}
	defer master.Stop()

	master.Start()

	entry := CreateInsertEntry("testdb", "users", map[string]interface{}{
		"name": "Alice",
		"age":  int64(30),
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := master.LogOperation(entry); err != nil {
			b.Fatalf("Failed to log operation: %v", err)
		}
	}
}

func BenchmarkSlaveApplyEntry(b *testing.B) {
	tmpDir := b.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "slave")))
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	client := &mockMasterClient{entries: make([]*OplogEntry, 0)}
	config := DefaultSlaveConfig("slave1", db, client)
	slave, err := NewSlave(config)
	if err != nil {
		b.Fatalf("Failed to create slave: %v", err)
	}

	entry := &OplogEntry{
		OpID:       1,
		OpType:     OpTypeInsert,
		Database:   "testdb",
		Collection: "users",
		Document: map[string]interface{}{
			"_id":  "user1",
			"name": "Alice",
			"age":  int64(30),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry.Document["_id"] = "user" + string(rune(i))
		if err := slave.applyEntry(entry); err != nil {
			b.Fatalf("Failed to apply entry: %v", err)
		}
	}
}

// TestMasterGetAllSlaves tests the GetAllSlaves function
func TestMasterGetAllSlaves(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "master")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := DefaultMasterConfig(db, filepath.Join(tmpDir, "oplog.bin"))
	master, err := NewMaster(config)
	if err != nil {
		t.Fatalf("Failed to create master: %v", err)
	}
	defer master.Stop()

	if err := master.Start(); err != nil {
		t.Fatalf("Failed to start master: %v", err)
	}

	// Initially no slaves
	slaves := master.GetAllSlaves()
	if len(slaves) != 0 {
		t.Errorf("Expected no slaves, got %d", len(slaves))
	}

	// Register multiple slaves
	for i := 1; i <= 3; i++ {
		slaveID := fmt.Sprintf("slave%d", i)
		if err := master.RegisterSlave(slaveID); err != nil {
			t.Fatalf("Failed to register slave: %v", err)
		}
		// Update heartbeat with different OpIDs
		if err := master.UpdateSlaveHeartbeat(slaveID, OpID(i*10)); err != nil {
			t.Fatalf("Failed to update heartbeat: %v", err)
		}
	}

	// Get all slaves
	slaves = master.GetAllSlaves()
	if len(slaves) != 3 {
		t.Errorf("Expected 3 slaves, got %d", len(slaves))
	}

	// Verify slave IDs and OpIDs
	slaveMap := make(map[string]*SlaveInfo)
	for _, slave := range slaves {
		slaveMap[slave.ID] = slave
	}

	for i := 1; i <= 3; i++ {
		slaveID := fmt.Sprintf("slave%d", i)
		if info, ok := slaveMap[slaveID]; !ok {
			t.Errorf("Slave %s not found", slaveID)
		} else if info.LastOpID != OpID(i*10) {
			t.Errorf("Expected LastOpID %d for %s, got %d", i*10, slaveID, info.LastOpID)
		}
	}
}

// TestMasterStats tests the Stats function
func TestMasterStats(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "master")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := DefaultMasterConfig(db, filepath.Join(tmpDir, "oplog.bin"))
	master, err := NewMaster(config)
	if err != nil {
		t.Fatalf("Failed to create master: %v", err)
	}
	defer master.Stop()

	if err := master.Start(); err != nil {
		t.Fatalf("Failed to start master: %v", err)
	}

	// Register slaves
	master.RegisterSlave("slave1")
	master.RegisterSlave("slave2")
	master.UpdateSlaveHeartbeat("slave1", 100)
	master.UpdateSlaveHeartbeat("slave2", 200)

	// Log operations
	for i := 0; i < 5; i++ {
		entry := CreateInsertEntry("testdb", "users", map[string]interface{}{
			"index": int64(i),
		})
		master.LogOperation(entry)
	}

	// Get stats
	stats := master.Stats()

	// Verify stats structure
	if currentOpID, ok := stats["current_op_id"].(OpID); !ok || currentOpID != 5 {
		t.Errorf("Expected current_op_id 5, got %v", stats["current_op_id"])
	}

	if slaveCount, ok := stats["slave_count"].(int); !ok || slaveCount != 2 {
		t.Errorf("Expected slave_count 2, got %v", stats["slave_count"])
	}

	if isRunning, ok := stats["is_running"].(bool); !ok || !isRunning {
		t.Errorf("Expected is_running true, got %v", stats["is_running"])
	}

	if slaves, ok := stats["slaves"].([]map[string]interface{}); !ok || len(slaves) != 2 {
		t.Errorf("Expected 2 slaves in stats, got %v", stats["slaves"])
	}
}

// TestMasterFlush tests the Flush function
func TestMasterFlush(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "master")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := DefaultMasterConfig(db, filepath.Join(tmpDir, "oplog.bin"))
	master, err := NewMaster(config)
	if err != nil {
		t.Fatalf("Failed to create master: %v", err)
	}
	defer master.Stop()

	if err := master.Start(); err != nil {
		t.Fatalf("Failed to start master: %v", err)
	}

	// Log operations
	for i := 0; i < 10; i++ {
		entry := CreateInsertEntry("testdb", "users", map[string]interface{}{
			"index": int64(i),
		})
		if err := master.LogOperation(entry); err != nil {
			t.Fatalf("Failed to log operation: %v", err)
		}
	}

	// Flush oplog to disk
	if err := master.Flush(); err != nil {
		t.Fatalf("Failed to flush: %v", err)
	}

	// Verify oplog file exists and has content
	oplogPath := filepath.Join(tmpDir, "oplog.bin")
	stat, err := os.Stat(oplogPath)
	if err != nil {
		t.Fatalf("Oplog file not found: %v", err)
	}

	if stat.Size() == 0 {
		t.Error("Oplog file is empty after flush")
	}
}

// TestMasterWaitForSlaves tests the WaitForSlaves function
func TestMasterWaitForSlaves(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "master")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := DefaultMasterConfig(db, filepath.Join(tmpDir, "oplog.bin"))
	master, err := NewMaster(config)
	if err != nil {
		t.Fatalf("Failed to create master: %v", err)
	}
	defer master.Stop()

	if err := master.Start(); err != nil {
		t.Fatalf("Failed to start master: %v", err)
	}

	// Register slaves
	master.RegisterSlave("slave1")
	master.RegisterSlave("slave2")

	// Update slaves to OpID 100
	master.UpdateSlaveHeartbeat("slave1", 100)
	master.UpdateSlaveHeartbeat("slave2", 100)

	// Wait for slaves should succeed immediately
	ctx := context.Background()
	err = master.WaitForSlaves(ctx, 100, 1*time.Second)
	if err != nil {
		t.Errorf("WaitForSlaves failed: %v", err)
	}

	// Wait for higher OpID should timeout
	ctx2 := context.Background()
	err = master.WaitForSlaves(ctx2, 200, 100*time.Millisecond)
	if err == nil {
		t.Error("Expected timeout error")
	}

	// Test context cancellation
	ctx3, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	err = master.WaitForSlaves(ctx3, 200, 1*time.Second)
	if err == nil {
		t.Error("Expected context cancellation error")
	}
}

// TestOpTypeString tests the String method of OpType
func TestOpTypeString(t *testing.T) {
	tests := []struct {
		opType   OpType
		expected string
	}{
		{OpTypeInsert, "insert"},
		{OpTypeUpdate, "update"},
		{OpTypeDelete, "delete"},
		{OpTypeCreateCollection, "createCollection"},
		{OpTypeDropCollection, "dropCollection"},
		{OpTypeCreateIndex, "createIndex"},
		{OpTypeDropIndex, "dropIndex"},
		{OpTypeNoop, "noop"},
		{OpType(99), "unknown"}, // Invalid op type
	}

	for _, tt := range tests {
		result := tt.opType.String()
		if result != tt.expected {
			t.Errorf("OpType(%d).String() = %s, expected %s", tt.opType, result, tt.expected)
		}
	}
}

// TestSlaveIsRunning tests the IsRunning function
func TestSlaveIsRunning(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "slave")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	client := &mockMasterClient{entries: make([]*OplogEntry, 0)}
	config := DefaultSlaveConfig("slave1", db, client)
	slave, err := NewSlave(config)
	if err != nil {
		t.Fatalf("Failed to create slave: %v", err)
	}

	// Initially not running
	if slave.IsRunning() {
		t.Error("Expected slave to not be running initially")
	}

	// Start slave
	if err := slave.Start(); err != nil {
		t.Fatalf("Failed to start slave: %v", err)
	}

	// Should be running
	if !slave.IsRunning() {
		t.Error("Expected slave to be running after Start")
	}

	// Stop slave
	slave.Stop()

	// Give it time to stop
	time.Sleep(100 * time.Millisecond)

	// Should not be running
	if slave.IsRunning() {
		t.Error("Expected slave to not be running after Stop")
	}
}

// TestSlaveStats tests the Stats function
func TestSlaveStats(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "slave")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	client := &mockMasterClient{entries: make([]*OplogEntry, 0)}
	config := DefaultSlaveConfig("slave1", db, client)
	slave, err := NewSlave(config)
	if err != nil {
		t.Fatalf("Failed to create slave: %v", err)
	}

	// Set some state
	slave.mu.Lock()
	slave.lastAppliedOpID = 150
	slave.replicationErrors = 3
	slave.mu.Unlock()

	// Get stats
	stats := slave.Stats()

	// Verify stats
	if slaveID, ok := stats["slave_id"].(string); !ok || slaveID != "slave1" {
		t.Errorf("Expected slave_id 'slave1', got %v", stats["slave_id"])
	}

	if lastOpID, ok := stats["last_applied_op_id"].(OpID); !ok || lastOpID != 150 {
		t.Errorf("Expected last_applied_op_id 150, got %v", stats["last_applied_op_id"])
	}

	if errors, ok := stats["replication_errors"].(int); !ok || errors != 3 {
		t.Errorf("Expected replication_errors 3, got %v", stats["replication_errors"])
	}

	if isRunning, ok := stats["is_running"].(bool); !ok || isRunning {
		t.Errorf("Expected is_running false, got %v", stats["is_running"])
	}
}

// TestSlaveReadDocument tests the ReadDocument function
func TestSlaveReadDocument(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "slave")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	client := &mockMasterClient{entries: make([]*OplogEntry, 0)}
	config := DefaultSlaveConfig("slave1", db, client)
	slave, err := NewSlave(config)
	if err != nil {
		t.Fatalf("Failed to create slave: %v", err)
	}

	// Insert test document
	coll := db.Collection("users")
	doc := map[string]interface{}{
		"_id":  "user1",
		"name": "Alice",
		"age":  int64(30),
	}
	if _, err := coll.InsertOne(doc); err != nil {
		t.Fatalf("Failed to insert test document: %v", err)
	}

	// Read document
	filter := map[string]interface{}{"_id": "user1"}
	result, err := slave.ReadDocument("users", filter)
	if err != nil {
		t.Fatalf("ReadDocument failed: %v", err)
	}

	// Verify result
	if result["name"] != "Alice" {
		t.Errorf("Expected name 'Alice', got %v", result["name"])
	}
	if result["age"] != int64(30) {
		t.Errorf("Expected age 30, got %v", result["age"])
	}

	// Read non-existent document
	filter2 := map[string]interface{}{"_id": "nonexistent"}
	_, err = slave.ReadDocument("users", filter2)
	if err == nil {
		t.Error("Expected error reading non-existent document")
	}
}

// TestSlaveReadDocuments tests the ReadDocuments function
func TestSlaveReadDocuments(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "slave")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	client := &mockMasterClient{entries: make([]*OplogEntry, 0)}
	config := DefaultSlaveConfig("slave1", db, client)
	slave, err := NewSlave(config)
	if err != nil {
		t.Fatalf("Failed to create slave: %v", err)
	}

	// Insert test documents
	coll := db.Collection("users")
	for i := 0; i < 5; i++ {
		doc := map[string]interface{}{
			"_id":  fmt.Sprintf("user%d", i),
			"name": fmt.Sprintf("User%d", i),
			"age":  int64(20 + i),
		}
		if _, err := coll.InsertOne(doc); err != nil {
			t.Fatalf("Failed to insert test document: %v", err)
		}
	}

	// Read all documents
	results, err := slave.ReadDocuments("users", nil)
	if err != nil {
		t.Fatalf("ReadDocuments failed: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 documents, got %d", len(results))
	}

	// Read filtered documents
	filter := map[string]interface{}{"age": map[string]interface{}{"$gte": int64(23)}}
	results, err = slave.ReadDocuments("users", filter)
	if err != nil {
		t.Fatalf("ReadDocuments with filter failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 documents matching filter, got %d", len(results))
	}
}

// TestWriteConcernGetTimeout tests the GetTimeout function
func TestWriteConcernGetTimeout(t *testing.T) {
	wc := &WriteConcern{
		W:        2,
		WTimeout: 5 * time.Second,
		J:        true,
	}

	timeout := wc.GetTimeout()
	if timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", timeout)
	}

	// Test with no timeout
	wc2 := &WriteConcern{W: 1}
	timeout2 := wc2.GetTimeout()
	if timeout2 != 0 {
		t.Errorf("Expected timeout 0, got %v", timeout2)
	}
}

// TestLocalMasterClientSendHeartbeat tests the SendHeartbeat function
func TestLocalMasterClientSendHeartbeat(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "master")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := DefaultMasterConfig(db, filepath.Join(tmpDir, "oplog.bin"))
	master, err := NewMaster(config)
	if err != nil {
		t.Fatalf("Failed to create master: %v", err)
	}
	defer master.Stop()

	if err := master.Start(); err != nil {
		t.Fatalf("Failed to start master: %v", err)
	}

	// Register slave
	if err := master.RegisterSlave("slave1"); err != nil {
		t.Fatalf("Failed to register slave: %v", err)
	}

	// Create client
	client := NewLocalMasterClient(master)

	// Send heartbeat
	ctx := context.Background()
	err = client.SendHeartbeat(ctx, "slave1", 100)
	if err != nil {
		t.Fatalf("SendHeartbeat failed: %v", err)
	}

	// Verify heartbeat was updated
	info, err := master.GetSlaveInfo("slave1")
	if err != nil {
		t.Fatalf("Failed to get slave info: %v", err)
	}

	if info.LastOpID != 100 {
		t.Errorf("Expected LastOpID 100, got %d", info.LastOpID)
	}

	// Test with cancelled context
	ctx2, cancel := context.WithCancel(context.Background())
	cancel()
	err = client.SendHeartbeat(ctx2, "slave1", 200)
	if err == nil {
		t.Error("Expected context cancellation error")
	}
}

// TestSlaveApplyEntry tests the applyEntry function with various operation types
func TestSlaveApplyEntry(t *testing.T) {
	tmpDir := t.TempDir()

	// Create master database
	masterDB, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "master")))
	if err != nil {
		t.Fatalf("Failed to open master database: %v", err)
	}
	defer masterDB.Close()

	// Create slave database
	slaveDB, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "slave")))
	if err != nil {
		t.Fatalf("Failed to open slave database: %v", err)
	}
	defer slaveDB.Close()

	// Create master
	masterConfig := DefaultMasterConfig(masterDB, filepath.Join(tmpDir, "oplog.bin"))
	master, err := NewMaster(masterConfig)
	if err != nil {
		t.Fatalf("Failed to create master: %v", err)
	}
	defer master.Stop()

	if err := master.Start(); err != nil {
		t.Fatalf("Failed to start master: %v", err)
	}

	// Create slave
	slaveConfig := &SlaveConfig{
		SlaveID:      "slave1",
		MasterClient: NewLocalMasterClient(master),
		Database:     slaveDB,
	}

	slave, err := NewSlave(slaveConfig)
	if err != nil {
		t.Fatalf("Failed to create slave: %v", err)
	}

	// Test OpTypeInsert
	insertEntry := &OplogEntry{
		OpID:       1,
		OpType:     OpTypeInsert,
		Collection: "users",
		Document:   map[string]interface{}{"name": "Alice", "age": int64(30)},
	}
	if err := slave.applyEntry(insertEntry); err != nil {
		t.Errorf("Failed to apply insert entry: %v", err)
	}

	// Verify document was inserted
	coll := slaveDB.Collection("users")
	docs, err := coll.Find(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to find documents: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("Expected 1 document, got %d", len(docs))
	}

	// Test OpTypeUpdate
	updateEntry := &OplogEntry{
		OpID:       2,
		OpType:     OpTypeUpdate,
		Collection: "users",
		Filter:     map[string]interface{}{"name": "Alice"},
		Update:     map[string]interface{}{"$set": map[string]interface{}{"age": int64(31)}},
	}
	if err := slave.applyEntry(updateEntry); err != nil {
		t.Errorf("Failed to apply update entry: %v", err)
	}

	// Test OpTypeDelete
	deleteEntry := &OplogEntry{
		OpID:       3,
		OpType:     OpTypeDelete,
		Collection: "users",
		Filter:     map[string]interface{}{"name": "Alice"},
	}
	if err := slave.applyEntry(deleteEntry); err != nil {
		t.Errorf("Failed to apply delete entry: %v", err)
	}

	// Verify document was deleted
	docs, err = coll.Find(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to find documents: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("Expected 0 documents after delete, got %d", len(docs))
	}

	// Test OpTypeCreateCollection
	createCollEntry := &OplogEntry{
		OpID:       4,
		OpType:     OpTypeCreateCollection,
		Collection: "orders",
	}
	if err := slave.applyEntry(createCollEntry); err != nil {
		t.Errorf("Failed to apply create collection entry: %v", err)
	}

	// Test OpTypeCreateCollection for existing collection (should not error)
	if err := slave.applyEntry(createCollEntry); err != nil {
		t.Errorf("Failed to apply create collection entry for existing collection: %v", err)
	}

	// Test OpTypeDropCollection
	dropCollEntry := &OplogEntry{
		OpID:       5,
		OpType:     OpTypeDropCollection,
		Collection: "orders",
	}
	if err := slave.applyEntry(dropCollEntry); err != nil {
		t.Errorf("Failed to apply drop collection entry: %v", err)
	}

	// Test OpTypeDropCollection for non-existent collection (should not error)
	if err := slave.applyEntry(dropCollEntry); err != nil {
		t.Errorf("Failed to apply drop collection entry for non-existent collection: %v", err)
	}

	// Test OpTypeCreateIndex (no-op for now)
	createIndexEntry := &OplogEntry{
		OpID:       6,
		OpType:     OpTypeCreateIndex,
		Collection: "users",
	}
	if err := slave.applyEntry(createIndexEntry); err != nil {
		t.Errorf("Failed to apply create index entry: %v", err)
	}

	// Test OpTypeDropIndex (no-op for now)
	dropIndexEntry := &OplogEntry{
		OpID:       7,
		OpType:     OpTypeDropIndex,
		Collection: "users",
	}
	if err := slave.applyEntry(dropIndexEntry); err != nil {
		t.Errorf("Failed to apply drop index entry: %v", err)
	}

	// Test OpTypeNoop
	noopEntry := &OplogEntry{
		OpID:   8,
		OpType: OpTypeNoop,
	}
	if err := slave.applyEntry(noopEntry); err != nil {
		t.Errorf("Failed to apply noop entry: %v", err)
	}

	// Test unknown operation type
	unknownEntry := &OplogEntry{
		OpID:       9,
		OpType:     100, // Invalid operation type
		Collection: "users",
	}
	err = slave.applyEntry(unknownEntry)
	if err == nil {
		t.Error("Expected error for unknown operation type")
	}
}

// TestSlaveSendHeartbeat tests the sendHeartbeat function
func TestSlaveSendHeartbeat(t *testing.T) {
	tmpDir := t.TempDir()

	// Create master database
	masterDB, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "master")))
	if err != nil {
		t.Fatalf("Failed to open master database: %v", err)
	}
	defer masterDB.Close()

	// Create slave database
	slaveDB, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "slave")))
	if err != nil {
		t.Fatalf("Failed to open slave database: %v", err)
	}
	defer slaveDB.Close()

	// Create master
	masterConfig := DefaultMasterConfig(masterDB, filepath.Join(tmpDir, "oplog.bin"))
	master, err := NewMaster(masterConfig)
	if err != nil {
		t.Fatalf("Failed to create master: %v", err)
	}
	defer master.Stop()

	if err := master.Start(); err != nil {
		t.Fatalf("Failed to start master: %v", err)
	}

	// Register slave
	if err := master.RegisterSlave("slave1"); err != nil {
		t.Fatalf("Failed to register slave: %v", err)
	}

	// Create slave
	slaveConfig := &SlaveConfig{
		SlaveID:      "slave1",
		MasterClient: NewLocalMasterClient(master),
		Database:     slaveDB,
	}

	slave, err := NewSlave(slaveConfig)
	if err != nil {
		t.Fatalf("Failed to create slave: %v", err)
	}

	// Set lastAppliedOpID
	slave.mu.Lock()
	slave.lastAppliedOpID = 123
	slave.mu.Unlock()

	// Call sendHeartbeat
	slave.sendHeartbeat()

	// Give some time for heartbeat to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify heartbeat was updated on master
	info, err := master.GetSlaveInfo("slave1")
	if err != nil {
		t.Fatalf("Failed to get slave info: %v", err)
	}

	if info.LastOpID != 123 {
		t.Errorf("Expected LastOpID 123, got %d", info.LastOpID)
	}
}
