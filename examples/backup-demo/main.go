package main

import (
	"fmt"
	"log"
	"os"

	"github.com/mnohosten/laura-db/pkg/backup"
	"github.com/mnohosten/laura-db/pkg/database"
)

func main() {
	fmt.Println("=== LauraDB Backup and Restore Demo ===")
	fmt.Println()

	// Create temporary directories for demo
	dataDir1 := "./data/backup-demo-1"
	dataDir2 := "./data/backup-demo-2"
	backupPath := "./data/backups/mydb-backup.json"

	// Clean up old demo data
	os.RemoveAll("./data/backup-demo-1")
	os.RemoveAll("./data/backup-demo-2")
	os.RemoveAll("./data/backups")

	// Demo 1: Create a database and populate it with data
	fmt.Println("1. Creating database and adding data...")
	db1, err := database.Open(database.DefaultConfig(dataDir1))
	if err != nil {
		log.Fatal(err)
	}

	// Create a users collection
	users := db1.Collection("users")

	// Insert sample documents
	users.InsertOne(map[string]interface{}{
		"name":  "Alice",
		"email": "alice@example.com",
		"age":   int64(30),
		"role":  "admin",
	})

	users.InsertOne(map[string]interface{}{
		"name":  "Bob",
		"email": "bob@example.com",
		"age":   int64(25),
		"role":  "user",
	})

	users.InsertOne(map[string]interface{}{
		"name":  "Charlie",
		"email": "charlie@example.com",
		"age":   int64(35),
		"role":  "moderator",
	})

	// Create indexes
	users.CreateIndex("email", true) // Unique index on email
	users.CreateCompoundIndex([]string{"role", "age"}, false)

	// Create a products collection
	products := db1.Collection("products")

	products.InsertOne(map[string]interface{}{
		"name":  "Widget",
		"price": int64(1999),
		"stock": int64(100),
	})

	products.InsertOne(map[string]interface{}{
		"name":  "Gadget",
		"price": int64(2999),
		"stock": int64(50),
	})

	products.CreateIndex("name", false)

	userDocs, _ := users.Find(map[string]interface{}{})
	productDocs, _ := products.Find(map[string]interface{}{})
	fmt.Printf("   - Created 'users' collection with %d documents and 2 indexes\n", len(userDocs))
	fmt.Printf("   - Created 'products' collection with %d documents and 1 index\n\n", len(productDocs))

	// Demo 2: Create a backup
	fmt.Println("2. Creating backup...")
	err = db1.BackupToFile(backupPath, true) // pretty=true for readable JSON
	if err != nil {
		log.Fatal(err)
	}

	// Get backup stats
	restorer := backup.NewRestorer()
	backupFormat, err := restorer.RestoreFromFile(backupPath)
	if err != nil {
		log.Fatal(err)
	}

	stats := backupFormat.Stats()
	fmt.Printf("   - Backup created: %s\n", backupPath)
	fmt.Printf("   - Database: %s\n", stats["database_name"])
	fmt.Printf("   - Collections: %d\n", stats["collections"])
	fmt.Printf("   - Total documents: %d\n", stats["total_documents"])
	fmt.Printf("   - Total indexes: %d\n", stats["total_indexes"])
	fmt.Printf("   - Timestamp: %s\n\n", backupFormat.Timestamp.Format("2006-01-02 15:04:05"))

	db1.Close()

	// Demo 3: Restore to a new database
	fmt.Println("3. Restoring backup to a new database...")
	db2, err := database.Open(database.DefaultConfig(dataDir2))
	if err != nil {
		log.Fatal(err)
	}
	defer db2.Close()

	opts := backup.DefaultRestoreOptions()
	err = db2.RestoreFromFile(backupPath, opts)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("   - Backup restored successfully")
	fmt.Println()

	// Demo 4: Verify restored data
	fmt.Println("4. Verifying restored data...")

	usersRestored := db2.Collection("users")
	productsRestored := db2.Collection("products")

	// Check document counts
	usersRestoredDocs, _ := usersRestored.Find(map[string]interface{}{})
	productsRestoredDocs, _ := productsRestored.Find(map[string]interface{}{})
	usersCount := len(usersRestoredDocs)
	productsCount := len(productsRestoredDocs)

	fmt.Printf("   - Users collection: %d documents\n", usersCount)
	fmt.Printf("   - Products collection: %d documents\n", productsCount)

	// Check indexes
	usersStats := usersRestored.Stats()
	productsStats := productsRestored.Stats()

	fmt.Printf("   - Users indexes: %d (including default _id_)\n", usersStats["index_count"])
	fmt.Printf("   - Products indexes: %d (including default _id_)\n\n", productsStats["index_count"])

	// Demo 5: Query restored data
	fmt.Println("5. Querying restored data...")

	// Find a specific user
	alice, err := usersRestored.FindOne(map[string]interface{}{"name": "Alice"})
	if err != nil {
		log.Fatal(err)
	}
	name, _ := alice.Get("name")
	email, _ := alice.Get("email")
	role, _ := alice.Get("role")
	fmt.Printf("   - Found user: %s <%s> - Role: %s\n", name, email, role)

	// Find all products
	allProducts, _ := productsRestored.Find(map[string]interface{}{})
	fmt.Printf("   - Found %d products:\n", len(allProducts))
	for _, product := range allProducts {
		prodName, _ := product.Get("name")
		price, _ := product.Get("price")
		stock, _ := product.Get("stock")
		fmt.Printf("     * %s - Price: $%.2f - Stock: %d\n", prodName, float64(price.(int64))/100, stock)
	}

	fmt.Println("\n6. Cleanup...")
	fmt.Println("   - Removing demo data directories")

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("  ✓ Creating and populating a database")
	fmt.Println("  ✓ Creating various types of indexes")
	fmt.Println("  ✓ Backing up entire database to JSON file")
	fmt.Println("  ✓ Restoring backup to a new database")
	fmt.Println("  ✓ Verifying data integrity after restore")
	fmt.Println("  ✓ Querying restored data")

	fmt.Println("\nBackup file location:", backupPath)
	fmt.Println("You can inspect the backup file to see the JSON structure.")
}
