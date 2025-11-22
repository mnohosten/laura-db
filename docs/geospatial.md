# Geospatial Queries

## Overview

Laura-DB provides comprehensive geospatial query capabilities through 2d and 2dsphere indexes and geospatial query methods. The implementation supports proximity queries, polygon containment, and bounding box intersections.

## Features

- **2d Indexes**: Planar coordinate indexing for flat maps and game worlds
- **2dsphere Indexes**: Spherical coordinate indexing for Earth geographic data
- **Haversine Distance**: Accurate spherical distance calculations (Earth surface)
- **Proximity Queries**: Find documents near a point within a distance
- **Polygon Queries**: Find documents within a polygon boundary
- **Bounding Box Queries**: Find documents intersecting a rectangular area
- **Automatic Index Maintenance**: Indexes updated on insert, update, and delete

## Index Types

### 2d Index (Planar)

For flat 2D coordinate systems (game maps, floor plans, CAD drawings).

**When to use:**
- Coordinates represent positions on a flat plane
- Working with X, Y coordinates
- Game development, simulations
- Distance calculated using Euclidean formula

**Creating a 2d index:**

```go
coll := db.Collection("locations")

// Create 2d index on the "position" field
err := coll.Create2DIndex("position")
```

### 2dsphere Index (Spherical)

For Earth geographic coordinates (longitude, latitude).

