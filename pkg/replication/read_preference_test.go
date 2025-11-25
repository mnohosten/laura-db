package replication

import (
	"context"
	"testing"
	"time"

	"github.com/mnohosten/laura-db/pkg/database"
)

func TestReadPreferenceModeString(t *testing.T) {
	tests := []struct {
		mode     ReadPreferenceMode
		expected string
	}{
		{ReadPrimary, "primary"},
		{ReadPrimaryPreferred, "primaryPreferred"},
		{ReadSecondary, "secondary"},
		{ReadSecondaryPreferred, "secondaryPreferred"},
		{ReadNearest, "nearest"},
	}

	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.expected {
			t.Errorf("ReadPreferenceMode.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestNewReadPreference(t *testing.T) {
	rp := NewReadPreference(ReadSecondary)

	if rp.GetMode() != ReadSecondary {
		t.Errorf("Expected mode ReadSecondary, got %v", rp.GetMode())
	}

	if rp.GetMaxStaleness() != 0 {
		t.Errorf("Expected MaxStaleness 0, got %v", rp.GetMaxStaleness())
	}

	tags := rp.GetTags()
	if len(tags) != 0 {
		t.Errorf("Expected empty tags, got %v", tags)
	}
}

func TestReadPreferenceHelpers(t *testing.T) {
	tests := []struct {
		name     string
		pref     *ReadPreference
		expected ReadPreferenceMode
	}{
		{"Primary", Primary(), ReadPrimary},
		{"PrimaryPreferred", PrimaryPreferred(), ReadPrimaryPreferred},
		{"Secondary", Secondary(), ReadSecondary},
		{"SecondaryPreferred", SecondaryPreferred(), ReadSecondaryPreferred},
		{"Nearest", Nearest(), ReadNearest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pref.GetMode() != tt.expected {
				t.Errorf("Expected mode %v, got %v", tt.expected, tt.pref.GetMode())
			}
		})
	}
}

func TestReadPreferenceWithMaxStaleness(t *testing.T) {
	rp := Secondary().WithMaxStaleness(30)

	if rp.GetMode() != ReadSecondary {
		t.Errorf("Expected mode ReadSecondary, got %v", rp.GetMode())
	}

	if rp.GetMaxStaleness() != 30 {
		t.Errorf("Expected MaxStaleness 30, got %v", rp.GetMaxStaleness())
	}
}

func TestReadPreferenceWithTags(t *testing.T) {
	tags := map[string]string{
		"dc":     "east",
		"region": "us",
	}

	rp := Secondary().WithTags(tags)

	if rp.GetMode() != ReadSecondary {
		t.Errorf("Expected mode ReadSecondary, got %v", rp.GetMode())
	}

	gotTags := rp.GetTags()
	if len(gotTags) != 2 {
		t.Errorf("Expected 2 tags, got %v", len(gotTags))
	}

	if gotTags["dc"] != "east" || gotTags["region"] != "us" {
		t.Errorf("Expected tags %v, got %v", tags, gotTags)
	}
}

func TestReadPreferenceFluentAPI(t *testing.T) {
	rp := Secondary().
		WithMaxStaleness(60).
		WithTags(map[string]string{"dc": "west"})

	if rp.GetMode() != ReadSecondary {
		t.Errorf("Expected mode ReadSecondary, got %v", rp.GetMode())
	}

	if rp.GetMaxStaleness() != 60 {
		t.Errorf("Expected MaxStaleness 60, got %v", rp.GetMaxStaleness())
	}

	tags := rp.GetTags()
	if tags["dc"] != "west" {
		t.Errorf("Expected dc=west, got %v", tags["dc"])
	}
}

func TestReadPreferenceString(t *testing.T) {
	tests := []struct {
		name     string
		pref     *ReadPreference
		contains []string
	}{
		{
			"Primary",
			Primary(),
			[]string{"mode=primary"},
		},
		{
			"SecondaryWithMaxStaleness",
			Secondary().WithMaxStaleness(30),
			[]string{"mode=secondary", "maxStaleness=30s"},
		},
		{
			"SecondaryWithTags",
			Secondary().WithTags(map[string]string{"dc": "east"}),
			[]string{"mode=secondary", "tags="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.pref.String()
			for _, substr := range tt.contains {
				if !containsStr(s, substr) {
					t.Errorf("String() = %v, should contain %v", s, substr)
				}
			}
		})
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestReadPreferenceSelectorSelectPrimary(t *testing.T) {
	// Create database and replica set
	db, err := database.Open(database.DefaultConfig(t.TempDir()))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	oplogPath := t.TempDir() + "/oplog"
	config := DefaultReplicaSetConfig("rs0", "node1", db, oplogPath)
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}

	// Add members
	rs.AddMember("node2", 1, true)
	rs.AddMember("node3", 1, true)

	// Make node1 primary
	if err := rs.BecomePrimary(); err != nil {
		t.Fatalf("Failed to become primary: %v", err)
	}

	// Create selector
	selector := NewReadPreferenceSelector(rs)

	// Test primary selection
	ctx := context.Background()
	pref := Primary()

	nodeID, err := selector.SelectNode(ctx, pref)
	if err != nil {
		t.Fatalf("Failed to select node: %v", err)
	}

	if nodeID != "node1" {
		t.Errorf("Expected primary node1, got %v", nodeID)
	}
}

func TestReadPreferenceSelectorSelectSecondary(t *testing.T) {
	// Create database and replica set
	db, err := database.Open(database.DefaultConfig(t.TempDir()))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	oplogPath := t.TempDir() + "/oplog"
	config := DefaultReplicaSetConfig("rs0", "node1", db, oplogPath)
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}

	// Add secondary members
	rs.AddMember("node2", 1, true)
	rs.AddMember("node3", 1, true)

	// Update heartbeats to mark them as healthy
	rs.UpdateMemberHeartbeat("node2", 100)
	rs.UpdateMemberHeartbeat("node3", 100)

	// Make node1 primary (others will be secondary)
	if err := rs.BecomePrimary(); err != nil {
		t.Fatalf("Failed to become primary: %v", err)
	}

	// Create selector
	selector := NewReadPreferenceSelector(rs)

	// Test secondary selection
	ctx := context.Background()
	pref := Secondary()

	nodeID, err := selector.SelectNode(ctx, pref)
	if err != nil {
		t.Fatalf("Failed to select node: %v", err)
	}

	// Should select either node2 or node3 (both are secondaries)
	if nodeID != "node2" && nodeID != "node3" {
		t.Errorf("Expected secondary (node2 or node3), got %v", nodeID)
	}
}

func TestReadPreferenceSelectorSelectPrimaryPreferred(t *testing.T) {
	// Create database and replica set
	db, err := database.Open(database.DefaultConfig(t.TempDir()))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	oplogPath := t.TempDir() + "/oplog"
	config := DefaultReplicaSetConfig("rs0", "node1", db, oplogPath)
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}

	// Add members
	rs.AddMember("node2", 1, true)

	// Make node1 primary
	if err := rs.BecomePrimary(); err != nil {
		t.Fatalf("Failed to become primary: %v", err)
	}

	// Create selector
	selector := NewReadPreferenceSelector(rs)

	// Test primary preferred - should select primary
	ctx := context.Background()
	pref := PrimaryPreferred()

	nodeID, err := selector.SelectNode(ctx, pref)
	if err != nil {
		t.Fatalf("Failed to select node: %v", err)
	}

	if nodeID != "node1" {
		t.Errorf("Expected primary node1, got %v", nodeID)
	}
}

