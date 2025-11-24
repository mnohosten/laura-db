package index

import (
	"testing"

	"github.com/mnohosten/laura-db/pkg/geo"
)

func TestNewGeoIndex_2D(t *testing.T) {
	gi := NewGeoIndex("geo_idx", "location", IndexType2D)
	if gi == nil {
		t.Fatal("NewGeoIndex returned nil")
	}
	if gi.Name() != "geo_idx" {
		t.Errorf("Expected name 'geo_idx', got '%s'", gi.Name())
	}
	if gi.FieldPath() != "location" {
		t.Errorf("Expected field path 'location', got '%s'", gi.FieldPath())
	}
	if gi.Type() != IndexType2D {
		t.Errorf("Expected type IndexType2D, got %v", gi.Type())
	}

	fieldPaths := gi.FieldPaths()
	if len(fieldPaths) != 1 || fieldPaths[0] != "location" {
		t.Errorf("Expected field paths ['location'], got %v", fieldPaths)
	}
}

func TestNewGeoIndex_2DSphere(t *testing.T) {
	gi := NewGeoIndex("geo_idx", "coordinates", IndexType2DSphere)
	if gi == nil {
		t.Fatal("NewGeoIndex returned nil")
	}
	if gi.Type() != IndexType2DSphere {
		t.Errorf("Expected type IndexType2DSphere, got %v", gi.Type())
	}
}

func TestGeoIndex_Index2D(t *testing.T) {
	gi := NewGeoIndex("geo_idx", "location", IndexType2D)

	// Index some points
	p1 := geo.NewPoint(0.0, 0.0)
	p2 := geo.NewPoint(1.0, 1.0)
	p3 := geo.NewPoint(5.0, 5.0)

	err := gi.Index("doc1", p1)
	if err != nil {
		t.Fatalf("Failed to index doc1: %v", err)
	}

	err = gi.Index("doc2", p2)
	if err != nil {
		t.Fatalf("Failed to index doc2: %v", err)
	}

	err = gi.Index("doc3", p3)
	if err != nil {
		t.Fatalf("Failed to index doc3: %v", err)
	}
}

func TestGeoIndex_Index2DSphere(t *testing.T) {
	gi := NewGeoIndex("geo_idx", "location", IndexType2DSphere)

	// Index some geographic points (longitude, latitude)
	// New York: -74.006, 40.7128
	// Los Angeles: -118.2437, 34.0522
	p1 := geo.NewPoint(-74.006, 40.7128)
	p2 := geo.NewPoint(-118.2437, 34.0522)

	err := gi.Index("nyc", p1)
	if err != nil {
		t.Fatalf("Failed to index NYC: %v", err)
	}

	err = gi.Index("la", p2)
	if err != nil {
		t.Fatalf("Failed to index LA: %v", err)
	}
}

func TestGeoIndex_Near2D(t *testing.T) {
	gi := NewGeoIndex("geo_idx", "location", IndexType2D)

	// Index points in a grid
	gi.Index("p1", geo.NewPoint(0.0, 0.0))
	gi.Index("p2", geo.NewPoint(1.0, 1.0))
	gi.Index("p3", geo.NewPoint(5.0, 5.0))
	gi.Index("p4", geo.NewPoint(10.0, 10.0))

	// Find points near (0, 0) within distance 2
	center := geo.NewPoint(0.0, 0.0)
	results := gi.Near(center, 2.0, 10)

	if len(results) == 0 {
		t.Fatal("Expected to find nearby points")
	}

	// p1 and p2 should be within distance 2 of origin
	// p3 and p4 are farther away
	foundP1, foundP2 := false, false
	for _, result := range results {
		if result.DocID == "p1" {
			foundP1 = true
		}
		if result.DocID == "p2" {
			foundP2 = true
		}
	}

	if !foundP1 {
		t.Error("Expected to find p1 near origin")
	}
	// Note: p2 at (1,1) has distance sqrt(2) ≈ 1.414, which is within 2.0
	if !foundP2 {
		t.Error("Expected to find p2 near origin")
	}
}

