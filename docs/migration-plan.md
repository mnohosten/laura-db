# Migration Plan: In-Memory to Disk Storage

## Overview

This document outlines the comprehensive migration plan for transitioning LauraDB from in-memory document storage to disk-based persistent storage. This is a critical architectural change that will enable data to survive server restarts while maintaining backwards compatibility and data integrity.

## Executive Summary

**Current State**: Documents stored in memory (`Collection.documents map[string]interface{}`), lost on server restart.

**Target State**: Documents persisted to disk using slotted page structure, surviving server restarts.

**Impact**: ~4,000-5,000 LOC changes across 10 implementation phases.

**Timeline**: Phased rollout over multiple releases to minimize risk.

**Risk Level**: High - affects core database functionality.

## Goals

1. **Zero Data Loss**: Ensure safe migration of existing data
2. **Backwards Compatibility**: Support both old and new storage formats during transition
3. **Rollback Capability**: Ability to revert to in-memory mode if issues arise
4. **Minimal Downtime**: Provide migration tools for live systems
5. **Data Integrity**: Comprehensive validation at each migration step
6. **Performance Transparency**: Maintain or improve performance through caching

## Migration Strategy

### Phase-Based Rollout

We will use a **multi-phase deployment strategy** with feature flags and compatibility layers:

```
Release v0.9.0 (Current): In-memory only
    ↓
Release v1.0.0: Hybrid mode (in-memory + disk infrastructure)
    ↓
Release v1.1.0: Disk-first with in-memory fallback
    ↓
Release v1.2.0: Disk-only (remove in-memory code)
```

### Three Migration Paths

We will support three migration paths to accommodate different user needs:

1. **New Database Path**: Fresh installations use disk storage by default
2. **In-Place Migration Path**: Existing databases upgraded automatically
3. **Manual Migration Path**: Users explicitly migrate when ready

## Backwards Compatibility Strategy

### 1. Data Format Compatibility

#### Version Detection

Add version marker to data directory:

```
data/
├── .lauradb-version    ← New file indicating storage format version
├── data.db             ← Disk storage (v2.0+)
├── wal.log             ← Write-ahead log (existing)
└── metadata.db         ← Collection metadata (v2.0+)
```

**.lauradb-version File Format**:
```json
{
  "formatVersion": "2.0",
  "storageMode": "disk",
  "migrationDate": "2025-01-15T10:30:00Z",
  "previousVersion": "1.0",
  "compatible": ["2.0", "2.1"]
}
```

#### Format Version Registry

```go
const (
    StorageFormatV1_0 = "1.0"  // In-memory only
    StorageFormatV2_0 = "2.0"  // Disk-based with slotted pages
    StorageFormatV2_1 = "2.1"  // With compression (future)
)

type FormatVersion struct {
    Version      string
    StorageMode  string  // "memory" or "disk"
    IsDeprecated bool
    UpgradePath  string
}
```

### 2. API Compatibility

#### Transparent Storage Abstraction

Introduce storage abstraction layer to hide implementation details:

```go
// Storage abstraction (NEW)
type DocumentStore interface {
    Insert(doc map[string]interface{}) (DocumentID, error)
    Find(id DocumentID) (map[string]interface{}, error)
    Update(id DocumentID, doc map[string]interface{}) error
    Delete(id DocumentID) error
    Scan(predicate func(doc map[string]interface{}) bool) ([]map[string]interface{}, error)
}

// In-memory implementation (existing, wrapped)
type MemoryDocumentStore struct {
    documents map[string]interface{}  // Existing field
}

// Disk-based implementation (NEW)
type DiskDocumentStore struct {
    storage      *StorageEngine
    pageManager  *DocumentPageManager
    docCache     *DocumentCache
}

// Collection uses abstraction
type Collection struct {
    name  string
    store DocumentStore  // Abstraction instead of direct map

    // ... other fields unchanged
}
```

**Benefits**:
- No API changes for users
- Internal implementation can change
- Can switch between storage modes dynamically
- Testing both implementations side-by-side

### 3. Configuration Compatibility

#### Storage Mode Selection

Add configuration option to choose storage mode:

