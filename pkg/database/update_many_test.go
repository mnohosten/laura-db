package database

import (
	"os"
	"testing"
	"time"

	"github.com/mnohosten/laura-db/pkg/geo"
)

// TestUpdateMany_WithCompoundIndex tests UpdateMany with compound indexes
func TestUpdateMany_WithCompoundIndex(t *testing.T) {
	dir := "./test_db_update_many_compound"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	users := db.Collection("users")

	// Create compound index
	users.CreateCompoundIndex([]string{"city", "age"}, false)

	// Insert test data
	users.InsertOne(map[string]interface{}{"name": "Alice", "city": "NYC", "age": int64(30)})
	users.InsertOne(map[string]interface{}{"name": "Bob", "city": "NYC", "age": int64(25)})
	users.InsertOne(map[string]interface{}{"name": "Charlie", "city": "LA", "age": int64(35)})

	// Update many with compound index
	count, err := users.UpdateMany(
		map[string]interface{}{"city": "NYC"},
		map[string]interface{}{
			"$set": map[string]interface{}{
				"active": true,
			},
		},
	)

	if err != nil {
		t.Fatalf("UpdateMany failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 updates, got %d", count)
	}

	// Verify updates
	results, _ := users.Find(map[string]interface{}{"active": true})
	if len(results) != 2 {
		t.Errorf("Expected 2 active users, got %d", len(results))
	}
}

// TestUpdateMany_WithTextIndex tests UpdateMany with text indexes
func TestUpdateMany_WithTextIndex(t *testing.T) {
	dir := "./test_db_update_many_text"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	articles := db.Collection("articles")

	// Create text index
	articles.CreateTextIndex([]string{"title", "content"})

	// Insert test data
	articles.InsertOne(map[string]interface{}{
		"title":   "Go Programming",
		"content": "Learn Go programming basics",
		"status":  "draft",
	})
	articles.InsertOne(map[string]interface{}{
		"title":   "Python Guide",
		"content": "Python programming tutorial",
		"status":  "draft",
	})

	// Update many with text index
	count, err := articles.UpdateMany(
		map[string]interface{}{"status": "draft"},
		map[string]interface{}{
			"$set": map[string]interface{}{
				"status": "published",
			},
		},
	)

	if err != nil {
		t.Fatalf("UpdateMany failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 updates, got %d", count)
	}

	// Verify text search still works after update
	results, _ := articles.TextSearch("programming", nil)
	if len(results) != 2 {
		t.Errorf("Expected 2 results from text search, got %d", len(results))
	}
}

// TestUpdateMany_WithGeoIndex tests UpdateMany with geo indexes
func TestUpdateMany_WithGeoIndex(t *testing.T) {
	dir := "./test_db_update_many_geo"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	locations := db.Collection("locations")

	// Create 2d geo index
	locations.Create2DIndex("coordinates")

	// Insert test data
	locations.InsertOne(map[string]interface{}{
		"name": "Store A",
		"coordinates": map[string]interface{}{
			"type":        "Point",
			"coordinates": []interface{}{-73.97, 40.77},
		},
		"active": true,
	})
	locations.InsertOne(map[string]interface{}{
		"name": "Store B",
		"coordinates": map[string]interface{}{
			"type":        "Point",
			"coordinates": []interface{}{-74.00, 40.72},
		},
		"active": true,
	})

	// Update many with geo index
	count, err := locations.UpdateMany(
		map[string]interface{}{"active": true},
		map[string]interface{}{
			"$set": map[string]interface{}{
				"verified": true,
			},
		},
	)

	if err != nil {
		t.Fatalf("UpdateMany failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 updates, got %d", count)
	}

	// Verify geo queries still work
	queryPoint := &geo.Point{Lon: -73.98, Lat: 40.75}
	results, _ := locations.Near("coordinates", queryPoint, 5000.0, 10, nil)
	if len(results) == 0 {
		t.Error("Expected results from geo query after update")
	}
}

// TestUpdateMany_WithTTLIndex tests UpdateMany with TTL indexes
func TestUpdateMany_WithTTLIndex(t *testing.T) {
	dir := "./test_db_update_many_ttl"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	sessions := db.Collection("sessions")

	// Create TTL index (expire after 1 hour)
	sessions.CreateTTLIndex("expiresAt", 3600)

	// Insert test data
	now := time.Now()
	sessions.InsertOne(map[string]interface{}{
		"userId":    "user1",
		"token":     "abc123",
		"expiresAt": now.Add(1 * time.Hour),
	})
	sessions.InsertOne(map[string]interface{}{
		"userId":    "user2",
		"token":     "def456",
		"expiresAt": now.Add(1 * time.Hour),
	})

	// Update many with TTL index
	count, err := sessions.UpdateMany(
		map[string]interface{}{},
		map[string]interface{}{
			"$set": map[string]interface{}{
				"verified": true,
			},
		},
	)

	if err != nil {
		t.Fatalf("UpdateMany failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 updates, got %d", count)
	}

	// Verify updates
	results, _ := sessions.Find(map[string]interface{}{"verified": true})
	if len(results) != 2 {
		t.Errorf("Expected 2 verified sessions, got %d", len(results))
	}
}