func TestGeoIndex_Within2D(t *testing.T) {
	gi := NewGeoIndex("geo_idx", "location", IndexType2D)

	// Index points
	gi.Index("inside1", geo.NewPoint(1.0, 1.0))
	gi.Index("inside2", geo.NewPoint(2.0, 2.0))
	gi.Index("outside", geo.NewPoint(10.0, 10.0))

	// Create a square polygon from (0,0) to (5,5)
	polygon := geo.NewPolygon([][]geo.Point{
		{
			*geo.NewPoint(0.0, 0.0),
			*geo.NewPoint(5.0, 0.0),
			*geo.NewPoint(5.0, 5.0),
			*geo.NewPoint(0.0, 5.0),
			*geo.NewPoint(0.0, 0.0), // Close the polygon
		},
	})

	results := gi.Within(polygon)

	if len(results) < 2 {
		t.Fatalf("Expected to find at least 2 points inside polygon, got %d", len(results))
	}

	// Check that inside points are found
	foundInside1, foundInside2 := false, false
	for _, docID := range results {
		if docID == "inside1" {
			foundInside1 = true
		}
		if docID == "inside2" {
			foundInside2 = true
		}
		if docID == "outside" {
			t.Error("Found 'outside' point which should not be in polygon")
		}
	}

	if !foundInside1 || !foundInside2 {
		t.Error("Expected to find both inside1 and inside2")
	}
}

func TestGeoIndex_InBox2D(t *testing.T) {
	gi := NewGeoIndex("geo_idx", "location", IndexType2D)

	// Index points
	gi.Index("in1", geo.NewPoint(1.0, 1.0))
	gi.Index("in2", geo.NewPoint(2.0, 2.0))
	gi.Index("out", geo.NewPoint(10.0, 10.0))

	// Create bounding box from (0,0) to (5,5)
	box := &geo.BoundingBox{
		MinLon: 0.0,
		MinLat: 0.0,
		MaxLon: 5.0,
		MaxLat: 5.0,
	}

	results := gi.InBox(box)

	if len(results) < 2 {
		t.Fatalf("Expected to find at least 2 points in box, got %d", len(results))
	}

	// Verify correct points found
	foundIn1, foundIn2 := false, false
	for _, docID := range results {
		if docID == "in1" {
			foundIn1 = true
		}
		if docID == "in2" {
			foundIn2 = true
		}
		if docID == "out" {
			t.Error("Found 'out' point which should not be in box")
		}
	}

	if !foundIn1 || !foundIn2 {
		t.Error("Expected to find both in1 and in2")
	}
}

func TestGeoIndex_Remove(t *testing.T) {
	gi := NewGeoIndex("geo_idx", "location", IndexType2D)

	// Index points
	p1 := geo.NewPoint(1.0, 1.0)
	p2 := geo.NewPoint(2.0, 2.0)

	gi.Index("doc1", p1)
	gi.Index("doc2", p2)

	// Find nearby points
	center := geo.NewPoint(0.0, 0.0)
	results := gi.Near(center, 10.0, 10)

	if len(results) != 2 {
		t.Fatalf("Expected 2 points before removal, got %d", len(results))
	}

	// Remove doc1
	gi.Remove("doc1")

	// Find again
	results = gi.Near(center, 10.0, 10)

	if len(results) != 1 {
		t.Fatalf("Expected 1 point after removal, got %d", len(results))
	}

	if results[0].DocID != "doc2" {
		t.Errorf("Expected doc2 after removal, got %s", results[0].DocID)
	}
}

func TestGeoIndex_Stats(t *testing.T) {
	gi := NewGeoIndex("test_geo", "location", IndexType2D)

	// Index some points
	gi.Index("doc1", geo.NewPoint(1.0, 1.0))
	gi.Index("doc2", geo.NewPoint(2.0, 2.0))

	stats := gi.Stats()

	// Verify stats structure
	if stats["name"] != "test_geo" {
		t.Errorf("Expected name 'test_geo', got %v", stats["name"])
	}

	if stats["field_path"] != "location" {
		t.Errorf("Expected field_path 'location', got %v", stats["field_path"])
	}

	if stats["type"] != "2d" {
		t.Errorf("Expected type '2d', got %v", stats["type"])
	}

	// Test 2dsphere stats
	gi2 := NewGeoIndex("test_2ds", "coords", IndexType2DSphere)
	stats2 := gi2.Stats()
	if stats2["type"] != "2dsphere" {
		t.Errorf("Expected type '2dsphere', got %v", stats2["type"])
	}
}

