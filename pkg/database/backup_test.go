package database

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mnohosten/laura-db/pkg/backup"
	"github.com/mnohosten/laura-db/pkg/document"
)

func TestDatabase_Backup(t *testing.T) {
	// Create temporary database
	dataDir := t.TempDir()
	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create collection and insert documents
	coll := db.Collection("users")
	_, err = coll.InsertOne(map[string]interface{}{
		"name": "Alice",
		"age":  int64(30),
	})
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	_, err = coll.InsertOne(map[string]interface{}{
		"name": "Bob",
		"age":  int64(25),
	})
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Create an index
	err = coll.CreateIndex("name", true)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	// Create backup
	backupFormat, err := db.Backup()
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Verify backup
	if backupFormat.DatabaseName != "default" {
		t.Errorf("Expected database name 'default', got %s", backupFormat.DatabaseName)
	}

	if len(backupFormat.Collections) != 1 {
		t.Fatalf("Expected 1 collection, got %d", len(backupFormat.Collections))
	}

	collBackup := backupFormat.Collections[0]
	if collBackup.Name != "users" {
		t.Errorf("Expected collection name 'users', got %s", collBackup.Name)
	}

	if len(collBackup.Documents) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(collBackup.Documents))
	}

	// Should have 1 index (excluding default _id_ index)
	if len(collBackup.Indexes) != 1 {
		t.Errorf("Expected 1 index, got %d", len(collBackup.Indexes))
	}

	if collBackup.Indexes[0].Name != "name_1" {
		t.Errorf("Expected index name 'name_1', got %s", collBackup.Indexes[0].Name)
	}
}

func TestDatabase_Restore(t *testing.T) {
	// Create temporary database
	dataDir := t.TempDir()
	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create a backup format manually
	backupFormat := backup.NewBackupFormat("test_db")

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

	indexes := []backup.IndexBackup{
		{
			Name:       "age_idx",
			Type:       "btree",
			FieldPaths: []string{"age"},
			Unique:     false,
		},
	}

	backupFormat.AddCollection("users", docs, indexes)

	// Restore the backup
	opts := backup.DefaultRestoreOptions()
	err = db.Restore(backupFormat, opts)
	if err != nil {
		t.Fatalf("Failed to restore backup: %v", err)
	}

	// Verify restored data
	coll := db.Collection("users")
	allDocs, err := coll.Find(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to find documents: %v", err)
	}

	if len(allDocs) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(allDocs))
	}

	// Verify index was created
	stats := coll.Stats()
	indexCount := stats["index_count"].(int)
	if indexCount != 2 { // _id_ + age_idx
		t.Errorf("Expected 2 indexes, got %d", indexCount)
	}
}