```go
type DatabaseConfig struct {
    // Existing fields
    DataDir        string
    BufferPoolSize int

    // NEW: Storage mode configuration
    StorageMode    string  // "auto", "memory", "disk"
    ForceMode      bool    // Force specific mode, ignore version file
    AutoMigrate    bool    // Automatically migrate on version mismatch

    // NEW: Migration options
    MigrationOptions MigrationConfig
}

type MigrationConfig struct {
    CreateBackup       bool          // Create backup before migration
    BackupPath         string        // Custom backup path
    ValidateAfter      bool          // Validate data after migration
    MaxMigrationTime   time.Duration // Timeout for migration
    BatchSize          int           // Documents per batch
    ProgressCallback   func(float64) // Progress reporting
}
```

#### Storage Mode Resolution

```
On Database.Open():

1. Check for .lauradb-version file
2. If exists:
   a. Read format version
   b. If version < 2.0 and StorageMode = "auto":
      - Offer to migrate
      - If AutoMigrate = true: proceed with migration
      - If AutoMigrate = false: warn and continue in memory mode
   c. If version >= 2.0:
      - Use disk storage
3. If not exists:
   a. If StorageMode = "disk": Initialize disk storage
   b. If StorageMode = "memory": Use in-memory storage
   c. If StorageMode = "auto": Use disk storage (v2.0 default)
4. If ForceMode = true:
   - Use specified StorageMode regardless of version
```

### 4. Feature Flags

Use feature flags to control rollout:

```go
type FeatureFlags struct {
    EnableDiskStorage       bool  // Master switch for disk storage
    EnableAutoMigration     bool  // Auto-migrate on open
    EnableHybridMode        bool  // Cache in memory + persist to disk
    EnableCompression       bool  // Page compression (future)
    EnablePrefetching       bool  // Prefetch pages for scans

    // Rollback flags
    FallbackToMemoryOnError bool  // Revert to memory if disk fails
}
```

## Data Migration Utilities

### 1. Migration Tool Architecture

```
┌────────────────────────────────────────────────────┐
│          Migration Coordinator                      │
│  - Orchestrates migration phases                    │
│  - Progress tracking                                │
│  - Error handling and rollback                      │
└────────────────────────────────────────────────────┘
                     ↓
┌────────────────────────────────────────────────────┐
│          Migration Phases                           │
│  1. Backup                                          │
│  2. Validation                                      │
│  3. Collection Migration                            │
│  4. Index Rebuild                                   │
│  5. Verification                                    │
└────────────────────────────────────────────────────┘
                     ↓
┌────────────────────────────────────────────────────┐
│          Migration Tools                            │
│  - BackupManager                                    │
│  - DataValidator                                    │
│  - CollectionMigrator                               │
│  - IndexRebuilder                                   │
└────────────────────────────────────────────────────┘
```

### 2. Migration Coordinator

```go
type MigrationCoordinator struct {
    source      *Database         // Source database (in-memory)
    target      *Database         // Target database (disk)
    config      MigrationConfig
    progress    *MigrationProgress
    logger      *log.Logger
}

type MigrationProgress struct {
    Phase               string
    TotalCollections    int
    MigratedCollections int
    TotalDocuments      int64
    MigratedDocuments   int64
    TotalIndexes        int
    RebuiltIndexes      int
    StartTime           time.Time
    EstimatedCompletion time.Time
    Errors              []error
}

func NewMigrationCoordinator(sourceDir, targetDir string, config MigrationConfig) *MigrationCoordinator

func (mc *MigrationCoordinator) Migrate() error {
    // Phase 1: Pre-migration checks
    if err := mc.validateSource(); err != nil {
        return err
    }

    // Phase 2: Create backup
    if mc.config.CreateBackup {
        if err := mc.createBackup(); err != nil {
            return err
        }
    }

    // Phase 3: Initialize target database
    if err := mc.initializeTarget(); err != nil {
        return err
    }

    // Phase 4: Migrate collections
    if err := mc.migrateCollections(); err != nil {
        return mc.rollback(err)
    }

    // Phase 5: Rebuild indexes
    if err := mc.rebuildIndexes(); err != nil {
        return mc.rollback(err)
    }

    // Phase 6: Validate migrated data
    if mc.config.ValidateAfter {
        if err := mc.validateMigration(); err != nil {
            return mc.rollback(err)
        }
    }

    // Phase 7: Update version file
    if err := mc.updateVersionFile(); err != nil {
        return mc.rollback(err)
    }

    return nil
}
```

