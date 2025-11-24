package replication

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mnohosten/laura-db/pkg/database"
)

// SlaveConfig holds configuration for a slave node
type SlaveConfig struct {
	SlaveID          string
	Database         *database.Database
	MasterClient     MasterClient
	PollInterval     time.Duration
	HeartbeatInterval time.Duration
	RetryInterval    time.Duration
	MaxRetries       int
}

// DefaultSlaveConfig returns default slave configuration
func DefaultSlaveConfig(slaveID string, db *database.Database, masterClient MasterClient) *SlaveConfig {
	return &SlaveConfig{
		SlaveID:          slaveID,
		Database:         db,
		MasterClient:     masterClient,
		PollInterval:     1 * time.Second,
		HeartbeatInterval: 5 * time.Second,
		RetryInterval:    5 * time.Second,
		MaxRetries:       3,
	}
}

// MasterClient is an interface for communicating with the master
type MasterClient interface {
	// GetOplogEntries fetches oplog entries since the given OpID
	GetOplogEntries(ctx context.Context, sinceID OpID) ([]*OplogEntry, error)

	// SendHeartbeat sends a heartbeat to the master
	SendHeartbeat(ctx context.Context, slaveID string, lastOpID OpID) error

	// Register registers the slave with the master
	Register(ctx context.Context, slaveID string) error

	// Unregister unregisters the slave from the master
	Unregister(ctx context.Context, slaveID string) error
}

// Slave represents a replication slave node
type Slave struct {
	config            *SlaveConfig
	db                *database.Database
	masterClient      MasterClient
	lastAppliedOpID   OpID
	mu                sync.RWMutex
	stopChan          chan struct{}
	wg                sync.WaitGroup
	isRunning         bool
	replicationErrors int
}

// NewSlave creates a new slave node
func NewSlave(config *SlaveConfig) (*Slave, error) {
	return &Slave{
		config:          config,
		db:              config.Database,
		masterClient:    config.MasterClient,
		lastAppliedOpID: 0,
		stopChan:        make(chan struct{}),
	}, nil
}

// Start starts the slave replication process
func (s *Slave) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("slave already running")
	}

	// Register with master
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.masterClient.Register(ctx, s.config.SlaveID); err != nil {
		return fmt.Errorf("failed to register with master: %w", err)
	}

	s.isRunning = true

	// Start replication goroutine
	s.wg.Add(1)
	go s.replicationLoop()

	// Start heartbeat goroutine
	s.wg.Add(1)
	go s.heartbeatLoop()

	return nil
}

// Stop stops the slave replication process
func (s *Slave) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return nil
	}

	// Signal stop
	close(s.stopChan)
	s.wg.Wait()

	// Unregister from master
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.masterClient.Unregister(ctx, s.config.SlaveID); err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: failed to unregister from master: %v\n", err)
	}

	s.isRunning = false
	return nil
}

// replicationLoop continuously fetches and applies oplog entries
func (s *Slave) replicationLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.fetchAndApplyEntries(); err != nil {
				fmt.Printf("Replication error: %v\n", err)
				s.mu.Lock()
				s.replicationErrors++
				s.mu.Unlock()
			}
		case <-s.stopChan:
			return
		}
	}
}

// heartbeatLoop sends periodic heartbeats to the master
func (s *Slave) heartbeatLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.sendHeartbeat()
		case <-s.stopChan:
			return
		}
	}
}

// fetchAndApplyEntries fetches new oplog entries and applies them
func (s *Slave) fetchAndApplyEntries() error {
	s.mu.RLock()
	lastOpID := s.lastAppliedOpID
	s.mu.RUnlock()

	// Fetch entries from master
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	entries, err := s.masterClient.GetOplogEntries(ctx, lastOpID)
	if err != nil {
		return fmt.Errorf("failed to fetch oplog entries: %w", err)
	}

	// Apply each entry
	for _, entry := range entries {
		if err := s.applyEntry(entry); err != nil {
			return fmt.Errorf("failed to apply entry %d: %w", entry.OpID, err)
		}

		s.mu.Lock()
		s.lastAppliedOpID = entry.OpID
		s.mu.Unlock()
	}

	return nil
}

