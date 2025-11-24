# LauraDB Repair Tools

This document describes the repair and maintenance tools available in LauraDB for database validation, repair, and optimization.

## Overview

LauraDB provides a comprehensive set of tools for maintaining database health and integrity:

- **Validator**: Validates database integrity and detects issues
- **Repairer**: Fixes database issues automatically or semi-automatically
- **Defragmenter**: Optimizes storage by compacting indexes and reclaiming space

These tools are available through:
1. The `laura-repair` command-line tool (`cmd/repair`)
2. The Go API (`pkg/repair` package)
3. Example programs (`examples/repair-demo`)

## Architecture

### Validator

The Validator performs comprehensive database integrity checks:

```
┌─────────────────────────────────────────────┐
│             Validator                       │
├─────────────────────────────────────────────┤
│  • Document Validation                      │
│    - Missing _id fields                     │
│    - Invalid ObjectIDs                      │
│    - Corrupt document structure             │
│                                             │
│  • Index Validation                         │
│    - Orphaned index entries                 │
│    - Missing index entries                  │
│    - Duplicate unique violations            │
│    - Index field mismatches                 │
│                                             │
│  • Integrity Checks                         │
│    - Collection accessibility               │
│    - Index functionality                    │
└─────────────────────────────────────────────┘
```

**Issue Types Detected:**

| Type | Description | Severity |
|------|-------------|----------|
| `missing_id` | Document missing `_id` field | Critical |
| `invalid_id` | Document `_id` is not an ObjectID | Warning |
| `orphaned_index_entry` | Index entry without document | Warning |
| `missing_index_entry` | Document not in index | Warning |
| `duplicate_unique` | Duplicate value in unique index | Critical |
| `corrupt_document` | Document structure corrupted | Critical |
| `invalid_index_order` | Index B+ tree structure invalid | Critical |
| `index_field_mismatch` | Index field doesn't match config | Warning |

### Repairer

The Repairer fixes issues identified by validation:

```
┌─────────────────────────────────────────────┐
│             Repairer                        │
├─────────────────────────────────────────────┤
│  Repair Options:                            │
│  • RebuildIndexes - Full index rebuild      │
│  • RemoveOrphans - Remove orphaned entries  │
│  • AddMissingEntries - Add missing entries  │
│  • UniqueConflictResolution - Conflict mode │
│  • DryRun - Preview without changes         │
└─────────────────────────────────────────────┘
```

**Repair Strategies:**

1. **Index Rebuild** - Drops and recreates all indexes from scratch
2. **Orphan Removal** - Removes index entries without corresponding documents
3. **Missing Entry Addition** - Adds missing index entries for documents
4. **Conflict Resolution** - Handles unique constraint violations:
   - `first`: Keep first occurrence
   - `last`: Keep last occurrence
   - `fail`: Report error without fixing

### Defragmenter

The Defragmenter optimizes storage by compacting indexes:

```
┌─────────────────────────────────────────────┐
│          Defragmenter                       │
├─────────────────────────────────────────────┤
│  Operations:                                │
│  • Rebuild B+ tree indexes compactly        │
│  • Reclaim fragmented space                 │
│  • Optimize index structure                 │
│  • Track space savings                      │
└─────────────────────────────────────────────┘
```

**Metrics Tracked:**
- Initial file size
- Final file size
- Space saved (bytes and percentage)
- Pages compacted
- Fragmentation ratio
- Operation duration

## Command-Line Tool

### Installation

Build the repair tool:

```bash
make repair
# or
go build -o bin/laura-repair cmd/repair/main.go
```

### Usage

```bash
laura-repair [options]
```

**Options:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-data-dir` | string | `./data` | Database data directory |
| `-collection` | string | `""` | Specific collection (empty = all) |
| `-operation` | string | `validate` | Operation: validate, repair, defragment |
| `-dry-run` | bool | `false` | Preview changes without applying |
| `-rebuild-indexes` | bool | `false` | Rebuild all indexes from scratch |
| `-remove-orphans` | bool | `true` | Remove orphaned index entries |
| `-add-missing` | bool | `true` | Add missing index entries |
| `-conflict-resolution` | string | `fail` | Unique conflict resolution: first, last, fail |
| `-verbose` | bool | `false` | Verbose output |
| `-version` | bool | `false` | Show version information |

### Examples

**1. Validate entire database:**

```bash
laura-repair -data-dir ./mydb -operation validate
```

Output:
```
╔═══════════════════════════════════════════╗
║     LauraDB Repair Tool v1.0.0            ║
╚═══════════════════════════════════════════╝

Running validation...
Target: Entire database

═══════════════════════════════════════════
Validation Results
═══════════════════════════════════════════
Duration:     234ms
Collections:  5
Documents:    1,234
Indexes:      12
Health:       ✓ Healthy
Issues:       0
═══════════════════════════════════════════

Database is healthy. Checked 1234 documents across 5 collections with 12 indexes.

