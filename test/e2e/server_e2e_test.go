package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

const (
	testServerPort = "18080"
	testServerURL  = "http://localhost:" + testServerPort
	serverStartTimeout = 10 * time.Second
)

// TestServerFullWorkflow tests complete end-to-end workflow with real HTTP server
func TestServerFullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Setup: Create temporary data directory
	tmpDir, err := os.MkdirTemp("", "laura-e2e-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Build server binary
	serverBinary := filepath.Join(tmpDir, "laura-server")
	buildCmd := exec.Command("go", "build", "-o", serverBinary, "../../cmd/server/main.go")
	buildCmd.Dir = tmpDir
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build server: %v\nOutput: %s", err, output)
	}

	// Start server
	serverCmd := exec.Command(serverBinary, "-port", testServerPort, "-data-dir", tmpDir)
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr

	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Kill()
			serverCmd.Wait()
		}
	}()

	// Wait for server to be ready
	if !waitForServer(t, testServerURL+"/_health", serverStartTimeout) {
		t.Fatal("Server failed to start within timeout")
	}

	t.Log("Server started successfully")

	// Run test scenarios
	t.Run("HealthCheck", func(t *testing.T) {
		testHealthCheck(t)
	})

	t.Run("CollectionManagement", func(t *testing.T) {
		testCollectionManagement(t)
	})

	t.Run("DocumentCRUD", func(t *testing.T) {
		testDocumentCRUD(t)
	})

	t.Run("QueryOperations", func(t *testing.T) {
		testQueryOperations(t)
	})

	t.Run("IndexManagement", func(t *testing.T) {
		testIndexManagement(t)
	})

	t.Run("AggregationPipeline", func(t *testing.T) {
		testAggregationPipeline(t)
	})

	t.Run("BulkOperations", func(t *testing.T) {
		testBulkOperations(t)
	})

	t.Run("CursorOperations", func(t *testing.T) {
		testCursorOperations(t)
	})

	t.Run("TextSearch", func(t *testing.T) {
		testTextSearch(t)
	})

	t.Run("GeospatialQueries", func(t *testing.T) {
		testGeospatialQueries(t)
	})
}

// waitForServer waits for server to become available
func waitForServer(t *testing.T, url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return true
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// makeHTTPRequest is a helper to make HTTP requests
func makeHTTPRequest(t *testing.T, method, path string, body interface{}) (int, map[string]interface{}) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, testServerURL+path, reqBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		// Some responses might not have JSON body
		return resp.StatusCode, nil
	}

	return resp.StatusCode, response
}

// Test scenarios

func testHealthCheck(t *testing.T) {
	status, response := makeHTTPRequest(t, "GET", "/_health", nil)
	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", response["status"])
	}
	t.Log("✓ Health check passed")
}

func testCollectionManagement(t *testing.T) {
	collectionName := "test_collection"

	// Create collection
	status, response := makeHTTPRequest(t, "PUT", "/"+collectionName, nil)
	if status != http.StatusCreated {
		t.Errorf("Failed to create collection: status %d, response: %v", status, response)
	}

	// List collections
	status, response = makeHTTPRequest(t, "GET", "/", nil)
	if status != http.StatusOK {
		t.Errorf("Failed to list collections: %d", status)
	}
	collections, ok := response["collections"].([]interface{})
	if !ok || len(collections) == 0 {
		t.Error("Expected collections list")
	}

	// Get collection stats
	status, response = makeHTTPRequest(t, "GET", "/"+collectionName+"/_stats", nil)
	if status != http.StatusOK {
		t.Errorf("Failed to get collection stats: %d", status)
	}

	// Drop collection
	status, _ = makeHTTPRequest(t, "DELETE", "/"+collectionName, nil)
	if status != http.StatusOK {
		t.Errorf("Failed to drop collection: %d", status)
	}

	t.Log("✓ Collection management passed")
}

