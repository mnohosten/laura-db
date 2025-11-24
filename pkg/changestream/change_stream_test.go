package changestream

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mnohosten/laura-db/pkg/replication"
)

func setupTestOplog(t *testing.T) (*replication.Oplog, string) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "changestream-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	oplogPath := filepath.Join(tmpDir, "oplog.bin")
	oplog, err := replication.NewOplog(oplogPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create oplog: %v", err)
	}

	return oplog, tmpDir
}

func cleanupTestOplog(oplog *replication.Oplog, tmpDir string) {
	oplog.Close()
	os.RemoveAll(tmpDir)
}

func TestNewChangeStream(t *testing.T) {
	oplog, tmpDir := setupTestOplog(t)
	defer cleanupTestOplog(oplog, tmpDir)

	cs := NewChangeStream(oplog, "testdb", "testcoll", nil)
	if cs == nil {
		t.Fatal("Expected change stream to be created")
	}

	if cs.database != "testdb" {
		t.Errorf("Expected database 'testdb', got '%s'", cs.database)
	}

	if cs.collection != "testcoll" {
		t.Errorf("Expected collection 'testcoll', got '%s'", cs.collection)
	}

	if cs.options == nil {
		t.Fatal("Expected options to be initialized")
	}

	cs.Close()
}

func TestChangeStreamInsertEvent(t *testing.T) {
	oplog, tmpDir := setupTestOplog(t)
	defer cleanupTestOplog(oplog, tmpDir)

	// Create change stream
	cs := NewChangeStream(oplog, "testdb", "users", nil)
	if err := cs.Start(); err != nil {
		t.Fatalf("Failed to start change stream: %v", err)
	}
	defer cs.Close()

	// Insert a document to oplog
	doc := map[string]interface{}{
		"_id":  "user1",
		"name": "Alice",
		"age":  int64(30),
	}
	entry := replication.CreateInsertEntry("testdb", "users", doc)
	if err := oplog.Append(entry); err != nil {
		t.Fatalf("Failed to append to oplog: %v", err)
	}

	// Wait for event
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	event, err := cs.Next(ctx)
	if err != nil {
		t.Fatalf("Failed to receive event: %v", err)
	}

	// Verify event
	if event.OperationType != OperationTypeInsert {
		t.Errorf("Expected operation type 'insert', got '%s'", event.OperationType)
	}

	if event.Database != "testdb" {
		t.Errorf("Expected database 'testdb', got '%s'", event.Database)
	}

	if event.Collection != "users" {
		t.Errorf("Expected collection 'users', got '%s'", event.Collection)
	}

	if event.FullDocument == nil {
		t.Fatal("Expected full document to be present")
	}

	if event.FullDocument["name"] != "Alice" {
		t.Errorf("Expected name 'Alice', got '%v'", event.FullDocument["name"])
	}
}

func TestChangeStreamUpdateEvent(t *testing.T) {
	oplog, tmpDir := setupTestOplog(t)
	defer cleanupTestOplog(oplog, tmpDir)

	cs := NewChangeStream(oplog, "testdb", "users", nil)
	if err := cs.Start(); err != nil {
		t.Fatalf("Failed to start change stream: %v", err)
	}
	defer cs.Close()

	// Create update entry
	filter := map[string]interface{}{"_id": "user1"}
	update := map[string]interface{}{
		"$set": map[string]interface{}{
			"name": "Alice Updated",
			"age":  int64(31),
		},
	}
	entry := replication.CreateUpdateEntry("testdb", "users", filter, update)
	entry.DocID = "user1" // Manually set for testing

	if err := oplog.Append(entry); err != nil {
		t.Fatalf("Failed to append to oplog: %v", err)
	}

	// Wait for event
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	event, err := cs.Next(ctx)
	if err != nil {
		t.Fatalf("Failed to receive event: %v", err)
	}

	// Verify event
	if event.OperationType != OperationTypeUpdate {
		t.Errorf("Expected operation type 'update', got '%s'", event.OperationType)
	}

	if event.UpdateDescription == nil {
		t.Fatal("Expected update description to be present")
	}

	if event.UpdateDescription.UpdatedFields["name"] != "Alice Updated" {
		t.Errorf("Expected updated name 'Alice Updated', got '%v'", event.UpdateDescription.UpdatedFields["name"])
	}
}

