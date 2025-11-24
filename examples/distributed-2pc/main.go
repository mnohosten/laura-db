package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/distributed"
	"github.com/mnohosten/laura-db/pkg/mvcc"
)

func main() {
	fmt.Println("LauraDB - Two-Phase Commit (2PC) Demo")
	fmt.Println("======================================")
	fmt.Println()

	// Clean up any existing test data
	os.RemoveAll("./data/bank1")
	os.RemoveAll("./data/bank2")
	os.RemoveAll("./data/clearing_house")

	// Demo 1: Successful distributed transaction
	fmt.Println("Demo 1: Successful Distributed Bank Transfer")
	fmt.Println("---------------------------------------------")
	demo1SuccessfulTransfer()

	fmt.Println()

	// Demo 2: Aborted distributed transaction
	fmt.Println("Demo 2: Aborted Distributed Transaction")
	fmt.Println("----------------------------------------")
	demo2AbortedTransaction()

	fmt.Println()

	// Demo 3: Multi-database e-commerce order
	fmt.Println("Demo 3: Multi-Database E-Commerce Order")
	fmt.Println("----------------------------------------")
	demo3EcommerceOrder()

	fmt.Println()

	// Demo 4: Write conflict detection
	fmt.Println("Demo 4: Write Conflict Detection")
	fmt.Println("---------------------------------")
	demo4WriteConflict()

	// Clean up
	os.RemoveAll("./data")
}

