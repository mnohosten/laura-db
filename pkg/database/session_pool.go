package database

import (
	"sync"

	"github.com/mnohosten/laura-db/pkg/document"
)

// SessionPool manages a pool of reusable Session objects
// to reduce allocation overhead for transactional operations.
// Uses sync.Pool for efficient session reuse.
type SessionPool struct {
	db   *Database
	pool sync.Pool
}

// NewSessionPool creates a new session pool for the given database
func NewSessionPool(db *Database) *SessionPool {
	sp := &SessionPool{
		db: db,
	}

	sp.pool.New = func() interface{} {
		return &Session{
			db:           db,
			operations:   make([]sessionOperation, 0, 16), // Pre-allocate capacity
			collections:  make(map[string]bool),
			snapshotDocs: make(map[string]map[string]*document.Document),
		}
	}

	return sp
}

// Get retrieves a Session from the pool or creates a new one
// The session will have a fresh MVCC transaction started
func (sp *SessionPool) Get() *Session {
	s := sp.pool.Get().(*Session)

	// Start a new MVCC transaction
	s.txn = sp.db.txnMgr.Begin()

	return s
}

// Put returns a Session to the pool after resetting its state
// This should be called after CommitTransaction or AbortTransaction
func (sp *SessionPool) Put(s *Session) {
	if s == nil {
		return
	}

	// Reset the session state for reuse
	s.reset()

	// Return to pool
	sp.pool.Put(s)
}

// reset clears the session state so it can be reused
func (s *Session) reset() {
	// Clear transaction (will be set when retrieved from pool)
	s.txn = nil

	// Reuse slices/maps by clearing them (avoids allocation)
	s.operations = s.operations[:0]

	// Clear collections map
	for k := range s.collections {
		delete(s.collections, k)
	}

	// Clear snapshot cache
	for k := range s.snapshotDocs {
		delete(s.snapshotDocs, k)
	}
}

// WithTransactionPooled executes a function within a pooled transaction session
// Automatically handles session lifecycle: Get from pool, commit/abort, and return to pool
// If the function returns an error, the transaction is aborted
// Otherwise, the transaction is committed
func (sp *SessionPool) WithTransactionPooled(fn func(session *Session) error) error {
	// Get session from pool
	session := sp.Get()
	defer sp.Put(session) // Always return to pool

	// Execute the function
	err := fn(session)
	if err != nil {
		// Abort the transaction on error
		if abortErr := session.AbortTransaction(); abortErr != nil {
			// Still return to pool even if abort fails
			return wrapError(err, "abort error: %v", abortErr)
		}
		return err
	}

	// Commit the transaction
	if commitErr := session.CommitTransaction(); commitErr != nil {
		// Try to abort if commit fails
		session.AbortTransaction()
		return wrapError(commitErr, "failed to commit transaction")
	}

	return nil
}

// wrapError creates a formatted error with context
func wrapError(err error, format string, args ...interface{}) error {
	if len(args) > 0 {
		return &poolError{
			err:     err,
			message: format,
			args:    args,
		}
	}
	return &poolError{
		err:     err,
		message: format,
	}
}

type poolError struct {
	err     error
	message string
	args    []interface{}
}

func (e *poolError) Error() string {
	if len(e.args) > 0 {
		return e.err.Error() + ", " + e.message
	}
	return e.message + ": " + e.err.Error()
}

func (e *poolError) Unwrap() error {
	return e.err
}
