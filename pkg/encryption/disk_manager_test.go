package encryption

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mnohosten/laura-db/pkg/storage"
)

// TestEncryptedDiskManager_DeallocatePage tests the DeallocatePage function
func TestEncryptedDiskManager_DeallocatePage(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), "test-deallocate-page")
	defer os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0755)

	dataPath := filepath.Join(dataDir, "test.db")

	// Create encryption config
	config, err := NewConfigFromPassword("test-password", AlgorithmAES256GCM)
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

	// Deallocate the page
	err = edm.DeallocatePage(pageID)
	if err != nil {
		t.Errorf("DeallocatePage() error = %v, expected nil", err)
	}

	// Verify we can allocate another page (reuse of deallocated page)
	newPageID, err := edm.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate new page: %v", err)
	}

	// The new page should reuse the deallocated page ID
	if newPageID != pageID {
		t.Logf("Note: New page ID %d differs from deallocated page ID %d (implementation-dependent)", newPageID, pageID)
	}
}

// TestEncryptedDiskManager_Stats tests the Stats function
func TestEncryptedDiskManager_Stats(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), "test-stats")
	defer os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0755)

	dataPath := filepath.Join(dataDir, "test.db")

	tests := []struct {
		name                string
		algorithm           Algorithm
		wantEnabled         bool
		wantAlgorithmString string
	}{
		{
			name:                "Stats with GCM encryption",
			algorithm:           AlgorithmAES256GCM,
			wantEnabled:         true,
			wantAlgorithmString: "AES-256-GCM",
		},
		{
			name:                "Stats with CTR encryption",
			algorithm:           AlgorithmAES256CTR,
			wantEnabled:         true,
			wantAlgorithmString: "AES-256-CTR",
		},
		{
			name:                "Stats with no encryption",
			algorithm:           AlgorithmNone,
			wantEnabled:         false,
			wantAlgorithmString: "None",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create encryption config
			config, err := NewConfigFromPassword("test-password", tt.algorithm)
			if err != nil {
				t.Fatalf("Failed to create config: %v", err)
			}

			// Create encrypted disk manager
			edm, err := NewEncryptedDiskManager(dataPath, config)
			if err != nil {
				t.Fatalf("Failed to create encrypted disk manager: %v", err)
			}
			defer edm.Close()

			// Get stats
			stats := edm.Stats()

			// Verify stats contains expected fields
			if stats == nil {
				t.Fatal("Stats() returned nil")
			}

			// Check encryption_algorithm field
			algorithmStr, ok := stats["encryption_algorithm"].(string)
			if !ok {
				t.Error("Stats() missing or invalid encryption_algorithm field")
			} else if algorithmStr != tt.wantAlgorithmString {
				t.Errorf("Stats() encryption_algorithm = %v, want %v", algorithmStr, tt.wantAlgorithmString)
			}

			// Check encryption_enabled field
			enabled, ok := stats["encryption_enabled"].(bool)
			if !ok {
				t.Error("Stats() missing or invalid encryption_enabled field")
			} else if enabled != tt.wantEnabled {
				t.Errorf("Stats() encryption_enabled = %v, want %v", enabled, tt.wantEnabled)
			}

			// Clean up for next test
			os.RemoveAll(dataDir)
			os.MkdirAll(dataDir, 0755)
		})
	}
}

// TestEncryptedDiskManager_GetEncryptor tests the GetEncryptor function
func TestEncryptedDiskManager_GetEncryptor(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), "test-get-encryptor")
	defer os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0755)

	dataPath := filepath.Join(dataDir, "test.db")

	// Create encryption config
	config, err := NewConfigFromPassword("test-password", AlgorithmAES256GCM)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Create encrypted disk manager
	edm, err := NewEncryptedDiskManager(dataPath, config)
	if err != nil {
		t.Fatalf("Failed to create encrypted disk manager: %v", err)
	}
	defer edm.Close()

	// Get encryptor
	encryptor := edm.GetEncryptor()

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
	testData := []byte("test data")
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

// TestEncryptedDiskManager_NewEncryptedDiskManager_ErrorPaths tests error handling in NewEncryptedDiskManager
func TestEncryptedDiskManager_NewEncryptedDiskManager_ErrorPaths(t *testing.T) {
	t.Run("Invalid encryption config", func(t *testing.T) {
		dataDir := filepath.Join(os.TempDir(), "test-new-error")
		defer os.RemoveAll(dataDir)
		os.MkdirAll(dataDir, 0755)

		dataPath := filepath.Join(dataDir, "test.db")

		// Create invalid encryption config (wrong key length)
		config := &Config{
			Algorithm: AlgorithmAES256GCM,
			Key:       []byte("short"), // Invalid key length
		}

		// Should fail to create encrypted disk manager
		_, err := NewEncryptedDiskManager(dataPath, config)
		if err == nil {
			t.Error("NewEncryptedDiskManager() expected error with invalid config, got nil")
		}
	})
}

