package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCollectionInsertOne(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected method POST, got %s", r.Method)
		}
		if r.URL.Path != "/users/_doc" {
			t.Errorf("expected path '/users/_doc', got '%s'", r.URL.Path)
		}

		// Verify request body
		body, _ := io.ReadAll(r.Body)
		var doc map[string]interface{}
		json.Unmarshal(body, &doc)
		if doc["name"] != "Alice" {
			t.Errorf("expected name 'Alice', got '%v'", doc["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"ok": true,
			"result": {
				"_id": "507f1f77bcf86cd799439011"
			}
		}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("users")

	doc := map[string]interface{}{
		"name": "Alice",
		"age":  int64(30),
	}

	id, err := coll.InsertOne(doc)
	if err != nil {
		t.Fatalf("InsertOne() failed: %v", err)
	}

	if id != "507f1f77bcf86cd799439011" {
		t.Errorf("expected ID '507f1f77bcf86cd799439011', got '%s'", id)
	}
}

func TestCollectionInsertOneWithID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected method POST, got %s", r.Method)
		}
		if r.URL.Path != "/users/_doc/custom123" {
			t.Errorf("expected path '/users/_doc/custom123', got '%s'", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("users")

	doc := map[string]interface{}{
		"name": "Bob",
	}

	err := coll.InsertOneWithID("custom123", doc)
	if err != nil {
		t.Fatalf("InsertOneWithID() failed: %v", err)
	}
}

func TestCollectionFindOne(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected method GET, got %s", r.Method)
		}
		if r.URL.Path != "/users/_doc/123" {
			t.Errorf("expected path '/users/_doc/123', got '%s'", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"ok": true,
			"result": {
				"_id": "123",
				"name": "Alice",
				"age": 30
			}
		}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("users")

	doc, err := coll.FindOne("123")
	if err != nil {
		t.Fatalf("FindOne() failed: %v", err)
	}

	if doc["_id"] != "123" {
		t.Errorf("expected _id '123', got '%v'", doc["_id"])
	}
	if doc["name"] != "Alice" {
		t.Errorf("expected name 'Alice', got '%v'", doc["name"])
	}
}

func TestCollectionUpdateOne(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected method PUT, got %s", r.Method)
		}
		if r.URL.Path != "/users/_doc/123" {
			t.Errorf("expected path '/users/_doc/123', got '%s'", r.URL.Path)
		}

		// Verify update document
		body, _ := io.ReadAll(r.Body)
		var update map[string]interface{}
		json.Unmarshal(body, &update)
		if set, ok := update["$set"].(map[string]interface{}); ok {
			if set["age"] != float64(31) {
				t.Errorf("expected age 31, got %v", set["age"])
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("users")

	update := map[string]interface{}{
		"$set": map[string]interface{}{
			"age": 31,
		},
	}

	err := coll.UpdateOne("123", update)
	if err != nil {
		t.Fatalf("UpdateOne() failed: %v", err)
	}
}

func TestCollectionDeleteOne(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected method DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/users/_doc/123" {
			t.Errorf("expected path '/users/_doc/123', got '%s'", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("users")

	err := coll.DeleteOne("123")
	if err != nil {
		t.Fatalf("DeleteOne() failed: %v", err)
	}
}

func TestCollectionSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected method POST, got %s", r.Method)
		}
		if r.URL.Path != "/users/_search" {
			t.Errorf("expected path '/users/_search', got '%s'", r.URL.Path)
		}

		// Verify search options
		body, _ := io.ReadAll(r.Body)
		var opts SearchOptions
		json.Unmarshal(body, &opts)
		if opts.Limit != 10 {
			t.Errorf("expected limit 10, got %d", opts.Limit)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"ok": true,
			"result": [
				{"_id": "1", "name": "Alice"},
				{"_id": "2", "name": "Bob"}
			]
		}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("users")

	opts := &SearchOptions{
		Filter: map[string]interface{}{
			"age": map[string]interface{}{"$gt": 25},
		},
		Limit: 10,
	}

	docs, err := coll.Search(opts)
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("expected 2 documents, got %d", len(docs))
	}
}

func TestCollectionFind(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"ok": true,
			"result": [
				{"_id": "1", "name": "Alice", "age": 30}
			]
		}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("users")

	filter := map[string]interface{}{
		"age": int64(30),
	}

	docs, err := coll.Find(filter)
	if err != nil {
		t.Fatalf("Find() failed: %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("expected 1 document, got %d", len(docs))
	}
}

func TestCollectionCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"ok": true,
			"result": {
				"count": 42
			}
		}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("users")

	count, err := coll.Count(nil)
	if err != nil {
		t.Fatalf("Count() failed: %v", err)
	}

	if count != 42 {
		t.Errorf("expected count 42, got %d", count)
	}
}

func TestCollectionCountWithFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected method POST for filtered count, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"ok": true,
			"result": {
				"count": 10
			}
		}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("users")

	filter := map[string]interface{}{
		"age": map[string]interface{}{"$gt": 25},
	}

	count, err := coll.Count(filter)
	if err != nil {
		t.Fatalf("Count() failed: %v", err)
	}

	if count != 10 {
		t.Errorf("expected count 10, got %d", count)
	}
}

func TestCollectionStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/_stats" {
			t.Errorf("expected path '/users/_stats', got '%s'", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"ok": true,
			"result": {
				"name": "users",
				"count": 100,
				"indexes": 3,
				"index_details": []
			}
		}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("users")

	stats, err := coll.Stats()
	if err != nil {
		t.Fatalf("Stats() failed: %v", err)
	}

	if stats.Name != "users" {
		t.Errorf("expected name 'users', got '%s'", stats.Name)
	}
	if stats.Count != 100 {
		t.Errorf("expected count 100, got %d", stats.Count)
	}
	if stats.Indexes != 3 {
		t.Errorf("expected 3 indexes, got %d", stats.Indexes)
	}
}

func TestCollectionDrop(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected method DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/users" {
			t.Errorf("expected path '/users', got '%s'", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("users")

	err := coll.Drop()
	if err != nil {
		t.Fatalf("Drop() failed: %v", err)
	}
}

func TestCollectionBulk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected method POST, got %s", r.Method)
		}
		if r.URL.Path != "/users/_bulk" {
			t.Errorf("expected path '/users/_bulk', got '%s'", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"ok": true,
			"result": {
				"inserted": 2,
				"updated": 1,
				"deleted": 1,
				"failed": 0
			}
		}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("users")

	operations := []BulkOperation{
		{Operation: "insert", Document: map[string]interface{}{"name": "Alice"}},
		{Operation: "insert", Document: map[string]interface{}{"name": "Bob"}},
		{Operation: "update", ID: "123", Update: map[string]interface{}{"$set": map[string]interface{}{"age": 31}}},
		{Operation: "delete", ID: "456"},
	}

	result, err := coll.Bulk(operations)
	if err != nil {
		t.Fatalf("Bulk() failed: %v", err)
	}

	if result.Inserted != 2 {
		t.Errorf("expected 2 inserted, got %d", result.Inserted)
	}
	if result.Updated != 1 {
		t.Errorf("expected 1 updated, got %d", result.Updated)
	}
	if result.Deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", result.Deleted)
	}
	if result.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", result.Failed)
	}
}
