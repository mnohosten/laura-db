package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mnohosten/laura-db/pkg/changestream"
	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/replication"
)

func main() {
	// Create a temporary directory for the database
	tmpDir, err := os.MkdirTemp("", "changestream-demo-*")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fmt.Println("=== LauraDB Change Streams Demo ===")
	fmt.Println()

	// Open database
	config := &database.Config{
		DataDir:        tmpDir,
		BufferPoolSize: 1000,
	}
	db, err := database.Open(config)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create oplog for change streams
	oplogPath := filepath.Join(tmpDir, "oplog.bin")
	oplog, err := replication.NewOplog(oplogPath)
	if err != nil {
		log.Fatalf("Failed to create oplog: %v", err)
	}
	defer oplog.Close()

	// Create a collection
	coll, err := db.CreateCollection("users")
	if err != nil {
		log.Fatalf("Failed to create collection: %v", err)
	}

	// Demo 1: Basic Change Stream
	fmt.Println("Demo 1: Basic Change Stream - Watch all changes")
	fmt.Println("------------------------------------------------")
	demo1BasicChangeStream(coll, oplog)

	// Demo 2: Filtered Change Stream
	fmt.Println("\nDemo 2: Filtered Change Stream - Watch only insert operations")
	fmt.Println("-------------------------------------------------------------")
	demo2FilteredChangeStream(coll, oplog)

	// Demo 3: Resume Tokens
	fmt.Println("\nDemo 3: Resume Tokens - Resume from a specific point")
	fmt.Println("---------------------------------------------------")
	demo3ResumeTokens(coll, oplog)

	// Demo 4: Collection-specific Change Stream
	fmt.Println("\nDemo 4: Collection-specific Change Stream")
	fmt.Println("----------------------------------------")
	demo4CollectionSpecific(db, oplog)

	// Demo 5: Pipeline with $match
	fmt.Println("\nDemo 5: Pipeline with $match stage")
	fmt.Println("---------------------------------")
	demo5Pipeline(coll, oplog)

	fmt.Println("\n=== Demo Complete ===")
}

func demo1BasicChangeStream(coll *database.Collection, oplog *replication.Oplog) {
	// Create change stream
	cs := changestream.NewChangeStream(oplog, "", "users", nil)
	if err := cs.Start(); err != nil {
		log.Fatalf("Failed to start change stream: %v", err)
	}
	defer cs.Close()

	// Start a goroutine to watch for changes
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		for {
			event, err := cs.Next(ctx)
			if err != nil {
				return // Timeout or closed
			}

			fmt.Printf("  [Event] Operation: %s, Collection: %s\n", event.OperationType, event.Collection)
			if event.FullDocument != nil {
				fmt.Printf("          Document: %v\n", event.FullDocument)
			}
		}
	}()

	// Insert some documents
	fmt.Println("Inserting documents...")
	coll.InsertOne(map[string]interface{}{"_id": "user1", "name": "Alice", "age": int64(30)})

	// Log to oplog
	oplog.Append(replication.CreateInsertEntry("testdb", "users",
		map[string]interface{}{"_id": "user1", "name": "Alice", "age": int64(30)}))

	coll.InsertOne(map[string]interface{}{"_id": "user2", "name": "Bob", "age": int64(25)})
	oplog.Append(replication.CreateInsertEntry("testdb", "users",
		map[string]interface{}{"_id": "user2", "name": "Bob", "age": int64(25)}))

	// Wait for events to be processed
	time.Sleep(1500 * time.Millisecond)
}

func demo2FilteredChangeStream(coll *database.Collection, oplog *replication.Oplog) {
	// Create change stream with filter
	cs := changestream.NewChangeStream(oplog, "", "users", nil)

	// Set filter to only match insert operations
	filter := map[string]interface{}{
		"operationType": "insert",
	}
	if err := cs.SetFilter(filter); err != nil {
		log.Fatalf("Failed to set filter: %v", err)
	}

	if err := cs.Start(); err != nil {
		log.Fatalf("Failed to start change stream: %v", err)
	}
	defer cs.Close()

	// Watch for changes
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		for {
			event, err := cs.Next(ctx)
			if err != nil {
				return
			}

			fmt.Printf("  [Insert Event] Document: %v\n", event.FullDocument)
		}
	}()

	// Insert and update
	fmt.Println("Inserting user3...")
	coll.InsertOne(map[string]interface{}{"_id": "user3", "name": "Charlie", "age": int64(35)})
	oplog.Append(replication.CreateInsertEntry("testdb", "users",
		map[string]interface{}{"_id": "user3", "name": "Charlie", "age": int64(35)}))

	fmt.Println("Updating user3 (should be filtered out)...")
	coll.UpdateOne(map[string]interface{}{"_id": "user3"},
		map[string]interface{}{"$set": map[string]interface{}{"age": int64(36)}})

	updateEntry := replication.CreateUpdateEntry("testdb", "users",
		map[string]interface{}{"_id": "user3"},
		map[string]interface{}{"$set": map[string]interface{}{"age": int64(36)}})
	updateEntry.DocID = "user3"
	oplog.Append(updateEntry)

	time.Sleep(1500 * time.Millisecond)
}

