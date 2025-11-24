package replication

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mnohosten/laura-db/pkg/database"
)

// NodeRole represents the role of a node in the replica set
type NodeRole int

const (
	RolePrimary NodeRole = iota
	RoleSecondary
	RoleArbiter // For tie-breaking in elections
)

func (r NodeRole) String() string {
	switch r {
	case RolePrimary:
		return "PRIMARY"
	case RoleSecondary:
		return "SECONDARY"
	case RoleArbiter:
		return "ARBITER"
	default:
		return "UNKNOWN"
	}
}

// NodeState represents the health state of a node
type NodeState int

const (
	StateHealthy NodeState = iota
	StateUnhealthy
	StateUnreachable
)

func (s NodeState) String() string {
	switch s {
	case StateHealthy:
		return "HEALTHY"
	case StateUnhealthy:
		return "UNHEALTHY"
	case StateUnreachable:
		return "UNREACHABLE"
	default:
		return "UNKNOWN"
	}
}

// ReplicaSetConfig holds configuration for a replica set
type ReplicaSetConfig struct {
	Name              string
	Database          *database.Database
	OplogPath         string
	NodeID            string
	Priority          int           // Higher priority nodes are preferred as primary
	HeartbeatInterval time.Duration // How often to send heartbeats
	ElectionTimeout   time.Duration // Timeout before starting election
	HeartbeatTimeout  time.Duration // Timeout for considering node dead
	VotingMembers     []string      // List of voting member node IDs
}

// DefaultReplicaSetConfig returns default replica set configuration
func DefaultReplicaSetConfig(rsName, nodeID string, db *database.Database, oplogPath string) *ReplicaSetConfig {
	return &ReplicaSetConfig{
		Name:              rsName,
		Database:          db,
		OplogPath:         oplogPath,
		NodeID:            nodeID,
		Priority:          1,
		HeartbeatInterval: 2 * time.Second,
		ElectionTimeout:   10 * time.Second,
		HeartbeatTimeout:  15 * time.Second,
		VotingMembers:     []string{},
	}
}

// ReplicaSetMember represents information about a replica set member
type ReplicaSetMember struct {
	NodeID         string
	Role           NodeRole
	State          NodeState
	Priority       int
	LastHeartbeat  time.Time
	LastOpID       OpID
	Lag            time.Duration
	IsVotingMember bool
	mu             sync.RWMutex
}

// ReplicaSet represents a group of nodes with automatic failover
type ReplicaSet struct {
	config         *ReplicaSetConfig
	db             *database.Database
	oplog          *Oplog

	// Current state
	role           NodeRole
	currentPrimary string
	currentTerm    int64 // Election term number
	votedFor       string // Candidate voted for in current term

	// Members
	members        map[string]*ReplicaSetMember
	membersMu      sync.RWMutex

	// Master/Slave components (used based on role)
	master         *Master
	slave          *Slave

	// Election state
	lastHeartbeat  time.Time
	electionTimer  *time.Timer
	heartbeatTimer *time.Timer

	// Control
	stopChan       chan struct{}
	wg             sync.WaitGroup
	isRunning      bool
	mu             sync.RWMutex
}

// NewReplicaSet creates a new replica set node
func NewReplicaSet(config *ReplicaSetConfig) (*ReplicaSet, error) {
	// Create oplog
	oplog, err := NewOplog(config.OplogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create oplog: %w", err)
	}

	rs := &ReplicaSet{
		config:      config,
		db:          config.Database,
		oplog:       oplog,
		role:        RoleSecondary, // Start as secondary
		members:     make(map[string]*ReplicaSetMember),
		stopChan:    make(chan struct{}),
		currentTerm: 0,
	}

	// Add self as member
	rs.members[config.NodeID] = &ReplicaSetMember{
		NodeID:         config.NodeID,
		Role:           RoleSecondary,
		State:          StateHealthy,
		Priority:       config.Priority,
		LastHeartbeat:  time.Now(),
		IsVotingMember: true,
	}

	return rs, nil
}

// Start starts the replica set node
func (rs *ReplicaSet) Start() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.isRunning {
		return fmt.Errorf("replica set already running")
	}

	rs.isRunning = true
	rs.lastHeartbeat = time.Now()

	// Start election timer
	rs.resetElectionTimer()

	// Start monitoring goroutine
	rs.wg.Add(1)
	go rs.monitorLoop()

	return nil
}