func TestReadPreferenceSelectorSelectSecondaryPreferred(t *testing.T) {
	// Create database and replica set
	db, err := database.Open(database.DefaultConfig(t.TempDir()))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	oplogPath := t.TempDir() + "/oplog"
	config := DefaultReplicaSetConfig("rs0", "node1", db, oplogPath)
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}

	// Add secondary member
	rs.AddMember("node2", 1, true)
	rs.UpdateMemberHeartbeat("node2", 100)

	// Make node1 primary
	if err := rs.BecomePrimary(); err != nil {
		t.Fatalf("Failed to become primary: %v", err)
	}

	// Create selector
	selector := NewReadPreferenceSelector(rs)

	// Test secondary preferred - should select secondary
	ctx := context.Background()
	pref := SecondaryPreferred()

	nodeID, err := selector.SelectNode(ctx, pref)
	if err != nil {
		t.Fatalf("Failed to select node: %v", err)
	}

	if nodeID != "node2" {
		t.Errorf("Expected secondary node2, got %v", nodeID)
	}
}

func TestReadPreferenceSelectorMaxStaleness(t *testing.T) {
	// Create database and replica set
	db, err := database.Open(database.DefaultConfig(t.TempDir()))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	oplogPath := t.TempDir() + "/oplog"
	config := DefaultReplicaSetConfig("rs0", "node1", db, oplogPath)
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}

	// Make node1 primary first
	if err := rs.BecomePrimary(); err != nil {
		t.Fatalf("Failed to become primary: %v", err)
	}

	// Add secondary members with different lag
	rs.AddMember("node2", 1, true)
	rs.AddMember("node3", 1, true)

	// Append enough entries to get a meaningful OpID
	// Each Append increments currentID by 1
	for i := 0; i < 10000; i++ {
		rs.oplog.Append(&OplogEntry{
			OpType:     OpTypeNoop,
			Collection: "test",
			Document:   map[string]interface{}{},
		})
	}

	// Get current OpID
	currentOpID := rs.oplog.GetCurrentID()

	// node2 has recent OpID (low lag) - caught up to current
	rs.UpdateMemberHeartbeat("node2", currentOpID)

	// node3 has old OpID (high lag) - 10 seconds behind
	// Lag = (currentOpID - lastOpID) * 1ms
	// To get 10s lag, we need difference of 10000
	rs.UpdateMemberHeartbeat("node3", currentOpID-10000)

	// Verify lag values
	members := rs.GetMembers()
	var node2Lag, node3Lag time.Duration
	for _, m := range members {
		if m.NodeID == "node2" {
			node2Lag = m.Lag
		} else if m.NodeID == "node3" {
			node3Lag = m.Lag
		}
	}

	// node2 should have 0 lag (caught up)
	// node3 should have ~10s lag
	if node2Lag != 0 {
		t.Logf("Warning: node2 lag is %v, expected 0", node2Lag)
	}
	if node3Lag < 5*time.Second {
		t.Logf("Warning: node3 lag is %v, expected >= 5s", node3Lag)
	}

	// Create selector
	selector := NewReadPreferenceSelector(rs)

	// Test with max staleness that excludes node3
	ctx := context.Background()
	pref := Secondary().WithMaxStaleness(5) // Max 5 second lag (excludes node3 with ~10s lag)

	nodeID, err := selector.SelectNode(ctx, pref)
	if err != nil {
		t.Fatalf("Failed to select node: %v", err)
	}

	// Should only select node2 (node3 is too stale)
	if nodeID != "node2" {
		// Get lag values for debugging
		t.Errorf("Expected node2 (low lag), got %v (node2 lag=%v, node3 lag=%v)", nodeID, node2Lag, node3Lag)
	}
}

