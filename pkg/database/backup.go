package database

import (
	"fmt"

	"github.com/mnohosten/laura-db/pkg/backup"
	"github.com/mnohosten/laura-db/pkg/index"
)

// Backup creates a backup of the entire database
func (db *Database) Backup() (*backup.BackupFormat, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if !db.isOpen {
		return nil, fmt.Errorf("database is closed")
	}

	// Create backup format
	backupFormat := backup.NewBackupFormat(db.name)

	// Backup each collection
	for name, coll := range db.collections {
		if err := db.backupCollection(backupFormat, name, coll); err != nil {
			return nil, fmt.Errorf("failed to backup collection %s: %w", name, err)
		}
	}

	return backupFormat, nil
}

// backupCollection backs up a single collection
func (db *Database) backupCollection(backupFormat *backup.BackupFormat, name string, coll *Collection) error {
	coll.mu.RLock()
	defer coll.mu.RUnlock()

	// Get all documents from document store
	docs, err := coll.getAllDocuments()
	if err != nil {
		return fmt.Errorf("failed to get documents: %w", err)
	}

	// Get all indexes
	indexes := make([]backup.IndexBackup, 0)

	// Backup B+ tree indexes (skip default _id_ index)
	for name, idx := range coll.indexes {
		if name == "_id_" {
			continue // Skip default index, it will be recreated automatically
		}
		indexBackup := backup.NewIndexBackup(idx)
		indexes = append(indexes, indexBackup)
	}

	// Backup text indexes
	for name, textIdx := range coll.textIndexes {
		indexBackup := backup.NewTextIndexBackup(name, textIdx.FieldPaths())
		indexes = append(indexes, indexBackup)
	}

	// Backup geo indexes
	for name, geoIdx := range coll.geoIndexes {
		var geoType string
		switch geoIdx.Type() {
		case index.IndexType2D:
			geoType = "2d"
		case index.IndexType2DSphere:
			geoType = "2dsphere"
		}
		indexBackup := backup.NewGeoIndexBackup(name, geoIdx.FieldPath(), geoType)
		indexes = append(indexes, indexBackup)
	}

	// Backup TTL indexes
	for name, ttlIdx := range coll.ttlIndexes {
		ttlSeconds := ttlIdx.TTLSeconds()
		indexBackup := backup.NewTTLIndexBackup(name, ttlIdx.FieldPath(), ttlSeconds)
		indexes = append(indexes, indexBackup)
	}

	// Add collection to backup
	backupFormat.AddCollection(name, docs, indexes)

	return nil
}

// Restore restores the database from a backup
func (db *Database) Restore(backupFormat *backup.BackupFormat, options *backup.RestoreOptions) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if !db.isOpen {
		return fmt.Errorf("database is closed")
	}

	if options == nil {
		options = backup.DefaultRestoreOptions()
	}

	// Validate backup
	if err := backup.ValidateBackup(backupFormat); err != nil {
		return fmt.Errorf("invalid backup: %w", err)
	}

	// Restore each collection
	for _, collBackup := range backupFormat.Collections {
		if err := db.restoreCollection(collBackup, options); err != nil {
			return fmt.Errorf("failed to restore collection %s: %w", collBackup.Name, err)
		}
	}

	return nil
}

// restoreCollection restores a single collection from backup
func (db *Database) restoreCollection(collBackup backup.CollectionBackup, options *backup.RestoreOptions) error {
	// Drop existing collection if requested
	if options.DropExisting {
		if _, exists := db.collections[collBackup.Name]; exists {
			delete(db.collections, collBackup.Name)
		}
	}

	// Create or get collection
	var coll *Collection
	if existing, exists := db.collections[collBackup.Name]; exists {
		coll = existing
	} else {
		// Create document store for this collection
		docStore := NewDocumentStore(db.storage.DiskManager(), 1000) // 1000 documents cache
		coll = NewCollection(collBackup.Name, db.txnMgr, docStore)
		db.collections[collBackup.Name] = coll
	}

	// Restore documents
	for _, docBackup := range collBackup.Documents {
		docMap, err := backup.ConvertDocumentFromBackup(docBackup)
		if err != nil {
			return fmt.Errorf("failed to convert document: %w", err)
		}

		// Insert document (unlock/lock to avoid deadlock)
		db.mu.Unlock()
		_, err = coll.InsertOne(docMap)
		db.mu.Lock()

		if err != nil {
			return fmt.Errorf("failed to insert document %s: %w", docBackup.ID, err)
		}
	}

	// Restore indexes if not skipped
	if !options.SkipIndexes {
		if err := db.restoreIndexes(coll, collBackup.Indexes); err != nil {
			return fmt.Errorf("failed to restore indexes: %w", err)
		}
	}

	return nil
}

// restoreIndexes restores indexes for a collection
func (db *Database) restoreIndexes(coll *Collection, indexes []backup.IndexBackup) error {
	for _, idxBackup := range indexes {
		// Unlock during index creation to avoid deadlock
		db.mu.Unlock()
		err := db.createIndexFromBackup(coll, idxBackup)
		db.mu.Lock()

		if err != nil {
			return fmt.Errorf("failed to create index %s: %w", idxBackup.Name, err)
		}
	}
	return nil
}

// createIndexFromBackup creates an index from backup definition
func (db *Database) createIndexFromBackup(coll *Collection, idxBackup backup.IndexBackup) error {
	switch idxBackup.Type {
	case "btree":
		if len(idxBackup.FieldPaths) == 1 {
			// Single-field index
			if idxBackup.Filter != nil {
				// Partial index
				return coll.CreatePartialIndex(idxBackup.FieldPaths[0], idxBackup.Filter, idxBackup.Unique)
			}
			// Regular index
			return coll.CreateIndex(idxBackup.FieldPaths[0], idxBackup.Unique)
		} else {
			// Compound index
			return coll.CreateCompoundIndex(idxBackup.FieldPaths, idxBackup.Unique)
		}

	case "text":
		return coll.CreateTextIndex(idxBackup.FieldPaths)

	case "geo":
		if len(idxBackup.FieldPaths) != 1 {
			return fmt.Errorf("geo index must have exactly one field")
		}
		if idxBackup.GeoType == "2d" {
			return coll.Create2DIndex(idxBackup.FieldPaths[0])
		} else if idxBackup.GeoType == "2dsphere" {
			return coll.Create2DSphereIndex(idxBackup.FieldPaths[0])
		}
		return fmt.Errorf("unsupported geo type: %s", idxBackup.GeoType)

	case "ttl":
		if len(idxBackup.FieldPaths) != 1 {
			return fmt.Errorf("ttl index must have exactly one field")
		}
		if idxBackup.TTLDuration == nil {
			return fmt.Errorf("ttl index must have ttl_duration")
		}
		return coll.CreateTTLIndex(idxBackup.FieldPaths[0], *idxBackup.TTLDuration)

	default:
		return fmt.Errorf("unsupported index type: %s", idxBackup.Type)
	}
}

// BackupToFile creates a backup file at the specified path
func (db *Database) BackupToFile(path string, pretty bool) error {
	backupFormat, err := db.Backup()
	if err != nil {
		return err
	}

	backuper := backup.NewBackuper(pretty)
	return backuper.BackupToFile(path, backupFormat)
}

// RestoreFromFile restores the database from a backup file
func (db *Database) RestoreFromFile(path string, options *backup.RestoreOptions) error {
	restorer := backup.NewRestorer()
	backupFormat, err := restorer.RestoreFromFile(path)
	if err != nil {
		return err
	}

	return db.Restore(backupFormat, options)
}
