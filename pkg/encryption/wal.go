package encryption

import (
	"encoding/binary"
	"fmt"

	"github.com/mnohosten/laura-db/pkg/storage"
)

const (
	// EncryptedWALHeaderSize is the size of the encrypted WAL record header
	// [1-byte algorithm][4-byte original size]
	EncryptedWALHeaderSize = 5
)

// EncryptedWAL wraps a WAL with transparent encryption
type EncryptedWAL struct {
	wal       *storage.WAL
	encryptor *Encryptor
}

// NewEncryptedWAL creates a new encrypted WAL
func NewEncryptedWAL(path string, config *Config) (*EncryptedWAL, error) {
	// Create underlying WAL
	wal, err := storage.NewWAL(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create WAL: %w", err)
	}

	// Create encryptor
	encryptor, err := NewEncryptor(config)
	if err != nil {
		wal.Close()
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	return &EncryptedWAL{
		wal:       wal,
		encryptor: encryptor,
	}, nil
}

// Append encrypts and writes a log record to the WAL
func (ew *EncryptedWAL) Append(record *storage.LogRecord) (uint64, error) {
	// If encryption is disabled, append as-is
	if ew.encryptor.config.Algorithm == AlgorithmNone {
		return ew.wal.Append(record)
	}

	// Create a copy to avoid modifying the original
	encryptedRecord := &storage.LogRecord{
		LSN:     record.LSN,
		Type:    record.Type,
		TxnID:   record.TxnID,
		PageID:  record.PageID,
		PrevLSN: record.PrevLSN,
	}

	// Encrypt record data if present
	if len(record.Data) > 0 {
		encryptedData, err := ew.encryptor.Encrypt(record.Data)
		if err != nil {
			return 0, fmt.Errorf("failed to encrypt WAL record: %w", err)
		}

		// Build encrypted data with header
		// Header: [1-byte algorithm][4-byte original size]
		headerSize := EncryptedWALHeaderSize
		encryptedRecord.Data = make([]byte, headerSize+len(encryptedData))
		encryptedRecord.Data[0] = byte(ew.encryptor.config.Algorithm)
		binary.LittleEndian.PutUint32(encryptedRecord.Data[1:5], uint32(len(record.Data)))
		copy(encryptedRecord.Data[headerSize:], encryptedData)
	}

	return ew.wal.Append(encryptedRecord)
}

// Replay reads and decrypts all log records from the WAL
func (ew *EncryptedWAL) Replay() ([]*storage.LogRecord, error) {
	// Read all records
	records, err := ew.wal.Replay()
	if err != nil {
		return nil, err
	}

	// If encryption is disabled, return as-is
	if ew.encryptor.config.Algorithm == AlgorithmNone {
		return records, nil
	}

	// Decrypt each record's data
	for i, record := range records {
		if len(record.Data) < EncryptedWALHeaderSize {
			// Unencrypted record (migration scenario or empty data)
			continue
		}

		// Read encryption header
		algorithm := Algorithm(record.Data[0])
		if algorithm == AlgorithmNone {
			// Unencrypted record
			continue
		}

		// Validate algorithm matches
		if algorithm != ew.encryptor.config.Algorithm {
			return nil, fmt.Errorf("encryption algorithm mismatch in WAL record %d: expected %v, got %v",
				i, ew.encryptor.config.Algorithm, algorithm)
		}

		originalSize := binary.LittleEndian.Uint32(record.Data[1:5])
		encryptedData := record.Data[EncryptedWALHeaderSize:]

		// Decrypt data
		decryptedData, err := ew.encryptor.Decrypt(encryptedData)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt WAL record %d: %w", i, err)
		}

		// Validate size
		if len(decryptedData) != int(originalSize) {
			return nil, fmt.Errorf("decrypted size mismatch for WAL record %d: expected %d, got %d",
				i, originalSize, len(decryptedData))
		}

		// Update record data
		record.Data = decryptedData
	}

	return records, nil
}

// Checkpoint creates a checkpoint in the WAL
func (ew *EncryptedWAL) Checkpoint() error {
	return ew.wal.Checkpoint()
}

// Flush flushes the WAL to disk
func (ew *EncryptedWAL) Flush() error {
	return ew.wal.Flush()
}

// Close closes the WAL
func (ew *EncryptedWAL) Close() error {
	return ew.wal.Close()
}

// GetEncryptor returns the encryptor
func (ew *EncryptedWAL) GetEncryptor() *Encryptor {
	return ew.encryptor
}
