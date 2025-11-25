package audit

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// OperationType represents the type of database operation
type OperationType string

const (
	OperationInsert           OperationType = "insert"
	OperationInsertMany       OperationType = "insertMany"
	OperationUpdate           OperationType = "update"
	OperationUpdateMany       OperationType = "updateMany"
	OperationDelete           OperationType = "delete"
	OperationDeleteMany       OperationType = "deleteMany"
	OperationFind             OperationType = "find"
	OperationFindOne          OperationType = "findOne"
	OperationAggregate        OperationType = "aggregate"
	OperationCreateIndex      OperationType = "createIndex"
	OperationDropIndex        OperationType = "dropIndex"
	OperationCreateCollection OperationType = "createCollection"
	OperationDropCollection   OperationType = "dropCollection"
	OperationTextSearch       OperationType = "textSearch"
	OperationCount            OperationType = "count"
)

// Severity represents the severity level of an audit event
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// AuditEvent represents a single audit log entry
type AuditEvent struct {
	Timestamp      time.Time              `json:"timestamp"`
	Operation      OperationType          `json:"operation"`
	Collection     string                 `json:"collection,omitempty"`
	Database       string                 `json:"database,omitempty"`
	User           string                 `json:"user,omitempty"`
	RemoteAddr     string                 `json:"remoteAddr,omitempty"`
	Success        bool                   `json:"success"`
	ErrorMessage   string                 `json:"errorMessage,omitempty"`
	Duration       time.Duration          `json:"duration,omitempty"`
	Severity       Severity               `json:"severity"`
	Details        map[string]interface{} `json:"details,omitempty"`
	DocumentCount  int                    `json:"documentCount,omitempty"`
	ModifiedCount  int                    `json:"modifiedCount,omitempty"`
	DeletedCount   int                    `json:"deletedCount,omitempty"`
	IndexName      string                 `json:"indexName,omitempty"`
	QueryFilter    interface{}            `json:"queryFilter,omitempty"`
	UpdateSpec     interface{}            `json:"updateSpec,omitempty"`
}

// Config holds audit logging configuration
type Config struct {
	Enabled          bool              // Enable/disable audit logging
	OutputWriter     io.Writer         // Output destination (file, stdout, etc.)
	Format           string            // "json" or "text"
	MinSeverity      Severity          // Minimum severity to log
	IncludeQueryData bool              // Include full query/update data
	MaxFieldSize     int               // Max size for query/update fields (0 = unlimited)
	Operations       []OperationType   // Operations to audit (empty = all)
}

// DefaultConfig returns a default audit configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:          true,
		OutputWriter:     os.Stdout,
		Format:           "json",
		MinSeverity:      SeverityInfo,
		IncludeQueryData: true,
		MaxFieldSize:     1000, // 1KB limit for query data
		Operations:       nil,  // Log all operations
	}
}

// AuditLogger handles audit logging
type AuditLogger struct {
	config *Config
	mu     sync.RWMutex
	file   *os.File // If logging to file
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(config *Config) (*AuditLogger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	return &AuditLogger{
		config: config,
	}, nil
}

// NewFileAuditLogger creates an audit logger that writes to a file
func NewFileAuditLogger(filePath string, config *Config) (*AuditLogger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Open file for appending
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}

	config.OutputWriter = file

	return &AuditLogger{
		config: config,
		file:   file,
	}, nil
}

// Log logs an audit event
func (l *AuditLogger) Log(event *AuditEvent) error {
	if !l.config.Enabled {
		return nil
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	// Check severity filter
	if !l.shouldLog(event.Severity) {
		return nil
	}

	// Check operation filter
	if !l.shouldLogOperation(event.Operation) {
		return nil
	}

	// Truncate large fields if needed
	if l.config.MaxFieldSize > 0 {
		l.truncateFields(event)
	}

	// Format and write
	var output []byte
	var err error

	if l.config.Format == "json" {
		output, err = json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal audit event: %w", err)
		}
		output = append(output, '\n')
	} else {
		output = []byte(l.formatText(event))
	}

	_, err = l.config.OutputWriter.Write(output)
	return err
}