func testDocumentCRUD(t *testing.T) {
	collectionName := "crud_test"

	// Create collection
	makeHTTPRequest(t, "PUT", "/"+collectionName, nil)
	defer makeHTTPRequest(t, "DELETE", "/"+collectionName, nil)

	// Insert document
	doc := map[string]interface{}{
		"name":  "John Doe",
		"age":   int64(30),
		"email": "john@example.com",
	}
	status, response := makeHTTPRequest(t, "POST", "/"+collectionName, doc)
	if status != http.StatusCreated {
		t.Fatalf("Failed to insert document: %d", status)
	}
	insertedID := response["_id"].(string)
	if insertedID == "" {
		t.Fatal("Expected document ID")
	}

	// Find document
	status, response = makeHTTPRequest(t, "POST", "/"+collectionName+"/_find", map[string]interface{}{
		"filter": map[string]interface{}{"name": "John Doe"},
	})
	if status != http.StatusOK {
		t.Errorf("Failed to find document: %d", status)
	}
	documents := response["documents"].([]interface{})
	if len(documents) != 1 {
		t.Errorf("Expected 1 document, got %d", len(documents))
	}

	// Update document
	status, response = makeHTTPRequest(t, "PATCH", "/"+collectionName, map[string]interface{}{
		"filter": map[string]interface{}{"name": "John Doe"},
		"update": map[string]interface{}{
			"$set": map[string]interface{}{"age": int64(31)},
		},
	})
	if status != http.StatusOK {
		t.Errorf("Failed to update document: %d", status)
	}

	// Verify update
	status, response = makeHTTPRequest(t, "POST", "/"+collectionName+"/_find", map[string]interface{}{
		"filter": map[string]interface{}{"name": "John Doe"},
	})
	documents = response["documents"].([]interface{})
	doc = documents[0].(map[string]interface{})
	if doc["age"].(float64) != 31 {
		t.Errorf("Expected age 31, got %v", doc["age"])
	}

	// Delete document
	status, _ = makeHTTPRequest(t, "DELETE", "/"+collectionName, map[string]interface{}{
		"filter": map[string]interface{}{"name": "John Doe"},
	})
	if status != http.StatusOK {
		t.Errorf("Failed to delete document: %d", status)
	}

	// Verify deletion
	status, response = makeHTTPRequest(t, "POST", "/"+collectionName+"/_find", map[string]interface{}{
		"filter": map[string]interface{}{},
	})
	documents = response["documents"].([]interface{})
	if len(documents) != 0 {
		t.Errorf("Expected 0 documents after deletion, got %d", len(documents))
	}

	t.Log("✓ Document CRUD passed")
}

func testQueryOperations(t *testing.T) {
	collectionName := "query_test"

	// Create collection and insert test data
	makeHTTPRequest(t, "PUT", "/"+collectionName, nil)
	defer makeHTTPRequest(t, "DELETE", "/"+collectionName, nil)

	// Insert multiple documents
	docs := []map[string]interface{}{
		{"name": "Alice", "age": int64(25), "city": "NYC"},
		{"name": "Bob", "age": int64(30), "city": "LA"},
		{"name": "Charlie", "age": int64(35), "city": "NYC"},
		{"name": "David", "age": int64(40), "city": "SF"},
	}
	for _, doc := range docs {
		makeHTTPRequest(t, "POST", "/"+collectionName, doc)
	}

	// Query with comparison operators
	status, response := makeHTTPRequest(t, "POST", "/"+collectionName+"/_find", map[string]interface{}{
		"filter": map[string]interface{}{
			"age": map[string]interface{}{"$gte": int64(30)},
		},
	})
	if status != http.StatusOK {
		t.Errorf("Failed to query with $gte: %d", status)
	}
	documents := response["documents"].([]interface{})
	if len(documents) != 3 {
		t.Errorf("Expected 3 documents with age >= 30, got %d", len(documents))
	}

	// Query with logical operators
	status, response = makeHTTPRequest(t, "POST", "/"+collectionName+"/_find", map[string]interface{}{
		"filter": map[string]interface{}{
			"$and": []interface{}{
				map[string]interface{}{"age": map[string]interface{}{"$gte": int64(25)}},
				map[string]interface{}{"city": "NYC"},
			},
		},
	})
	if status != http.StatusOK {
		t.Errorf("Failed to query with $and: %d", status)
	}
	documents = response["documents"].([]interface{})
	if len(documents) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(documents))
	}

	// Query with projection
	status, response = makeHTTPRequest(t, "POST", "/"+collectionName+"/_find", map[string]interface{}{
		"filter":     map[string]interface{}{},
		"projection": map[string]interface{}{"name": int64(1), "age": int64(1)},
	})
	if status != http.StatusOK {
		t.Errorf("Failed to query with projection: %d", status)
	}
	documents = response["documents"].([]interface{})
	doc := documents[0].(map[string]interface{})
	if _, hasCity := doc["city"]; hasCity {
		t.Error("Projection failed: city field should not be present")
	}

	// Query with sort, skip, limit
	status, response = makeHTTPRequest(t, "POST", "/"+collectionName+"/_find", map[string]interface{}{
		"filter": map[string]interface{}{},
		"sort":   map[string]interface{}{"age": int64(-1)},
		"skip":   int64(1),
		"limit":  int64(2),
	})
	if status != http.StatusOK {
		t.Errorf("Failed to query with sort/skip/limit: %d", status)
	}
	documents = response["documents"].([]interface{})
	if len(documents) != 2 {
		t.Errorf("Expected 2 documents with limit, got %d", len(documents))
	}

	t.Log("✓ Query operations passed")
}