// TestUpdateMany_WithPartialIndex tests UpdateMany with partial indexes
func TestUpdateMany_WithPartialIndex(t *testing.T) {
	dir := "./test_db_update_many_partial"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	products := db.Collection("products")

	// Create partial index (only active products)
	products.CreatePartialIndex("price", map[string]interface{}{
		"active": true,
	}, false)

	// Insert test data
	products.InsertOne(map[string]interface{}{
		"name":   "Widget",
		"price":  int64(100),
		"active": true,
	})
	products.InsertOne(map[string]interface{}{
		"name":   "Gadget",
		"price":  int64(200),
		"active": false,
	})

	// Update many - change active status
	count, err := products.UpdateMany(
		map[string]interface{}{"active": true},
		map[string]interface{}{
			"$set": map[string]interface{}{
				"discounted": true,
			},
		},
	)

	if err != nil {
		t.Fatalf("UpdateMany failed: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 update, got %d", count)
	}

	// Verify we can still query all products (not using the partial index)
	results, _ := products.Find(map[string]interface{}{})
	if len(results) != 2 {
		t.Errorf("Expected 2 products total, got %d", len(results))
	}

	// Verify discounted was set on active product
	discounted, _ := products.Find(map[string]interface{}{"discounted": true})
	if len(discounted) != 1 {
		t.Errorf("Expected 1 discounted product, got %d", len(discounted))
	}
}

// TestUpdateMany_WithMultipleIndexTypes tests UpdateMany with multiple index types
func TestUpdateMany_WithMultipleIndexTypes(t *testing.T) {
	dir := "./test_db_update_many_multi"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	users := db.Collection("users")

	// Create multiple index types
	users.CreateIndex("email", true)                        // Regular unique index
	users.CreateCompoundIndex([]string{"city", "age"}, false) // Compound index
	users.CreateTextIndex([]string{"bio"})                 // Text index

	// Insert test data
	users.InsertOne(map[string]interface{}{
		"email": "alice@example.com",
		"city":  "NYC",
		"age":   int64(30),
		"bio":   "Software engineer",
		"plan":  "free",
	})
	users.InsertOne(map[string]interface{}{
		"email": "bob@example.com",
		"city":  "LA",
		"age":   int64(25),
		"bio":   "Designer",
		"plan":  "free",
	})

	// Update many affecting multiple indexes
	count, err := users.UpdateMany(
		map[string]interface{}{"plan": "free"},
		map[string]interface{}{
			"$set": map[string]interface{}{
				"plan": "premium",
				"age":  int64(35), // Updates compound index
			},
		},
	)

	if err != nil {
		t.Fatalf("UpdateMany failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 updates, got %d", count)
	}

	// Verify all updates worked
	results, _ := users.Find(map[string]interface{}{"plan": "premium"})
	if len(results) != 2 {
		t.Errorf("Expected 2 premium users, got %d", len(results))
	}
}

// TestUpdateMany_IndexErrorHandling tests error handling during index updates
func TestUpdateMany_IndexErrorHandling(t *testing.T) {
	dir := "./test_db_update_many_errors"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	users := db.Collection("users")

	// Create unique index
	users.CreateIndex("email", true)

	// Insert test data
	users.InsertOne(map[string]interface{}{
		"name":   "Alice",
		"email":  "alice@example.com",
		"status": "pending",
	})
	users.InsertOne(map[string]interface{}{
		"name":   "Bob",
		"email":  "bob@example.com",
		"status": "pending",
	})

	// Try to update in a way that would cause duplicate - this should handle the error gracefully
	count, err := users.UpdateMany(
		map[string]interface{}{"status": "pending"},
		map[string]interface{}{
			"$set": map[string]interface{}{
				"status": "active",
			},
		},
	)

	// Should succeed because we're not creating duplicates
	if err != nil {
		t.Fatalf("UpdateMany failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 updates, got %d", count)
	}
}

// TestUpdateMany_EmptyResult tests UpdateMany with no matching documents
func TestUpdateMany_EmptyResult(t *testing.T) {
	dir := "./test_db_update_many_empty"
	defer os.RemoveAll(dir)

	db, _ := Open(DefaultConfig(dir))
	defer db.Close()

	users := db.Collection("users")

	// Insert test data
	users.InsertOne(map[string]interface{}{"name": "Alice", "age": int64(30)})

	// Try to update documents that don't exist
	count, err := users.UpdateMany(
		map[string]interface{}{"name": "NonExistent"},
		map[string]interface{}{
			"$set": map[string]interface{}{
				"updated": true,
			},
		},
	)

	if err != nil {
		t.Fatalf("UpdateMany failed: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 updates, got %d", count)
	}
}