func TestChangeStreamDeleteEvent(t *testing.T) {
	oplog, tmpDir := setupTestOplog(t)
	defer cleanupTestOplog(oplog, tmpDir)

	cs := NewChangeStream(oplog, "testdb", "users", nil)
	if err := cs.Start(); err != nil {
		t.Fatalf("Failed to start change stream: %v", err)
	}
	defer cs.Close()

	// Create delete entry
	filter := map[string]interface{}{"_id": "user1"}
	entry := replication.CreateDeleteEntry("testdb", "users", filter)
	entry.DocID = "user1" // Manually set for testing

	if err := oplog.Append(entry); err != nil {
		t.Fatalf("Failed to append to oplog: %v", err)
	}

	// Wait for event
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	event, err := cs.Next(ctx)
	if err != nil {
		t.Fatalf("Failed to receive event: %v", err)
	}

	// Verify event
	if event.OperationType != OperationTypeDelete {
		t.Errorf("Expected operation type 'delete', got '%s'", event.OperationType)
	}

	if event.DocumentKey["_id"] != "user1" {
		t.Errorf("Expected document key '_id' to be 'user1', got '%v'", event.DocumentKey["_id"])
	}
}

func TestChangeStreamFilter(t *testing.T) {
	oplog, tmpDir := setupTestOplog(t)
	defer cleanupTestOplog(oplog, tmpDir)

	cs := NewChangeStream(oplog, "testdb", "users", nil)

	// Set filter to only match insert operations
	filter := map[string]interface{}{
		"operationType": "insert",
	}
	if err := cs.SetFilter(filter); err != nil {
		t.Fatalf("Failed to set filter: %v", err)
	}

	if err := cs.Start(); err != nil {
		t.Fatalf("Failed to start change stream: %v", err)
	}
	defer cs.Close()

	// Insert a document (should match)
	doc := map[string]interface{}{"_id": "user1", "name": "Alice"}
	entry := replication.CreateInsertEntry("testdb", "users", doc)
	if err := oplog.Append(entry); err != nil {
		t.Fatalf("Failed to append insert: %v", err)
	}

	// Delete a document (should not match)
	deleteEntry := replication.CreateDeleteEntry("testdb", "users", map[string]interface{}{"_id": "user2"})
	deleteEntry.DocID = "user2"
	if err := oplog.Append(deleteEntry); err != nil {
		t.Fatalf("Failed to append delete: %v", err)
	}

	// Wait for event
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	event, err := cs.Next(ctx)
	if err != nil {
		t.Fatalf("Failed to receive event: %v", err)
	}

	// Should only receive insert event
	if event.OperationType != OperationTypeInsert {
		t.Errorf("Expected only insert events, got '%s'", event.OperationType)
	}

	// Should not receive delete event (timeout expected)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel2()

	_, err = cs.Next(ctx2)
	if err == nil {
		t.Error("Expected timeout, but received an event")
	}
}

