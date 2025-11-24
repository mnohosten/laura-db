package distributed

import (
	"context"
	"fmt"
	"sync"

	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/mvcc"
)

// DatabaseParticipant implements the Participant interface for a LauraDB database
// It allows a database to participate in distributed 2PC transactions
type DatabaseParticipant struct {
	id       ParticipantID
	db       *database.Database
	sessions map[mvcc.TxnID]*database.Session // Active sessions by transaction ID
	mu       sync.RWMutex
}

// NewDatabaseParticipant creates a new database participant for 2PC
func NewDatabaseParticipant(id string, db *database.Database) *DatabaseParticipant {
	return &DatabaseParticipant{
		id:       ParticipantID(id),
		db:       db,
		sessions: make(map[mvcc.TxnID]*database.Session),
	}
}

// ID returns the participant's unique identifier
func (dp *DatabaseParticipant) ID() ParticipantID {
	return dp.id
}

// StartTransaction starts a new transaction session for the given transaction ID
// This should be called before any operations are performed
func (dp *DatabaseParticipant) StartTransaction(txnID mvcc.TxnID) *database.Session {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	session := dp.db.StartSession()
	dp.sessions[txnID] = session
	return session
}

// GetSession returns the session for a given transaction ID
func (dp *DatabaseParticipant) GetSession(txnID mvcc.TxnID) (*database.Session, error) {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	session, exists := dp.sessions[txnID]
	if !exists {
		return nil, fmt.Errorf("no session found for transaction %d", txnID)
	}
	return session, nil
}

// Prepare implements Phase 1 of 2PC
// It validates that the transaction can be committed
func (dp *DatabaseParticipant) Prepare(ctx context.Context, txnID mvcc.TxnID) (bool, error) {
	dp.mu.RLock()
	session, exists := dp.sessions[txnID]
	dp.mu.RUnlock()

	if !exists {
		return false, fmt.Errorf("no session found for transaction %d", txnID)
	}

	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	// Validate the transaction can be committed
	// For LauraDB, we check:
	// 1. Transaction is still active
	// 2. No conflicts detected so far
	txn := session.Transaction()
	if txn.State != mvcc.TxnStateActive {
		return false, fmt.Errorf("transaction %d is not active", txnID)
	}

	// Vote YES to prepare - we're ready to commit
	// The actual conflict detection will happen during commit
	return true, nil
}

// Commit implements Phase 2 of 2PC for the commit path
// It commits the transaction to the database
func (dp *DatabaseParticipant) Commit(ctx context.Context, txnID mvcc.TxnID) error {
	dp.mu.Lock()
	session, exists := dp.sessions[txnID]
	if !exists {
		dp.mu.Unlock()
		return fmt.Errorf("no session found for transaction %d", txnID)
	}
	// Remove session from map after commit
	delete(dp.sessions, txnID)
	dp.mu.Unlock()

	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Commit the session
	if err := session.CommitTransaction(); err != nil {
		return fmt.Errorf("failed to commit transaction %d: %w", txnID, err)
	}

	return nil
}

// Abort implements Phase 2 of 2PC for the abort path
// It aborts the transaction
func (dp *DatabaseParticipant) Abort(ctx context.Context, txnID mvcc.TxnID) error {
	dp.mu.Lock()
	session, exists := dp.sessions[txnID]
	if !exists {
		dp.mu.Unlock()
		return fmt.Errorf("no session found for transaction %d", txnID)
	}
	// Remove session from map after abort
	delete(dp.sessions, txnID)
	dp.mu.Unlock()

	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Abort the session
	if err := session.AbortTransaction(); err != nil {
		return fmt.Errorf("failed to abort transaction %d: %w", txnID, err)
	}

	return nil
}

// GetActiveSessionCount returns the number of active sessions
func (dp *DatabaseParticipant) GetActiveSessionCount() int {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	return len(dp.sessions)
}
