package encryption

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mnohosten/laura-db/pkg/storage"
)

// TestEncryptedWAL_Checkpoint tests the Checkpoint function
func TestEncryptedWAL_Checkpoint(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), "test-wal-checkpoint")
	defer os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0755)

	walPath := filepath.Join(dataDir, "test.wal")

	tests := []struct {
		name      string
		algorithm Algorithm
	}{
		{"Checkpoint with GCM encryption", AlgorithmAES256GCM},
		{"Checkpoint with CTR encryption", AlgorithmAES256CTR},
		{"Checkpoint with no encryption", AlgorithmNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create encryption config
			config, err := NewConfigFromPassword("test-password", tt.algorithm)
			if err != nil {
				t.Fatalf("Failed to create config: %v", err)
			}

			// Create encrypted WAL
			wal, err := NewEncryptedWAL(walPath, config)
			if err != nil {
				t.Fatalf("Failed to create encrypted WAL: %v", err)
			}
			defer wal.Close()

			// Write some records
			record := &storage.LogRecord{
				Type:   storage.LogRecordInsert,
				TxnID:  1,
				PageID: 0,
				Data:   []byte("test data"),
			}

			_, err = wal.Append(record)
			if err != nil {
				t.Fatalf("Failed to append record: %v", err)
			}

			// Flush to ensure data is written
			err = wal.Flush()
			if err != nil {
				t.Fatalf("Failed to flush WAL: %v", err)
			}

			// Create checkpoint
			err = wal.Checkpoint()
			if err != nil {
				t.Errorf("Checkpoint() error = %v, expected nil", err)
			}

			// Clean up for next test
			os.RemoveAll(dataDir)
			os.MkdirAll(dataDir, 0755)
		})
	}
}

// TestEncryptedWAL_GetEncryptor tests the GetEncryptor function
func TestEncryptedWAL_GetEncryptor(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), "test-wal-get-encryptor")
	defer os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0755)

	walPath := filepath.Join(dataDir, "test.wal")

	// Create encryption config
	config, err := NewConfigFromPassword("test-password", AlgorithmAES256GCM)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Create encrypted WAL
	wal, err := NewEncryptedWAL(walPath, config)
	if err != nil {
		t.Fatalf("Failed to create encrypted WAL: %v", err)
	}
	defer wal.Close()

	// Get encryptor
	encryptor := wal.GetEncryptor()

	// Verify encryptor is not nil
	if encryptor == nil {
		t.Fatal("GetEncryptor() returned nil")
	}

	// Verify encryptor config matches
	retrievedConfig := encryptor.GetConfig()
	if retrievedConfig.Algorithm != config.Algorithm {
		t.Errorf("GetEncryptor() algorithm = %v, want %v", retrievedConfig.Algorithm, config.Algorithm)
	}

	// Verify encryptor works by encrypting/decrypting
	testData := []byte("test wal data")
	encrypted, err := encryptor.Encrypt(testData)
	if err != nil {
		t.Fatalf("Encryptor.Encrypt() error = %v", err)
	}

	decrypted, err := encryptor.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Encryptor.Decrypt() error = %v", err)
	}

	if string(decrypted) != string(testData) {
		t.Errorf("Encryptor decrypt mismatch: got %v, want %v", string(decrypted), string(testData))
	}
}

// TestEncryptedWAL_NewEncryptedWAL_ErrorPaths tests error handling in NewEncryptedWAL
func TestEncryptedWAL_NewEncryptedWAL_ErrorPaths(t *testing.T) {
	t.Run("Invalid encryption config", func(t *testing.T) {
		dataDir := filepath.Join(os.TempDir(), "test-wal-new-error")
		defer os.RemoveAll(dataDir)
		os.MkdirAll(dataDir, 0755)

		walPath := filepath.Join(dataDir, "test.wal")

		// Create invalid encryption config (wrong key length)
		config := &Config{
			Algorithm: AlgorithmAES256GCM,
			Key:       []byte("short"), // Invalid key length
		}

		// Should fail to create encrypted WAL
		_, err := NewEncryptedWAL(walPath, config)
		if err == nil {
			t.Error("NewEncryptedWAL() expected error with invalid config, got nil")
		}
	})
}