func demo1SuccessfulTransfer() {
	// Set up three databases representing different banking services
	bank1DB, err := database.Open(database.DefaultConfig("./data/bank1"))
	if err != nil {
		log.Fatal(err)
	}
	defer bank1DB.Close()

	bank2DB, err := database.Open(database.DefaultConfig("./data/bank2"))
	if err != nil {
		log.Fatal(err)
	}
	defer bank2DB.Close()

	clearingHouseDB, err := database.Open(database.DefaultConfig("./data/clearing_house"))
	if err != nil {
		log.Fatal(err)
	}
	defer clearingHouseDB.Close()

	// Initialize account balances
	bank1Coll := bank1DB.Collection("accounts")
	bank1Coll.InsertOne(map[string]interface{}{
		"account_id": "BANK1-12345",
		"balance":    int64(10000),
		"name":       "Alice Smith",
	})

	bank2Coll := bank2DB.Collection("accounts")
	bank2Coll.InsertOne(map[string]interface{}{
		"account_id": "BANK2-67890",
		"balance":    int64(5000),
		"name":       "Bob Johnson",
	})

	fmt.Println("Initial balances:")
	fmt.Println("  Alice (Bank1): $10,000")
	fmt.Println("  Bob (Bank2):   $5,000")
	fmt.Println()

	// Create participants
	bank1Participant := distributed.NewDatabaseParticipant("bank1", bank1DB)
	bank2Participant := distributed.NewDatabaseParticipant("bank2", bank2DB)
	clearingHouseParticipant := distributed.NewDatabaseParticipant("clearing_house", clearingHouseDB)

	// Create 2PC coordinator
	txnID := mvcc.TxnID(1)
	coordinator := distributed.NewCoordinator(txnID, 0)

	coordinator.AddParticipant(bank1Participant)
	coordinator.AddParticipant(bank2Participant)
	coordinator.AddParticipant(clearingHouseParticipant)

	// Start sessions
	bank1Session := bank1Participant.StartTransaction(txnID)
	bank2Session := bank2Participant.StartTransaction(txnID)
	clearingHouseSession := clearingHouseParticipant.StartTransaction(txnID)

	// Transfer $3,000 from Alice to Bob
	transferAmount := int64(3000)
	fmt.Printf("Transferring $3,000 from Alice to Bob...\n")

	// Debit Bank1 (Alice)
	err = bank1Session.UpdateOne("accounts",
		map[string]interface{}{"account_id": "BANK1-12345"},
		map[string]interface{}{"$inc": map[string]interface{}{"balance": -transferAmount}},
	)
	if err != nil {
		log.Fatal(err)
	}

	// Credit Bank2 (Bob)
	err = bank2Session.UpdateOne("accounts",
		map[string]interface{}{"account_id": "BANK2-67890"},
		map[string]interface{}{"$inc": map[string]interface{}{"balance": transferAmount}},
	)
	if err != nil {
		log.Fatal(err)
	}

	// Record in clearing house
	_, err = clearingHouseSession.InsertOne("transfers", map[string]interface{}{
		"transfer_id": "TXN-001",
		"from_bank":   "bank1",
		"to_bank":     "bank2",
		"amount":      transferAmount,
		"status":      "completed",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Execute 2PC protocol
	fmt.Println("Executing two-phase commit protocol...")
	fmt.Println("  Phase 1: Prepare - asking all participants if ready to commit")

	ctx := context.Background()
	if err := coordinator.Execute(ctx); err != nil {
		log.Fatalf("2PC failed: %v", err)
	}

	fmt.Println("  Phase 2: Commit - all participants voted YES, committing transaction")
	fmt.Println("✓ Transaction committed successfully!")
	fmt.Println()

	// Verify final balances
	aliceDoc, _ := bank1Coll.FindOne(map[string]interface{}{"account_id": "BANK1-12345"})
	aliceBalance, _ := aliceDoc.Get("balance")

	bobDoc, _ := bank2Coll.FindOne(map[string]interface{}{"account_id": "BANK2-67890"})
	bobBalance, _ := bobDoc.Get("balance")

	fmt.Println("Final balances:")
	fmt.Printf("  Alice (Bank1): $%d\n", aliceBalance)
	fmt.Printf("  Bob (Bank2):   $%d\n", bobBalance)
	fmt.Println()

	// Verify clearing house record
	clearingColl := clearingHouseDB.Collection("transfers")
	transferDoc, _ := clearingColl.FindOne(map[string]interface{}{"transfer_id": "TXN-001"})
	status, _ := transferDoc.Get("status")
	fmt.Printf("Clearing house status: %s\n", status)
}

func demo2AbortedTransaction() {
	// Set up databases
	db1, err := database.Open(database.DefaultConfig("./data/demo2_db1"))
	if err != nil {
		log.Fatal(err)
	}
	defer db1.Close()

	db2, err := database.Open(database.DefaultConfig("./data/demo2_db2"))
	if err != nil {
		log.Fatal(err)
	}
	defer db2.Close()

	// Initialize data
	coll1 := db1.Collection("inventory")
	coll1.InsertOne(map[string]interface{}{
		"product_id": "PROD-001",
		"quantity":   int64(100),
	})

	fmt.Println("Initial inventory:")
	fmt.Println("  Product PROD-001: 100 units")
	fmt.Println()

	// Create participants
	p1 := distributed.NewDatabaseParticipant("inventory_db", db1)
	p2 := distributed.NewDatabaseParticipant("orders_db", db2)

	// Create a mock participant that will vote NO
	mockParticipant := &MockParticipant{
		ID_:             "payment_gateway",
		PrepareResponse: false, // Will vote NO
	}

	// Create coordinator
	txnID := mvcc.TxnID(2)
	coordinator := distributed.NewCoordinator(txnID, 0)
	coordinator.AddParticipant(p1)
	coordinator.AddParticipant(p2)
	coordinator.AddParticipant(mockParticipant)

	// Start sessions
	inventorySession := p1.StartTransaction(txnID)
	ordersSession := p2.StartTransaction(txnID)

	// Try to process an order
	fmt.Println("Processing order for 50 units...")

	// Deduct inventory
	err = inventorySession.UpdateOne("inventory",
		map[string]interface{}{"product_id": "PROD-001"},
		map[string]interface{}{"$inc": map[string]interface{}{"quantity": int64(-50)}},
	)
	if err != nil {
		log.Fatal(err)
	}

	// Create order
	_, err = ordersSession.InsertOne("orders", map[string]interface{}{
		"order_id": "ORD-001",
		"quantity": int64(50),
		"status":   "pending",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Executing two-phase commit protocol...")
	fmt.Println("  Phase 1: Prepare - payment_gateway votes NO (payment declined)")

	// Execute 2PC - will abort because mock votes NO
	ctx := context.Background()
	err = coordinator.Execute(ctx)
	if err != nil {
		fmt.Printf("✗ Transaction aborted: %v\n", err)
	}
	fmt.Println()

	// Verify inventory unchanged
	prodDoc, _ := coll1.FindOne(map[string]interface{}{"product_id": "PROD-001"})
	quantity, _ := prodDoc.Get("quantity")

	fmt.Println("Final state:")
	fmt.Printf("  Product PROD-001: %d units (unchanged - transaction rolled back)\n", quantity)

	// Verify order was not created
	ordersColl := db2.Collection("orders")
	_, err = ordersColl.FindOne(map[string]interface{}{"order_id": "ORD-001"})
	if err != nil {
		fmt.Println("  Order ORD-001: not created (correctly aborted)")
	}
}

func demo3EcommerceOrder() {
	// Set up databases for different microservices
	inventoryDB, _ := database.Open(database.DefaultConfig("./data/inventory_service"))
	defer inventoryDB.Close()

	ordersDB, _ := database.Open(database.DefaultConfig("./data/orders_service"))
	defer ordersDB.Close()

	paymentsDB, _ := database.Open(database.DefaultConfig("./data/payments_service"))
	defer paymentsDB.Close()

	shippingDB, _ := database.Open(database.DefaultConfig("./data/shipping_service"))
	defer shippingDB.Close()

	// Initialize inventory
	inventoryColl := inventoryDB.Collection("products")
	inventoryColl.InsertOne(map[string]interface{}{
		"sku":      "LAPTOP-X1",
		"name":     "UltraBook Pro",
		"quantity": int64(25),
		"price":    int64(1299),
	})

	fmt.Println("E-commerce order processing across 4 microservices:")
	fmt.Println("  - Inventory Service")
	fmt.Println("  - Orders Service")
	fmt.Println("  - Payments Service")
	fmt.Println("  - Shipping Service")
	fmt.Println()

	// Create participants
	inventoryP := distributed.NewDatabaseParticipant("inventory", inventoryDB)
	ordersP := distributed.NewDatabaseParticipant("orders", ordersDB)
	paymentsP := distributed.NewDatabaseParticipant("payments", paymentsDB)
	shippingP := distributed.NewDatabaseParticipant("shipping", shippingDB)

	// Create coordinator
	txnID := mvcc.TxnID(3)
	coordinator := distributed.NewCoordinator(txnID, 0)
	coordinator.AddParticipant(inventoryP)
	coordinator.AddParticipant(ordersP)
	coordinator.AddParticipant(paymentsP)
	coordinator.AddParticipant(shippingP)

	// Start sessions
	inventorySession := inventoryP.StartTransaction(txnID)
	ordersSession := ordersP.StartTransaction(txnID)
	paymentsSession := paymentsP.StartTransaction(txnID)
	shippingSession := shippingP.StartTransaction(txnID)

	// Process order for 2 laptops
	quantity := int64(2)
	price := int64(1299)
	totalAmount := quantity * price

	fmt.Printf("Customer order: %d x UltraBook Pro @ $%d = $%d\n", quantity, price, totalAmount)
	fmt.Println()

	// Deduct inventory
	inventorySession.UpdateOne("products",
		map[string]interface{}{"sku": "LAPTOP-X1"},
		map[string]interface{}{"$inc": map[string]interface{}{"quantity": -quantity}},
	)

	// Create order
	ordersSession.InsertOne("orders", map[string]interface{}{
		"order_id":   "ORD-2024-001",
		"sku":        "LAPTOP-X1",
		"quantity":   quantity,
		"total":      totalAmount,
		"status":     "confirmed",
		"customer":   "customer@example.com",
	})

	// Process payment
	paymentsSession.InsertOne("transactions", map[string]interface{}{
		"payment_id": "PAY-001",
		"order_id":   "ORD-2024-001",
		"amount":     totalAmount,
		"status":     "captured",
	})

	// Create shipping label
	shippingSession.InsertOne("shipments", map[string]interface{}{
		"shipment_id": "SHIP-001",
		"order_id":    "ORD-2024-001",
		"address":     "123 Main St, Anytown, USA",
		"status":      "label_created",
	})

	fmt.Println("Executing distributed transaction across all services...")

	// Execute 2PC
	ctx := context.Background()
	if err := coordinator.Execute(ctx); err != nil {
		log.Fatalf("2PC failed: %v", err)
	}

	fmt.Println("✓ Order processed successfully across all services!")
	fmt.Println()

	// Verify all updates
	inventoryDoc, _ := inventoryColl.FindOne(map[string]interface{}{"sku": "LAPTOP-X1"})
	remainingQty, _ := inventoryDoc.Get("quantity")
	fmt.Printf("Inventory: %d units remaining\n", remainingQty)

	ordersColl := ordersDB.Collection("orders")
	orderDoc, _ := ordersColl.FindOne(map[string]interface{}{"order_id": "ORD-2024-001"})
	orderStatus, _ := orderDoc.Get("status")
	fmt.Printf("Order: %s\n", orderStatus)

	paymentsColl := paymentsDB.Collection("transactions")
	paymentDoc, _ := paymentsColl.FindOne(map[string]interface{}{"payment_id": "PAY-001"})
	paymentStatus, _ := paymentDoc.Get("status")
	fmt.Printf("Payment: %s\n", paymentStatus)

	shippingColl := shippingDB.Collection("shipments")
	shipmentDoc, _ := shippingColl.FindOne(map[string]interface{}{"shipment_id": "SHIP-001"})
	shipmentStatus, _ := shipmentDoc.Get("status")
	fmt.Printf("Shipping: %s\n", shipmentStatus)
}

func demo4WriteConflict() {
	// Set up database
	db, _ := database.Open(database.DefaultConfig("./data/conflict_demo"))
	defer db.Close()

	// Initialize account
	coll := db.Collection("accounts")
	docID, _ := coll.InsertOne(map[string]interface{}{
		"account_id": "ACC-999",
		"balance":    int64(1000),
	})

	fmt.Println("Initial balance: $1,000")
	fmt.Println()

	// Create participant
	participant := distributed.NewDatabaseParticipant("db", db)

	// Create coordinator
	txnID := mvcc.TxnID(4)
	coordinator := distributed.NewCoordinator(txnID, 0)
	coordinator.AddParticipant(participant)

	// Start session
	session := participant.StartTransaction(txnID)

	// Read balance in transaction
	doc, _ := session.FindOne("accounts", map[string]interface{}{"_id": docID})
	balance, _ := doc.Get("balance")
	fmt.Printf("Transaction reads balance: $%d\n", balance)

	// Concurrent update outside transaction (simulating another process)
	fmt.Println("Another process updates the balance to $1,500...")
	coll.UpdateOne(
		map[string]interface{}{"_id": docID},
		map[string]interface{}{"$set": map[string]interface{}{"balance": int64(1500)}},
	)

	// Try to update in transaction
	fmt.Println("Transaction attempts to set balance to $2,000...")
	session.UpdateOne("accounts",
		map[string]interface{}{"_id": docID},
		map[string]interface{}{"$set": map[string]interface{}{"balance": int64(2000)}},
	)

	// Execute 2PC - should detect write conflict
	ctx := context.Background()
	err := coordinator.Execute(ctx)
	if err != nil {
		fmt.Printf("✗ Write conflict detected: %v\n", err)
		fmt.Println("  Transaction aborted to maintain consistency")
	}
	fmt.Println()

	// Verify final balance
	finalDoc, _ := coll.FindOne(map[string]interface{}{"_id": docID})
	finalBalance, _ := finalDoc.Get("balance")
	fmt.Printf("Final balance: $%d (from concurrent update, transaction rolled back)\n", finalBalance)
}

// MockParticipant for demo purposes
type MockParticipant struct {
	ID_             distributed.ParticipantID
	PrepareResponse bool
}

func (m *MockParticipant) ID() distributed.ParticipantID {
	return m.ID_
}

func (m *MockParticipant) Prepare(ctx context.Context, txnID mvcc.TxnID) (bool, error) {
	return m.PrepareResponse, nil
}

func (m *MockParticipant) Commit(ctx context.Context, txnID mvcc.TxnID) error {
	return nil
}

func (m *MockParticipant) Abort(ctx context.Context, txnID mvcc.TxnID) error {
	return nil
}
