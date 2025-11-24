package lsm

import "errors"

var (
	// ErrInvalidBloomFilter is returned when bloom filter data is invalid
	ErrInvalidBloomFilter = errors.New("invalid bloom filter data")

	// ErrKeyNotFound is returned when a key is not found
	ErrKeyNotFound = errors.New("key not found")

	// ErrClosed is returned when operation is attempted on closed LSM tree
	ErrClosed = errors.New("lsm tree is closed")
)
