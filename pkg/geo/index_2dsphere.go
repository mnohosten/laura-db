package geo

import (
	"fmt"
	"sync"
)

// Index2DSphere represents a 2dsphere spherical coordinate index
// Uses geographic coordinates (longitude, latitude) on Earth's surface
type Index2DSphere struct {
	mu sync.RWMutex

	// Grid configuration (in degrees)
	gridSize float64 // Size of each grid cell in degrees

	// Map from grid cell to document IDs
	grid map[string]map[string]*Point

	// Document ID to point mapping
	docPoints map[string]*Point
}

// NewIndex2DSphere creates a new 2dsphere spherical index
func NewIndex2DSphere(gridSize float64) *Index2DSphere {
	if gridSize <= 0 {
		gridSize = 1.0 // Default 1 degree grid
	}

	return &Index2DSphere{
		gridSize:  gridSize,
		grid:      make(map[string]map[string]*Point),
		docPoints: make(map[string]*Point),
	}
}

// Insert adds a point to the index
func (idx *Index2DSphere) Insert(docID string, point *Point) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Validate coordinates
	if point.Lon < -180 || point.Lon > 180 {
		return fmt.Errorf("longitude must be between -180 and 180")
	}
	if point.Lat < -90 || point.Lat > 90 {
		return fmt.Errorf("latitude must be between -90 and 90")
	}

	// Remove existing point if any
	if existingPoint, exists := idx.docPoints[docID]; exists {
		idx.removeFromGrid(docID, existingPoint)
	}

	// Add to grid
	cellKey := idx.getCellKey(point)
	if idx.grid[cellKey] == nil {
		idx.grid[cellKey] = make(map[string]*Point)
	}
	idx.grid[cellKey][docID] = point

	// Update document mapping
	idx.docPoints[docID] = point

	return nil
}

// Remove removes a document from the index
func (idx *Index2DSphere) Remove(docID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	point, exists := idx.docPoints[docID]
	if !exists {
		return
	}

	idx.removeFromGrid(docID, point)
	delete(idx.docPoints, docID)
}

// removeFromGrid removes a document from the grid (internal, not thread-safe)
func (idx *Index2DSphere) removeFromGrid(docID string, point *Point) {
	cellKey := idx.getCellKey(point)
	if cell, exists := idx.grid[cellKey]; exists {
		delete(cell, docID)
		if len(cell) == 0 {
			delete(idx.grid, cellKey)
		}
	}
}

// getCellKey returns the grid cell key for a point
func (idx *Index2DSphere) getCellKey(point *Point) string {
	x := int(point.Lon / idx.gridSize)
	y := int(point.Lat / idx.gridSize)
	return fmt.Sprintf("%d,%d", x, y)
}

// FindNear finds documents near a point within a maximum distance (in meters)
// Returns results sorted by distance (ascending)
func (idx *Index2DSphere) FindNear(center *Point, maxDistanceMeters float64, limit int) []NearbyResult {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Convert max distance to approximate degrees for bounding box
	// At equator: 1 degree ≈ 111,320 meters
	// This is approximate but sufficient for initial filtering
	maxDistanceDegrees := maxDistanceMeters / 111320.0

	// Calculate search bounding box
	searchBox := &BoundingBox{
		MinLon: center.Lon - maxDistanceDegrees,
		MaxLon: center.Lon + maxDistanceDegrees,
		MinLat: center.Lat - maxDistanceDegrees,
		MaxLat: center.Lat + maxDistanceDegrees,
	}

	// Find all grid cells that intersect with search box
	results := make([]NearbyResult, 0)

	for cellKey, cell := range idx.grid {
		if idx.cellIntersectsBox(cellKey, searchBox) {
			for docID, point := range cell {
				// Calculate actual spherical distance
				distance := HaversineDistance(center, point)
				if distance <= maxDistanceMeters {
					results = append(results, NearbyResult{
						DocID:    docID,
						Point:    point,
						Distance: distance,
					})
				}
			}
		}
	}

	// Sort by distance
	sortByDistance(results)

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

// FindWithin finds all documents within a polygon (spherical)
func (idx *Index2DSphere) FindWithin(polygon *Polygon) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	results := make([]string, 0)
	bounds := polygon.Bounds()

	// Find all grid cells that intersect with polygon bounds
	for cellKey, cell := range idx.grid {
		if idx.cellIntersectsBox(cellKey, bounds) {
			for docID, point := range cell {
				// Use planar point-in-polygon test
				// For small areas, the error is minimal
				if PointInPolygon(point, polygon) {
					results = append(results, docID)
				}
			}
		}
	}

	return results
}

// FindInBox finds all documents within a bounding box
func (idx *Index2DSphere) FindInBox(box *BoundingBox) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	results := make([]string, 0)

	for cellKey, cell := range idx.grid {
		if idx.cellIntersectsBox(cellKey, box) {
			for docID, point := range cell {
				if box.Contains(point) {
					results = append(results, docID)
				}
			}
		}
	}

	return results
}

// cellIntersectsBox checks if a grid cell intersects with a bounding box
func (idx *Index2DSphere) cellIntersectsBox(cellKey string, box *BoundingBox) bool {
	var x, y int
	fmt.Sscanf(cellKey, "%d,%d", &x, &y)

	cellBox := &BoundingBox{
		MinLon: float64(x) * idx.gridSize,
		MaxLon: float64(x+1) * idx.gridSize,
		MinLat: float64(y) * idx.gridSize,
		MaxLat: float64(y+1) * idx.gridSize,
	}

	return cellBox.Intersects(box)
}

// FindInRadius is an alias for FindNear for better API clarity
func (idx *Index2DSphere) FindInRadius(center *Point, radiusMeters float64, limit int) []NearbyResult {
	return idx.FindNear(center, radiusMeters, limit)
}
