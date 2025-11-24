package distributed

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mnohosten/laura-db/pkg/mvcc"
)

// MockParticipant is a mock implementation of the Participant interface for testing
type MockParticipant struct {
	id              ParticipantID
	prepareResponse bool
	prepareError    error
	commitError     error
	abortError      error
	prepareDelay    time.Duration
	commitDelay     time.Duration
	abortDelay      time.Duration
	prepareCalled   int
	commitCalled    int
	abortCalled     int
	mu              sync.Mutex
}

func NewMockParticipant(id string) *MockParticipant {
	return &MockParticipant{
		id:              ParticipantID(id),
		prepareResponse: true,
	}
}

func (m *MockParticipant) ID() ParticipantID {
	return m.id
}

func (m *MockParticipant) Prepare(ctx context.Context, txnID mvcc.TxnID) (bool, error) {
	m.mu.Lock()
	m.prepareCalled++
	delay := m.prepareDelay
	resp := m.prepareResponse
	err := m.prepareError
	m.mu.Unlock()

	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}

	return resp, err
}

func (m *MockParticipant) Commit(ctx context.Context, txnID mvcc.TxnID) error {
	m.mu.Lock()
	m.commitCalled++
	delay := m.commitDelay
	err := m.commitError
	m.mu.Unlock()

	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return err
}

func (m *MockParticipant) Abort(ctx context.Context, txnID mvcc.TxnID) error {
	m.mu.Lock()
	m.abortCalled++
	delay := m.abortDelay
	err := m.abortError
	m.mu.Unlock()

	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return err
}

func (m *MockParticipant) GetCallCounts() (prepare, commit, abort int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.prepareCalled, m.commitCalled, m.abortCalled
}

// TestCoordinatorBasic tests basic coordinator creation and state
func TestCoordinatorBasic(t *testing.T) {
	coord := NewCoordinator(1, 5*time.Second)

	if coord.txnID != 1 {
		t.Errorf("expected txnID 1, got %d", coord.txnID)
	}

	if coord.GetState() != CoordinatorStateInit {
		t.Errorf("expected state Init, got %v", coord.GetState())
	}

	if coord.GetParticipantCount() != 0 {
		t.Errorf("expected 0 participants, got %d", coord.GetParticipantCount())
	}
}

// TestAddParticipants tests adding participants to a coordinator
func TestAddParticipants(t *testing.T) {
	coord := NewCoordinator(1, 5*time.Second)

	p1 := NewMockParticipant("p1")
	p2 := NewMockParticipant("p2")

	if err := coord.AddParticipant(p1); err != nil {
		t.Fatalf("failed to add participant 1: %v", err)
	}

	if err := coord.AddParticipant(p2); err != nil {
		t.Fatalf("failed to add participant 2: %v", err)
	}

	if coord.GetParticipantCount() != 2 {
		t.Errorf("expected 2 participants, got %d", coord.GetParticipantCount())
	}

	// Try to add duplicate
	if err := coord.AddParticipant(p1); err == nil {
		t.Error("expected error when adding duplicate participant")
	}
}

// TestSuccessfulCommit tests a successful 2PC commit
func TestSuccessfulCommit(t *testing.T) {
	coord := NewCoordinator(1, 5*time.Second)

	p1 := NewMockParticipant("p1")
	p2 := NewMockParticipant("p2")
	p3 := NewMockParticipant("p3")

	coord.AddParticipant(p1)
	coord.AddParticipant(p2)
	coord.AddParticipant(p3)

	ctx := context.Background()

	// Execute the 2PC protocol
	if err := coord.Execute(ctx); err != nil {
		t.Fatalf("2PC execution failed: %v", err)
	}

	// Verify final state
	if coord.GetState() != CoordinatorStateCommitted {
		t.Errorf("expected state Committed, got %v", coord.GetState())
	}

	// Verify all participants were called
	for _, p := range []*MockParticipant{p1, p2, p3} {
		prep, comm, abrt := p.GetCallCounts()
		if prep != 1 {
			t.Errorf("participant %s: expected 1 prepare call, got %d", p.ID(), prep)
		}
		if comm != 1 {
			t.Errorf("participant %s: expected 1 commit call, got %d", p.ID(), comm)
		}
		if abrt != 0 {
			t.Errorf("participant %s: expected 0 abort calls, got %d", p.ID(), abrt)
		}
	}
}

