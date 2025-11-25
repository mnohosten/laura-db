package encryption

import (
	"encoding/binary"
	"fmt"

	"github.com/mnohosten/laura-db/pkg/storage"
)

const (
	// EncryptedPageHeaderSize is the size of the encrypted page header
	// [1-byte algorithm][4-byte original size]
	EncryptedPageHeaderSize = 5

	// EncryptionOverhead is the maximum overhead from encryption
	// GCM: 12 bytes (nonce) + 16 bytes (auth tag) = 28 bytes
	// CTR: 16 bytes (IV) = 16 bytes
	// We use the larger value for safety
	EncryptionOverhead = 28
)

// EncryptedDiskManager wraps a DiskManager with transparent encryption
type EncryptedDiskManager struct {
	diskMgr   *storage.DiskManager
	encryptor *Encryptor
}

// NewEncryptedDiskManager creates a new encrypted disk manager
func NewEncryptedDiskManager(path string, config *Config) (*EncryptedDiskManager, error) {
	// Create underlying disk manager
	diskMgr, err := storage.NewDiskManager(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create disk manager: %w", err)
	}

	// Create encryptor
	encryptor, err := NewEncryptor(config)
	if err != nil {
		diskMgr.Close()
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	return &EncryptedDiskManager{
		diskMgr:   diskMgr,
		encryptor: encryptor,
	}, nil
}

// ReadPage reads and decrypts a page from disk
func (edm *EncryptedDiskManager) ReadPage(pageID storage.PageID) (*storage.Page, error) {
	// Read encrypted page
	encryptedPage, err := edm.diskMgr.ReadPage(pageID)
	if err != nil {
		return nil, err
	}

	// If encryption is disabled, return as-is
	if edm.encryptor.config.Algorithm == AlgorithmNone {
		return encryptedPage, nil
	}

	// Check if page has encrypted data
	if len(encryptedPage.Data) < EncryptedPageHeaderSize {
		// Unencrypted page (migration scenario or new page)
		return encryptedPage, nil
	}

	// Read encryption header
	algorithm := Algorithm(encryptedPage.Data[0])
	if algorithm == AlgorithmNone {
		// Unencrypted page
		return encryptedPage, nil
	}

	// Validate algorithm matches
	if algorithm != edm.encryptor.config.Algorithm {
		return nil, fmt.Errorf("encryption algorithm mismatch: expected %v, got %v",
			edm.encryptor.config.Algorithm, algorithm)
	}

	originalSize := binary.LittleEndian.Uint32(encryptedPage.Data[1:5])
	encryptedData := encryptedPage.Data[EncryptedPageHeaderSize:]

	// Decrypt data
	decryptedData, err := edm.encryptor.Decrypt(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt page %d: %w", pageID, err)
	}

	// Validate size
	if len(decryptedData) != int(originalSize) {
		return nil, fmt.Errorf("decrypted size mismatch for page %d: expected %d, got %d",
			pageID, originalSize, len(decryptedData))
	}

	// Create properly sized page data buffer and copy decrypted data
	pageDataSize := storage.PageSize - storage.PageHeaderSize
	newPageData := make([]byte, pageDataSize)
	copy(newPageData, decryptedData)
	encryptedPage.Data = newPageData

	return encryptedPage, nil
}

// WritePage encrypts and writes a page to disk
func (edm *EncryptedDiskManager) WritePage(page *storage.Page) error {
	// If encryption is disabled, write as-is
	if edm.encryptor.config.Algorithm == AlgorithmNone {
		return edm.diskMgr.WritePage(page)
	}

	// Create a copy to avoid modifying the original
	encryptedPage := &storage.Page{
		ID:       page.ID,
		Type:     page.Type,
		Flags:    page.Flags,
		LSN:      page.LSN,
		IsDirty:  page.IsDirty,
		PinCount: page.PinCount,
	}

	// Encrypt page data
	encryptedData, err := edm.encryptor.Encrypt(page.Data)
	if err != nil {
		return fmt.Errorf("failed to encrypt page %d: %w", page.ID, err)
	}

	// Build encrypted page data with header
	// Header: [1-byte algorithm][4-byte original size]
	// We need to ensure the page data fits in the page (4KB - header size)
	headerSize := EncryptedPageHeaderSize
	totalEncryptedSize := headerSize + len(encryptedData)

	// If encrypted data is too large, we have a problem
	pageDataSize := storage.PageSize - storage.PageHeaderSize
	if totalEncryptedSize > pageDataSize {
		return fmt.Errorf("encrypted data too large: %d bytes (max %d)", totalEncryptedSize, pageDataSize)
	}

	encryptedPage.Data = make([]byte, pageDataSize)
	encryptedPage.Data[0] = byte(edm.encryptor.config.Algorithm)
	binary.LittleEndian.PutUint32(encryptedPage.Data[1:5], uint32(len(page.Data)))
	copy(encryptedPage.Data[headerSize:], encryptedData)

	// Write encrypted page
	return edm.diskMgr.WritePage(encryptedPage)
}

// AllocatePage allocates a new page
func (edm *EncryptedDiskManager) AllocatePage() (storage.PageID, error) {
	return edm.diskMgr.AllocatePage()
}

// DeallocatePage marks a page as free
func (edm *EncryptedDiskManager) DeallocatePage(pageID storage.PageID) error {
	return edm.diskMgr.DeallocatePage(pageID)
}

// Sync flushes all data to disk
func (edm *EncryptedDiskManager) Sync() error {
	return edm.diskMgr.Sync()
}

// Close closes the disk manager
func (edm *EncryptedDiskManager) Close() error {
	return edm.diskMgr.Close()
}

// Stats returns disk manager statistics
func (edm *EncryptedDiskManager) Stats() map[string]interface{} {
	stats := edm.diskMgr.Stats()
	stats["encryption_algorithm"] = edm.encryptor.config.Algorithm.String()
	stats["encryption_enabled"] = edm.encryptor.config.Algorithm != AlgorithmNone
	return stats
}

// GetEncryptor returns the encryptor
func (edm *EncryptedDiskManager) GetEncryptor() *Encryptor {
	return edm.encryptor
}