✓ Operation completed successfully
```

**2. Validate specific collection:**

```bash
laura-repair -data-dir ./mydb -collection users -operation validate
```

**3. Repair in dry-run mode (preview):**

```bash
laura-repair -data-dir ./mydb -operation repair -dry-run
```

Output:
```
Running repair operation...
Target: Entire database
Mode: DRY RUN (no changes will be made)

Repair Options:
  Rebuild Indexes:       false
  Remove Orphans:        true
  Add Missing Entries:   true
  Conflict Resolution:   fail

═══════════════════════════════════════════
Repair Results
═══════════════════════════════════════════
Duration:     456ms
Issues Found: 3
Fixed:        0 (dry run)
Failed:       0
═══════════════════════════════════════════

Repair completed in 456ms. Fixed 0 issues, failed to fix 0 issues.
```

**4. Repair with index rebuild:**

```bash
laura-repair -data-dir ./mydb -operation repair -rebuild-indexes
```

**5. Defragment database:**

```bash
laura-repair -data-dir ./mydb -operation defragment
```

Output:
```
Running defragmentation...
Target: Entire database

═══════════════════════════════════════════
Defragmentation Results
═══════════════════════════════════════════
Duration:          1.23s
Pages Compacted:   45
Initial Size:      12.45 MB
Final Size:        11.23 MB
Space Saved:       1.22 MB (9.80%)
Fragmentation:     15.30%
═══════════════════════════════════════════

Defragmentation completed in 1.23s. Compacted 45 pages, saved 1279590 bytes (9.80%), fragmentation reduced from 15.30%

✓ Operation completed successfully
```

**6. Verbose validation:**

```bash
laura-repair -data-dir ./mydb -operation validate -verbose
```

## Go API

### Validation

```go
import (
    "github.com/mnohosten/laura-db/pkg/database"
    "github.com/mnohosten/laura-db/pkg/repair"
)

// Open database
db, err := database.Open("./data")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Create validator
validator := repair.NewValidator(db)

// Validate entire database
report, err := validator.Validate()
if err != nil {
    log.Fatal(err)
}

// Check results
fmt.Printf("Health: %v\n", report.IsHealthy)
fmt.Printf("Issues: %d\n", len(report.Issues))
fmt.Println(report.Summary())

// Print issues
for _, issue := range report.Issues {
    fmt.Printf("[%s] %s: %s\n",
        issue.Severity, issue.Type, issue.Description)
}

// Validate specific collection
collReport, err := validator.ValidateCollection("users")
```

### Repair

```go
// Create repairer
repairer := repair.NewRepairer(db)

// Configure repair options
options := &repair.RepairOptions{
    RebuildIndexes:           false,
    RemoveOrphans:            true,
    AddMissingEntries:        true,
    UniqueConflictResolution: "fail",
    DryRun:                   false,
}

// Or use defaults
options := repair.DefaultRepairOptions()

// Repair entire database
repairReport, err := repairer.Repair(options)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Fixed: %d\n", repairReport.Fixed)
fmt.Printf("Failed: %d\n", repairReport.Failed)
fmt.Println(repairReport.Summary())

// Repair specific collection
collRepairReport, err := repairer.RepairCollection("users", options)
```

### Defragmentation

```go
// Create defragmenter
defragmenter := repair.NewDefragmenter(db)

// Defragment entire database
defragReport, err := defragmenter.Defragment()
if err != nil {
    log.Fatal(err)
}

percentSaved := float64(defragReport.SpaceSaved) /
    float64(defragReport.InitialFileSize) * 100.0

fmt.Printf("Space saved: %d bytes (%.2f%%)\n",
    defragReport.SpaceSaved, percentSaved)
fmt.Printf("Pages compacted: %d\n", defragReport.PagesCompacted)
fmt.Println(defragReport.Summary())