// TestAbortOnPrepareFailure tests that abort is called when prepare fails
func TestAbortOnPrepareFailure(t *testing.T) {
	coord := NewCoordinator(1, 5*time.Second)

	p1 := NewMockParticipant("p1")
	p2 := NewMockParticipant("p2")
	p2.prepareResponse = false // p2 votes NO

	coord.AddParticipant(p1)
	coord.AddParticipant(p2)

	ctx := context.Background()

	// Execute the 2PC protocol
	err := coord.Execute(ctx)
	if err == nil {
		t.Fatal("expected error when participant votes NO")
	}

	// Verify final state
	if coord.GetState() != CoordinatorStateAborted {
		t.Errorf("expected state Aborted, got %v", coord.GetState())
	}

	// Verify abort was called on all participants
	for _, p := range []*MockParticipant{p1, p2} {
		prep, comm, abrt := p.GetCallCounts()
		if prep != 1 {
			t.Errorf("participant %s: expected 1 prepare call, got %d", p.ID(), prep)
		}
		if comm != 0 {
			t.Errorf("participant %s: expected 0 commit calls, got %d", p.ID(), comm)
		}
		if abrt != 1 {
			t.Errorf("participant %s: expected 1 abort call, got %d", p.ID(), abrt)
		}
	}
}

// TestAbortOnPrepareError tests that abort is called when prepare returns an error
func TestAbortOnPrepareError(t *testing.T) {
	coord := NewCoordinator(1, 5*time.Second)

	p1 := NewMockParticipant("p1")
	p2 := NewMockParticipant("p2")
	p2.prepareError = errors.New("prepare error")

	coord.AddParticipant(p1)
	coord.AddParticipant(p2)

	ctx := context.Background()

	// Execute the 2PC protocol
	err := coord.Execute(ctx)
	if err == nil {
		t.Fatal("expected error when prepare fails")
	}

	// Verify final state
	if coord.GetState() != CoordinatorStateAborted {
		t.Errorf("expected state Aborted, got %v", coord.GetState())
	}

	// Verify abort was called on all participants
	for _, p := range []*MockParticipant{p1, p2} {
		_, _, abrt := p.GetCallCounts()
		if abrt != 1 {
			t.Errorf("participant %s: expected 1 abort call, got %d", p.ID(), abrt)
		}
	}
}

// TestCommitError tests handling of errors during commit phase
func TestCommitError(t *testing.T) {
	coord := NewCoordinator(1, 5*time.Second)

	p1 := NewMockParticipant("p1")
	p2 := NewMockParticipant("p2")
	p2.commitError = errors.New("commit error")

	coord.AddParticipant(p1)
	coord.AddParticipant(p2)

	ctx := context.Background()

	// Execute the 2PC protocol
	err := coord.Execute(ctx)
	if err == nil {
		t.Fatal("expected error when commit fails")
	}

	// Verify final state
	if coord.GetState() != CoordinatorStateAborted {
		t.Errorf("expected state Aborted, got %v", coord.GetState())
	}
}

// TestPrepareTimeout tests timeout during prepare phase
func TestPrepareTimeout(t *testing.T) {
	coord := NewCoordinator(1, 100*time.Millisecond)

	p1 := NewMockParticipant("p1")
	p2 := NewMockParticipant("p2")
	p2.prepareDelay = 200 * time.Millisecond // Delay longer than timeout

	coord.AddParticipant(p1)
	coord.AddParticipant(p2)

	ctx := context.Background()

	// Execute the 2PC protocol
	err := coord.Execute(ctx)
	if err == nil {
		t.Fatal("expected timeout error")
	}

	// Verify final state
	if coord.GetState() != CoordinatorStateAborted {
		t.Errorf("expected state Aborted, got %v", coord.GetState())
	}
}

// TestManualPrepareAndCommit tests manual prepare and commit calls
func TestManualPrepareAndCommit(t *testing.T) {
	coord := NewCoordinator(1, 5*time.Second)

	p1 := NewMockParticipant("p1")
	p2 := NewMockParticipant("p2")

	coord.AddParticipant(p1)
	coord.AddParticipant(p2)

	ctx := context.Background()

	// Manual prepare
	allPrepared, err := coord.Prepare(ctx)
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	if !allPrepared {
		t.Fatal("expected all participants to vote YES")
	}

	if coord.GetState() != CoordinatorStatePreparing {
		t.Errorf("expected state Preparing, got %v", coord.GetState())
	}

	// Manual commit
	if err := coord.Commit(ctx); err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	if coord.GetState() != CoordinatorStateCommitted {
		t.Errorf("expected state Committed, got %v", coord.GetState())
	}
}

