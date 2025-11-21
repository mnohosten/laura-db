package storage

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
)

// LogRecordType represents the type of WAL record
type LogRecordType uint8

const (
	LogRecordInsert LogRecordType = iota
	LogRecordUpdate
	LogRecordDelete
	LogRecordCheckpoint
	LogRecordCommit
	LogRecordAbort
)

// LogRecord represents a single WAL entry
type LogRecord struct {
	LSN       uint64        // Log Sequence Number
	Type      LogRecordType
	TxnID     uint64        // Transaction ID
	PageID    PageID
	Data      []byte
	PrevLSN   uint64        // Previous LSN for this transaction
}

// WAL (Write-Ahead Log) ensures durability
type WAL struct {
	file       *os.File
	mu         sync.Mutex
	currentLSN uint64
	buffer     []byte
	bufferSize int
}

// NewWAL creates a new Write-Ahead Log
func NewWAL(path string) (*WAL, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAL file: %w", err)
	}

	// Get current position to set LSN
	pos, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to seek WAL file: %w", err)
	}

	return &WAL{
		file:       file,
		currentLSN: uint64(pos),
		buffer:     make([]byte, 0, 4096),
		bufferSize: 4096,
	}, nil
}

// Append writes a log record to the WAL and returns its LSN
func (w *WAL) Append(record *LogRecord) (uint64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Assign LSN
	w.currentLSN++
	record.LSN = w.currentLSN

	// Serialize record
	data := w.serializeRecord(record)

	// Write to file (in production, would buffer and batch writes)
	if _, err := w.file.Write(data); err != nil {
		return 0, fmt.Errorf("failed to write WAL record: %w", err)
	}

	return record.LSN, nil
}

// serializeRecord converts a log record to bytes
// Format: [8-byte LSN][1-byte Type][8-byte TxnID][4-byte PageID][8-byte PrevLSN][4-byte DataLen][Data]
func (w *WAL) serializeRecord(record *LogRecord) []byte {
	dataLen := len(record.Data)
	buf := make([]byte, 33+dataLen)

	binary.LittleEndian.PutUint64(buf[0:8], record.LSN)
	buf[8] = byte(record.Type)
	binary.LittleEndian.PutUint64(buf[9:17], record.TxnID)
	binary.LittleEndian.PutUint32(buf[17:21], uint32(record.PageID))
	binary.LittleEndian.PutUint64(buf[21:29], record.PrevLSN)
	binary.LittleEndian.PutUint32(buf[29:33], uint32(dataLen))
	copy(buf[33:], record.Data)

	return buf
}

// deserializeRecord converts bytes to a log record
func (w *WAL) deserializeRecord(data []byte) (*LogRecord, error) {
	if len(data) < 33 {
		return nil, fmt.Errorf("invalid WAL record: too short")
	}

	record := &LogRecord{
		LSN:     binary.LittleEndian.Uint64(data[0:8]),
		Type:    LogRecordType(data[8]),
		TxnID:   binary.LittleEndian.Uint64(data[9:17]),
		PageID:  PageID(binary.LittleEndian.Uint32(data[17:21])),
		PrevLSN: binary.LittleEndian.Uint64(data[21:29]),
	}

	dataLen := binary.LittleEndian.Uint32(data[29:33])
	if len(data) < 33+int(dataLen) {
		return nil, fmt.Errorf("invalid WAL record: data truncated")
	}

	record.Data = make([]byte, dataLen)
	copy(record.Data, data[33:33+dataLen])

	return record, nil
}

// Flush ensures all buffered data is written to disk
func (w *WAL) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.file.Sync()
}

// Replay reads the WAL and returns all log records for recovery
func (w *WAL) Replay() ([]*LogRecord, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Seek to beginning
	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek WAL: %w", err)
	}

	records := make([]*LogRecord, 0)
	buf := make([]byte, 4096)

	for {
		// Read record header
		n, err := w.file.Read(buf[:33])
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read WAL record header: %w", err)
		}
		if n < 33 {
			break // Incomplete record at end
		}

		// Read data length
		dataLen := binary.LittleEndian.Uint32(buf[29:33])

		// Read full record
		fullRecord := make([]byte, 33+dataLen)
		copy(fullRecord[:33], buf[:33])

		if dataLen > 0 {
			if _, err := io.ReadFull(w.file, fullRecord[33:]); err != nil {
				return nil, fmt.Errorf("failed to read WAL record data: %w", err)
			}
		}

		// Deserialize
		record, err := w.deserializeRecord(fullRecord)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize WAL record: %w", err)
		}

		records = append(records, record)
	}

	// Seek back to end
	w.file.Seek(0, io.SeekEnd)

	return records, nil
}

// Checkpoint writes a checkpoint record
func (w *WAL) Checkpoint() error {
	record := &LogRecord{
		Type:  LogRecordCheckpoint,
		TxnID: 0,
		Data:  nil,
	}

	_, err := w.Append(record)
	if err != nil {
		return err
	}

	return w.Flush()
}

// Truncate removes WAL records before the given LSN
func (w *WAL) Truncate(beforeLSN uint64) error {
	// In production, would implement WAL archival and truncation
	// For educational purposes, we'll just note this is where it would happen
	return nil
}

// Close closes the WAL file
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.file.Sync(); err != nil {
		return err
	}

	return w.file.Close()
}
