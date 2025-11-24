package replication

import (
	"context"
	"fmt"
	"time"
)

// WriteConcern specifies the level of write durability required
type WriteConcern struct {
	// W specifies the number of nodes that must acknowledge a write
	// Special values:
	//   0: No acknowledgment (fire and forget)
	//   1: Primary only (default)
	//   majority: Majority of voting members
	//   N (>1): Wait for N nodes to acknowledge
	W interface{}

	// WTimeout specifies how long to wait for acknowledgment
	// 0 means wait indefinitely (default)
	WTimeout time.Duration

	// J specifies whether to wait for journal/oplog sync
	// true: Wait for write to be persisted to oplog
	// false: Just wait for in-memory acknowledgment (default)
	J bool
}

// DefaultWriteConcern returns the default write concern (w:1, no timeout)
func DefaultWriteConcern() *WriteConcern {
	return &WriteConcern{
		W:        1,
		WTimeout: 0,
		J:        false,
	}
}

// MajorityWriteConcern returns write concern requiring majority acknowledgment
func MajorityWriteConcern() *WriteConcern {
	return &WriteConcern{
		W:        "majority",
		WTimeout: 0,
		J:        false,
	}
}

// UnacknowledgedWriteConcern returns fire-and-forget write concern (w:0)
func UnacknowledgedWriteConcern() *WriteConcern {
	return &WriteConcern{
		W:        0,
		WTimeout: 0,
		J:        false,
	}
}

// W1WriteConcern returns write concern for primary-only acknowledgment
func W1WriteConcern() *WriteConcern {
	return &WriteConcern{
		W:        1,
		WTimeout: 0,
		J:        false,
	}
}

// W2WriteConcern returns write concern for 2-node acknowledgment
func W2WriteConcern() *WriteConcern {
	return &WriteConcern{
		W:        2,
		WTimeout: 0,
		J:        false,
	}
}

// W3WriteConcern returns write concern for 3-node acknowledgment
func W3WriteConcern() *WriteConcern {
	return &WriteConcern{
		W:        3,
		WTimeout: 0,
		J:        false,
	}
}

// WithTimeout returns a copy of the write concern with specified timeout
func (wc *WriteConcern) WithTimeout(timeout time.Duration) *WriteConcern {
	return &WriteConcern{
		W:        wc.W,
		WTimeout: timeout,
		J:        wc.J,
	}
}

// WithJournal returns a copy of the write concern with journal sync enabled
func (wc *WriteConcern) WithJournal(j bool) *WriteConcern {
	return &WriteConcern{
		W:        wc.W,
		WTimeout: wc.WTimeout,
		J:        j,
	}
}

// GetRequiredAcknowledgments calculates the number of required acknowledgments
// Returns the number and whether it's majority-based
func (wc *WriteConcern) GetRequiredAcknowledgments(totalVotingMembers int) (int, bool, error) {
	switch v := wc.W.(type) {
	case int:
		if v < 0 {
			return 0, false, fmt.Errorf("invalid w value: %d (must be >= 0)", v)
		}
		if v > totalVotingMembers {
			return 0, false, fmt.Errorf("w value %d exceeds total voting members %d", v, totalVotingMembers)
		}
		return v, false, nil
	case string:
		if v == "majority" {
			majority := (totalVotingMembers / 2) + 1
			return majority, true, nil
		}
		return 0, false, fmt.Errorf("invalid w value: %s (must be int or 'majority')", v)
	default:
		return 0, false, fmt.Errorf("invalid w type: %T", v)
	}
}

// Validate validates the write concern configuration
func (wc *WriteConcern) Validate() error {
	if wc == nil {
		return fmt.Errorf("write concern cannot be nil")
	}

	switch v := wc.W.(type) {
	case int:
		if v < 0 {
			return fmt.Errorf("invalid w value: %d (must be >= 0)", v)
		}
	case string:
		if v != "majority" {
			return fmt.Errorf("invalid w value: %s (must be int or 'majority')", v)
		}
	default:
		return fmt.Errorf("invalid w type: %T (must be int or string)", v)
	}

	if wc.WTimeout < 0 {
		return fmt.Errorf("invalid wtimeout: %v (must be >= 0)", wc.WTimeout)
	}

	return nil
}

// String returns a string representation of the write concern
func (wc *WriteConcern) String() string {
	j := "false"
	if wc.J {
		j = "true"
	}

	timeout := "none"
	if wc.WTimeout > 0 {
		timeout = wc.WTimeout.String()
	}

	return fmt.Sprintf("{w:%v, wtimeout:%s, j:%s}", wc.W, timeout, j)
}

// IsAcknowledged returns true if the write concern requires acknowledgment
func (wc *WriteConcern) IsAcknowledged() bool {
	if intVal, ok := wc.W.(int); ok {
		return intVal > 0
	}
	return true // "majority" is always acknowledged
}

// RequiresJournal returns true if journal sync is required
func (wc *WriteConcern) RequiresJournal() bool {
	return wc.J
}

// GetTimeout returns the timeout duration
func (wc *WriteConcern) GetTimeout() time.Duration {
	return wc.WTimeout
}