// Stop stops the replica set node
func (rs *ReplicaSet) Stop() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if !rs.isRunning {
		return nil
	}

	// Signal stop
	close(rs.stopChan)

	// Stop timers
	if rs.electionTimer != nil {
		rs.electionTimer.Stop()
	}
	if rs.heartbeatTimer != nil {
		rs.heartbeatTimer.Stop()
	}

	// Stop master or slave if running
	if rs.master != nil {
		rs.master.Stop()
	}
	if rs.slave != nil {
		rs.slave.Stop()
	}

	// Wait for goroutines
	rs.wg.Wait()

	rs.isRunning = false

	return rs.oplog.Close()
}

// monitorLoop is the main monitoring loop
func (rs *ReplicaSet) monitorLoop() {
	defer rs.wg.Done()

	for {
		select {
		case <-rs.stopChan:
			return
		case <-time.After(1 * time.Second):
			rs.checkMemberHealth()
		}
	}
}

// resetElectionTimer resets the election timer with random timeout
func (rs *ReplicaSet) resetElectionTimer() {
	if rs.electionTimer != nil {
		rs.electionTimer.Stop()
	}

	// Use election timeout
	timeout := rs.config.ElectionTimeout

	rs.electionTimer = time.AfterFunc(timeout, func() {
		rs.startElection()
	})
}

// checkMemberHealth checks the health of all members
func (rs *ReplicaSet) checkMemberHealth() {
	now := time.Now()

	// Update member health states
	rs.membersMu.Lock()
	for _, member := range rs.members {
		member.mu.Lock()
		if now.Sub(member.LastHeartbeat) > rs.config.HeartbeatTimeout {
			member.State = StateUnreachable
		}
		member.mu.Unlock()
	}
	rs.membersMu.Unlock()

	// If we're a secondary and haven't heard from primary, start election
	rs.mu.RLock()
	role := rs.role
	lastHB := rs.lastHeartbeat
	rs.mu.RUnlock()

	if role == RoleSecondary && now.Sub(lastHB) > rs.config.ElectionTimeout {
		rs.startElection()
	}
}

// startElection initiates a leader election
func (rs *ReplicaSet) startElection() {
	rs.mu.Lock()

	// Increment term
	rs.currentTerm++
	term := rs.currentTerm

	// Vote for self
	rs.votedFor = rs.config.NodeID
	rs.role = RoleSecondary // Candidate state (simplified)

	rs.mu.Unlock()

	// Count votes (simplified - in real implementation would send vote requests)
	votes := rs.collectVotes(term)

	// Need majority to win
	// Count ALL voting members (including unreachable) for majority calculation
	// This is standard Raft/MongoDB behavior
	votingMembers := rs.countVotingMembers()
	majority := (votingMembers / 2) + 1

	if votes >= majority {
		rs.becomePrimary()
	} else {
		// Election failed, reset timer
		rs.resetElectionTimer()
	}
}

// collectVotes collects votes from members (simplified version)
func (rs *ReplicaSet) collectVotes(term int64) int {
	votes := 1 // Vote for self

	rs.membersMu.RLock()
	defer rs.membersMu.RUnlock()

	// In a real implementation, would send VoteRequest RPCs to all members
	// For now, we'll simulate: healthy members with lower priority vote for us
	for nodeID, member := range rs.members {
		if nodeID == rs.config.NodeID {
			continue // Already voted for self
		}

		member.mu.RLock()
		state := member.State
		priority := member.Priority
		isVoting := member.IsVotingMember
		member.mu.RUnlock()

		// Simplified voting logic:
		// - Member must be healthy and voting
		// - Members vote for candidates with higher or equal priority
		if state == StateHealthy && isVoting && priority <= rs.config.Priority {
			votes++
		}
	}

	return votes
}

// countVotingMembers counts the number of voting members
func (rs *ReplicaSet) countVotingMembers() int {
	rs.membersMu.RLock()
	defer rs.membersMu.RUnlock()

	count := 0
	for _, member := range rs.members {
		member.mu.RLock()
		if member.IsVotingMember {
			count++
		}
		member.mu.RUnlock()
	}

	return count
}

// BecomePrimary transitions this node to primary role (exported for testing and demos)
func (rs *ReplicaSet) BecomePrimary() error {
	return rs.becomePrimary()
}

