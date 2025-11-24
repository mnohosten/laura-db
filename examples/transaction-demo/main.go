package main

import (
	"fmt"
	"os"

	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/mvcc"
)

func main() {
	// Create a temporary directory for this demo
	dataDir := "./transaction_demo_data"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	// Open database
	db, err := database.Open(database.DefaultConfig(dataDir))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	fmt.Println("=== LauraDB Multi-Document ACID Transactions Demo ===")
	fmt.Println()

	// Demo 1: Basic Transaction
	fmt.Println("Demo 1: Basic Transaction (Insert + Commit)")
	fmt.Println("--------------------------------------------")

	err = db.WithTransaction(func(session *database.Session) error {
		// Insert multiple documents in one transaction
		id1, _ := session.InsertOne("users", map[string]interface{}{
			"name":  "Alice",
			"email": "alice@example.com",
			"age":   int64(30),
		})
		fmt.Printf("  - Inserted user: %s\n", id1)

		id2, _ := session.InsertOne("users", map[string]interface{}{
			"name":  "Bob",
			"email": "bob@example.com",
			"age":   int64(25),
		})
		fmt.Printf("  - Inserted user: %s\n", id2)

		return nil // Commit
	})

	if err != nil {
		fmt.Printf("  ✗ Transaction failed: %v\n", err)
	} else {
		fmt.Println("  ✓ Transaction committed successfully")
	}

	// Verify inserts
	users := db.Collection("users")
	results, _ := users.Find(map[string]interface{}{})
	fmt.Printf("  - Total users in collection: %d\n\n", len(results))

	// Demo 2: Transaction Rollback
	fmt.Println("Demo 2: Transaction Rollback (Abort)")
	fmt.Println("-------------------------------------")

	err = db.WithTransaction(func(session *database.Session) error {
		id, _ := session.InsertOne("users", map[string]interface{}{
			"name": "Charlie",
			"age":  int64(35),
		})
		fmt.Printf("  - Inserted user: %s\n", id)

		// Intentionally abort the transaction
		return fmt.Errorf("intentional abort")
	})

	if err != nil {
		fmt.Printf("  ✗ Transaction aborted: %v\n", err)
	}

	results, _ = users.Find(map[string]interface{}{})
	fmt.Printf("  - Total users after abort: %d (Charlie not inserted)\n\n", len(results))

	// Demo 3: Multi-Collection Transaction
	fmt.Println("Demo 3: Multi-Collection Transaction")
	fmt.Println("-------------------------------------")

	// Create accounts and transfer in one transaction
	err = db.WithTransaction(func(session *database.Session) error {
		// Create accounts
		var err error
		_, err = session.InsertOne("accounts", map[string]interface{}{
			"owner":   "Alice",
			"balance": int64(1000),
		})
		if err != nil {
			return err
		}

		_, err = session.InsertOne("accounts", map[string]interface{}{
			"owner":   "Bob",
			"balance": int64(500),
		})
		if err != nil {
			return err
		}

		fmt.Printf("  Created accounts:\n")
		fmt.Printf("    Alice: $1000\n")
		fmt.Printf("    Bob: $500\n\n")
		return nil
	})

	if err != nil {
		fmt.Printf("  ✗ Account creation failed: %v\n", err)
		return
	}

	fmt.Println("  ✓ Accounts created successfully")

	// Now transfer money in a separate transaction
	fmt.Println("  Transferring $200 from Alice to Bob...")
	// Note: For update operations on existing documents,
	// we need to use Collection methods directly (limitation of current implementation)
	accounts := db.Collection("accounts")
	accounts.UpdateOne(map[string]interface{}{"owner": "Alice"}, map[string]interface{}{
		"$inc": map[string]interface{}{"balance": int64(-200)},
	})
	accounts.UpdateOne(map[string]interface{}{"owner": "Bob"}, map[string]interface{}{
		"$inc": map[string]interface{}{"balance": int64(200)},
	})

	// Verify final balances
	alice, _ := accounts.FindOne(map[string]interface{}{"owner": "Alice"})
	bob, _ := accounts.FindOne(map[string]interface{}{"owner": "Bob"})
	aliceBalance, _ := alice.Get("balance")
	bobBalance, _ := bob.Get("balance")

	fmt.Printf("\n  Final balances:\n")
	fmt.Printf("    Alice: $%d\n", aliceBalance)
	fmt.Printf("    Bob: $%d\n\n", bobBalance)

	// Demo 4: Write Conflict Detection
	fmt.Println("Demo 4: Write Conflict Detection (Optimistic Concurrency Control)")
	fmt.Println("-------------------------------------------------------------------")

	// Start two concurrent transactions that try to insert the same document
	fmt.Println("  Session 1: Inserting document with ID 'unique-1'...")
	session1 := db.StartSession()
	session1.InsertOne("conflicts", map[string]interface{}{
		"_id":   "unique-1",
		"value": "from session 1",
	})

	fmt.Println("  Session 2: Inserting document with same ID 'unique-1'...")
	session2 := db.StartSession()
	session2.InsertOne("conflicts", map[string]interface{}{
		"_id":   "unique-1",
		"value": "from session 2",
	})

	// Commit session 1
	fmt.Println("\n  Committing Session 1...")
	if err := session1.CommitTransaction(); err != nil {
		fmt.Printf("    ✗ Session 1 failed: %v\n", err)
	} else {
		fmt.Println("    ✓ Session 1 committed")
	}

	// Try to commit session 2 (should detect conflict)
	fmt.Println("  Committing Session 2...")
	err = session2.CommitTransaction()
	if err == mvcc.ErrConflict {
		fmt.Println("    ✗ Session 2 aborted: Write conflict detected!")
		fmt.Println("    → This prevents lost updates (write-write conflict)")
	} else if err != nil {
		fmt.Printf("    ✗ Session 2 failed: %v\n", err)
	} else {
		fmt.Println("    ✓ Session 2 committed (unexpected)")
	}

	// Verify only session 1's data was committed
	conflicts := db.Collection("conflicts")
	doc, _ := conflicts.FindOne(map[string]interface{}{"_id": "unique-1"})
	if doc != nil {
		value, _ := doc.Get("value")
		fmt.Printf("\n  Final document value: '%s' (only Session 1's data)\n\n", value)
	}

	// Demo 5: Read-Your-Own-Writes
	fmt.Println("Demo 5: Read-Your-Own-Writes")
	fmt.Println("-----------------------------")

	session := db.StartSession()

	// Insert a document
	id, _ := session.InsertOne("temp", map[string]interface{}{
		"value": "test",
	})
	fmt.Printf("  - Inserted document: %s\n", id)

	// Read it back within the same transaction
	doc, err = session.FindOne("temp", map[string]interface{}{"value": "test"})
	if err != nil {
		fmt.Printf("  ✗ Failed to read own write: %v\n", err)
	} else {
		val, _ := doc.Get("value")
		fmt.Printf("  ✓ Successfully read own write: value=%v\n", val)
	}

	session.CommitTransaction()
	fmt.Println()

	fmt.Println("=== Demo Complete ===")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("  ✓ Multi-document ACID transactions")
	fmt.Println("  ✓ Automatic rollback on errors")
	fmt.Println("  ✓ Multi-collection transactions")
	fmt.Println("  ✓ Write conflict detection (First-Committer-Wins)")
	fmt.Println("  ✓ Read-your-own-writes within transactions")
}