func demo3ResumeTokens(coll *database.Collection, oplog *replication.Oplog) {
	// Create first change stream
	cs1 := changestream.NewChangeStream(oplog, "", "users", nil)
	if err := cs1.Start(); err != nil {
		log.Fatalf("Failed to start change stream: %v", err)
	}

	// Insert documents
	fmt.Println("Inserting user4 and user5...")
	coll.InsertOne(map[string]interface{}{"_id": "user4", "name": "David"})
	oplog.Append(replication.CreateInsertEntry("testdb", "users",
		map[string]interface{}{"_id": "user4", "name": "David"}))

	coll.InsertOne(map[string]interface{}{"_id": "user5", "name": "Eve"})
	oplog.Append(replication.CreateInsertEntry("testdb", "users",
		map[string]interface{}{"_id": "user5", "name": "Eve"}))

	// Get first event and save resume token
	ctx1, cancel1 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel1()

	event1, err := cs1.Next(ctx1)
	if err != nil {
		log.Printf("Failed to get first event: %v", err)
		return
	}

	resumeToken := event1.ID
	fmt.Printf("  Received event 1: %v (Resume Token: %d)\n", event1.FullDocument["_id"], resumeToken.OpID)
	cs1.Close()

	// Insert another document
	fmt.Println("Inserting user6...")
	coll.InsertOne(map[string]interface{}{"_id": "user6", "name": "Frank"})
	oplog.Append(replication.CreateInsertEntry("testdb", "users",
		map[string]interface{}{"_id": "user6", "name": "Frank"}))

	// Create second change stream with resume token
	fmt.Printf("Resuming from token %d...\n", resumeToken.OpID)
	options := changestream.DefaultChangeStreamOptions()
	options.ResumeAfter = &resumeToken

	cs2 := changestream.NewChangeStream(oplog, "", "users", options)
	if err := cs2.Start(); err != nil {
		log.Fatalf("Failed to start resumed change stream: %v", err)
	}
	defer cs2.Close()

	// Get remaining events
	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()

	count := 0
	for {
		event, err := cs2.Next(ctx2)
		if err != nil {
			break
		}
		count++
		fmt.Printf("  Resumed event %d: %v\n", count, event.FullDocument["_id"])
		if count >= 2 {
			break
		}
	}
}

func demo4CollectionSpecific(db *database.Database, oplog *replication.Oplog) {
	// Create two collections
	usersColl, err := db.CreateCollection("users2")
	if err != nil {
		log.Fatalf("Failed to create users2 collection: %v", err)
	}
	productsColl, err := db.CreateCollection("products")
	if err != nil {
		log.Fatalf("Failed to create products collection: %v", err)
	}

	// Watch only users collection
	cs := changestream.NewChangeStream(oplog, "", "users", nil)
	if err := cs.Start(); err != nil {
		log.Fatalf("Failed to start change stream: %v", err)
	}
	defer cs.Close()

	// Watch for changes
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		for {
			event, err := cs.Next(ctx)
			if err != nil {
				return
			}

			fmt.Printf("  [Users Event] %v\n", event.FullDocument["_id"])
		}
	}()

	// Insert into both collections
	fmt.Println("Inserting into users collection...")
	usersColl.InsertOne(map[string]interface{}{"_id": "user7", "name": "Grace"})
	oplog.Append(replication.CreateInsertEntry("testdb", "users",
		map[string]interface{}{"_id": "user7", "name": "Grace"}))

	fmt.Println("Inserting into products collection (should be filtered out)...")
	productsColl.InsertOne(map[string]interface{}{"_id": "prod1", "name": "Widget"})
	oplog.Append(replication.CreateInsertEntry("testdb", "products",
		map[string]interface{}{"_id": "prod1", "name": "Widget"}))

	time.Sleep(1500 * time.Millisecond)
}

func demo5Pipeline(coll *database.Collection, oplog *replication.Oplog) {
	// Create change stream with pipeline
	options := changestream.DefaultChangeStreamOptions()
	options.Pipeline = []map[string]interface{}{
		{
			"$match": map[string]interface{}{
				"operationType": "insert",
			},
		},
	}

	cs := changestream.NewChangeStream(oplog, "", "users", options)
	if err := cs.Start(); err != nil {
		log.Fatalf("Failed to start change stream: %v", err)
	}
	defer cs.Close()

	// Watch for changes
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		for {
			event, err := cs.Next(ctx)
			if err != nil {
				return
			}

			fmt.Printf("  [Pipeline Matched] Operation: %s, Document: %v\n",
				event.OperationType, event.FullDocument["_id"])
		}
	}()

	// Insert and delete
	fmt.Println("Inserting user8 (should match pipeline)...")
	coll.InsertOne(map[string]interface{}{"_id": "user8", "name": "Henry"})
	oplog.Append(replication.CreateInsertEntry("testdb", "users",
		map[string]interface{}{"_id": "user8", "name": "Henry"}))

	fmt.Println("Deleting user8 (should be filtered by pipeline)...")
	coll.DeleteOne(map[string]interface{}{"_id": "user8"})

	deleteEntry := replication.CreateDeleteEntry("testdb", "users",
		map[string]interface{}{"_id": "user8"})
	deleteEntry.DocID = "user8"
	oplog.Append(deleteEntry)

	time.Sleep(1500 * time.Millisecond)
}
