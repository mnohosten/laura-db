package storage

import (
	"os"
	"testing"
)

func TestNewStorageEngine(t *testing.T) {
	dir := "./test_storage"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}
	defer engine.Close()

	if engine == nil {
		t.Fatal("Expected non-nil storage engine")
	}
}

func TestAllocateAndFetchPage(t *testing.T) {
	dir := "./test_storage_page"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}
	defer engine.Close()

	// Allocate page
	page, err := engine.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}

	pageID := page.ID

	// Write data to page
	testData := []byte("Hello, Storage!")
	copy(page.Data, testData)
	page.MarkDirty()

	// Unpin page
	engine.UnpinPage(pageID, true)

	// Fetch page back
	fetchedPage, err := engine.FetchPage(pageID)
	if err != nil {
		t.Fatalf("Failed to fetch page: %v", err)
	}

	// Verify data
	fetchedData := fetchedPage.Data[:len(testData)]
	if string(fetchedData) != string(testData) {
		t.Errorf("Expected %s, got %s", testData, fetchedData)
	}

	engine.UnpinPage(fetchedPage.ID, false)
}

func TestWALLogOperation(t *testing.T) {
	dir := "./test_storage_wal"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}
	defer engine.Close()

	// Log an operation
	record := &LogRecord{
		Type:   LogRecordInsert,
		TxnID:  1,
		PageID: 0,
		Data:   []byte("test data"),
	}

	lsn, err := engine.LogOperation(record)
	if err != nil {
		t.Fatalf("Failed to log operation: %v", err)
	}

	if lsn == 0 {
		t.Error("Expected non-zero LSN")
	}
}

func TestCheckpoint(t *testing.T) {
	dir := "./test_storage_checkpoint"
	defer os.RemoveAll(dir)

	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}
	defer engine.Close()

	// Allocate and modify a page
	page, _ := engine.AllocatePage()
	copy(page.Data, []byte("checkpoint test"))
	page.MarkDirty()
	engine.UnpinPage(page.ID, true)

	// Checkpoint
	err = engine.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint failed: %v", err)
	}
}

func TestStorageEngineRecovery(t *testing.T) {
	dir := "./test_storage_recovery"
	defer os.RemoveAll(dir)

	// Create engine and write data
	config := DefaultConfig(dir)
	engine, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}

	page, _ := engine.AllocatePage()
	testData := []byte("recovery test")
	copy(page.Data, testData)
	page.MarkDirty()

	// Log operation
	record := &LogRecord{
		Type:   LogRecordInsert,
		TxnID:  1,
		PageID: page.ID,
		Data:   []byte("metadata"),
	}
	lsn, _ := engine.LogOperation(record)
	page.LSN = lsn

	engine.UnpinPage(page.ID, true)
	pageID := page.ID

	// Close engine
	engine.Close()

	// Reopen engine (should trigger recovery)
	engine2, err := NewStorageEngine(config)
	if err != nil {
		t.Fatalf("Failed to reopen storage engine: %v", err)
	}
	defer engine2.Close()

	// Fetch page and verify data persisted
	recoveredPage, err := engine2.FetchPage(pageID)
	if err != nil {
		t.Fatalf("Failed to fetch page after recovery: %v", err)
	}

	recoveredData := recoveredPage.Data[:len(testData)]
	if string(recoveredData) != string(testData) {
		t.Errorf("Data not recovered correctly: expected %s, got %s", testData, recoveredData)
	}

	engine2.UnpinPage(recoveredPage.ID, false)
}
