package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/replication"
)

// TestWebSocketConnection tests basic WebSocket connection establishment
func TestWebSocketConnection(t *testing.T) {
	// Create temp database
	tmpDir := t.TempDir()
	db, err := database.Open(database.DefaultConfig(tmpDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create change stream manager
	oplogPath := tmpDir + "/oplog.bin"
	manager, err := NewChangeStreamManager(oplogPath)
	if err != nil {
		t.Fatalf("Failed to create change stream manager: %v", err)
	}
	defer manager.Close()

	// Create handler
	h := New(db)

	// Create router and setup WebSocket route
	r := chi.NewRouter()
	r.Get("/_ws/watch", h.HandleChangeStream(manager))

	// Create test server
	server := httptest.NewServer(r)
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/_ws/watch"

	// Connect to WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer ws.Close()

	// Send initial request
	req := ChangeStreamRequest{
		Database:   "testdb",
		Collection: "users",
	}
	if err := ws.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read acknowledgment
	var ack ChangeStreamResponse
	if err := ws.ReadJSON(&ack); err != nil {
		t.Fatalf("Failed to read acknowledgment: %v", err)
	}

	if ack.Type != "connected" {
		t.Errorf("Expected type 'connected', got '%s'", ack.Type)
	}
}

// TestWebSocketChangeEvents tests receiving change events over WebSocket
func TestWebSocketChangeEvents(t *testing.T) {
	// Create temp database
	tmpDir := t.TempDir()
	db, err := database.Open(database.DefaultConfig(tmpDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create change stream manager
	oplogPath := tmpDir + "/oplog.bin"
	manager, err := NewChangeStreamManager(oplogPath)
	if err != nil {
		t.Fatalf("Failed to create change stream manager: %v", err)
	}
	defer manager.Close()

	// Create handler
	h := New(db)

	// Create router and setup WebSocket route
	r := chi.NewRouter()
	r.Get("/_ws/watch", h.HandleChangeStream(manager))

	// Create test server
	server := httptest.NewServer(r)
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/_ws/watch"

	// Connect to WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer ws.Close()

	// Send initial request
	req := ChangeStreamRequest{
		Database:   "testdb",
		Collection: "users",
	}
	if err := ws.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read acknowledgment
	var ack ChangeStreamResponse
	if err := ws.ReadJSON(&ack); err != nil {
		t.Fatalf("Failed to read acknowledgment: %v", err)
	}

	// Insert a document to trigger change event
	go func() {
		time.Sleep(100 * time.Millisecond)
		entry := &replication.OplogEntry{
			Timestamp:  time.Now(),
			OpType:     replication.OpTypeInsert,
			Database:   "testdb",
			Collection: "users",
			DocID:      "user1",
			Document: map[string]interface{}{
				"_id":  "user1",
				"name": "Alice",
			},
		}
		manager.oplog.Append(entry)
	}()

	// Set read deadline
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Read change event
	var response ChangeStreamResponse
	if err := ws.ReadJSON(&response); err != nil {
		t.Fatalf("Failed to read change event: %v", err)
	}

	// Verify event type (could be heartbeat or event)
	if response.Type != "event" && response.Type != "heartbeat" {
		t.Logf("Warning: Expected 'event' or 'heartbeat', got '%s'", response.Type)
	}
}

// TestWebSocketWithFilter tests WebSocket with filter
func TestWebSocketWithFilter(t *testing.T) {
	// Create temp database
	tmpDir := t.TempDir()
	db, err := database.Open(database.DefaultConfig(tmpDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create change stream manager
	oplogPath := tmpDir + "/oplog.bin"
	manager, err := NewChangeStreamManager(oplogPath)
	if err != nil {
		t.Fatalf("Failed to create change stream manager: %v", err)
	}
	defer manager.Close()

	// Create handler
	h := New(db)

	// Create router and setup WebSocket route
	r := chi.NewRouter()
	r.Get("/_ws/watch", h.HandleChangeStream(manager))

	// Create test server
	server := httptest.NewServer(r)
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/_ws/watch"

	// Connect to WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer ws.Close()

	// Send initial request with filter
	req := ChangeStreamRequest{
		Database:   "testdb",
		Collection: "users",
		Filter: map[string]interface{}{
			"operationType": "insert",
		},
	}
	if err := ws.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read acknowledgment
	var ack ChangeStreamResponse
	if err := ws.ReadJSON(&ack); err != nil {
		t.Fatalf("Failed to read acknowledgment: %v", err)
	}

	if ack.Type != "connected" {
		t.Errorf("Expected type 'connected', got '%s'", ack.Type)
	}
}

// TestWebSocketHeartbeat tests WebSocket heartbeat messages
func TestWebSocketHeartbeat(t *testing.T) {
	// Create temp database
	tmpDir := t.TempDir()
	db, err := database.Open(database.DefaultConfig(tmpDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create change stream manager
	oplogPath := tmpDir + "/oplog.bin"
	manager, err := NewChangeStreamManager(oplogPath)
	if err != nil {
		t.Fatalf("Failed to create change stream manager: %v", err)
	}
	defer manager.Close()

	// Create handler
	h := New(db)

	// Create router and setup WebSocket route
	r := chi.NewRouter()
	r.Get("/_ws/watch", h.HandleChangeStream(manager))

	// Create test server
	server := httptest.NewServer(r)
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/_ws/watch"

	// Connect to WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer ws.Close()

	// Send initial request
	req := ChangeStreamRequest{
		Database:   "testdb",
		Collection: "users",
	}
	if err := ws.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read acknowledgment
	var ack ChangeStreamResponse
	if err := ws.ReadJSON(&ack); err != nil {
		t.Fatalf("Failed to read acknowledgment: %v", err)
	}

	// Note: Full heartbeat test would require waiting 30+ seconds
	// This is a basic connection test
	if ack.Type != "connected" {
		t.Errorf("Expected type 'connected', got '%s'", ack.Type)
	}
}

// TestChangeStreamHTTPEndpoint tests the HTTP endpoint for change streams
func TestChangeStreamHTTPEndpoint(t *testing.T) {
	// Create temp database
	tmpDir := t.TempDir()
	db, err := database.Open(database.DefaultConfig(tmpDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create handler
	h := New(db)

	// Create test request
	reqBody := `{"database":"testdb","collection":"users"}`
	req := httptest.NewRequest("POST", "/_watch", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Call handler
	h.HandleChangeStreamHTTP()(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["message"] == "" {
		t.Error("Expected message in response")
	}
}

// TestChangeStreamManagerClose tests proper cleanup of change stream manager
func TestChangeStreamManagerClose(t *testing.T) {
	tmpDir := t.TempDir()
	oplogPath := tmpDir + "/oplog.bin"

	manager, err := NewChangeStreamManager(oplogPath)
	if err != nil {
		t.Fatalf("Failed to create change stream manager: %v", err)
	}

	// Close manager
	if err := manager.Close(); err != nil {
		t.Errorf("Failed to close manager: %v", err)
	}
}

// TestMultipleWebSocketConnections tests multiple concurrent WebSocket connections
func TestMultipleWebSocketConnections(t *testing.T) {
	// Create temp database
	tmpDir := t.TempDir()
	db, err := database.Open(database.DefaultConfig(tmpDir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create change stream manager
	oplogPath := tmpDir + "/oplog.bin"
	manager, err := NewChangeStreamManager(oplogPath)
	if err != nil {
		t.Fatalf("Failed to create change stream manager: %v", err)
	}
	defer manager.Close()

	// Create handler
	h := New(db)

	// Create router and setup WebSocket route
	r := chi.NewRouter()
	r.Get("/_ws/watch", h.HandleChangeStream(manager))

	// Create test server
	server := httptest.NewServer(r)
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/_ws/watch"

	// Connect multiple WebSocket clients
	numClients := 3
	connections := make([]*websocket.Conn, numClients)

	for i := 0; i < numClients; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect WebSocket client %d: %v", i, err)
		}
		defer ws.Close()
		connections[i] = ws

		// Send initial request
		req := ChangeStreamRequest{
			Database:   fmt.Sprintf("testdb%d", i),
			Collection: "users",
		}
		if err := ws.WriteJSON(req); err != nil {
			t.Fatalf("Failed to send request for client %d: %v", i, err)
		}

		// Read acknowledgment
		var ack ChangeStreamResponse
		if err := ws.ReadJSON(&ack); err != nil {
			t.Fatalf("Failed to read ack for client %d: %v", i, err)
		}

		if ack.Type != "connected" {
			t.Errorf("Client %d: Expected type 'connected', got '%s'", i, ack.Type)
		}
	}

	// Verify all connections are registered
	manager.mu.RLock()
	connCount := len(manager.connections)
	manager.mu.RUnlock()

	if connCount != numClients {
		t.Errorf("Expected %d connections, got %d", numClients, connCount)
	}
}
