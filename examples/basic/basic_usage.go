package main

import (
	"fmt"
	"log"

	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/mvcc"
	"github.com/mnohosten/laura-db/pkg/storage"
)

func main() {
	// Example 1: Document Operations
	fmt.Println("=== Document Operations ===")
	documentExample()

	// Example 2: BSON Encoding/Decoding
	fmt.Println("\n=== BSON Encoding ===")
	bsonExample()

	// Example 3: Storage Engine
	fmt.Println("\n=== Storage Engine ===")
	storageExample()

	// Example 4: MVCC Transactions
	fmt.Println("\n=== MVCC Transactions ===")
	mvccExample()
}

func documentExample() {
	// Create a document
	doc := document.NewDocument()
	doc.Set("name", "Alice")
	doc.Set("age", int64(30))
	doc.Set("email", "alice@example.com")
	doc.Set("tags", []interface{}{"admin", "user"})

	// Create nested document
	address := map[string]interface{}{
		"city":    "New York",
		"zip":     "10001",
		"country": "USA",
	}
	doc.Set("address", address)

	// Read fields
	if name, ok := doc.Get("name"); ok {
		fmt.Printf("Name: %v\n", name)
	}

	// Convert to map
	fmt.Printf("Document: %v\n", doc.ToMap())

	// Clone
	cloned := doc.Clone()
	fmt.Printf("Cloned: %v\n", cloned.ToMap())
}

func bsonExample() {
	// Create document
	doc := document.NewDocument()
	doc.Set("_id", document.NewObjectID())
	doc.Set("title", "Hello World")
	doc.Set("count", int64(42))
	doc.Set("active", true)

	// Encode to BSON
	encoder := document.NewEncoder()
	data, err := encoder.Encode(doc)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Encoded size: %d bytes\n", len(data))

	// Decode from BSON
	decoder := document.NewDecoder(data)
	decoded, err := decoder.Decode()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Decoded: %v\n", decoded.ToMap())
}

func storageExample() {
	// Create storage engine
	config := storage.DefaultConfig("./example_data")
	engine, err := storage.NewStorageEngine(config)
	if err != nil {
		log.Fatal(err)
	}
	defer engine.Close()

	// Allocate a page
	page, err := engine.AllocatePage()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Allocated page ID: %d\n", page.ID)

	// Write data to page
	testData := []byte("Hello, Database!")
	copy(page.Data, testData)
	page.MarkDirty()

	// Log operation
	logRecord := &storage.LogRecord{
		Type:   storage.LogRecordInsert,
		TxnID:  1,
		PageID: page.ID,
		Data:   []byte("metadata"),
	}
	lsn, err := engine.LogOperation(logRecord)
	if err != nil {
		log.Fatal(err)
	}
	page.LSN = lsn
	fmt.Printf("Logged with LSN: %d\n", lsn)

	// Unpin page
	engine.UnpinPage(page.ID, true)

	// Fetch page back
	fetchedPage, err := engine.FetchPage(page.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Fetched data: %s\n", string(fetchedPage.Data[:len(testData)]))
	engine.UnpinPage(fetchedPage.ID, false)

	// Checkpoint
	if err := engine.Checkpoint(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Checkpoint completed")

	// View stats
	stats := engine.Stats()
	fmt.Printf("Storage stats: %+v\n", stats)
}

func mvccExample() {
	// Create transaction manager
	txnMgr := mvcc.NewTransactionManager()

	// Transaction 1: Write some data
	fmt.Println("\n--- Transaction 1: Write data ---")
	t1 := txnMgr.Begin()
	fmt.Printf("T1 started with ID: %d\n", t1.ID)

	txnMgr.Write(t1, "user:alice", map[string]interface{}{
		"name":  "Alice",
		"age":   30,
		"email": "alice@example.com",
	})
	txnMgr.Write(t1, "user:bob", map[string]interface{}{
		"name":  "Bob",
		"age":   25,
		"email": "bob@example.com",
	})

	if err := txnMgr.Commit(t1); err != nil {
		log.Fatal(err)
	}
	fmt.Println("T1 committed")

	// Transaction 2: Read data
	fmt.Println("\n--- Transaction 2: Read data ---")
	t2 := txnMgr.Begin()
	fmt.Printf("T2 started with ID: %d\n", t2.ID)

	alice, exists, _ := txnMgr.Read(t2, "user:alice")
	if exists {
		fmt.Printf("Read Alice: %v\n", alice)
	}

	// Transaction 3: Update Alice (concurrent with T2)
	fmt.Println("\n--- Transaction 3: Update Alice ---")
	t3 := txnMgr.Begin()
	aliceData, _, _ := txnMgr.Read(t3, "user:alice")
	aliceMap := aliceData.(map[string]interface{})
	aliceMap["age"] = 31 // Birthday!
	txnMgr.Write(t3, "user:alice", aliceMap)
	txnMgr.Commit(t3)
	fmt.Println("T3 committed (Alice age updated to 31)")

	// T2 reads Alice again - should still see age 30 (snapshot isolation)
	alice2, _, _ := txnMgr.Read(t2, "user:alice")
	fmt.Printf("T2 reads Alice again: %v\n", alice2)
	fmt.Println("(Notice: T2 still sees age=30 due to snapshot isolation)")

	txnMgr.Commit(t2)

	// Transaction 4: Read latest data
	fmt.Println("\n--- Transaction 4: Read latest ---")
	t4 := txnMgr.Begin()
	alice3, _, _ := txnMgr.Read(t4, "user:alice")
	fmt.Printf("T4 reads Alice: %v\n", alice3)
	fmt.Println("(Notice: T4 sees age=31)")
	txnMgr.Commit(t4)

	fmt.Printf("\nActive transactions: %d\n", txnMgr.GetActiveTransactions())
}
