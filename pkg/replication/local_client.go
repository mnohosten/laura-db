package replication

import (
	"context"
	"fmt"
)

// LocalMasterClient implements MasterClient for in-process replication
// This is useful for testing and embedded scenarios where master and slave
// run in the same process
type LocalMasterClient struct {
	master *Master
}

// NewLocalMasterClient creates a new local master client
func NewLocalMasterClient(master *Master) *LocalMasterClient {
	return &LocalMasterClient{
		master: master,
	}
}

// GetOplogEntries fetches oplog entries since the given OpID
func (c *LocalMasterClient) GetOplogEntries(ctx context.Context, sinceID OpID) ([]*OplogEntry, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return c.master.GetOplogEntries(sinceID)
}

// SendHeartbeat sends a heartbeat to the master
func (c *LocalMasterClient) SendHeartbeat(ctx context.Context, slaveID string, lastOpID OpID) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return c.master.UpdateSlaveHeartbeat(slaveID, lastOpID)
}

// Register registers the slave with the master
func (c *LocalMasterClient) Register(ctx context.Context, slaveID string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return c.master.RegisterSlave(slaveID)
}

// Unregister unregisters the slave from the master
func (c *LocalMasterClient) Unregister(ctx context.Context, slaveID string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return c.master.UnregisterSlave(slaveID)
}

// Verify that LocalMasterClient implements MasterClient
var _ MasterClient = (*LocalMasterClient)(nil)

// ReplicationPair represents a master-slave pair for easy setup
type ReplicationPair struct {
	Master *Master
	Slave  *Slave
}

// NewReplicationPair creates a new master-slave replication pair
// This is a convenience function for setting up replication in a single process
func NewReplicationPair(masterConfig *MasterConfig, slaveConfig *SlaveConfig) (*ReplicationPair, error) {
	// Create master
	master, err := NewMaster(masterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create master: %w", err)
	}

	// Create local client
	client := NewLocalMasterClient(master)

	// Update slave config with client
	slaveConfig.MasterClient = client

	// Create slave
	slave, err := NewSlave(slaveConfig)
	if err != nil {
		master.Stop()
		return nil, fmt.Errorf("failed to create slave: %w", err)
	}

	return &ReplicationPair{
		Master: master,
		Slave:  slave,
	}, nil
}

// Start starts both master and slave
func (p *ReplicationPair) Start() error {
	// Start master
	if err := p.Master.Start(); err != nil {
		return fmt.Errorf("failed to start master: %w", err)
	}

	// Start slave
	if err := p.Slave.Start(); err != nil {
		p.Master.Stop()
		return fmt.Errorf("failed to start slave: %w", err)
	}

	return nil
}

// Stop stops both slave and master
func (p *ReplicationPair) Stop() error {
	// Stop slave first
	if err := p.Slave.Stop(); err != nil {
		return fmt.Errorf("failed to stop slave: %w", err)
	}

	// Stop master
	if err := p.Master.Stop(); err != nil {
		return fmt.Errorf("failed to stop master: %w", err)
	}

	return nil
}
