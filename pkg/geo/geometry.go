package geo

import (
	"fmt"
	"math"
)

// GeometryType represents the type of geometry
type GeometryType string

const (
	GeometryTypePoint      GeometryType = "Point"
	GeometryTypeLineString GeometryType = "LineString"
	GeometryTypePolygon    GeometryType = "Polygon"
)

// Geometry represents a geographic shape
type Geometry interface {
	Type() GeometryType
	Coordinates() interface{}
	Bounds() *BoundingBox
}

// Point represents a geographic point [longitude, latitude]
// For 2d: [x, y]
// For 2dsphere: [longitude, latitude] in degrees
type Point struct {
	Lon float64 // X coordinate or Longitude
	Lat float64 // Y coordinate or Latitude
}

func NewPoint(lon, lat float64) *Point {
	return &Point{Lon: lon, Lat: lat}
}

func (p *Point) Type() GeometryType {
	return GeometryTypePoint
}

func (p *Point) Coordinates() interface{} {
	return []float64{p.Lon, p.Lat}
}

func (p *Point) Bounds() *BoundingBox {
	return &BoundingBox{
		MinLon: p.Lon,
		MinLat: p.Lat,
		MaxLon: p.Lon,
		MaxLat: p.Lat,
	}
}

// Polygon represents a closed polygon
type Polygon struct {
	// Outer ring (first element) and holes (remaining elements)
	Rings [][]Point
}

func NewPolygon(rings [][]Point) *Polygon {
	return &Polygon{Rings: rings}
}

func (p *Polygon) Type() GeometryType {
	return GeometryTypePolygon
}

func (p *Polygon) Coordinates() interface{} {
	coords := make([][][]float64, len(p.Rings))
	for i, ring := range p.Rings {
		coords[i] = make([][]float64, len(ring))
		for j, point := range ring {
			coords[i][j] = []float64{point.Lon, point.Lat}
		}
	}
	return coords
}

func (p *Polygon) Bounds() *BoundingBox {
	if len(p.Rings) == 0 || len(p.Rings[0]) == 0 {
		return nil
	}

	// Calculate bounds from outer ring
	minLon := p.Rings[0][0].Lon
	maxLon := p.Rings[0][0].Lon
	minLat := p.Rings[0][0].Lat
	maxLat := p.Rings[0][0].Lat

	for _, point := range p.Rings[0] {
		if point.Lon < minLon {
			minLon = point.Lon
		}
		if point.Lon > maxLon {
			maxLon = point.Lon
		}
		if point.Lat < minLat {
			minLat = point.Lat
		}
		if point.Lat > maxLat {
			maxLat = point.Lat
		}
	}

	return &BoundingBox{
		MinLon: minLon,
		MinLat: minLat,
		MaxLon: maxLon,
		MaxLat: maxLat,
	}
}

// BoundingBox represents a rectangular bounding box
type BoundingBox struct {
	MinLon float64
	MinLat float64
	MaxLon float64
	MaxLat float64
}

// Contains checks if a point is within the bounding box
func (bb *BoundingBox) Contains(p *Point) bool {
	return p.Lon >= bb.MinLon && p.Lon <= bb.MaxLon &&
		p.Lat >= bb.MinLat && p.Lat <= bb.MaxLat
}

// Intersects checks if two bounding boxes intersect
func (bb *BoundingBox) Intersects(other *BoundingBox) bool {
	return !(bb.MaxLon < other.MinLon || bb.MinLon > other.MaxLon ||
		bb.MaxLat < other.MinLat || bb.MinLat > other.MaxLat)
}

// Distance calculations

// Distance2D calculates Euclidean distance between two points (planar)
func Distance2D(p1, p2 *Point) float64 {
	dx := p2.Lon - p1.Lon
	dy := p2.Lat - p1.Lat
	return math.Sqrt(dx*dx + dy*dy)
}