// TestEncryptedWAL_Append_NoEncryption tests Append with no encryption
func TestEncryptedWAL_Append_NoEncryption(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), "test-wal-append-none")
	defer os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0755)

	walPath := filepath.Join(dataDir, "test.wal")

	// Create config with no encryption
	config := DefaultConfig()

	// Create encrypted WAL (but with no encryption)
	wal, err := NewEncryptedWAL(walPath, config)
	if err != nil {
		t.Fatalf("Failed to create encrypted WAL: %v", err)
	}
	defer wal.Close()

	// Append record
	record := &storage.LogRecord{
		Type:   storage.LogRecordInsert,
		TxnID:  1,
		PageID: 0,
		Data:   []byte("unencrypted data"),
	}

	lsn, err := wal.Append(record)
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	if lsn == 0 {
		t.Error("Append() returned LSN 0")
	}

	// Flush and replay
	wal.Flush()
	wal.Close()

	// Reopen and replay
	wal, _ = NewEncryptedWAL(walPath, config)
	defer wal.Close()

	records, err := wal.Replay()
	if err != nil {
		t.Fatalf("Replay() error = %v", err)
	}

	if len(records) != 1 {
		t.Errorf("Replay() returned %d records, want 1", len(records))
	}

	if string(records[0].Data) != string(record.Data) {
		t.Errorf("Replay() data = %v, want %v", string(records[0].Data), string(record.Data))
	}
}

// TestEncryptedWAL_Append_EmptyData tests Append with empty data
func TestEncryptedWAL_Append_EmptyData(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), "test-wal-append-empty")
	defer os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0755)

	walPath := filepath.Join(dataDir, "test.wal")

	config, _ := NewConfigFromPassword("test-password", AlgorithmAES256GCM)

	wal, err := NewEncryptedWAL(walPath, config)
	if err != nil {
		t.Fatalf("Failed to create encrypted WAL: %v", err)
	}
	defer wal.Close()

	// Append record with empty data
	record := &storage.LogRecord{
		Type:   storage.LogRecordCommit,
		TxnID:  1,
		PageID: 0,
		Data:   []byte{}, // Empty data
	}

	lsn, err := wal.Append(record)
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	if lsn == 0 {
		t.Error("Append() returned LSN 0")
	}
}

// TestEncryptedWAL_Replay_NoEncryption tests Replay with no encryption
func TestEncryptedWAL_Replay_NoEncryption(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), "test-wal-replay-none")
	defer os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0755)

	walPath := filepath.Join(dataDir, "test.wal")

	// Create and write with no encryption
	config := DefaultConfig()
	wal, _ := NewEncryptedWAL(walPath, config)

	record := &storage.LogRecord{
		Type:   storage.LogRecordInsert,
		TxnID:  1,
		PageID: 0,
		Data:   []byte("unencrypted data"),
	}

	wal.Append(record)
	wal.Flush()
	wal.Close()

	// Replay
	wal, _ = NewEncryptedWAL(walPath, config)
	defer wal.Close()

	records, err := wal.Replay()
	if err != nil {
		t.Fatalf("Replay() error = %v", err)
	}

	if len(records) != 1 {
		t.Errorf("Replay() returned %d records, want 1", len(records))
	}

	if string(records[0].Data) != string(record.Data) {
		t.Errorf("Replay() data mismatch")
	}
}

// TestEncryptedWAL_Replay_MigrationScenario tests replaying with mixed encrypted/unencrypted records
func TestEncryptedWAL_Replay_MigrationScenario(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), "test-wal-replay-migration")
	defer os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0755)

	walPath := filepath.Join(dataDir, "test.wal")

	// Write unencrypted record first
	config1 := DefaultConfig()
	wal1, _ := NewEncryptedWAL(walPath, config1)

	// Use short data that will be less than header size (migration scenario)
	record1 := &storage.LogRecord{
		Type:   storage.LogRecordInsert,
		TxnID:  1,
		PageID: 0,
		Data:   []byte("old"), // Short data (3 bytes < 5 byte header)
	}

	wal1.Append(record1)
	wal1.Flush()
	wal1.Close()

	// Now append encrypted record with encryption enabled
	config2, _ := NewConfigFromPassword("test-password", AlgorithmAES256GCM)
	wal2, _ := NewEncryptedWAL(walPath, config2)

	record2 := &storage.LogRecord{
		Type:   storage.LogRecordUpdate,
		TxnID:  2,
		PageID: 1,
		Data:   []byte("new encrypted data"),
	}

	wal2.Append(record2)
	wal2.Flush()
	wal2.Close()

	// Replay with encryption enabled - should handle both encrypted and unencrypted records
	wal3, _ := NewEncryptedWAL(walPath, config2)
	defer wal3.Close()

	records, err := wal3.Replay()
	if err != nil {
		t.Fatalf("Replay() error = %v", err)
	}

	// Should get both records
	if len(records) != 2 {
		t.Errorf("Replay() returned %d records, want 2", len(records))
	}

	// First record should be unencrypted (migration scenario - short data)
	if string(records[0].Data) != string(record1.Data) {
		t.Errorf("Replay() first record data mismatch")
	}

	// Second record should be encrypted and decrypted
	if string(records[1].Data) != string(record2.Data) {
		t.Errorf("Replay() second record data mismatch")
	}
}

