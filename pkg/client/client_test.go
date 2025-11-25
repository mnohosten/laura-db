package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Host != "localhost" {
		t.Errorf("expected host 'localhost', got '%s'", config.Host)
	}
	if config.Port != 8080 {
		t.Errorf("expected port 8080, got %d", config.Port)
	}
	if config.Timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", config.Timeout)
	}
	if config.MaxIdleConns != 10 {
		t.Errorf("expected MaxIdleConns 10, got %d", config.MaxIdleConns)
	}
}

func TestNewClient(t *testing.T) {
	config := &Config{
		Host:    "example.com",
		Port:    9090,
		Timeout: 10 * time.Second,
	}

	client := NewClient(config)

	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.baseURL != "http://example.com:9090" {
		t.Errorf("expected baseURL 'http://example.com:9090', got '%s'", client.baseURL)
	}
	if client.httpClient == nil {
		t.Error("expected non-nil httpClient")
	}
}

func TestNewClientWithDefaults(t *testing.T) {
	// Test with partially filled config
	config := &Config{
		Host: "testhost",
	}

	client := NewClient(config)

	if client.baseURL != "http://testhost:8080" {
		t.Errorf("expected baseURL 'http://testhost:8080', got '%s'", client.baseURL)
	}
}

func TestNewDefaultClient(t *testing.T) {
	client := NewDefaultClient()

	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.baseURL != "http://localhost:8080" {
		t.Errorf("expected baseURL 'http://localhost:8080', got '%s'", client.baseURL)
	}
}

func TestHealthEndpoint(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_health" {
			t.Errorf("expected path '/_health', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected method GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"ok": true,
			"result": {
				"status": "healthy",
				"uptime": "5m30s",
				"time": "2025-11-24T10:00:00Z"
			}
		}`))
	}))
	defer server.Close()

	// Create client pointing to test server
	client := NewClient(&Config{
		Host: server.URL[7:], // Remove "http://"
		Port: 0,              // Will be ignored
	})
	// Override baseURL to use test server
	client.baseURL = server.URL

	health, err := client.Health()
	if err != nil {
		t.Fatalf("Health() failed: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", health.Status)
	}
	if health.Uptime != "5m30s" {
		t.Errorf("expected uptime '5m30s', got '%s'", health.Uptime)
	}
}

func TestStatsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_stats" {
			t.Errorf("expected path '/_stats', got '%s'", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"ok": true,
			"result": {
				"name": "default",
				"collections": 2,
				"active_transactions": 0,
				"collection_stats": {},
				"storage_stats": {
					"buffer_pool": {
						"capacity": 1000,
						"size": 0,
						"hits": 100,
						"misses": 10,
						"hit_rate": 0.91,
						"evictions": 5
					},
					"disk": {
						"total_reads": 50,
						"total_writes": 30,
						"next_page_id": 10,
						"free_pages": 5
					}
				}
			}
		}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL

	stats, err := client.Stats()
	if err != nil {
		t.Fatalf("Stats() failed: %v", err)
	}

	if stats.Name != "default" {
		t.Errorf("expected name 'default', got '%s'", stats.Name)
	}
	if stats.Collections != 2 {
		t.Errorf("expected 2 collections, got %d", stats.Collections)
	}
	if stats.StorageStats.BufferPool.Capacity != 1000 {
		t.Errorf("expected buffer pool capacity 1000, got %d", stats.StorageStats.BufferPool.Capacity)
	}
}

func TestListCollections(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"ok": true,
			"result": {
				"collections": ["users", "products", "orders"]
			}
		}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL

	collections, err := client.ListCollections()
	if err != nil {
		t.Fatalf("ListCollections() failed: %v", err)
	}

	if len(collections) != 3 {
		t.Errorf("expected 3 collections, got %d", len(collections))
	}
	if collections[0] != "users" || collections[1] != "products" || collections[2] != "orders" {
		t.Errorf("unexpected collections: %v", collections)
	}
}

func TestCreateCollection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected method PUT, got %s", r.Method)
		}
		if r.URL.Path != "/testcoll" {
			t.Errorf("expected path '/testcoll', got '%s'", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL

	err := client.CreateCollection("testcoll")
	if err != nil {
		t.Fatalf("CreateCollection() failed: %v", err)
	}
}

func TestDropCollection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected method DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/testcoll" {
			t.Errorf("expected path '/testcoll', got '%s'", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL

	err := client.DropCollection("testcoll")
	if err != nil {
		t.Fatalf("DropCollection() failed: %v", err)
	}
}

func TestCollectionMethod(t *testing.T) {
	client := NewDefaultClient()
	coll := client.Collection("users")

	if coll == nil {
		t.Fatal("expected non-nil collection")
	}
	if coll.Name() != "users" {
		t.Errorf("expected collection name 'users', got '%s'", coll.Name())
	}
	if coll.client != client {
		t.Error("expected collection to reference parent client")
	}
}

func TestErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{
			"ok": false,
			"error": "NotFound",
			"message": "collection not found",
			"code": 404
		}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL

	_, err := client.ListCollections()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expectedErr := "API error: NotFound - collection not found"
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestClose(t *testing.T) {
	client := NewDefaultClient()

	err := client.Close()
	if err != nil {
		t.Fatalf("Close() failed: %v", err)
	}
}