### 3. Collection Migrator

```go
type CollectionMigrator struct {
    source      *Collection  // In-memory collection
    target      *Collection  // Disk-based collection
    batchSize   int
    validator   *DataValidator
}

func (cm *CollectionMigrator) MigrateCollection() error {
    // Get all documents from in-memory collection
    docs := cm.source.getAllDocuments()

    total := len(docs)
    migrated := 0

    // Migrate in batches
    for i := 0; i < total; i += cm.batchSize {
        end := min(i+cm.batchSize, total)
        batch := docs[i:end]

        // Begin transaction
        txn := cm.target.db.txnMgr.Begin()

        // Insert documents into disk storage
        for _, doc := range batch {
            if err := cm.target.InsertOne(txn, doc); err != nil {
                txn.Abort()
                return fmt.Errorf("failed to insert document: %w", err)
            }
            migrated++
        }

        // Commit transaction
        if err := txn.Commit(); err != nil {
            return fmt.Errorf("failed to commit batch: %w", err)
        }

        // Report progress
        cm.reportProgress(float64(migrated) / float64(total))
    }

    return nil
}
```

### 4. Index Rebuilder

```go
type IndexRebuilder struct {
    collection  *Collection
    indexes     []*Index
}

func (ir *IndexRebuilder) RebuildAllIndexes() error {
    for _, idx := range ir.indexes {
        if err := ir.rebuildIndex(idx); err != nil {
            return fmt.Errorf("failed to rebuild index %s: %w", idx.name, err)
        }
    }
    return nil
}

func (ir *IndexRebuilder) rebuildIndex(idx *Index) error {
    // 1. Scan all documents
    docs := ir.collection.getAllDocuments()

    // 2. Sort by indexed field (for bulk loading)
    sortedDocs := ir.sortByIndexField(docs, idx)

    // 3. Bulk load into B+ tree
    return idx.bulkLoad(sortedDocs)
}
```

### 5. Data Validator

```go
type DataValidator struct {
    source *Database
    target *Database
}

func (dv *DataValidator) ValidateMigration() error {
    // 1. Compare collection count
    if err := dv.validateCollectionCount(); err != nil {
        return err
    }

    // 2. For each collection, compare document count
    for _, collName := range dv.source.listCollections() {
        if err := dv.validateCollectionDocumentCount(collName); err != nil {
            return err
        }
    }

    // 3. Sample-based content validation (10% of documents)
    if err := dv.sampleValidation(0.1); err != nil {
        return err
    }

    // 4. Index integrity check
    if err := dv.validateIndexes(); err != nil {
        return err
    }

    return nil
}

func (dv *DataValidator) sampleValidation(sampleRate float64) error {
    for _, collName := range dv.source.listCollections() {
        sourceColl := dv.source.GetCollection(collName)
        targetColl := dv.target.GetCollection(collName)

        // Sample documents
        docs := sourceColl.getAllDocuments()
        sampleSize := int(float64(len(docs)) * sampleRate)

        for i := 0; i < sampleSize; i++ {
            // Random sampling
            idx := rand.Intn(len(docs))
            sourceDoc := docs[idx]

            // Find same document in target
            id := sourceDoc["_id"]
            targetDoc, err := targetColl.FindOne(bson.M{"_id": id})
            if err != nil {
                return fmt.Errorf("document %v not found in target: %w", id, err)
            }

            // Deep compare
            if !deepEqual(sourceDoc, targetDoc) {
                return fmt.Errorf("document %v mismatch", id)
            }
        }
    }
    return nil
}
```

### 6. Backup Manager