**When to use:**
- Working with real-world geographic data
- Longitude and latitude coordinates
- Maps, location services, GPS data
- Distance calculated using Haversine formula (accounts for Earth's curvature)

**Creating a 2dsphere index:**

```go
coll := db.Collection("places")

// Create 2dsphere index on the "coords" field
err := coll.Create2DSphereIndex("coords")
```

**Coordinate validation:**
- Longitude must be between -180 and 180
- Latitude must be between -90 and 90

## Document Format

Geospatial data should be stored in GeoJSON-like format:

### Point

```go
{
    "name": "San Francisco",
    "location": {
        "type": "Point",
        "coordinates": [-122.4194, 37.7749]  // [longitude, latitude]
    }
}
```

**Important:** Coordinates are always `[longitude, latitude]` (not lat/lon!)

### Polygon

```go
{
    "name": "California Region",
    "boundary": {
        "type": "Polygon",
        "coordinates": [
            [  // Outer ring
                [-125.0, 32.0],
                [-114.0, 32.0],
                [-114.0, 42.0],
                [-125.0, 42.0],
                [-125.0, 32.0]  // Closing point (must match first point)
            ]
            // Additional rings here would be holes
        ]
    }
}
```

**Polygon rules:**
- First ring is the outer boundary
- Additional rings are holes (interior exclusions)
- Rings must be closed (first point == last point)
- Minimum 3 unique points (4 total including closing point)

## Query Operations

### Near Query ($near)

Find documents within a specified distance from a point, sorted by distance.

**2d Example (Euclidean distance):**

```go
coll := db.Collection("gameObjects")
coll.Create2DIndex("position")

// Find objects near (50, 50) within distance 10
center := geo.NewPoint(50, 50)
results, err := coll.Near("position", center, 10.0, 100, nil)

for _, doc := range results {
    name, _ := doc.Get("name")
    distance, _ := doc.Get("_distance")  // Automatically included
    fmt.Printf("%s is %f units away\n", name, distance)
}
```

**2dsphere Example (Haversine distance in meters):**

```go
coll := db.Collection("stores")
coll.Create2DSphereIndex("location")

// Find stores near user's location within 5 km
userLocation := geo.NewPoint(-122.4194, 37.7749)  // San Francisco
results, err := coll.Near("location", userLocation, 5000, 10, nil)  // 5000 meters

for _, doc := range results {
    name, _ := doc.Get("name")
    distance, _ := doc.Get("_distance")  // Distance in meters
    fmt.Printf("%s is %.2f km away\n", name, distance.(float64)/1000)
}
```

**Parameters:**
- `fieldPath`: The field containing geospatial data
- `center`: The center point (`geo.Point`)
- `maxDistance`: Maximum distance from center
  - For 2d: distance in coordinate units
  - For 2dsphere: distance in meters
- `limit`: Maximum number of results to return
- `options`: Query options (projection, etc.)

**Results:**
- Automatically sorted by distance (closest first)
- Include `_distance` field with calculated distance

### GeoWithin Query ($geoWithin)

Find documents whose location is within a specified polygon.

```go
coll := db.Collection("properties")
coll.Create2DSphereIndex("location")

// Define search area (polygon around downtown)
searchArea := geo.NewPolygon([][]geo.Point{{
    {Lon: -122.45, Lat: 37.75},
    {Lon: -122.40, Lat: 37.75},
    {Lon: -122.40, Lat: 37.80},
    {Lon: -122.45, Lat: 37.80},
    {Lon: -122.45, Lat: 37.75},  // Closing point
}})

// Find all properties within the polygon
results, err := coll.GeoWithin("location", searchArea, nil)
```

**Use cases:**
- Finding points within a city boundary
- Delivery zones
- Service areas
- Game world regions

### GeoIntersects Query ($geoIntersects)

Find documents whose location intersects with a bounding box.

```go
coll := db.Collection("sensors")
coll.Create2DIndex("position")

// Define bounding box
box := &geo.BoundingBox{
    MinLon: 10,
    MaxLon: 20,
    MinLat: 30,
    MaxLat: 40,
}

// Find all sensors in the bounding box
results, err := coll.GeoIntersects("position", box, nil)
```

**Use cases:**
- Map viewport queries
- Grid-based searches
- Rectangular selection areas

## Complete Examples

### Store Locator Application

```go
// Setup
db, _ := database.Open(database.DefaultConfig("./storedb"))
stores := db.Collection("stores")

// Create geospatial index
stores.Create2DSphereIndex("location")

// Insert stores
stores.InsertOne(map[string]interface{}{
    "name":    "Downtown Store",
    "address": "123 Main St, San Francisco, CA",
    "location": map[string]interface{}{
        "type":        "Point",
        "coordinates": []interface{}{-122.4194, 37.7749},
    },
    "hours": "9 AM - 9 PM",
})

stores.InsertOne(map[string]interface{}{
    "name":    "Marina Store",
    "address": "456 Beach St, San Francisco, CA",
    "location": map[string]interface{}{
        "type":        "Point",
        "coordinates": []interface{}{-122.4376, 37.8044},
    },
    "hours": "10 AM - 8 PM",
})

// User's location
userLoc := geo.NewPoint(-122.4194, 37.7749)

// Find nearest stores within 5 km
nearby, _ := stores.Near("location", userLoc, 5000, 5, &database.QueryOptions{
    Projection: map[string]bool{
        "name":    true,
        "address": true,
        "hours":   true,
    },
})

fmt.Println("Stores near you:")
for i, store := range nearby {
    name, _ := store.Get("name")
    address, _ := store.Get("address")
    distance, _ := store.Get("_distance")

    fmt.Printf("%d. %s\n", i+1, name)
    fmt.Printf("   %s\n", address)
    fmt.Printf("   %.2f km away\n\n", distance.(float64)/1000)
}
```

### Delivery Zone Verification

```go
// Define delivery zone
deliveryZone := geo.NewPolygon([][]geo.Point{{
    {Lon: -122.50, Lat: 37.70},
    {Lon: -122.35, Lat: 37.70},
    {Lon: -122.35, Lat: 37.85},
    {Lon: -122.50, Lat: 37.85},
    {Lon: -122.50, Lat: 37.70},
}})

// Check if customer location is in delivery zone
customerLoc := map[string]interface{}{
    "type":        "Point",
    "coordinates": []interface{}{-122.4194, 37.7749},
}

customer := db.Collection("temp")
customer.Create2DSphereIndex("location")
customer.InsertOne(map[string]interface{}{
    "id":       "customer1",
    "location": customerLoc,
})

results, _ := customer.GeoWithin("location", deliveryZone, nil)

if len(results) > 0 {
    fmt.Println("Address is within delivery zone!")
} else {
    fmt.Println("Sorry, we don't deliver to this area.")
}
```

### Game World Position Tracking

```go
// Setup game world (2d planar coordinates)
db, _ := database.Open(database.DefaultConfig("./gamedb"))
players := db.Collection("players")

players.Create2DIndex("position")

// Insert players
players.InsertOne(map[string]interface{}{
    "username": "warrior123",
    "position": map[string]interface{}{
        "type":        "Point",
        "coordinates": []interface{}{50.0, 75.0},
    },
    "level": 10,
})

players.InsertOne(map[string]interface{}{
    "username": "mage456",
    "position": map[string]interface{}{
        "type":        "Point",
        "coordinates": []interface{}{52.0, 73.0},
    },
    "level": 12,
})

// Find players near a specific location
location := geo.NewPoint(50, 75)
nearbyPlayers, _ := players.Near("position", location, 10.0, 50, nil)

fmt.Printf("Found %d players nearby:\n", len(nearbyPlayers))
for _, player := range nearbyPlayers {
    username, _ := player.Get("username")
    distance, _ := player.Get("_distance")
    fmt.Printf("- %s (%.1f units away)\n", username, distance)
}
```

## Index Maintenance

Geospatial indexes are automatically maintained during CRUD operations:

### Insert

```go
// Document is automatically indexed
coll.InsertOne(map[string]interface{}{
    "name": "New Location",
    "coords": map[string]interface{}{
        "type":        "Point",
        "coordinates": []interface{}{-122.4, 37.8},
    },
})
```

### Update

```go
// Old index entry removed, new one added
coll.UpdateOne(
    map[string]interface{}{"name": "Store A"},
    map[string]interface{}{
        "$set": map[string]interface{}{
            "coords": map[string]interface{}{
                "type":        "Point",
                "coordinates": []interface{}{-122.5, 37.9},
            },
        },
    },
)
```

### Delete

```go
// Document removed from geospatial index
coll.DeleteOne(map[string]interface{}{"name": "Store A"})
```

## Performance Characteristics

### Time Complexity

| Operation | 2d Index | 2dsphere Index |
|-----------|----------|----------------|
| Index Creation | O(N × log N) | O(N × log N) |
| Insert | O(1) | O(1) |
| Near Query | O(C + K × log K) | O(C + K × log K) |
| GeoWithin | O(C + K) | O(C + K) |
| Update | O(1) | O(1) |
| Delete | O(1) | O(1) |

Where:
- N = number of documents
- C = number of grid cells to check
- K = number of results

### Benchmark Results

Based on 10,000 documents:

```
BenchmarkNearQuery2D                  58      17.1ms/op
BenchmarkNearQuery2DSphere           192       6.2ms/op (1000 docs)
BenchmarkGeoWithinQuery              244       4.9ms/op
BenchmarkGeoIntersects               250       4.8ms/op
BenchmarkGeoIndexInsert          1578082       760ns/op
BenchmarkHaversineDistance      94017668       12.7ns/op
```

**Key insights:**
- Near queries on 2dsphere with 1000 docs: ~6.2ms
- Haversine distance calculation: ~13ns (very fast)
- Index insertion: ~760ns per document

## Best Practices

### 1. Choose the Right Index Type

```go
// Good: Use 2dsphere for real-world coordinates
coll.Create2DSphereIndex("location")  // For GPS, maps, Earth data

// Good: Use 2d for planar coordinates
coll.Create2DIndex("position")  // For games, simulations, diagrams

// Bad: Using 2d for longitude/latitude
coll.Create2DIndex("coords")  // Wrong! Use 2dsphere instead
```

### 2. Validate Coordinates

```go
// Good: Validate before insertion
lon, lat := -122.4194, 37.7749

if lon < -180 || lon > 180 {
    return errors.New("invalid longitude")
}
if lat < -90 || lat > 90 {
    return errors.New("invalid latitude")
}

// Then insert
coll.InsertOne(map[string]interface{}{
    "coords": map[string]interface{}{
        "type":        "Point",
        "coordinates": []interface{}{lon, lat},
    },
})
```

### 3. Use Appropriate Distance Units

```go
// Good: Clearly specify units
const fiveKilometers = 5000  // meters for 2dsphere
results, _ := coll.Near("location", center, fiveKilometers, 10, nil)

// Bad: Unclear units
results, _ := coll.Near("location", center, 5, 10, nil)  // 5 what?
```

### 4. Limit Result Sets

```go
// Good: Use limit to prevent returning too many results
coll.Near("location", center, 10000, 20, nil)  // Max 20 results

// Poor: Unbounded query
coll.Near("location", center, 10000, 0, nil)  // Could return thousands
```

### 5. Use Projection for Large Documents

```go
// Good: Project only needed fields
coll.Near("location", center, 5000, 10, &database.QueryOptions{
    Projection: map[string]bool{
        "name":    true,
        "address": true,
    },
})

// Poor: Return entire documents (might be large)
coll.Near("location", center, 5000, 10, nil)
```

## Common Patterns

### Find Closest Item

```go
// Find the single closest store
results, _ := coll.Near("location", userLocation, 100000, 1, nil)

if len(results) > 0 {
    closestStore := results[0]
    name, _ := closestStore.Get("name")
    distance, _ := closestStore.Get("_distance")
    fmt.Printf("Closest store: %s (%.1f km)\n", name, distance.(float64)/1000)
}
```

### Count Items in Area

```go
// Count how many items are in a specific region
results, _ := coll.GeoWithin("location", polygon, nil)
count := len(results)
fmt.Printf("Found %d items in the region\n", count)
```

### Paginated Proximity Results

```go
// Page 1
page1, _ := coll.Near("location", center, 10000, 20, &database.QueryOptions{
    Limit: 20,
})

// Page 2 (Note: Need to implement skip logic or use cursor)
// LauraDB doesn't support skip on Near queries yet
```

## Limitations

### Current Limitations

1. **Geometry Types**: Only Point and Polygon supported (no LineString, MultiPoint, etc.)
2. **Query Types**: No $nearSphere operator (use Near with 2dsphere instead)
3. **Index Combinations**: Cannot combine multiple geospatial conditions in one query
4. **Coordinate Reference Systems**: Only WGS84 supported for 2dsphere
5. **Grid Size**: Fixed grid size (1.0 degrees for 2dsphere, 1.0 units for 2d)

### Future Enhancements

- Additional geometry types (LineString, MultiPolygon)
- $nearSphere operator
- Configurable grid sizes for optimization
- Multiple coordinate reference systems
- Geospatial aggregation stages
- Distance matrix calculations

## Troubleshooting

### No Results Found

**Check if index exists:**
```go
indexes := coll.ListIndexes()
for _, idx := range indexes {
    if idx["type"] == "2d" || idx["type"] == "2dsphere" {
        fmt.Printf("Geo index: %s (type: %s)\n", idx["name"], idx["type"])
    }
}
```

**Verify coordinates are valid:**
```go
// Make sure coordinates are [lon, lat], not [lat, lon]
// Longitude: -180 to 180
// Latitude: -90 to 90
```

### Wrong Index Type Error

```go
// Error: no geospatial index found for field "location"
// Solution: Create the index first
coll.Create2DSphereIndex("location")
```

### Inaccurate Distances

**2dsphere:**
- Ensure you're using Haversine distance (automatic with 2dsphere)
- Verify coordinates are in correct order: [longitude, latitude]

**2d:**
- Remember that 2d uses Euclidean distance (straight line)
- For Earth coordinates, use 2dsphere instead

## Comparison with MongoDB

### Similarities

```javascript
// MongoDB
db.collection.createIndex({ location: "2dsphere" })
db.collection.find({
    location: {
        $near: {
            $geometry: { type: "Point", coordinates: [-122.4, 37.8] },
            $maxDistance: 5000
        }
    }
})

// Laura-DB (equivalent)
coll.Create2DSphereIndex("location")
center := geo.NewPoint(-122.4, 37.8)
results, _ := coll.Near("location", center, 5000, 100, nil)
```

### Differences

| Feature | MongoDB | Laura-DB |
|---------|---------|----------|
| API Style | Query operators | Direct method calls |
| Geometry Types | 10+ types | Point, Polygon |
| CRS Support | Multiple | WGS84 only |
| Grid Tuning | Configurable | Fixed |
| $nearSphere | Yes | Use Near with 2dsphere |

## Summary

Laura-DB's geospatial features provide:
- Easy-to-use API for location-based queries
- Both planar (2d) and spherical (2dsphere) coordinate systems
- Accurate distance calculations (Haversine for Earth coordinates)
- Efficient grid-based spatial indexing
- Automatic index maintenance
- Flexible query options (projection, limits)

Perfect for applications needing location search, proximity detection, and geographic data management!