func TestChangeStreamResumeToken(t *testing.T) {
	oplog, tmpDir := setupTestOplog(t)
	defer cleanupTestOplog(oplog, tmpDir)

	// Create first change stream
	cs1 := NewChangeStream(oplog, "testdb", "users", nil)
	if err := cs1.Start(); err != nil {
		t.Fatalf("Failed to start change stream: %v", err)
	}

	// Insert some documents AFTER starting the stream
	doc1 := map[string]interface{}{"_id": "user1", "name": "Alice"}
	entry1 := replication.CreateInsertEntry("testdb", "users", doc1)
	if err := oplog.Append(entry1); err != nil {
		t.Fatalf("Failed to append: %v", err)
	}

	doc2 := map[string]interface{}{"_id": "user2", "name": "Bob"}
	entry2 := replication.CreateInsertEntry("testdb", "users", doc2)
	if err := oplog.Append(entry2); err != nil {
		t.Fatalf("Failed to append: %v", err)
	}

	// Get first event
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	event1, err := cs1.Next(ctx)
	if err != nil {
		t.Fatalf("Failed to receive first event: %v", err)
	}

	// Save resume token after first event
	resumeToken := event1.ID
	cs1.Close()

	// Insert third document
	doc3 := map[string]interface{}{"_id": "user3", "name": "Charlie"}
	entry3 := replication.CreateInsertEntry("testdb", "users", doc3)
	if err := oplog.Append(entry3); err != nil {
		t.Fatalf("Failed to append: %v", err)
	}

	// Create second change stream with resume token
	options := DefaultChangeStreamOptions()
	options.ResumeAfter = &resumeToken

	cs2 := NewChangeStream(oplog, "testdb", "users", options)
	if err := cs2.Start(); err != nil {
		t.Fatalf("Failed to start second change stream: %v", err)
	}
	defer cs2.Close()

	// Should receive events after resume token (user2 and user3)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()

	event2, err := cs2.Next(ctx2)
	if err != nil {
		t.Fatalf("Failed to receive second event: %v", err)
	}

	if event2.FullDocument["_id"] != "user2" {
		t.Errorf("Expected user2, got '%v'", event2.FullDocument["_id"])
	}

	event3, err := cs2.Next(ctx2)
	if err != nil {
		t.Fatalf("Failed to receive third event: %v", err)
	}

	if event3.FullDocument["_id"] != "user3" {
		t.Errorf("Expected user3, got '%v'", event3.FullDocument["_id"])
	}
}

func TestChangeStreamDatabaseFilter(t *testing.T) {
	oplog, tmpDir := setupTestOplog(t)
	defer cleanupTestOplog(oplog, tmpDir)

	// Watch only "testdb"
	cs := NewChangeStream(oplog, "testdb", "", nil)
	if err := cs.Start(); err != nil {
		t.Fatalf("Failed to start change stream: %v", err)
	}
	defer cs.Close()

	// Insert into testdb (should match)
	doc1 := map[string]interface{}{"_id": "user1", "name": "Alice"}
	entry1 := replication.CreateInsertEntry("testdb", "users", doc1)
	if err := oplog.Append(entry1); err != nil {
		t.Fatalf("Failed to append: %v", err)
	}

	// Insert into otherdb (should not match)
	doc2 := map[string]interface{}{"_id": "user2", "name": "Bob"}
	entry2 := replication.CreateInsertEntry("otherdb", "users", doc2)
	if err := oplog.Append(entry2); err != nil {
		t.Fatalf("Failed to append: %v", err)
	}

	// Wait for event
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	event, err := cs.Next(ctx)
	if err != nil {
		t.Fatalf("Failed to receive event: %v", err)
	}

	// Should only receive testdb event
	if event.Database != "testdb" {
		t.Errorf("Expected database 'testdb', got '%s'", event.Database)
	}

	if event.FullDocument["_id"] != "user1" {
		t.Errorf("Expected user1, got '%v'", event.FullDocument["_id"])
	}
}

func TestChangeStreamCollectionFilter(t *testing.T) {
	oplog, tmpDir := setupTestOplog(t)
	defer cleanupTestOplog(oplog, tmpDir)

	// Watch only "users" collection
	cs := NewChangeStream(oplog, "testdb", "users", nil)
	if err := cs.Start(); err != nil {
		t.Fatalf("Failed to start change stream: %v", err)
	}
	defer cs.Close()

	// Insert into users (should match)
	doc1 := map[string]interface{}{"_id": "user1", "name": "Alice"}
	entry1 := replication.CreateInsertEntry("testdb", "users", doc1)
	if err := oplog.Append(entry1); err != nil {
		t.Fatalf("Failed to append: %v", err)
	}

	// Insert into products (should not match)
	doc2 := map[string]interface{}{"_id": "prod1", "name": "Widget"}
	entry2 := replication.CreateInsertEntry("testdb", "products", doc2)
	if err := oplog.Append(entry2); err != nil {
		t.Fatalf("Failed to append: %v", err)
	}

	// Wait for event
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	event, err := cs.Next(ctx)
	if err != nil {
		t.Fatalf("Failed to receive event: %v", err)
	}

	// Should only receive users collection event
	if event.Collection != "users" {
		t.Errorf("Expected collection 'users', got '%s'", event.Collection)
	}

	if event.FullDocument["_id"] != "user1" {
		t.Errorf("Expected user1, got '%v'", event.FullDocument["_id"])
	}
}

