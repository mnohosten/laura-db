package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "github.com/mnohosten/laura-db/pkg/cluster/proto"
)

// ClusterServiceImpl implements the ClusterService gRPC service
type ClusterServiceImpl struct {
	pb.UnimplementedClusterServiceServer

	mu       sync.RWMutex
	nodes    map[string]*pb.Node // nodeID -> Node
	topology *pb.ClusterTopology
	version  int64
}

// NewClusterServiceImpl creates a new cluster service implementation
func NewClusterServiceImpl() *ClusterServiceImpl {
	return &ClusterServiceImpl{
		nodes: make(map[string]*pb.Node),
		topology: &pb.ClusterTopology{
			Version: 1,
			Nodes:   []*pb.Node{},
		},
		version: 1,
	}
}

// RegisterNode handles node registration requests
func (s *ClusterServiceImpl) RegisterNode(ctx context.Context, req *pb.NodeRegistrationRequest) (*pb.NodeRegistrationResponse, error) {
	if req.Node == nil {
		return &pb.NodeRegistrationResponse{
			Success: false,
			Message: "node information required",
		}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if node already exists
	if existingNode, exists := s.nodes[req.Node.Id]; exists {
		return &pb.NodeRegistrationResponse{
			Success: false,
			Message: fmt.Sprintf("node %s already registered", req.Node.Id),
			ClusterNodes: []*pb.Node{existingNode},
			Topology: s.topology,
		}, nil
	}

	// Add node to cluster
	s.nodes[req.Node.Id] = req.Node
	s.updateTopology()

	fmt.Printf("Node registered: %s (%s:%d) role=%s\n",
		req.Node.Id, req.Node.Host, req.Node.Port, req.Node.Role.String())

	// Return current cluster state
	nodes := make([]*pb.Node, 0, len(s.nodes))
	for _, node := range s.nodes {
		nodes = append(nodes, node)
	}

	return &pb.NodeRegistrationResponse{
		Success:      true,
		Message:      "node registered successfully",
		ClusterNodes: nodes,
		Topology:     s.topology,
	}, nil
}

// DeregisterNode handles node deregistration requests
func (s *ClusterServiceImpl) DeregisterNode(ctx context.Context, req *pb.NodeDeregistrationRequest) (*pb.NodeDeregistrationResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if node exists
	if _, exists := s.nodes[req.NodeId]; !exists {
		return &pb.NodeDeregistrationResponse{
			Success: false,
			Message: fmt.Sprintf("node %s not found", req.NodeId),
		}, nil
	}

	// Remove node from cluster
	delete(s.nodes, req.NodeId)
	s.updateTopology()

	fmt.Printf("Node deregistered: %s (reason: %s)\n", req.NodeId, req.Reason)

	return &pb.NodeDeregistrationResponse{
		Success: true,
		Message: "node deregistered successfully",
	}, nil
}

// Heartbeat handles heartbeat requests from nodes
func (s *ClusterServiceImpl) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	s.mu.RLock()
	node, exists := s.nodes[req.NodeId]
	currentTopology := s.topology
	s.mu.RUnlock()

	if !exists {
		return &pb.HeartbeatResponse{
			Ack:       false,
			Timestamp: time.Now().UnixMilli(),
		}, fmt.Errorf("node %s not registered", req.NodeId)
	}

	// Update node's last heartbeat time (in a real implementation)
	// For now, just acknowledge
	_ = node

	return &pb.HeartbeatResponse{
		Ack:       true,
		Timestamp: time.Now().UnixMilli(),
		Topology:  currentTopology,
	}, nil
}

// GetTopology returns the current cluster topology
func (s *ClusterServiceImpl) GetTopology(ctx context.Context, req *pb.GetTopologyRequest) (*pb.ClusterTopology, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If client already has this version, don't send
	if req.MinVersion >= s.topology.Version {
		return nil, nil
	}

	return s.topology, nil
}

// updateTopology updates the cluster topology (must be called with lock held)
func (s *ClusterServiceImpl) updateTopology() {
	s.version++

	nodes := make([]*pb.Node, 0, len(s.nodes))
	for _, node := range s.nodes {
		nodes = append(nodes, node)
	}

	s.topology = &pb.ClusterTopology{
		Version:     s.version,
		Nodes:       nodes,
		ReplicaSets: []*pb.ReplicaSet{}, // To be implemented
		Shards:      []*pb.ShardInfo{},  // To be implemented
	}
}

// GetNodes returns all registered nodes (helper method)
func (s *ClusterServiceImpl) GetNodes() []*pb.Node {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := make([]*pb.Node, 0, len(s.nodes))
	for _, node := range s.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetNode returns a specific node by ID (helper method)
func (s *ClusterServiceImpl) GetNode(nodeID string) (*pb.Node, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	node, exists := s.nodes[nodeID]
	return node, exists
}

// GetTopologyVersion returns the current topology version (helper method)
func (s *ClusterServiceImpl) GetTopologyVersion() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}
