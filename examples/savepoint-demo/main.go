package main

import (
	"fmt"
	"os"

	"github.com/mnohosten/laura-db/pkg/database"
)

func main() {
	// Create temporary directory for demo
	dataDir := "savepoint-demo-data"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	// Open database
	db, err := database.Open(database.DefaultConfig(dataDir))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	fmt.Println("=== LauraDB Savepoint Demo ===")
	fmt.Println()

	// Demo 1: Basic Savepoint Usage
	fmt.Println("Demo 1: Basic Savepoint - Insert with Rollback")
	fmt.Println("-----------------------------------------------")

	session := db.StartSession()

	// Insert first document
	id1, _ := session.InsertOne("users", map[string]interface{}{
		"name": "Alice",
		"age":  int64(30),
	})
	fmt.Printf("Inserted Alice (ID: %s)\n", id1)

	// Create savepoint before second insert
	session.CreateSavepoint("before_bob")
	fmt.Println("Created savepoint 'before_bob'")

	// Insert second document
	id2, _ := session.InsertOne("users", map[string]interface{}{
		"name": "Bob",
		"age":  int64(25),
	})
	fmt.Printf("Inserted Bob (ID: %s)\n", id2)

	// Rollback to savepoint (removes Bob)
	session.RollbackToSavepoint("before_bob")
	fmt.Println("Rolled back to 'before_bob' - Bob's insert undone")

	// Commit transaction (only Alice is committed)
	session.CommitTransaction()
	fmt.Println("Transaction committed - only Alice exists in database")

	// Verify
	coll := db.Collection("users")
	count, _ := coll.Count(map[string]interface{}{})
	fmt.Printf("Total users in database: %d (expected: 1)\n", count)
	fmt.Println()

	// Demo 2: Multiple Nested Savepoints
	fmt.Println("Demo 2: Multiple Nested Savepoints")
	fmt.Println("-----------------------------------")

	session2 := db.StartSession()

	// Start with balance = 1000
	accID, _ := session2.InsertOne("accounts", map[string]interface{}{
		"name":    "Trading Account",
		"balance": int64(1000),
	})
	fmt.Println("Created trading account with balance: 1000")

	// Trade 1: Deduct 100
	session2.CreateSavepoint("before_trade1")
	session2.UpdateOne("accounts", map[string]interface{}{"_id": accID}, map[string]interface{}{
		"$inc": map[string]interface{}{"balance": int64(-100)},
	})
	fmt.Println("Savepoint 'before_trade1' - Deducted 100 (balance: 900)")

	// Trade 2: Deduct another 200
	session2.CreateSavepoint("before_trade2")
	session2.UpdateOne("accounts", map[string]interface{}{"_id": accID}, map[string]interface{}{
		"$inc": map[string]interface{}{"balance": int64(-200)},
	})
	fmt.Println("Savepoint 'before_trade2' - Deducted 200 (balance: 700)")

	// Trade 3: Deduct another 300
	session2.CreateSavepoint("before_trade3")
	session2.UpdateOne("accounts", map[string]interface{}{"_id": accID}, map[string]interface{}{
		"$inc": map[string]interface{}{"balance": int64(-300)},
	})
	fmt.Println("Savepoint 'before_trade3' - Deducted 300 (balance: 400)")

	// Rollback trade 3 (undo last deduction)
	session2.RollbackToSavepoint("before_trade3")
	fmt.Println("Rolled back to 'before_trade3' - undid 300 deduction (balance: 700)")

	// Commit the transaction
	session2.CommitTransaction()
	fmt.Println("Transaction committed with balance: 700")
	fmt.Println()

	// Demo 3: Error Recovery with Savepoints
	fmt.Println("Demo 3: Error Recovery - Banking Transaction")
	fmt.Println("---------------------------------------------")

	// Setup two accounts
	coll2 := db.Collection("bank")
	acc1, _ := coll2.InsertOne(map[string]interface{}{
		"name":    "Account A",
		"balance": int64(1000),
	})
	acc2, _ := coll2.InsertOne(map[string]interface{}{
		"name":    "Account B",
		"balance": int64(500),
	})
	fmt.Println("Account A: 1000, Account B: 500")

	session3 := db.StartSession()

	// Transfer 200 from A to B
	fmt.Println("\nAttempting transfer: 200 from A to B")

	// Deduct from Account A
	session3.UpdateOne("bank", map[string]interface{}{"_id": acc1}, map[string]interface{}{
		"$inc": map[string]interface{}{"balance": int64(-200)},
	})
	fmt.Println("  - Deducted 200 from Account A")

	// Create savepoint before crediting (in case we need to retry)
	session3.CreateSavepoint("before_credit")

	// Simulate an error - wrong account, need to rollback and retry
	fmt.Println("  - ERROR: Credited wrong account!")
	session3.RollbackToSavepoint("before_credit")
	fmt.Println("  - Rolled back credit operation")

	// Retry with correct account
	session3.UpdateOne("bank", map[string]interface{}{"_id": acc2}, map[string]interface{}{
		"$inc": map[string]interface{}{"balance": int64(200)},
	})
	fmt.Println("  - Credited 200 to Account B (correct account)")

	// Commit the corrected transaction
	session3.CommitTransaction()
	fmt.Println("\nTransaction committed successfully!")
	fmt.Println("Final balances: Account A: 800, Account B: 700")
	fmt.Println()

	// Demo 4: Savepoint Management
	fmt.Println("Demo 4: Savepoint Management")
	fmt.Println("-----------------------------")

	session4 := db.StartSession()

	// Create multiple savepoints
	session4.CreateSavepoint("sp1")
	session4.CreateSavepoint("sp2")
	session4.CreateSavepoint("sp3")

	savepoints := session4.ListSavepoints()
	fmt.Printf("Active savepoints: %v\n", savepoints)

	// Release one savepoint
	session4.ReleaseSavepoint("sp2")
	savepoints = session4.ListSavepoints()
	fmt.Printf("After releasing 'sp2': %v\n", savepoints)

	// Rollback to sp1 (automatically removes sp3)
	session4.RollbackToSavepoint("sp1")
	savepoints = session4.ListSavepoints()
	fmt.Printf("After rollback to 'sp1': %v (sp3 auto-removed)\n", savepoints)

	session4.AbortTransaction()
	fmt.Println()

	// Demo 5: Complex Workflow with Validation Points
	fmt.Println("Demo 5: Complex Workflow with Validation Points")
	fmt.Println("------------------------------------------------")

	session5 := db.StartSession()

	// Step 1: Create order
	orderID, _ := session5.InsertOne("orders", map[string]interface{}{
		"product": "Laptop",
		"qty":     int64(5),
		"status":  "pending",
	})
	fmt.Println("Step 1: Created order for 5 laptops")
	session5.CreateSavepoint("order_created")

	// Step 2: Reserve inventory
	session5.InsertOne("inventory", map[string]interface{}{
		"order_id": orderID,
		"reserved": int64(5),
	})
	fmt.Println("Step 2: Reserved 5 units in inventory")
	session5.CreateSavepoint("inventory_reserved")

	// Step 3: Process payment (simulate failure)
	fmt.Println("Step 3: Processing payment... FAILED!")

	// Rollback inventory reservation but keep order
	session5.RollbackToSavepoint("order_created")
	fmt.Println("  - Rolled back inventory reservation")

	// Update order status to failed
	session5.UpdateOne("orders", map[string]interface{}{"_id": orderID}, map[string]interface{}{
		"$set": map[string]interface{}{"status": "payment_failed"},
	})
	fmt.Println("  - Updated order status to 'payment_failed'")

	// Commit the transaction (order exists with failed status, no inventory reserved)
	session5.CommitTransaction()
	fmt.Println("\nTransaction committed: Order saved with failed status, no inventory reserved")
	fmt.Println()

	fmt.Println("=== Savepoint Demo Complete ===")
	fmt.Println()
	fmt.Println("Key Features Demonstrated:")
	fmt.Println("1. Basic savepoint creation and rollback")
	fmt.Println("2. Multiple nested savepoints")
	fmt.Println("3. Error recovery without losing all work")
	fmt.Println("4. Savepoint lifecycle management (create, release, list)")
	fmt.Println("5. Complex workflows with validation points")
}
