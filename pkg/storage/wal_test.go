package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewWAL(t *testing.T) {
	dir := "./test_wal_new"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.wal")
	wal, err := NewWAL(path)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	if wal == nil {
		t.Fatal("Expected non-nil WAL")
	}
	if wal.currentLSN != 0 {
		t.Errorf("Expected currentLSN 0, got %d", wal.currentLSN)
	}
}

func TestWALAppend(t *testing.T) {
	dir := "./test_wal_append"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.wal")
	wal, err := NewWAL(path)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Append a record
	record := &LogRecord{
		Type:   LogRecordInsert,
		TxnID:  1,
		PageID: 5,
		Data:   []byte("test data"),
	}

	lsn, err := wal.Append(record)
	if err != nil {
		t.Fatalf("Failed to append record: %v", err)
	}

	if lsn == 0 {
		t.Error("Expected non-zero LSN")
	}
	if record.LSN != lsn {
		t.Errorf("Expected record LSN %d, got %d", lsn, record.LSN)
	}
}

func TestWALMultipleAppends(t *testing.T) {
	dir := "./test_wal_multiple"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.wal")
	wal, err := NewWAL(path)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Append multiple records
	lsns := make([]uint64, 0)
	for i := 0; i < 5; i++ {
		record := &LogRecord{
			Type:   LogRecordInsert,
			TxnID:  uint64(i + 1),
			PageID: PageID(i),
			Data:   []byte("test"),
		}

		lsn, err := wal.Append(record)
		if err != nil {
			t.Fatalf("Failed to append record %d: %v", i, err)
		}
		lsns = append(lsns, lsn)
	}

	// Verify LSNs are increasing
	for i := 1; i < len(lsns); i++ {
		if lsns[i] <= lsns[i-1] {
			t.Errorf("Expected LSN %d > %d", lsns[i], lsns[i-1])
		}
	}
}

func TestWALFlush(t *testing.T) {
	dir := "./test_wal_flush"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.wal")
	wal, err := NewWAL(path)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Append a record
	record := &LogRecord{
		Type:   LogRecordUpdate,
		TxnID:  10,
		PageID: 3,
		Data:   []byte("flush test"),
	}
	_, err = wal.Append(record)
	if err != nil {
		t.Fatalf("Failed to append record: %v", err)
	}

	// Flush
	err = wal.Flush()
	if err != nil {
		t.Fatalf("Failed to flush: %v", err)
	}
}

func TestWALReplay(t *testing.T) {
	dir := "./test_wal_replay"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.wal")

	// Create WAL and write records
	wal, err := NewWAL(path)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	// Append various record types
	records := []*LogRecord{
		{Type: LogRecordInsert, TxnID: 1, PageID: 0, Data: []byte("insert")},
		{Type: LogRecordUpdate, TxnID: 2, PageID: 1, Data: []byte("update")},
		{Type: LogRecordDelete, TxnID: 3, PageID: 2, Data: []byte("delete")},
		{Type: LogRecordCommit, TxnID: 1, PageID: 0, Data: nil},
		{Type: LogRecordAbort, TxnID: 3, PageID: 0, Data: nil},
	}

	for _, record := range records {
		_, err := wal.Append(record)
		if err != nil {
			t.Fatalf("Failed to append record: %v", err)
		}
	}

	wal.Flush()
	wal.Close()

	// Reopen and replay
	wal2, err := NewWAL(path)
	if err != nil {
		t.Fatalf("Failed to reopen WAL: %v", err)
	}
	defer wal2.Close()

	replayed, err := wal2.Replay()
	if err != nil {
		t.Fatalf("Failed to replay WAL: %v", err)
	}

	if len(replayed) != len(records) {
		t.Errorf("Expected %d records, got %d", len(records), len(replayed))
	}

	// Verify record types
	for i, record := range replayed {
		if record.Type != records[i].Type {
			t.Errorf("Record %d: expected type %d, got %d", i, records[i].Type, record.Type)
		}
		if record.TxnID != records[i].TxnID {
			t.Errorf("Record %d: expected TxnID %d, got %d", i, records[i].TxnID, record.TxnID)
		}
		if record.PageID != records[i].PageID {
			t.Errorf("Record %d: expected PageID %d, got %d", i, records[i].PageID, record.PageID)
		}
		if string(record.Data) != string(records[i].Data) {
			t.Errorf("Record %d: expected data %s, got %s", i, records[i].Data, record.Data)
		}
	}
}

