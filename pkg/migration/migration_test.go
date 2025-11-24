package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mnohosten/laura-db/pkg/database"
)

func setupTestDB(t *testing.T) (*database.Database, func()) {
	dir := t.TempDir()
	config := database.DefaultConfig(dir)
	db, err := database.Open(config)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func TestCreateMigration(t *testing.T) {
	migration := CreateMigration("test_migration", "Test migration description")

	if migration.Name != "test_migration" {
		t.Errorf("Expected name 'test_migration', got '%s'", migration.Name)
	}

	if migration.Description != "Test migration description" {
		t.Errorf("Expected description 'Test migration description', got '%s'", migration.Description)
	}

	if migration.Version <= 0 {
		t.Errorf("Expected positive version, got %d", migration.Version)
	}

	if migration.UpScript == nil || migration.DownScript == nil {
		t.Error("Expected UpScript and DownScript to be initialized")
	}
}

func TestMigratorAddMigration(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	migrator := NewMigrator(db)

	migration := &Migration{
		Version:     1,
		Name:        "test",
		Description: "Test migration",
		Up:          func(db *database.Database) error { return nil },
		Down:        func(db *database.Database) error { return nil },
	}

	err := migrator.AddMigration(migration)
	if err != nil {
		t.Errorf("Failed to add migration: %v", err)
	}

	if len(migrator.migrations) != 1 {
		t.Errorf("Expected 1 migration, got %d", len(migrator.migrations))
	}
}

func TestMigratorAddMigrationValidation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	migrator := NewMigrator(db)

	tests := []struct {
		name        string
		migration   *Migration
		expectError bool
	}{
		{
			name: "invalid version",
			migration: &Migration{
				Version: 0,
				Name:    "test",
			},
			expectError: true,
		},
		{
			name: "empty name",
			migration: &Migration{
				Version: 1,
				Name:    "",
			},
			expectError: true,
		},
		{
			name: "valid migration",
			migration: &Migration{
				Version: 1,
				Name:    "test",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := migrator.AddMigration(tt.migration)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestMigratorDuplicateVersion(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	migrator := NewMigrator(db)

	migration1 := &Migration{
		Version: 1,
		Name:    "test1",
	}
	migration2 := &Migration{
		Version: 1,
		Name:    "test2",
	}

	err := migrator.AddMigration(migration1)
	if err != nil {
		t.Errorf("Failed to add first migration: %v", err)
	}

	err = migrator.AddMigration(migration2)
	if err == nil {
		t.Error("Expected error for duplicate version")
	}
}

func TestMigrationUp(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	migrator := NewMigrator(db)

	executed := false
	migration := &Migration{
		Version:     1,
		Name:        "create_users",
		Description: "Create users collection",
		Up: func(db *database.Database) error {
			_, err := db.CreateCollection("users")
			executed = true
			return err
		},
	}

	err := migrator.AddMigration(migration)
	if err != nil {
		t.Fatalf("Failed to add migration: %v", err)
	}

	err = migrator.Up()
	if err != nil {
		t.Fatalf("Failed to run migration: %v", err)
	}

	if !executed {
		t.Error("Migration was not executed")
	}

	// Verify collection was created
	collections := db.ListCollections()
	found := false
	for _, name := range collections {
		if name == "users" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Users collection was not created")
	}

	// Verify migration was recorded
	history, err := migrator.GetMigrationHistory()
	if err != nil {
		t.Fatalf("Failed to get migration history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 history entry, got %d", len(history))
	}

	if history[0].Version != 1 || !history[0].Success {
		t.Error("Migration history is incorrect")
	}
}

func TestMigrationDown(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	migrator := NewMigrator(db)

	migration := &Migration{
		Version:     1,
		Name:        "create_users",
		Description: "Create users collection",
		Up: func(db *database.Database) error {
			_, err := db.CreateCollection("users")
			return err
		},
		Down: func(db *database.Database) error {
			return db.DropCollection("users")
		},
	}

	err := migrator.AddMigration(migration)
	if err != nil {
		t.Fatalf("Failed to add migration: %v", err)
	}

	// Apply migration
	err = migrator.Up()
	if err != nil {
		t.Fatalf("Failed to run migration: %v", err)
	}

	// Rollback migration
	err = migrator.Down()
	if err != nil {
		t.Fatalf("Failed to rollback migration: %v", err)
	}

	// Verify collection was dropped
	collections := db.ListCollections()
	for _, name := range collections {
		if name == "users" {
			t.Error("Users collection should have been dropped")
		}
	}
}

func TestMultipleMigrations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	migrator := NewMigrator(db)

	// Add multiple migrations
	migrations := []*Migration{
		{
			Version:     1,
			Name:        "create_users",
			Description: "Create users collection",
			Up: func(db *database.Database) error {
				_, err := db.CreateCollection("users")
				return err
			},
		},
		{
			Version:     2,
			Name:        "create_posts",
			Description: "Create posts collection",
			Up: func(db *database.Database) error {
				_, err := db.CreateCollection("posts")
				return err
			},
		},
		{
			Version:     3,
			Name:        "create_comments",
			Description: "Create comments collection",
			Up: func(db *database.Database) error {
				_, err := db.CreateCollection("comments")
				return err
			},
		},
	}

	for _, m := range migrations {
		if err := migrator.AddMigration(m); err != nil {
			t.Fatalf("Failed to add migration: %v", err)
		}
	}

	// Apply all migrations
	err := migrator.Up()
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Verify all collections were created
	collections := db.ListCollections()
	expectedCollections := []string{"users", "posts", "comments"}

	for _, expected := range expectedCollections {
		found := false
		for _, name := range collections {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected collection %s was not created", expected)
		}
	}

	// Verify migration history
	history, err := migrator.GetMigrationHistory()
	if err != nil {
		t.Fatalf("Failed to get migration history: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("Expected 3 history entries, got %d", len(history))
	}
}

func TestMigrationStatus(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	migrator := NewMigrator(db)

	// Add migrations
	migrations := []*Migration{
		{
			Version:     1,
			Name:        "migration1",
			Description: "First migration",
			Up:          func(db *database.Database) error { return nil },
		},
		{
			Version:     2,
			Name:        "migration2",
			Description: "Second migration",
			Up:          func(db *database.Database) error { return nil },
		},
	}

	for _, m := range migrations {
		if err := migrator.AddMigration(m); err != nil {
			t.Fatalf("Failed to add migration: %v", err)
		}
	}

	// Get initial status
	status, err := migrator.Status()
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if status.TotalMigrations != 2 {
		t.Errorf("Expected 2 total migrations, got %d", status.TotalMigrations)
	}

	if status.AppliedMigrations != 0 {
		t.Errorf("Expected 0 applied migrations, got %d", status.AppliedMigrations)
	}

	if status.PendingMigrations != 2 {
		t.Errorf("Expected 2 pending migrations, got %d", status.PendingMigrations)
	}

	// Apply first migration manually
	if err := migrator.applyMigration(migrations[0], true); err != nil {
		t.Fatalf("Failed to apply migration: %v", err)
	}

	// Get updated status
	status, err = migrator.Status()
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if status.AppliedMigrations != 1 {
		t.Errorf("Expected 1 applied migration, got %d", status.AppliedMigrations)
	}

	if status.PendingMigrations != 1 {
		t.Errorf("Expected 1 pending migration, got %d", status.PendingMigrations)
	}
}

func TestGetPendingMigrations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	migrator := NewMigrator(db)

	// Add migrations
	migrations := []*Migration{
		{
			Version:     1,
			Name:        "migration1",
			Description: "First migration",
			Up:          func(db *database.Database) error { return nil },
		},
		{
			Version:     2,
			Name:        "migration2",
			Description: "Second migration",
			Up:          func(db *database.Database) error { return nil },
		},
	}

	for _, m := range migrations {
		if err := migrator.AddMigration(m); err != nil {
			t.Fatalf("Failed to add migration: %v", err)
		}
	}

	// Get pending migrations (all should be pending)
	pending, err := migrator.GetPendingMigrations()
	if err != nil {
		t.Fatalf("Failed to get pending migrations: %v", err)
	}

	if len(pending) != 2 {
		t.Errorf("Expected 2 pending migrations, got %d", len(pending))
	}

	// Apply first migration
	if err := migrator.applyMigration(migrations[0], true); err != nil {
		t.Fatalf("Failed to apply migration: %v", err)
	}

	// Get pending migrations (only second should be pending)
	pending, err = migrator.GetPendingMigrations()
	if err != nil {
		t.Fatalf("Failed to get pending migrations: %v", err)
	}

	if len(pending) != 1 {
		t.Errorf("Expected 1 pending migration, got %d", len(pending))
	}

	if pending[0].Version != 2 {
		t.Errorf("Expected pending migration version 2, got %d", pending[0].Version)
	}
}

func TestSaveMigrationToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test_migration.json")

	migration := &Migration{
		Version:     time.Now().Unix(),
		Name:        "test_migration",
		Description: "Test migration description",
		UpScript: map[string]interface{}{
			"operations": []interface{}{
				map[string]interface{}{
					"type": "create_collection",
					"name": "users",
				},
			},
		},
		DownScript: map[string]interface{}{
			"operations": []interface{}{
				map[string]interface{}{
					"type": "drop_collection",
					"name": "users",
				},
			},
		},
	}

	err := SaveMigrationToFile(migration, path)
	if err != nil {
		t.Fatalf("Failed to save migration: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Migration file was not created")
	}
}

func TestLoadMigrationFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test_migration.json")

	originalMigration := &Migration{
		Version:     time.Now().Unix(),
		Name:        "test_migration",
		Description: "Test migration description",
		UpScript: map[string]interface{}{
			"operations": []interface{}{
				map[string]interface{}{
					"type": "create_collection",
					"name": "users",
				},
			},
		},
		DownScript: map[string]interface{}{
			"operations": []interface{}{
				map[string]interface{}{
					"type": "drop_collection",
					"name": "users",
				},
			},
		},
	}

	err := SaveMigrationToFile(originalMigration, path)
	if err != nil {
		t.Fatalf("Failed to save migration: %v", err)
	}

	loadedMigration, err := LoadMigrationFromFile(path)
	if err != nil {
		t.Fatalf("Failed to load migration: %v", err)
	}

	if loadedMigration.Version != originalMigration.Version {
		t.Errorf("Expected version %d, got %d", originalMigration.Version, loadedMigration.Version)
	}

	if loadedMigration.Name != originalMigration.Name {
		t.Errorf("Expected name %s, got %s", originalMigration.Name, loadedMigration.Name)
	}

	if loadedMigration.Description != originalMigration.Description {
		t.Errorf("Expected description %s, got %s", originalMigration.Description, loadedMigration.Description)
	}
}

func TestLoadMigrationsFromDir(t *testing.T) {
	dir := t.TempDir()

	// Create multiple migration files
	migrations := []*Migration{
		{
			Version:     1,
			Name:        "migration1",
			Description: "First migration",
			UpScript:    map[string]interface{}{"operations": []interface{}{}},
			DownScript:  map[string]interface{}{"operations": []interface{}{}},
		},
		{
			Version:     2,
			Name:        "migration2",
			Description: "Second migration",
			UpScript:    map[string]interface{}{"operations": []interface{}{}},
			DownScript:  map[string]interface{}{"operations": []interface{}{}},
		},
	}

	for i, m := range migrations {
		path := filepath.Join(dir, fmt.Sprintf("migration_%d.json", i+1))
		if err := SaveMigrationToFile(m, path); err != nil {
			t.Fatalf("Failed to save migration: %v", err)
		}
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	migrator := NewMigrator(db)
	err := migrator.LoadMigrationsFromDir(dir)
	if err != nil {
		t.Fatalf("Failed to load migrations: %v", err)
	}

	if len(migrator.migrations) != 2 {
		t.Errorf("Expected 2 migrations, got %d", len(migrator.migrations))
	}

	// Verify migrations are sorted by version
	if migrator.migrations[0].Version > migrator.migrations[1].Version {
		t.Error("Migrations are not sorted by version")
	}
}

func TestExecuteCreateCollection(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	op := map[string]interface{}{
		"type": "create_collection",
		"name": "users",
	}

	err := executeOperation(db, op)
	if err != nil {
		t.Fatalf("Failed to execute operation: %v", err)
	}

	collections := db.ListCollections()
	found := false
	for _, name := range collections {
		if name == "users" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Collection was not created")
	}
}

func TestExecuteDropCollection(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create collection first
	_, err := db.CreateCollection("users")
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	op := map[string]interface{}{
		"type": "drop_collection",
		"name": "users",
	}

	err = executeOperation(db, op)
	if err != nil {
		t.Fatalf("Failed to execute operation: %v", err)
	}

	collections := db.ListCollections()
	for _, name := range collections {
		if name == "users" {
			t.Error("Collection should have been dropped")
		}
	}
}

func TestExecuteCreateIndex(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create collection
	coll := db.Collection("users")

	op := map[string]interface{}{
		"type":       "create_index",
		"collection": "users",
		"field":      "email",
		"unique":     true,
	}

	err := executeOperation(db, op)
	if err != nil {
		t.Fatalf("Failed to execute operation: %v", err)
	}

	// Verify index was created
	indexes := coll.ListIndexes()
	found := false
	for _, idx := range indexes {
		if name, ok := idx["name"].(string); ok && name == "email_1" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Index was not created")
	}
}

func TestExecuteRenameCollection(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create collection
	_, err := db.CreateCollection("users")
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	op := map[string]interface{}{
		"type":     "rename_collection",
		"old_name": "users",
		"new_name": "people",
	}

	err = executeOperation(db, op)
	if err != nil {
		t.Fatalf("Failed to execute operation: %v", err)
	}

	collections := db.ListCollections()
	foundOld := false
	foundNew := false
	for _, name := range collections {
		if name == "users" {
			foundOld = true
		}
		if name == "people" {
			foundNew = true
		}
	}

	if foundOld {
		t.Error("Old collection name still exists")
	}

	if !foundNew {
		t.Error("New collection name was not found")
	}
}

func TestExecuteInsertDocuments(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	coll := db.Collection("users")

	op := map[string]interface{}{
		"type":       "insert_documents",
		"collection": "users",
		"documents": []interface{}{
			map[string]interface{}{
				"name":  "Alice",
				"email": "alice@example.com",
			},
			map[string]interface{}{
				"name":  "Bob",
				"email": "bob@example.com",
			},
		},
	}

	err := executeOperation(db, op)
	if err != nil {
		t.Fatalf("Failed to execute operation: %v", err)
	}

	// Verify documents were inserted
	docs, err := coll.Find(nil)
	if err != nil {
		t.Fatalf("Failed to find documents: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(docs))
	}
}

func TestExecuteUpdateDocuments(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	coll := db.Collection("users")

	// Insert a document
	_, err := coll.InsertOne(map[string]interface{}{
		"name":  "Alice",
		"email": "alice@example.com",
	})
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	op := map[string]interface{}{
		"type":       "update_documents",
		"collection": "users",
		"filter": map[string]interface{}{
			"name": "Alice",
		},
		"update": map[string]interface{}{
			"$set": map[string]interface{}{
				"email": "alice.new@example.com",
			},
		},
	}

	err = executeOperation(db, op)
	if err != nil {
		t.Fatalf("Failed to execute operation: %v", err)
	}

	// Verify document was updated
	docs, err := coll.Find(map[string]interface{}{"name": "Alice"})
	if err != nil {
		t.Fatalf("Failed to find documents: %v", err)
	}

	if len(docs) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(docs))
	}

	docMap := docs[0].ToMap()
	if docMap["email"] != "alice.new@example.com" {
		t.Errorf("Expected updated email, got %v", docMap["email"])
	}
}

func TestExecuteDeleteDocuments(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	coll := db.Collection("users")

	// Insert documents
	_, err := coll.InsertOne(map[string]interface{}{
		"name":  "Alice",
		"email": "alice@example.com",
	})
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	op := map[string]interface{}{
		"type":       "delete_documents",
		"collection": "users",
		"filter": map[string]interface{}{
			"name": "Alice",
		},
	}

	err = executeOperation(db, op)
	if err != nil {
		t.Fatalf("Failed to execute operation: %v", err)
	}

	// Verify document was deleted
	docs, err := coll.Find(map[string]interface{}{"name": "Alice"})
	if err != nil {
		t.Fatalf("Failed to find documents: %v", err)
	}

	if len(docs) != 0 {
		t.Errorf("Expected 0 documents, got %d", len(docs))
	}
}

func TestMigrationWithScriptOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	migrator := NewMigrator(db)

	migration := &Migration{
		Version:     1,
		Name:        "setup_database",
		Description: "Setup initial database structure",
		UpScript: map[string]interface{}{
			"operations": []interface{}{
				map[string]interface{}{
					"type": "create_collection",
					"name": "users",
				},
				map[string]interface{}{
					"type":       "create_index",
					"collection": "users",
					"field":      "email",
					"unique":     true,
				},
			},
		},
		DownScript: map[string]interface{}{
			"operations": []interface{}{
				map[string]interface{}{
					"type": "drop_collection",
					"name": "users",
				},
			},
		},
	}

	// Convert scripts to functions
	migration.Up = createMigrationFunc(migration.UpScript)
	migration.Down = createMigrationFunc(migration.DownScript)

	err := migrator.AddMigration(migration)
	if err != nil {
		t.Fatalf("Failed to add migration: %v", err)
	}

	// Apply migration
	err = migrator.Up()
	if err != nil {
		t.Fatalf("Failed to run migration: %v", err)
	}

	// Verify collection was created
	collections := db.ListCollections()
	found := false
	for _, name := range collections {
		if name == "users" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Users collection was not created")
	}

	// Verify index was created
	coll := db.Collection("users")
	indexes := coll.ListIndexes()
	indexFound := false
	for _, idx := range indexes {
		if name, ok := idx["name"].(string); ok && name == "email_1" {
			indexFound = true
			break
		}
	}

	if !indexFound {
		t.Error("Email index was not created")
	}
}

