package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mnohosten/laura-db/pkg/encryption"
	"github.com/mnohosten/laura-db/pkg/storage"
)

func main() {
	fmt.Println("=== LauraDB Encryption at Rest Demo ===\n")

	// Clean up any existing data
	dataDir := "./encryption-demo-data"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	// Demo 1: Basic Encryption with Password
	fmt.Println("Demo 1: Basic Encryption with Password")
	fmt.Println("---------------------------------------")
	demo1BasicEncryption(dataDir)

	// Demo 2: Different Encryption Algorithms
	fmt.Println("\nDemo 2: Encryption Algorithms Comparison")
	fmt.Println("----------------------------------------")
	demo2EncryptionAlgorithms(dataDir)

	// Demo 3: Encrypted WAL
	fmt.Println("\nDemo 3: Encrypted Write-Ahead Log")
	fmt.Println("----------------------------------")
	demo3EncryptedWAL(dataDir)

	// Demo 4: Wrong Key Protection
	fmt.Println("\nDemo 4: Protection Against Wrong Keys")
	fmt.Println("--------------------------------------")
	demo4WrongKeyProtection(dataDir)

	fmt.Println("\n=== All Demos Completed Successfully ===")
}

func demo1BasicEncryption(baseDir string) {
	dataDir := filepath.Join(baseDir, "demo1")
	os.MkdirAll(dataDir, 0755)
	defer os.RemoveAll(dataDir)

	dataPath := filepath.Join(dataDir, "encrypted.db")

	// Create encryption config from password
	config, err := encryption.NewConfigFromPassword("my-secure-password-123", encryption.AlgorithmAES256GCM)
	if err != nil {
		fmt.Printf("Error creating config: %v\n", err)
		return
	}

	fmt.Printf("✓ Created encryption config with algorithm: %s\n", config.Algorithm)
	fmt.Printf("✓ Encryption key derived from password (32 bytes)\n")

	// Create encrypted disk manager
	edm, err := encryption.NewEncryptedDiskManager(dataPath, config)
	if err != nil {
		fmt.Printf("Error creating encrypted disk manager: %v\n", err)
		return
	}
	defer edm.Close()

	// Write encrypted data
	pageID, _ := edm.AllocatePage()
	page := storage.NewPage(pageID, storage.PageTypeData)

	secretData := []byte("This is confidential customer data!")
	maxSize := len(page.Data) - encryption.EncryptionOverhead - encryption.EncryptedPageHeaderSize
	page.Data = page.Data[:maxSize]
	copy(page.Data, secretData)

	if err := edm.WritePage(page); err != nil {
		fmt.Printf("Error writing page: %v\n", err)
		return
	}

	edm.Sync()
	fmt.Printf("✓ Wrote encrypted page to disk\n")

	// Verify data is encrypted on disk
	rawData, _ := os.ReadFile(dataPath)
	if containsBytes(rawData, secretData) {
		fmt.Println("✗ WARNING: Data is NOT encrypted!")
	} else {
		fmt.Println("✓ Verified: Data is encrypted on disk (plaintext not found)")
	}

	// Read and decrypt
	readPage, err := edm.ReadPage(pageID)
	if err != nil {
		fmt.Printf("Error reading page: %v\n", err)
		return
	}

	if string(readPage.Data[:len(secretData)]) == string(secretData) {
		fmt.Println("✓ Successfully decrypted and verified data")
	}
}

func demo2EncryptionAlgorithms(baseDir string) {
	algorithms := []encryption.Algorithm{
		encryption.AlgorithmAES256GCM,
		encryption.AlgorithmAES256CTR,
	}

	for _, alg := range algorithms {
		dataDir := filepath.Join(baseDir, "demo2", alg.String())
		os.MkdirAll(dataDir, 0755)
		defer os.RemoveAll(dataDir)

		dataPath := filepath.Join(dataDir, "test.db")

		config, _ := encryption.NewConfigFromPassword("test-password", alg)
		edm, _ := encryption.NewEncryptedDiskManager(dataPath, config)

		pageID, _ := edm.AllocatePage()
		page := storage.NewPage(pageID, storage.PageTypeData)

		testData := []byte("Test data for algorithm comparison")
		maxSize := len(page.Data) - encryption.EncryptionOverhead - encryption.EncryptedPageHeaderSize
		page.Data = page.Data[:maxSize]
		copy(page.Data, testData)

		edm.WritePage(page)
		edm.Sync()

		// Check file size
		fileInfo, _ := os.Stat(dataPath)
		fmt.Printf("✓ %s: Encrypted %d bytes (file size: %d bytes)\n",
			alg, len(testData), fileInfo.Size())

		edm.Close()
	}
}

