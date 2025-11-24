package database

import (
	"fmt"
	"os"
	"testing"

	"github.com/mnohosten/laura-db/pkg/geo"
)

func TestCreate2DIndex(t *testing.T) {
	dir := "./test_geo_2d"
	defer os.RemoveAll(dir)

	db, err := Open(DefaultConfig(dir))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	coll := db.Collection("locations")

	// Insert documents with location data
	_, err = coll.InsertOne(map[string]interface{}{
		"name": "Location A",
		"location": map[string]interface{}{
			"type":        "Point",
			"coordinates": []interface{}{10.0, 20.0},
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Create 2d index
	err = coll.Create2DIndex("location")
	if err != nil {
		t.Fatalf("Failed to create 2d index: %v", err)
	}

	// Verify index exists
	indexes := coll.ListIndexes()
	found := false
	for _, idx := range indexes {
		if idx["name"] == "location_2d" {
			found = true
			break
		}
	}

	if !found {
		t.Error("2d index not found in index list")
	}
}

func TestCreate2DSphereIndex(t *testing.T) {
	dir := "./test_geo_2dsphere"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("places")

	// Create 2dsphere index
	err := coll.Create2DSphereIndex("coords")
	if err != nil {
		t.Fatalf("Failed to create 2dsphere index: %v", err)
	}

	// Verify index exists
	indexes := coll.ListIndexes()
	found := false
	for _, idx := range indexes {
		if idx["name"] == "coords_2dsphere" {
			found = true
			if idx["type"] != "2dsphere" {
				t.Errorf("Expected type 2dsphere, got %v", idx["type"])
			}
			break
		}
	}

	if !found {
		t.Error("2dsphere index not found in index list")
	}
}

func TestNearQuery2D(t *testing.T) {
	dir := "./test_near_2d"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("points")

	// Insert documents
	coll.InsertOne(map[string]interface{}{
		"name": "Point A",
		"loc": map[string]interface{}{
			"type":        "Point",
			"coordinates": []interface{}{0.0, 0.0},
		},
	})

	coll.InsertOne(map[string]interface{}{
		"name": "Point B",
		"loc": map[string]interface{}{
			"type":        "Point",
			"coordinates": []interface{}{1.0, 1.0},
		},
	})

	coll.InsertOne(map[string]interface{}{
		"name": "Point C",
		"loc": map[string]interface{}{
			"type":        "Point",
			"coordinates": []interface{}{10.0, 10.0},
		},
	})

	// Create 2d index
	coll.Create2DIndex("loc")

	// Query near (0, 0) with max distance 2
	center := geo.NewPoint(0, 0)
	results, err := coll.Near("loc", center, 2.0, 10, nil)
	if err != nil {
		t.Fatalf("Near query failed: %v", err)
	}

	// Should find Point A and Point B, but not Point C
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Verify distance metadata is included
	if _, exists := results[0].Get("_distance"); !exists {
		t.Error("Expected _distance field in results")
	}
}

func TestNearQuery2DSphere(t *testing.T) {
	dir := "./test_near_2dsphere"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("cities")

	// Insert cities
	coll.InsertOne(map[string]interface{}{
		"name": "San Francisco",
		"coords": map[string]interface{}{
			"type":        "Point",
			"coordinates": []interface{}{-122.4194, 37.7749},
		},
	})

	coll.InsertOne(map[string]interface{}{
		"name": "Los Angeles",
		"coords": map[string]interface{}{
			"type":        "Point",
			"coordinates": []interface{}{-118.2437, 34.0522},
		},
	})

	coll.InsertOne(map[string]interface{}{
		"name": "New York",
		"coords": map[string]interface{}{
			"type":        "Point",
			"coordinates": []interface{}{-74.0060, 40.7128},
		},
	})

	// Create 2dsphere index
	coll.Create2DSphereIndex("coords")

	// Query near San Francisco within 1000 km
	sf := geo.NewPoint(-122.4194, 37.7749)
	results, err := coll.Near("coords", sf, 1000000, 10, nil)
	if err != nil {
		t.Fatalf("Near query failed: %v", err)
	}

	// Should find SF and LA, but not NY
	if len(results) < 2 {
		t.Errorf("Expected at least 2 results, got %d", len(results))
	}

	// First result should be SF (distance 0)
	name, _ := results[0].Get("name")
	if name != "San Francisco" {
		t.Errorf("Expected San Francisco first, got %s", name)
	}

	distance, _ := results[0].Get("_distance")
	if distance.(float64) > 1.0 {
		t.Errorf("Expected distance to SF to be ~0, got %f", distance)
	}
}

func TestGeoWithinQuery(t *testing.T) {
	dir := "./test_within"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("locations")

	// Insert points
	coll.InsertOne(map[string]interface{}{
		"name": "Inside",
		"loc": map[string]interface{}{
			"type":        "Point",
			"coordinates": []interface{}{5.0, 5.0},
		},
	})

	coll.InsertOne(map[string]interface{}{
		"name": "Outside",
		"loc": map[string]interface{}{
			"type":        "Point",
			"coordinates": []interface{}{15.0, 15.0},
		},
	})

	// Create 2d index
	coll.Create2DIndex("loc")

	// Create polygon (square from 0,0 to 10,10)
	polygon := geo.NewPolygon([][]geo.Point{{
		{Lon: 0, Lat: 0},
		{Lon: 10, Lat: 0},
		{Lon: 10, Lat: 10},
		{Lon: 0, Lat: 10},
		{Lon: 0, Lat: 0},
	}})

	// Query within polygon
	results, err := coll.GeoWithin("loc", polygon, nil)
	if err != nil {
		t.Fatalf("GeoWithin query failed: %v", err)
	}

	// Should only find "Inside"
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	name, _ := results[0].Get("name")
	if name != "Inside" {
		t.Errorf("Expected 'Inside', got %s", name)
	}
}

func TestGeoIntersectsQuery(t *testing.T) {
	dir := "./test_intersects"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("points")

	// Insert points
	coll.InsertOne(map[string]interface{}{
		"name": "Point A",
		"pos": map[string]interface{}{
			"type":        "Point",
			"coordinates": []interface{}{5.0, 5.0},
		},
	})

	coll.InsertOne(map[string]interface{}{
		"name": "Point B",
		"pos": map[string]interface{}{
			"type":        "Point",
			"coordinates": []interface{}{15.0, 15.0},
		},
	})

	// Create 2d index
	coll.Create2DIndex("pos")

	// Create bounding box
	box := &geo.BoundingBox{
		MinLon: 0,
		MaxLon: 10,
		MinLat: 0,
		MaxLat: 10,
	}

	// Query intersects
	results, err := coll.GeoIntersects("pos", box, nil)
	if err != nil {
		t.Fatalf("GeoIntersects query failed: %v", err)
	}

	// Should only find Point A
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	name, _ := results[0].Get("name")
	if name != "Point A" {
		t.Errorf("Expected 'Point A', got %s", name)
	}
}

func TestGeoIndexMaintenance(t *testing.T) {
	dir := "./test_geo_maintenance"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("places")

	// Create 2d index
	coll.Create2DIndex("loc")

	// Insert document
	coll.InsertOne(map[string]interface{}{
		"name": "Place A",
		"loc": map[string]interface{}{
			"type":        "Point",
			"coordinates": []interface{}{5.0, 5.0},
		},
	})

	// Query should find it
	center := geo.NewPoint(5, 5)
	results, _ := coll.Near("loc", center, 1.0, 10, nil)
	if len(results) != 1 {
		t.Fatalf("Expected 1 result after insert, got %d", len(results))
	}

	// Update location
	coll.UpdateOne(
		map[string]interface{}{"name": "Place A"},
		map[string]interface{}{
			"$set": map[string]interface{}{
				"loc": map[string]interface{}{
					"type":        "Point",
					"coordinates": []interface{}{15.0, 15.0},
				},
			},
		},
	)

	// Old location should not find it
	results, _ = coll.Near("loc", center, 1.0, 10, nil)
	if len(results) != 0 {
		t.Errorf("Expected 0 results after update at old location, got %d", len(results))
	}

	// New location should find it
	newCenter := geo.NewPoint(15, 15)
	results, _ = coll.Near("loc", newCenter, 1.0, 10, nil)
	if len(results) != 1 {
		t.Errorf("Expected 1 result after update at new location, got %d", len(results))
	}

	// Delete document
	coll.DeleteOne(map[string]interface{}{"name": "Place A"})

	// Should not find it anymore
	results, _ = coll.Near("loc", newCenter, 1.0, 10, nil)
	if len(results) != 0 {
		t.Errorf("Expected 0 results after delete, got %d", len(results))
	}
}

func TestGeoQueryWithOptions(t *testing.T) {
	dir := "./test_geo_options"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("data")

	// Insert multiple documents
	for i := 0; i < 10; i++ {
		coll.InsertOne(map[string]interface{}{
			"num": i,
			"loc": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{float64(i), float64(i)},
			},
			"extra": "data",
		})
	}

	// Create 2d index
	coll.Create2DIndex("loc")

	// Query with projection
	center := geo.NewPoint(0, 0)
	results, _ := coll.Near("loc", center, 100.0, 5, &QueryOptions{
		Projection: map[string]bool{
			"num": true,
		},
	})

	if len(results) > 5 {
		t.Errorf("Limit not respected: expected max 5 results, got %d", len(results))
	}

	// Check projection
	if _, exists := results[0].Get("extra"); exists {
		t.Error("Projection not applied: 'extra' field should not be present")
	}

	if _, exists := results[0].Get("num"); !exists {
		t.Error("Projection not applied: 'num' field should be present")
	}
}

func TestDropGeoIndex(t *testing.T) {
	dir := "./test_drop_geo"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("test")

	// Create 2d index
	coll.Create2DIndex("location")

	// Verify it exists
	indexes := coll.ListIndexes()
	if len(indexes) < 2 { // _id_ + location_2d
		t.Fatal("Index not created")
	}

	// Drop the index
	err := coll.DropIndex("location_2d")
	if err != nil {
		t.Fatalf("Failed to drop index: %v", err)
	}

	// Verify it's gone
	indexes = coll.ListIndexes()
	for _, idx := range indexes {
		if idx["name"] == "location_2d" {
			t.Error("Index still exists after drop")
		}
	}
}

func TestGeoIndexNoIndexError(t *testing.T) {
	dir := "./test_geo_noindex"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("test")

	// Try to query without creating index
	center := geo.NewPoint(0, 0)
	_, err := coll.Near("loc", center, 10.0, 10, nil)

	if err == nil {
		t.Error("Expected error when querying without geospatial index")
	}
}

// TestGeoWithin_Projection tests GeoWithin with projection options
func TestGeoWithin_Projection(t *testing.T) {
	dir := "./test_geo_within_projection"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	places := db.Collection("places")
	places.Create2DIndex("location")

	// Insert test data
	places.InsertOne(map[string]interface{}{
		"name":   "Place A",
		"secret": "hidden",
		"location": map[string]interface{}{
			"type":        "Point",
			"coordinates": []interface{}{1.0, 1.0},
		},
	})
	places.InsertOne(map[string]interface{}{
		"name":   "Place B",
		"secret": "confidential",
		"location": map[string]interface{}{
			"type":        "Point",
			"coordinates": []interface{}{1.5, 1.5},
		},
	})

	// Create polygon
	polygon := geo.NewPolygon([][]geo.Point{{
		{Lon: 0, Lat: 0},
		{Lon: 3, Lat: 0},
		{Lon: 3, Lat: 3},
		{Lon: 0, Lat: 3},
		{Lon: 0, Lat: 0},
	}})

	// Query with projection (include only name)
	results, err := places.GeoWithin("location", polygon, &QueryOptions{
		Projection: map[string]bool{
			"name": true,
		},
	})

	if err != nil {
		t.Fatalf("GeoWithin failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Verify projection worked - secret should not be present
	for _, doc := range results {
		if doc.Has("secret") {
			t.Error("Expected secret field to be excluded by projection")
		}
		if !doc.Has("name") {
			t.Error("Expected name field to be included")
		}
		if !doc.Has("_id") {
			t.Error("Expected _id to be included by default")
		}
	}

	// Test with _id exclusion
	results2, _ := places.GeoWithin("location", polygon, &QueryOptions{
		Projection: map[string]bool{
			"name": true,
			"_id":  false,
		},
	})

	for _, doc := range results2 {
		if doc.Has("_id") {
			t.Error("Expected _id to be excluded when explicitly set to false")
		}
	}
}

// TestGeoWithin_SkipLimit tests GeoWithin with skip and limit options
func TestGeoWithin_SkipLimit(t *testing.T) {
	dir := "./test_geo_within_skiplimit"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	places := db.Collection("places")
	places.Create2DIndex("location")

	// Insert 5 test points
	for i := 0; i < 5; i++ {
		places.InsertOne(map[string]interface{}{
			"name": fmt.Sprintf("Place %d", i),
			"location": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{float64(i), float64(i)},
			},
		})
	}

	polygon := geo.NewPolygon([][]geo.Point{{
		{Lon: -1, Lat: -1},
		{Lon: 10, Lat: -1},
		{Lon: 10, Lat: 10},
		{Lon: -1, Lat: 10},
		{Lon: -1, Lat: -1},
	}})

	// Test with limit
	results, _ := places.GeoWithin("location", polygon, &QueryOptions{
		Limit: 2,
	})
	if len(results) != 2 {
		t.Errorf("Expected 2 results with limit, got %d", len(results))
	}

	// Test with skip
	results2, _ := places.GeoWithin("location", polygon, &QueryOptions{
		Skip: 2,
	})
	if len(results2) != 3 {
		t.Errorf("Expected 3 results with skip, got %d", len(results2))
	}

	// Test with skip >= len(docs) - should return empty
	results3, _ := places.GeoWithin("location", polygon, &QueryOptions{
		Skip: 10,
	})
	if len(results3) != 0 {
		t.Errorf("Expected 0 results with skip >= total, got %d", len(results3))
	}

	// Test with both skip and limit
	results4, _ := places.GeoWithin("location", polygon, &QueryOptions{
		Skip:  1,
		Limit: 2,
	})
	if len(results4) != 2 {
		t.Errorf("Expected 2 results with skip and limit, got %d", len(results4))
	}
}

// TestGeoIntersects_Projection tests GeoIntersects with projection options
func TestGeoIntersects_Projection(t *testing.T) {
	dir := "./test_geo_intersects_projection"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	places := db.Collection("places")
	places.Create2DIndex("location")

	// Insert test data
	places.InsertOne(map[string]interface{}{
		"name":   "Place X",
		"hidden": "data",
		"location": map[string]interface{}{
			"type":        "Point",
			"coordinates": []interface{}{5.0, 5.0},
		},
	})

	bbox := &geo.BoundingBox{
		MinLon: 0, MinLat: 0,
		MaxLon: 10, MaxLat: 10,
	}

	// Query with projection
	results, err := places.GeoIntersects("location", bbox, &QueryOptions{
		Projection: map[string]bool{
			"name": true,
		},
	})

	if err != nil {
		t.Fatalf("GeoIntersects failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected at least 1 result")
	}

	// Verify projection
	for _, doc := range results {
		if doc.Has("hidden") {
			t.Error("Expected hidden field to be excluded")
		}
		if !doc.Has("name") {
			t.Error("Expected name field to be included")
		}
	}
}

// TestGeoIntersects_SkipLimit tests GeoIntersects with skip and limit options
func TestGeoIntersects_SkipLimit(t *testing.T) {
	dir := "./test_geo_intersects_skiplimit"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	places := db.Collection("places")
	places.Create2DIndex("location")

	// Insert multiple points
	for i := 0; i < 5; i++ {
		places.InsertOne(map[string]interface{}{
			"name": fmt.Sprintf("Place %d", i),
			"location": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{float64(i), float64(i)},
			},
		})
	}

	bbox := &geo.BoundingBox{
		MinLon: -1, MinLat: -1,
		MaxLon: 10, MaxLat: 10,
	}

	// Test with limit
	results, _ := places.GeoIntersects("location", bbox, &QueryOptions{
		Limit: 3,
	})
	if len(results) != 3 {
		t.Errorf("Expected 3 results with limit, got %d", len(results))
	}

	// Test with skip
	results2, _ := places.GeoIntersects("location", bbox, &QueryOptions{
		Skip: 1,
	})
	if len(results2) != 4 {
		t.Errorf("Expected 4 results with skip, got %d", len(results2))
	}

	// Test with skip >= total
	results3, _ := places.GeoIntersects("location", bbox, &QueryOptions{
		Skip: 20,
	})
	if len(results3) != 0 {
		t.Errorf("Expected 0 results with large skip, got %d", len(results3))
	}
}