// WriteResult contains the result of a write operation with write concern
type WriteResult struct {
	// Acknowledged indicates if the write was acknowledged
	Acknowledged bool

	// OpID is the operation ID in the oplog
	OpID OpID

	// NodesAcknowledged is the number of nodes that acknowledged
	NodesAcknowledged int

	// NodesRequired is the number of nodes required by write concern
	NodesRequired int

	// JournalSynced indicates if the write was synced to journal/oplog
	JournalSynced bool

	// ElapsedTime is how long the write took
	ElapsedTime time.Duration
}

// String returns a string representation of the write result
func (wr *WriteResult) String() string {
	return fmt.Sprintf(
		"{acked:%v, opid:%d, nodes:%d/%d, journal:%v, time:%v}",
		wr.Acknowledged,
		wr.OpID,
		wr.NodesAcknowledged,
		wr.NodesRequired,
		wr.JournalSynced,
		wr.ElapsedTime,
	)
}

// WriteWithConcern performs a write operation and waits for the specified write concern
func (rs *ReplicaSet) WriteWithConcern(ctx context.Context, entry *OplogEntry, wc *WriteConcern) (*WriteResult, error) {
	startTime := time.Now()

	// Validate write concern
	if err := wc.Validate(); err != nil {
		return nil, fmt.Errorf("invalid write concern: %w", err)
	}

	// Only primary can accept writes
	if !rs.IsPrimary() {
		return nil, fmt.Errorf("not primary")
	}

	// Log the operation
	if err := rs.LogOperation(entry); err != nil {
		return nil, fmt.Errorf("failed to log operation: %w", err)
	}

	// Get the current opID from the master (not from rs.oplog, as they may be different instances)
	rs.mu.RLock()
	master := rs.master
	rs.mu.RUnlock()

	if master == nil {
		return nil, fmt.Errorf("master not initialized")
	}

	opID := master.GetCurrentOpID()

	// Update primary's own LastOpID
	rs.membersMu.RLock()
	if primaryMember, exists := rs.members[rs.config.NodeID]; exists {
		primaryMember.mu.Lock()
		primaryMember.LastOpID = opID
		primaryMember.mu.Unlock()
	}
	rs.membersMu.RUnlock()

	// If w:0 (unacknowledged), return immediately
	if !wc.IsAcknowledged() {
		return &WriteResult{
			Acknowledged:      false,
			OpID:              opID,
			NodesAcknowledged: 0,
			NodesRequired:     0,
			JournalSynced:     false,
			ElapsedTime:       time.Since(startTime),
		}, nil
	}

	// Calculate required acknowledgments
	totalVotingMembers := rs.countVotingMembers()
	required, isMajority, err := wc.GetRequiredAcknowledgments(totalVotingMembers)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate required acknowledgments: %w", err)
	}

	// If w:1 and no journal required, return immediately (primary already has it)
	if required == 1 && !wc.RequiresJournal() {
		return &WriteResult{
			Acknowledged:      true,
			OpID:              opID,
			NodesAcknowledged: 1,
			NodesRequired:     1,
			JournalSynced:     false,
			ElapsedTime:       time.Since(startTime),
		}, nil
	}

	// Set up context with timeout if specified
	waitCtx := ctx
	if wc.WTimeout > 0 {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, wc.WTimeout)
		defer cancel()
	}

	// Wait for replication to required nodes
	if required > 1 {
		if err := rs.waitForReplicationToNodes(waitCtx, opID, required); err != nil {
			// Return partial success info on timeout/error
			acknowledged := rs.countAcknowledgedNodes(opID)
			return &WriteResult{
				Acknowledged:      false,
				OpID:              opID,
				NodesAcknowledged: acknowledged,
				NodesRequired:     required,
				JournalSynced:     false,
				ElapsedTime:       time.Since(startTime),
			}, fmt.Errorf("replication failed: %w", err)
		}
	}

	// If journal sync is required, wait for it
	journalSynced := false
	if wc.RequiresJournal() {
		// In a real implementation, would wait for fsync
		// For now, we assume oplog writes are immediately persisted
		journalSynced = true
	}

	acknowledged := rs.countAcknowledgedNodes(opID)

	result := &WriteResult{
		Acknowledged:      true,
		OpID:              opID,
		NodesAcknowledged: acknowledged,
		NodesRequired:     required,
		JournalSynced:     journalSynced,
		ElapsedTime:       time.Since(startTime),
	}

	// Extra validation for majority writes
	if isMajority && acknowledged < required {
		return result, fmt.Errorf("majority write concern not satisfied: got %d, needed %d", acknowledged, required)
	}

	return result, nil
}

// waitForReplicationToNodes waits for replication to the specified number of nodes
func (rs *ReplicaSet) waitForReplicationToNodes(ctx context.Context, opID OpID, required int) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			acknowledged := rs.countAcknowledgedNodes(opID)
			if acknowledged >= required {
				return nil
			}
		}
	}
}

// countAcknowledgedNodes counts how many nodes have replicated the operation
func (rs *ReplicaSet) countAcknowledgedNodes(opID OpID) int {
	rs.membersMu.RLock()
	defer rs.membersMu.RUnlock()

	count := 0
	for _, member := range rs.members {
		member.mu.RLock()
		if member.IsVotingMember && member.LastOpID >= opID {
			count++
		}
		member.mu.RUnlock()
	}

	return count
}
