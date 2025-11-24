package replication

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mnohosten/laura-db/pkg/database"
)

// Helper function to create a test database
func createTestDatabase(t *testing.T) *database.Database {
	tmpDir := t.TempDir()
	db, err := database.Open(database.DefaultConfig(filepath.Join(tmpDir, "testdb")))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	return db
}

func TestWriteConcern_DefaultWriteConcern(t *testing.T) {
	wc := DefaultWriteConcern()
	if wc.W != 1 {
		t.Errorf("expected w=1, got %v", wc.W)
	}
	if wc.WTimeout != 0 {
		t.Errorf("expected wtimeout=0, got %v", wc.WTimeout)
	}
	if wc.J {
		t.Errorf("expected j=false, got %v", wc.J)
	}
}

func TestWriteConcern_MajorityWriteConcern(t *testing.T) {
	wc := MajorityWriteConcern()
	if wc.W != "majority" {
		t.Errorf("expected w='majority', got %v", wc.W)
	}
	if !wc.IsAcknowledged() {
		t.Errorf("expected majority to be acknowledged")
	}
}

func TestWriteConcern_UnacknowledgedWriteConcern(t *testing.T) {
	wc := UnacknowledgedWriteConcern()
	if wc.W != 0 {
		t.Errorf("expected w=0, got %v", wc.W)
	}
	if wc.IsAcknowledged() {
		t.Errorf("expected w=0 to be unacknowledged")
	}
}

func TestWriteConcern_WithTimeout(t *testing.T) {
	wc := DefaultWriteConcern().WithTimeout(5 * time.Second)
	if wc.WTimeout != 5*time.Second {
		t.Errorf("expected wtimeout=5s, got %v", wc.WTimeout)
	}
	if wc.W != 1 {
		t.Errorf("expected w to remain 1, got %v", wc.W)
	}
}

func TestWriteConcern_WithJournal(t *testing.T) {
	wc := DefaultWriteConcern().WithJournal(true)
	if !wc.J {
		t.Errorf("expected j=true, got %v", wc.J)
	}
	if !wc.RequiresJournal() {
		t.Errorf("expected RequiresJournal() to return true")
	}
}

func TestWriteConcern_GetRequiredAcknowledgments(t *testing.T) {
	tests := []struct {
		name                string
		wc                  *WriteConcern
		totalVotingMembers  int
		expectedRequired    int
		expectedIsMajority  bool
		expectError         bool
	}{
		{
			name:               "w=1 with 3 members",
			wc:                 W1WriteConcern(),
			totalVotingMembers: 3,
			expectedRequired:   1,
			expectedIsMajority: false,
			expectError:        false,
		},
		{
			name:               "w=2 with 3 members",
			wc:                 W2WriteConcern(),
			totalVotingMembers: 3,
			expectedRequired:   2,
			expectedIsMajority: false,
			expectError:        false,
		},
		{
			name:               "w=majority with 3 members",
			wc:                 MajorityWriteConcern(),
			totalVotingMembers: 3,
			expectedRequired:   2, // (3/2)+1 = 2
			expectedIsMajority: true,
			expectError:        false,
		},
		{
			name:               "w=majority with 5 members",
			wc:                 MajorityWriteConcern(),
			totalVotingMembers: 5,
			expectedRequired:   3, // (5/2)+1 = 3
			expectedIsMajority: true,
			expectError:        false,
		},
		{
			name:               "w=0 (unacknowledged)",
			wc:                 UnacknowledgedWriteConcern(),
			totalVotingMembers: 3,
			expectedRequired:   0,
			expectedIsMajority: false,
			expectError:        false,
		},
		{
			name:               "w exceeds total members",
			wc:                 &WriteConcern{W: 5},
			totalVotingMembers: 3,
			expectError:        true,
		},
		{
			name:               "w is negative",
			wc:                 &WriteConcern{W: -1},
			totalVotingMembers: 3,
			expectError:        true,
		},
		{
			name:               "invalid w string",
			wc:                 &WriteConcern{W: "invalid"},
			totalVotingMembers: 3,
			expectError:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			required, isMajority, err := tt.wc.GetRequiredAcknowledgments(tt.totalVotingMembers)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if required != tt.expectedRequired {
				t.Errorf("expected required=%d, got %d", tt.expectedRequired, required)
			}
			if isMajority != tt.expectedIsMajority {
				t.Errorf("expected isMajority=%v, got %v", tt.expectedIsMajority, isMajority)
			}
		})
	}
}

