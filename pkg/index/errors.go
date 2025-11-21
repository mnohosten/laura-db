package index

import "errors"

var (
	// ErrDuplicateKey is returned when inserting a duplicate key in a unique index
	ErrDuplicateKey = errors.New("duplicate key")

	// ErrKeyNotFound is returned when a key is not found
	ErrKeyNotFound = errors.New("key not found")

	// ErrInvalidOrder is returned when B-tree order is invalid
	ErrInvalidOrder = errors.New("invalid B-tree order")
)