func TestDatabase_BackupToFile(t *testing.T) {
	// Create temporary database
	dataDir := t.TempDir()
	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create collection and insert documents
	coll := db.Collection("users")
	_, err = coll.InsertOne(map[string]interface{}{
		"name": "Alice",
	})
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Backup to file
	backupPath := filepath.Join(dataDir, "backup.json")
	err = db.BackupToFile(backupPath, true)
	if err != nil {
		t.Fatalf("Failed to backup to file: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("Backup file was not created")
	}
}

func TestDatabase_RestoreFromFile(t *testing.T) {
	// Create temporary database and backup
	dataDir1 := t.TempDir()
	db1, err := Open(DefaultConfig(dataDir1))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	coll := db1.Collection("users")
	_, err = coll.InsertOne(map[string]interface{}{
		"name": "Alice",
		"age":  int64(30),
	})
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	backupPath := filepath.Join(dataDir1, "backup.json")
	err = db1.BackupToFile(backupPath, false)
	if err != nil {
		t.Fatalf("Failed to backup: %v", err)
	}
	db1.Close()

	// Create new database and restore
	dataDir2 := t.TempDir()
	db2, err := Open(DefaultConfig(dataDir2))
	if err != nil {
		t.Fatalf("Failed to open second database: %v", err)
	}
	defer db2.Close()

	opts := backup.DefaultRestoreOptions()
	err = db2.RestoreFromFile(backupPath, opts)
	if err != nil {
		t.Fatalf("Failed to restore from file: %v", err)
	}

	// Verify restored data
	coll2 := db2.Collection("users")
	doc, err := coll2.FindOne(map[string]interface{}{"name": "Alice"})
	if err != nil {
		t.Fatalf("Failed to find restored document: %v", err)
	}

	age, _ := doc.Get("age")
	if age != int64(30) {
		t.Errorf("Expected age 30, got %v", age)
	}
}

func TestDatabase_BackupWithMultipleCollections(t *testing.T) {
	dataDir := t.TempDir()
	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create multiple collections
	users := db.Collection("users")
	_, err = users.InsertOne(map[string]interface{}{"name": "Alice"})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	products := db.Collection("products")
	_, err = products.InsertOne(map[string]interface{}{"name": "Widget"})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Backup
	backupFormat, err := db.Backup()
	if err != nil {
		t.Fatalf("Failed to backup: %v", err)
	}

	if len(backupFormat.Collections) != 2 {
		t.Errorf("Expected 2 collections, got %d", len(backupFormat.Collections))
	}
}

func TestDatabase_RestoreWithDropExisting(t *testing.T) {
	dataDir := t.TempDir()
	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Insert initial data
	coll := db.Collection("users")
	_, err = coll.InsertOne(map[string]interface{}{"name": "Old Data"})
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Create backup with different data
	backupFormat := backup.NewBackupFormat("test_db")
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "new1",
			"name": "New Data",
		}),
	}
	backupFormat.AddCollection("users", docs, []backup.IndexBackup{})

	// Restore with DropExisting=true
	opts := &backup.RestoreOptions{
		DropExisting: true,
	}
	err = db.Restore(backupFormat, opts)
	if err != nil {
		t.Fatalf("Failed to restore: %v", err)
	}

	// Verify only new data exists
	allDocs, err := coll.Find(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to find: %v", err)
	}

	if len(allDocs) != 1 {
		t.Errorf("Expected 1 document after restore, got %d", len(allDocs))
	}
}

func TestDatabase_RestoreWithCompoundIndexes(t *testing.T) {
	dataDir := t.TempDir()
	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create backup with compound index
	backupFormat := backup.NewBackupFormat("test_db")
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "doc1",
			"city": "NYC",
			"age":  int64(30),
		}),
	}

	indexes := []backup.IndexBackup{
		{
			Name:       "city_age_idx",
			Type:       "btree",
			FieldPaths: []string{"city", "age"},
			Unique:     false,
		},
	}

	backupFormat.AddCollection("users", docs, indexes)

	// Restore
	opts := backup.DefaultRestoreOptions()
	err = db.Restore(backupFormat, opts)
	if err != nil {
		t.Fatalf("Failed to restore: %v", err)
	}

	// Verify compound index was created
	coll := db.Collection("users")
	stats := coll.Stats()
	indexCount := stats["index_count"].(int)
	if indexCount != 2 { // _id_ + city_age_idx
		t.Errorf("Expected 2 indexes, got %d", indexCount)
	}
}

func TestDatabase_RestoreWithTextIndex(t *testing.T) {
	dataDir := t.TempDir()
	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create backup with text index
	backupFormat := backup.NewBackupFormat("test_db")
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":   "doc1",
			"title": "Hello World",
			"body":  "This is a test document",
		}),
	}

	indexes := []backup.IndexBackup{
		{
			Name:       "text_idx",
			Type:       "text",
			FieldPaths: []string{"title", "body"},
		},
	}

	backupFormat.AddCollection("articles", docs, indexes)

	// Restore
	opts := backup.DefaultRestoreOptions()
	err = db.Restore(backupFormat, opts)
	if err != nil {
		t.Fatalf("Failed to restore: %v", err)
	}

	// Verify text index was created
	coll := db.Collection("articles")
	stats := coll.Stats()
	textIndexCount := stats["text_index_count"].(int)
	if textIndexCount != 1 {
		t.Errorf("Expected 1 text index, got %d", textIndexCount)
	}
}

