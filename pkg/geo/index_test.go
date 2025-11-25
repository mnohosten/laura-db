package geo

import (
	"testing"
)

func TestIndex2D_InsertAndNear(t *testing.T) {
	idx := NewIndex2D(1.0)

	// Insert some points
	idx.Insert("doc1", NewPoint(0, 0))
	idx.Insert("doc2", NewPoint(1, 1))
	idx.Insert("doc3", NewPoint(5, 5))
	idx.Insert("doc4", NewPoint(10, 10))

	// Search near (0, 0) with max distance 2
	results := idx.FindNear(NewPoint(0, 0), 2.0, 10)

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Results should be sorted by distance
	if results[0].DocID != "doc1" {
		t.Errorf("Expected doc1 first, got %s", results[0].DocID)
	}

	if results[1].DocID != "doc2" {
		t.Errorf("Expected doc2 second, got %s", results[1].DocID)
	}
}

func TestIndex2D_Remove(t *testing.T) {
	idx := NewIndex2D(1.0)

	idx.Insert("doc1", NewPoint(0, 0))
	idx.Insert("doc2", NewPoint(1, 1))

	// Remove doc1
	idx.Remove("doc1")

	// Search should only find doc2
	results := idx.FindNear(NewPoint(0, 0), 2.0, 10)

	if len(results) != 1 {
		t.Errorf("Expected 1 result after removal, got %d", len(results))
	}

	if results[0].DocID != "doc2" {
		t.Errorf("Expected doc2, got %s", results[0].DocID)
	}
}

func TestIndex2D_FindWithin(t *testing.T) {
	idx := NewIndex2D(1.0)

	// Insert points
	idx.Insert("inside1", NewPoint(2, 2))
	idx.Insert("inside2", NewPoint(8, 8))
	idx.Insert("outside", NewPoint(15, 15))

	// Create square polygon
	polygon := NewPolygon([][]Point{{
		{Lon: 0, Lat: 0},
		{Lon: 10, Lat: 0},
		{Lon: 10, Lat: 10},
		{Lon: 0, Lat: 10},
		{Lon: 0, Lat: 0},
	}})

	results := idx.FindWithin(polygon)

	if len(results) != 2 {
		t.Errorf("Expected 2 results inside polygon, got %d", len(results))
	}

	// Check that "outside" is not in results
	for _, docID := range results {
		if docID == "outside" {
			t.Error("Point outside polygon was included in results")
		}
	}
}

func TestIndex2D_FindInBox(t *testing.T) {
	idx := NewIndex2D(1.0)

	idx.Insert("inside1", NewPoint(2, 2))
	idx.Insert("inside2", NewPoint(8, 8))
	idx.Insert("outside", NewPoint(15, 15))

	box := &BoundingBox{
		MinLon: 0,
		MaxLon: 10,
		MinLat: 0,
		MaxLat: 10,
	}

	results := idx.FindInBox(box)

	if len(results) != 2 {
		t.Errorf("Expected 2 results in box, got %d", len(results))
	}
}

func TestIndex2D_NearWithLimit(t *testing.T) {
	idx := NewIndex2D(1.0)

	// Insert 10 points
	for i := 0; i < 10; i++ {
		idx.Insert(string(rune('a'+i)), NewPoint(float64(i), float64(i)))
	}

	// Search with limit 3
	results := idx.FindNear(NewPoint(0, 0), 100.0, 3)

	if len(results) != 3 {
		t.Errorf("Expected 3 results with limit, got %d", len(results))
	}

	// Should be the 3 closest points
	if results[0].DocID != "a" || results[1].DocID != "b" || results[2].DocID != "c" {
		t.Error("Limit did not return closest points")
	}
}

func TestIndex2DSphere_InsertAndNear(t *testing.T) {
	idx := NewIndex2DSphere(1.0)

	// Insert cities (longitude, latitude)
	// San Francisco
	idx.Insert("sf", NewPoint(-122.4194, 37.7749))
	// Los Angeles
	idx.Insert("la", NewPoint(-118.2437, 34.0522))
	// New York
	idx.Insert("ny", NewPoint(-74.0060, 40.7128))

	// Search near San Francisco within 1000 km
	results := idx.FindNear(NewPoint(-122.4194, 37.7749), 1000000, 10)

	// Should find SF (distance 0) and LA (distance ~559 km)
	// Should NOT find NY (distance ~4,130 km)
	if len(results) < 2 {
		t.Errorf("Expected at least 2 results (SF and LA), got %d", len(results))
	}

	// First result should be SF with distance ~0
	if results[0].DocID != "sf" {
		t.Errorf("Expected sf first, got %s", results[0].DocID)
	}

	if results[0].Distance > 1.0 {
		t.Errorf("Expected distance to SF to be ~0, got %f", results[0].Distance)
	}
}

