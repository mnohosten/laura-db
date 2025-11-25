package encryption

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mnohosten/laura-db/pkg/storage"
)

// TestEncryptedDiskManagerIntegration tests the encrypted disk manager with real disk I/O
func TestEncryptedDiskManagerIntegration(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), "test-encrypted-disk")
	defer os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0755)

	dataPath := filepath.Join(dataDir, "test.db")

	// Create encryption config
	config, err := NewConfigFromPassword("test-password-123", AlgorithmAES256GCM)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Create encrypted disk manager
	edm, err := NewEncryptedDiskManager(dataPath, config)
	if err != nil {
		t.Fatalf("Failed to create encrypted disk manager: %v", err)
	}
	defer edm.Close()

	// Allocate a page
	pageID, err := edm.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}

	// Create a page with test data
	// Note: When using encryption, the encrypted data will be larger due to nonce/tag overhead.
	// In a production system, you'd need to account for this in your page size calculations.
	// For this test, we create a fresh page which has empty data by default.
	page := storage.NewPage(pageID, storage.PageTypeData)
	testData := []byte("This is secret encrypted data that should be protected!")

	// Only use a portion of the page to leave room for encryption overhead
	maxDataSize := len(page.Data) - EncryptionOverhead - EncryptedPageHeaderSize
	if len(testData) < maxDataSize {
		copy(page.Data[:len(testData)], testData)
		// Zero out the rest to avoid encrypting garbage
		for i := len(testData); i < maxDataSize; i++ {
			page.Data[i] = 0
		}
		// Trim page data to only what we're actually using
		page.Data = page.Data[:maxDataSize]
	}

	// Write encrypted page
	if err := edm.WritePage(page); err != nil {
		t.Fatalf("Failed to write page: %v", err)
	}

	// Sync to disk
	if err := edm.Sync(); err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}

	// Close and reopen
	edm.Close()

	// Verify data is encrypted on disk by reading raw file
	rawData, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatalf("Failed to read raw file: %v", err)
	}

	// The raw data should NOT contain our plaintext (it's encrypted)
	if containsBytes(rawData, testData) {
		t.Error("Raw file contains plaintext data - encryption failed!")
	}

	// Reopen with same key
	edm, err = NewEncryptedDiskManager(dataPath, config)
	if err != nil {
		t.Fatalf("Failed to reopen encrypted disk manager: %v", err)
	}
	defer edm.Close()

	// Read and decrypt page
	readPage, err := edm.ReadPage(pageID)
	if err != nil {
		t.Fatalf("Failed to read page: %v", err)
	}

	// Verify decrypted data matches original
	if string(readPage.Data[:len(testData)]) != string(testData) {
		t.Error("Decrypted data does not match original")
	}
}

// TestEncryptedWALIntegration tests the encrypted WAL with real disk I/O
func TestEncryptedWALIntegration(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), "test-encrypted-wal")
	defer os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0755)

	walPath := filepath.Join(dataDir, "test.wal")

	// Create encryption config
	config, err := NewConfigFromPassword("wal-password-456", AlgorithmAES256GCM)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Create encrypted WAL
	wal, err := NewEncryptedWAL(walPath, config)
	if err != nil {
		t.Fatalf("Failed to create encrypted WAL: %v", err)
	}

	// Write some encrypted log records
	testRecords := []*storage.LogRecord{
		{
			Type:   storage.LogRecordInsert,
			TxnID:  1,
			PageID: 0,
			Data:   []byte("Secret log entry 1"),
		},
		{
			Type:   storage.LogRecordUpdate,
			TxnID:  1,
			PageID:  1,
			Data:   []byte("Secret log entry 2"),
		},
		{
			Type:   storage.LogRecordCommit,
			TxnID:  1,
			PageID: 0,
			Data:   []byte{},
		},
	}

	for _, record := range testRecords {
		if _, err := wal.Append(record); err != nil {
			t.Fatalf("Failed to append record: %v", err)
		}
	}

	// Flush to disk
	if err := wal.Flush(); err != nil {
		t.Fatalf("Failed to flush WAL: %v", err)
	}

	// Close WAL
	wal.Close()

	// Verify data is encrypted on disk
	rawData, err := os.ReadFile(walPath)
	if err != nil {
		t.Fatalf("Failed to read raw WAL file: %v", err)
	}

	// The raw data should NOT contain our plaintext
	if containsBytes(rawData, []byte("Secret log entry 1")) {
		t.Error("Raw WAL file contains plaintext data - encryption failed!")
	}

	// Reopen and replay
	wal, err = NewEncryptedWAL(walPath, config)
	if err != nil {
		t.Fatalf("Failed to reopen encrypted WAL: %v", err)
	}
	defer wal.Close()

	// Replay encrypted records
	records, err := wal.Replay()
	if err != nil {
		t.Fatalf("Failed to replay WAL: %v", err)
	}

	// Verify record count
	if len(records) != len(testRecords) {
		t.Errorf("Expected %d records, got %d", len(testRecords), len(records))
	}

	// Verify decrypted data matches
	for i, record := range records {
		if string(record.Data) != string(testRecords[i].Data) {
			t.Errorf("Record %d data mismatch", i)
		}
	}
}

// Test opening encrypted data with wrong key
func TestWrongKeyFailure(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), "test-wrong-key")
	defer os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0755)

	dataPath := filepath.Join(dataDir, "test.db")

	// Create and write with first key
	config1, _ := NewConfigFromPassword("correct-password", AlgorithmAES256GCM)
	edm, _ := NewEncryptedDiskManager(dataPath, config1)

	pageID, _ := edm.AllocatePage()
	page := storage.NewPage(pageID, storage.PageTypeData)
	copy(page.Data, []byte("Secret data"))
	edm.WritePage(page)
	edm.Sync()
	edm.Close()

	// Try to open with wrong key
	config2, _ := NewConfigFromPassword("wrong-password", AlgorithmAES256GCM)
	edm, _ = NewEncryptedDiskManager(dataPath, config2)
	defer edm.Close()

	// Reading should fail with authentication error
	_, err := edm.ReadPage(pageID)
	if err == nil {
		t.Error("Expected error when reading with wrong key")
	}
}

// Helper function to check if haystack contains needle
func containsBytes(haystack, needle []byte) bool {
	if len(needle) == 0 {
		return true
	}
	if len(haystack) < len(needle) {
		return false
	}

	for i := 0; i <= len(haystack)-len(needle); i++ {
		found := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				found = false
				break
			}
		}
		if found {
			return true
		}
	}
	return false
}
