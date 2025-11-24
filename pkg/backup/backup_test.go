package backup

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/index"
)

func TestNewBackupFormat(t *testing.T) {
	backup := NewBackupFormat("test_db")

	if backup.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", backup.Version)
	}

	if backup.DatabaseName != "test_db" {
		t.Errorf("Expected database name test_db, got %s", backup.DatabaseName)
	}

	if len(backup.Collections) != 0 {
		t.Errorf("Expected 0 collections, got %d", len(backup.Collections))
	}

	if time.Since(backup.Timestamp) > time.Second {
		t.Errorf("Timestamp should be recent")
	}
}

func TestAddCollection(t *testing.T) {
	backup := NewBackupFormat("test_db")

	// Create test documents
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "doc1",
			"name": "Alice",
			"age":  int64(30),
		}),
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "doc2",
			"name": "Bob",
			"age":  int64(25),
		}),
	}

	// Create test indexes
	indexes := []IndexBackup{
		{
			Name:       "name_idx",
			Type:       "btree",
			FieldPaths: []string{"name"},
			Unique:     false,
		},
	}

	backup.AddCollection("users", docs, indexes)

	if len(backup.Collections) != 1 {
		t.Fatalf("Expected 1 collection, got %d", len(backup.Collections))
	}

	coll := backup.Collections[0]
	if coll.Name != "users" {
		t.Errorf("Expected collection name users, got %s", coll.Name)
	}

	if len(coll.Documents) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(coll.Documents))
	}

	if len(coll.Indexes) != 1 {
		t.Errorf("Expected 1 index, got %d", len(coll.Indexes))
	}
}

func TestConvertDocumentToBackup(t *testing.T) {
	// Test with various field types
	oid := document.NewObjectID()
	now := time.Now()

	doc := document.NewDocumentFromMap(map[string]interface{}{
		"_id":       oid,
		"name":      "Alice",
		"age":       int64(30),
		"active":    true,
		"score":     95.5,
		"timestamp": now,
		"tags":      []interface{}{"tag1", "tag2"},
		"metadata": map[string]interface{}{
			"created": now,
		},
	})

	docBackup := convertDocumentToBackup(doc)

	// Check _id
	if docBackup.ID != oid.Hex() {
		t.Errorf("Expected ID %s, got %s", oid.Hex(), docBackup.ID)
	}

	// Check fields
	if docBackup.Fields["name"] != "Alice" {
		t.Errorf("Expected name Alice, got %v", docBackup.Fields["name"])
	}

	if docBackup.Fields["age"] != int64(30) {
		t.Errorf("Expected age 30, got %v", docBackup.Fields["age"])
	}

	// Check timestamp conversion
	if timestampStr, ok := docBackup.Fields["timestamp"].(string); !ok {
		t.Errorf("Expected timestamp to be string, got %T", docBackup.Fields["timestamp"])
	} else {
		parsed, err := time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			t.Errorf("Failed to parse timestamp: %v", err)
		}
		if parsed.Unix() != now.Unix() {
			t.Errorf("Timestamp mismatch")
		}
	}

	// Check nested metadata
	if metadata, ok := docBackup.Fields["metadata"].(map[string]interface{}); !ok {
		t.Errorf("Expected metadata to be map, got %T", docBackup.Fields["metadata"])
	} else {
		if _, ok := metadata["created"].(string); !ok {
			t.Errorf("Expected nested timestamp to be string")
		}
	}
}

func TestBackuper_BackupToWriter(t *testing.T) {
	backuper := NewBackuper(true)
	backup := NewBackupFormat("test_db")

	// Add a collection
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "doc1",
			"name": "Alice",
		}),
	}

	indexes := []IndexBackup{
		{
			Name:       "name_idx",
			Type:       "btree",
			FieldPaths: []string{"name"},
		},
	}

	backup.AddCollection("users", docs, indexes)

	// Write to buffer
	var buf bytes.Buffer
	err := backuper.BackupToWriter(&buf, backup)
	if err != nil {
		t.Fatalf("BackupToWriter failed: %v", err)
	}

	// Verify JSON is valid
	var parsed BackupFormat
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse backup JSON: %v", err)
	}

	if parsed.DatabaseName != "test_db" {
		t.Errorf("Expected database name test_db, got %s", parsed.DatabaseName)
	}

	if len(parsed.Collections) != 1 {
		t.Errorf("Expected 1 collection, got %d", len(parsed.Collections))
	}
}

func TestNewIndexBackup(t *testing.T) {
	// Create a B+ tree index
	idx := index.NewIndex(&index.IndexConfig{
		Name:      "age_idx",
		FieldPath: "age",
		Type:      index.IndexTypeBTree,
		Unique:    true,
		Order:     32,
	})

	backup := NewIndexBackup(idx)

	if backup.Name != "age_idx" {
		t.Errorf("Expected name age_idx, got %s", backup.Name)
	}

	if backup.Type != "btree" {
		t.Errorf("Expected type btree, got %s", backup.Type)
	}

	if !backup.Unique {
		t.Errorf("Expected unique to be true")
	}

	if len(backup.FieldPaths) != 1 || backup.FieldPaths[0] != "age" {
		t.Errorf("Expected field paths [age], got %v", backup.FieldPaths)
	}
}

