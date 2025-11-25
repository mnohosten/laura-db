package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

// ChangeStreamRequest represents the WebSocket connection request
type ChangeStreamRequest struct {
	Database   string                 `json:"database"`
	Collection string                 `json:"collection"`
	Filter     map[string]interface{} `json:"filter,omitempty"`
	Pipeline   []map[string]interface{} `json:"pipeline,omitempty"`
}

// ChangeStreamResponse represents a response from the WebSocket server
type ChangeStreamResponse struct {
	Type    string                 `json:"type"` // "event", "error", "heartbeat", "connected"
	Event   map[string]interface{} `json:"event,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Message string                 `json:"message,omitempty"`
}

func main() {
	// Parse command line arguments
	serverURL := "localhost:8080"
	if len(os.Args) > 1 {
		serverURL = os.Args[1]
	}

	fmt.Println("=== LauraDB WebSocket Change Streams Demo ===")
	fmt.Println()

	// Setup interrupt handler
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Connect to WebSocket server
	u := url.URL{Scheme: "ws", Host: serverURL, Path: "/_ws/watch"}
	fmt.Printf("Connecting to %s\n", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Println("✅ Connected to WebSocket server")
	fmt.Println()

	// Channel for receiving messages
	done := make(chan struct{})

	// Send initial change stream request
	req := ChangeStreamRequest{
		Database:   "testdb",
		Collection: "users",
		Filter: map[string]interface{}{
			// Optionally filter by operation type
			// "operationType": "insert",
		},
	}

	fmt.Printf("📡 Subscribing to changes:\n")
	fmt.Printf("   Database: %s\n", req.Database)
	fmt.Printf("   Collection: %s\n", req.Collection)
	fmt.Println()

	if err := conn.WriteJSON(req); err != nil {
		log.Fatalf("Failed to send request: %v", err)
	}

	// Start message reader goroutine
	go func() {
		defer close(done)
		for {
			var response ChangeStreamResponse
			err := conn.ReadJSON(&response)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("Error reading message: %v", err)
				}
				return
			}

			handleResponse(response)
		}
	}()

	fmt.Println("🔍 Watching for changes... (Press Ctrl+C to exit)")
	fmt.Println()
	fmt.Println("To generate change events, open another terminal and run:")
	fmt.Println("  curl -X POST http://%s/users/_doc \\", serverURL)
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"name\":\"Alice\",\"age\":30}'")
	fmt.Println()

	// Wait for interrupt or connection close
	select {
	case <-done:
		fmt.Println("\n🔌 Connection closed")
	case <-interrupt:
		fmt.Println("\n⚠️  Interrupt received, closing connection...")

		// Cleanly close the connection
		err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Printf("Error sending close message: %v", err)
		}

		select {
		case <-done:
		case <-time.After(time.Second):
		}
	}
}

func handleResponse(response ChangeStreamResponse) {
	switch response.Type {
	case "connected":
		fmt.Printf("✅ %s\n", response.Message)
		fmt.Println()

	case "event":
		if response.Event != nil {
			fmt.Println("📨 Change Event Received:")
			fmt.Println("─────────────────────────────────────")

			// Pretty print the event
			eventJSON, err := json.MarshalIndent(response.Event, "   ", "  ")
			if err == nil {
				fmt.Printf("   %s\n", string(eventJSON))
			} else {
				fmt.Printf("   %v\n", response.Event)
			}

			fmt.Println("─────────────────────────────────────")
			fmt.Println()
		}

	case "heartbeat":
		// Optional: log heartbeats if desired
		// fmt.Println("💓 Heartbeat")

	case "error":
		fmt.Printf("❌ Error: %s\n", response.Error)

	default:
		fmt.Printf("⚠️  Unknown message type: %s\n", response.Type)
	}
}
