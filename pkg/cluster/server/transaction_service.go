package server

import (
	"context"
	"fmt"
	"sync"

	pb "github.com/mnohosten/laura-db/pkg/cluster/proto"
)

// TransactionServiceImpl implements the TransactionService gRPC service
type TransactionServiceImpl struct {
	pb.UnimplementedTransactionServiceServer

	mu           sync.RWMutex
	transactions map[string]*pb.Transaction // txnID -> Transaction
}

// NewTransactionServiceImpl creates a new transaction service implementation
func NewTransactionServiceImpl() *TransactionServiceImpl {
	return &TransactionServiceImpl{
		transactions: make(map[string]*pb.Transaction),
	}
}

// Prepare handles the prepare phase of 2PC
func (s *TransactionServiceImpl) Prepare(ctx context.Context, req *pb.PrepareRequest) (*pb.PrepareResponse, error) {
	// TODO: Implement prepare logic
	// This should:
	// 1. Validate all operations can be performed
	// 2. Acquire necessary locks
	// 3. Prepare transaction state
	// 4. Vote yes/no on whether to commit

	fmt.Printf("Prepare request: txn_id=%s coordinator=%s operations=%d\n",
		req.TxnId, req.CoordinatorId, len(req.Operations))

	s.mu.Lock()
	s.transactions[req.TxnId] = &pb.Transaction{
		TxnId:        req.TxnId,
		Timestamp:    0,
		Participants: []string{},
		State:        pb.TransactionState_PREPARING,
	}
	s.mu.Unlock()

	// For now, always vote to prepare
	return &pb.PrepareResponse{
		TxnId:         req.TxnId,
		ParticipantId: "local-node",
		Vote:          true,
	}, nil
}

// Commit handles the commit phase of 2PC
func (s *TransactionServiceImpl) Commit(ctx context.Context, req *pb.CommitRequest) (*pb.CommitResponse, error) {
	// TODO: Implement commit logic
	// This should:
	// 1. Apply prepared changes
	// 2. Release locks
	// 3. Update transaction state

	fmt.Printf("Commit request: txn_id=%s coordinator=%s\n", req.TxnId, req.CoordinatorId)

	s.mu.Lock()
	if txn, exists := s.transactions[req.TxnId]; exists {
		txn.State = pb.TransactionState_COMMITTED
	}
	s.mu.Unlock()

	return &pb.CommitResponse{
		TxnId:         req.TxnId,
		ParticipantId: "local-node",
		Success:       true,
	}, nil
}

// Abort handles the abort phase of 2PC
func (s *TransactionServiceImpl) Abort(ctx context.Context, req *pb.AbortRequest) (*pb.AbortResponse, error) {
	// TODO: Implement abort logic
	// This should:
	// 1. Rollback prepared changes
	// 2. Release locks
	// 3. Update transaction state

	fmt.Printf("Abort request: txn_id=%s coordinator=%s reason=%s\n",
		req.TxnId, req.CoordinatorId, req.Reason)

	s.mu.Lock()
	if txn, exists := s.transactions[req.TxnId]; exists {
		txn.State = pb.TransactionState_ABORTED
	}
	delete(s.transactions, req.TxnId) // Clean up after abort
	s.mu.Unlock()

	return &pb.AbortResponse{
		TxnId:         req.TxnId,
		ParticipantId: "local-node",
		Success:       true,
	}, nil
}

// QueryTransactionState handles transaction state queries for recovery
func (s *TransactionServiceImpl) QueryTransactionState(ctx context.Context, req *pb.RecoveryRequest) (*pb.RecoveryResponse, error) {
	// TODO: Implement transaction state query for recovery
	fmt.Printf("QueryTransactionState request: txn_id=%s from_node=%s\n",
		req.TxnId, req.RequestingNodeId)

	s.mu.RLock()
	txn, exists := s.transactions[req.TxnId]
	s.mu.RUnlock()

	if !exists {
		return &pb.RecoveryResponse{
			TxnId:         req.TxnId,
			State:         pb.TransactionState_UNKNOWN_STATE,
			Participants:  []string{},
			CoordinatorId: "",
		}, nil
	}

	return &pb.RecoveryResponse{
		TxnId:         txn.TxnId,
		State:         txn.State,
		Participants:  txn.Participants,
		CoordinatorId: "",
	}, nil
}

// GetTransaction returns a transaction by ID (helper method)
func (s *TransactionServiceImpl) GetTransaction(txnID string) (*pb.Transaction, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	txn, exists := s.transactions[txnID]
	return txn, exists
}

// CleanupTransaction removes a transaction from memory (helper method)
func (s *TransactionServiceImpl) CleanupTransaction(txnID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.transactions, txnID)
}
