package database

import (
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/mnohosten/laura-db/pkg/geo"
)

func BenchmarkCreate2DIndex(b *testing.B) {
	dir := "./bench_create_2d"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("locations")

	// Insert 1000 documents
	for i := 0; i < 1000; i++ {
		coll.InsertOne(map[string]interface{}{
			"name": fmt.Sprintf("Location %d", i),
			"loc": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{float64(i % 100), float64(i / 100)},
			},
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.Create2DIndex("loc")
		if i < b.N-1 {
			coll.DropIndex("loc_2d")
		}
	}
}

func BenchmarkNearQuery2D(b *testing.B) {
	dir := "./bench_near_2d"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("points")

	// Insert 10,000 random points
	for i := 0; i < 10000; i++ {
		coll.InsertOne(map[string]interface{}{
			"id": i,
			"loc": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{rand.Float64() * 100, rand.Float64() * 100},
			},
		})
	}

	coll.Create2DIndex("loc")

	center := geo.NewPoint(50, 50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.Near("loc", center, 10.0, 100, nil)
	}
}

func BenchmarkNearQuery2DSphere(b *testing.B) {
	dir := "./bench_near_2dsphere"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("cities")

	// Insert 1000 random cities (realistic longitude/latitude ranges)
	for i := 0; i < 1000; i++ {
		coll.InsertOne(map[string]interface{}{
			"id": i,
			"coords": map[string]interface{}{
				"type": "Point",
				"coordinates": []interface{}{
					-180 + rand.Float64()*360, // Longitude: -180 to 180
					-90 + rand.Float64()*180,  // Latitude: -90 to 90
				},
			},
		})
	}

	coll.Create2DSphereIndex("coords")

	// Search near equator
	center := geo.NewPoint(0, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Within 1000 km
		coll.Near("coords", center, 1000000, 100, nil)
	}
}

func BenchmarkGeoWithinQuery(b *testing.B) {
	dir := "./bench_within"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("locations")

	// Insert 5000 random points
	for i := 0; i < 5000; i++ {
		coll.InsertOne(map[string]interface{}{
			"id": i,
			"loc": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{rand.Float64() * 100, rand.Float64() * 100},
			},
		})
	}

	coll.Create2DIndex("loc")

	// Create polygon (square from 40,40 to 60,60)
	polygon := geo.NewPolygon([][]geo.Point{{
		{Lon: 40, Lat: 40},
		{Lon: 60, Lat: 40},
		{Lon: 60, Lat: 60},
		{Lon: 40, Lat: 60},
		{Lon: 40, Lat: 40},
	}})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.GeoWithin("loc", polygon, nil)
	}
}

func BenchmarkGeoIntersects(b *testing.B) {
	dir := "./bench_intersects"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("points")

	// Insert 5000 random points
	for i := 0; i < 5000; i++ {
		coll.InsertOne(map[string]interface{}{
			"id": i,
			"pos": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{rand.Float64() * 100, rand.Float64() * 100},
			},
		})
	}

	coll.Create2DIndex("pos")

	// Create bounding box
	box := &geo.BoundingBox{
		MinLon: 40,
		MaxLon: 60,
		MinLat: 40,
		MaxLat: 60,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.GeoIntersects("pos", box, nil)
	}
}

func BenchmarkGeoIndexInsert(b *testing.B) {
	dir := "./bench_geo_insert"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("locations")
	coll.Create2DIndex("loc")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.InsertOne(map[string]interface{}{
			"id": i,
			"loc": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{rand.Float64() * 100, rand.Float64() * 100},
			},
		})
	}
}

func BenchmarkGeoIndexUpdate(b *testing.B) {
	dir := "./bench_geo_update"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("places")
	coll.Create2DIndex("loc")

	// Insert initial documents
	for i := 0; i < 1000; i++ {
		coll.InsertOne(map[string]interface{}{
			"name": fmt.Sprintf("place%d", i),
			"loc": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{rand.Float64() * 100, rand.Float64() * 100},
			},
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := i % 1000
		coll.UpdateOne(
			map[string]interface{}{"name": fmt.Sprintf("place%d", idx)},
			map[string]interface{}{
				"$set": map[string]interface{}{
					"loc": map[string]interface{}{
						"type":        "Point",
						"coordinates": []interface{}{rand.Float64() * 100, rand.Float64() * 100},
					},
				},
			},
		)
	}
}

func BenchmarkGeoIndexDelete(b *testing.B) {
	dir := "./bench_geo_delete"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	coll := db.Collection("data")
	coll.Create2DIndex("loc")

	// Insert many documents
	for i := 0; i < b.N+1000; i++ {
		coll.InsertOne(map[string]interface{}{
			"name": fmt.Sprintf("doc%d", i),
			"loc": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{rand.Float64() * 100, rand.Float64() * 100},
			},
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coll.DeleteOne(map[string]interface{}{"name": fmt.Sprintf("doc%d", i)})
	}
}

func BenchmarkHaversineDistance(b *testing.B) {
	p1 := geo.NewPoint(-122.4194, 37.7749) // San Francisco
	p2 := geo.NewPoint(-74.0060, 40.7128)  // New York

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		geo.HaversineDistance(p1, p2)
	}
}

func BenchmarkPointInPolygon(b *testing.B) {
	// Create polygon
	ring := make([]geo.Point, 100)
	for i := 0; i < 100; i++ {
		ring[i] = geo.Point{
			Lon: 50 + 10*float64(i)/100,
			Lat: 50 + 10*float64(i)/100,
		}
	}
	polygon := geo.NewPolygon([][]geo.Point{ring})

	point := geo.NewPoint(50, 50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		geo.PointInPolygon(point, polygon)
	}
}

func BenchmarkGeoQueryComparison(b *testing.B) {
	dir := "./bench_geo_comparison"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	// Collection with geo index
	collWithIndex := db.Collection("with_index")
	collWithIndex.Create2DIndex("loc")

	// Collection without geo index
	collNoIndex := db.Collection("no_index")

	// Insert same data in both
	for i := 0; i < 1000; i++ {
		doc := map[string]interface{}{
			"id": i,
			"loc": map[string]interface{}{
				"type":        "Point",
				"coordinates": []interface{}{rand.Float64() * 100, rand.Float64() * 100},
			},
		}
		collWithIndex.InsertOne(doc)
		collNoIndex.InsertOne(doc)
	}

	center := geo.NewPoint(50, 50)
	box := &geo.BoundingBox{MinLon: 40, MaxLon: 60, MinLat: 40, MaxLat: 60}

	b.Run("With geo index", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			collWithIndex.Near("loc", center, 10.0, 100, nil)
		}
	})

	b.Run("Without geo index (collection scan)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Manual bounding box check
			collNoIndex.GeoIntersects("loc", box, nil)
		}
	})
}