func TestReadPreferenceSelectorNoNodesAvailable(t *testing.T) {
	// Create database and replica set
	db, err := database.Open(database.DefaultConfig(t.TempDir()))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	oplogPath := t.TempDir() + "/oplog"
	config := DefaultReplicaSetConfig("rs0", "node1", db, oplogPath)
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}

	// Make this node primary (so no secondaries available)
	if err := rs.BecomePrimary(); err != nil {
		t.Fatalf("Failed to become primary: %v", err)
	}

	// Create selector
	selector := NewReadPreferenceSelector(rs)

	// Test secondary selection with no secondaries
	ctx := context.Background()
	pref := Secondary()

	_, err = selector.SelectNode(ctx, pref)
	if err == nil {
		t.Error("Expected error when no secondary nodes available")
	}
}

func TestReadPreferenceSelectorNoPrimaryAvailable(t *testing.T) {
	// Create database and replica set
	db, err := database.Open(database.DefaultConfig(t.TempDir()))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	oplogPath := t.TempDir() + "/oplog"
	config := DefaultReplicaSetConfig("rs0", "node1", db, oplogPath)
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}

	// Don't make any node primary

	// Create selector
	selector := NewReadPreferenceSelector(rs)

	// Test primary selection with no primary
	ctx := context.Background()
	pref := Primary()

	_, err = selector.SelectNode(ctx, pref)
	if err == nil {
		t.Error("Expected error when no primary node available")
	}
}

func TestReadRouter(t *testing.T) {
	// Create database and replica set
	db, err := database.Open(database.DefaultConfig(t.TempDir()))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	oplogPath := t.TempDir() + "/oplog"
	config := DefaultReplicaSetConfig("rs0", "node1", db, oplogPath)
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}

	// Make node1 primary
	if err := rs.BecomePrimary(); err != nil {
		t.Fatalf("Failed to become primary: %v", err)
	}

	// Create collection and insert test data
	coll, err := db.CreateCollection("users")
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	_, err = coll.InsertOne(map[string]interface{}{
		"name": "Alice",
		"age":  int64(30),
	})
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Create read router
	router := NewReadRouter(rs)

	// Test read with primary preference
	ctx := context.Background()
	pref := Primary()

	doc, err := router.ReadDocument(ctx, "users", map[string]interface{}{"name": "Alice"}, pref)
	if err != nil {
		t.Fatalf("Failed to read document: %v", err)
	}

	if doc["name"] != "Alice" {
		t.Errorf("Expected name=Alice, got %v", doc["name"])
	}
}