func TestIndex2DSphere_ValidateCoordinates(t *testing.T) {
	idx := NewIndex2DSphere(1.0)

	tests := []struct {
		name      string
		lon       float64
		lat       float64
		shouldErr bool
	}{
		{"Valid", -122.4194, 37.7749, false},
		{"Invalid lon > 180", 181.0, 37.7749, true},
		{"Invalid lon < -180", -181.0, 37.7749, true},
		{"Invalid lat > 90", -122.4194, 91.0, true},
		{"Invalid lat < -90", -122.4194, -91.0, true},
		{"Edge case lon 180", 180.0, 0.0, false},
		{"Edge case lat 90", 0.0, 90.0, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := idx.Insert("test", NewPoint(test.lon, test.lat))
			if test.shouldErr && err == nil {
				t.Error("Expected error for invalid coordinates")
			}
			if !test.shouldErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestIndex2DSphere_Remove(t *testing.T) {
	idx := NewIndex2DSphere(1.0)

	idx.Insert("sf", NewPoint(-122.4194, 37.7749))
	idx.Insert("la", NewPoint(-118.2437, 34.0522))

	idx.Remove("sf")

	results := idx.FindNear(NewPoint(-122.4194, 37.7749), 1000000, 10)

	// Should only find LA now
	for _, result := range results {
		if result.DocID == "sf" {
			t.Error("Removed document still in index")
		}
	}
}

func TestIndex2DSphere_FindWithin(t *testing.T) {
	idx := NewIndex2DSphere(1.0)

	// Insert points in California
	idx.Insert("sf", NewPoint(-122.4194, 37.7749))   // San Francisco
	idx.Insert("la", NewPoint(-118.2437, 34.0522))   // Los Angeles
	idx.Insert("nyc", NewPoint(-74.0060, 40.7128))   // New York (outside)

	// Simple bounding box around California
	polygon := NewPolygon([][]Point{{
		{Lon: -125, Lat: 32},
		{Lon: -114, Lat: 32},
		{Lon: -114, Lat: 42},
		{Lon: -125, Lat: 42},
		{Lon: -125, Lat: 32},
	}})

	results := idx.FindWithin(polygon)

	// Should find SF and LA, but not NYC
	if len(results) != 2 {
		t.Errorf("Expected 2 results (SF and LA), got %d", len(results))
	}

	for _, docID := range results {
		if docID == "nyc" {
			t.Error("NYC should not be in California polygon")
		}
	}
}

func TestIndex2DSphere_FindInBox(t *testing.T) {
	idx := NewIndex2DSphere(1.0)

	idx.Insert("sf", NewPoint(-122.4194, 37.7749))
	idx.Insert("la", NewPoint(-118.2437, 34.0522))
	idx.Insert("nyc", NewPoint(-74.0060, 40.7128))

	// Bounding box around California
	box := &BoundingBox{
		MinLon: -125,
		MaxLon: -114,
		MinLat: 32,
		MaxLat: 42,
	}

	results := idx.FindInBox(box)

	if len(results) != 2 {
		t.Errorf("Expected 2 results in California box, got %d", len(results))
	}
}

func TestIndex2DSphere_HaversineAccuracy(t *testing.T) {
	idx := NewIndex2DSphere(1.0)

	// Test well-known city distances
	// San Francisco to Los Angeles: ~559 km
	idx.Insert("sf", NewPoint(-122.4194, 37.7749))
	idx.Insert("la", NewPoint(-118.2437, 34.0522))

	results := idx.FindNear(NewPoint(-122.4194, 37.7749), 1000000, 10)

	// Find LA in results
	var laDistance float64
	for _, result := range results {
		if result.DocID == "la" {
			laDistance = result.Distance
			break
		}
	}

	// LA should be ~559,000 meters from SF (allow 10% error)
	expectedMin := 500000.0
	expectedMax := 615000.0

	if laDistance < expectedMin || laDistance > expectedMax {
		t.Errorf("SF to LA distance = %f meters, expected between %f and %f",
			laDistance, expectedMin, expectedMax)
	}
}

// TestNewIndex2DWithZeroGridSize tests NewIndex2D with invalid grid size
func TestNewIndex2DWithZeroGridSize(t *testing.T) {
	idx := NewIndex2D(0)
	if idx.gridSize != 1.0 {
		t.Errorf("Expected default grid size 1.0, got %f", idx.gridSize)
	}
}

// TestNewIndex2DWithNegativeGridSize tests NewIndex2D with negative grid size
func TestNewIndex2DWithNegativeGridSize(t *testing.T) {
	idx := NewIndex2D(-5.0)
	if idx.gridSize != 1.0 {
		t.Errorf("Expected default grid size 1.0, got %f", idx.gridSize)
	}
}

// TestNewIndex2DSphereWithZeroGridSize tests NewIndex2DSphere with invalid grid size
func TestNewIndex2DSphereWithZeroGridSize(t *testing.T) {
	idx := NewIndex2DSphere(0)
	if idx.gridSize != 1.0 {
		t.Errorf("Expected default grid size 1.0, got %f", idx.gridSize)
	}
}

// TestNewIndex2DSphereWithNegativeGridSize tests NewIndex2DSphere with negative grid size
func TestNewIndex2DSphereWithNegativeGridSize(t *testing.T) {
	idx := NewIndex2DSphere(-5.0)
	if idx.gridSize != 1.0 {
		t.Errorf("Expected default grid size 1.0, got %f", idx.gridSize)
	}
}

// TestIndex2DUpdateBounds tests the updateBounds function
func TestIndex2DUpdateBounds(t *testing.T) {
	idx := NewIndex2D(1.0)

	// Insert first point - should initialize bounds
	idx.Insert("p1", NewPoint(5, 5))
	if idx.bounds.MinLon != 5 || idx.bounds.MaxLon != 5 || idx.bounds.MinLat != 5 || idx.bounds.MaxLat != 5 {
		t.Errorf("First point should set all bounds to (5,5), got min=(%f,%f) max=(%f,%f)",
			idx.bounds.MinLon, idx.bounds.MinLat, idx.bounds.MaxLon, idx.bounds.MaxLat)
	}

	// Insert point that extends bounds
	idx.Insert("p2", NewPoint(10, 15))
	if idx.bounds.MinLon != 5 || idx.bounds.MaxLon != 10 || idx.bounds.MinLat != 5 || idx.bounds.MaxLat != 15 {
		t.Errorf("Expected bounds min=(5,5) max=(10,15), got min=(%f,%f) max=(%f,%f)",
			idx.bounds.MinLon, idx.bounds.MinLat, idx.bounds.MaxLon, idx.bounds.MaxLat)
	}

	// Insert point that extends bounds in opposite direction
	idx.Insert("p3", NewPoint(0, 0))
	if idx.bounds.MinLon != 0 || idx.bounds.MaxLon != 10 || idx.bounds.MinLat != 0 || idx.bounds.MaxLat != 15 {
		t.Errorf("Expected bounds min=(0,0) max=(10,15), got min=(%f,%f) max=(%f,%f)",
			idx.bounds.MinLon, idx.bounds.MinLat, idx.bounds.MaxLon, idx.bounds.MaxLat)
	}
}

// TestPolygonBoundsWithEmptyRings tests bounds calculation with empty polygon
func TestPolygonBoundsWithEmptyRings(t *testing.T) {
	polygon := NewPolygon([][]Point{})
	bounds := polygon.Bounds()

	// Empty polygon should return nil bounds
	if bounds != nil {
		t.Errorf("Empty polygon should return nil bounds, got %v", bounds)
	}
}

// TestPolygonBoundsWithMultipleRings tests bounds calculation with multiple rings (holes)
func TestPolygonBoundsWithMultipleRings(t *testing.T) {
	outer := []Point{
		{Lon: 0, Lat: 0},
		{Lon: 20, Lat: 0},
		{Lon: 20, Lat: 20},
		{Lon: 0, Lat: 20},
		{Lon: 0, Lat: 0},
	}

	// Hole is inside outer ring, should not affect bounds
	hole := []Point{
		{Lon: 5, Lat: 5},
		{Lon: 10, Lat: 5},
		{Lon: 10, Lat: 10},
		{Lon: 5, Lat: 10},
		{Lon: 5, Lat: 5},
	}

	polygon := NewPolygon([][]Point{outer, hole})
	bounds := polygon.Bounds()

	// Bounds should be determined by outer ring only
	if bounds.MinLon != 0 || bounds.MaxLon != 20 || bounds.MinLat != 0 || bounds.MaxLat != 20 {
		t.Errorf("Expected bounds min=(0,0) max=(20,20), got min=(%f,%f) max=(%f,%f)",
			bounds.MinLon, bounds.MinLat, bounds.MaxLon, bounds.MaxLat)
	}
}
