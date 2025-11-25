package database

import (
	"testing"

	"github.com/mnohosten/laura-db/pkg/storage"
)

func TestNewCollectionCatalog(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	dataFile := tmpDir + "/catalog_test.db"

	// Create disk manager
	diskMgr, err := storage.NewDiskManager(dataFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	// Create catalog
	catalog, err := NewCollectionCatalog(diskMgr)
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	// Verify initial state
	if catalog.header.MagicNumber != CatalogMagicNumber {
		t.Errorf("Invalid magic number: got 0x%X, want 0x%X", catalog.header.MagicNumber, CatalogMagicNumber)
	}
	if catalog.header.Version != CatalogVersion {
		t.Errorf("Invalid version: got %d, want %d", catalog.header.Version, CatalogVersion)
	}
	if catalog.header.CollectionCount != 0 {
		t.Errorf("Initial collection count should be 0, got %d", catalog.header.CollectionCount)
	}
	if catalog.header.NextCollectionID != 1 {
		t.Errorf("Initial next collection ID should be 1, got %d", catalog.header.NextCollectionID)
	}
}

func TestRegisterCollection(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	dataFile := tmpDir + "/catalog_test.db"

	// Create disk manager
	diskMgr, err := storage.NewDiskManager(dataFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	// Create catalog
	catalog, err := NewCollectionCatalog(diskMgr)
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	// Register a collection
	metadataPageID := storage.PageID(100)
	collectionID, err := catalog.RegisterCollection("users", metadataPageID)
	if err != nil {
		t.Fatalf("Failed to register collection: %v", err)
	}

	// Verify collection ID
	if collectionID != 1 {
		t.Errorf("Expected collection ID 1, got %d", collectionID)
	}

	// Verify catalog state
	if catalog.header.CollectionCount != 1 {
		t.Errorf("Expected collection count 1, got %d", catalog.header.CollectionCount)
	}
	if catalog.header.NextCollectionID != 2 {
		t.Errorf("Expected next collection ID 2, got %d", catalog.header.NextCollectionID)
	}

	// Verify collection entry
	entry, err := catalog.GetCollection("users")
	if err != nil {
		t.Fatalf("Failed to get collection: %v", err)
	}
	if entry.CollectionID != collectionID {
		t.Errorf("Collection ID mismatch: got %d, want %d", entry.CollectionID, collectionID)
	}
	if entry.Name != "users" {
		t.Errorf("Collection name mismatch: got %s, want users", entry.Name)
	}
	if entry.MetadataPageID != metadataPageID {
		t.Errorf("Metadata page ID mismatch: got %d, want %d", entry.MetadataPageID, metadataPageID)
	}
	if entry.Flags&CollectionFlagActive == 0 {
		t.Error("Collection should be active")
	}
}

func TestRegisterMultipleCollections(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	dataFile := tmpDir + "/catalog_test.db"

	// Create disk manager
	diskMgr, err := storage.NewDiskManager(dataFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	// Create catalog
	catalog, err := NewCollectionCatalog(diskMgr)
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	// Register multiple collections
	collections := []string{"users", "products", "orders"}
	for i, name := range collections {
		metadataPageID := storage.PageID(100 + i)
		collectionID, err := catalog.RegisterCollection(name, metadataPageID)
		if err != nil {
			t.Fatalf("Failed to register collection %s: %v", name, err)
		}
		if collectionID != uint32(i+1) {
			t.Errorf("Expected collection ID %d, got %d", i+1, collectionID)
		}
	}

	// Verify collection count
	if catalog.header.CollectionCount != 3 {
		t.Errorf("Expected collection count 3, got %d", catalog.header.CollectionCount)
	}

	// Verify all collections can be retrieved
	for _, name := range collections {
		_, err := catalog.GetCollection(name)
		if err != nil {
			t.Errorf("Failed to get collection %s: %v", name, err)
		}
	}
}

func TestRegisterDuplicateCollection(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	dataFile := tmpDir + "/catalog_test.db"

	// Create disk manager
	diskMgr, err := storage.NewDiskManager(dataFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	// Create catalog
	catalog, err := NewCollectionCatalog(diskMgr)
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	// Register a collection
	_, err = catalog.RegisterCollection("users", storage.PageID(100))
	if err != nil {
		t.Fatalf("Failed to register collection: %v", err)
	}

	// Try to register the same collection again
	_, err = catalog.RegisterCollection("users", storage.PageID(200))
	if err == nil {
		t.Error("Expected error when registering duplicate collection")
	}
}

func TestGetNonExistentCollection(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	dataFile := tmpDir + "/catalog_test.db"

	// Create disk manager
	diskMgr, err := storage.NewDiskManager(dataFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	// Create catalog
	catalog, err := NewCollectionCatalog(diskMgr)
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	// Try to get non-existent collection
	_, err = catalog.GetCollection("nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent collection")
	}
}

func TestCatalogListCollections(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	dataFile := tmpDir + "/catalog_test.db"

	// Create disk manager
	diskMgr, err := storage.NewDiskManager(dataFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	// Create catalog
	catalog, err := NewCollectionCatalog(diskMgr)
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	// Initially no collections
	collections := catalog.ListCollections()
	if len(collections) != 0 {
		t.Errorf("Expected 0 collections, got %d", len(collections))
	}

	// Register some collections
	catalog.RegisterCollection("users", storage.PageID(100))
	catalog.RegisterCollection("products", storage.PageID(200))
	catalog.RegisterCollection("orders", storage.PageID(300))

	// List collections
	collections = catalog.ListCollections()
	if len(collections) != 3 {
		t.Errorf("Expected 3 collections, got %d", len(collections))
	}

	// Verify all collections are in the list
	collectionMap := make(map[string]bool)
	for _, name := range collections {
		collectionMap[name] = true
	}
	if !collectionMap["users"] || !collectionMap["products"] || !collectionMap["orders"] {
		t.Error("Not all collections found in list")
	}
}

func TestCatalogDropCollection(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	dataFile := tmpDir + "/catalog_test.db"

	// Create disk manager
	diskMgr, err := storage.NewDiskManager(dataFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	// Create catalog
	catalog, err := NewCollectionCatalog(diskMgr)
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	// Register a collection
	catalog.RegisterCollection("users", storage.PageID(100))

	// Verify it exists
	_, err = catalog.GetCollection("users")
	if err != nil {
		t.Fatalf("Collection should exist: %v", err)
	}

	// Drop the collection
	err = catalog.DropCollection("users")
	if err != nil {
		t.Fatalf("Failed to drop collection: %v", err)
	}

	// Verify it no longer exists
	_, err = catalog.GetCollection("users")
	if err == nil {
		t.Error("Collection should not exist after drop")
	}

	// Verify collection count
	if catalog.header.CollectionCount != 0 {
		t.Errorf("Expected collection count 0, got %d", catalog.header.CollectionCount)
	}
}

func TestDropNonExistentCollection(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	dataFile := tmpDir + "/catalog_test.db"

	// Create disk manager
	diskMgr, err := storage.NewDiskManager(dataFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	// Create catalog
	catalog, err := NewCollectionCatalog(diskMgr)
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	// Try to drop non-existent collection
	err = catalog.DropCollection("nonexistent")
	if err == nil {
		t.Error("Expected error when dropping non-existent collection")
	}
}

func TestCatalogPersistence(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	dataFile := tmpDir + "/catalog_test.db"

	// Create disk manager
	diskMgr, err := storage.NewDiskManager(dataFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}

	// Create catalog and register collections
	catalog1, err := NewCollectionCatalog(diskMgr)
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	catalog1.RegisterCollection("users", storage.PageID(100))
	catalog1.RegisterCollection("products", storage.PageID(200))

	// Close disk manager
	diskMgr.Close()

	// Reopen disk manager
	diskMgr, err = storage.NewDiskManager(dataFile)
	if err != nil {
		t.Fatalf("Failed to reopen disk manager: %v", err)
	}
	defer diskMgr.Close()

	// Load catalog from disk
	catalog2, err := NewCollectionCatalog(diskMgr)
	if err != nil {
		t.Fatalf("Failed to load catalog: %v", err)
	}

	// Verify collections were persisted
	if catalog2.header.CollectionCount != 2 {
		t.Errorf("Expected collection count 2, got %d", catalog2.header.CollectionCount)
	}

	entry1, err := catalog2.GetCollection("users")
	if err != nil {
		t.Errorf("Failed to get users collection: %v", err)
	}
	if entry1.MetadataPageID != storage.PageID(100) {
		t.Errorf("Users metadata page ID mismatch: got %d, want 100", entry1.MetadataPageID)
	}

	entry2, err := catalog2.GetCollection("products")
	if err != nil {
		t.Errorf("Failed to get products collection: %v", err)
	}
	if entry2.MetadataPageID != storage.PageID(200) {
		t.Errorf("Products metadata page ID mismatch: got %d, want 200", entry2.MetadataPageID)
	}
}

func TestValidateCollectionName(t *testing.T) {
	tests := []struct {
		name      string
		wantError bool
	}{
		{"users", false},
		{"my_collection", false},
		{"collection-123", false},
		{"ABC123", false},
		{"", true},                                         // Empty
		{"system.internal", true},                          // Reserved prefix
		{"invalid name", true},                             // Space
		{"invalid@name", true},                             // Special character
		{string(make([]byte, 256)), true},                  // Too long
	}

	for _, tt := range tests {
		err := validateCollectionName(tt.name)
		if (err != nil) != tt.wantError {
			t.Errorf("validateCollectionName(%q) error = %v, wantError %v", tt.name, err, tt.wantError)
		}
	}
}

func TestGetCollectionCount(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	dataFile := tmpDir + "/catalog_test.db"

	// Create disk manager
	diskMgr, err := storage.NewDiskManager(dataFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	// Create catalog
	catalog, err := NewCollectionCatalog(diskMgr)
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	// Initially 0
	if catalog.GetCollectionCount() != 0 {
		t.Errorf("Expected count 0, got %d", catalog.GetCollectionCount())
	}

	// Add collections
	catalog.RegisterCollection("users", storage.PageID(100))
	if catalog.GetCollectionCount() != 1 {
		t.Errorf("Expected count 1, got %d", catalog.GetCollectionCount())
	}

	catalog.RegisterCollection("products", storage.PageID(200))
	if catalog.GetCollectionCount() != 2 {
		t.Errorf("Expected count 2, got %d", catalog.GetCollectionCount())
	}

	// Drop one
	catalog.DropCollection("users")
	if catalog.GetCollectionCount() != 1 {
		t.Errorf("Expected count 1 after drop, got %d", catalog.GetCollectionCount())
	}
}
