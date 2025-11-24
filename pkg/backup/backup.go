package backup

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/index"
)

// BackupFormat represents the format of a backup file
type BackupFormat struct {
	Version      string                 `json:"version"`
	Timestamp    time.Time              `json:"timestamp"`
	DatabaseName string                 `json:"database_name"`
	Collections  []CollectionBackup     `json:"collections"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// CollectionBackup represents a backed-up collection
type CollectionBackup struct {
	Name      string            `json:"name"`
	Documents []DocumentBackup  `json:"documents"`
	Indexes   []IndexBackup     `json:"indexes"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// DocumentBackup represents a backed-up document
type DocumentBackup struct {
	ID     string                 `json:"_id"`
	Fields map[string]interface{} `json:"fields"`
}

// IndexBackup represents a backed-up index definition
type IndexBackup struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // "btree", "text", "geo", "ttl"
	FieldPaths  []string               `json:"field_paths"`
	Unique      bool                   `json:"unique"`
	Sparse      bool                   `json:"sparse,omitempty"`
	TTLDuration *int64                 `json:"ttl_duration,omitempty"` // Seconds for TTL indexes
	GeoType     string                 `json:"geo_type,omitempty"`     // "2d" or "2dsphere"
	Filter      map[string]interface{} `json:"filter,omitempty"`       // For partial indexes
	Config      map[string]interface{} `json:"config,omitempty"`       // Additional config
}

// Backuper creates database backups
type Backuper struct {
	Pretty bool // Enable pretty-printing for JSON
}

// NewBackuper creates a new backuper
func NewBackuper(pretty bool) *Backuper {
	return &Backuper{Pretty: pretty}
}

// BackupToFile creates a backup file at the specified path
func (b *Backuper) BackupToFile(path string, backup *BackupFormat) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create file
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	return b.BackupToWriter(file, backup)
}

// BackupToWriter writes backup to an io.Writer
func (b *Backuper) BackupToWriter(writer io.Writer, backup *BackupFormat) error {
	encoder := json.NewEncoder(writer)
	if b.Pretty {
		encoder.SetIndent("", "  ")
	}

	if err := encoder.Encode(backup); err != nil {
		return fmt.Errorf("failed to encode backup: %w", err)
	}

	return nil
}

// NewBackupFormat creates a new backup format structure
func NewBackupFormat(databaseName string) *BackupFormat {
	return &BackupFormat{
		Version:      "1.0",
		Timestamp:    time.Now(),
		DatabaseName: databaseName,
		Collections:  make([]CollectionBackup, 0),
		Metadata:     make(map[string]interface{}),
	}
}

// AddCollection adds a collection to the backup
func (bf *BackupFormat) AddCollection(name string, docs []*document.Document, indexes []IndexBackup) {
	// Convert documents to backup format
	docBackups := make([]DocumentBackup, 0, len(docs))
	for _, doc := range docs {
		docBackup := convertDocumentToBackup(doc)
		docBackups = append(docBackups, docBackup)
	}

	collBackup := CollectionBackup{
		Name:      name,
		Documents: docBackups,
		Indexes:   indexes,
		Metadata:  make(map[string]interface{}),
	}

	bf.Collections = append(bf.Collections, collBackup)
}

// convertDocumentToBackup converts a Document to DocumentBackup
func convertDocumentToBackup(doc *document.Document) DocumentBackup {
	docMap := doc.ToMap()

	// Extract _id
	var id string
	if idVal, exists := docMap["_id"]; exists {
		id = fmt.Sprintf("%v", convertValueForBackup(idVal))
		delete(docMap, "_id") // Remove from fields map
	}

	// Convert all field values
	fields := make(map[string]interface{})
	for key, value := range docMap {
		fields[key] = convertValueForBackup(value)
	}

	return DocumentBackup{
		ID:     id,
		Fields: fields,
	}
}

// convertValueForBackup converts document values to backup-compatible format
func convertValueForBackup(value interface{}) interface{} {
	switch v := value.(type) {
	case document.ObjectID:
		return v.Hex()
	case time.Time:
		return v.Format(time.RFC3339)
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, elem := range v {
			result[i] = convertValueForBackup(elem)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = convertValueForBackup(val)
		}
		return result
	default:
		return v
	}
}

// NewIndexBackup creates an index backup from index configuration
func NewIndexBackup(idx *index.Index) IndexBackup {
	backup := IndexBackup{
		Name:       idx.Name(),
		Type:       "btree",
		Unique:     idx.IsUnique(),
		Sparse:     false, // Default value
		Config:     make(map[string]interface{}),
	}

	// Handle single vs compound indexes
	if idx.IsCompound() {
		backup.FieldPaths = idx.FieldPaths()
	} else {
		backup.FieldPaths = []string{idx.FieldPath()}
	}

	// Add partial index filter if present
	if filter := idx.Filter(); filter != nil {
		backup.Filter = filter
	}

	return backup
}

// NewTextIndexBackup creates an index backup for a text index
func NewTextIndexBackup(name string, fieldPaths []string) IndexBackup {
	return IndexBackup{
		Name:       name,
		Type:       "text",
		FieldPaths: fieldPaths,
		Config:     make(map[string]interface{}),
	}
}

// NewGeoIndexBackup creates an index backup for a geo index
func NewGeoIndexBackup(name string, fieldPath string, geoType string) IndexBackup {
	return IndexBackup{
		Name:       name,
		Type:       "geo",
		FieldPaths: []string{fieldPath},
		GeoType:    geoType,
		Config:     make(map[string]interface{}),
	}
}

// NewTTLIndexBackup creates an index backup for a TTL index
func NewTTLIndexBackup(name string, fieldPath string, ttlSeconds int64) IndexBackup {
	return IndexBackup{
		Name:        name,
		Type:        "ttl",
		FieldPaths:  []string{fieldPath},
		TTLDuration: &ttlSeconds,
		Config:      make(map[string]interface{}),
	}
}

// Stats returns statistics about the backup
func (bf *BackupFormat) Stats() map[string]interface{} {
	totalDocs := 0
	totalIndexes := 0

	for _, coll := range bf.Collections {
		totalDocs += len(coll.Documents)
		totalIndexes += len(coll.Indexes)
	}

	return map[string]interface{}{
		"version":           bf.Version,
		"timestamp":         bf.Timestamp,
		"database_name":     bf.DatabaseName,
		"collections":       len(bf.Collections),
		"total_documents":   totalDocs,
		"total_indexes":     totalIndexes,
	}
}