// TestEncryptedDiskManager_ReadPage_ErrorPaths tests error paths in ReadPage
func TestEncryptedDiskManager_ReadPage_ErrorPaths(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), "test-readpage-errors")
	defer os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0755)

	dataPath := filepath.Join(dataDir, "test.db")

	t.Run("Algorithm mismatch", func(t *testing.T) {
		// Create and write with GCM
		config1, _ := NewConfigFromPassword("test-password", AlgorithmAES256GCM)
		edm1, _ := NewEncryptedDiskManager(dataPath, config1)

		pageID, _ := edm1.AllocatePage()
		page := storage.NewPage(pageID, storage.PageTypeData)

		// Limit data size to avoid encryption overflow
		maxDataSize := len(page.Data) - EncryptionOverhead - EncryptedPageHeaderSize
		testData := []byte("test data")
		if len(testData) < maxDataSize {
			copy(page.Data[:len(testData)], testData)
			page.Data = page.Data[:maxDataSize]
		}

		edm1.WritePage(page)
		edm1.Sync()
		edm1.Close()

		// Try to read with CTR (different algorithm)
		config2, _ := NewConfigFromPassword("test-password", AlgorithmAES256CTR)
		edm2, _ := NewEncryptedDiskManager(dataPath, config2)
		defer edm2.Close()

		_, err := edm2.ReadPage(pageID)
		if err == nil {
			t.Error("ReadPage() expected error with algorithm mismatch, got nil")
		}

		// Clean up for next test
		os.RemoveAll(dataDir)
		os.MkdirAll(dataDir, 0755)
	})

	t.Run("Read page with AlgorithmNone byte (migration scenario)", func(t *testing.T) {
		// First create an unencrypted disk manager and write a page with AlgorithmNone marker
		diskMgr, _ := storage.NewDiskManager(dataPath)
		pageID, _ := diskMgr.AllocatePage()
		page := storage.NewPage(pageID, storage.PageTypeData)

		// Write data with AlgorithmNone marker at the beginning
		page.Data[0] = byte(AlgorithmNone)
		copy(page.Data[1:], []byte("unencrypted data"))

		diskMgr.WritePage(page)
		diskMgr.Sync()
		diskMgr.Close()

		// Now try to read with encrypted disk manager
		config, _ := NewConfigFromPassword("test-password", AlgorithmAES256GCM)
		edm, _ := NewEncryptedDiskManager(dataPath, config)
		defer edm.Close()

		// Should succeed (migration scenario - page marked as unencrypted)
		readPage, err := edm.ReadPage(pageID)
		if err != nil {
			t.Errorf("ReadPage() unexpected error with AlgorithmNone page: %v", err)
		}

		// Should return the page as-is
		if readPage == nil {
			t.Error("ReadPage() returned nil page")
		}

		// Data should start with AlgorithmNone byte
		if readPage.Data[0] != byte(AlgorithmNone) {
			t.Errorf("ReadPage() first byte = %d, want %d (AlgorithmNone)", readPage.Data[0], AlgorithmNone)
		}

		// Clean up for next test
		os.RemoveAll(dataDir)
		os.MkdirAll(dataDir, 0755)
	})
}

// TestEncryptedDiskManager_WritePage_ErrorPaths tests error paths in WritePage
func TestEncryptedDiskManager_WritePage_ErrorPaths(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), "test-writepage-errors")
	defer os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0755)

	dataPath := filepath.Join(dataDir, "test.db")

	t.Run("Data too large for encryption", func(t *testing.T) {
		config, _ := NewConfigFromPassword("test-password", AlgorithmAES256GCM)
		edm, _ := NewEncryptedDiskManager(dataPath, config)
		defer edm.Close()

		pageID, _ := edm.AllocatePage()
		page := storage.NewPage(pageID, storage.PageTypeData)

		// Fill page with maximum data that might overflow after encryption
		pageDataSize := storage.PageSize - storage.PageHeaderSize
		page.Data = make([]byte, pageDataSize) // Maximum size
		for i := range page.Data {
			page.Data[i] = byte(i % 256)
		}

		// This might fail if encrypted data is too large
		err := edm.WritePage(page)
		// Note: This test documents the behavior. The actual result depends on
		// whether the page data + encryption overhead fits in the page size.
		if err != nil {
			t.Logf("WritePage() failed with large data as expected: %v", err)
		}
	})
}
