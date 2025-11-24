package distributed

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mnohosten/laura-db/pkg/mvcc"
)

// CoordinatorState represents the state of a 2PC coordinator
type CoordinatorState int

const (
	CoordinatorStateInit CoordinatorState = iota
	CoordinatorStatePreparing
	CoordinatorStateCommitting
	CoordinatorStateAborting
	CoordinatorStateCommitted
	CoordinatorStateAborted
)

// ParticipantState represents the state of a 2PC participant
type ParticipantState int

const (
	ParticipantStateInit ParticipantState = iota
	ParticipantStatePrepared
	ParticipantStateCommitted
	ParticipantStateAborted
)

// ParticipantID uniquely identifies a participant in the 2PC protocol
type ParticipantID string

// Participant represents a resource that participates in 2PC
type Participant interface {
	// Prepare asks the participant to prepare for commit
	// Returns true if ready to commit, false otherwise
	Prepare(ctx context.Context, txnID mvcc.TxnID) (bool, error)

	// Commit tells the participant to commit the transaction
	Commit(ctx context.Context, txnID mvcc.TxnID) error

	// Abort tells the participant to abort the transaction
	Abort(ctx context.Context, txnID mvcc.TxnID) error

	// ID returns the participant's unique identifier
	ID() ParticipantID
}

// participantRecord tracks the state of a participant during 2PC
type participantRecord struct {
	participant Participant
	state       ParticipantState
	prepareVote bool
	mu          sync.RWMutex
}

// Coordinator manages the two-phase commit protocol
type Coordinator struct {
	txnID        mvcc.TxnID
	state        CoordinatorState
	participants map[ParticipantID]*participantRecord
	mu           sync.RWMutex
	timeout      time.Duration
}

// NewCoordinator creates a new 2PC coordinator for a transaction
func NewCoordinator(txnID mvcc.TxnID, timeout time.Duration) *Coordinator {
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	return &Coordinator{
		txnID:        txnID,
		state:        CoordinatorStateInit,
		participants: make(map[ParticipantID]*participantRecord),
		timeout:      timeout,
	}
}

// AddParticipant adds a participant to the transaction
func (c *Coordinator) AddParticipant(participant Participant) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state != CoordinatorStateInit {
		return fmt.Errorf("cannot add participant: coordinator not in init state")
	}

	id := participant.ID()
	if _, exists := c.participants[id]; exists {
		return fmt.Errorf("participant %s already added", id)
	}

	c.participants[id] = &participantRecord{
		participant: participant,
		state:       ParticipantStateInit,
		prepareVote: false,
	}

	return nil
}

// Prepare executes Phase 1 of 2PC: sends prepare requests to all participants
// Returns true if all participants vote YES, false otherwise
func (c *Coordinator) Prepare(ctx context.Context) (bool, error) {
	c.mu.Lock()
	if c.state != CoordinatorStateInit {
		c.mu.Unlock()
		return false, fmt.Errorf("cannot prepare: coordinator not in init state")
	}
	c.state = CoordinatorStatePreparing
	c.mu.Unlock()

	// Create context with timeout
	prepareCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Send prepare to all participants in parallel
	type prepareResult struct {
		participantID ParticipantID
		vote          bool
		err           error
	}

	resultsChan := make(chan prepareResult, len(c.participants))
	var wg sync.WaitGroup

	c.mu.RLock()
	for id, record := range c.participants {
		wg.Add(1)
		go func(pid ParticipantID, rec *participantRecord) {
			defer wg.Done()

			vote, err := rec.participant.Prepare(prepareCtx, c.txnID)

			rec.mu.Lock()
			if err == nil {
				rec.prepareVote = vote
				if vote {
					rec.state = ParticipantStatePrepared
				}
			}
			rec.mu.Unlock()

			resultsChan <- prepareResult{
				participantID: pid,
				vote:          vote,
				err:           err,
			}
		}(id, record)
	}
	c.mu.RUnlock()

	// Wait for all prepare requests to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	allVotedYes := true
	var prepareErrors []error

	for result := range resultsChan {
		if result.err != nil {
			prepareErrors = append(prepareErrors, fmt.Errorf("participant %s: %w", result.participantID, result.err))
			allVotedYes = false
		} else if !result.vote {
			allVotedYes = false
		}
	}

	if len(prepareErrors) > 0 {
		return false, fmt.Errorf("prepare phase failed: %v", prepareErrors)
	}

	return allVotedYes, nil
}

