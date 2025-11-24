package metrics

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// SlowQueryLog tracks and logs queries that exceed a threshold duration
type SlowQueryLog struct {
	threshold      time.Duration
	maxEntries     int
	logFile        *os.File
	entries        []SlowQueryEntry
	mu             sync.RWMutex
	enabled        bool
	logToFile      bool
	includeProfile bool // Include profiling information
}

// SlowQueryEntry represents a single slow query log entry
type SlowQueryEntry struct {
	Timestamp      time.Time              `json:"timestamp"`
	Duration       time.Duration          `json:"duration_ns"`
	DurationMS     float64                `json:"duration_ms"`
	Operation      string                 `json:"operation"` // "query", "insert", "update", "delete"
	Collection     string                 `json:"collection"`
	Filter         map[string]interface{} `json:"filter,omitempty"`
	Update         map[string]interface{} `json:"update,omitempty"`
	Document       map[string]interface{} `json:"document,omitempty"`
	DocsExamined   int                    `json:"docs_examined,omitempty"`
	DocsReturned   int                    `json:"docs_returned,omitempty"`
	IndexUsed      string                 `json:"index_used,omitempty"`
	ExecutionPlan  string                 `json:"execution_plan,omitempty"`
	Error          string                 `json:"error,omitempty"`
	UserInfo       map[string]string      `json:"user_info,omitempty"` // User, IP, session ID
}

// SlowQueryLogConfig holds configuration for the slow query log
type SlowQueryLogConfig struct {
	Threshold      time.Duration // Minimum duration to log (default: 100ms)
	MaxEntries     int           // Maximum in-memory entries (default: 1000)
	LogFilePath    string        // Optional file path for persistent logging
	Enabled        bool          // Enable/disable logging (default: true)
	IncludeProfile bool          // Include profiling information (default: true)
}

// DefaultSlowQueryLogConfig returns default configuration
func DefaultSlowQueryLogConfig() *SlowQueryLogConfig {
	return &SlowQueryLogConfig{
		Threshold:      100 * time.Millisecond,
		MaxEntries:     1000,
		Enabled:        true,
		IncludeProfile: true,
	}
}

// NewSlowQueryLog creates a new slow query log
func NewSlowQueryLog(config *SlowQueryLogConfig) (*SlowQueryLog, error) {
	if config == nil {
		config = DefaultSlowQueryLogConfig()
	}

	sql := &SlowQueryLog{
		threshold:      config.Threshold,
		maxEntries:     config.MaxEntries,
		entries:        make([]SlowQueryEntry, 0, config.MaxEntries),
		enabled:        config.Enabled,
		includeProfile: config.IncludeProfile,
	}

	// Open log file if path is provided
	if config.LogFilePath != "" {
		f, err := os.OpenFile(config.LogFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open slow query log file: %w", err)
		}
		sql.logFile = f
		sql.logToFile = true
	}

	return sql, nil
}

// LogQuery logs a query if it exceeds the threshold
func (sql *SlowQueryLog) LogQuery(entry SlowQueryEntry) {
	if !sql.enabled {
		return
	}

	// Only log if duration exceeds threshold
	if entry.Duration < sql.threshold {
		return
	}

	// Set timestamp and duration in ms
	entry.Timestamp = time.Now()
	entry.DurationMS = float64(entry.Duration.Nanoseconds()) / 1e6

	sql.mu.Lock()
	defer sql.mu.Unlock()

	// Add to in-memory buffer
	if len(sql.entries) >= sql.maxEntries {
		// Remove oldest entry (FIFO)
		sql.entries = sql.entries[1:]
	}
	sql.entries = append(sql.entries, entry)

	// Write to file if enabled
	if sql.logToFile && sql.logFile != nil {
		sql.writeToFile(entry)
	}
}

// writeToFile writes an entry to the log file (caller must hold lock)
func (sql *SlowQueryLog) writeToFile(entry SlowQueryEntry) {
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		// Silently ignore errors - logging should not crash the application
		return
	}

	_, _ = sql.logFile.Write(jsonBytes)
	_, _ = sql.logFile.Write([]byte("\n"))
}

// GetEntries returns all slow query log entries
func (sql *SlowQueryLog) GetEntries() []SlowQueryEntry {
	sql.mu.RLock()
	defer sql.mu.RUnlock()

	// Return a copy to prevent modification
	entries := make([]SlowQueryEntry, len(sql.entries))
	copy(entries, sql.entries)
	return entries
}

// GetRecentEntries returns the N most recent entries
func (sql *SlowQueryLog) GetRecentEntries(n int) []SlowQueryEntry {
	sql.mu.RLock()
	defer sql.mu.RUnlock()

	if n > len(sql.entries) {
		n = len(sql.entries)
	}

	// Get last n entries
	start := len(sql.entries) - n
	entries := make([]SlowQueryEntry, n)
	copy(entries, sql.entries[start:])
	return entries
}

