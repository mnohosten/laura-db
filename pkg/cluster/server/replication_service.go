package server

import (
	"context"
	"fmt"

	pb "github.com/mnohosten/laura-db/pkg/cluster/proto"
)

// ReplicationServiceImpl implements the ReplicationService gRPC service
type ReplicationServiceImpl struct {
	pb.UnimplementedReplicationServiceServer
}

// NewReplicationServiceImpl creates a new replication service implementation
func NewReplicationServiceImpl() *ReplicationServiceImpl {
	return &ReplicationServiceImpl{}
}

// StreamOplog streams oplog entries to a replica
func (s *ReplicationServiceImpl) StreamOplog(req *pb.ReplicationStreamRequest, stream pb.ReplicationService_StreamOplogServer) error {
	// TODO: Implement oplog streaming
	// This is a streaming RPC that will send oplog entries to replicas
	fmt.Printf("StreamOplog request from replica: %s starting at oplog ID: %d\n",
		req.ReplicaId, req.StartOplogId)

	// For now, return empty response to avoid errors
	return stream.Send(&pb.ReplicationStreamResponse{
		Entries:        []*pb.OplogEntry{},
		LatestOplogId:  req.StartOplogId,
		MoreAvailable:  false,
	})
}

// InitialSync performs initial sync for a new replica
func (s *ReplicationServiceImpl) InitialSync(req *pb.InitialSyncRequest, stream pb.ReplicationService_InitialSyncServer) error {
	// TODO: Implement initial sync
	fmt.Printf("InitialSync request from replica: %s\n", req.ReplicaId)

	// For now, return complete response
	return stream.Send(&pb.InitialSyncResponse{
		DataChunk:         []byte{},
		Complete:          true,
		SyncPointOplogId:  0,
	})
}

// RequestVote handles vote requests during leader election
func (s *ReplicationServiceImpl) RequestVote(ctx context.Context, req *pb.VoteRequest) (*pb.VoteResponse, error) {
	// TODO: Implement Raft-style voting logic
	fmt.Printf("Vote request from candidate: %s for term: %d\n",
		req.CandidateId, req.Term)

	// For now, grant vote
	return &pb.VoteResponse{
		VoteGranted: true,
		Term:        req.Term,
		VoterId:     "local-node",
	}, nil
}

// AcknowledgeWrite handles write acknowledgments from replicas
func (s *ReplicationServiceImpl) AcknowledgeWrite(ctx context.Context, req *pb.WriteAcknowledgmentRequest) (*pb.WriteAcknowledgmentResponse, error) {
	// TODO: Implement write concern tracking
	return &pb.WriteAcknowledgmentResponse{
		AllAcknowledged: true,
		AckCount:        int32(len(req.Acks)),
	}, nil
}

// AppendOplog appends an oplog entry (called by primary)
func (s *ReplicationServiceImpl) AppendOplog(ctx context.Context, req *pb.OplogEntry) (*pb.AppendOplogResponse, error) {
	// TODO: Implement oplog appending
	fmt.Printf("AppendOplog: op_id=%d namespace=%s op_type=%s\n",
		req.OpId, req.Namespace, req.OpType.String())

	return &pb.AppendOplogResponse{
		Success: true,
		OplogId: req.OpId,
	}, nil
}