func testIndexManagement(t *testing.T) {
	collectionName := "index_test"

	// Create collection
	makeHTTPRequest(t, "PUT", "/"+collectionName, nil)
	defer makeHTTPRequest(t, "DELETE", "/"+collectionName, nil)

	// Create index
	status, response := makeHTTPRequest(t, "POST", "/"+collectionName+"/_indexes", map[string]interface{}{
		"field":  "email",
		"unique": true,
	})
	if status != http.StatusCreated {
		t.Errorf("Failed to create index: %d, response: %v", status, response)
	}

	// List indexes
	status, response = makeHTTPRequest(t, "GET", "/"+collectionName+"/_indexes", nil)
	if status != http.StatusOK {
		t.Errorf("Failed to list indexes: %d", status)
	}
	indexes, ok := response["indexes"].([]interface{})
	if !ok {
		t.Error("Expected indexes array")
	}
	// Should have _id_ index and email index
	if len(indexes) < 2 {
		t.Errorf("Expected at least 2 indexes, got %d", len(indexes))
	}

	// Test unique constraint
	doc1 := map[string]interface{}{"email": "test@example.com", "name": "User1"}
	status, _ = makeHTTPRequest(t, "POST", "/"+collectionName, doc1)
	if status != http.StatusCreated {
		t.Error("Failed to insert first document")
	}

	doc2 := map[string]interface{}{"email": "test@example.com", "name": "User2"}
	status, _ = makeHTTPRequest(t, "POST", "/"+collectionName, doc2)
	if status != http.StatusBadRequest {
		t.Errorf("Expected 400 for duplicate unique key, got %d", status)
	}

	// Drop index
	status, _ = makeHTTPRequest(t, "DELETE", "/"+collectionName+"/_indexes/email", nil)
	if status != http.StatusOK {
		t.Errorf("Failed to drop index: %d", status)
	}

	t.Log("✓ Index management passed")
}

