package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCollectionCreateIndex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected method POST, got %s", r.Method)
		}
		if r.URL.Path != "/users/_index" {
			t.Errorf("expected path '/users/_index', got '%s'", r.URL.Path)
		}

		// Verify index options
		body, _ := io.ReadAll(r.Body)
		var opts IndexOptions
		json.Unmarshal(body, &opts)
		if opts.Name != "age_idx" {
			t.Errorf("expected name 'age_idx', got '%s'", opts.Name)
		}
		if opts.Type != IndexTypeBTree {
			t.Errorf("expected type 'btree', got '%s'", opts.Type)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("users")

	opts := IndexOptions{
		Name:   "age_idx",
		Type:   IndexTypeBTree,
		Field:  "age",
		Unique: false,
	}

	err := coll.CreateIndex(opts)
	if err != nil {
		t.Fatalf("CreateIndex() failed: %v", err)
	}
}

func TestCollectionListIndexes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected method GET, got %s", r.Method)
		}
		if r.URL.Path != "/users/_index" {
			t.Errorf("expected path '/users/_index', got '%s'", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"ok": true,
			"result": {
				"indexes": [
					{
						"name": "_id_",
						"type": "btree",
						"fields": {"_id": 1},
						"unique": true,
						"sparse": false
					},
					{
						"name": "age_idx",
						"type": "btree",
						"fields": {"age": 1},
						"unique": false,
						"sparse": false
					}
				]
			}
		}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("users")

	indexes, err := coll.ListIndexes()
	if err != nil {
		t.Fatalf("ListIndexes() failed: %v", err)
	}

	if len(indexes) != 2 {
		t.Errorf("expected 2 indexes, got %d", len(indexes))
	}

	if indexes[0].Name != "_id_" {
		t.Errorf("expected first index name '_id_', got '%s'", indexes[0].Name)
	}
	if indexes[1].Name != "age_idx" {
		t.Errorf("expected second index name 'age_idx', got '%s'", indexes[1].Name)
	}
}

func TestCollectionDropIndex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected method DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/users/_index/age_idx" {
			t.Errorf("expected path '/users/_index/age_idx', got '%s'", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("users")

	err := coll.DropIndex("age_idx")
	if err != nil {
		t.Fatalf("DropIndex() failed: %v", err)
	}
}

func TestCreateBTreeIndex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var opts IndexOptions
		json.Unmarshal(body, &opts)

		if opts.Type != IndexTypeBTree {
			t.Errorf("expected type 'btree', got '%s'", opts.Type)
		}
		if opts.Field != "email" {
			t.Errorf("expected field 'email', got '%s'", opts.Field)
		}
		if !opts.Unique {
			t.Error("expected unique to be true")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("users")

	err := coll.CreateBTreeIndex("email_idx", "email", true)
	if err != nil {
		t.Fatalf("CreateBTreeIndex() failed: %v", err)
	}
}

func TestCreateCompoundIndex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var opts IndexOptions
		json.Unmarshal(body, &opts)

		if opts.Type != IndexTypeCompound {
			t.Errorf("expected type 'compound', got '%s'", opts.Type)
		}
		if len(opts.Fields) != 2 {
			t.Errorf("expected 2 fields, got %d", len(opts.Fields))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("users")

	fields := map[string]int{
		"city": 1,
		"age":  -1,
	}

	err := coll.CreateCompoundIndex("city_age_idx", fields, false)
	if err != nil {
		t.Fatalf("CreateCompoundIndex() failed: %v", err)
	}
}

func TestCreateTextIndex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var opts IndexOptions
		json.Unmarshal(body, &opts)

		if opts.Type != IndexTypeText {
			t.Errorf("expected type 'text', got '%s'", opts.Type)
		}
		if opts.Field != "description" {
			t.Errorf("expected field 'description', got '%s'", opts.Field)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("products")

	err := coll.CreateTextIndex("desc_text_idx", "description")
	if err != nil {
		t.Fatalf("CreateTextIndex() failed: %v", err)
	}
}

func TestCreateGeo2DIndex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var opts IndexOptions
		json.Unmarshal(body, &opts)

		if opts.Type != IndexTypeGeo2D {
			t.Errorf("expected type '2d', got '%s'", opts.Type)
		}
		if opts.Field != "location" {
			t.Errorf("expected field 'location', got '%s'", opts.Field)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("places")

	err := coll.CreateGeo2DIndex("location_2d", "location")
	if err != nil {
		t.Fatalf("CreateGeo2DIndex() failed: %v", err)
	}
}

func TestCreateGeo2DSphereIndex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var opts IndexOptions
		json.Unmarshal(body, &opts)

		if opts.Type != IndexTypeGeo2DSphere {
			t.Errorf("expected type '2dsphere', got '%s'", opts.Type)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("places")

	err := coll.CreateGeo2DSphereIndex("location_sphere", "coordinates")
	if err != nil {
		t.Fatalf("CreateGeo2DSphereIndex() failed: %v", err)
	}
}

func TestCreateTTLIndex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var opts IndexOptions
		json.Unmarshal(body, &opts)

		if opts.Type != IndexTypeTTL {
			t.Errorf("expected type 'ttl', got '%s'", opts.Type)
		}
		if opts.Field != "createdAt" {
			t.Errorf("expected field 'createdAt', got '%s'", opts.Field)
		}
		if opts.TTL != "24h" {
			t.Errorf("expected TTL '24h', got '%s'", opts.TTL)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("sessions")

	err := coll.CreateTTLIndex("created_ttl", "createdAt", "24h")
	if err != nil {
		t.Fatalf("CreateTTLIndex() failed: %v", err)
	}
}

func TestCreatePartialIndex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var opts IndexOptions
		json.Unmarshal(body, &opts)

		if opts.Type != IndexTypeBTree {
			t.Errorf("expected type 'btree', got '%s'", opts.Type)
		}
		if opts.PartialFilter == nil {
			t.Error("expected partial filter")
		}
		if !opts.Unique {
			t.Error("expected unique to be true")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("users")

	filter := map[string]interface{}{
		"active": true,
	}

	err := coll.CreatePartialIndex("active_email_idx", "email", filter, true)
	if err != nil {
		t.Fatalf("CreatePartialIndex() failed: %v", err)
	}
}
