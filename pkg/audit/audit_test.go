package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewAuditLogger(t *testing.T) {
	logger, err := NewAuditLogger(nil)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}

	if logger == nil {
		t.Fatal("Expected non-nil logger")
	}

	if !logger.IsEnabled() {
		t.Error("Expected logger to be enabled by default")
	}
}

func TestNewFileAuditLogger(t *testing.T) {
	tmpFile := "test_audit.log"
	defer os.Remove(tmpFile)

	logger, err := NewFileAuditLogger(tmpFile, nil)
	if err != nil {
		t.Fatalf("Failed to create file audit logger: %v", err)
	}
	defer logger.Close()

	if logger == nil {
		t.Fatal("Expected non-nil logger")
	}

	// Test logging to file
	event := &AuditEvent{
		Timestamp:  time.Now(),
		Operation:  OperationInsert,
		Collection: "test",
		Database:   "testdb",
		Success:    true,
		Severity:   SeverityInfo,
	}

	err = logger.Log(event)
	if err != nil {
		t.Fatalf("Failed to log event: %v", err)
	}

	// Close to flush
	logger.Close()

	// Verify file was written
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected log file to have content")
	}
}

func TestLogOperation(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:      true,
		OutputWriter: &buf,
		Format:       "json",
		MinSeverity:  SeverityInfo,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	details := map[string]interface{}{
		"field1": "value1",
		"field2": int64(123),
	}

	err = logger.LogOperation(
		OperationInsert,
		"users",
		"testdb",
		"admin",
		true,
		100*time.Millisecond,
		nil,
		details,
	)
	if err != nil {
		t.Fatalf("Failed to log operation: %v", err)
	}

	// Parse JSON
	var event AuditEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if event.Operation != OperationInsert {
		t.Errorf("Expected operation %s, got %s", OperationInsert, event.Operation)
	}

	if event.Collection != "users" {
		t.Errorf("Expected collection users, got %s", event.Collection)
	}

	if event.Database != "testdb" {
		t.Errorf("Expected database testdb, got %s", event.Database)
	}

	if event.User != "admin" {
		t.Errorf("Expected user admin, got %s", event.User)
	}

	if !event.Success {
		t.Error("Expected success to be true")
	}

	if event.Severity != SeverityInfo {
		t.Errorf("Expected severity %s, got %s", SeverityInfo, event.Severity)
	}
}

func TestLogInsert(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:      true,
		OutputWriter: &buf,
		Format:       "json",
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	err = logger.LogInsert("users", "testdb", "admin", true, 5, 50*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("Failed to log insert: %v", err)
	}

	var event AuditEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if event.DocumentCount != 5 {
		t.Errorf("Expected document count 5, got %d", event.DocumentCount)
	}

	if event.Duration != 50*time.Millisecond {
		t.Errorf("Expected duration 50ms, got %v", event.Duration)
	}
}

func TestLogUpdate(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:          true,
		OutputWriter:     &buf,
		Format:           "json",
		IncludeQueryData: true,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	filter := map[string]interface{}{"age": map[string]interface{}{"$gt": int64(18)}}
	update := map[string]interface{}{"$set": map[string]interface{}{"status": "adult"}}

	err = logger.LogUpdate("users", "testdb", "admin", true, 10, 100*time.Millisecond, filter, update, nil)
	if err != nil {
		t.Fatalf("Failed to log update: %v", err)
	}

	var event AuditEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if event.ModifiedCount != 10 {
		t.Errorf("Expected modified count 10, got %d", event.ModifiedCount)
	}

	if event.QueryFilter == nil {
		t.Error("Expected query filter to be included")
	}

	if event.UpdateSpec == nil {
		t.Error("Expected update spec to be included")
	}
}

func TestLogDelete(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:          true,
		OutputWriter:     &buf,
		Format:           "json",
		IncludeQueryData: true,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	filter := map[string]interface{}{"status": "inactive"}

	err = logger.LogDelete("users", "testdb", "admin", true, 3, 75*time.Millisecond, filter, nil)
	if err != nil {
		t.Fatalf("Failed to log delete: %v", err)
	}

	var event AuditEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if event.DeletedCount != 3 {
		t.Errorf("Expected deleted count 3, got %d", event.DeletedCount)
	}

	if event.QueryFilter == nil {
		t.Error("Expected query filter to be included")
	}
}