// TestManualAbort tests manual abort call
func TestManualAbort(t *testing.T) {
	coord := NewCoordinator(1, 5*time.Second)

	p1 := NewMockParticipant("p1")
	p2 := NewMockParticipant("p2")

	coord.AddParticipant(p1)
	coord.AddParticipant(p2)

	ctx := context.Background()

	// Prepare
	coord.Prepare(ctx)

	// Manual abort
	if err := coord.Abort(ctx); err != nil {
		t.Fatalf("abort failed: %v", err)
	}

	if coord.GetState() != CoordinatorStateAborted {
		t.Errorf("expected state Aborted, got %v", coord.GetState())
	}

	// Verify abort was called on all participants
	for _, p := range []*MockParticipant{p1, p2} {
		_, _, abrt := p.GetCallCounts()
		if abrt != 1 {
			t.Errorf("participant %s: expected 1 abort call, got %d", p.ID(), abrt)
		}
	}
}

// TestGetParticipantState tests retrieving participant state
func TestGetParticipantState(t *testing.T) {
	coord := NewCoordinator(1, 5*time.Second)

	p1 := NewMockParticipant("p1")
	coord.AddParticipant(p1)

	// Check initial state
	state, err := coord.GetParticipantState(p1.ID())
	if err != nil {
		t.Fatalf("failed to get participant state: %v", err)
	}
	if state != ParticipantStateInit {
		t.Errorf("expected state Init, got %v", state)
	}

	// After prepare
	ctx := context.Background()
	coord.Prepare(ctx)

	state, err = coord.GetParticipantState(p1.ID())
	if err != nil {
		t.Fatalf("failed to get participant state: %v", err)
	}
	if state != ParticipantStatePrepared {
		t.Errorf("expected state Prepared, got %v", state)
	}

	// After commit
	coord.Commit(ctx)

	state, err = coord.GetParticipantState(p1.ID())
	if err != nil {
		t.Fatalf("failed to get participant state: %v", err)
	}
	if state != ParticipantStateCommitted {
		t.Errorf("expected state Committed, got %v", state)
	}
}

// TestContextCancellation tests that context cancellation is respected
func TestContextCancellation(t *testing.T) {
	coord := NewCoordinator(1, 30*time.Second)

	p1 := NewMockParticipant("p1")
	p1.prepareDelay = 1 * time.Second

	coord.AddParticipant(p1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Execute should fail due to cancelled context
	err := coord.Execute(ctx)
	if err == nil {
		t.Fatal("expected error due to cancelled context")
	}
}

// TestConcurrentParticipants tests handling of many participants in parallel
func TestConcurrentParticipants(t *testing.T) {
	coord := NewCoordinator(1, 10*time.Second)

	// Add 100 participants
	numParticipants := 100
	for i := 0; i < numParticipants; i++ {
		p := NewMockParticipant(string(rune('a' + i%26)))
		p.prepareDelay = time.Duration(i%10) * time.Millisecond
		coord.AddParticipant(p)
	}

	ctx := context.Background()

	start := time.Now()
	if err := coord.Execute(ctx); err != nil {
		t.Fatalf("2PC execution failed: %v", err)
	}
	duration := time.Since(start)

	// With parallelism, should complete faster than sequential execution
	// Sequential would be ~900ms (sum of delays), parallel should be <100ms
	if duration > 500*time.Millisecond {
		t.Errorf("execution took too long: %v (expected parallel execution)", duration)
	}

	if coord.GetState() != CoordinatorStateCommitted {
		t.Errorf("expected state Committed, got %v", coord.GetState())
	}
}

// TestAbortAfterCommit tests that abort fails after commit
func TestAbortAfterCommit(t *testing.T) {
	coord := NewCoordinator(1, 5*time.Second)

	p1 := NewMockParticipant("p1")
	coord.AddParticipant(p1)

	ctx := context.Background()

	// Execute successful commit
	coord.Execute(ctx)

	// Try to abort
	err := coord.Abort(ctx)
	if err == nil {
		t.Fatal("expected error when aborting committed transaction")
	}
}