// becomePrimary transitions this node to primary role
func (rs *ReplicaSet) becomePrimary() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.role == RolePrimary {
		return nil // Already primary
	}

	// Stop slave if running
	if rs.slave != nil {
		rs.slave.Stop()
		rs.slave = nil
	}

	// Create and start master
	masterConfig := DefaultMasterConfig(rs.db, rs.config.OplogPath)
	master, err := NewMaster(masterConfig)
	if err != nil {
		return fmt.Errorf("failed to create master: %w", err)
	}

	if err := master.Start(); err != nil {
		return fmt.Errorf("failed to start master: %w", err)
	}

	rs.master = master
	rs.role = RolePrimary
	rs.currentPrimary = rs.config.NodeID

	// Update member role
	rs.membersMu.Lock()
	if member, exists := rs.members[rs.config.NodeID]; exists {
		member.mu.Lock()
		member.Role = RolePrimary
		member.mu.Unlock()
	}
	rs.membersMu.Unlock()

	// Stop election timer and start heartbeat timer
	if rs.electionTimer != nil {
		rs.electionTimer.Stop()
		rs.electionTimer = nil
	}

	rs.startHeartbeatTimer()

	return nil
}

// becomeSecondary transitions this node to secondary role
func (rs *ReplicaSet) becomeSecondary(primaryID string) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.role == RoleSecondary && rs.currentPrimary == primaryID {
		return nil // Already secondary following this primary
	}

	// Stop master if running
	if rs.master != nil {
		rs.master.Stop()
		rs.master = nil
	}

	// Stop heartbeat timer and start election timer
	if rs.heartbeatTimer != nil {
		rs.heartbeatTimer.Stop()
		rs.heartbeatTimer = nil
	}

	rs.role = RoleSecondary
	rs.currentPrimary = primaryID
	rs.lastHeartbeat = time.Now()

	rs.resetElectionTimer()

	// In a real implementation, would create and start slave here
	// For now, we'll keep it simple

	return nil
}

// startHeartbeatTimer starts the heartbeat timer for primary
func (rs *ReplicaSet) startHeartbeatTimer() {
	if rs.heartbeatTimer != nil {
		rs.heartbeatTimer.Stop()
	}

	rs.heartbeatTimer = time.AfterFunc(rs.config.HeartbeatInterval, func() {
		rs.sendHeartbeats()
		rs.startHeartbeatTimer() // Restart timer
	})
}

// sendHeartbeats sends heartbeats to all members (primary only)
func (rs *ReplicaSet) sendHeartbeats() {
	rs.mu.RLock()
	if rs.role != RolePrimary {
		rs.mu.RUnlock()
		return
	}
	rs.mu.RUnlock()

	// In a real implementation, would send AppendEntries RPCs to all members
	// For now, we'll just update our own heartbeat
	rs.membersMu.Lock()
	if member, exists := rs.members[rs.config.NodeID]; exists {
		member.mu.Lock()
		member.LastHeartbeat = time.Now()
		member.mu.Unlock()
	}
	rs.membersMu.Unlock()
}

// AddMember adds a member to the replica set
func (rs *ReplicaSet) AddMember(nodeID string, priority int, isVoting bool) error {
	rs.membersMu.Lock()
	defer rs.membersMu.Unlock()

	if _, exists := rs.members[nodeID]; exists {
		return fmt.Errorf("member %s already exists", nodeID)
	}

	rs.members[nodeID] = &ReplicaSetMember{
		NodeID:         nodeID,
		Role:           RoleSecondary,
		State:          StateHealthy,
		Priority:       priority,
		LastHeartbeat:  time.Now(),
		IsVotingMember: isVoting,
	}

	return nil
}

// RemoveMember removes a member from the replica set
func (rs *ReplicaSet) RemoveMember(nodeID string) error {
	rs.membersMu.Lock()
	defer rs.membersMu.Unlock()

	if nodeID == rs.config.NodeID {
		return fmt.Errorf("cannot remove self from replica set")
	}

	if _, exists := rs.members[nodeID]; !exists {
		return fmt.Errorf("member %s does not exist", nodeID)
	}

	delete(rs.members, nodeID)
	return nil
}

// UpdateMemberHeartbeat updates the heartbeat for a member
func (rs *ReplicaSet) UpdateMemberHeartbeat(nodeID string, opID OpID) error {
	rs.membersMu.RLock()
	member, exists := rs.members[nodeID]
	rs.membersMu.RUnlock()

	if !exists {
		return fmt.Errorf("member %s not found", nodeID)
	}

	member.mu.Lock()
	defer member.mu.Unlock()

	member.LastHeartbeat = time.Now()
	member.LastOpID = opID
	member.State = StateHealthy

	// Calculate lag
	currentOpID := rs.oplog.GetCurrentID()
	if currentOpID > opID {
		member.Lag = time.Duration(currentOpID-opID) * time.Millisecond
	} else {
		member.Lag = 0
	}

	// Update our last heartbeat time if this is from primary
	rs.mu.Lock()
	if nodeID == rs.currentPrimary {
		rs.lastHeartbeat = time.Now()
		rs.resetElectionTimer()
	}
	rs.mu.Unlock()

	return nil
}