func TestDatabase_RestoreWithGeoIndex(t *testing.T) {
	dataDir := t.TempDir()
	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create backup with geo index
	backupFormat := backup.NewBackupFormat("test_db")
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id": "doc1",
			"location": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{float64(-122.4), float64(37.8)},
			},
		}),
	}

	indexes := []backup.IndexBackup{
		{
			Name:       "location_idx",
			Type:       "geo",
			FieldPaths: []string{"location"},
			GeoType:    "2dsphere",
		},
	}

	backupFormat.AddCollection("places", docs, indexes)

	// Restore
	opts := backup.DefaultRestoreOptions()
	err = db.Restore(backupFormat, opts)
	if err != nil {
		t.Fatalf("Failed to restore: %v", err)
	}

	// Verify geo index was created
	coll := db.Collection("places")
	stats := coll.Stats()
	geoIndexCount := stats["geo_index_count"].(int)
	if geoIndexCount != 1 {
		t.Errorf("Expected 1 geo index, got %d", geoIndexCount)
	}
}

func TestDatabase_RestoreWithTTLIndex(t *testing.T) {
	dataDir := t.TempDir()
	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create backup with TTL index
	backupFormat := backup.NewBackupFormat("test_db")
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":        "doc1",
			"created_at": time.Now(),
		}),
	}

	ttlSeconds := int64(3600)
	indexes := []backup.IndexBackup{
		{
			Name:        "ttl_idx",
			Type:        "ttl",
			FieldPaths:  []string{"created_at"},
			TTLDuration: &ttlSeconds,
		},
	}

	backupFormat.AddCollection("sessions", docs, indexes)

	// Restore
	opts := backup.DefaultRestoreOptions()
	err = db.Restore(backupFormat, opts)
	if err != nil {
		t.Fatalf("Failed to restore: %v", err)
	}

	// Verify TTL index was created
	coll := db.Collection("sessions")
	stats := coll.Stats()
	ttlIndexCount := stats["ttl_index_count"].(int)
	if ttlIndexCount != 1 {
		t.Errorf("Expected 1 TTL index, got %d", ttlIndexCount)
	}
}

func TestDatabase_RestoreWithPartialIndex(t *testing.T) {
	dataDir := t.TempDir()
	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create backup with partial index
	backupFormat := backup.NewBackupFormat("test_db")
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "doc1",
			"age":  int64(25),
			"name": "Alice",
		}),
	}

	filter := map[string]interface{}{
		"age": map[string]interface{}{
			"$gt": int64(18),
		},
	}

	indexes := []backup.IndexBackup{
		{
			Name:       "adult_idx",
			Type:       "btree",
			FieldPaths: []string{"name"},
			Unique:     false,
			Filter:     filter,
		},
	}

	backupFormat.AddCollection("users", docs, indexes)

	// Restore
	opts := backup.DefaultRestoreOptions()
	err = db.Restore(backupFormat, opts)
	if err != nil {
		t.Fatalf("Failed to restore: %v", err)
	}

	// Verify partial index was created
	coll := db.Collection("users")
	stats := coll.Stats()
	indexCount := stats["index_count"].(int)
	if indexCount != 2 { // _id_ + adult_idx
		t.Errorf("Expected 2 indexes, got %d", indexCount)
	}
}

func TestDatabase_RestoreSkipIndexes(t *testing.T) {
	dataDir := t.TempDir()
	db, err := Open(DefaultConfig(dataDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create backup with indexes
	backupFormat := backup.NewBackupFormat("test_db")
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "doc1",
			"name": "Alice",
		}),
	}

	indexes := []backup.IndexBackup{
		{
			Name:       "name_idx",
			Type:       "btree",
			FieldPaths: []string{"name"},
		},
	}

	backupFormat.AddCollection("users", docs, indexes)

	// Restore with SkipIndexes=true
	opts := &backup.RestoreOptions{
		SkipIndexes: true,
	}
	err = db.Restore(backupFormat, opts)
	if err != nil {
		t.Fatalf("Failed to restore: %v", err)
	}

	// Verify only default _id_ index exists
	coll := db.Collection("users")
	stats := coll.Stats()
	indexCount := stats["index_count"].(int)
	if indexCount != 1 { // Only _id_
		t.Errorf("Expected 1 index (only _id_), got %d", indexCount)
	}
}