func TestLogFind(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:          true,
		OutputWriter:     &buf,
		Format:           "json",
		IncludeQueryData: true,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	filter := map[string]interface{}{"age": map[string]interface{}{"$gte": int64(21)}}

	err = logger.LogFind("users", "testdb", "reader", true, 25, 30*time.Millisecond, filter, nil)
	if err != nil {
		t.Fatalf("Failed to log find: %v", err)
	}

	var event AuditEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if event.DocumentCount != 25 {
		t.Errorf("Expected document count 25, got %d", event.DocumentCount)
	}

	if event.QueryFilter == nil {
		t.Error("Expected query filter to be included")
	}
}

func TestLogIndexOperation(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:      true,
		OutputWriter: &buf,
		Format:       "json",
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	err = logger.LogIndexOperation(OperationCreateIndex, "users", "testdb", "admin", "age_1", true, 200*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("Failed to log index operation: %v", err)
	}

	var event AuditEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if event.IndexName != "age_1" {
		t.Errorf("Expected index name age_1, got %s", event.IndexName)
	}

	if event.Operation != OperationCreateIndex {
		t.Errorf("Expected operation %s, got %s", OperationCreateIndex, event.Operation)
	}
}

func TestSeverityFiltering(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:      true,
		OutputWriter: &buf,
		Format:       "json",
		MinSeverity:  SeverityError,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Log info event (should be filtered)
	event1 := &AuditEvent{
		Timestamp:  time.Now(),
		Operation:  OperationFind,
		Collection: "test",
		Success:    true,
		Severity:   SeverityInfo,
	}
	logger.Log(event1)

	// Buffer should be empty
	if buf.Len() > 0 {
		t.Error("Expected info event to be filtered out")
	}

	// Log error event (should be logged)
	event2 := &AuditEvent{
		Timestamp:  time.Now(),
		Operation:  OperationInsert,
		Collection: "test",
		Success:    false,
		Severity:   SeverityError,
	}
	logger.Log(event2)

	// Buffer should have content
	if buf.Len() == 0 {
		t.Error("Expected error event to be logged")
	}
}

func TestOperationFiltering(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:      true,
		OutputWriter: &buf,
		Format:       "json",
		Operations:   []OperationType{OperationInsert, OperationUpdate, OperationDelete},
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Log find event (should be filtered)
	event1 := &AuditEvent{
		Timestamp:  time.Now(),
		Operation:  OperationFind,
		Collection: "test",
		Success:    true,
		Severity:   SeverityInfo,
	}
	logger.Log(event1)

	if buf.Len() > 0 {
		t.Error("Expected find operation to be filtered out")
	}

	// Log insert event (should be logged)
	event2 := &AuditEvent{
		Timestamp:  time.Now(),
		Operation:  OperationInsert,
		Collection: "test",
		Success:    true,
		Severity:   SeverityInfo,
	}
	logger.Log(event2)

	if buf.Len() == 0 {
		t.Error("Expected insert operation to be logged")
	}
}

func TestFieldTruncation(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:          true,
		OutputWriter:     &buf,
		Format:           "json",
		IncludeQueryData: true,
		MaxFieldSize:     50, // Small size to trigger truncation
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create a large filter
	largeFilter := map[string]interface{}{
		"field1": "this is a very long string that should be truncated",
		"field2": "another long string to ensure we exceed the limit",
		"field3": map[string]interface{}{
			"nested": "more data to make it even larger",
		},
	}

	err = logger.LogFind("users", "testdb", "admin", true, 10, 50*time.Millisecond, largeFilter, nil)
	if err != nil {
		t.Fatalf("Failed to log find: %v", err)
	}

	// Check that output was truncated
	output := buf.String()
	if !strings.Contains(output, "truncated") {
		t.Error("Expected large field to be truncated")
	}
}

func TestDisabledLogger(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:      false,
		OutputWriter: &buf,
		Format:       "json",
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	event := &AuditEvent{
		Timestamp:  time.Now(),
		Operation:  OperationInsert,
		Collection: "test",
		Success:    true,
		Severity:   SeverityInfo,
	}

	logger.Log(event)

	if buf.Len() > 0 {
		t.Error("Expected no output when logger is disabled")
	}
}