```go
type BackupManager struct {
    sourceDir string
    backupDir string
}

func (bm *BackupManager) CreateBackup() error {
    // 1. Create timestamped backup directory
    timestamp := time.Now().Format("20060102-150405")
    backupPath := filepath.Join(bm.backupDir, fmt.Sprintf("backup-%s", timestamp))

    if err := os.MkdirAll(backupPath, 0755); err != nil {
        return err
    }

    // 2. Export all collections to JSON
    db, err := Open(bm.sourceDir)
    if err != nil {
        return err
    }
    defer db.Close()

    for _, collName := range db.listCollections() {
        if err := bm.exportCollection(db, collName, backupPath); err != nil {
            return err
        }
    }

    // 3. Create backup manifest
    if err := bm.createManifest(backupPath); err != nil {
        return err
    }

    return nil
}

func (bm *BackupManager) exportCollection(db *Database, collName, backupPath string) error {
    coll := db.GetCollection(collName)
    docs := coll.getAllDocuments()

    // Export to JSONL format (one JSON object per line)
    filename := filepath.Join(backupPath, collName+".jsonl")
    f, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer f.Close()

    encoder := json.NewEncoder(f)
    for _, doc := range docs {
        if err := encoder.Encode(doc); err != nil {
            return err
        }
    }

    return nil
}
```

### 7. Command-Line Migration Tool

```bash
laura-migrate - LauraDB Storage Migration Tool

USAGE:
    laura-migrate [OPTIONS] <SOURCE_DIR> [TARGET_DIR]

OPTIONS:
    --mode <mode>           Migration mode: auto, export, import, validate
    --backup                Create backup before migration
    --backup-dir <dir>      Custom backup directory
    --batch-size <n>        Documents per batch (default: 1000)
    --validate              Validate data after migration
    --dry-run               Show what would be migrated without making changes
    --force                 Force migration even if target exists
    --progress              Show progress bar
    --log-file <file>       Write migration log to file

MODES:
    auto        Automatic in-place migration (default)
    export      Export data to backup format
    import      Import data from backup format
    validate    Validate migration without making changes

EXAMPLES:
    # In-place migration with backup
    laura-migrate --backup /data/lauradb

    # Export to backup, then migrate to new directory
    laura-migrate --mode export /data/lauradb /backup/lauradb-20250115
    laura-migrate --mode import /backup/lauradb-20250115 /data/lauradb-v2

    # Validate migration without changes
    laura-migrate --dry-run --validate /data/lauradb

    # Force migration to new directory
    laura-migrate --force /data/old /data/new
```

### 8. Migration Tool Implementation

```go
package main

import (
    "flag"
    "fmt"
    "log"
    "os"
    "time"
)

type MigrationMode string

const (
    ModeAuto     MigrationMode = "auto"
    ModeExport   MigrationMode = "export"
    ModeImport   MigrationMode = "import"
    ModeValidate MigrationMode = "validate"
)

type MigrationOptions struct {
    Mode         MigrationMode
    SourceDir    string
    TargetDir    string
    BackupDir    string
    CreateBackup bool
    BatchSize    int
    Validate     bool
    DryRun       bool
    Force        bool
    ShowProgress bool
    LogFile      string
}

func main() {
    opts := parseCLIArgs()

    // Setup logging
    logger := setupLogging(opts.LogFile)

    // Create migration coordinator
    config := MigrationConfig{
        CreateBackup:     opts.CreateBackup,
        BackupPath:       opts.BackupDir,
        ValidateAfter:    opts.Validate,
        BatchSize:        opts.BatchSize,
        MaxMigrationTime: 24 * time.Hour,
        ProgressCallback: createProgressCallback(opts.ShowProgress),
    }

    coordinator := NewMigrationCoordinator(opts.SourceDir, opts.TargetDir, config)

    // Execute migration based on mode
    switch opts.Mode {
    case ModeAuto:
        if opts.DryRun {
            logger.Println("DRY RUN: No changes will be made")
            if err := coordinator.analyzeAndReport(); err != nil {
                log.Fatal(err)
            }
        } else {
            if err := coordinator.Migrate(); err != nil {
                log.Fatal(err)
            }
            logger.Println("Migration completed successfully")
        }

    case ModeExport:
        backupMgr := NewBackupManager(opts.SourceDir, opts.TargetDir)
        if err := backupMgr.CreateBackup(); err != nil {
            log.Fatal(err)
        }
        logger.Println("Export completed successfully")

    case ModeImport:
        // Import from backup format
        if err := coordinator.ImportFromBackup(opts.SourceDir); err != nil {
            log.Fatal(err)
        }
        logger.Println("Import completed successfully")

    case ModeValidate:
        validator := NewDataValidator(opts.SourceDir, opts.TargetDir)
        if err := validator.ValidateMigration(); err != nil {
            log.Fatal(err)
        }
        logger.Println("Validation passed")
    }
}
```

