# Migration Tools

LauraDB provides a comprehensive migration system for managing database schema changes and data transformations over time. Migrations allow you to version control your database structure and safely apply or rollback changes.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Migration Structure](#migration-structure)
- [Creating Migrations](#creating-migrations)
- [Applying Migrations](#applying-migrations)
- [Rolling Back Migrations](#rolling-back-migrations)
- [Migration Operations](#migration-operations)
- [Best Practices](#best-practices)
- [API Reference](#api-reference)
- [Examples](#examples)

## Overview

### What are Migrations?

Migrations are versioned changes to your database schema or data. Each migration consists of:
- **Up**: Instructions to apply the change (e.g., create a collection, add an index)
- **Down**: Instructions to rollback the change (undo the up operation)
- **Version**: Unique timestamp or sequence number
- **Name and Description**: Human-readable metadata

### Why Use Migrations?

- **Version Control**: Track database schema changes alongside code changes
- **Reproducibility**: Apply the same changes across development, staging, and production
- **Rollback**: Safely undo changes if something goes wrong
- **Team Collaboration**: Multiple developers can work on schema changes without conflicts
- **Documentation**: Migrations serve as a history of database changes

## Quick Start

### 1. Create a Migration

```go
package main

import (
    "github.com/mnohosten/laura-db/pkg/database"
    "github.com/mnohosten/laura-db/pkg/migration"
)

func main() {
    // Open database
    db, _ := database.Open(database.DefaultConfig("./data"))
    defer db.Close()

    // Create migrator
    migrator := migration.NewMigrator(db)

    // Add a migration
    m := &migration.Migration{
        Version:     1,
        Name:        "create_users",
        Description: "Create users collection with email index",
        Up: func(db *database.Database) error {
            coll, err := db.CreateCollection("users")
            if err != nil {
                return err
            }
            return coll.CreateIndex("email", true)
        },
        Down: func(db *database.Database) error {
            return db.DropCollection("users")
        },
    }

    migrator.AddMigration(m)

    // Apply migration
    migrator.Up()
}
```

### 2. Check Migration Status

```go
status, _ := migrator.Status()
fmt.Printf("Total: %d, Applied: %d, Pending: %d\n",
    status.TotalMigrations,
    status.AppliedMigrations,
    status.PendingMigrations)
```

### 3. Rollback a Migration

```go
// Rollback the last applied migration
migrator.Down()
```

## Migration Structure

### Programmatic Migrations

Migrations can be defined programmatically using Go functions:

```go
migration := &migration.Migration{
    Version:     time.Now().Unix(),  // Unique version number
    Name:        "migration_name",   // Short name
    Description: "What this does",   // Description
    Up: func(db *database.Database) error {
        // Apply changes
        return nil
    },
    Down: func(db *database.Database) error {
        // Rollback changes
        return nil
    },
}
```

### File-Based Migrations

Migrations can also be stored as JSON files:

```json
{
  "version": 1732435200,
  "name": "create_users_collection",
  "description": "Create users collection with email index",
  "up_script": {
    "operations": [
      {
        "type": "create_collection",
        "name": "users"
      },
      {
        "type": "create_index",
        "collection": "users",
        "field": "email",
        "unique": true
      }
    ]
  },
  "down_script": {
    "operations": [
      {
        "type": "drop_collection",
        "name": "users"
      }
    ]
  }
}
```

Save and load from files:

```go
// Save to file
migration.SaveMigrationToFile(m, "migrations/001_create_users.json")

// Load from file
m, _ := migration.LoadMigrationFromFile("migrations/001_create_users.json")

// Load all from directory
migrator.LoadMigrationsFromDir("./migrations")
```

## Creating Migrations

### Using the Helper Function

```go
m := migration.CreateMigration(
    "add_posts_collection",
    "Create posts collection with author index",
)

// Add operations to UpScript
m.UpScript["operations"] = []interface{}{
    map[string]interface{}{
        "type": "create_collection",
        "name": "posts",
    },
}

// Define Up and Down functions
m.Up = migration.CreateMigrationFunc(m.UpScript)
m.Down = migration.CreateMigrationFunc(m.DownScript)
```

### Naming Convention

A good migration naming convention includes:
- Sequential number: `001_`, `002_`, `003_`
- Timestamp: `20251124120000_`
- Descriptive name: `create_users`, `add_email_index`

Example: `001_create_users.json` or `20251124120000_create_users.json`

### Version Numbers

Version numbers should be:
- **Unique**: No two migrations can have the same version
- **Sequential**: Applied in order from lowest to highest
- **Immutable**: Never change after deployment

Common strategies:
- Unix timestamps: `time.Now().Unix()` (automatically unique if created sequentially)
- Sequential integers: `1`, `2`, `3`, ... (simple but requires coordination)
- Date-based: `20251124001`, `20251124002` (year+month+day+sequence)

## Applying Migrations

### Apply All Pending Migrations

```go
err := migrator.Up()
```

This applies all migrations that haven't been applied yet, in version order.

### Check Which Migrations Will Be Applied

```go
pending, _ := migrator.GetPendingMigrations()
for _, m := range pending {
    fmt.Printf("Will apply: %s (v%d)\n", m.Name, m.Version)
}
```

### Migration History

LauraDB tracks migration history in the `_migrations` collection:

```go
history, _ := migrator.GetMigrationHistory()
for _, h := range history {
    fmt.Printf("%s - Applied: %v at %v\n",
        h.Name, h.Success, h.AppliedAt)
}
```

## Rolling Back Migrations

### Rollback the Last Migration

```go
err := migrator.Down()
```

This rolls back the most recently applied migration using its `Down` function.

### Important Notes on Rollbacks

- Only the **last applied** migration can be rolled back
- Rollbacks should be tested in development before using in production
- Some changes (like data deletions) may not be fully reversible
- The migration is removed from history after successful rollback

## Migration Operations

### Supported Operations in JSON Migrations

#### create_collection

Creates a new collection.

```json
{
  "type": "create_collection",
  "name": "users"
}
```

#### drop_collection

Drops an existing collection.

```json
{
  "type": "drop_collection",
  "name": "users"
}
```

#### create_index

Creates an index on a collection.

```json
{
  "type": "create_index",
  "collection": "users",
  "field": "email",
  "unique": true,
  "index_type": "btree"
}
```

Supported index types:
- `btree` (default): Regular B+ tree index
- `text`: Text index for full-text search
- `geo_2d`: 2D geospatial index
- `geo_2dsphere`: 2D sphere geospatial index

#### drop_index

Drops an index from a collection.

```json
{
  "type": "drop_index",
  "collection": "users",
  "name": "email_1"
}
```

#### rename_collection

Renames a collection.

```json
{
  "type": "rename_collection",
  "old_name": "users",
  "new_name": "people"
}
```

#### insert_documents

Inserts documents into a collection.

```json
{
  "type": "insert_documents",
  "collection": "users",
  "documents": [
    {
      "name": "Alice",
      "email": "alice@example.com"
    },
    {
      "name": "Bob",
      "email": "bob@example.com"
    }
  ]
}
```

#### update_documents

Updates documents in a collection.

```json
{
  "type": "update_documents",
  "collection": "users",
  "filter": {
    "status": "inactive"
  },
  "update": {
    "$set": {
      "status": "archived"
    }
  }
}
```

#### delete_documents

Deletes documents from a collection.

```json
{
  "type": "delete_documents",
  "collection": "users",
  "filter": {
    "status": "archived"
  }
}
```

## Best Practices

### 1. Always Include Down Migrations

Every `Up` migration should have a corresponding `Down` migration:

```go
m := &migration.Migration{
    Up: func(db *database.Database) error {
        _, err := db.CreateCollection("users")
        return err
    },
    Down: func(db *database.Database) error {
        return db.DropCollection("users")
    },
}
```

### 2. Make Migrations Idempotent

Migrations should handle being run multiple times:

```go
Up: func(db *database.Database) error {
    // Check if collection exists first
    collections := db.ListCollections()
    for _, name := range collections {
        if name == "users" {
            return nil // Already exists
        }
    }
    _, err := db.CreateCollection("users")
    return err
}
```

### 3. Use Descriptive Names

Good: `"create_users_table_with_email_index"`
Bad: `"migration_1"`

### 4. Keep Migrations Small

Each migration should do one logical change. Split large changes into multiple migrations.

### 5. Test Migrations Before Production

Always test both `Up` and `Down` in development:

```go
// Test in development
migrator.Up()
// Verify changes
migrator.Down()
// Verify rollback worked
```

### 6. Never Modify Deployed Migrations

Once a migration is deployed to production, never modify it. Create a new migration instead.

### 7. Use Version Control

Store migration files in git alongside your code:

```
project/
  migrations/
    001_create_users.json
    002_add_posts.json
    003_seed_data.json
```

### 8. Handle Errors Gracefully

```go
Up: func(db *database.Database) error {
    coll, err := db.CreateCollection("users")
    if err != nil {
        return fmt.Errorf("failed to create users collection: %w", err)
    }

    if err := coll.CreateIndex("email", true); err != nil {
        // Cleanup on error
        db.DropCollection("users")
        return fmt.Errorf("failed to create email index: %w", err)
    }

    return nil
}
```

### 9. Document Complex Migrations

Add comments explaining why a migration exists:

```go
// Migration to fix data inconsistency from v1.2.0
// Normalizes email addresses to lowercase
Up: func(db *database.Database) error {
    // Implementation
}
```

### 10. Backup Before Large Migrations

For migrations that modify or delete data:

```go
// Create backup first
backup, _ := db.Backup()
// ... save backup ...

// Then apply migration
migrator.Up()
```

## API Reference

### Migration

```go
type Migration struct {
    Version     int64                  // Unique version number
    Name        string                 // Migration name
    Description string                 // What this migration does
    Up          MigrationFunc          // Function to apply migration
    Down        MigrationFunc          // Function to rollback
    UpScript    map[string]interface{} // JSON operations for Up
    DownScript  map[string]interface{} // JSON operations for Down
}
```

### Migrator

```go
type Migrator struct {
    // Methods
    AddMigration(m *Migration) error
    LoadMigrationsFromDir(dir string) error
    Up() error
    Down() error
    Status() (*MigrationStatus, error)
    GetPendingMigrations() ([]*Migration, error)
    GetMigrationHistory() ([]*MigrationHistory, error)
}

// Create a new migrator
migrator := migration.NewMigrator(db)
```

### Helper Functions

```go
// Create a new migration
CreateMigration(name, description string) *Migration

// Save/load migrations
SaveMigrationToFile(m *Migration, path string) error
LoadMigrationFromFile(path string) (*Migration, error)
```

### Database Extension

```go
// Rename a collection
db.RenameCollection(oldName, newName string) error
```

## Examples

### Example 1: Simple Collection Creation

```go
m := &migration.Migration{
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
```

### Example 2: Adding Indexes

```go
m := &migration.Migration{
    Version:     2,
    Name:        "add_email_index",
    Description: "Add unique email index to users",
    Up: func(db *database.Database) error {
        coll := db.Collection("users")
        return coll.CreateIndex("email", true)
    },
    Down: func(db *database.Database) error {
        coll := db.Collection("users")
        return coll.DropIndex("email_1")
    },
}
```

### Example 3: Data Transformation

```go
m := &migration.Migration{
    Version:     3,
    Name:        "normalize_emails",
    Description: "Convert all emails to lowercase",
    Up: func(db *database.Database) error {
        coll := db.Collection("users")

        // Find all users
        docs, err := coll.Find(nil)
        if err != nil {
            return err
        }

        // Update each user
        for _, doc := range docs {
            docMap := doc.ToMap()
            if email, ok := docMap["email"].(string); ok {
                _, err := coll.UpdateOne(
                    map[string]interface{}{"_id": docMap["_id"]},
                    map[string]interface{}{
                        "$set": map[string]interface{}{
                            "email": strings.ToLower(email),
                        },
                    },
                )
                if err != nil {
                    return err
                }
            }
        }

        return nil
    },
    Down: func(db *database.Database) error {
        // Emails can't be reliably restored, so just warn
        fmt.Println("Warning: Email case normalization is not reversible")
        return nil
    },
}
```

### Example 4: Multiple Collections

```go
m := &migration.Migration{
    Version:     4,
    Name:        "create_blog_schema",
    Description: "Create posts and comments collections",
    Up: func(db *database.Database) error {
        // Create posts collection
        posts, err := db.CreateCollection("posts")
        if err != nil {
            return err
        }
        if err := posts.CreateIndex("author", false); err != nil {
            return err
        }

        // Create comments collection
        comments, err := db.CreateCollection("comments")
        if err != nil {
            return err
        }
        if err := comments.CreateIndex("post_id", false); err != nil {
            return err
        }

        return nil
    },
    Down: func(db *database.Database) error {
        if err := db.DropCollection("comments"); err != nil {
            return err
        }
        return db.DropCollection("posts")
    },
}
```

### Example 5: Using JSON Files

Create `migrations/005_add_timestamps.json`:

```json
{
  "version": 5,
  "name": "add_timestamps",
  "description": "Add created_at field to existing users",
  "up_script": {
    "operations": [
      {
        "type": "update_documents",
        "collection": "users",
        "filter": {
          "created_at": {
            "$exists": false
          }
        },
        "update": {
          "$currentDate": {
            "created_at": true
          }
        }
      }
    ]
  },
  "down_script": {
    "operations": [
      {
        "type": "update_documents",
        "collection": "users",
        "filter": {},
        "update": {
          "$unset": {
            "created_at": ""
          }
        }
      }
    ]
  }
}
```

Load and apply:

```go
migrator.LoadMigrationsFromDir("./migrations")
migrator.Up()
```

## Performance Considerations

### Large Data Migrations

For migrations affecting many documents:

1. **Batch Processing**: Process documents in batches
2. **Indexes**: Drop indexes before bulk operations, recreate after
3. **Progress Tracking**: Log progress for long-running migrations
4. **Timeouts**: Consider migration timeout limits

Example:

```go
Up: func(db *database.Database) error {
    coll := db.Collection("users")

    // Drop index temporarily
    coll.DropIndex("email_1")
    defer coll.CreateIndex("email", true)

    // Process in batches
    batchSize := 1000
    offset := 0

    for {
        docs, _ := coll.FindWithOptions(nil, &database.FindOptions{
            Skip:  int64(offset),
            Limit: int64(batchSize),
        })

        if len(docs) == 0 {
            break
        }

        // Process batch...

        offset += batchSize
    }

    return nil
}
```

## Troubleshooting

### Migration Failed Midway

If a migration fails partway through:

1. Check the error message in the migration history
2. Manually inspect the database state
3. Fix the issue or rollback manually if needed
4. Update the migration to handle edge cases
5. Re-apply from a clean state

### Duplicate Version Numbers

Error: `migration with version X already exists`

Solution: Ensure each migration has a unique version number.

### Rollback Not Working

Common issues:
- `Down` function not properly implemented
- Manual changes to database after migration
- Data that can't be restored (deleted data)

Prevention: Always test rollbacks in development first.

## Conclusion

LauraDB's migration system provides a robust way to manage database schema evolution. By following best practices and using the provided tools, you can safely evolve your database schema alongside your application code.

For more examples, see:
- `examples/migration-demo/` - Complete working examples
- `pkg/migration/migration_test.go` - Unit tests with various scenarios
