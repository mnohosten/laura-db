package distributed

import "errors"

var (
	// ErrCoordinatorNotInit is returned when trying to perform an operation on a coordinator not in init state
	ErrCoordinatorNotInit = errors.New("coordinator not in init state")

	// ErrCoordinatorNotPreparing is returned when trying to commit without preparing first
	ErrCoordinatorNotPreparing = errors.New("coordinator not in preparing state")

	// ErrAlreadyCommitted is returned when trying to abort an already committed transaction
	ErrAlreadyCommitted = errors.New("transaction already committed")

	// ErrParticipantNotFound is returned when a participant ID is not found
	ErrParticipantNotFound = errors.New("participant not found")

	// ErrParticipantAlreadyAdded is returned when trying to add a duplicate participant
	ErrParticipantAlreadyAdded = errors.New("participant already added")

	// ErrPrepareFailed is returned when the prepare phase fails
	ErrPrepareFailed = errors.New("prepare phase failed")

	// ErrNotAllPrepared is returned when not all participants vote YES
	ErrNotAllPrepared = errors.New("not all participants voted YES to prepare")

	// ErrCommitFailed is returned when the commit phase fails
	ErrCommitFailed = errors.New("commit phase failed")

	// ErrSessionNotFound is returned when a transaction session is not found
	ErrSessionNotFound = errors.New("session not found for transaction")

	// ErrTransactionNotActive is returned when trying to prepare/commit/abort an inactive transaction
	ErrTransactionNotActive = errors.New("transaction not active")
)