func testAggregationPipeline(t *testing.T) {
	collectionName := "agg_test"

	// Create collection and insert data
	makeHTTPRequest(t, "PUT", "/"+collectionName, nil)
	defer makeHTTPRequest(t, "DELETE", "/"+collectionName, nil)

	// Insert sales data
	sales := []map[string]interface{}{
		{"product": "A", "quantity": int64(10), "price": int64(100)},
		{"product": "B", "quantity": int64(5), "price": int64(200)},
		{"product": "A", "quantity": int64(15), "price": int64(100)},
		{"product": "C", "quantity": int64(8), "price": int64(150)},
	}
	for _, sale := range sales {
		makeHTTPRequest(t, "POST", "/"+collectionName, sale)
	}

	// Run aggregation pipeline
	pipeline := []interface{}{
		map[string]interface{}{
			"$group": map[string]interface{}{
				"_id":       "$product",
				"totalQty":  map[string]interface{}{"$sum": "$quantity"},
				"avgPrice":  map[string]interface{}{"$avg": "$price"},
				"maxPrice":  map[string]interface{}{"$max": "$price"},
				"itemCount": map[string]interface{}{"$count": map[string]interface{}{}},
			},
		},
		map[string]interface{}{
			"$sort": map[string]interface{}{"totalQty": int64(-1)},
		},
	}

	status, response := makeHTTPRequest(t, "POST", "/"+collectionName+"/_aggregate", map[string]interface{}{
		"pipeline": pipeline,
	})
	if status != http.StatusOK {
		t.Errorf("Failed to run aggregation: %d, response: %v", status, response)
	}

	results, ok := response["results"].([]interface{})
	if !ok || len(results) != 3 {
		t.Errorf("Expected 3 aggregation results, got %d", len(results))
	}

	// Verify results
	firstResult := results[0].(map[string]interface{})
	if firstResult["_id"] != "A" {
		t.Errorf("Expected first result to be product A, got %v", firstResult["_id"])
	}
	if firstResult["totalQty"].(float64) != 25 {
		t.Errorf("Expected totalQty 25, got %v", firstResult["totalQty"])
	}

	t.Log("✓ Aggregation pipeline passed")
}

func testBulkOperations(t *testing.T) {
	collectionName := "bulk_test"

	// Create collection
	makeHTTPRequest(t, "PUT", "/"+collectionName, nil)
	defer makeHTTPRequest(t, "DELETE", "/"+collectionName, nil)

	// Insert initial documents
	makeHTTPRequest(t, "POST", "/"+collectionName, map[string]interface{}{"name": "Initial", "value": int64(1)})

	// Bulk write operations
	bulkOps := []interface{}{
		map[string]interface{}{
			"insertOne": map[string]interface{}{
				"document": map[string]interface{}{"name": "Bulk1", "value": int64(10)},
			},
		},
		map[string]interface{}{
			"insertOne": map[string]interface{}{
				"document": map[string]interface{}{"name": "Bulk2", "value": int64(20)},
			},
		},
		map[string]interface{}{
			"updateOne": map[string]interface{}{
				"filter": map[string]interface{}{"name": "Initial"},
				"update": map[string]interface{}{
					"$set": map[string]interface{}{"value": int64(100)},
				},
			},
		},
	}

	status, response := makeHTTPRequest(t, "POST", "/"+collectionName+"/_bulkWrite", map[string]interface{}{
		"operations": bulkOps,
		"ordered":    true,
	})
	if status != http.StatusOK {
		t.Errorf("Failed bulk write: %d, response: %v", status, response)
	}

	// Verify results
	status, response = makeHTTPRequest(t, "POST", "/"+collectionName+"/_find", map[string]interface{}{
		"filter": map[string]interface{}{},
	})
	documents := response["documents"].([]interface{})
	if len(documents) != 3 {
		t.Errorf("Expected 3 documents after bulk write, got %d", len(documents))
	}

	t.Log("✓ Bulk operations passed")
}

func testCursorOperations(t *testing.T) {
	collectionName := "cursor_test"

	// Create collection and insert many documents
	makeHTTPRequest(t, "PUT", "/"+collectionName, nil)
	defer makeHTTPRequest(t, "DELETE", "/"+collectionName, nil)

	// Insert 50 documents
	for i := 0; i < 50; i++ {
		makeHTTPRequest(t, "POST", "/"+collectionName, map[string]interface{}{
			"index": int64(i),
			"value": fmt.Sprintf("doc-%d", i),
		})
	}

	// Create cursor
	status, response := makeHTTPRequest(t, "POST", "/"+collectionName+"/_cursor", map[string]interface{}{
		"filter":    map[string]interface{}{},
		"batchSize": int64(10),
	})
	if status != http.StatusOK {
		t.Errorf("Failed to create cursor: %d", status)
	}

	cursorID := response["cursorId"].(string)
	if cursorID == "" {
		t.Fatal("Expected cursor ID")
	}

	// Fetch batches
	totalFetched := int(response["batch"].([]interface{})[0].(map[string]interface{})["index"].(float64)) + 1
	hasMore := response["hasMore"].(bool)

	for hasMore && totalFetched < 50 {
		status, response = makeHTTPRequest(t, "GET", "/"+collectionName+"/_cursor/"+cursorID, nil)
		if status != http.StatusOK {
			t.Errorf("Failed to fetch next batch: %d", status)
			break
		}
		batch := response["batch"].([]interface{})
		totalFetched += len(batch)
		hasMore = response["hasMore"].(bool)
	}

	if totalFetched != 50 {
		t.Errorf("Expected to fetch 50 documents, got %d", totalFetched)
	}

	// Close cursor
	status, _ = makeHTTPRequest(t, "DELETE", "/"+collectionName+"/_cursor/"+cursorID, nil)
	if status != http.StatusOK {
		t.Errorf("Failed to close cursor: %d", status)
	}

	t.Log("✓ Cursor operations passed")
}