func TestWriteConcern_Validate(t *testing.T) {
	tests := []struct {
		name        string
		wc          *WriteConcern
		expectError bool
	}{
		{
			name:        "valid w=1",
			wc:          W1WriteConcern(),
			expectError: false,
		},
		{
			name:        "valid w=majority",
			wc:          MajorityWriteConcern(),
			expectError: false,
		},
		{
			name:        "valid w=0",
			wc:          UnacknowledgedWriteConcern(),
			expectError: false,
		},
		{
			name:        "invalid w=-1",
			wc:          &WriteConcern{W: -1},
			expectError: true,
		},
		{
			name:        "invalid w=string",
			wc:          &WriteConcern{W: "invalid"},
			expectError: true,
		},
		{
			name:        "invalid wtimeout",
			wc:          &WriteConcern{W: 1, WTimeout: -1 * time.Second},
			expectError: true,
		},
		{
			name:        "nil write concern",
			wc:          nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.wc.Validate()
			if tt.expectError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestWriteConcern_String(t *testing.T) {
	tests := []struct {
		name     string
		wc       *WriteConcern
		contains []string
	}{
		{
			name:     "w=1",
			wc:       W1WriteConcern(),
			contains: []string{"w:1", "wtimeout:none", "j:false"},
		},
		{
			name:     "w=majority",
			wc:       MajorityWriteConcern(),
			contains: []string{"w:majority", "wtimeout:none", "j:false"},
		},
		{
			name:     "w=2 with timeout",
			wc:       W2WriteConcern().WithTimeout(5 * time.Second),
			contains: []string{"w:2", "wtimeout:5s", "j:false"},
		},
		{
			name:     "w=1 with journal",
			wc:       W1WriteConcern().WithJournal(true),
			contains: []string{"w:1", "wtimeout:none", "j:true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.wc.String()
			for _, substr := range tt.contains {
				if !contains(str, substr) {
					t.Errorf("expected string to contain '%s', got '%s'", substr, str)
				}
			}
		})
	}
}

func TestWriteResult_String(t *testing.T) {
	wr := &WriteResult{
		Acknowledged:      true,
		OpID:              123,
		NodesAcknowledged: 2,
		NodesRequired:     2,
		JournalSynced:     true,
		ElapsedTime:       100 * time.Millisecond,
	}

	str := wr.String()
	expectedSubstrings := []string{"acked:true", "opid:123", "nodes:2/2", "journal:true"}
	for _, substr := range expectedSubstrings {
		if !contains(str, substr) {
			t.Errorf("expected string to contain '%s', got '%s'", substr, str)
		}
	}
}

func TestReplicaSet_WriteWithConcern_W0(t *testing.T) {
	// Create a simple replica set for testing
	db := createTestDatabase(t)
	defer db.Close()

	config := DefaultReplicaSetConfig("rs0", "node1", db, t.TempDir()+"/oplog")
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("failed to create replica set: %v", err)
	}
	defer rs.Stop()

	// Start and become primary
	if err := rs.Start(); err != nil {
		t.Fatalf("failed to start replica set: %v", err)
	}
	if err := rs.BecomePrimary(); err != nil {
		t.Fatalf("failed to become primary: %v", err)
	}

	// Test w=0 (unacknowledged)
	wc := UnacknowledgedWriteConcern()
	entry := CreateInsertEntry("testdb", "testcoll", map[string]interface{}{"_id": "test1", "value": int64(123)})

	ctx := context.Background()
	result, err := rs.WriteWithConcern(ctx, entry, wc)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Acknowledged {
		t.Errorf("expected unacknowledged result")
	}
	if result.NodesAcknowledged != 0 {
		t.Errorf("expected 0 nodes acknowledged, got %d", result.NodesAcknowledged)
	}
}

func TestReplicaSet_WriteWithConcern_W1(t *testing.T) {
	db := createTestDatabase(t)
	defer db.Close()

	config := DefaultReplicaSetConfig("rs0", "node1", db, t.TempDir()+"/oplog")
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("failed to create replica set: %v", err)
	}
	defer rs.Stop()

	if err := rs.Start(); err != nil {
		t.Fatalf("failed to start replica set: %v", err)
	}
	if err := rs.BecomePrimary(); err != nil {
		t.Fatalf("failed to become primary: %v", err)
	}

	// Test w=1 (primary only)
	wc := W1WriteConcern()
	entry := CreateInsertEntry("testdb", "testcoll", map[string]interface{}{"_id": "test1", "value": int64(123)})

	ctx := context.Background()
	result, err := rs.WriteWithConcern(ctx, entry, wc)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result.Acknowledged {
		t.Errorf("expected acknowledged result")
	}
	if result.NodesAcknowledged < 1 {
		t.Errorf("expected at least 1 node acknowledged, got %d", result.NodesAcknowledged)
	}
}

func TestReplicaSet_WriteWithConcern_W2(t *testing.T) {
	db := createTestDatabase(t)
	defer db.Close()

	config := DefaultReplicaSetConfig("rs0", "node1", db, t.TempDir()+"/oplog")
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("failed to create replica set: %v", err)
	}
	defer rs.Stop()

	if err := rs.Start(); err != nil {
		t.Fatalf("failed to start replica set: %v", err)
	}
	if err := rs.BecomePrimary(); err != nil {
		t.Fatalf("failed to become primary: %v", err)
	}

	// Add a secondary member and simulate replication
	rs.AddMember("node2", 1, true)

	// Test w=2
	wc := W2WriteConcern().WithTimeout(2 * time.Second)
	entry := CreateInsertEntry("testdb", "testcoll", map[string]interface{}{"_id": "test1", "value": int64(123)})

	ctx := context.Background()

	// Simulate secondary catching up in a goroutine
	go func() {
		time.Sleep(100 * time.Millisecond)
		// Need to get the opID after the write happens
		time.Sleep(50 * time.Millisecond)
		// Simulate that node2 has caught up to the latest operation
		rs.UpdateMemberHeartbeat("node2", OpID(1))
	}()

	result, err := rs.WriteWithConcern(ctx, entry, wc)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result.Acknowledged {
		t.Errorf("expected acknowledged result")
	}
	if result.NodesAcknowledged < 2 {
		t.Errorf("expected at least 2 nodes acknowledged, got %d", result.NodesAcknowledged)
	}
}