func TestSetEnabled(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:      true,
		OutputWriter: &buf,
		Format:       "json",
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Disable logger
	logger.SetEnabled(false)
	if logger.IsEnabled() {
		t.Error("Expected logger to be disabled")
	}

	// Try to log (should not write)
	event := &AuditEvent{
		Timestamp:  time.Now(),
		Operation:  OperationInsert,
		Collection: "test",
		Success:    true,
		Severity:   SeverityInfo,
	}
	logger.Log(event)

	if buf.Len() > 0 {
		t.Error("Expected no output when logger is disabled")
	}

	// Re-enable logger
	logger.SetEnabled(true)
	if !logger.IsEnabled() {
		t.Error("Expected logger to be enabled")
	}

	// Log again (should work)
	logger.Log(event)

	if buf.Len() == 0 {
		t.Error("Expected output when logger is re-enabled")
	}
}

func TestTextFormat(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:      true,
		OutputWriter: &buf,
		Format:       "text",
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	err = logger.LogInsert("users", "testdb", "admin", true, 5, 50*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("Failed to log insert: %v", err)
	}

	output := buf.String()

	// Check for expected text format elements
	if !strings.Contains(output, "insert") {
		t.Error("Expected output to contain 'insert'")
	}

	if !strings.Contains(output, "users") {
		t.Error("Expected output to contain 'users'")
	}

	if !strings.Contains(output, "SUCCESS") {
		t.Error("Expected output to contain 'SUCCESS'")
	}

	if !strings.Contains(output, "5 documents") {
		t.Error("Expected output to contain document count")
	}
}

func TestErrorLogging(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:      true,
		OutputWriter: &buf,
		Format:       "json",
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	testError := fmt.Errorf("test error message")

	err = logger.LogInsert("users", "testdb", "admin", false, 0, 10*time.Millisecond, testError)
	if err != nil {
		t.Fatalf("Failed to log insert: %v", err)
	}

	var event AuditEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if event.Success {
		t.Error("Expected success to be false")
	}

	if event.Severity != SeverityError {
		t.Errorf("Expected severity %s, got %s", SeverityError, event.Severity)
	}

	if event.ErrorMessage != "test error message" {
		t.Errorf("Expected error message, got: %s", event.ErrorMessage)
	}
}

func TestGlobalLogger(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:      true,
		OutputWriter: &buf,
		Format:       "json",
	}

	err := InitGlobalLogger(config)
	if err != nil {
		t.Fatalf("Failed to initialize global logger: %v", err)
	}

	if GlobalAuditLogger == nil {
		t.Fatal("Expected global logger to be initialized")
	}

	err = GlobalAuditLogger.LogInsert("test", "testdb", "admin", true, 1, 10*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("Failed to log with global logger: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("Expected global logger to write output")
	}
}

func TestConcurrentLogging(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:      true,
		OutputWriter: &buf,
		Format:       "json",
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Log from multiple goroutines
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			event := &AuditEvent{
				Timestamp:  time.Now(),
				Operation:  OperationInsert,
				Collection: fmt.Sprintf("test%d", id),
				Success:    true,
				Severity:   SeverityInfo,
			}
			logger.Log(event)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have 10 lines of output
	lines := strings.Count(buf.String(), "\n")
	if lines != 10 {
		t.Errorf("Expected 10 log lines, got %d", lines)
	}
}

// TestInitGlobalFileLogger tests initializing global logger with file output
func TestInitGlobalFileLogger(t *testing.T) {
	tmpFile := "test_global_audit.log"
	defer os.Remove(tmpFile)

	config := &Config{
		Enabled: true,
		Format:  "json",
	}

	err := InitGlobalFileLogger(tmpFile, config)
	if err != nil {
		t.Fatalf("Failed to initialize global file logger: %v", err)
	}
	defer GlobalAuditLogger.Close()

	if GlobalAuditLogger == nil {
		t.Fatal("Expected global logger to be initialized")
	}

	// Test logging
	err = GlobalAuditLogger.LogInsert("test", "testdb", "admin", true, 1, 10*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("Failed to log with global file logger: %v", err)
	}

	// Close to flush
	GlobalAuditLogger.Close()

	// Verify file exists and has content
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected log file to have content")
	}
}