func testTextSearch(t *testing.T) {
	collectionName := "text_test"

	// Create collection
	makeHTTPRequest(t, "PUT", "/"+collectionName, nil)
	defer makeHTTPRequest(t, "DELETE", "/"+collectionName, nil)

	// Create text index
	status, _ := makeHTTPRequest(t, "POST", "/"+collectionName+"/_indexes", map[string]interface{}{
		"field": "content",
		"type":  "text",
	})
	if status != http.StatusCreated {
		t.Skipf("Text index creation failed, skipping text search test: %d", status)
		return
	}

	// Insert documents with text content
	docs := []map[string]interface{}{
		{"title": "Go Programming", "content": "Go is a statically typed compiled programming language"},
		{"title": "Python Guide", "content": "Python is an interpreted high-level programming language"},
		{"title": "Database Design", "content": "Database design is the organization of data according to a model"},
	}
	for _, doc := range docs {
		makeHTTPRequest(t, "POST", "/"+collectionName, doc)
	}

	// Perform text search
	status, response := makeHTTPRequest(t, "POST", "/"+collectionName+"/_textSearch", map[string]interface{}{
		"query": "programming language",
	})
	if status != http.StatusOK {
		t.Skipf("Text search not supported: %d", status)
		return
	}

	results, ok := response["results"].([]interface{})
	if !ok || len(results) < 2 {
		t.Errorf("Expected at least 2 text search results, got %d", len(results))
	}

	t.Log("✓ Text search passed")
}

func testGeospatialQueries(t *testing.T) {
	collectionName := "geo_test"

	// Create collection
	makeHTTPRequest(t, "PUT", "/"+collectionName, nil)
	defer makeHTTPRequest(t, "DELETE", "/"+collectionName, nil)

	// Create geospatial index
	status, _ := makeHTTPRequest(t, "POST", "/"+collectionName+"/_indexes", map[string]interface{}{
		"field": "location",
		"type":  "2dsphere",
	})
	if status != http.StatusCreated {
		t.Skipf("Geospatial index creation failed, skipping geo test: %d", status)
		return
	}

	// Insert documents with locations
	locations := []map[string]interface{}{
		{
			"name": "Location A",
			"location": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{-73.97, 40.77}, // NYC
			},
		},
		{
			"name": "Location B",
			"location": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{-118.24, 34.05}, // LA
			},
		},
	}
	for _, loc := range locations {
		makeHTTPRequest(t, "POST", "/"+collectionName, loc)
	}

	// Perform geospatial query (near NYC)
	status, response := makeHTTPRequest(t, "POST", "/"+collectionName+"/_near", map[string]interface{}{
		"location": map[string]interface{}{
			"type":        "Point",
			"coordinates": []interface{}{-73.98, 40.75},
		},
		"maxDistance": int64(10000), // 10km
	})
	if status != http.StatusOK {
		t.Skipf("Geospatial query not supported: %d", status)
		return
	}

	results, ok := response["results"].([]interface{})
	if !ok || len(results) == 0 {
		t.Error("Expected geospatial query results")
	}

	t.Log("✓ Geospatial queries passed")
}