func TestChangeStreamMultipleEvents(t *testing.T) {
	oplog, tmpDir := setupTestOplog(t)
	defer cleanupTestOplog(oplog, tmpDir)

	cs := NewChangeStream(oplog, "testdb", "users", nil)
	if err := cs.Start(); err != nil {
		t.Fatalf("Failed to start change stream: %v", err)
	}
	defer cs.Close()

	// Insert multiple documents
	numDocs := 10
	for i := 0; i < numDocs; i++ {
		doc := map[string]interface{}{
			"_id":  i,
			"name": "User",
			"num":  int64(i),
		}
		entry := replication.CreateInsertEntry("testdb", "users", doc)
		if err := oplog.Append(entry); err != nil {
			t.Fatalf("Failed to append: %v", err)
		}
	}

	// Receive all events
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := 0; i < numDocs; i++ {
		event, err := cs.Next(ctx)
		if err != nil {
			t.Fatalf("Failed to receive event %d: %v", i, err)
		}

		if event.OperationType != OperationTypeInsert {
			t.Errorf("Event %d: expected insert, got '%s'", i, event.OperationType)
		}
	}
}

func TestChangeStreamTryNext(t *testing.T) {
	oplog, tmpDir := setupTestOplog(t)
	defer cleanupTestOplog(oplog, tmpDir)

	cs := NewChangeStream(oplog, "testdb", "users", nil)
	if err := cs.Start(); err != nil {
		t.Fatalf("Failed to start change stream: %v", err)
	}
	defer cs.Close()

	// Try to get event when none available
	event, err := cs.TryNext()
	if err != nil {
		t.Fatalf("TryNext should not return error: %v", err)
	}

	if event != nil {
		t.Error("Expected no event, but got one")
	}

	// Insert a document
	doc := map[string]interface{}{"_id": "user1", "name": "Alice"}
	entry := replication.CreateInsertEntry("testdb", "users", doc)
	if err := oplog.Append(entry); err != nil {
		t.Fatalf("Failed to append: %v", err)
	}

	// Wait a bit for event to be processed
	time.Sleep(1500 * time.Millisecond)

	// Try to get event
	event, err = cs.TryNext()
	if err != nil {
		t.Fatalf("TryNext failed: %v", err)
	}

	if event == nil {
		t.Error("Expected to receive event")
	}
}

func TestChangeStreamClose(t *testing.T) {
	oplog, tmpDir := setupTestOplog(t)
	defer cleanupTestOplog(oplog, tmpDir)

	cs := NewChangeStream(oplog, "testdb", "users", nil)
	if err := cs.Start(); err != nil {
		t.Fatalf("Failed to start change stream: %v", err)
	}

	// Close the stream
	if err := cs.Close(); err != nil {
		t.Fatalf("Failed to close change stream: %v", err)
	}

	// Verify channels are closed
	select {
	case _, ok := <-cs.Events():
		if ok {
			t.Error("Events channel should be closed")
		}
	default:
	}

	// Double close should not error
	if err := cs.Close(); err != nil {
		t.Errorf("Double close should not error: %v", err)
	}
}

