package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/mnohosten/laura-db/pkg/changestream"
	"github.com/mnohosten/laura-db/pkg/replication"
)

// WebSocket upgrader with default settings
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins (can be restricted in production)
		return true
	},
}

// ChangeStreamManager manages active change stream connections
type ChangeStreamManager struct {
	oplog       *replication.Oplog
	connections map[string]*ChangeStreamConnection
	mu          sync.RWMutex
}

// ChangeStreamConnection represents an active WebSocket connection with a change stream
type ChangeStreamConnection struct {
	id         string
	conn       *websocket.Conn
	stream     *changestream.ChangeStream
	cancelFunc context.CancelFunc
	mu         sync.Mutex
}

// NewChangeStreamManager creates a new change stream manager
func NewChangeStreamManager(oplogPath string) (*ChangeStreamManager, error) {
	oplog, err := replication.NewOplog(oplogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create oplog: %w", err)
	}

	return &ChangeStreamManager{
		oplog:       oplog,
		connections: make(map[string]*ChangeStreamConnection),
	}, nil
}

// GetOplog returns the underlying oplog
func (m *ChangeStreamManager) GetOplog() *replication.Oplog {
	return m.oplog
}

// Close closes the change stream manager and all active connections
func (m *ChangeStreamManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close all active connections
	for _, conn := range m.connections {
		conn.Close()
	}
	m.connections = make(map[string]*ChangeStreamConnection)

	// Close the oplog
	if m.oplog != nil {
		return m.oplog.Close()
	}
	return nil
}

// addConnection registers a new connection
func (m *ChangeStreamManager) addConnection(conn *ChangeStreamConnection) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connections[conn.id] = conn
}

// removeConnection unregisters a connection
func (m *ChangeStreamManager) removeConnection(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.connections, id)
}

// Close closes a change stream connection
func (c *ChangeStreamConnection) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancelFunc != nil {
		c.cancelFunc()
	}
	if c.stream != nil {
		c.stream.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}

// ChangeStreamRequest represents the WebSocket connection request
type ChangeStreamRequest struct {
	Database   string                 `json:"database"`
	Collection string                 `json:"collection"`
	Filter     map[string]interface{} `json:"filter,omitempty"`
	Pipeline   []map[string]interface{} `json:"pipeline,omitempty"`
	ResumeToken *changestream.ResumeToken `json:"resumeToken,omitempty"`
}

// ChangeStreamResponse represents a response sent over WebSocket
type ChangeStreamResponse struct {
	Type    string                  `json:"type"` // "event", "error", "heartbeat"
	Event   *changestream.ChangeEvent `json:"event,omitempty"`
	Error   string                  `json:"error,omitempty"`
	Message string                  `json:"message,omitempty"`
}