func demo3EncryptedWAL(baseDir string) {
	dataDir := filepath.Join(baseDir, "demo3")
	os.MkdirAll(dataDir, 0755)
	defer os.RemoveAll(dataDir)

	walPath := filepath.Join(dataDir, "encrypted.wal")

	config, _ := encryption.NewConfigFromPassword("wal-password", encryption.AlgorithmAES256GCM)
	wal, _ := encryption.NewEncryptedWAL(walPath, config)

	// Write multiple log entries
	entries := []string{
		"Transaction 1: INSERT user data",
		"Transaction 1: UPDATE user preferences",
		"Transaction 1: COMMIT",
	}

	fmt.Println("Writing encrypted log entries...")
	for i, entry := range entries {
		record := &storage.LogRecord{
			Type:   storage.LogRecordInsert,
			TxnID:  1,
			PageID: storage.PageID(i),
			Data:   []byte(entry),
		}
		lsn, _ := wal.Append(record)
		fmt.Printf("  ✓ LSN %d: %s\n", lsn, entry)
	}

	wal.Flush()
	wal.Close()

	// Verify WAL is encrypted
	rawData, _ := os.ReadFile(walPath)
	if containsBytes(rawData, []byte("user data")) {
		fmt.Println("✗ WARNING: WAL is NOT encrypted!")
	} else {
		fmt.Println("✓ Verified: WAL is encrypted on disk")
	}

	// Replay encrypted WAL
	wal, _ = encryption.NewEncryptedWAL(walPath, config)
	defer wal.Close()

	records, _ := wal.Replay()
	fmt.Printf("✓ Successfully replayed %d encrypted WAL records\n", len(records))
}

func demo4WrongKeyProtection(baseDir string) {
	dataDir := filepath.Join(baseDir, "demo4")
	os.MkdirAll(dataDir, 0755)
	defer os.RemoveAll(dataDir)

	dataPath := filepath.Join(dataDir, "protected.db")

	// Write with correct key
	correctConfig, _ := encryption.NewConfigFromPassword("correct-password", encryption.AlgorithmAES256GCM)
	edm, _ := encryption.NewEncryptedDiskManager(dataPath, correctConfig)

	pageID, _ := edm.AllocatePage()
	page := storage.NewPage(pageID, storage.PageTypeData)

	secretData := []byte("Top secret information")
	maxSize := len(page.Data) - encryption.EncryptionOverhead - encryption.EncryptedPageHeaderSize
	page.Data = page.Data[:maxSize]
	copy(page.Data, secretData)

	edm.WritePage(page)
	edm.Sync()
	edm.Close()

	fmt.Println("✓ Wrote data with correct password")

	// Try to read with wrong key
	wrongConfig, _ := encryption.NewConfigFromPassword("wrong-password", encryption.AlgorithmAES256GCM)
	edm, _ = encryption.NewEncryptedDiskManager(dataPath, wrongConfig)
	defer edm.Close()

	_, err := edm.ReadPage(pageID)
	if err != nil {
		fmt.Println("✓ Access denied with wrong password (authentication failed)")
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Println("✗ WARNING: Wrong password was accepted!")
	}

	// Read with correct key
	edm2, _ := encryption.NewEncryptedDiskManager(dataPath, correctConfig)
	defer edm2.Close()

	readPage, err := edm2.ReadPage(pageID)
	if err == nil && string(readPage.Data[:len(secretData)]) == string(secretData) {
		fmt.Println("✓ Access granted with correct password")
	}
}

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