// Defragment specific collection
collDefragReport, err := defragmenter.DefragmentCollection("users")
```

## Report Structures

### ValidationReport

```go
type ValidationReport struct {
    StartTime     time.Time  // When validation started
    EndTime       time.Time  // When validation ended
    Collections   []string   // Collections validated
    Issues        []Issue    // Issues found
    DocumentCount int64      // Total documents checked
    IndexCount    int64      // Total indexes checked
    IsHealthy     bool       // Overall health status
}
```

### RepairReport

```go
type RepairReport struct {
    StartTime    time.Time  // When repair started
    EndTime      time.Time  // When repair ended
    Issues       []Issue    // All issues found
    Fixed        int        // Number of issues fixed
    Failed       int        // Number of issues failed to fix
    FixedIssues  []Issue    // Successfully fixed issues
    FailedIssues []Issue    // Failed to fix issues
}
```

### DefragmentationReport

```go
type DefragmentationReport struct {
    StartTime          time.Time  // When defrag started
    EndTime            time.Time  // When defrag ended
    InitialFileSize    int64      // Size before defrag
    FinalFileSize      int64      // Size after defrag
    SpaceSaved         int64      // Bytes saved
    PagesCompacted     int        // Number of pages compacted
    FragmentationRatio float64    // Free pages / total pages before
}
```

### Issue

```go
type Issue struct {
    Type        IssueType              // Type of issue
    Severity    string                 // "critical", "warning", "info"
    Collection  string                 // Collection name
    DocumentID  string                 // Document ID if applicable
    IndexName   string                 // Index name if applicable
    Description string                 // Human-readable description
    Details     map[string]interface{} // Additional details
}
```

## Best Practices

### When to Validate

- **After crashes** - Check for corruption after unexpected shutdowns
- **Before upgrades** - Ensure data integrity before version upgrades
- **Periodic checks** - Schedule regular validations (daily/weekly)
- **After bulk operations** - Validate after large data imports or migrations
- **Performance issues** - Check for index problems when queries slow down

### When to Repair

- **After validation finds issues** - Fix problems detected by validator
- **Index corruption suspected** - Rebuild indexes if queries fail
- **Data migration** - Clean up after importing from other systems
- **After recovery** - Fix issues after crash recovery

### When to Defragment

- **High fragmentation** - When fragmentation ratio > 20%
- **After many deletes** - Significant data removal creates fragmentation
- **Performance degradation** - Slow queries despite proper indexes
- **Storage optimization** - Periodic maintenance to reclaim space
- **Before backups** - Reduce backup size and improve backup speed

### Safety Tips

1. **Always backup first** - Use backup tools before repair operations
2. **Use dry-run** - Preview changes with `-dry-run` flag
3. **Start with validation** - Understand issues before fixing
4. **Test on copy** - Test repair on database copy if possible
5. **Monitor progress** - Use `-verbose` for detailed output
6. **Verify after repair** - Validate again after repairs

## Performance Considerations

### Validation Performance

- **Time complexity**: O(D + I) where D = documents, I = index entries
- **Memory usage**: Minimal, streams documents
- **Typical speed**: ~10,000 documents/second

### Repair Performance

- **Index rebuild**: O(D log D) per index where D = documents
- **Typical speed**: ~5,000 documents/second for full rebuild
- **Downtime**: Database should be offline during repair

### Defragmentation Performance

- **Time complexity**: O(I log I) where I = index entries
- **Typical speed**: ~15,000 index entries/second
- **Space savings**: 5-20% typically, varies by fragmentation

## Troubleshooting

### Validation Fails

```
Error: Failed to retrieve documents: collection not found
```

**Solution**: Check collection name spelling, ensure database is open

### Repair Fails

```
Error: Failed to rebuild index: duplicate key error
```

**Solution**: Use unique conflict resolution strategy or fix duplicates manually

### Defragmentation Doesn't Save Space

```
Space Saved: 0 bytes (0.00%)
```

**Causes**:
- Database already optimized
- No deleted documents or index fragmentation
- Recent defragmentation already performed

**Solution**: This is normal for well-maintained databases

### High Memory Usage

```
Error: Out of memory during repair
```

**Solution**:
- Repair collections individually instead of entire database
- Increase available memory
- Use index rebuild sparingly on large collections

## Examples

See `examples/repair-demo` for comprehensive examples:

```bash
cd examples/repair-demo
go run main.go
```

Demos included:
1. Basic validation
2. Repair with dry-run
3. Index rebuild
4. Database defragmentation
5. Collection-specific operations

## Integration with Other Tools

### With Backup/Restore

```bash
# Backup before repair
laura-backup -data-dir ./mydb -output backup.json

# Validate
laura-repair -data-dir ./mydb -operation validate

# Repair if needed
laura-repair -data-dir ./mydb -operation repair

# Restore if something goes wrong
laura-restore -data-dir ./mydb -input backup.json
```

### With CLI Tool

```bash
# Start laura-cli
laura-cli ./mydb

# In another terminal, run repair
laura-repair -data-dir ./mydb -operation validate
```

**Note**: For repair operations, database should be offline (close CLI first)

## Limitations

1. **Limited API access**: Some internal index structures not exposed through public API
2. **In-memory storage**: Defragmentation primarily optimizes logical structure
3. **No automatic scheduling**: Must run manually or via cron/systemd
4. **Single-threaded**: Repair operations are sequential
5. **No incremental repair**: Always validates/repairs entire collection or database

## Future Enhancements

Planned improvements:
- Automatic repair scheduling
- Incremental validation (only changed data)
- Parallel repair operations
- More granular repair options
- Corruption auto-detection
- Real-time health monitoring
- Integration with admin console UI

## See Also

- [Database Backup](../docs/backup.md) - Backup and restore documentation
- [CLI Tool](../docs/cli.md) - LauraDB CLI documentation
- [Indexing](../docs/indexing.md) - Index types and management
- [MVCC](../docs/mvcc.md) - Transaction and concurrency control
