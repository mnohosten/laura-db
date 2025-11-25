package geo

import (
	"math"
	"testing"
)

func TestPointCreation(t *testing.T) {
	p := NewPoint(10.5, 20.3)

	if p.Lon != 10.5 {
		t.Errorf("Expected Lon=10.5, got %f", p.Lon)
	}

	if p.Lat != 20.3 {
		t.Errorf("Expected Lat=20.3, got %f", p.Lat)
	}
}

func TestPointBounds(t *testing.T) {
	p := NewPoint(10, 20)
	bounds := p.Bounds()

	if bounds.MinLon != 10 || bounds.MaxLon != 10 {
		t.Error("Point bounds longitude incorrect")
	}

	if bounds.MinLat != 20 || bounds.MaxLat != 20 {
		t.Error("Point bounds latitude incorrect")
	}
}

func TestPolygonCreation(t *testing.T) {
	ring := []Point{
		{Lon: 0, Lat: 0},
		{Lon: 10, Lat: 0},
		{Lon: 10, Lat: 10},
		{Lon: 0, Lat: 10},
		{Lon: 0, Lat: 0}, // Closing point
	}

	polygon := NewPolygon([][]Point{ring})

	if len(polygon.Rings) != 1 {
		t.Errorf("Expected 1 ring, got %d", len(polygon.Rings))
	}

	if len(polygon.Rings[0]) != 5 {
		t.Errorf("Expected 5 points in ring, got %d", len(polygon.Rings[0]))
	}
}

func TestPolygonBounds(t *testing.T) {
	ring := []Point{
		{Lon: 0, Lat: 0},
		{Lon: 10, Lat: 0},
		{Lon: 10, Lat: 10},
		{Lon: 0, Lat: 10},
		{Lon: 0, Lat: 0},
	}

	polygon := NewPolygon([][]Point{ring})
	bounds := polygon.Bounds()

	if bounds.MinLon != 0 || bounds.MaxLon != 10 {
		t.Errorf("Polygon bounds longitude incorrect: min=%f, max=%f", bounds.MinLon, bounds.MaxLon)
	}

	if bounds.MinLat != 0 || bounds.MaxLat != 10 {
		t.Errorf("Polygon bounds latitude incorrect: min=%f, max=%f", bounds.MinLat, bounds.MaxLat)
	}
}

func TestBoundingBoxContains(t *testing.T) {
	box := &BoundingBox{MinLon: 0, MaxLon: 10, MinLat: 0, MaxLat: 10}

	tests := []struct {
		point    *Point
		expected bool
	}{
		{NewPoint(5, 5), true},     // Inside
		{NewPoint(0, 0), true},     // On corner
		{NewPoint(10, 10), true},   // On corner
		{NewPoint(-1, 5), false},   // Outside (left)
		{NewPoint(11, 5), false},   // Outside (right)
		{NewPoint(5, -1), false},   // Outside (bottom)
		{NewPoint(5, 11), false},   // Outside (top)
	}

	for _, test := range tests {
		result := box.Contains(test.point)
		if result != test.expected {
			t.Errorf("BoundingBox.Contains(%v) = %v, expected %v", test.point, result, test.expected)
		}
	}
}

func TestBoundingBoxIntersects(t *testing.T) {
	box1 := &BoundingBox{MinLon: 0, MaxLon: 10, MinLat: 0, MaxLat: 10}

	tests := []struct {
		name     string
		box2     *BoundingBox
		expected bool
	}{
		{
			name:     "Overlapping",
			box2:     &BoundingBox{MinLon: 5, MaxLon: 15, MinLat: 5, MaxLat: 15},
			expected: true,
		},
		{
			name:     "Contained",
			box2:     &BoundingBox{MinLon: 2, MaxLon: 8, MinLat: 2, MaxLat: 8},
			expected: true,
		},
		{
			name:     "Touching edge",
			box2:     &BoundingBox{MinLon: 10, MaxLon: 20, MinLat: 0, MaxLat: 10},
			expected: true,
		},
		{
			name:     "Separate",
			box2:     &BoundingBox{MinLon: 20, MaxLon: 30, MinLat: 20, MaxLat: 30},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := box1.Intersects(test.box2)
			if result != test.expected {
				t.Errorf("Expected %v, got %v", test.expected, result)
			}
		})
	}
}