// LogOperation logs a database operation
func (l *AuditLogger) LogOperation(op OperationType, collection, database, user string, success bool, duration time.Duration, err error, details map[string]interface{}) error {
	severity := SeverityInfo
	if !success {
		severity = SeverityError
	}

	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	event := &AuditEvent{
		Timestamp:    time.Now(),
		Operation:    op,
		Collection:   collection,
		Database:     database,
		User:         user,
		Success:      success,
		ErrorMessage: errorMsg,
		Duration:     duration,
		Severity:     severity,
		Details:      details,
	}

	return l.Log(event)
}

// LogInsert logs an insert operation
func (l *AuditLogger) LogInsert(collection, database, user string, success bool, documentCount int, duration time.Duration, err error) error {
	event := &AuditEvent{
		Timestamp:     time.Now(),
		Operation:     OperationInsert,
		Collection:    collection,
		Database:      database,
		User:          user,
		Success:       success,
		Duration:      duration,
		Severity:      l.getSeverity(success),
		DocumentCount: documentCount,
	}

	if err != nil {
		event.ErrorMessage = err.Error()
	}

	return l.Log(event)
}

// LogUpdate logs an update operation
func (l *AuditLogger) LogUpdate(collection, database, user string, success bool, modifiedCount int, duration time.Duration, filter, update interface{}, err error) error {
	event := &AuditEvent{
		Timestamp:     time.Now(),
		Operation:     OperationUpdate,
		Collection:    collection,
		Database:      database,
		User:          user,
		Success:       success,
		Duration:      duration,
		Severity:      l.getSeverity(success),
		ModifiedCount: modifiedCount,
	}

	if l.config.IncludeQueryData {
		event.QueryFilter = filter
		event.UpdateSpec = update
	}

	if err != nil {
		event.ErrorMessage = err.Error()
	}

	return l.Log(event)
}

// LogDelete logs a delete operation
func (l *AuditLogger) LogDelete(collection, database, user string, success bool, deletedCount int, duration time.Duration, filter interface{}, err error) error {
	event := &AuditEvent{
		Timestamp:    time.Now(),
		Operation:    OperationDelete,
		Collection:   collection,
		Database:     database,
		User:         user,
		Success:      success,
		Duration:     duration,
		Severity:     l.getSeverity(success),
		DeletedCount: deletedCount,
	}

	if l.config.IncludeQueryData {
		event.QueryFilter = filter
	}

	if err != nil {
		event.ErrorMessage = err.Error()
	}

	return l.Log(event)
}

// LogFind logs a find/query operation
func (l *AuditLogger) LogFind(collection, database, user string, success bool, documentCount int, duration time.Duration, filter interface{}, err error) error {
	event := &AuditEvent{
		Timestamp:     time.Now(),
		Operation:     OperationFind,
		Collection:    collection,
		Database:      database,
		User:          user,
		Success:       success,
		Duration:      duration,
		Severity:      l.getSeverity(success),
		DocumentCount: documentCount,
	}

	if l.config.IncludeQueryData {
		event.QueryFilter = filter
	}

	if err != nil {
		event.ErrorMessage = err.Error()
	}

	return l.Log(event)
}

// LogIndexOperation logs an index operation (create/drop)
func (l *AuditLogger) LogIndexOperation(op OperationType, collection, database, user, indexName string, success bool, duration time.Duration, err error) error {
	event := &AuditEvent{
		Timestamp:  time.Now(),
		Operation:  op,
		Collection: collection,
		Database:   database,
		User:       user,
		IndexName:  indexName,
		Success:    success,
		Duration:   duration,
		Severity:   l.getSeverity(success),
	}

	if err != nil {
		event.ErrorMessage = err.Error()
	}

	return l.Log(event)
}