// TestEncryptedWAL_Replay_AlgorithmMismatch tests error handling for algorithm mismatch
func TestEncryptedWAL_Replay_AlgorithmMismatch(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), "test-wal-replay-mismatch")
	defer os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0755)

	walPath := filepath.Join(dataDir, "test.wal")

	// Write with GCM
	config1, _ := NewConfigFromPassword("test-password", AlgorithmAES256GCM)
	wal1, _ := NewEncryptedWAL(walPath, config1)

	record := &storage.LogRecord{
		Type:   storage.LogRecordInsert,
		TxnID:  1,
		PageID: 0,
		Data:   []byte("encrypted with GCM"),
	}

	wal1.Append(record)
	wal1.Flush()
	wal1.Close()

	// Try to replay with CTR (different algorithm)
	config2, _ := NewConfigFromPassword("test-password", AlgorithmAES256CTR)
	wal2, _ := NewEncryptedWAL(walPath, config2)
	defer wal2.Close()

	_, err := wal2.Replay()
	if err == nil {
		t.Error("Replay() expected error with algorithm mismatch, got nil")
	}
}

// TestEncryptedWAL_Replay_ShortRecord tests handling of records with insufficient header
func TestEncryptedWAL_Replay_ShortRecord(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), "test-wal-replay-short")
	defer os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0755)

	walPath := filepath.Join(dataDir, "test.wal")

	// Write a record with very short data (less than header size)
	config := DefaultConfig()
	wal1, _ := NewEncryptedWAL(walPath, config)

	record := &storage.LogRecord{
		Type:   storage.LogRecordInsert,
		TxnID:  1,
		PageID: 0,
		Data:   []byte("abc"), // Less than EncryptedWALHeaderSize (5 bytes)
	}

	wal1.Append(record)
	wal1.Flush()
	wal1.Close()

	// Replay with encryption enabled
	config2, _ := NewConfigFromPassword("test-password", AlgorithmAES256GCM)
	wal2, _ := NewEncryptedWAL(walPath, config2)
	defer wal2.Close()

	// Should handle short records gracefully (migration scenario)
	records, err := wal2.Replay()
	if err != nil {
		t.Fatalf("Replay() error = %v", err)
	}

	if len(records) != 1 {
		t.Errorf("Replay() returned %d records, want 1", len(records))
	}

	// Data should be unchanged (treated as unencrypted)
	if string(records[0].Data) != string(record.Data) {
		t.Errorf("Replay() data mismatch for short record")
	}
}

// TestEncryptedWAL_Replay_DecryptionError tests error handling during decryption
func TestEncryptedWAL_Replay_DecryptionError(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), "test-wal-replay-decrypt-error")
	defer os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0755)

	walPath := filepath.Join(dataDir, "test.wal")

	// Write encrypted record
	config, _ := NewConfigFromPassword("test-password", AlgorithmAES256GCM)
	wal1, _ := NewEncryptedWAL(walPath, config)

	record := &storage.LogRecord{
		Type:   storage.LogRecordInsert,
		TxnID:  1,
		PageID: 0,
		Data:   []byte("encrypted data that will be corrupted"),
	}

	wal1.Append(record)
	wal1.Flush()
	wal1.Close()

	// Manually corrupt the WAL file to trigger decryption error
	rawData, _ := os.ReadFile(walPath)
	if len(rawData) > 50 {
		// Corrupt some bytes in the encrypted data
		rawData[len(rawData)-10] ^= 0xFF
		os.WriteFile(walPath, rawData, 0644)
	}

	// Try to replay - should fail due to corruption
	wal2, _ := NewEncryptedWAL(walPath, config)
	defer wal2.Close()

	_, err := wal2.Replay()
	if err == nil {
		t.Error("Replay() should fail with corrupted encrypted data")
	}
}

// TestReadPage_WithNoEncryption tests reading pages when encryption is disabled
func TestReadPage_WithNoEncryption(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), "test-no-encryption-read")
	defer os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0755)

	dataPath := filepath.Join(dataDir, "test.db")

	// Create encrypted disk manager with no encryption
	config := DefaultConfig() // AlgorithmNone
	edm, _ := NewEncryptedDiskManager(dataPath, config)
	defer edm.Close()

	// Write a page
	pageID, _ := edm.AllocatePage()
	page := storage.NewPage(pageID, storage.PageTypeData)
	testData := []byte("unencrypted data")
	copy(page.Data[:len(testData)], testData)

	err := edm.WritePage(page)
	if err != nil {
		t.Fatalf("WritePage() error = %v", err)
	}

	edm.Sync()

	// Read page - should succeed without encryption
	readPage, err := edm.ReadPage(pageID)
	if err != nil {
		t.Errorf("ReadPage() unexpected error: %v", err)
	}

	if readPage == nil {
		t.Error("ReadPage() returned nil page")
	}

	// Verify data matches
	if string(readPage.Data[:len(testData)]) != string(testData) {
		t.Error("ReadPage() data mismatch")
	}
}