## Testing Strategy

### 1. Test Pyramid

```
                    ┌─────────────────┐
                    │  E2E Tests      │  10% - Full migration scenarios
                    └─────────────────┘
                  ┌───────────────────────┐
                  │  Integration Tests    │  30% - Multi-component migration
                  └───────────────────────┘
              ┌─────────────────────────────────┐
              │      Unit Tests                 │  60% - Individual components
              └─────────────────────────────────┘
```

### 2. Unit Tests

Test individual components in isolation:

#### Storage Abstraction Tests
```go
func TestMemoryDocumentStore(t *testing.T) {
    store := NewMemoryDocumentStore()

    // Test insert
    doc := map[string]interface{}{"_id": "1", "name": "Alice"}
    id, err := store.Insert(doc)
    assert.NoError(t, err)

    // Test find
    found, err := store.Find(id)
    assert.NoError(t, err)
    assert.Equal(t, "Alice", found["name"])

    // Test update
    doc["name"] = "Bob"
    err = store.Update(id, doc)
    assert.NoError(t, err)

    // Test delete
    err = store.Delete(id)
    assert.NoError(t, err)
}

func TestDiskDocumentStore(t *testing.T) {
    // Same tests as above, but with DiskDocumentStore
    // This ensures both implementations behave identically
}
```

#### Migration Utility Tests
```go
func TestCollectionMigrator(t *testing.T) {
    // Create in-memory collection with test data
    source := createTestCollection(1000)
    target := createEmptyDiskCollection()

    migrator := NewCollectionMigrator(source, target, 100)
    err := migrator.MigrateCollection()
    assert.NoError(t, err)

    // Verify document count
    assert.Equal(t, source.Count(), target.Count())
}

func TestDataValidator(t *testing.T) {
    source := createTestDatabase()
    target := createMigratedDatabase(source)

    validator := NewDataValidator(source, target)
    err := validator.ValidateMigration()
    assert.NoError(t, err)
}

func TestBackupManager(t *testing.T) {
    db := createTestDatabase()
    backupMgr := NewBackupManager(db.dataDir, "/tmp/backup")

    err := backupMgr.CreateBackup()
    assert.NoError(t, err)

    // Verify backup files exist
    assert.FileExists(t, "/tmp/backup/backup-*/users.jsonl")
}
```

### 3. Integration Tests

Test component interactions:

#### End-to-End Migration Test
```go
func TestFullMigration(t *testing.T) {
    // Setup: Create in-memory database with sample data
    sourceDir := t.TempDir()
    targetDir := t.TempDir()

    db := createTestDatabaseWithData(sourceDir, 10000)
    db.Close()

    // Execute migration
    config := MigrationConfig{
        CreateBackup:  true,
        ValidateAfter: true,
        BatchSize:     100,
    }

    coordinator := NewMigrationCoordinator(sourceDir, targetDir, config)
    err := coordinator.Migrate()
    assert.NoError(t, err)

    // Verify: Open target database and check data
    targetDB, err := Open(targetDir)
    assert.NoError(t, err)
    defer targetDB.Close()

    // Check all collections migrated
    assert.Equal(t, db.listCollections(), targetDB.listCollections())

    // Check document counts
    for _, collName := range db.listCollections() {
        sourceColl := db.GetCollection(collName)
        targetColl := targetDB.GetCollection(collName)
        assert.Equal(t, sourceColl.Count(), targetColl.Count())
    }
}
```

#### Incremental Migration Test
```go
func TestIncrementalMigration(t *testing.T) {
    // Test migrating in multiple batches
    db := createLargeTestDatabase(100000)

    migrator := NewCollectionMigrator(db.GetCollection("users"), target, 1000)

    // Migrate in chunks, simulating interruption
    for i := 0; i < 10; i++ {
        err := migrator.MigrateBatch()
        assert.NoError(t, err)
    }

    // Verify all documents migrated
    assert.Equal(t, 100000, target.Count())
}
```