func TestDistance2D(t *testing.T) {
	p1 := NewPoint(0, 0)
	p2 := NewPoint(3, 4)

	distance := Distance2D(p1, p2)
	expected := 5.0 // 3-4-5 triangle

	if math.Abs(distance-expected) > 0.0001 {
		t.Errorf("Distance2D = %f, expected %f", distance, expected)
	}
}

func TestHaversineDistance(t *testing.T) {
	// Test distance between two cities
	// New York: 40.7128° N, 74.0060° W (-74.0060 lon)
	// London: 51.5074° N, 0.1278° W (-0.1278 lon)
	nyc := NewPoint(-74.0060, 40.7128)
	london := NewPoint(-0.1278, 51.5074)

	distance := HaversineDistance(nyc, london)

	// Approximate distance is ~5,570 km = 5,570,000 meters
	// Allow 1% error
	expectedMin := 5500000.0
	expectedMax := 5640000.0

	if distance < expectedMin || distance > expectedMax {
		t.Errorf("HaversineDistance = %f, expected between %f and %f", distance, expectedMin, expectedMax)
	}
}

func TestPointInPolygon(t *testing.T) {
	// Simple square
	square := NewPolygon([][]Point{{
		{Lon: 0, Lat: 0},
		{Lon: 10, Lat: 0},
		{Lon: 10, Lat: 10},
		{Lon: 0, Lat: 10},
		{Lon: 0, Lat: 0},
	}})

	tests := []struct {
		name     string
		point    *Point
		expected bool
	}{
		{"Center", NewPoint(5, 5), true},
		{"On edge", NewPoint(0, 5), true},
		{"On corner", NewPoint(0, 0), true},
		{"Outside", NewPoint(15, 15), false},
		{"Outside negative", NewPoint(-5, -5), false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := PointInPolygon(test.point, square)
			if result != test.expected {
				t.Errorf("PointInPolygon(%v) = %v, expected %v", test.point, result, test.expected)
			}
		})
	}
}

func TestPointInPolygonWithHole(t *testing.T) {
	// Square with square hole
	outer := []Point{
		{Lon: 0, Lat: 0},
		{Lon: 10, Lat: 0},
		{Lon: 10, Lat: 10},
		{Lon: 0, Lat: 10},
		{Lon: 0, Lat: 0},
	}

	hole := []Point{
		{Lon: 3, Lat: 3},
		{Lon: 7, Lat: 3},
		{Lon: 7, Lat: 7},
		{Lon: 3, Lat: 7},
		{Lon: 3, Lat: 3},
	}

	polygon := NewPolygon([][]Point{outer, hole})

	tests := []struct {
		name     string
		point    *Point
		expected bool
	}{
		{"In outer ring", NewPoint(1, 1), true},
		{"In hole", NewPoint(5, 5), false},
		{"Between rings", NewPoint(2, 2), true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := PointInPolygon(test.point, polygon)
			if result != test.expected {
				t.Errorf("PointInPolygon(%v) = %v, expected %v", test.point, result, test.expected)
			}
		})
	}
}

func TestParseGeoJSONPoint(t *testing.T) {
	data := map[string]interface{}{
		"type":        "Point",
		"coordinates": []interface{}{10.5, 20.3},
	}

	point, err := ParseGeoJSONPoint(data)
	if err != nil {
		t.Fatalf("ParseGeoJSONPoint failed: %v", err)
	}

	if point.Lon != 10.5 || point.Lat != 20.3 {
		t.Errorf("Parsed point incorrect: %v", point)
	}
}

func TestParseGeoJSONPointInvalid(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "Missing coordinates",
			data: map[string]interface{}{"type": "Point"},
		},
		{
			name: "Invalid coordinates type",
			data: map[string]interface{}{"coordinates": "invalid"},
		},
		{
			name: "Wrong number of coordinates",
			data: map[string]interface{}{"coordinates": []interface{}{10.5}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseGeoJSONPoint(test.data)
			if err == nil {
				t.Error("Expected error for invalid GeoJSON point")
			}
		})
	}
}