func TestReadRouterReadDocuments(t *testing.T) {
	// Create database and replica set
	db, err := database.Open(database.DefaultConfig(t.TempDir()))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	oplogPath := t.TempDir() + "/oplog"
	config := DefaultReplicaSetConfig("rs0", "node1", db, oplogPath)
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}

	// Make node1 primary
	if err := rs.BecomePrimary(); err != nil {
		t.Fatalf("Failed to become primary: %v", err)
	}

	// Create collection and insert test data
	coll, err := db.CreateCollection("users")
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	docs := []map[string]interface{}{
		{"name": "Alice", "age": int64(30)},
		{"name": "Bob", "age": int64(25)},
		{"name": "Charlie", "age": int64(35)},
	}

	for _, doc := range docs {
		if _, err := coll.InsertOne(doc); err != nil {
			t.Fatalf("Failed to insert document: %v", err)
		}
	}

	// Create read router
	router := NewReadRouter(rs)

	// Test read multiple documents with primary preference
	ctx := context.Background()
	pref := Primary()

	results, err := router.ReadDocuments(ctx, "users", map[string]interface{}{}, pref)
	if err != nil {
		t.Fatalf("Failed to read documents: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 documents, got %v", len(results))
	}
}

func TestReadRouterGetSelectedNode(t *testing.T) {
	// Create database and replica set
	db, err := database.Open(database.DefaultConfig(t.TempDir()))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	oplogPath := t.TempDir() + "/oplog"
	config := DefaultReplicaSetConfig("rs0", "node1", db, oplogPath)
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}

	// Make node1 primary
	if err := rs.BecomePrimary(); err != nil {
		t.Fatalf("Failed to become primary: %v", err)
	}

	// Create read router
	router := NewReadRouter(rs)

	// Test get selected node
	ctx := context.Background()
	pref := Primary()

	nodeID, err := router.GetSelectedNode(ctx, pref)
	if err != nil {
		t.Fatalf("Failed to get selected node: %v", err)
	}

	if nodeID != "node1" {
		t.Errorf("Expected node1, got %v", nodeID)
	}
}

// TestReadPreferenceSelectorSelectNearest tests the selectNearest function
func TestReadPreferenceSelectorSelectNearest(t *testing.T) {
	// Create database and replica set
	db, err := database.Open(database.DefaultConfig(t.TempDir()))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	oplogPath := t.TempDir() + "/oplog"
	config := DefaultReplicaSetConfig("rs0", "node1", db, oplogPath)
	rs, err := NewReplicaSet(config)
	if err != nil {
		t.Fatalf("Failed to create replica set: %v", err)
	}

	// Add members
	rs.AddMember("node2", 1, true)
	rs.AddMember("node3", 1, true)

	// Update heartbeats to make all members healthy
	rs.UpdateMemberHeartbeat("node2", 0)
	rs.UpdateMemberHeartbeat("node3", 0)

	// Make node1 primary
	if err := rs.BecomePrimary(); err != nil {
		t.Fatalf("Failed to become primary: %v", err)
	}

	selector := NewReadPreferenceSelector(rs)
	ctx := context.Background()

	// Test nearest without filters
	pref := Nearest()
	nodeID, err := selector.SelectNode(ctx, pref)
	if err != nil {
		t.Fatalf("Failed to select nearest node: %v", err)
	}
	if nodeID == "" {
		t.Error("Expected non-empty node ID")
	}

	// Test nearest with max staleness
	pref = Nearest().WithMaxStaleness(10)
	nodeID, err = selector.SelectNode(ctx, pref)
	if err != nil {
		t.Fatalf("Failed to select nearest node with max staleness: %v", err)
	}
	if nodeID == "" {
		t.Error("Expected non-empty node ID")
	}

	// Test successful nearest selection - should select any healthy node
	pref = Nearest()
	nodeID, err = selector.SelectNode(ctx, pref)
	if err != nil {
		t.Fatalf("Failed to select nearest node: %v", err)
	}
	// Should select one of the healthy nodes
	validNodes := map[string]bool{"node1": true, "node2": true, "node3": true}
	if !validNodes[nodeID] {
		t.Errorf("Expected one of node1/node2/node3, got %v", nodeID)
	}
}