// Commit executes Phase 2 of 2PC: sends commit requests to all participants
// This should only be called if Prepare returned true
func (c *Coordinator) Commit(ctx context.Context) error {
	c.mu.Lock()
	if c.state != CoordinatorStatePreparing {
		c.mu.Unlock()
		return fmt.Errorf("cannot commit: coordinator not in preparing state")
	}
	c.state = CoordinatorStateCommitting
	c.mu.Unlock()

	// Create context with timeout
	commitCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Send commit to all participants in parallel
	type commitResult struct {
		participantID ParticipantID
		err           error
	}

	resultsChan := make(chan commitResult, len(c.participants))
	var wg sync.WaitGroup

	c.mu.RLock()
	for id, record := range c.participants {
		wg.Add(1)
		go func(pid ParticipantID, rec *participantRecord) {
			defer wg.Done()

			err := rec.participant.Commit(commitCtx, c.txnID)

			rec.mu.Lock()
			if err == nil {
				rec.state = ParticipantStateCommitted
			}
			rec.mu.Unlock()

			resultsChan <- commitResult{
				participantID: pid,
				err:           err,
			}
		}(id, record)
	}
	c.mu.RUnlock()

	// Wait for all commit requests to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	var commitErrors []error
	for result := range resultsChan {
		if result.err != nil {
			commitErrors = append(commitErrors, fmt.Errorf("participant %s: %w", result.participantID, result.err))
		}
	}

	c.mu.Lock()
	if len(commitErrors) > 0 {
		c.state = CoordinatorStateAborted
		c.mu.Unlock()
		return fmt.Errorf("commit phase failed: %v", commitErrors)
	}

	c.state = CoordinatorStateCommitted
	c.mu.Unlock()

	return nil
}

// Abort executes the abort protocol: sends abort requests to all participants
func (c *Coordinator) Abort(ctx context.Context) error {
	c.mu.Lock()
	if c.state == CoordinatorStateCommitted {
		c.mu.Unlock()
		return fmt.Errorf("cannot abort: transaction already committed")
	}
	c.state = CoordinatorStateAborting
	c.mu.Unlock()

	// Create context with timeout
	abortCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Send abort to all participants in parallel
	type abortResult struct {
		participantID ParticipantID
		err           error
	}

	resultsChan := make(chan abortResult, len(c.participants))
	var wg sync.WaitGroup

	c.mu.RLock()
	for id, record := range c.participants {
		wg.Add(1)
		go func(pid ParticipantID, rec *participantRecord) {
			defer wg.Done()

			err := rec.participant.Abort(abortCtx, c.txnID)

			rec.mu.Lock()
			rec.state = ParticipantStateAborted
			rec.mu.Unlock()

			resultsChan <- abortResult{
				participantID: pid,
				err:           err,
			}
		}(id, record)
	}
	c.mu.RUnlock()

	// Wait for all abort requests to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results (errors during abort are logged but not critical)
	var abortErrors []error
	for result := range resultsChan {
		if result.err != nil {
			abortErrors = append(abortErrors, fmt.Errorf("participant %s: %w", result.participantID, result.err))
		}
	}

	c.mu.Lock()
	c.state = CoordinatorStateAborted
	c.mu.Unlock()

	if len(abortErrors) > 0 {
		return fmt.Errorf("abort phase had errors (non-critical): %v", abortErrors)
	}

	return nil
}

// Execute runs the full 2PC protocol: prepare, then commit or abort
func (c *Coordinator) Execute(ctx context.Context) error {
	// Phase 1: Prepare
	allPrepared, err := c.Prepare(ctx)
	if err != nil {
		// Prepare failed, abort
		_ = c.Abort(ctx)
		return fmt.Errorf("prepare failed: %w", err)
	}

	if !allPrepared {
		// Not all participants voted YES, abort
		_ = c.Abort(ctx)
		return fmt.Errorf("not all participants voted YES to prepare")
	}

	// Phase 2: Commit
	if err := c.Commit(ctx); err != nil {
		// Commit failed - this is problematic as some may have committed
		// In a real system, we'd log this and retry or use recovery protocols
		return fmt.Errorf("commit failed: %w", err)
	}

	return nil
}

// GetState returns the current state of the coordinator
func (c *Coordinator) GetState() CoordinatorState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// GetParticipantState returns the state of a specific participant
func (c *Coordinator) GetParticipantState(id ParticipantID) (ParticipantState, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	record, exists := c.participants[id]
	if !exists {
		return ParticipantStateInit, fmt.Errorf("participant %s not found", id)
	}

	record.mu.RLock()
	defer record.mu.RUnlock()
	return record.state, nil
}

// GetParticipantCount returns the number of participants
func (c *Coordinator) GetParticipantCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.participants)
}