func TestChangeStreamUnsetOperator(t *testing.T) {
	oplog, tmpDir := setupTestOplog(t)
	defer cleanupTestOplog(oplog, tmpDir)

	cs := NewChangeStream(oplog, "testdb", "users", nil)
	if err := cs.Start(); err != nil {
		t.Fatalf("Failed to start change stream: %v", err)
	}
	defer cs.Close()

	// Create update with $unset
	filter := map[string]interface{}{"_id": "user1"}
	update := map[string]interface{}{
		"$unset": map[string]interface{}{
			"email": "",
			"phone": "",
		},
	}
	entry := replication.CreateUpdateEntry("testdb", "users", filter, update)
	entry.DocID = "user1"

	if err := oplog.Append(entry); err != nil {
		t.Fatalf("Failed to append: %v", err)
	}

	// Wait for event
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	event, err := cs.Next(ctx)
	if err != nil {
		t.Fatalf("Failed to receive event: %v", err)
	}

	// Verify removed fields
	if len(event.UpdateDescription.RemovedFields) != 2 {
		t.Errorf("Expected 2 removed fields, got %d", len(event.UpdateDescription.RemovedFields))
	}
}

func TestChangeStreamPipeline(t *testing.T) {
	oplog, tmpDir := setupTestOplog(t)
	defer cleanupTestOplog(oplog, tmpDir)

	// Create change stream with pipeline
	options := DefaultChangeStreamOptions()
	options.Pipeline = []map[string]interface{}{
		{
			"$match": map[string]interface{}{
				"operationType": "insert",
			},
		},
	}

	cs := NewChangeStream(oplog, "testdb", "users", options)
	if err := cs.Start(); err != nil {
		t.Fatalf("Failed to start change stream: %v", err)
	}
	defer cs.Close()

	// Insert and update
	doc := map[string]interface{}{"_id": "user1", "name": "Alice"}
	insertEntry := replication.CreateInsertEntry("testdb", "users", doc)
	if err := oplog.Append(insertEntry); err != nil {
		t.Fatalf("Failed to append insert: %v", err)
	}

	updateEntry := replication.CreateUpdateEntry("testdb", "users",
		map[string]interface{}{"_id": "user1"},
		map[string]interface{}{"$set": map[string]interface{}{"name": "Alice Updated"}})
	updateEntry.DocID = "user1"
	if err := oplog.Append(updateEntry); err != nil {
		t.Fatalf("Failed to append update: %v", err)
	}

	// Wait for event
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	event, err := cs.Next(ctx)
	if err != nil {
		t.Fatalf("Failed to receive event: %v", err)
	}

	// Should only receive insert (filtered by pipeline)
	if event.OperationType != OperationTypeInsert {
		t.Errorf("Expected only insert events, got '%s'", event.OperationType)
	}
}

func TestChangeStreamIndexOperations(t *testing.T) {
	oplog, tmpDir := setupTestOplog(t)
	defer cleanupTestOplog(oplog, tmpDir)

	cs := NewChangeStream(oplog, "testdb", "users", nil)
	if err := cs.Start(); err != nil {
		t.Fatalf("Failed to start change stream: %v", err)
	}
	defer cs.Close()

	// Create index
	indexDef := map[string]interface{}{
		"name":  "age_idx",
		"field": "age",
	}
	createEntry := replication.CreateIndexEntry("testdb", "users", indexDef, true)
	if err := oplog.Append(createEntry); err != nil {
		t.Fatalf("Failed to append create index: %v", err)
	}

	// Drop index
	dropEntry := replication.CreateIndexEntry("testdb", "users", indexDef, false)
	if err := oplog.Append(dropEntry); err != nil {
		t.Fatalf("Failed to append drop index: %v", err)
	}

	// Wait for events
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	event1, err := cs.Next(ctx)
	if err != nil {
		t.Fatalf("Failed to receive create index event: %v", err)
	}

	if event1.OperationType != OperationTypeCreateIndex {
		t.Errorf("Expected createIndex, got '%s'", event1.OperationType)
	}

	event2, err := cs.Next(ctx)
	if err != nil {
		t.Fatalf("Failed to receive drop index event: %v", err)
	}

	if event2.OperationType != OperationTypeDropIndex {
		t.Errorf("Expected dropIndex, got '%s'", event2.OperationType)
	}
}