func TestRenameCollection(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a collection
	_, err := db.CreateCollection("users")
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	// Rename the collection
	err = db.RenameCollection("users", "people")
	if err != nil {
		t.Fatalf("Failed to rename collection: %v", err)
	}

	// Verify old name doesn't exist
	collections := db.ListCollections()
	for _, name := range collections {
		if name == "users" {
			t.Error("Old collection name still exists")
		}
	}

	// Verify new name exists
	found := false
	for _, name := range collections {
		if name == "people" {
			found = true
			break
		}
	}

	if !found {
		t.Error("New collection name was not found")
	}
}

func TestRenameCollectionErrors(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Test renaming non-existent collection
	err := db.RenameCollection("nonexistent", "new")
	if err == nil {
		t.Error("Expected error when renaming non-existent collection")
	}

	// Create two collections
	_, err = db.CreateCollection("users")
	if err != nil {
		t.Fatalf("Failed to create first collection: %v", err)
	}

	_, err = db.CreateCollection("people")
	if err != nil {
		t.Fatalf("Failed to create second collection: %v", err)
	}

	// Test renaming to existing name
	err = db.RenameCollection("users", "people")
	if err == nil {
		t.Error("Expected error when renaming to existing collection name")
	}
}

// Test executeDropIndex (0% coverage)
func TestExecuteDropIndex(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	coll := db.Collection("users")

	// Create an index first
	err := coll.CreateIndex("email", false)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	// Drop the index
	op := map[string]interface{}{
		"type":       "drop_index",
		"collection": "users",
		"name":       "email_1",
	}

	err = executeOperation(db, op)
	if err != nil {
		t.Fatalf("Failed to execute drop index operation: %v", err)
	}

	// Verify index was dropped
	indexes := coll.ListIndexes()
	for _, idx := range indexes {
		if name, ok := idx["name"].(string); ok && name == "email_1" {
			t.Error("Index should have been dropped")
		}
	}
}

