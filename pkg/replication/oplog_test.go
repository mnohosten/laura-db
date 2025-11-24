package replication

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestOplogBasic(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	oplogPath := filepath.Join(tmpDir, "oplog.bin")

	// Create oplog
	oplog, err := NewOplog(oplogPath)
	if err != nil {
		t.Fatalf("Failed to create oplog: %v", err)
	}
	defer oplog.Close()

	// Append an entry
	entry := CreateInsertEntry("testdb", "users", map[string]interface{}{
		"name": "Alice",
		"age":  int64(30),
	})

	if err := oplog.Append(entry); err != nil {
		t.Fatalf("Failed to append entry: %v", err)
	}

	// Check OpID was assigned
	if entry.OpID != 1 {
		t.Errorf("Expected OpID 1, got %d", entry.OpID)
	}

	// Check timestamp was set
	if entry.Timestamp.IsZero() {
		t.Error("Timestamp not set")
	}
}

func TestOplogMultipleEntries(t *testing.T) {
	tmpDir := t.TempDir()
	oplogPath := filepath.Join(tmpDir, "oplog.bin")

	oplog, err := NewOplog(oplogPath)
	if err != nil {
		t.Fatalf("Failed to create oplog: %v", err)
	}
	defer oplog.Close()

	// Append multiple entries
	for i := 0; i < 10; i++ {
		entry := CreateInsertEntry("testdb", "users", map[string]interface{}{
			"index": int64(i),
		})

		if err := oplog.Append(entry); err != nil {
			t.Fatalf("Failed to append entry %d: %v", i, err)
		}

		if entry.OpID != OpID(i+1) {
			t.Errorf("Expected OpID %d, got %d", i+1, entry.OpID)
		}
	}

	// Check current ID
	if oplog.GetCurrentID() != 10 {
		t.Errorf("Expected current ID 10, got %d", oplog.GetCurrentID())
	}
}

func TestOplogGetEntriesSince(t *testing.T) {
	tmpDir := t.TempDir()
	oplogPath := filepath.Join(tmpDir, "oplog.bin")

	oplog, err := NewOplog(oplogPath)
	if err != nil {
		t.Fatalf("Failed to create oplog: %v", err)
	}
	defer oplog.Close()

	// Append 10 entries
	for i := 0; i < 10; i++ {
		entry := CreateInsertEntry("testdb", "users", map[string]interface{}{
			"index": int64(i),
		})
		if err := oplog.Append(entry); err != nil {
			t.Fatalf("Failed to append entry: %v", err)
		}
	}

	// Get entries since OpID 5
	entries, err := oplog.GetEntriesSince(5)
	if err != nil {
		t.Fatalf("Failed to get entries: %v", err)
	}

	if len(entries) != 5 {
		t.Errorf("Expected 5 entries, got %d", len(entries))
	}

	// Check OpIDs
	for i, entry := range entries {
		expectedID := OpID(6 + i)
		if entry.OpID != expectedID {
			t.Errorf("Expected OpID %d, got %d", expectedID, entry.OpID)
		}
	}
}

func TestOplogPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	oplogPath := filepath.Join(tmpDir, "oplog.bin")

	// Create oplog and write entries
	oplog1, err := NewOplog(oplogPath)
	if err != nil {
		t.Fatalf("Failed to create oplog: %v", err)
	}

	for i := 0; i < 5; i++ {
		entry := CreateInsertEntry("testdb", "users", map[string]interface{}{
			"index": int64(i),
		})
		if err := oplog1.Append(entry); err != nil {
			t.Fatalf("Failed to append entry: %v", err)
		}
	}

	if err := oplog1.Close(); err != nil {
		t.Fatalf("Failed to close oplog: %v", err)
	}

	// Reopen oplog
	oplog2, err := NewOplog(oplogPath)
	if err != nil {
		t.Fatalf("Failed to reopen oplog: %v", err)
	}
	defer oplog2.Close()

	// Check current ID was restored
	if oplog2.GetCurrentID() != 5 {
		t.Errorf("Expected current ID 5, got %d", oplog2.GetCurrentID())
	}

	// Append another entry
	entry := CreateInsertEntry("testdb", "users", map[string]interface{}{
		"index": int64(5),
	})
	if err := oplog2.Append(entry); err != nil {
		t.Fatalf("Failed to append entry: %v", err)
	}

	if entry.OpID != 6 {
		t.Errorf("Expected OpID 6, got %d", entry.OpID)
	}
}

