package replication

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mnohosten/laura-db/pkg/database"
)

// MasterConfig holds configuration for a master node
type MasterConfig struct {
	Database         *database.Database
	OplogPath        string
	HeartbeatTimeout time.Duration
	MaxSlaves        int
}

// DefaultMasterConfig returns default master configuration
func DefaultMasterConfig(db *database.Database, oplogPath string) *MasterConfig {
	return &MasterConfig{
		Database:         db,
		OplogPath:        oplogPath,
		HeartbeatTimeout: 30 * time.Second,
		MaxSlaves:        10,
	}
}

// Master represents a replication master node
type Master struct {
	db              *database.Database
	oplog           *Oplog
	config          *MasterConfig
	slaves          map[string]*SlaveInfo
	mu              sync.RWMutex
	stopChan        chan struct{}
	heartbeatTicker *time.Ticker
	isRunning       bool
}

// SlaveInfo tracks information about a connected slave
type SlaveInfo struct {
	ID              string
	LastHeartbeat   time.Time
	LastOpID        OpID
	Lag             time.Duration
	mu              sync.RWMutex
}

// NewMaster creates a new master node
func NewMaster(config *MasterConfig) (*Master, error) {
	// Create oplog
	oplog, err := NewOplog(config.OplogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create oplog: %w", err)
	}

	return &Master{
		db:       config.Database,
		oplog:    oplog,
		config:   config,
		slaves:   make(map[string]*SlaveInfo),
		stopChan: make(chan struct{}),
	}, nil
}

// Start starts the master node
func (m *Master) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return fmt.Errorf("master already running")
	}

	m.isRunning = true

	// Start heartbeat monitor
	m.heartbeatTicker = time.NewTicker(10 * time.Second)
	go m.monitorHeartbeats()

	return nil
}

// Stop stops the master node
func (m *Master) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return nil
	}

	close(m.stopChan)
	m.heartbeatTicker.Stop()
	m.isRunning = false

	return m.oplog.Close()
}

// LogOperation logs an operation to the oplog
func (m *Master) LogOperation(entry *OplogEntry) error {
	if err := m.oplog.Append(entry); err != nil {
		return fmt.Errorf("failed to append to oplog: %w", err)
	}
	return nil
}

// GetOplogEntries returns oplog entries since the given OpID
func (m *Master) GetOplogEntries(sinceID OpID) ([]*OplogEntry, error) {
	return m.oplog.GetEntriesSince(sinceID)
}

// RegisterSlave registers a new slave
func (m *Master) RegisterSlave(slaveID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.slaves) >= m.config.MaxSlaves {
		return fmt.Errorf("maximum number of slaves reached")
	}

	if _, exists := m.slaves[slaveID]; exists {
		return fmt.Errorf("slave %s already registered", slaveID)
	}

	m.slaves[slaveID] = &SlaveInfo{
		ID:            slaveID,
		LastHeartbeat: time.Now(),
		LastOpID:      0,
		Lag:           0,
	}

	return nil
}

// UnregisterSlave removes a slave
func (m *Master) UnregisterSlave(slaveID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.slaves[slaveID]; !exists {
		return fmt.Errorf("slave %s not registered", slaveID)
	}

	delete(m.slaves, slaveID)
	return nil
}

// UpdateSlaveHeartbeat updates the last heartbeat time for a slave
func (m *Master) UpdateSlaveHeartbeat(slaveID string, lastOpID OpID) error {
	m.mu.RLock()
	slave, exists := m.slaves[slaveID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("slave %s not registered", slaveID)
	}

	slave.mu.Lock()
	defer slave.mu.Unlock()

	slave.LastHeartbeat = time.Now()
	slave.LastOpID = lastOpID

	// Calculate lag (approximate based on OpID difference)
	currentOpID := m.oplog.GetCurrentID()
	if currentOpID > lastOpID {
		// Estimate 1 operation per millisecond for lag calculation
		slave.Lag = time.Duration(currentOpID-lastOpID) * time.Millisecond
	} else {
		slave.Lag = 0
	}

	return nil
}

// GetSlaveInfo returns information about a slave
func (m *Master) GetSlaveInfo(slaveID string) (*SlaveInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	slave, exists := m.slaves[slaveID]
	if !exists {
		return nil, fmt.Errorf("slave %s not registered", slaveID)
	}

	// Return a copy to avoid race conditions
	slave.mu.RLock()
	defer slave.mu.RUnlock()

	return &SlaveInfo{
		ID:            slave.ID,
		LastHeartbeat: slave.LastHeartbeat,
		LastOpID:      slave.LastOpID,
		Lag:           slave.Lag,
	}, nil
}

// GetAllSlaves returns information about all registered slaves
func (m *Master) GetAllSlaves() []*SlaveInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*SlaveInfo, 0, len(m.slaves))
	for _, slave := range m.slaves {
		slave.mu.RLock()
		result = append(result, &SlaveInfo{
			ID:            slave.ID,
			LastHeartbeat: slave.LastHeartbeat,
			LastOpID:      slave.LastOpID,
			Lag:           slave.Lag,
		})
		slave.mu.RUnlock()
	}

	return result
}

// monitorHeartbeats monitors slave heartbeats and removes stale slaves
func (m *Master) monitorHeartbeats() {
	for {
		select {
		case <-m.heartbeatTicker.C:
			m.checkHeartbeats()
		case <-m.stopChan:
			return
		}
	}
}

// checkHeartbeats checks for slaves that haven't sent heartbeats recently
func (m *Master) checkHeartbeats() {
	m.mu.RLock()
	now := time.Now()
	staleSlaves := make([]string, 0)

	for id, slave := range m.slaves {
		slave.mu.RLock()
		if now.Sub(slave.LastHeartbeat) > m.config.HeartbeatTimeout {
			staleSlaves = append(staleSlaves, id)
		}
		slave.mu.RUnlock()
	}
	m.mu.RUnlock()

	// Remove stale slaves
	if len(staleSlaves) > 0 {
		m.mu.Lock()
		for _, id := range staleSlaves {
			delete(m.slaves, id)
		}
		m.mu.Unlock()
	}
}

// GetCurrentOpID returns the current oplog ID
func (m *Master) GetCurrentOpID() OpID {
	return m.oplog.GetCurrentID()
}

// Stats returns statistics about the master
func (m *Master) Stats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	slaves := m.GetAllSlaves()
	slaveStats := make([]map[string]interface{}, 0, len(slaves))
	for _, slave := range slaves {
		slaveStats = append(slaveStats, map[string]interface{}{
			"id":              slave.ID,
			"last_heartbeat":  slave.LastHeartbeat,
			"last_op_id":      slave.LastOpID,
			"lag":             slave.Lag.String(),
		})
	}

	return map[string]interface{}{
		"current_op_id":  m.oplog.GetCurrentID(),
		"slave_count":    len(m.slaves),
		"slaves":         slaveStats,
		"is_running":     m.isRunning,
	}
}

// Flush flushes the oplog to disk
func (m *Master) Flush() error {
	return m.oplog.Flush()
}

// WaitForSlaves waits for all slaves to catch up to the specified OpID
func (m *Master) WaitForSlaves(ctx context.Context, opID OpID, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for slaves to catch up")
		}

		allCaughtUp := true
		m.mu.RLock()
		for _, slave := range m.slaves {
			slave.mu.RLock()
			if slave.LastOpID < opID {
				allCaughtUp = false
			}
			slave.mu.RUnlock()
		}
		m.mu.RUnlock()

		if allCaughtUp {
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}
}
