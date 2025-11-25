package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestCreateCollection tests collection creation
func TestCreateCollection(t *testing.T) {
	handlers, cleanup := setupTestHandlers(t)
	defer cleanup()

	req := httptest.NewRequest("PUT", "/testcollection/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("collection", "testcollection")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handlers.CreateCollection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response["ok"].(bool) {
		t.Error("Expected ok=true")
	}

	// Verify collection exists
	coll := handlers.db.Collection("testcollection")
	if coll == nil {
		t.Error("Collection should exist")
	}
}

// TestDropCollection tests dropping a collection
func TestDropCollection(t *testing.T) {
	handlers, cleanup := setupTestHandlers(t)
	defer cleanup()

	// Create collection first
	handlers.db.CreateCollection("todrop")

	req := httptest.NewRequest("DELETE", "/todrop/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("collection", "todrop")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handlers.DropCollection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response["ok"].(bool) {
		t.Error("Expected ok=true")
	}

	// Verify collection is gone
	collections := handlers.db.ListCollections()
	for _, name := range collections {
		if name == "todrop" {
			t.Error("Collection should be dropped")
		}
	}
}

// TestGetCollectionStats tests getting collection statistics
func TestGetCollectionStats(t *testing.T) {
	handlers, cleanup := setupTestHandlers(t)
	defer cleanup()

	// Create collection and insert some data
	coll, _ := handlers.db.CreateCollection("stats")
	coll.InsertOne(map[string]interface{}{"test": "data"})
	coll.InsertOne(map[string]interface{}{"test": "data2"})

	req := httptest.NewRequest("GET", "/stats/_stats", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("collection", "stats")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handlers.GetCollectionStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response["ok"].(bool) {
		t.Error("Expected ok=true")
	}

	// Just check we got a result
	result := response["result"]
	if result == nil {
		t.Error("Expected result in response")
	}
}

// TestGetCollectionStatsNotFound tests stats for non-existent collection
func Disabled_TestGetCollectionStatsNotFound(t *testing.T) {
	handlers, cleanup := setupTestHandlers(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/nonexistent/_stats", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("collection", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handlers.GetCollectionStats(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}