#### Rollback Test
```go
func TestMigrationRollback(t *testing.T) {
    db := createTestDatabase()

    // Inject error during migration
    coordinator := NewMigrationCoordinator(sourceDir, targetDir, config)
    coordinator.injectError = true

    err := coordinator.Migrate()
    assert.Error(t, err)

    // Verify rollback occurred
    assert.DirNotExists(t, targetDir)

    // Verify backup still exists
    assert.DirExists(t, coordinator.backupPath)
}
```

### 4. Performance Tests

Measure migration performance:

```go
func BenchmarkMigration(b *testing.B) {
    sizes := []int{1000, 10000, 100000, 1000000}

    for _, size := range sizes {
        b.Run(fmt.Sprintf("docs_%d", size), func(b *testing.B) {
            db := createTestDatabaseWithData(b.TempDir(), size)

            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                coordinator := NewMigrationCoordinator(...)
                coordinator.Migrate()
            }

            b.ReportMetric(float64(size)/b.Elapsed().Seconds(), "docs/sec")
        })
    }
}

func TestMigrationPerformance(t *testing.T) {
    // Measure migration throughput
    db := createTestDatabaseWithData(t.TempDir(), 100000)

    start := time.Now()
    coordinator := NewMigrationCoordinator(...)
    err := coordinator.Migrate()
    elapsed := time.Since(start)

    assert.NoError(t, err)

    throughput := float64(100000) / elapsed.Seconds()
    t.Logf("Migration throughput: %.2f docs/sec", throughput)

    // Assert minimum throughput
    assert.Greater(t, throughput, 1000.0, "Migration too slow")
}
```

### 5. Stress Tests

Test under extreme conditions:

```go
func TestLargeDatabaseMigration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping stress test in short mode")
    }

    // 10 million documents across 100 collections
    db := createTestDatabaseWithData(t.TempDir(), 10_000_000)

    coordinator := NewMigrationCoordinator(...)
    err := coordinator.Migrate()
    assert.NoError(t, err)
}

func TestConcurrentMigration(t *testing.T) {
    // Migrate multiple collections concurrently
    db := createTestDatabaseWithMultipleCollections(t.TempDir(), 10)

    var wg sync.WaitGroup
    for _, collName := range db.listCollections() {
        wg.Add(1)
        go func(name string) {
            defer wg.Done()
            migrator := NewCollectionMigrator(...)
            err := migrator.MigrateCollection()
            assert.NoError(t, err)
        }(collName)
    }

    wg.Wait()
}

func TestMigrationWithCorruption(t *testing.T) {
    // Test migration with corrupted documents
    db := createTestDatabase()

    // Inject corrupted document
    db.GetCollection("users").forceInsert(corruptedDocument)

    coordinator := NewMigrationCoordinator(...)
    err := coordinator.Migrate()

    // Should detect and handle corruption
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "corrupted")
}
```

### 6. Compatibility Tests

Test backwards compatibility:

```go
func TestVersionDetection(t *testing.T) {
    tests := []struct {
        version      string
        storageMode  string
        expectError  bool
    }{
        {"1.0", "memory", false},
        {"2.0", "disk", false},
        {"2.1", "disk", false},
        {"3.0", "disk", true},  // Future version
    }

    for _, tt := range tests {
        t.Run(tt.version, func(t *testing.T) {
            db := createDatabaseWithVersion(tt.version)
            _, err := Open(db.dataDir)

            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

func TestHybridMode(t *testing.T) {
    // Test running with both in-memory and disk storage
    config := DatabaseConfig{
        StorageMode: "auto",
        FeatureFlags: FeatureFlags{
            EnableHybridMode: true,
        },
    }

    db, err := OpenWithConfig(t.TempDir(), config)
    assert.NoError(t, err)

    // Insert document
    coll := db.GetCollection("users")
    coll.InsertOne(map[string]interface{}{"name": "Alice"})

    // Should be in both memory and disk
    assert.True(t, coll.store.(*HybridDocumentStore).inMemory.Exists("Alice"))
    assert.True(t, coll.store.(*HybridDocumentStore).onDisk.Exists("Alice"))
}
```

### 7. Recovery Tests

Test crash recovery during migration:

