package index

import (
	"fmt"
	"sync"

	"github.com/mnohosten/laura-db/pkg/geo"
)

// GeoIndex represents a geospatial index (2d or 2dsphere)
type GeoIndex struct {
	name       string
	fieldPath  string // Field containing geospatial data
	indexType  IndexType
	index2d    *geo.Index2D
	index2ds   *geo.Index2DSphere
	stats      *IndexStats
	mu         sync.RWMutex
}

// NewGeoIndex creates a new geospatial index
func NewGeoIndex(name, fieldPath string, indexType IndexType) *GeoIndex {
	gi := &GeoIndex{
		name:      name,
		fieldPath: fieldPath,
		indexType: indexType,
		stats:     NewIndexStats(),
	}

	switch indexType {
	case IndexType2D:
		gi.index2d = geo.NewIndex2D(1.0) // Default grid size
	case IndexType2DSphere:
		gi.index2ds = geo.NewIndex2DSphere(1.0) // Default 1 degree grid
	default:
		panic(fmt.Sprintf("invalid geospatial index type: %v", indexType))
	}

	return gi
}

// Index adds a point to the geospatial index
func (gi *GeoIndex) Index(docID string, point *geo.Point) error {
	gi.mu.Lock()
	defer gi.mu.Unlock()

	var err error
	switch gi.indexType {
	case IndexType2D:
		gi.index2d.Insert(docID, point)
	case IndexType2DSphere:
		err = gi.index2ds.Insert(docID, point)
	}

	if err == nil {
		gi.stats.Update()
	}

	return err
}

// Remove removes a document from the geospatial index
func (gi *GeoIndex) Remove(docID string) {
	gi.mu.Lock()
	defer gi.mu.Unlock()

	switch gi.indexType {
	case IndexType2D:
		gi.index2d.Remove(docID)
	case IndexType2DSphere:
		gi.index2ds.Remove(docID)
	}

	gi.stats.Update()
}

// Near finds documents near a point within a maximum distance
// For 2d: distance is in coordinate units
// For 2dsphere: distance is in meters
func (gi *GeoIndex) Near(center *geo.Point, maxDistance float64, limit int) []geo.NearbyResult {
	gi.mu.RLock()
	defer gi.mu.RUnlock()

	switch gi.indexType {
	case IndexType2D:
		return gi.index2d.FindNear(center, maxDistance, limit)
	case IndexType2DSphere:
		return gi.index2ds.FindNear(center, maxDistance, limit)
	}

	return nil
}

// Within finds documents within a polygon
func (gi *GeoIndex) Within(polygon *geo.Polygon) []string {
	gi.mu.RLock()
	defer gi.mu.RUnlock()

	switch gi.indexType {
	case IndexType2D:
		return gi.index2d.FindWithin(polygon)
	case IndexType2DSphere:
		return gi.index2ds.FindWithin(polygon)
	}

	return nil
}

// InBox finds documents within a bounding box
func (gi *GeoIndex) InBox(box *geo.BoundingBox) []string {
	gi.mu.RLock()
	defer gi.mu.RUnlock()

	switch gi.indexType {
	case IndexType2D:
		return gi.index2d.FindInBox(box)
	case IndexType2DSphere:
		return gi.index2ds.FindInBox(box)
	}

	return nil
}

// Name returns the index name
func (gi *GeoIndex) Name() string {
	return gi.name
}

// FieldPath returns the field path
func (gi *GeoIndex) FieldPath() string {
	return gi.fieldPath
}

// FieldPaths returns the field paths (single field for geo indexes)
func (gi *GeoIndex) FieldPaths() []string {
	return []string{gi.fieldPath}
}

// Type returns the index type
func (gi *GeoIndex) Type() IndexType {
	return gi.indexType
}

// Stats returns index statistics
func (gi *GeoIndex) Stats() map[string]interface{} {
	gi.mu.RLock()
	defer gi.mu.RUnlock()

	typeStr := "2d"
	if gi.indexType == IndexType2DSphere {
		typeStr = "2dsphere"
	}

	stats := map[string]interface{}{
		"name":       gi.name,
		"field_path": gi.fieldPath,
		"type":       typeStr,
	}

	// Add index statistics
	for k, v := range gi.stats.ToMap() {
		stats[k] = v
	}

	return stats
}

// Analyze recalculates index statistics
func (gi *GeoIndex) Analyze() {
	gi.mu.Lock()
	defer gi.mu.Unlock()

	// For geospatial indexes, we mainly track document count
	// More sophisticated stats could be added later
	gi.stats.Update()
}
