package geo

import (
	"fmt"
	"sync"
)

// Index2D represents a 2d planar coordinate index
// Uses a grid-based spatial indexing approach for efficient range queries
type Index2D struct {
	mu sync.RWMutex

	// Grid configuration
	gridSize float64 // Size of each grid cell

	// Map from grid cell to document IDs
	// Grid cell key: "x,y" where x,y are grid coordinates
	grid map[string]map[string]*Point

	// Document ID to point mapping
	docPoints map[string]*Point

	// Index bounds
	bounds *BoundingBox
}

// NewIndex2D creates a new 2d planar index
func NewIndex2D(gridSize float64) *Index2D {
	if gridSize <= 0 {
		gridSize = 1.0 // Default grid size
	}

	return &Index2D{
		gridSize:  gridSize,
		grid:      make(map[string]map[string]*Point),
		docPoints: make(map[string]*Point),
		bounds: &BoundingBox{
			MinLon: 0,
			MinLat: 0,
			MaxLon: 0,
			MaxLat: 0,
		},
	}
}

// Insert adds a point to the index
func (idx *Index2D) Insert(docID string, point *Point) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

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

	// Update bounds
	idx.updateBounds(point)
}

// Remove removes a document from the index
func (idx *Index2D) Remove(docID string) {
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
func (idx *Index2D) removeFromGrid(docID string, point *Point) {
	cellKey := idx.getCellKey(point)
	if cell, exists := idx.grid[cellKey]; exists {
		delete(cell, docID)
		if len(cell) == 0 {
			delete(idx.grid, cellKey)
		}
	}
}

// getCellKey returns the grid cell key for a point
func (idx *Index2D) getCellKey(point *Point) string {
	x := int(point.Lon / idx.gridSize)
	y := int(point.Lat / idx.gridSize)
	return fmt.Sprintf("%d,%d", x, y)
}

// updateBounds updates the index bounds
func (idx *Index2D) updateBounds(point *Point) {
	if len(idx.docPoints) == 1 {
		// First point
		idx.bounds.MinLon = point.Lon
		idx.bounds.MaxLon = point.Lon
		idx.bounds.MinLat = point.Lat
		idx.bounds.MaxLat = point.Lat
		return
	}

	if point.Lon < idx.bounds.MinLon {
		idx.bounds.MinLon = point.Lon
	}
	if point.Lon > idx.bounds.MaxLon {
		idx.bounds.MaxLon = point.Lon
	}
	if point.Lat < idx.bounds.MinLat {
		idx.bounds.MinLat = point.Lat
	}
	if point.Lat > idx.bounds.MaxLat {
		idx.bounds.MaxLat = point.Lat
	}
}

// NearbyResult represents a document with its distance from query point
type NearbyResult struct {
	DocID    string
	Point    *Point
	Distance float64
}

// FindNear finds documents near a point within a maximum distance
// Returns results sorted by distance (ascending)
func (idx *Index2D) FindNear(center *Point, maxDistance float64, limit int) []NearbyResult {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Calculate search bounding box
	searchBox := &BoundingBox{
		MinLon: center.Lon - maxDistance,
		MaxLon: center.Lon + maxDistance,
		MinLat: center.Lat - maxDistance,
		MaxLat: center.Lat + maxDistance,
	}

	// Find all grid cells that intersect with search box
	results := make([]NearbyResult, 0)

	for cellKey, cell := range idx.grid {
		// Check if cell might contain relevant points
		if idx.cellIntersectsBox(cellKey, searchBox) {
			for docID, point := range cell {
				distance := Distance2D(center, point)
				if distance <= maxDistance {
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

// FindWithin finds all documents within a polygon
func (idx *Index2D) FindWithin(polygon *Polygon) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	results := make([]string, 0)
	bounds := polygon.Bounds()

	// Find all grid cells that intersect with polygon bounds
	for cellKey, cell := range idx.grid {
		if idx.cellIntersectsBox(cellKey, bounds) {
			for docID, point := range cell {
				if PointInPolygon(point, polygon) {
					results = append(results, docID)
				}
			}
		}
	}

	return results
}

// FindInBox finds all documents within a bounding box
func (idx *Index2D) FindInBox(box *BoundingBox) []string {
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
func (idx *Index2D) cellIntersectsBox(cellKey string, box *BoundingBox) bool {
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

// sortByDistance sorts results by distance using insertion sort (efficient for small lists)
func sortByDistance(results []NearbyResult) {
	for i := 1; i < len(results); i++ {
		key := results[i]
		j := i - 1
		for j >= 0 && results[j].Distance > key.Distance {
			results[j+1] = results[j]
			j--
		}
		results[j+1] = key
	}
}