func TestReplicaSet_WriteWithConcern_Majority(t *testing.T) {
	db := createTestDatabase(t)
	defer db.Close()

	config := DefaultReplicaSetConfig("rs0", "node1", db, t.TempDir()+"/oplog")
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("failed to create replica set: %v", err)
	}
	defer rs.Stop()

	if err := rs.Start(); err != nil {
		t.Fatalf("failed to start replica set: %v", err)
	}
	if err := rs.BecomePrimary(); err != nil {
		t.Fatalf("failed to become primary: %v", err)
	}

	// Add 2 secondary members (total 3, majority = 2)
	rs.AddMember("node2", 1, true)
	rs.AddMember("node3", 1, true)

	// Test w=majority
	wc := MajorityWriteConcern().WithTimeout(2 * time.Second)
	entry := CreateInsertEntry("testdb", "testcoll", map[string]interface{}{"_id": "test1", "value": int64(123)})

	ctx := context.Background()

	// Simulate one secondary catching up
	go func() {
		time.Sleep(100 * time.Millisecond)
		// Simulate that node2 has caught up to operation 1
		rs.UpdateMemberHeartbeat("node2", OpID(1))
	}()

	result, err := rs.WriteWithConcern(ctx, entry, wc)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result.Acknowledged {
		t.Errorf("expected acknowledged result")
	}
	// Should have at least 2 nodes (majority of 3)
	if result.NodesAcknowledged < 2 {
		t.Errorf("expected at least 2 nodes acknowledged, got %d", result.NodesAcknowledged)
	}
}