// applyEntry applies a single oplog entry to the local database
func (s *Slave) applyEntry(entry *OplogEntry) error {
	// Get the collection
	coll := s.db.Collection(entry.Collection)

	switch entry.OpType {
	case OpTypeInsert:
		// Insert document
		if _, err := coll.InsertOne(entry.Document); err != nil {
			return fmt.Errorf("insert failed: %w", err)
		}

	case OpTypeUpdate:
		// Update document
		if err := coll.UpdateOne(entry.Filter, entry.Update); err != nil {
			return fmt.Errorf("update failed: %w", err)
		}

	case OpTypeDelete:
		// Delete document
		if err := coll.DeleteOne(entry.Filter); err != nil {
			return fmt.Errorf("delete failed: %w", err)
		}

	case OpTypeCreateCollection:
		// Create collection
		if _, err := s.db.CreateCollection(entry.Collection); err != nil {
			// Ignore if collection already exists
			if err.Error() != fmt.Sprintf("collection %s already exists", entry.Collection) {
				return fmt.Errorf("create collection failed: %w", err)
			}
		}

	case OpTypeDropCollection:
		// Drop collection
		if err := s.db.DropCollection(entry.Collection); err != nil {
			// Ignore if collection doesn't exist
			if err.Error() != fmt.Sprintf("collection %s does not exist", entry.Collection) {
				return fmt.Errorf("drop collection failed: %w", err)
			}
		}

	case OpTypeCreateIndex:
		// Create index - simplified for now
		// In a full implementation, would need to parse index definition
		// and call appropriate CreateIndex method

	case OpTypeDropIndex:
		// Drop index - simplified for now

	case OpTypeNoop:
		// No operation - do nothing

	default:
		return fmt.Errorf("unknown operation type: %d", entry.OpType)
	}

	return nil
}

// sendHeartbeat sends a heartbeat to the master
func (s *Slave) sendHeartbeat() {
	s.mu.RLock()
	lastOpID := s.lastAppliedOpID
	s.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.masterClient.SendHeartbeat(ctx, s.config.SlaveID, lastOpID); err != nil {
		fmt.Printf("Heartbeat error: %v\n", err)
	}
}

// GetLastAppliedOpID returns the last applied OpID
func (s *Slave) GetLastAppliedOpID() OpID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastAppliedOpID
}

// IsRunning returns whether the slave is running
func (s *Slave) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// Stats returns statistics about the slave
func (s *Slave) Stats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"slave_id":            s.config.SlaveID,
		"last_applied_op_id":  s.lastAppliedOpID,
		"is_running":          s.isRunning,
		"replication_errors":  s.replicationErrors,
	}
}

// InitialSync performs an initial sync from the master
// This should be called before Start() for a new slave
func (s *Slave) InitialSync(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("cannot perform initial sync while running")
	}

	// Fetch all oplog entries from the beginning
	entries, err := s.masterClient.GetOplogEntries(ctx, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch oplog for initial sync: %w", err)
	}

	// Apply all entries
	for _, entry := range entries {
		if err := s.applyEntry(entry); err != nil {
			return fmt.Errorf("failed to apply entry %d during initial sync: %w", entry.OpID, err)
		}
		s.lastAppliedOpID = entry.OpID
	}

	return nil
}

// GetLag returns the estimated replication lag
func (s *Slave) GetLag(masterOpID OpID) time.Duration {
	s.mu.RLock()
	lastOpID := s.lastAppliedOpID
	s.mu.RUnlock()

	if masterOpID <= lastOpID {
		return 0
	}

	// Estimate 1 operation per millisecond
	return time.Duration(masterOpID-lastOpID) * time.Millisecond
}

// ReadDocument reads a document from the local database (read-only operation)
func (s *Slave) ReadDocument(collName string, filter map[string]interface{}) (map[string]interface{}, error) {
	coll := s.db.Collection(collName)

	// Execute query
	result, err := coll.FindOne(filter)
	if err != nil {
		return nil, err
	}

	return result.ToMap(), nil
}

// ReadDocuments reads multiple documents from the local database (read-only operation)
func (s *Slave) ReadDocuments(collName string, filter map[string]interface{}) ([]map[string]interface{}, error) {
	coll := s.db.Collection(collName)

	// Execute query
	docs, err := coll.Find(filter)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(docs))
	for i, doc := range docs {
		result[i] = doc.ToMap()
	}

	return result, nil
}
