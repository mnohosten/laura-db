package database

import "errors"

var (
	// ErrDocumentNotFound is returned when a document is not found
	ErrDocumentNotFound = errors.New("document not found")

	// ErrCollectionNotFound is returned when a collection is not found
	ErrCollectionNotFound = errors.New("collection not found")

	// ErrDatabaseClosed is returned when operating on a closed database
	ErrDatabaseClosed = errors.New("database is closed")
)