// Test executeDropIndex error paths
func TestExecuteDropIndexErrors(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name string
		op   map[string]interface{}
	}{
		{
			name: "missing collection",
			op: map[string]interface{}{
				"type": "drop_index",
				"name": "email_1",
			},
		},
		{
			name: "missing index name",
			op: map[string]interface{}{
				"type":       "drop_index",
				"collection": "users",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executeOperation(db, tt.op)
			if err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

// Test executeCreateIndex with text index
func TestExecuteCreateTextIndex(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.Collection("articles") // Ensure collection exists

	op := map[string]interface{}{
		"type":       "create_index",
		"collection": "articles",
		"field":      "content",
		"index_type": "text",
	}

	err := executeOperation(db, op)
	if err != nil {
		t.Fatalf("Failed to create text index: %v", err)
	}

	// Verify text index was created
	coll := db.Collection("articles")
	indexes := coll.ListIndexes()
	found := false
	for _, idx := range indexes {
		if indexType, ok := idx["type"].(string); ok && indexType == "text" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Text index was not created")
	}
}

// Test executeCreateIndex with text index on multiple fields
func TestExecuteCreateTextIndexMultipleFields(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.Collection("articles") // Ensure collection exists

	op := map[string]interface{}{
		"type":       "create_index",
		"collection": "articles",
		"field":      "content",
		"index_type": "text",
		"fields":     []interface{}{"title", "content", "tags"},
	}

	err := executeOperation(db, op)
	if err != nil {
		t.Fatalf("Failed to create multi-field text index: %v", err)
	}

	// Verify text index was created
	coll := db.Collection("articles")
	indexes := coll.ListIndexes()
	found := false
	for _, idx := range indexes {
		if indexType, ok := idx["type"].(string); ok && indexType == "text" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Multi-field text index was not created")
	}
}

// Test executeCreateIndex with geo_2d index
func TestExecuteCreateGeo2DIndex(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.Collection("locations") // Ensure collection exists

	op := map[string]interface{}{
		"type":       "create_index",
		"collection": "locations",
		"field":      "position",
		"index_type": "geo_2d",
	}

	err := executeOperation(db, op)
	if err != nil {
		t.Fatalf("Failed to create geo_2d index: %v", err)
	}

	// Verify geo index was created
	coll := db.Collection("locations")
	indexes := coll.ListIndexes()
	found := false
	for _, idx := range indexes {
		if indexType, ok := idx["type"].(string); ok && indexType == "2d" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Geo 2D index was not created")
	}
}

// Test executeCreateIndex with geo_2dsphere index
func TestExecuteCreateGeo2DSphereIndex(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.Collection("places") // Ensure collection exists

	op := map[string]interface{}{
		"type":       "create_index",
		"collection": "places",
		"field":      "location",
		"index_type": "geo_2dsphere",
	}

	err := executeOperation(db, op)
	if err != nil {
		t.Fatalf("Failed to create geo_2dsphere index: %v", err)
	}

	// Verify geo sphere index was created
	coll := db.Collection("places")
	indexes := coll.ListIndexes()
	found := false
	for _, idx := range indexes {
		if indexType, ok := idx["type"].(string); ok && indexType == "2dsphere" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Geo 2DSphere index was not created")
	}
}

// Test executeCreateIndex error paths
func TestExecuteCreateIndexErrors(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name string
		op   map[string]interface{}
	}{
		{
			name: "missing collection",
			op: map[string]interface{}{
				"type":  "create_index",
				"field": "email",
			},
		},
		{
			name: "missing field",
			op: map[string]interface{}{
				"type":       "create_index",
				"collection": "users",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executeOperation(db, tt.op)
			if err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

// Test executeCreateCollection error path
func TestExecuteCreateCollectionError(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	op := map[string]interface{}{
		"type": "create_collection",
		// Missing "name" field
	}

	err := executeOperation(db, op)
	if err == nil {
		t.Error("Expected error for missing collection name")
	}
}

// Test executeDropCollection error path
func TestExecuteDropCollectionError(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	op := map[string]interface{}{
		"type": "drop_collection",
		// Missing "name" field
	}

	err := executeOperation(db, op)
	if err == nil {
		t.Error("Expected error for missing collection name")
	}
}

// Test executeRenameCollection error paths
func TestExecuteRenameCollectionErrors(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name string
		op   map[string]interface{}
	}{
		{
			name: "missing old_name",
			op: map[string]interface{}{
				"type":     "rename_collection",
				"new_name": "people",
			},
		},
		{
			name: "missing new_name",
			op: map[string]interface{}{
				"type":     "rename_collection",
				"old_name": "users",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executeOperation(db, tt.op)
			if err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

// Test executeUpdateDocuments error paths
func TestExecuteUpdateDocumentsErrors(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name string
		op   map[string]interface{}
	}{
		{
			name: "missing collection",
			op: map[string]interface{}{
				"type":   "update_documents",
				"filter": map[string]interface{}{"name": "Alice"},
				"update": map[string]interface{}{"$set": map[string]interface{}{"age": int64(30)}},
			},
		},
		{
			name: "missing filter",
			op: map[string]interface{}{
				"type":       "update_documents",
				"collection": "users",
				"update":     map[string]interface{}{"$set": map[string]interface{}{"age": int64(30)}},
			},
		},
		{
			name: "missing update",
			op: map[string]interface{}{
				"type":       "update_documents",
				"collection": "users",
				"filter":     map[string]interface{}{"name": "Alice"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executeOperation(db, tt.op)
			if err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

// Test executeDeleteDocuments error paths
func TestExecuteDeleteDocumentsErrors(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name string
		op   map[string]interface{}
	}{
		{
			name: "missing collection",
			op: map[string]interface{}{
				"type":   "delete_documents",
				"filter": map[string]interface{}{"name": "Alice"},
			},
		},
		{
			name: "missing filter",
			op: map[string]interface{}{
				"type":       "delete_documents",
				"collection": "users",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executeOperation(db, tt.op)
			if err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

// Test executeInsertDocuments error paths
func TestExecuteInsertDocumentsErrors(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name        string
		op          map[string]interface{}
		expectError bool
	}{
		{
			name: "missing collection",
			op: map[string]interface{}{
				"type":      "insert_documents",
				"documents": []interface{}{map[string]interface{}{"name": "Alice"}},
			},
			expectError: true,
		},
		{
			name: "missing documents",
			op: map[string]interface{}{
				"type":       "insert_documents",
				"collection": "users",
			},
			expectError: true,
		},
		{
			name: "invalid document in array",
			op: map[string]interface{}{
				"type":       "insert_documents",
				"collection": "users",
				"documents":  []interface{}{"not a map", int64(123)}, // Invalid documents are skipped
			},
			expectError: false, // executeInsertDocuments continues on invalid documents
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executeOperation(db, tt.op)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// Test executeOperation with unknown type
func TestExecuteOperationUnknownType(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	op := map[string]interface{}{
		"type": "unknown_operation",
	}

	err := executeOperation(db, op)
	if err == nil {
		t.Error("Expected error for unknown operation type")
	}
}

// Test executeOperation with missing type
func TestExecuteOperationMissingType(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	op := map[string]interface{}{
		"name": "test",
	}

	err := executeOperation(db, op)
	if err == nil {
		t.Error("Expected error for missing operation type")
	}
}

// Test createMigrationFunc with nil script
func TestCreateMigrationFuncNilScript(t *testing.T) {
	fn := createMigrationFunc(nil)
	if fn == nil {
		t.Fatal("Expected function, got nil")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := fn(db)
	if err != nil {
		t.Errorf("Expected nil error for nil script, got %v", err)
	}
}

// Test createMigrationFunc with empty operations
func TestCreateMigrationFuncEmptyOperations(t *testing.T) {
	script := map[string]interface{}{
		"operations": []interface{}{},
	}

	fn := createMigrationFunc(script)
	if fn == nil {
		t.Fatal("Expected function, got nil")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := fn(db)
	if err != nil {
		t.Errorf("Expected nil error for empty operations, got %v", err)
	}
}

// Test createMigrationFunc with invalid operation
func TestCreateMigrationFuncInvalidOperation(t *testing.T) {
	script := map[string]interface{}{
		"operations": []interface{}{
			"not a map", // Invalid operation
			map[string]interface{}{
				"type": "create_collection",
				"name": "test",
			},
		},
	}

	fn := createMigrationFunc(script)
	if fn == nil {
		t.Fatal("Expected function, got nil")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Should succeed (invalid operation is skipped, valid one is executed)
	err := fn(db)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify valid operation was executed
	collections := db.ListCollections()
	found := false
	for _, name := range collections {
		if name == "test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Valid operation after invalid one was not executed")
	}
}

// Test Up with error in migration
func TestUpWithError(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	migrator := NewMigrator(db)

	migration := &Migration{
		Version:     1,
		Name:        "failing_migration",
		Description: "Migration that fails",
		Up: func(db *database.Database) error {
			return fmt.Errorf("migration failed intentionally")
		},
	}

	err := migrator.AddMigration(migration)
	if err != nil {
		t.Fatalf("Failed to add migration: %v", err)
	}

	err = migrator.Up()
	if err == nil {
		t.Error("Expected error from failing migration")
	}

	// Verify migration was recorded as failed
	history, err := migrator.GetMigrationHistory()
	if err != nil {
		t.Fatalf("Failed to get migration history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 history entry, got %d", len(history))
	}

	if history[0].Success {
		t.Error("Failed migration should not be marked as success")
	}

	if history[0].Error == "" {
		t.Error("Failed migration should have error message")
	}
}

// Test Down with no migrations to rollback
func TestDownWithNoMigrations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	migrator := NewMigrator(db)

	err := migrator.Down()
	if err == nil {
		t.Error("Expected error when no migrations to roll back")
	}
}

// Test Down with error in migration
func TestDownWithError(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	migrator := NewMigrator(db)

	migration := &Migration{
		Version:     1,
		Name:        "test_migration",
		Description: "Test migration",
		Up: func(db *database.Database) error {
			_, err := db.CreateCollection("test")
			return err
		},
		Down: func(db *database.Database) error {
			return fmt.Errorf("rollback failed intentionally")
		},
	}

	err := migrator.AddMigration(migration)
	if err != nil {
		t.Fatalf("Failed to add migration: %v", err)
	}

	// Apply migration
	err = migrator.Up()
	if err != nil {
		t.Fatalf("Failed to run migration: %v", err)
	}

	// Try to rollback (should fail)
	err = migrator.Down()
	if err == nil {
		t.Error("Expected error from failing rollback")
	}
}

// Test LoadMigrationsFromDir with non-existent directory
func TestLoadMigrationsFromDirNonExistent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	migrator := NewMigrator(db)

	err := migrator.LoadMigrationsFromDir("/path/that/does/not/exist")
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
}

// Test LoadMigrationsFromDir with invalid JSON file
func TestLoadMigrationsFromDirInvalidJSON(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	dir := t.TempDir()

	// Create invalid JSON file
	invalidPath := filepath.Join(dir, "invalid.json")
	err := os.WriteFile(invalidPath, []byte("invalid json content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid JSON file: %v", err)
	}

	migrator := NewMigrator(db)
	err = migrator.LoadMigrationsFromDir(dir)
	if err == nil {
		t.Error("Expected error for invalid JSON file")
	}
}

// Test LoadMigrationsFromDir ignores non-JSON files
func TestLoadMigrationsFromDirIgnoresNonJSON(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	dir := t.TempDir()

	// Create a non-JSON file
	txtPath := filepath.Join(dir, "readme.txt")
	err := os.WriteFile(txtPath, []byte("Some text"), 0644)
	if err != nil {
		t.Fatalf("Failed to write text file: %v", err)
	}

	// Create a valid migration file
	migration := &Migration{
		Version:     1,
		Name:        "test",
		Description: "Test",
		UpScript:    map[string]interface{}{"operations": []interface{}{}},
		DownScript:  map[string]interface{}{"operations": []interface{}{}},
	}
	jsonPath := filepath.Join(dir, "migration.json")
	err = SaveMigrationToFile(migration, jsonPath)
	if err != nil {
		t.Fatalf("Failed to save migration: %v", err)
	}

	migrator := NewMigrator(db)
	err = migrator.LoadMigrationsFromDir(dir)
	if err != nil {
		t.Fatalf("Failed to load migrations: %v", err)
	}

	// Should only have loaded the JSON file
	if len(migrator.migrations) != 1 {
		t.Errorf("Expected 1 migration, got %d", len(migrator.migrations))
	}
}

// Test Up skips already applied migrations
func TestUpSkipsAppliedMigrations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	migrator := NewMigrator(db)

	executionCount := 0

	migration := &Migration{
		Version:     1,
		Name:        "test_migration",
		Description: "Test migration",
		Up: func(db *database.Database) error {
			executionCount++
			return nil
		},
	}

	err := migrator.AddMigration(migration)
	if err != nil {
		t.Fatalf("Failed to add migration: %v", err)
	}

	// Run Up twice
	err = migrator.Up()
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	err = migrator.Up()
	if err != nil {
		t.Fatalf("Failed to run migrations second time: %v", err)
	}

	// Migration should have been executed only once
	if executionCount != 1 {
		t.Errorf("Expected migration to be executed once, got %d times", executionCount)
	}
}