func TestOplogOperationTypes(t *testing.T) {
	tmpDir := t.TempDir()
	oplogPath := filepath.Join(tmpDir, "oplog.bin")

	oplog, err := NewOplog(oplogPath)
	if err != nil {
		t.Fatalf("Failed to create oplog: %v", err)
	}
	defer oplog.Close()

	tests := []struct {
		name  string
		entry *OplogEntry
	}{
		{
			name: "Insert",
			entry: CreateInsertEntry("testdb", "users", map[string]interface{}{
				"name": "Alice",
			}),
		},
		{
			name: "Update",
			entry: CreateUpdateEntry("testdb", "users",
				map[string]interface{}{"name": "Alice"},
				map[string]interface{}{"$set": map[string]interface{}{"age": int64(31)}},
			),
		},
		{
			name: "Delete",
			entry: CreateDeleteEntry("testdb", "users",
				map[string]interface{}{"name": "Alice"},
			),
		},
		{
			name:  "CreateCollection",
			entry: CreateCollectionEntry("testdb", "products", true),
		},
		{
			name:  "DropCollection",
			entry: CreateCollectionEntry("testdb", "products", false),
		},
		{
			name: "CreateIndex",
			entry: CreateIndexEntry("testdb", "users",
				map[string]interface{}{"field": "name"},
				true,
			),
		},
		{
			name: "DropIndex",
			entry: CreateIndexEntry("testdb", "users",
				map[string]interface{}{"field": "name"},
				false,
			),
		},
		{
			name:  "Noop",
			entry: CreateNoopEntry("testdb"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := oplog.Append(tt.entry); err != nil {
				t.Fatalf("Failed to append %s entry: %v", tt.name, err)
			}

			if tt.entry.OpID == 0 {
				t.Error("OpID not assigned")
			}

			if tt.entry.Timestamp.IsZero() {
				t.Error("Timestamp not set")
			}
		})
	}
}