// GetEntriesByCollection returns entries for a specific collection
func (sql *SlowQueryLog) GetEntriesByCollection(collection string) []SlowQueryEntry {
	sql.mu.RLock()
	defer sql.mu.RUnlock()

	var filtered []SlowQueryEntry
	for _, entry := range sql.entries {
		if entry.Collection == collection {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// GetEntriesByOperation returns entries for a specific operation type
func (sql *SlowQueryLog) GetEntriesByOperation(operation string) []SlowQueryEntry {
	sql.mu.RLock()
	defer sql.mu.RUnlock()

	var filtered []SlowQueryEntry
	for _, entry := range sql.entries {
		if entry.Operation == operation {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// GetEntriesSince returns entries since a specific time
func (sql *SlowQueryLog) GetEntriesSince(since time.Time) []SlowQueryEntry {
	sql.mu.RLock()
	defer sql.mu.RUnlock()

	var filtered []SlowQueryEntry
	for _, entry := range sql.entries {
		if entry.Timestamp.After(since) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// GetStatistics returns statistics about slow queries
func (sql *SlowQueryLog) GetStatistics() map[string]interface{} {
	sql.mu.RLock()
	defer sql.mu.RUnlock()

	if len(sql.entries) == 0 {
		return map[string]interface{}{
			"total_entries": 0,
			"threshold_ms":  sql.threshold.Milliseconds(),
		}
	}

	// Calculate statistics
	var totalDuration time.Duration
	var maxDuration time.Duration
	var minDuration time.Duration = 1<<63 - 1 // Max int64

	byOperation := make(map[string]int)
	byCollection := make(map[string]int)

	for _, entry := range sql.entries {
		totalDuration += entry.Duration
		if entry.Duration > maxDuration {
			maxDuration = entry.Duration
		}
		if entry.Duration < minDuration {
			minDuration = entry.Duration
		}

		byOperation[entry.Operation]++
		if entry.Collection != "" {
			byCollection[entry.Collection]++
		}
	}

	avgDuration := totalDuration / time.Duration(len(sql.entries))

	return map[string]interface{}{
		"total_entries":   len(sql.entries),
		"threshold_ms":    sql.threshold.Milliseconds(),
		"avg_duration_ms": float64(avgDuration.Nanoseconds()) / 1e6,
		"min_duration_ms": float64(minDuration.Nanoseconds()) / 1e6,
		"max_duration_ms": float64(maxDuration.Nanoseconds()) / 1e6,
		"by_operation":    byOperation,
		"by_collection":   byCollection,
	}
}

// Clear removes all entries from the log
func (sql *SlowQueryLog) Clear() {
	sql.mu.Lock()
	defer sql.mu.Unlock()

	sql.entries = make([]SlowQueryEntry, 0, sql.maxEntries)
}

// SetThreshold updates the threshold duration
func (sql *SlowQueryLog) SetThreshold(threshold time.Duration) {
	sql.mu.Lock()
	defer sql.mu.Unlock()

	sql.threshold = threshold
}

// GetThreshold returns the current threshold
func (sql *SlowQueryLog) GetThreshold() time.Duration {
	sql.mu.RLock()
	defer sql.mu.RUnlock()

	return sql.threshold
}

// Enable enables slow query logging
func (sql *SlowQueryLog) Enable() {
	sql.mu.Lock()
	defer sql.mu.Unlock()

	sql.enabled = true
}

// Disable disables slow query logging
func (sql *SlowQueryLog) Disable() {
	sql.mu.Lock()
	defer sql.mu.Unlock()

	sql.enabled = false
}

// IsEnabled returns whether logging is enabled
func (sql *SlowQueryLog) IsEnabled() bool {
	sql.mu.RLock()
	defer sql.mu.RUnlock()

	return sql.enabled
}

// ExportToJSON exports all entries to a JSON writer
func (sql *SlowQueryLog) ExportToJSON(w io.Writer) error {
	sql.mu.RLock()
	defer sql.mu.RUnlock()

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(sql.entries)
}

// Close closes the log file if open
func (sql *SlowQueryLog) Close() error {
	sql.mu.Lock()
	defer sql.mu.Unlock()

	if sql.logFile != nil {
		err := sql.logFile.Close()
		sql.logFile = nil
		sql.logToFile = false
		return err
	}
	return nil
}

// GetTopSlowest returns the N slowest queries
func (sql *SlowQueryLog) GetTopSlowest(n int) []SlowQueryEntry {
	sql.mu.RLock()
	defer sql.mu.RUnlock()

	if len(sql.entries) == 0 {
		return nil
	}

	// Create a copy for sorting
	entries := make([]SlowQueryEntry, len(sql.entries))
	copy(entries, sql.entries)

	// Sort by duration (descending) using simple insertion sort
	for i := 1; i < len(entries); i++ {
		key := entries[i]
		j := i - 1
		for j >= 0 && entries[j].Duration < key.Duration {
			entries[j+1] = entries[j]
			j--
		}
		entries[j+1] = key
	}

	// Return top N
	if n > len(entries) {
		n = len(entries)
	}
	return entries[:n]
}

// GetSlowestByCollection returns the slowest query for each collection
func (sql *SlowQueryLog) GetSlowestByCollection() map[string]SlowQueryEntry {
	sql.mu.RLock()
	defer sql.mu.RUnlock()

	slowest := make(map[string]SlowQueryEntry)

	for _, entry := range sql.entries {
		if entry.Collection == "" {
			continue
		}

		if existing, exists := slowest[entry.Collection]; !exists || entry.Duration > existing.Duration {
			slowest[entry.Collection] = entry
		}
	}

	return slowest
}
