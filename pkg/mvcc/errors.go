package mvcc

import "errors"

var (
	// ErrTransactionNotActive is returned when operating on a non-active transaction
	ErrTransactionNotActive = errors.New("transaction is not active")

	// ErrConflict is returned when a write conflict is detected
	ErrConflict = errors.New("write conflict detected")

	// ErrKeyNotFound is returned when a key doesn't exist
	ErrKeyNotFound = errors.New("key not found")
)