func TestNewIndexBackup_Compound(t *testing.T) {
	// Create a compound index
	idx := index.NewIndex(&index.IndexConfig{
		Name:       "city_age_idx",
		FieldPaths: []string{"city", "age"},
		Type:       index.IndexTypeBTree,
		Unique:     false,
		Order:      32,
	})

	backup := NewIndexBackup(idx)

	if backup.Name != "city_age_idx" {
		t.Errorf("Expected name city_age_idx, got %s", backup.Name)
	}

	if len(backup.FieldPaths) != 2 {
		t.Errorf("Expected 2 field paths, got %d", len(backup.FieldPaths))
	}

	if backup.FieldPaths[0] != "city" || backup.FieldPaths[1] != "age" {
		t.Errorf("Expected field paths [city, age], got %v", backup.FieldPaths)
	}
}

func TestNewIndexBackup_Partial(t *testing.T) {
	// Create a partial index
	filter := map[string]interface{}{
		"age": map[string]interface{}{
			"$gt": int64(18),
		},
	}

	idx := index.NewIndex(&index.IndexConfig{
		Name:      "adult_idx",
		FieldPath: "age",
		Type:      index.IndexTypeBTree,
		Filter:    filter,
		Unique:    false,
		Order:     32,
	})

	backup := NewIndexBackup(idx)

	if backup.Filter == nil {
		t.Errorf("Expected filter to be set")
	}

	if len(backup.Filter) == 0 {
		t.Errorf("Expected filter to have content")
	}
}

func TestNewTextIndexBackup(t *testing.T) {
	backup := NewTextIndexBackup("text_idx", []string{"title", "body"})

	if backup.Name != "text_idx" {
		t.Errorf("Expected name text_idx, got %s", backup.Name)
	}

	if backup.Type != "text" {
		t.Errorf("Expected type text, got %s", backup.Type)
	}

	if len(backup.FieldPaths) != 2 {
		t.Errorf("Expected 2 field paths, got %d", len(backup.FieldPaths))
	}
}

func TestNewGeoIndexBackup(t *testing.T) {
	backup := NewGeoIndexBackup("location_idx", "location", "2dsphere")

	if backup.Name != "location_idx" {
		t.Errorf("Expected name location_idx, got %s", backup.Name)
	}

	if backup.Type != "geo" {
		t.Errorf("Expected type geo, got %s", backup.Type)
	}

	if backup.GeoType != "2dsphere" {
		t.Errorf("Expected geo type 2dsphere, got %s", backup.GeoType)
	}

	if len(backup.FieldPaths) != 1 || backup.FieldPaths[0] != "location" {
		t.Errorf("Expected field paths [location], got %v", backup.FieldPaths)
	}
}

func TestNewTTLIndexBackup(t *testing.T) {
	backup := NewTTLIndexBackup("ttl_idx", "created_at", 3600)

	if backup.Name != "ttl_idx" {
		t.Errorf("Expected name ttl_idx, got %s", backup.Name)
	}

	if backup.Type != "ttl" {
		t.Errorf("Expected type ttl, got %s", backup.Type)
	}

	if backup.TTLDuration == nil {
		t.Fatalf("Expected TTL duration to be set")
	}

	if *backup.TTLDuration != 3600 {
		t.Errorf("Expected TTL duration 3600, got %d", *backup.TTLDuration)
	}
}

func TestBackupFormat_Stats(t *testing.T) {
	backup := NewBackupFormat("test_db")

	// Add first collection
	docs1 := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"_id": "1"}),
		document.NewDocumentFromMap(map[string]interface{}{"_id": "2"}),
	}
	indexes1 := []IndexBackup{
		{Name: "idx1", Type: "btree", FieldPaths: []string{"field1"}},
	}
	backup.AddCollection("coll1", docs1, indexes1)

	// Add second collection
	docs2 := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{"_id": "3"}),
	}
	indexes2 := []IndexBackup{
		{Name: "idx2", Type: "btree", FieldPaths: []string{"field2"}},
		{Name: "idx3", Type: "text", FieldPaths: []string{"field3"}},
	}
	backup.AddCollection("coll2", docs2, indexes2)

	stats := backup.Stats()

	if stats["collections"] != 2 {
		t.Errorf("Expected 2 collections, got %v", stats["collections"])
	}

	if stats["total_documents"] != 3 {
		t.Errorf("Expected 3 documents, got %v", stats["total_documents"])
	}

	if stats["total_indexes"] != 3 {
		t.Errorf("Expected 3 indexes, got %v", stats["total_indexes"])
	}

	if stats["database_name"] != "test_db" {
		t.Errorf("Expected database name test_db, got %v", stats["database_name"])
	}
}

func TestBackuper_PrettyPrint(t *testing.T) {
	// Test with pretty printing enabled
	backuper := NewBackuper(true)
	backup := NewBackupFormat("test_db")

	var buf bytes.Buffer
	err := backuper.BackupToWriter(&buf, backup)
	if err != nil {
		t.Fatalf("BackupToWriter failed: %v", err)
	}

	// Check if output contains indentation
	if !bytes.Contains(buf.Bytes(), []byte("  ")) {
		t.Errorf("Expected pretty-printed output to contain indentation")
	}

	// Test without pretty printing
	backuper2 := NewBackuper(false)
	var buf2 bytes.Buffer
	err = backuper2.BackupToWriter(&buf2, backup)
	if err != nil {
		t.Fatalf("BackupToWriter failed: %v", err)
	}

	// Compact output should be shorter
	if buf2.Len() >= buf.Len() {
		t.Errorf("Compact output should be shorter than pretty output")
	}
}