```go
func TestCrashDuringMigration(t *testing.T) {
    db := createTestDatabase()

    coordinator := NewMigrationCoordinator(...)

    // Simulate crash halfway through
    coordinator.crashAt = 0.5
    err := coordinator.Migrate()
    assert.Error(t, err)

    // Verify partial migration detected
    resumeCoordinator := NewMigrationCoordinator(...)
    assert.True(t, resumeCoordinator.canResume())

    // Resume migration
    err = resumeCoordinator.Resume()
    assert.NoError(t, err)
}

func TestWALRecoveryAfterMigration(t *testing.T) {
    // Migrate database
    coordinator := NewMigrationCoordinator(...)
    coordinator.Migrate()

    // Open migrated database
    db, err := Open(targetDir)
    assert.NoError(t, err)

    // Insert document
    db.GetCollection("users").InsertOne(map[string]interface{}{"name": "Bob"})

    // Simulate crash (close without checkpoint)
    db.storage.Close()

    // Reopen and verify WAL recovery
    db, err = Open(targetDir)
    assert.NoError(t, err)

    doc, err := db.GetCollection("users").FindOne(bson.M{"name": "Bob"})
    assert.NoError(t, err)
    assert.NotNil(t, doc)
}
```

### 8. Test Data Generators

```go
func createTestDatabaseWithData(dir string, docCount int) *Database {
    db, _ := Open(dir)

    coll := db.GetCollection("users")

    for i := 0; i < docCount; i++ {
        doc := map[string]interface{}{
            "_id":   fmt.Sprintf("user_%d", i),
            "name":  fmt.Sprintf("User %d", i),
            "email": fmt.Sprintf("user%d@example.com", i),
            "age":   int64(20 + i%60),
            "tags":  []string{"tag1", "tag2", "tag3"},
        }
        coll.InsertOne(doc)
    }

    // Create indexes
    coll.CreateIndex("email", true, false)
    coll.CreateIndex("age", false, false)

    return db
}

func createLargeDocument() map[string]interface{} {
    // Create document near 16MB limit
    largeArray := make([]string, 100000)
    for i := range largeArray {
        largeArray[i] = strings.Repeat("x", 100)
    }

    return map[string]interface{}{
        "_id":   "large_doc",
        "data":  largeArray,
    }
}
```

## Migration Best Practices

### For Developers

1. **Test Thoroughly**: Run full test suite on both in-memory and disk modes
2. **Incremental Rollout**: Deploy to staging before production
3. **Monitor Metrics**: Track migration performance and errors
4. **Backup Always**: Always create backup before migration
5. **Version Control**: Keep migration scripts in version control

### For Users

1. **Backup First**: Always backup your data before migration
2. **Test on Copy**: Test migration on a copy of your database first
3. **Schedule Downtime**: Plan for brief downtime during migration
4. **Validate After**: Run validation checks after migration
5. **Keep Backups**: Retain backups for at least 30 days after migration

### Common Pitfalls

1. **Insufficient Disk Space**: Ensure 2x data size available
2. **Long-Running Transactions**: Commit or rollback before migration
3. **Concurrent Writes**: Stop all writes during migration
4. **Index Corruption**: Rebuild indexes if migration interrupted
5. **Version Mismatch**: Ensure compatible versions

## Rollback Procedures

### Automatic Rollback

Migration coordinator automatically rolls back on errors:

```go
func (mc *MigrationCoordinator) rollback(err error) error {
    mc.logger.Printf("Migration failed: %v. Rolling back...", err)

    // 1. Abort any active transactions
    mc.target.txnMgr.AbortAll()

    // 2. Delete partially migrated data
    if err := os.RemoveAll(mc.targetDir); err != nil {
        mc.logger.Printf("Failed to clean up target: %v", err)
    }

    // 3. Restore from backup if created
    if mc.config.CreateBackup {
        if err := mc.restoreFromBackup(); err != nil {
            return fmt.Errorf("rollback failed: %w", err)
        }
    }

    // 4. Log rollback completion
    mc.logger.Println("Rollback completed")

    return err
}
```

### Manual Rollback

If automatic rollback fails:

```bash
# 1. Stop the database
systemctl stop laura-db

# 2. Remove new version file
rm /data/lauradb/.lauradb-version

# 3. Restore from backup
laura-migrate --mode import /backup/lauradb-20250115 /data/lauradb

# 4. Restart with in-memory mode
laura-server --storage-mode memory --data-dir /data/lauradb
```

## Migration Monitoring

### Metrics to Track