// Close closes the audit logger and any open files
func (l *AuditLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// SetEnabled enables or disables audit logging at runtime
func (l *AuditLogger) SetEnabled(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Enabled = enabled
}

// IsEnabled returns whether audit logging is enabled
func (l *AuditLogger) IsEnabled() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.config.Enabled
}

// shouldLog determines if an event should be logged based on severity
func (l *AuditLogger) shouldLog(severity Severity) bool {
	severityLevels := map[Severity]int{
		SeverityInfo:    1,
		SeverityWarning: 2,
		SeverityError:   3,
	}

	return severityLevels[severity] >= severityLevels[l.config.MinSeverity]
}

// shouldLogOperation determines if an operation should be logged
func (l *AuditLogger) shouldLogOperation(op OperationType) bool {
	if len(l.config.Operations) == 0 {
		return true // Log all operations
	}

	for _, allowedOp := range l.config.Operations {
		if op == allowedOp {
			return true
		}
	}
	return false
}

// truncateFields truncates large fields to the configured max size
func (l *AuditLogger) truncateFields(event *AuditEvent) {
	if event.QueryFilter != nil {
		event.QueryFilter = l.truncateValue(event.QueryFilter)
	}
	if event.UpdateSpec != nil {
		event.UpdateSpec = l.truncateValue(event.UpdateSpec)
	}
}

// truncateValue truncates a value if it exceeds max field size
func (l *AuditLogger) truncateValue(value interface{}) interface{} {
	data, err := json.Marshal(value)
	if err != nil {
		return value
	}

	if len(data) > l.config.MaxFieldSize {
		truncated := string(data[:l.config.MaxFieldSize])
		return truncated + "... (truncated)"
	}

	return value
}

// getSeverity determines severity based on success
func (l *AuditLogger) getSeverity(success bool) Severity {
	if success {
		return SeverityInfo
	}
	return SeverityError
}

// formatText formats an event as human-readable text
func (l *AuditLogger) formatText(event *AuditEvent) string {
	status := "SUCCESS"
	if !event.Success {
		status = "FAILURE"
	}

	msg := fmt.Sprintf("[%s] [%s] [%s] %s operation on %s.%s",
		event.Timestamp.Format(time.RFC3339),
		event.Severity,
		status,
		event.Operation,
		event.Database,
		event.Collection,
	)

	if event.User != "" {
		msg += fmt.Sprintf(" by user %s", event.User)
	}

	if event.Duration > 0 {
		msg += fmt.Sprintf(" (took %v)", event.Duration)
	}

	if event.DocumentCount > 0 {
		msg += fmt.Sprintf(" - %d documents", event.DocumentCount)
	}

	if event.ModifiedCount > 0 {
		msg += fmt.Sprintf(" - %d modified", event.ModifiedCount)
	}

	if event.DeletedCount > 0 {
		msg += fmt.Sprintf(" - %d deleted", event.DeletedCount)
	}

	if event.IndexName != "" {
		msg += fmt.Sprintf(" - index: %s", event.IndexName)
	}

	if event.ErrorMessage != "" {
		msg += fmt.Sprintf(" - error: %s", event.ErrorMessage)
	}

	msg += "\n"
	return msg
}

// GlobalAuditLogger is a global audit logger instance
var GlobalAuditLogger *AuditLogger

// InitGlobalLogger initializes the global audit logger
func InitGlobalLogger(config *Config) error {
	logger, err := NewAuditLogger(config)
	if err != nil {
		return err
	}
	GlobalAuditLogger = logger
	return nil
}

// InitGlobalFileLogger initializes the global audit logger with file output
func InitGlobalFileLogger(filePath string, config *Config) error {
	logger, err := NewFileAuditLogger(filePath, config)
	if err != nil {
		return err
	}
	GlobalAuditLogger = logger
	return nil
}