func TestWALReplayEmpty(t *testing.T) {
	dir := "./test_wal_replay_empty"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.wal")
	wal, err := NewWAL(path)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Replay empty WAL
	records, err := wal.Replay()
	if err != nil {
		t.Fatalf("Failed to replay empty WAL: %v", err)
	}

	if len(records) != 0 {
		t.Errorf("Expected 0 records, got %d", len(records))
	}
}

func TestWALCheckpoint(t *testing.T) {
	dir := "./test_wal_checkpoint"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.wal")
	wal, err := NewWAL(path)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Append some records
	record := &LogRecord{
		Type:   LogRecordInsert,
		TxnID:  1,
		PageID: 0,
		Data:   []byte("before checkpoint"),
	}
	_, err = wal.Append(record)
	if err != nil {
		t.Fatalf("Failed to append record: %v", err)
	}

	// Create checkpoint
	err = wal.Checkpoint()
	if err != nil {
		t.Fatalf("Failed to checkpoint: %v", err)
	}

	// Append after checkpoint
	record2 := &LogRecord{
		Type:   LogRecordInsert,
		TxnID:  2,
		PageID: 1,
		Data:   []byte("after checkpoint"),
	}
	_, err = wal.Append(record2)
	if err != nil {
		t.Fatalf("Failed to append record after checkpoint: %v", err)
	}
}

func TestWALTruncate(t *testing.T) {
	dir := "./test_wal_truncate"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.wal")
	wal, err := NewWAL(path)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Append records
	lsn1, _ := wal.Append(&LogRecord{Type: LogRecordInsert, TxnID: 1, Data: []byte("old")})
	lsn2, _ := wal.Append(&LogRecord{Type: LogRecordInsert, TxnID: 2, Data: []byte("new")})

	// Truncate (currently a stub, but test it doesn't error)
	err = wal.Truncate(lsn1)
	if err != nil {
		t.Fatalf("Failed to truncate: %v", err)
	}

	// Verify truncate is safe to call with various LSNs
	err = wal.Truncate(lsn2)
	if err != nil {
		t.Fatalf("Failed to truncate at lsn2: %v", err)
	}

	err = wal.Truncate(0)
	if err != nil {
		t.Fatalf("Failed to truncate at 0: %v", err)
	}
}

func TestWALClose(t *testing.T) {
	dir := "./test_wal_close"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.wal")
	wal, err := NewWAL(path)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	// Append a record
	record := &LogRecord{
		Type:   LogRecordInsert,
		TxnID:  1,
		PageID: 0,
		Data:   []byte("test"),
	}
	_, err = wal.Append(record)
	if err != nil {
		t.Fatalf("Failed to append record: %v", err)
	}

	// Close
	err = wal.Close()
	if err != nil {
		t.Fatalf("Failed to close WAL: %v", err)
	}

	// Second close should fail
	err = wal.Close()
	if err == nil {
		t.Error("Expected error on second close")
	}
}

func TestWALSerializeDeserialize(t *testing.T) {
	dir := "./test_wal_serde"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.wal")
	wal, err := NewWAL(path)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Test record with various fields
	original := &LogRecord{
		LSN:     100,
		Type:    LogRecordUpdate,
		TxnID:   42,
		PageID:  7,
		PrevLSN: 99,
		Data:    []byte("serialization test data"),
	}

	// Serialize
	data := wal.serializeRecord(original)

	// Deserialize
	deserialized, err := wal.deserializeRecord(data)
	if err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	// Verify fields
	if deserialized.LSN != original.LSN {
		t.Errorf("LSN mismatch: expected %d, got %d", original.LSN, deserialized.LSN)
	}
	if deserialized.Type != original.Type {
		t.Errorf("Type mismatch: expected %d, got %d", original.Type, deserialized.Type)
	}
	if deserialized.TxnID != original.TxnID {
		t.Errorf("TxnID mismatch: expected %d, got %d", original.TxnID, deserialized.TxnID)
	}
	if deserialized.PageID != original.PageID {
		t.Errorf("PageID mismatch: expected %d, got %d", original.PageID, deserialized.PageID)
	}
	if deserialized.PrevLSN != original.PrevLSN {
		t.Errorf("PrevLSN mismatch: expected %d, got %d", original.PrevLSN, deserialized.PrevLSN)
	}
	if string(deserialized.Data) != string(original.Data) {
		t.Errorf("Data mismatch: expected %s, got %s", original.Data, deserialized.Data)
	}
}