// HandleChangeStream handles WebSocket connections for change streams
func (h *Handlers) HandleChangeStream(manager *ChangeStreamManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Upgrade HTTP connection to WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Failed to upgrade connection: %v", err)
			return
		}

		// Generate connection ID
		connID := fmt.Sprintf("ws-%d", time.Now().UnixNano())

		// Create context for this connection
		ctx, cancel := context.WithCancel(context.Background())

		// Create connection object
		wsConn := &ChangeStreamConnection{
			id:         connID,
			conn:       conn,
			cancelFunc: cancel,
		}

		// Register connection
		manager.addConnection(wsConn)
		defer func() {
			manager.removeConnection(connID)
			wsConn.Close()
		}()

		// Read initial request from client
		var req ChangeStreamRequest
		if err := conn.ReadJSON(&req); err != nil {
			sendError(conn, fmt.Sprintf("Failed to read request: %v", err))
			return
		}

		// Validate database name
		if req.Database == "" {
			req.Database = "default"
		}

		// Create change stream options
		options := changestream.DefaultChangeStreamOptions()
		if req.Pipeline != nil {
			options.Pipeline = req.Pipeline
		}
		if req.ResumeToken != nil {
			options.ResumeAfter = req.ResumeToken
		}

		// Create change stream
		stream := changestream.NewChangeStream(manager.oplog, req.Database, req.Collection, options)
		if req.Filter != nil {
			stream.SetFilter(req.Filter)
		}

		wsConn.mu.Lock()
		wsConn.stream = stream
		wsConn.mu.Unlock()

		// Start the change stream
		stream.Start()

		// Send acknowledgment
		ack := ChangeStreamResponse{
			Type:    "connected",
			Message: "Change stream connected successfully",
		}
		if err := conn.WriteJSON(ack); err != nil {
			log.Printf("Failed to send acknowledgment: %v", err)
			return
		}

		// Start heartbeat goroutine to keep connection alive
		heartbeatTicker := time.NewTicker(30 * time.Second)
		defer heartbeatTicker.Stop()

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-heartbeatTicker.C:
					wsConn.mu.Lock()
					err := conn.WriteJSON(ChangeStreamResponse{
						Type:    "heartbeat",
						Message: "keepalive",
					})
					wsConn.mu.Unlock()
					if err != nil {
						log.Printf("Failed to send heartbeat: %v", err)
						cancel()
						return
					}
				}
			}
		}()

		// Read control messages from client (e.g., close)
		go func() {
			for {
				var msg map[string]interface{}
				if err := conn.ReadJSON(&msg); err != nil {
					cancel()
					return
				}
				// Handle control messages if needed
			}
		}()

		// Stream events to client
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Try to get next event (non-blocking with timeout)
				eventCtx, eventCancel := context.WithTimeout(ctx, 1*time.Second)
				event, err := stream.Next(eventCtx)
				eventCancel()

				if err != nil {
					if err == context.DeadlineExceeded {
						// No event available, continue waiting
						continue
					}
					if err == context.Canceled {
						// Context cancelled, exit
						return
					}
					// Send error to client
					sendError(conn, fmt.Sprintf("Stream error: %v", err))
					return
				}

				if event == nil {
					// No event, continue
					continue
				}

				// Send event to client
				response := ChangeStreamResponse{
					Type:  "event",
					Event: event,
				}

				wsConn.mu.Lock()
				err = conn.WriteJSON(response)
				wsConn.mu.Unlock()

				if err != nil {
					log.Printf("Failed to send event: %v", err)
					return
				}
			}
		}
	}
}

// sendError sends an error message to the WebSocket client
func sendError(conn *websocket.Conn, message string) {
	response := ChangeStreamResponse{
		Type:  "error",
		Error: message,
	}
	conn.WriteJSON(response)
}

// HandleChangeStreamHTTP handles HTTP endpoint for creating change streams (alternative to WebSocket)
func (h *Handlers) HandleChangeStreamHTTP() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ChangeStreamRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		encoder := json.NewEncoder(w)
		encoder.Encode(map[string]string{
			"message": "Use WebSocket endpoint /_ws/watch for streaming change events",
			"endpoint": "ws://<host>:<port>/_ws/watch",
		})
	}
}

// InitializeOplogForCollection logs a collection operation to the oplog
func InitializeOplogForCollection(oplog *replication.Oplog, database, collection string, opType replication.OpType, data map[string]interface{}) error {
	if oplog == nil {
		return nil // Oplog not initialized, skip
	}

	entry := &replication.OplogEntry{
		Timestamp:  time.Now(),
		OpType:     opType,
		Database:   database,
		Collection: collection,
	}

	switch opType {
	case replication.OpTypeInsert:
		entry.Document = data
		if docID, ok := data["_id"]; ok {
			entry.DocID = docID
		}
	case replication.OpTypeUpdate:
		if docID, ok := data["_id"]; ok {
			entry.DocID = docID
		}
		entry.Update = data
	case replication.OpTypeDelete:
		if docID, ok := data["_id"]; ok {
			entry.DocID = docID
		}
	case replication.OpTypeCreateIndex:
		entry.IndexDef = data
	}

	return oplog.Append(entry)
}

// SetupWebSocketRoutes adds WebSocket routes to the server
func SetupWebSocketRoutes(r chi.Router, h *Handlers, dataDir string) (*ChangeStreamManager, error) {
	// Create change stream manager with oplog in data directory
	oplogPath := filepath.Join(dataDir, "oplog.bin")
	manager, err := NewChangeStreamManager(oplogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create change stream manager: %w", err)
	}

	// Add WebSocket route for change streams
	r.Get("/_ws/watch", h.HandleChangeStream(manager))

	// Add HTTP endpoint for documentation
	r.Post("/_watch", h.HandleChangeStreamHTTP())

	return manager, nil
}