func TestParseGeoJSONPolygon(t *testing.T) {
	data := map[string]interface{}{
		"type": "Polygon",
		"coordinates": []interface{}{
			[]interface{}{
				[]interface{}{0.0, 0.0},
				[]interface{}{10.0, 0.0},
				[]interface{}{10.0, 10.0},
				[]interface{}{0.0, 10.0},
				[]interface{}{0.0, 0.0},
			},
		},
	}

	polygon, err := ParseGeoJSONPolygon(data)
	if err != nil {
		t.Fatalf("ParseGeoJSONPolygon failed: %v", err)
	}

	if len(polygon.Rings) != 1 {
		t.Errorf("Expected 1 ring, got %d", len(polygon.Rings))
	}

	if len(polygon.Rings[0]) != 5 {
		t.Errorf("Expected 5 points, got %d", len(polygon.Rings[0]))
	}
}

func TestParseGeoJSONPolygonInvalidCoordinates(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "Missing coordinates",
			data: map[string]interface{}{"type": "Polygon"},
		},
		{
			name: "Invalid coordinates type",
			data: map[string]interface{}{"coordinates": "invalid"},
		},
		{
			name: "Ring not an array",
			data: map[string]interface{}{"coordinates": []interface{}{"invalid"}},
		},
		{
			name: "Point not an array",
			data: map[string]interface{}{
				"coordinates": []interface{}{
					[]interface{}{"invalid_point"},
				},
			},
		},
		{
			name: "Point with wrong number of coordinates",
			data: map[string]interface{}{
				"coordinates": []interface{}{
					[]interface{}{
						[]interface{}{0.0}, // Only one coordinate
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseGeoJSONPolygon(test.data)
			if err == nil {
				t.Error("Expected error for invalid GeoJSON polygon")
			}
		})
	}
}

func TestToFloat64AllTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
		ok       bool
	}{
		{"float64", float64(10.5), 10.5, true},
		{"float32", float32(10.5), 10.5, true},
		{"int", int(10), 10.0, true},
		{"int64", int64(10), 10.0, true},
		{"int32", int32(10), 10.0, true},
		{"string", "invalid", 0, false},
		{"nil", nil, 0, false},
		{"bool", true, 0, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, ok := toFloat64(test.input)
			if ok != test.ok {
				t.Errorf("toFloat64(%v) ok = %v, expected %v", test.input, ok, test.ok)
			}
			if ok && math.Abs(result-test.expected) > 0.0001 {
				t.Errorf("toFloat64(%v) = %f, expected %f", test.input, result, test.expected)
			}
		})
	}
}

func TestParseGeoJSONPointWithDifferentNumericTypes(t *testing.T) {
	tests := []struct {
		name   string
		coords []interface{}
		expLon float64
		expLat float64
	}{
		{"float64", []interface{}{float64(10.5), float64(20.3)}, 10.5, 20.3},
		{"float32", []interface{}{float32(10.5), float32(20.3)}, 10.5, 20.3},
		{"int", []interface{}{int(10), int(20)}, 10.0, 20.0},
		{"int64", []interface{}{int64(10), int64(20)}, 10.0, 20.0},
		{"int32", []interface{}{int32(10), int32(20)}, 10.0, 20.0},
		{"mixed", []interface{}{int64(10), float32(20.5)}, 10.0, 20.5},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := map[string]interface{}{
				"type":        "Point",
				"coordinates": test.coords,
			}
			point, err := ParseGeoJSONPoint(data)
			if err != nil {
				t.Fatalf("ParseGeoJSONPoint failed: %v", err)
			}
			if math.Abs(point.Lon-test.expLon) > 0.0001 || math.Abs(point.Lat-test.expLat) > 0.0001 {
				t.Errorf("Expected (%f, %f), got (%f, %f)", test.expLon, test.expLat, point.Lon, point.Lat)
			}
		})
	}
}

func TestParseGeoJSONPointInvalidCoordinateType(t *testing.T) {
	data := map[string]interface{}{
		"type":        "Point",
		"coordinates": []interface{}{"invalid", 20.0},
	}
	_, err := ParseGeoJSONPoint(data)
	if err == nil {
		t.Error("Expected error for invalid coordinate type")
	}
}