// TestInitGlobalFileLoggerInvalidPath tests error handling for invalid file path
func TestInitGlobalFileLoggerInvalidPath(t *testing.T) {
	err := InitGlobalFileLogger("/invalid/path/that/does/not/exist/audit.log", nil)
	if err == nil {
		t.Error("Expected error for invalid file path")
	}
}

// TestCloseWithoutFile tests Close when no file is open
func TestCloseWithoutFile(t *testing.T) {
	logger, err := NewAuditLogger(nil)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Close should not error when no file is open
	err = logger.Close()
	if err != nil {
		t.Errorf("Close should not error when no file is open: %v", err)
	}
}

// TestLogOperationWithError tests LogOperation with error
func TestLogOperationWithError(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:      true,
		OutputWriter: &buf,
		Format:       "json",
		MinSeverity:  SeverityInfo,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	testError := fmt.Errorf("operation failed")
	details := map[string]interface{}{
		"reason": "test failure",
	}

	err = logger.LogOperation(
		OperationInsert,
		"users",
		"testdb",
		"admin",
		false,
		100*time.Millisecond,
		testError,
		details,
	)
	if err != nil {
		t.Fatalf("Failed to log operation: %v", err)
	}

	var event AuditEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if event.Success {
		t.Error("Expected success to be false")
	}

	if event.Severity != SeverityError {
		t.Errorf("Expected severity %s, got %s", SeverityError, event.Severity)
	}

	if event.ErrorMessage != "operation failed" {
		t.Errorf("Expected error message 'operation failed', got: %s", event.ErrorMessage)
	}
}

// TestLogIndexOperationWithError tests LogIndexOperation with error
func TestLogIndexOperationWithError(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:      true,
		OutputWriter: &buf,
		Format:       "json",
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	testError := fmt.Errorf("index creation failed")

	err = logger.LogIndexOperation(
		OperationCreateIndex,
		"users",
		"testdb",
		"admin",
		"email_1",
		false,
		150*time.Millisecond,
		testError,
	)
	if err != nil {
		t.Fatalf("Failed to log index operation: %v", err)
	}

	var event AuditEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if event.Success {
		t.Error("Expected success to be false")
	}

	if event.ErrorMessage != "index creation failed" {
		t.Errorf("Expected error message, got: %s", event.ErrorMessage)
	}
}

