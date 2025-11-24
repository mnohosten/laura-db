package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/mnohosten/laura-db/pkg/database"
)

// Migration represents a single database migration
type Migration struct {
	Version     int64                  `json:"version"`     // Unix timestamp or sequence number
	Name        string                 `json:"name"`        // Descriptive name
	Description string                 `json:"description"` // What this migration does
	Up          MigrationFunc          `json:"-"`           // Function to apply migration
	Down        MigrationFunc          `json:"-"`           // Function to rollback migration
	UpScript    map[string]interface{} `json:"up_script"`   // JSON representation for file storage
	DownScript  map[string]interface{} `json:"down_script"` // JSON representation for file storage
}

// MigrationFunc is a function that performs a migration
type MigrationFunc func(db *database.Database) error

// MigrationHistory represents a record of applied migrations
type MigrationHistory struct {
	Version   int64     `json:"version"`
	Name      string    `json:"name"`
	AppliedAt time.Time `json:"applied_at"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}

// Migrator manages database migrations
type Migrator struct {
	db         *database.Database
	migrations []*Migration
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *database.Database) *Migrator {
	return &Migrator{
		db:         db,
		migrations: make([]*Migration, 0),
	}
}

// AddMigration adds a migration to the migrator
func (m *Migrator) AddMigration(migration *Migration) error {
	if migration.Version <= 0 {
		return fmt.Errorf("migration version must be positive")
	}
	if migration.Name == "" {
		return fmt.Errorf("migration name cannot be empty")
	}

	// Check for duplicate version
	for _, existing := range m.migrations {
		if existing.Version == migration.Version {
			return fmt.Errorf("migration with version %d already exists", migration.Version)
		}
	}

	m.migrations = append(m.migrations, migration)
	return nil
}

// LoadMigrationsFromDir loads migration files from a directory
func (m *Migrator) LoadMigrationsFromDir(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read migration directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		path := filepath.Join(dir, file.Name())
		migration, err := LoadMigrationFromFile(path)
		if err != nil {
			return fmt.Errorf("failed to load migration from %s: %w", path, err)
		}

		if err := m.AddMigration(migration); err != nil {
			return fmt.Errorf("failed to add migration from %s: %w", path, err)
		}
	}

	// Sort migrations by version
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Version < m.migrations[j].Version
	})

	return nil
}

// LoadMigrationFromFile loads a migration from a JSON file
func LoadMigrationFromFile(path string) (*Migration, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration file: %w", err)
	}

	var migration Migration
	if err := json.Unmarshal(data, &migration); err != nil {
		return nil, fmt.Errorf("failed to parse migration file: %w", err)
	}

	// Convert JSON scripts to functions
	migration.Up = createMigrationFunc(migration.UpScript)
	migration.Down = createMigrationFunc(migration.DownScript)

	return &migration, nil
}

// SaveMigrationToFile saves a migration to a JSON file
func SaveMigrationToFile(migration *Migration, path string) error {
	data, err := json.MarshalIndent(migration, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal migration: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write migration file: %w", err)
	}

	return nil
}

// createMigrationFunc creates a migration function from a script map
func createMigrationFunc(script map[string]interface{}) MigrationFunc {
	return func(db *database.Database) error {
		if script == nil {
			return nil
		}

		// Execute migration operations based on script
		if ops, ok := script["operations"].([]interface{}); ok {
			for _, op := range ops {
				opMap, ok := op.(map[string]interface{})
				if !ok {
					continue
				}

				if err := executeOperation(db, opMap); err != nil {
					return err
				}
			}
		}

		return nil
	}
}

// executeOperation executes a single migration operation
func executeOperation(db *database.Database, op map[string]interface{}) error {
	opType, ok := op["type"].(string)
	if !ok {
		return fmt.Errorf("operation type not specified")
	}

	switch opType {
	case "create_collection":
		return executeCreateCollection(db, op)
	case "drop_collection":
		return executeDropCollection(db, op)
	case "create_index":
		return executeCreateIndex(db, op)
	case "drop_index":
		return executeDropIndex(db, op)
	case "rename_collection":
		return executeRenameCollection(db, op)
	case "update_documents":
		return executeUpdateDocuments(db, op)
	case "delete_documents":
		return executeDeleteDocuments(db, op)
	case "insert_documents":
		return executeInsertDocuments(db, op)
	default:
		return fmt.Errorf("unknown operation type: %s", opType)
	}
}

// executeCreateCollection creates a new collection
func executeCreateCollection(db *database.Database, op map[string]interface{}) error {
	name, ok := op["name"].(string)
	if !ok {
		return fmt.Errorf("collection name not specified")
	}

	_, err := db.CreateCollection(name)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}

// executeDropCollection drops a collection
func executeDropCollection(db *database.Database, op map[string]interface{}) error {
	name, ok := op["name"].(string)
	if !ok {
		return fmt.Errorf("collection name not specified")
	}

	err := db.DropCollection(name)
	if err != nil {
		return fmt.Errorf("failed to drop collection: %w", err)
	}

	return nil
}

// executeCreateIndex creates an index
func executeCreateIndex(db *database.Database, op map[string]interface{}) error {
	collection, ok := op["collection"].(string)
	if !ok {
		return fmt.Errorf("collection name not specified")
	}

	field, ok := op["field"].(string)
	if !ok {
		return fmt.Errorf("field name not specified")
	}

	coll := db.Collection(collection)

	// Check for index type
	indexType, _ := op["index_type"].(string)
	switch indexType {
	case "text":
		fields := []string{field}
		if fieldList, ok := op["fields"].([]interface{}); ok {
			fields = make([]string, len(fieldList))
			for i, f := range fieldList {
				fields[i] = f.(string)
			}
		}
		return coll.CreateTextIndex(fields)
	case "geo_2d":
		return coll.Create2DIndex(field)
	case "geo_2dsphere":
		return coll.Create2DSphereIndex(field)
	default:
		// Regular B+ tree index
		unique := false
		if u, ok := op["unique"].(bool); ok {
			unique = u
		}
		return coll.CreateIndex(field, unique)
	}
}

// executeDropIndex drops an index
func executeDropIndex(db *database.Database, op map[string]interface{}) error {
	collection, ok := op["collection"].(string)
	if !ok {
		return fmt.Errorf("collection name not specified")
	}

	name, ok := op["name"].(string)
	if !ok {
		return fmt.Errorf("index name not specified")
	}

	coll := db.Collection(collection)
	return coll.DropIndex(name)
}

// executeRenameCollection renames a collection
func executeRenameCollection(db *database.Database, op map[string]interface{}) error {
	oldName, ok := op["old_name"].(string)
	if !ok {
		return fmt.Errorf("old collection name not specified")
	}

	newName, ok := op["new_name"].(string)
	if !ok {
		return fmt.Errorf("new collection name not specified")
	}

	return db.RenameCollection(oldName, newName)
}

// executeUpdateDocuments updates documents in a collection
func executeUpdateDocuments(db *database.Database, op map[string]interface{}) error {
	collection, ok := op["collection"].(string)
	if !ok {
		return fmt.Errorf("collection name not specified")
	}

	filter, ok := op["filter"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("filter not specified")
	}

	update, ok := op["update"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("update not specified")
	}

	coll := db.Collection(collection)
	_, err := coll.UpdateMany(filter, update)
	return err
}

// executeDeleteDocuments deletes documents from a collection
func executeDeleteDocuments(db *database.Database, op map[string]interface{}) error {
	collection, ok := op["collection"].(string)
	if !ok {
		return fmt.Errorf("collection name not specified")
	}

	filter, ok := op["filter"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("filter not specified")
	}

	coll := db.Collection(collection)
	_, err := coll.DeleteMany(filter)
	return err
}

// executeInsertDocuments inserts documents into a collection
func executeInsertDocuments(db *database.Database, op map[string]interface{}) error {
	collection, ok := op["collection"].(string)
	if !ok {
		return fmt.Errorf("collection name not specified")
	}

	documents, ok := op["documents"].([]interface{})
	if !ok {
		return fmt.Errorf("documents not specified")
	}

	coll := db.Collection(collection)

	for _, doc := range documents {
		docMap, ok := doc.(map[string]interface{})
		if !ok {
			continue
		}
		if _, err := coll.InsertOne(docMap); err != nil {
			return fmt.Errorf("failed to insert document: %w", err)
		}
	}

	return nil
}

// Up applies all pending migrations
func (m *Migrator) Up() error {
	appliedVersions, err := m.getAppliedVersions()
	if err != nil {
		return fmt.Errorf("failed to get applied versions: %w", err)
	}

	for _, migration := range m.migrations {
		if appliedVersions[migration.Version] {
			continue // Skip already applied migrations
		}

		if err := m.applyMigration(migration, true); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration.Name, err)
		}
	}

	return nil
}

// Down rolls back the last applied migration
func (m *Migrator) Down() error {
	appliedVersions, err := m.getAppliedVersions()
	if err != nil {
		return fmt.Errorf("failed to get applied versions: %w", err)
	}

	// Find the latest applied migration
	var latestMigration *Migration
	for i := len(m.migrations) - 1; i >= 0; i-- {
		if appliedVersions[m.migrations[i].Version] {
			latestMigration = m.migrations[i]
			break
		}
	}

	if latestMigration == nil {
		return fmt.Errorf("no migrations to roll back")
	}

	if err := m.applyMigration(latestMigration, false); err != nil {
		return fmt.Errorf("failed to roll back migration %s: %w", latestMigration.Name, err)
	}

	return nil
}

// applyMigration applies or rolls back a migration
func (m *Migrator) applyMigration(migration *Migration, up bool) error {
	history := &MigrationHistory{
		Version:   migration.Version,
		Name:      migration.Name,
		AppliedAt: time.Now(),
		Success:   false,
	}

	var err error
	if up {
		if migration.Up != nil {
			err = migration.Up(m.db)
		}
	} else {
		if migration.Down != nil {
			err = migration.Down(m.db)
		}
	}

	if err != nil {
		history.Error = err.Error()
		_ = m.recordHistory(history, !up) // Record failed attempt
		return err
	}

	history.Success = true
	return m.recordHistory(history, !up)
}

// getAppliedVersions returns a map of applied migration versions
func (m *Migrator) getAppliedVersions() (map[int64]bool, error) {
	coll := m.db.Collection("_migrations")

	docs, err := coll.Find(nil)
	if err != nil {
		return nil, err
	}

	versions := make(map[int64]bool)
	for _, doc := range docs {
		docMap := doc.ToMap()
		if version, ok := docMap["version"].(int64); ok {
			if success, ok := docMap["success"].(bool); ok && success {
				versions[version] = true
			}
		}
	}

	return versions, nil
}

// recordHistory records a migration in the history collection
func (m *Migrator) recordHistory(history *MigrationHistory, remove bool) error {
	coll := m.db.Collection("_migrations")

	if remove {
		// Remove the migration record on rollback
		_, err := coll.DeleteMany(map[string]interface{}{
			"version": history.Version,
		})
		return err
	}

	// Add the migration record
	historyDoc := map[string]interface{}{
		"version":    history.Version,
		"name":       history.Name,
		"applied_at": history.AppliedAt,
		"success":    history.Success,
	}
	if history.Error != "" {
		historyDoc["error"] = history.Error
	}

	_, err := coll.InsertOne(historyDoc)
	return err
}

// GetMigrationHistory returns the migration history
func (m *Migrator) GetMigrationHistory() ([]*MigrationHistory, error) {
	coll := m.db.Collection("_migrations")

	docs, err := coll.Find(nil)
	if err != nil {
		return nil, err
	}

	history := make([]*MigrationHistory, 0, len(docs))
	for _, doc := range docs {
		docMap := doc.ToMap()
		h := &MigrationHistory{}
		if v, ok := docMap["version"].(int64); ok {
			h.Version = v
		}
		if n, ok := docMap["name"].(string); ok {
			h.Name = n
		}
		if at, ok := docMap["applied_at"].(time.Time); ok {
			h.AppliedAt = at
		}
		if s, ok := docMap["success"].(bool); ok {
			h.Success = s
		}
		if e, ok := docMap["error"].(string); ok {
			h.Error = e
		}
		history = append(history, h)
	}

	return history, nil
}

// GetPendingMigrations returns migrations that have not been applied
func (m *Migrator) GetPendingMigrations() ([]*Migration, error) {
	appliedVersions, err := m.getAppliedVersions()
	if err != nil {
		return nil, err
	}

	pending := make([]*Migration, 0)
	for _, migration := range m.migrations {
		if !appliedVersions[migration.Version] {
			pending = append(pending, migration)
		}
	}

	return pending, nil
}

// Status returns the current migration status
func (m *Migrator) Status() (*MigrationStatus, error) {
	appliedVersions, err := m.getAppliedVersions()
	if err != nil {
		return nil, err
	}

	status := &MigrationStatus{
		TotalMigrations:   len(m.migrations),
		AppliedMigrations: len(appliedVersions),
		PendingMigrations: 0,
		Migrations:        make([]*MigrationStatusEntry, 0),
	}

	for _, migration := range m.migrations {
		entry := &MigrationStatusEntry{
			Version:     migration.Version,
			Name:        migration.Name,
			Description: migration.Description,
			Applied:     appliedVersions[migration.Version],
		}
		status.Migrations = append(status.Migrations, entry)
		if !entry.Applied {
			status.PendingMigrations++
		}
	}

	return status, nil
}

// MigrationStatus represents the current migration status
type MigrationStatus struct {
	TotalMigrations   int                      `json:"total_migrations"`
	AppliedMigrations int                      `json:"applied_migrations"`
	PendingMigrations int                      `json:"pending_migrations"`
	Migrations        []*MigrationStatusEntry  `json:"migrations"`
}

// MigrationStatusEntry represents the status of a single migration
type MigrationStatusEntry struct {
	Version     int64  `json:"version"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Applied     bool   `json:"applied"`
}

// CreateMigration creates a new migration with the given name
func CreateMigration(name, description string) *Migration {
	return &Migration{
		Version:     time.Now().Unix(),
		Name:        name,
		Description: description,
		UpScript: map[string]interface{}{
			"operations": []interface{}{},
		},
		DownScript: map[string]interface{}{
			"operations": []interface{}{},
		},
	}
}