// GetRole returns the current role of this node
func (rs *ReplicaSet) GetRole() NodeRole {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.role
}

// GetPrimary returns the current primary node ID
func (rs *ReplicaSet) GetPrimary() string {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.currentPrimary
}

// IsPrimary returns true if this node is the primary
func (rs *ReplicaSet) IsPrimary() bool {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.role == RolePrimary
}

// GetMembers returns information about all members
func (rs *ReplicaSet) GetMembers() []*ReplicaSetMember {
	rs.membersMu.RLock()
	defer rs.membersMu.RUnlock()

	result := make([]*ReplicaSetMember, 0, len(rs.members))
	for _, member := range rs.members {
		member.mu.RLock()
		result = append(result, &ReplicaSetMember{
			NodeID:         member.NodeID,
			Role:           member.Role,
			State:          member.State,
			Priority:       member.Priority,
			LastHeartbeat:  member.LastHeartbeat,
			LastOpID:       member.LastOpID,
			Lag:            member.Lag,
			IsVotingMember: member.IsVotingMember,
		})
		member.mu.RUnlock()
	}

	return result
}

// Stats returns statistics about the replica set
func (rs *ReplicaSet) Stats() map[string]interface{} {
	rs.mu.RLock()
	role := rs.role
	primary := rs.currentPrimary
	term := rs.currentTerm
	rs.mu.RUnlock()

	members := rs.GetMembers()
	memberStats := make([]map[string]interface{}, 0, len(members))
	for _, member := range members {
		memberStats = append(memberStats, map[string]interface{}{
			"node_id":         member.NodeID,
			"role":            member.Role.String(),
			"state":           member.State.String(),
			"priority":        member.Priority,
			"last_heartbeat":  member.LastHeartbeat,
			"last_op_id":      member.LastOpID,
			"lag":             member.Lag.String(),
			"is_voting":       member.IsVotingMember,
		})
	}

	return map[string]interface{}{
		"replica_set_name": rs.config.Name,
		"node_id":          rs.config.NodeID,
		"role":             role.String(),
		"primary":          primary,
		"term":             term,
		"member_count":     len(members),
		"members":          memberStats,
		"is_running":       rs.isRunning,
	}
}

// StepDown forces the primary to step down (manual failover)
func (rs *ReplicaSet) StepDown() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.role != RolePrimary {
		return fmt.Errorf("node is not primary")
	}

	// Stop master
	if rs.master != nil {
		rs.master.Stop()
		rs.master = nil
	}

	// Become secondary
	rs.role = RoleSecondary
	rs.currentPrimary = ""

	// Stop heartbeat timer and start election timer
	if rs.heartbeatTimer != nil {
		rs.heartbeatTimer.Stop()
		rs.heartbeatTimer = nil
	}

	rs.resetElectionTimer()

	return nil
}

// SimulateFailure simulates a node failure (for testing)
func (rs *ReplicaSet) SimulateFailure(nodeID string) error {
	rs.membersMu.RLock()
	member, exists := rs.members[nodeID]
	rs.membersMu.RUnlock()

	if !exists {
		return fmt.Errorf("member %s not found", nodeID)
	}

	member.mu.Lock()
	defer member.mu.Unlock()

	member.State = StateUnreachable
	member.LastHeartbeat = time.Now().Add(-2 * rs.config.HeartbeatTimeout)

	return nil
}

// LogOperation logs an operation to the oplog (primary only)
func (rs *ReplicaSet) LogOperation(entry *OplogEntry) error {
	rs.mu.RLock()
	if rs.role != RolePrimary {
		rs.mu.RUnlock()
		return fmt.Errorf("only primary can log operations")
	}
	master := rs.master
	rs.mu.RUnlock()

	if master == nil {
		return fmt.Errorf("master not initialized")
	}

	return master.LogOperation(entry)
}

// WaitForReplication waits for operations to replicate to majority
func (rs *ReplicaSet) WaitForReplication(ctx context.Context, opID OpID, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for replication")
		}

		// Count members that have replicated
		replicated := 0
		votingMembers := 0

		rs.membersMu.RLock()
		for _, member := range rs.members {
			member.mu.RLock()
			if member.IsVotingMember {
				votingMembers++
				if member.LastOpID >= opID {
					replicated++
				}
			}
			member.mu.RUnlock()
		}
		rs.membersMu.RUnlock()

		// Need majority
		majority := (votingMembers / 2) + 1
		if replicated >= majority {
			return nil
		}

		time.Sleep(50 * time.Millisecond)
	}
}