// TestTextFormatWithAllFields tests text format with all possible fields
func TestTextFormatWithAllFields(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:      true,
		OutputWriter: &buf,
		Format:       "text",
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test with user field
	err = logger.LogInsert("users", "testdb", "admin", true, 5, 50*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("Failed to log insert: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "by user admin") {
		t.Error("Expected output to contain user")
	}
	if !strings.Contains(output, "took") {
		t.Error("Expected output to contain duration")
	}

	// Test with modified count
	buf.Reset()
	filter := map[string]interface{}{"age": int64(30)}
	update := map[string]interface{}{"$set": map[string]interface{}{"status": "active"}}
	err = logger.LogUpdate("users", "testdb", "admin", true, 10, 100*time.Millisecond, filter, update, nil)
	if err != nil {
		t.Fatalf("Failed to log update: %v", err)
	}

	output = buf.String()
	if !strings.Contains(output, "10 modified") {
		t.Error("Expected output to contain modified count")
	}

	// Test with deleted count
	buf.Reset()
	err = logger.LogDelete("users", "testdb", "admin", true, 3, 75*time.Millisecond, filter, nil)
	if err != nil {
		t.Fatalf("Failed to log delete: %v", err)
	}

	output = buf.String()
	if !strings.Contains(output, "3 deleted") {
		t.Error("Expected output to contain deleted count")
	}

	// Test with index name
	buf.Reset()
	err = logger.LogIndexOperation(OperationCreateIndex, "users", "testdb", "admin", "age_1", true, 200*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("Failed to log index operation: %v", err)
	}

	output = buf.String()
	if !strings.Contains(output, "index: age_1") {
		t.Error("Expected output to contain index name")
	}

	// Test with error
	buf.Reset()
	testError := fmt.Errorf("something went wrong")
	err = logger.LogInsert("users", "testdb", "admin", false, 0, 10*time.Millisecond, testError)
	if err != nil {
		t.Fatalf("Failed to log insert: %v", err)
	}

	output = buf.String()
	if !strings.Contains(output, "FAILURE") {
		t.Error("Expected output to contain FAILURE")
	}
	if !strings.Contains(output, "error: something went wrong") {
		t.Error("Expected output to contain error message")
	}
}

// TestTruncateValueWithNonJSONValue tests truncateValue with values that can't be marshaled
func TestTruncateValueWithNonJSONValue(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:          true,
		OutputWriter:     &buf,
		Format:           "json",
		IncludeQueryData: true,
		MaxFieldSize:     50,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create a filter with a channel (can't be JSON marshaled)
	invalidFilter := make(chan int)

	// This will error at JSON marshaling level (which is expected)
	// The truncateValue function handles unmarshalable values gracefully,
	// but the overall event still needs to be JSON serializable
	err = logger.LogFind("users", "testdb", "admin", true, 10, 50*time.Millisecond, invalidFilter, nil)
	if err == nil {
		t.Error("Expected error when logging with unmarshalable filter")
	}
}

// TestLogUpdateWithoutQueryData tests LogUpdate with IncludeQueryData=false
func TestLogUpdateWithoutQueryData(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:          true,
		OutputWriter:     &buf,
		Format:           "json",
		IncludeQueryData: false,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	filter := map[string]interface{}{"age": int64(30)}
	update := map[string]interface{}{"$set": map[string]interface{}{"status": "active"}}

	err = logger.LogUpdate("users", "testdb", "admin", true, 5, 100*time.Millisecond, filter, update, nil)
	if err != nil {
		t.Fatalf("Failed to log update: %v", err)
	}

	var event AuditEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if event.QueryFilter != nil {
		t.Error("Expected query filter to be nil when IncludeQueryData=false")
	}

	if event.UpdateSpec != nil {
		t.Error("Expected update spec to be nil when IncludeQueryData=false")
	}
}

// TestLogDeleteWithoutQueryData tests LogDelete with IncludeQueryData=false
func TestLogDeleteWithoutQueryData(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:          true,
		OutputWriter:     &buf,
		Format:           "json",
		IncludeQueryData: false,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	filter := map[string]interface{}{"status": "inactive"}

	err = logger.LogDelete("users", "testdb", "admin", true, 3, 75*time.Millisecond, filter, nil)
	if err != nil {
		t.Fatalf("Failed to log delete: %v", err)
	}

	var event AuditEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if event.QueryFilter != nil {
		t.Error("Expected query filter to be nil when IncludeQueryData=false")
	}
}

// TestLogFindWithoutQueryData tests LogFind with IncludeQueryData=false
func TestLogFindWithoutQueryData(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:          true,
		OutputWriter:     &buf,
		Format:           "json",
		IncludeQueryData: false,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	filter := map[string]interface{}{"age": map[string]interface{}{"$gte": int64(21)}}

	err = logger.LogFind("users", "testdb", "reader", true, 25, 30*time.Millisecond, filter, nil)
	if err != nil {
		t.Fatalf("Failed to log find: %v", err)
	}

	var event AuditEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if event.QueryFilter != nil {
		t.Error("Expected query filter to be nil when IncludeQueryData=false")
	}
}

// TestSeverityWarning tests warning severity filtering
func TestSeverityWarning(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Enabled:      true,
		OutputWriter: &buf,
		Format:       "json",
		MinSeverity:  SeverityWarning,
	}

	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Log info event (should be filtered)
	event1 := &AuditEvent{
		Timestamp:  time.Now(),
		Operation:  OperationFind,
		Collection: "test",
		Success:    true,
		Severity:   SeverityInfo,
	}
	logger.Log(event1)

	if buf.Len() > 0 {
		t.Error("Expected info event to be filtered out when MinSeverity=Warning")
	}

	// Log warning event (should be logged)
	event2 := &AuditEvent{
		Timestamp:  time.Now(),
		Operation:  OperationUpdate,
		Collection: "test",
		Success:    true,
		Severity:   SeverityWarning,
	}
	logger.Log(event2)

	if buf.Len() == 0 {
		t.Error("Expected warning event to be logged")
	}

	// Reset buffer
	buf.Reset()

	// Log error event (should be logged)
	event3 := &AuditEvent{
		Timestamp:  time.Now(),
		Operation:  OperationInsert,
		Collection: "test",
		Success:    false,
		Severity:   SeverityError,
	}
	logger.Log(event3)

	if buf.Len() == 0 {
		t.Error("Expected error event to be logged when MinSeverity=Warning")
	}
}