func TestWALDeserializeErrors(t *testing.T) {
	dir := "./test_wal_deserialize_errors"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.wal")
	wal, err := NewWAL(path)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Test with too short data
	shortData := make([]byte, 10)
	_, err = wal.deserializeRecord(shortData)
	if err == nil {
		t.Error("Expected error with too short data")
	}

	// Test with truncated data field
	truncatedData := make([]byte, 33)
	// Set data length to 100 but don't provide the data
	truncatedData[29] = 100
	_, err = wal.deserializeRecord(truncatedData)
	if err == nil {
		t.Error("Expected error with truncated data")
	}
}

func TestWALRecordWithNoData(t *testing.T) {
	dir := "./test_wal_no_data"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.wal")
	wal, err := NewWAL(path)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Record with nil data
	record := &LogRecord{
		Type:   LogRecordCommit,
		TxnID:  1,
		PageID: 0,
		Data:   nil,
	}

	lsn, err := wal.Append(record)
	if err != nil {
		t.Fatalf("Failed to append record with nil data: %v", err)
	}
	if lsn == 0 {
		t.Error("Expected non-zero LSN")
	}

	// Replay and verify
	records, err := wal.Replay()
	if err != nil {
		t.Fatalf("Failed to replay: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}
	if len(records[0].Data) != 0 {
		t.Errorf("Expected empty data, got %d bytes", len(records[0].Data))
	}
}

func TestWALRecordTypes(t *testing.T) {
	dir := "./test_wal_record_types"
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "test.wal")
	wal, err := NewWAL(path)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Test all record types
	recordTypes := []LogRecordType{
		LogRecordInsert,
		LogRecordUpdate,
		LogRecordDelete,
		LogRecordCheckpoint,
		LogRecordCommit,
		LogRecordAbort,
	}

	for _, recordType := range recordTypes {
		record := &LogRecord{
			Type:   recordType,
			TxnID:  1,
			PageID: 0,
			Data:   []byte("test"),
		}

		_, err := wal.Append(record)
		if err != nil {
			t.Fatalf("Failed to append %v record: %v", recordType, err)
		}
	}

	// Replay and verify all types
	records, err := wal.Replay()
	if err != nil {
		t.Fatalf("Failed to replay: %v", err)
	}

	if len(records) != len(recordTypes) {
		t.Errorf("Expected %d records, got %d", len(recordTypes), len(records))
	}

	for i, record := range records {
		if record.Type != recordTypes[i] {
			t.Errorf("Record %d: expected type %v, got %v", i, recordTypes[i], record.Type)
		}
	}
}

// TestNewWALWithInvalidPath tests NewWAL with invalid path
func TestNewWALWithInvalidPath(t *testing.T) {
	// Try to create WAL in a non-existent directory
	_, err := NewWAL("/non/existent/directory/wal.log")
	if err == nil {
		t.Error("Expected error when creating WAL with invalid path")
	}
}

// TestWALFlushError tests error handling during flush
func TestWALFlushError(t *testing.T) {
	walPath := t.TempDir() + "/test.wal"
	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	// Append a record
	record := &LogRecord{
		Type:   LogRecordInsert,
		PageID: 1,
		Data:   []byte("test data"),
	}
	_, err = wal.Append(record)
	if err != nil {
		t.Fatalf("Failed to append record: %v", err)
	}

	// Close the file to simulate error during flush
	wal.file.Close()

	// Try to flush - should handle error gracefully
	err = wal.Flush()
	if err == nil {
		t.Error("Expected error when flushing closed WAL")
	}
}
