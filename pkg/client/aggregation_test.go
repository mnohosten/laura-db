package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCollectionAggregate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected method POST, got %s", r.Method)
		}
		if r.URL.Path != "/users/_aggregate" {
			t.Errorf("expected path '/users/_aggregate', got '%s'", r.URL.Path)
		}

		// Verify pipeline
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)
		pipeline := req["pipeline"].([]interface{})
		if len(pipeline) != 2 {
			t.Errorf("expected 2 stages, got %d", len(pipeline))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"ok": true,
			"result": [
				{"_id": "active", "count": 10},
				{"_id": "inactive", "count": 5}
			]
		}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("users")

	pipeline := AggregationPipeline{
		{"$match": map[string]interface{}{"age": map[string]interface{}{"$gt": 25}}},
		{"$group": map[string]interface{}{
			"_id":   "$status",
			"count": map[string]interface{}{"$sum": 1},
		}},
	}

	results, err := coll.Aggregate(pipeline)
	if err != nil {
		t.Fatalf("Aggregate() failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestPipelineBuilder(t *testing.T) {
	pipeline := NewPipeline().
		Match(map[string]interface{}{"age": map[string]interface{}{"$gt": 25}}).
		Group("$city", map[string]interface{}{
			"avgAge": Avg("age"),
			"total":  Count(),
		}).
		Sort(map[string]interface{}{"total": -1}).
		Limit(10).
		Skip(5).
		Project(map[string]interface{}{
			"city":   1,
			"avgAge": 1,
		}).
		Build()

	if len(pipeline) != 6 {
		t.Errorf("expected 6 stages, got %d", len(pipeline))
	}

	// Verify $match stage
	matchStage := pipeline[0]["$match"].(map[string]interface{})
	if matchStage == nil {
		t.Error("expected $match stage")
	}

	// Verify $group stage
	groupStage := pipeline[1]["$group"].(map[string]interface{})
	if groupStage["_id"] != "$city" {
		t.Errorf("expected group by '$city', got '%v'", groupStage["_id"])
	}

	// Verify $sort stage
	sortStage := pipeline[2]["$sort"]
	if sortStage == nil {
		t.Error("expected $sort stage")
	}

	// Verify $limit stage
	limitStage := pipeline[3]["$limit"]
	if limitStage != 10 {
		t.Errorf("expected limit 10, got %v", limitStage)
	}

	// Verify $skip stage
	skipStage := pipeline[4]["$skip"]
	if skipStage != 5 {
		t.Errorf("expected skip 5, got %v", skipStage)
	}

	// Verify $project stage
	projectStage := pipeline[5]["$project"]
	if projectStage == nil {
		t.Error("expected $project stage")
	}
}

func TestPipelineBuilderExecute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"ok": true,
			"result": [
				{"_id": "NYC", "count": 100}
			]
		}`))
	}))
	defer server.Close()

	client := NewDefaultClient()
	client.baseURL = server.URL
	coll := client.Collection("users")

	results, err := NewPipeline().
		Match(map[string]interface{}{"age": map[string]interface{}{"$gt": 25}}).
		Group("$city", map[string]interface{}{"count": Count()}).
		Execute(coll)

	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestAccumulators(t *testing.T) {
	// Test Sum
	sumAcc := Sum("price")
	if sumAcc["$sum"] != "$price" {
		t.Errorf("expected '$price', got '%v'", sumAcc["$sum"])
	}

	// Test SumValue
	sumValAcc := SumValue(1)
	if sumValAcc["$sum"] != 1 {
		t.Errorf("expected 1, got '%v'", sumValAcc["$sum"])
	}

	// Test Avg
	avgAcc := Avg("age")
	if avgAcc["$avg"] != "$age" {
		t.Errorf("expected '$age', got '%v'", avgAcc["$avg"])
	}

	// Test Min
	minAcc := Min("score")
	if minAcc["$min"] != "$score" {
		t.Errorf("expected '$score', got '%v'", minAcc["$min"])
	}

	// Test Max
	maxAcc := Max("score")
	if maxAcc["$max"] != "$score" {
		t.Errorf("expected '$score', got '%v'", maxAcc["$max"])
	}

	// Test Push
	pushAcc := Push("name")
	if pushAcc["$push"] != "$name" {
		t.Errorf("expected '$name', got '%v'", pushAcc["$push"])
	}

	// Test Count
	countAcc := Count()
	if countAcc["$sum"] != 1 {
		t.Errorf("expected 1, got '%v'", countAcc["$sum"])
	}
}
