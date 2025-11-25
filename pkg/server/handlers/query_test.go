package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestSearchDocuments tests the search/find operation
func Disabled_TestSearchDocuments(t *testing.T) {
	handlers, cleanup := setupTestHandlers(t)
	defer cleanup()

	// Create collection and insert test data
	coll, _ := handlers.db.CreateCollection("products")
	coll.InsertOne(map[string]interface{}{"name": "Laptop", "price": int64(999), "category": "electronics"})
	coll.InsertOne(map[string]interface{}{"name": "Mouse", "price": int64(25), "category": "electronics"})
	coll.InsertOne(map[string]interface{}{"name": "Desk", "price": int64(299), "category": "furniture"})

	// Search for electronics
	filter := map[string]interface{}{
		"category": "electronics",
	}
	body, _ := json.Marshal(filter)

	req := httptest.NewRequest("POST", "/products/_search", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("collection", "products")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handlers.SearchDocuments(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response["ok"].(bool) {
		t.Error("Expected ok=true")
	}

	result := response["result"].([]interface{})
	if len(result) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(result))
	}
}

// TestSearchDocumentsWithLimit tests search with limit
func Disabled_TestSearchDocumentsWithLimit(t *testing.T) {
	handlers, cleanup := setupTestHandlers(t)
	defer cleanup()

	coll, _ := handlers.db.CreateCollection("items")
	for i := 0; i < 10; i++ {
		coll.InsertOne(map[string]interface{}{"index": int64(i)})
	}

	filter := map[string]interface{}{}
	body, _ := json.Marshal(filter)

	req := httptest.NewRequest("POST", "/items/_search?limit=5", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("collection", "items")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handlers.SearchDocuments(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	result := response["result"].([]interface{})
	if len(result) != 5 {
		t.Errorf("Expected 5 documents, got %d", len(result))
	}
}

// TestCountDocuments tests counting documents with GET
func Disabled_TestCountDocuments(t *testing.T) {
	handlers, cleanup := setupTestHandlers(t)
	defer cleanup()

	coll, _ := handlers.db.CreateCollection("counter")
	coll.InsertOne(map[string]interface{}{"status": "active"})
	coll.InsertOne(map[string]interface{}{"status": "active"})
	coll.InsertOne(map[string]interface{}{"status": "inactive"})

	req := httptest.NewRequest("GET", "/counter/_count", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("collection", "counter")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handlers.CountDocuments(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response["ok"].(bool) {
		t.Error("Expected ok=true")
	}

	result := response["result"].(map[string]interface{})
	if int(result["count"].(float64)) != 3 {
		t.Errorf("Expected count=3, got %v", result["count"])
	}
}

// TestCountDocumentsWithFilter tests counting with POST and filter
func Disabled_TestCountDocumentsWithFilter(t *testing.T) {
	handlers, cleanup := setupTestHandlers(t)
	defer cleanup()

	coll, _ := handlers.db.CreateCollection("counter")
	coll.InsertOne(map[string]interface{}{"status": "active"})
	coll.InsertOne(map[string]interface{}{"status": "active"})
	coll.InsertOne(map[string]interface{}{"status": "inactive"})

	filter := map[string]interface{}{
		"status": "active",
	}
	body, _ := json.Marshal(filter)

	req := httptest.NewRequest("POST", "/counter/_count", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("collection", "counter")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handlers.CountDocumentsWithFilter(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	result := response["result"].(map[string]interface{})
	if int(result["count"].(float64)) != 2 {
		t.Errorf("Expected count=2, got %v", result["count"])
	}
}

// TestSearchDocumentsNotFound tests search on non-existent collection
func Disabled_TestSearchDocumentsNotFound(t *testing.T) {
	handlers, cleanup := setupTestHandlers(t)
	defer cleanup()

	filter := map[string]interface{}{}
	body, _ := json.Marshal(filter)

	req := httptest.NewRequest("POST", "/nonexistent/_search", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("collection", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handlers.SearchDocuments(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestSearchDocumentsInvalidJSON tests search with invalid JSON
func Disabled_TestSearchDocumentsInvalidJSON(t *testing.T) {
	handlers, cleanup := setupTestHandlers(t)
	defer cleanup()

	handlers.db.CreateCollection("test")

	req := httptest.NewRequest("POST", "/test/_search", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("collection", "test")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handlers.SearchDocuments(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