func TestGeoIndex_Analyze(t *testing.T) {
	gi := NewGeoIndex("geo_idx", "location", IndexType2D)

	// Index points
	gi.Index("doc1", geo.NewPoint(1.0, 1.0))

	// Analyze should update statistics
	gi.Analyze()

	// Verify stats are updated (implementation detail)
	stats := gi.Stats()
	if stats == nil {
		t.Error("Stats should not be nil after Analyze")
	}
}

func TestGeoIndex_Concurrent(t *testing.T) {
	gi := NewGeoIndex("geo_idx", "location", IndexType2D)

	// Index initial point
	gi.Index("doc1", geo.NewPoint(0.0, 0.0))

	// Test concurrent reads
	done := make(chan bool, 3)
	center := geo.NewPoint(0.0, 0.0)

	for i := 0; i < 3; i++ {
		go func() {
			results := gi.Near(center, 10.0, 10)
			if len(results) == 0 {
				t.Error("Expected to find points in concurrent search")
			}
			done <- true
		}()
	}

	// Wait for reads
	for i := 0; i < 3; i++ {
		<-done
	}

	// Test concurrent writes
	for i := 0; i < 3; i++ {
		go func(id int) {
			p := geo.NewPoint(float64(id), float64(id))
			gi.Index("concurrent_"+string(rune('A'+id)), p)
			done <- true
		}(i)
	}

	// Wait for writes
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify all points indexed
	results := gi.Near(center, 100.0, 100)
	if len(results) < 1 {
		t.Error("Expected to find points after concurrent writes")
	}
}

func TestGeoIndex_LimitResults(t *testing.T) {
	gi := NewGeoIndex("geo_idx", "location", IndexType2D)

	// Index many points
	for i := 0; i < 10; i++ {
		p := geo.NewPoint(float64(i), float64(i))
		gi.Index("doc"+string(rune('0'+i)), p)
	}

	// Search with limit
	center := geo.NewPoint(0.0, 0.0)
	results := gi.Near(center, 100.0, 3)

	if len(results) > 3 {
		t.Errorf("Expected at most 3 results with limit=3, got %d", len(results))
	}

	// Results should be sorted by distance (closest first)
	for i := 0; i < len(results)-1; i++ {
		if results[i].Distance > results[i+1].Distance {
			t.Error("Results should be sorted by distance")
		}
	}
}

func TestGeoIndex_2DSphere_Operations(t *testing.T) {
	// Test 2DSphere specific operations
	gi := NewGeoIndex("geo_sphere", "coords", IndexType2DSphere)

	// Index some points
	p1 := geo.NewPoint(0.0, 0.0)
	p2 := geo.NewPoint(1.0, 1.0)
	p3 := geo.NewPoint(5.0, 5.0)

	gi.Index("point1", p1)
	gi.Index("point2", p2)
	gi.Index("point3", p3)

	// Test Near for 2DSphere
	results := gi.Near(geo.NewPoint(0.0, 0.0), 200000.0, 10) // 200km radius
	if len(results) == 0 {
		t.Error("Expected to find points within 200km")
	}

	// Test Within for 2DSphere
	polygon := geo.NewPolygon([][]geo.Point{
		{
			*geo.NewPoint(-1.0, -1.0),
			*geo.NewPoint(2.0, -1.0),
			*geo.NewPoint(2.0, 2.0),
			*geo.NewPoint(-1.0, 2.0),
			*geo.NewPoint(-1.0, -1.0),
		},
	})
	within := gi.Within(polygon)
	if len(within) == 0 {
		t.Error("Expected to find points within polygon")
	}

	// Test InBox for 2DSphere
	box := &geo.BoundingBox{
		MinLon: -1.0,
		MinLat: -1.0,
		MaxLon: 2.0,
		MaxLat: 2.0,
	}
	inBox := gi.InBox(box)
	if len(inBox) == 0 {
		t.Error("Expected to find points in box")
	}
}
