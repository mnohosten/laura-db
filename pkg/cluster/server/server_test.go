package server

import (
	"context"
	"testing"
	"time"

	pb "github.com/mnohosten/laura-db/pkg/cluster/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestServerStartStop(t *testing.T) {
	// Create server with custom port to avoid conflicts
	config := DefaultConfig()
	config.Port = 0 // Use random port

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Verify server is started
	if !server.IsStarted() {
		t.Error("Server should be started")
	}

	// Get actual address
	addr := server.Addr()
	if addr == nil {
		t.Fatal("Server address should not be nil")
	}

	// Stop server
	if err := server.Stop(); err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}

	// Verify server is stopped
	if server.IsStarted() {
		t.Error("Server should be stopped")
	}
}

func TestClusterServiceRegisterNode(t *testing.T) {
	// Create and start server
	config := DefaultConfig()
	config.Port = 0

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Create client connection
	addr := server.Addr().String()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Create client
	client := pb.NewClusterServiceClient(conn)

	// Register a node
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.RegisterNode(ctx, &pb.NodeRegistrationRequest{
		Node: &pb.Node{
			Id:       "test-node-1",
			Host:     "localhost",
			Port:     27017,
			Role:     pb.NodeRole_SECONDARY,
			Priority: 5,
		},
	})

	if err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	if !resp.Success {
		t.Errorf("Expected success=true, got: %v", resp.Success)
	}

	if len(resp.ClusterNodes) != 1 {
		t.Errorf("Expected 1 cluster node, got: %d", len(resp.ClusterNodes))
	}

	// Verify node was registered
	nodes := server.GetClusterService().GetNodes()
	if len(nodes) != 1 {
		t.Errorf("Expected 1 registered node, got: %d", len(nodes))
	}
}

func TestClusterServiceHeartbeat(t *testing.T) {
	// Create and start server
	config := DefaultConfig()
	config.Port = 0

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Register a node first
	clusterService := server.GetClusterService()
	_, err = clusterService.RegisterNode(context.Background(), &pb.NodeRegistrationRequest{
		Node: &pb.Node{
			Id:   "test-node-1",
			Host: "localhost",
			Port: 27017,
			Role: pb.NodeRole_SECONDARY,
		},
	})
	if err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	// Create client connection
	addr := server.Addr().String()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewClusterServiceClient(conn)

	// Send heartbeat
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Heartbeat(ctx, &pb.HeartbeatRequest{
		NodeId:    "test-node-1",
		Timestamp: time.Now().UnixMilli(),
		Status: &pb.NodeStatus{
			Role: pb.NodeRole_SECONDARY,
		},
	})

	if err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}

	if !resp.Ack {
		t.Error("Expected heartbeat acknowledgment")
	}

	if resp.Topology == nil {
		t.Error("Expected topology in heartbeat response")
	}
}

func TestClusterServiceDeregisterNode(t *testing.T) {
	// Create service
	service := NewClusterServiceImpl()

	// Register a node
	_, err := service.RegisterNode(context.Background(), &pb.NodeRegistrationRequest{
		Node: &pb.Node{
			Id:   "test-node-1",
			Host: "localhost",
			Port: 27017,
			Role: pb.NodeRole_SECONDARY,
		},
	})
	if err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	// Deregister the node
	resp, err := service.DeregisterNode(context.Background(), &pb.NodeDeregistrationRequest{
		NodeId: "test-node-1",
		Reason: "test shutdown",
	})

	if err != nil {
		t.Fatalf("DeregisterNode failed: %v", err)
	}

	if !resp.Success {
		t.Errorf("Expected success=true, got: %v", resp.Success)
	}

	// Verify node was removed
	nodes := service.GetNodes()
	if len(nodes) != 0 {
		t.Errorf("Expected 0 nodes after deregistration, got: %d", len(nodes))
	}
}

func TestServerHealthCheck(t *testing.T) {
	config := DefaultConfig()
	config.Port = 0

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Health check should fail before start
	if err := server.HealthCheck(context.Background()); err == nil {
		t.Error("Expected health check to fail before server start")
	}

	// Start server
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Health check should succeed after start
	if err := server.HealthCheck(context.Background()); err != nil {
		t.Errorf("Expected health check to succeed after server start: %v", err)
	}
}