func TestReplicaSet_WriteWithConcern_Timeout(t *testing.T) {
	db := createTestDatabase(t)
	defer db.Close()

	config := DefaultReplicaSetConfig("rs0", "node1", db, t.TempDir()+"/oplog")
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("failed to create replica set: %v", err)
	}
	defer rs.Stop()

	if err := rs.Start(); err != nil {
		t.Fatalf("failed to start replica set: %v", err)
	}
	if err := rs.BecomePrimary(); err != nil {
		t.Fatalf("failed to become primary: %v", err)
	}

	// Add 2 secondaries that won't replicate
	rs.AddMember("node2", 1, true)
	rs.AddMember("node3", 1, true)

	// Test w=3 with short timeout (should timeout - need 3 nodes but only primary has it)
	wc := W3WriteConcern().WithTimeout(100 * time.Millisecond)
	entry := CreateInsertEntry("testdb", "testcoll", map[string]interface{}{"_id": "test1", "value": int64(123)})

	ctx := context.Background()
	result, err := rs.WriteWithConcern(ctx, entry, wc)
	if err == nil {
		t.Errorf("expected timeout error, got nil (result: %+v)", result)
	}
	if result == nil {
		t.Fatalf("expected non-nil result even on error")
	}
	// Should show partial success (only primary acknowledged)
	if result.NodesAcknowledged < 1 {
		t.Errorf("expected at least 1 node acknowledged (primary), got %d", result.NodesAcknowledged)
	}
	if result.Acknowledged {
		t.Errorf("expected unacknowledged on timeout")
	}
	if result.NodesRequired != 3 {
		t.Errorf("expected nodes required = 3, got %d", result.NodesRequired)
	}
}

func TestReplicaSet_WriteWithConcern_NotPrimary(t *testing.T) {
	db := createTestDatabase(t)
	defer db.Close()

	config := DefaultReplicaSetConfig("rs0", "node1", db, t.TempDir()+"/oplog")
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("failed to create replica set: %v", err)
	}
	defer rs.Stop()

	if err := rs.Start(); err != nil {
		t.Fatalf("failed to start replica set: %v", err)
	}

	// Don't become primary - stay as secondary
	wc := W1WriteConcern()
	entry := CreateInsertEntry("testdb", "testcoll", map[string]interface{}{"_id": "test1", "value": int64(123)})

	ctx := context.Background()
	_, err = rs.WriteWithConcern(ctx, entry, wc)
	if err == nil {
		t.Errorf("expected error when writing to secondary, got nil")
	}
	if !contains(err.Error(), "not primary") {
		t.Errorf("expected 'not primary' error, got: %v", err)
	}
}

func TestReplicaSet_WriteWithConcern_InvalidWriteConcern(t *testing.T) {
	db := createTestDatabase(t)
	defer db.Close()

	config := DefaultReplicaSetConfig("rs0", "node1", db, t.TempDir()+"/oplog")
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("failed to create replica set: %v", err)
	}
	defer rs.Stop()

	if err := rs.Start(); err != nil {
		t.Fatalf("failed to start replica set: %v", err)
	}
	if err := rs.BecomePrimary(); err != nil {
		t.Fatalf("failed to become primary: %v", err)
	}

	// Test with invalid write concern
	wc := &WriteConcern{W: -1}
	entry := CreateInsertEntry("testdb", "testcoll", map[string]interface{}{"_id": "test1", "value": int64(123)})

	ctx := context.Background()
	_, err = rs.WriteWithConcern(ctx, entry, wc)
	if err == nil {
		t.Errorf("expected error with invalid write concern, got nil")
	}
}

func TestReplicaSet_WriteWithConcern_WithJournal(t *testing.T) {
	db := createTestDatabase(t)
	defer db.Close()

	config := DefaultReplicaSetConfig("rs0", "node1", db, t.TempDir()+"/oplog")
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("failed to create replica set: %v", err)
	}
	defer rs.Stop()

	if err := rs.Start(); err != nil {
		t.Fatalf("failed to start replica set: %v", err)
	}
	if err := rs.BecomePrimary(); err != nil {
		t.Fatalf("failed to become primary: %v", err)
	}

	// Test with journal sync
	wc := W1WriteConcern().WithJournal(true)
	entry := CreateInsertEntry("testdb", "testcoll", map[string]interface{}{"_id": "test1", "value": int64(123)})

	ctx := context.Background()
	result, err := rs.WriteWithConcern(ctx, entry, wc)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result.Acknowledged {
		t.Errorf("expected acknowledged result")
	}
	if !result.JournalSynced {
		t.Errorf("expected journal synced")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