func TestOplogSerialization(t *testing.T) {
	tmpDir := t.TempDir()
	oplogPath := filepath.Join(tmpDir, "oplog.bin")

	oplog, err := NewOplog(oplogPath)
	if err != nil {
		t.Fatalf("Failed to create oplog: %v", err)
	}
	defer oplog.Close()

	// Create entry with various field types
	original := &OplogEntry{
		OpType:     OpTypeInsert,
		Database:   "testdb",
		Collection: "users",
		DocID:      "12345",
		Document: map[string]interface{}{
			"name":   "Alice",
			"age":    int64(30),
			"active": true,
			"tags":   []interface{}{"user", "admin"},
		},
	}

	if err := oplog.Append(original); err != nil {
		t.Fatalf("Failed to append entry: %v", err)
	}

	// Read back
	entries, err := oplog.GetEntriesSince(0)
	if err != nil {
		t.Fatalf("Failed to get entries: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]

	// Verify fields
	if entry.OpType != original.OpType {
		t.Errorf("OpType mismatch: expected %v, got %v", original.OpType, entry.OpType)
	}

	if entry.Database != original.Database {
		t.Errorf("Database mismatch: expected %s, got %s", original.Database, entry.Database)
	}

	if entry.Collection != original.Collection {
		t.Errorf("Collection mismatch: expected %s, got %s", original.Collection, entry.Collection)
	}

	if entry.DocID != original.DocID {
		t.Errorf("DocID mismatch: expected %v, got %v", original.DocID, entry.DocID)
	}

	// Verify document fields
	if name, ok := entry.Document["name"].(string); !ok || name != "Alice" {
		t.Errorf("Name mismatch: expected Alice, got %v", entry.Document["name"])
	}
}

func TestOplogConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	oplogPath := filepath.Join(tmpDir, "oplog.bin")

	oplog, err := NewOplog(oplogPath)
	if err != nil {
		t.Fatalf("Failed to create oplog: %v", err)
	}
	defer oplog.Close()

	// Concurrent writes
	done := make(chan bool)
	numGoroutines := 10
	entriesPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < entriesPerGoroutine; j++ {
				entry := CreateInsertEntry("testdb", "users", map[string]interface{}{
					"goroutine": int64(id),
					"index":     int64(j),
				})
				if err := oplog.Append(entry); err != nil {
					t.Errorf("Failed to append: %v", err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Check total entries
	expectedTotal := numGoroutines * entriesPerGoroutine
	if oplog.GetCurrentID() != OpID(expectedTotal) {
		t.Errorf("Expected %d entries, got %d", expectedTotal, oplog.GetCurrentID())
	}
}

func TestOplogFlush(t *testing.T) {
	tmpDir := t.TempDir()
	oplogPath := filepath.Join(tmpDir, "oplog.bin")

	oplog, err := NewOplog(oplogPath)
	if err != nil {
		t.Fatalf("Failed to create oplog: %v", err)
	}

	// Write entry
	entry := CreateInsertEntry("testdb", "users", map[string]interface{}{
		"name": "Alice",
	})
	if err := oplog.Append(entry); err != nil {
		t.Fatalf("Failed to append entry: %v", err)
	}

	// Flush
	if err := oplog.Flush(); err != nil {
		t.Fatalf("Failed to flush: %v", err)
	}

	oplog.Close()

	// Check file exists and has content
	info, err := os.Stat(oplogPath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	if info.Size() == 0 {
		t.Error("File is empty after flush")
	}
}

func TestOplogLargeEntries(t *testing.T) {
	tmpDir := t.TempDir()
	oplogPath := filepath.Join(tmpDir, "oplog.bin")

	oplog, err := NewOplog(oplogPath)
	if err != nil {
		t.Fatalf("Failed to create oplog: %v", err)
	}
	defer oplog.Close()

	// Create large document
	largeDoc := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		key := string(rune('A'+i%26)) + fmt.Sprintf("%d", i)
		value := fmt.Sprintf("value_%d", i)
		largeDoc[key] = value
	}

	entry := CreateInsertEntry("testdb", "users", largeDoc)
	if err := oplog.Append(entry); err != nil {
		t.Fatalf("Failed to append large entry: %v", err)
	}

	// Read back
	entries, err := oplog.GetEntriesSince(0)
	if err != nil {
		t.Fatalf("Failed to get entries: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if len(entries[0].Document) != len(largeDoc) {
		t.Errorf("Document size mismatch: expected %d fields, got %d", len(largeDoc), len(entries[0].Document))
	}
}

func BenchmarkOplogAppend(b *testing.B) {
	tmpDir := b.TempDir()
	oplogPath := filepath.Join(tmpDir, "oplog.bin")

	oplog, err := NewOplog(oplogPath)
	if err != nil {
		b.Fatalf("Failed to create oplog: %v", err)
	}
	defer oplog.Close()

	entry := CreateInsertEntry("testdb", "users", map[string]interface{}{
		"name": "Alice",
		"age":  int64(30),
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := oplog.Append(entry); err != nil {
			b.Fatalf("Failed to append: %v", err)
		}
	}
}

func BenchmarkOplogGetEntriesSince(b *testing.B) {
	tmpDir := b.TempDir()
	oplogPath := filepath.Join(tmpDir, "oplog.bin")

	oplog, err := NewOplog(oplogPath)
	if err != nil {
		b.Fatalf("Failed to create oplog: %v", err)
	}
	defer oplog.Close()

	// Append 10000 entries
	for i := 0; i < 10000; i++ {
		entry := CreateInsertEntry("testdb", "users", map[string]interface{}{
			"index": int64(i),
		})
		if err := oplog.Append(entry); err != nil {
			b.Fatalf("Failed to append: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := oplog.GetEntriesSince(9000)
		if err != nil {
			b.Fatalf("Failed to get entries: %v", err)
		}
	}
}
