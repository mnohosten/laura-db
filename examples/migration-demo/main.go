package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/migration"
)

func main() {
	fmt.Println("=== LauraDB Migration Demo ===")
	fmt.Println()

	// Clean up test directory
	dir := "./migration_demo_data"
	os.RemoveAll(dir)
	defer os.RemoveAll(dir)

	// Create migrations directory
	migrationsDir := filepath.Join(dir, "migrations")
	os.MkdirAll(migrationsDir, 0755)

	// Demo 1: Create and save migrations to files
	demo1CreateMigrationFiles(migrationsDir)

	// Demo 2: Load migrations and apply them
	demo2ApplyMigrations(dir, migrationsDir)

	// Demo 3: View migration status
	demo3MigrationStatus(dir, migrationsDir)

	// Demo 4: Rollback a migration
	demo4RollbackMigration(dir, migrationsDir)

	// Demo 5: Programmatic migrations
	demo5ProgrammaticMigrations(dir)

	fmt.Println("\n=== Demo Complete ===")
}

func demo1CreateMigrationFiles(migrationsDir string) {
	fmt.Println("--- Demo 1: Creating Migration Files ---")

	// Create first migration: setup users collection
	migration1 := &migration.Migration{
		Version:     time.Now().Unix(),
		Name:        "create_users_collection",
		Description: "Create users collection with email index",
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

	path1 := filepath.Join(migrationsDir, "001_create_users.json")
	if err := migration.SaveMigrationToFile(migration1, path1); err != nil {
		log.Fatalf("Failed to save migration: %v", err)
	}
	fmt.Printf("✓ Created migration file: %s\n", path1)

	// Wait a moment to ensure unique timestamps
	time.Sleep(time.Second)

	// Create second migration: add posts collection
	migration2 := &migration.Migration{
		Version:     time.Now().Unix(),
		Name:        "create_posts_collection",
		Description: "Create posts collection with author index",
		UpScript: map[string]interface{}{
			"operations": []interface{}{
				map[string]interface{}{
					"type": "create_collection",
					"name": "posts",
				},
				map[string]interface{}{
					"type":       "create_index",
					"collection": "posts",
					"field":      "author",
				},
			},
		},
		DownScript: map[string]interface{}{
			"operations": []interface{}{
				map[string]interface{}{
					"type": "drop_collection",
					"name": "posts",
				},
			},
		},
	}

	path2 := filepath.Join(migrationsDir, "002_create_posts.json")
	if err := migration.SaveMigrationToFile(migration2, path2); err != nil {
		log.Fatalf("Failed to save migration: %v", err)
	}
	fmt.Printf("✓ Created migration file: %s\n", path2)

	// Wait a moment to ensure unique timestamps
	time.Sleep(time.Second)

	// Create third migration: seed initial data
	migration3 := &migration.Migration{
		Version:     time.Now().Unix(),
		Name:        "seed_users",
		Description: "Add initial user accounts",
		UpScript: map[string]interface{}{
			"operations": []interface{}{
				map[string]interface{}{
					"type":       "insert_documents",
					"collection": "users",
					"documents": []interface{}{
						map[string]interface{}{
							"name":       "Alice",
							"email":      "alice@example.com",
							"created_at": time.Now(),
						},
						map[string]interface{}{
							"name":       "Bob",
							"email":      "bob@example.com",
							"created_at": time.Now(),
						},
					},
				},
			},
		},
		DownScript: map[string]interface{}{
			"operations": []interface{}{
				map[string]interface{}{
					"type":       "delete_documents",
					"collection": "users",
					"filter": map[string]interface{}{
						"email": map[string]interface{}{
							"$in": []interface{}{"alice@example.com", "bob@example.com"},
						},
					},
				},
			},
		},
	}

	path3 := filepath.Join(migrationsDir, "003_seed_users.json")
	if err := migration.SaveMigrationToFile(migration3, path3); err != nil {
		log.Fatalf("Failed to save migration: %v", err)
	}
	fmt.Printf("✓ Created migration file: %s\n\n", path3)
}

func demo2ApplyMigrations(dataDir, migrationsDir string) {
	fmt.Println("--- Demo 2: Applying Migrations ---")

	// Open database
	config := database.DefaultConfig(dataDir)
	db, err := database.Open(config)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create migrator
	migrator := migration.NewMigrator(db)

	// Load migrations from directory
	if err := migrator.LoadMigrationsFromDir(migrationsDir); err != nil {
		log.Fatalf("Failed to load migrations: %v", err)
	}
	fmt.Printf("✓ Loaded migrations from: %s\n", migrationsDir)

	// Apply all migrations
	if err := migrator.Up(); err != nil {
		log.Fatalf("Failed to apply migrations: %v", err)
	}
	fmt.Println("✓ Applied all migrations")
	fmt.Println()

	// Verify collections were created
	collections := db.ListCollections()
	fmt.Println("Collections created:")
	for _, name := range collections {
		if name != "_migrations" { // Skip internal collection
			fmt.Printf("  - %s\n", name)
		}
	}

	// Verify users were created
	users := db.Collection("users")
	docs, err := users.Find(nil)
	if err != nil {
		log.Fatalf("Failed to find users: %v", err)
	}
	fmt.Printf("\nUsers seeded: %d\n", len(docs))
	for _, doc := range docs {
		docMap := doc.ToMap()
		fmt.Printf("  - %s (%s)\n", docMap["name"], docMap["email"])
	}
	fmt.Println()
}

func demo3MigrationStatus(dataDir, migrationsDir string) {
	fmt.Println("--- Demo 3: Migration Status ---")

	// Open database
	config := database.DefaultConfig(dataDir)
	db, err := database.Open(config)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create migrator and load migrations
	migrator := migration.NewMigrator(db)
	if err := migrator.LoadMigrationsFromDir(migrationsDir); err != nil {
		log.Fatalf("Failed to load migrations: %v", err)
	}

	// Get migration status
	status, err := migrator.Status()
	if err != nil {
		log.Fatalf("Failed to get status: %v", err)
	}

	fmt.Printf("Total migrations: %d\n", status.TotalMigrations)
	fmt.Printf("Applied: %d\n", status.AppliedMigrations)
	fmt.Printf("Pending: %d\n\n", status.PendingMigrations)

	fmt.Println("Migration history:")
	for _, m := range status.Migrations {
		appliedStatus := "✗ Pending"
		if m.Applied {
			appliedStatus = "✓ Applied"
		}
		fmt.Printf("  %s - %s\n", appliedStatus, m.Name)
		fmt.Printf("    Description: %s\n", m.Description)
		fmt.Printf("    Version: %d\n\n", m.Version)
	}
}

func demo4RollbackMigration(dataDir, migrationsDir string) {
	fmt.Println("--- Demo 4: Rollback Migration ---")

	// Open database
	config := database.DefaultConfig(dataDir)
	db, err := database.Open(config)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create migrator and load migrations
	migrator := migration.NewMigrator(db)
	if err := migrator.LoadMigrationsFromDir(migrationsDir); err != nil {
		log.Fatalf("Failed to load migrations: %v", err)
	}

	// Check if there are any migrations to rollback
	history, err := migrator.GetMigrationHistory()
	if err != nil {
		log.Fatalf("Failed to get migration history: %v", err)
	}

	appliedCount := 0
	for _, h := range history {
		if h.Success {
			appliedCount++
		}
	}

	fmt.Printf("Applied migrations: %d\n", appliedCount)

	if appliedCount == 0 {
		fmt.Println("⚠ No migrations to rollback (this is expected if DB was closed)")
		fmt.Println()
		return
	}

	// Get pending migrations before rollback
	pending, err := migrator.GetPendingMigrations()
	if err != nil {
		log.Fatalf("Failed to get pending migrations: %v", err)
	}
	fmt.Printf("Pending migrations before rollback: %d\n", len(pending))

	// Rollback the last migration
	if err := migrator.Down(); err != nil {
		log.Fatalf("Failed to rollback: %v", err)
	}
	fmt.Println("✓ Rolled back last migration")

	// Verify users were removed
	users := db.Collection("users")
	docs, err := users.Find(nil)
	if err != nil {
		log.Fatalf("Failed to find users: %v", err)
	}
	fmt.Printf("Users after rollback: %d\n", len(docs))

	// Get pending migrations after rollback
	pending, err = migrator.GetPendingMigrations()
	if err != nil {
		log.Fatalf("Failed to get pending migrations: %v", err)
	}
	fmt.Printf("Pending migrations after rollback: %d\n\n", len(pending))
}

func demo5ProgrammaticMigrations(dataDir string) {
	fmt.Println("--- Demo 5: Programmatic Migrations ---")

	// Open database
	config := database.DefaultConfig(dataDir)
	db, err := database.Open(config)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create migrator
	migrator := migration.NewMigrator(db)

	// Add programmatic migration (no files needed)
	programmaticMigration := &migration.Migration{
		Version:     time.Now().Unix(),
		Name:        "add_comments_collection",
		Description: "Add comments collection with indexes",
		Up: func(db *database.Database) error {
			// Create collection
			coll, err := db.CreateCollection("comments")
			if err != nil {
				return err
			}

			// Create indexes
			if err := coll.CreateIndex("post_id", false); err != nil {
				return err
			}
			if err := coll.CreateIndex("author", false); err != nil {
				return err
			}

			// Insert sample comment
			_, err = coll.InsertOne(map[string]interface{}{
				"post_id":    "post123",
				"author":     "Alice",
				"content":    "Great post!",
				"created_at": time.Now(),
			})
			return err
		},
		Down: func(db *database.Database) error {
			return db.DropCollection("comments")
		},
	}

	if err := migrator.AddMigration(programmaticMigration); err != nil {
		log.Fatalf("Failed to add migration: %v", err)
	}
	fmt.Println("✓ Added programmatic migration")

	// Apply migration
	if err := migrator.Up(); err != nil {
		log.Fatalf("Failed to apply migration: %v", err)
	}
	fmt.Println("✓ Applied programmatic migration")

	// Verify collection and data
	comments := db.Collection("comments")
	docs, err := comments.Find(nil)
	if err != nil {
		log.Fatalf("Failed to find comments: %v", err)
	}
	fmt.Printf("Comments created: %d\n", len(docs))

	indexes := comments.ListIndexes()
	fmt.Printf("Indexes created: %d\n", len(indexes))
	for _, idx := range indexes {
		if name, ok := idx["name"].(string); ok {
			fmt.Printf("  - %s\n", name)
		}
	}
	fmt.Println()
}