// HaversineDistance calculates the great-circle distance between two points
// on a sphere using the Haversine formula
// Returns distance in meters
func HaversineDistance(p1, p2 *Point) float64 {
	const earthRadius = 6371000.0 // Earth's radius in meters

	// Convert to radians
	lat1 := toRadians(p1.Lat)
	lat2 := toRadians(p2.Lat)
	deltaLat := toRadians(p2.Lat - p1.Lat)
	deltaLon := toRadians(p2.Lon - p1.Lon)

	// Haversine formula
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// toRadians converts degrees to radians
func toRadians(degrees float64) float64 {
	return degrees * math.Pi / 180.0
}

// toDegrees converts radians to degrees
func toDegrees(radians float64) float64 {
	return radians * 180.0 / math.Pi
}

// PointInPolygon checks if a point is inside a polygon using ray casting algorithm
func PointInPolygon(point *Point, polygon *Polygon) bool {
	if len(polygon.Rings) == 0 {
		return false
	}

	// Check if point is in outer ring
	if !pointInRing(point, polygon.Rings[0]) {
		return false
	}

	// Check if point is in any hole (if so, it's not in the polygon)
	for i := 1; i < len(polygon.Rings); i++ {
		if pointInRing(point, polygon.Rings[i]) {
			return false
		}
	}

	return true
}

// pointInRing uses ray casting algorithm to determine if point is in ring
func pointInRing(point *Point, ring []Point) bool {
	if len(ring) < 3 {
		return false
	}

	inside := false
	j := len(ring) - 1

	for i := 0; i < len(ring); i++ {
		xi, yi := ring[i].Lon, ring[i].Lat
		xj, yj := ring[j].Lon, ring[j].Lat

		intersect := ((yi > point.Lat) != (yj > point.Lat)) &&
			(point.Lon < (xj-xi)*(point.Lat-yi)/(yj-yi)+xi)

		if intersect {
			inside = !inside
		}

		j = i
	}

	return inside
}

// ParseGeoJSONPoint parses a GeoJSON-like point from a map
func ParseGeoJSONPoint(data map[string]interface{}) (*Point, error) {
	coordsRaw, ok := data["coordinates"]
	if !ok {
		return nil, fmt.Errorf("missing coordinates field")
	}

	coords, ok := coordsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("coordinates must be an array")
	}

	if len(coords) != 2 {
		return nil, fmt.Errorf("point coordinates must have 2 elements")
	}

	lon, ok := toFloat64(coords[0])
	if !ok {
		return nil, fmt.Errorf("invalid longitude")
	}

	lat, ok := toFloat64(coords[1])
	if !ok {
		return nil, fmt.Errorf("invalid latitude")
	}

	return NewPoint(lon, lat), nil
}

// ParseGeoJSONPolygon parses a GeoJSON-like polygon from a map
func ParseGeoJSONPolygon(data map[string]interface{}) (*Polygon, error) {
	coordsRaw, ok := data["coordinates"]
	if !ok {
		return nil, fmt.Errorf("missing coordinates field")
	}

	// coordinates is array of rings, each ring is array of points
	rings, ok := coordsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("coordinates must be an array")
	}

	polygonRings := make([][]Point, len(rings))

	for i, ringRaw := range rings {
		ring, ok := ringRaw.([]interface{})
		if !ok {
			return nil, fmt.Errorf("ring must be an array")
		}

		points := make([]Point, len(ring))
		for j, pointRaw := range ring {
			pointCoords, ok := pointRaw.([]interface{})
			if !ok {
				return nil, fmt.Errorf("point must be an array")
			}

			if len(pointCoords) != 2 {
				return nil, fmt.Errorf("point must have 2 coordinates")
			}

			lon, ok := toFloat64(pointCoords[0])
			if !ok {
				return nil, fmt.Errorf("invalid longitude")
			}

			lat, ok := toFloat64(pointCoords[1])
			if !ok {
				return nil, fmt.Errorf("invalid latitude")
			}

			points[j] = Point{Lon: lon, Lat: lat}
		}

		polygonRings[i] = points
	}

	return NewPolygon(polygonRings), nil
}

// toFloat64 converts various numeric types to float64
func toFloat64(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	default:
		return 0, false
	}
}