```go
type MigrationMetrics struct {
    // Progress metrics
    TotalDocuments     int64
    MigratedDocuments  int64
    FailedDocuments    int64

    // Performance metrics
    DocsPerSecond      float64
    AvgDocSize         int64

    // Resource metrics
    MemoryUsage        int64
    DiskUsage          int64
    CPUUsage           float64

    // Error metrics
    ErrorCount         int
    WarningCount       int

    // Time metrics
    StartTime          time.Time
    EstimatedCompletion time.Time
}

func (mm *MigrationMetrics) Report() string {
    progress := float64(mm.MigratedDocuments) / float64(mm.TotalDocuments) * 100

    return fmt.Sprintf(`
Migration Progress: %.2f%%
Migrated: %d / %d documents
Throughput: %.2f docs/sec
Failed: %d documents
Errors: %d
Warnings: %d
Estimated completion: %s
`, progress, mm.MigratedDocuments, mm.TotalDocuments,
   mm.DocsPerSecond, mm.FailedDocuments, mm.ErrorCount,
   mm.WarningCount, mm.EstimatedCompletion.Format(time.RFC3339))
}
```

### Progress Reporting

```go
func createProgressCallback(showProgress bool) func(float64) {
    if !showProgress {
        return nil
    }

    return func(progress float64) {
        bar := strings.Repeat("=", int(progress*50))
        spaces := strings.Repeat(" ", 50-len(bar))
        fmt.Printf("\r[%s%s] %.1f%%", bar, spaces, progress*100)
    }
}
```

## Documentation Updates

After migration implementation, update:

1. **README.md**: Add migration guide
2. **CLAUDE.md**: Remove in-memory limitation
3. **docs/architecture.md**: Update with disk storage details
4. **docs/configuration.md**: Add storage mode options
5. **docs/migration-guide.md**: User-facing migration guide
6. **API documentation**: Update any affected APIs

## Timeline

### Phase 1: Infrastructure (Release v1.0.0)
- **Week 1-2**: Implement storage abstraction layer
- **Week 3-4**: Add version detection and compatibility checks
- **Week 5-6**: Testing and documentation

### Phase 2: Migration Tools (Release v1.0.0)
- **Week 7-8**: Implement migration coordinator
- **Week 9-10**: Build CLI migration tool
- **Week 11-12**: Testing and documentation

### Phase 3: Disk Storage (Release v1.1.0)
- **Month 4-5**: Implement disk storage (Phases 2-7 from TODO.md)
- **Month 6**: Testing and optimization

### Phase 4: Production Deployment (Release v1.2.0)
- **Month 7**: Beta testing with early adopters
- **Month 8**: Production release and documentation

## Success Criteria

Migration is successful when:

1. ✅ All documents migrated correctly
2. ✅ All indexes rebuilt and functional
3. ✅ Data validation passes 100%
4. ✅ Performance meets or exceeds in-memory mode (with caching)
5. ✅ Zero data loss
6. ✅ Backwards compatibility maintained
7. ✅ Rollback works correctly
8. ✅ Documentation complete

## Risk Mitigation

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Data loss during migration | Low | Critical | Mandatory backups, validation checks |
| Performance degradation | Medium | High | Extensive caching, prefetching |
| Migration tool bugs | Medium | High | Comprehensive testing, staged rollout |
| Disk space exhaustion | Low | Medium | Pre-migration space checks |
| Corruption during crash | Low | High | WAL logging, atomic operations |
| User adoption resistance | Medium | Low | Clear documentation, migration tools |

## Conclusion

This migration plan provides a comprehensive strategy for transitioning LauraDB from in-memory to disk-based storage while maintaining backwards compatibility, data integrity, and minimal downtime. The phased approach with extensive testing and rollback capabilities minimizes risk and ensures a smooth migration for all users.

**Key Takeaways**:
1. Storage abstraction layer enables transparent migration
2. Multiple migration paths accommodate different user needs
3. Comprehensive testing ensures data integrity
4. Automated tools reduce manual effort and errors
5. Rollback procedures provide safety net
6. Monitoring and validation ensure migration success

**Next Steps**:
- Begin Phase 1 implementation (storage abstraction layer)
- Set up CI/CD for migration testing
- Create user documentation and migration guides
- Plan beta testing program with early adopters
