package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHealth tests the health check endpoint
func TestHealth(t *testing.T) {
	handlers, cleanup := setupTestHandlers(t)
	defer cleanup()

	startTime := time.Now()
	handler := handlers.Health(startTime)

	req := httptest.NewRequest("GET", "/_health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response["ok"].(bool) {
		t.Error("Expected ok=true")
	}

	result := response["result"].(map[string]interface{})
	if result["status"] != "healthy" {
		t.Errorf("Expected status=healthy, got %v", result["status"])
	}

	if result["uptime"] == nil {
		t.Error("Expected uptime in response")
	}
}

// TestGetDatabaseStats tests getting database statistics
func TestGetDatabaseStats(t *testing.T) {
	handlers, cleanup := setupTestHandlers(t)
	defer cleanup()

	// Create some collections and data
	coll1, _ := handlers.db.CreateCollection("coll1")
	coll1.InsertOne(map[string]interface{}{"data": "test"})

	coll2, _ := handlers.db.CreateCollection("coll2")
	coll2.InsertOne(map[string]interface{}{"data": "test"})
	coll2.InsertOne(map[string]interface{}{"data": "test2"})

	req := httptest.NewRequest("GET", "/_stats", nil)
	w := httptest.NewRecorder()

	handlers.GetDatabaseStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response["ok"].(bool) {
		t.Error("Expected ok=true")
	}

	// Stats returns various database statistics
	// Just check that we got a valid response
	result := response["result"]
	if result == nil {
		t.Error("Expected result in response")
	}
}

// TestListCollections tests listing all collections
func TestListCollections(t *testing.T) {
	handlers, cleanup := setupTestHandlers(t)
	defer cleanup()

	// Create test collections
	handlers.db.CreateCollection("users")
	handlers.db.CreateCollection("products")
	handlers.db.CreateCollection("orders")

	req := httptest.NewRequest("GET", "/_collections", nil)
	w := httptest.NewRecorder()

	handlers.ListCollections(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response["ok"].(bool) {
		t.Error("Expected ok=true")
	}

	result := response["result"].(map[string]interface{})
	collectionsList := result["collections"].([]interface{})
	if len(collectionsList) != 3 {
		t.Errorf("Expected 3 collections, got %d", len(collectionsList))
	}

	// Check collection names
	collections := make(map[string]bool)
	for _, c := range collectionsList {
		collections[c.(string)] = true
	}

	if !collections["users"] || !collections["products"] || !collections["orders"] {
		t.Error("Expected users, products, and orders collections")
	}
}

// TestListCollectionsEmpty tests listing when no collections exist
func TestListCollectionsEmpty(t *testing.T) {
	handlers, cleanup := setupTestHandlers(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/_collections", nil)
	w := httptest.NewRecorder()

	handlers.ListCollections(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	result := response["result"].(map[string]interface{})
	collectionsList := result["collections"].([]interface{})
	if len(collectionsList) != 0 {
		t.Errorf("Expected 0 collections, got %d", len(collectionsList))
	}
}
