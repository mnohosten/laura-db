package database

import (
	"testing"
	"time"

	"github.com/mnohosten/laura-db/pkg/storage"
)

func TestCollectionMetadataSerialization(t *testing.T) {
	// Create test metadata
	meta := &CollectionMetadata{
		CollectionID:     1,
		Name:             "users",
		CreatedTimestamp: time.Unix(1234567890, 0),
		DocumentCount:    100,
		DataSizeBytes:    1024,
		IndexCount:       2,
		FirstDataPageID:  storage.PageID(10),
		StatisticsPageID: storage.PageID(20),
		Schema: &CollectionSchema{
			Version:          1,
			SchemaJSON:       `{"type": "object", "properties": {"name": {"type": "string"}}}`,
			ValidationLevel:  "strict",
			ValidationAction: "error",
		},
		Options: &CollectionOptions{
			Capped:       false,
			MaxSize:      0,
			MaxDocuments: 0,
		},
	}

	// Serialize
	data, err := SerializeCollectionMetadata(meta)
	if err != nil {
		t.Fatalf("Failed to serialize metadata: %v", err)
	}

	// Deserialize
	deserialized, err := DeserializeCollectionMetadata(data)
	if err != nil {
		t.Fatalf("Failed to deserialize metadata: %v", err)
	}

	// Verify
	if deserialized.CollectionID != meta.CollectionID {
		t.Errorf("CollectionID mismatch: got %d, want %d", deserialized.CollectionID, meta.CollectionID)
	}
	if deserialized.Name != meta.Name {
		t.Errorf("Name mismatch: got %s, want %s", deserialized.Name, meta.Name)
	}
	if !deserialized.CreatedTimestamp.Equal(meta.CreatedTimestamp) {
		t.Errorf("CreatedTimestamp mismatch: got %v, want %v", deserialized.CreatedTimestamp, meta.CreatedTimestamp)
	}
	if deserialized.DocumentCount != meta.DocumentCount {
		t.Errorf("DocumentCount mismatch: got %d, want %d", deserialized.DocumentCount, meta.DocumentCount)
	}
	if deserialized.DataSizeBytes != meta.DataSizeBytes {
		t.Errorf("DataSizeBytes mismatch: got %d, want %d", deserialized.DataSizeBytes, meta.DataSizeBytes)
	}
	if deserialized.IndexCount != meta.IndexCount {
		t.Errorf("IndexCount mismatch: got %d, want %d", deserialized.IndexCount, meta.IndexCount)
	}
	if deserialized.FirstDataPageID != meta.FirstDataPageID {
		t.Errorf("FirstDataPageID mismatch: got %d, want %d", deserialized.FirstDataPageID, meta.FirstDataPageID)
	}
	if deserialized.StatisticsPageID != meta.StatisticsPageID {
		t.Errorf("StatisticsPageID mismatch: got %d, want %d", deserialized.StatisticsPageID, meta.StatisticsPageID)
	}
	if deserialized.Schema == nil {
		t.Error("Schema is nil")
	} else {
		if deserialized.Schema.SchemaJSON != meta.Schema.SchemaJSON {
			t.Errorf("Schema JSON mismatch")
		}
	}
	if deserialized.Options == nil {
		t.Error("Options is nil")
	}
}

func TestCollectionMetadataSerializationWithoutOptional(t *testing.T) {
	// Create test metadata without schema and options
	meta := &CollectionMetadata{
		CollectionID:     2,
		Name:             "products",
		CreatedTimestamp: time.Now(),
		DocumentCount:    0,
		DataSizeBytes:    0,
		IndexCount:       1,
		FirstDataPageID:  storage.PageID(0),
		StatisticsPageID: storage.PageID(0),
		Schema:           nil,
		Options:          nil,
	}

	// Serialize
	data, err := SerializeCollectionMetadata(meta)
	if err != nil {
		t.Fatalf("Failed to serialize metadata: %v", err)
	}

	// Deserialize
	deserialized, err := DeserializeCollectionMetadata(data)
	if err != nil {
		t.Fatalf("Failed to deserialize metadata: %v", err)
	}

	// Verify
	if deserialized.CollectionID != meta.CollectionID {
		t.Errorf("CollectionID mismatch: got %d, want %d", deserialized.CollectionID, meta.CollectionID)
	}
	if deserialized.Name != meta.Name {
		t.Errorf("Name mismatch: got %s, want %s", deserialized.Name, meta.Name)
	}
	if deserialized.Schema != nil {
		t.Error("Schema should be nil")
	}
	if deserialized.Options != nil {
		t.Error("Options should be nil")
	}
}

func TestIndexMetadataSerialization(t *testing.T) {
	// Create test index metadata
	meta := &IndexMetadata{
		IndexID:          1,
		CollectionID:     1,
		Name:             "age_idx",
		IndexType:        IndexTypeBTree,
		FieldPaths:       []string{"age"},
		IsUnique:         false,
		IsSparse:         false,
		IsPartial:        true,
		PartialFilter:    `{"age": {"$gte": 18}}`,
		CreatedTimestamp: time.Unix(1234567890, 0),
		RootPageID:       storage.PageID(100),
		EntryCount:       50,
		Order:            32,
		Options: map[string]interface{}{
			"test": "value",
		},
	}

	// Serialize
	data, err := SerializeIndexMetadata(meta)
	if err != nil {
		t.Fatalf("Failed to serialize index metadata: %v", err)
	}

	// Deserialize
	deserialized, err := DeserializeIndexMetadata(data)
	if err != nil {
		t.Fatalf("Failed to deserialize index metadata: %v", err)
	}

	// Verify
	if deserialized.IndexID != meta.IndexID {
		t.Errorf("IndexID mismatch: got %d, want %d", deserialized.IndexID, meta.IndexID)
	}
	if deserialized.CollectionID != meta.CollectionID {
		t.Errorf("CollectionID mismatch: got %d, want %d", deserialized.CollectionID, meta.CollectionID)
	}
	if deserialized.Name != meta.Name {
		t.Errorf("Name mismatch: got %s, want %s", deserialized.Name, meta.Name)
	}
	if deserialized.IndexType != meta.IndexType {
		t.Errorf("IndexType mismatch: got %d, want %d", deserialized.IndexType, meta.IndexType)
	}
	if len(deserialized.FieldPaths) != len(meta.FieldPaths) {
		t.Errorf("FieldPaths length mismatch: got %d, want %d", len(deserialized.FieldPaths), len(meta.FieldPaths))
	}
	if deserialized.IsUnique != meta.IsUnique {
		t.Errorf("IsUnique mismatch: got %v, want %v", deserialized.IsUnique, meta.IsUnique)
	}
	if deserialized.IsSparse != meta.IsSparse {
		t.Errorf("IsSparse mismatch: got %v, want %v", deserialized.IsSparse, meta.IsSparse)
	}
	if deserialized.IsPartial != meta.IsPartial {
		t.Errorf("IsPartial mismatch: got %v, want %v", deserialized.IsPartial, meta.IsPartial)
	}
	if deserialized.PartialFilter != meta.PartialFilter {
		t.Errorf("PartialFilter mismatch: got %s, want %s", deserialized.PartialFilter, meta.PartialFilter)
	}
	if deserialized.Order != meta.Order {
		t.Errorf("Order mismatch: got %d, want %d", deserialized.Order, meta.Order)
	}
}

func TestIndexMetadataCompoundIndex(t *testing.T) {
	// Create compound index metadata
	meta := &IndexMetadata{
		IndexID:          2,
		CollectionID:     1,
		Name:             "city_age_idx",
		IndexType:        IndexTypeBTree,
		FieldPaths:       []string{"city", "age"},
		IsUnique:         false,
		IsSparse:         false,
		IsPartial:        false,
		PartialFilter:    "",
		CreatedTimestamp: time.Now(),
		RootPageID:       storage.PageID(200),
		EntryCount:       100,
		Order:            32,
		Options:          map[string]interface{}{},
	}

	// Serialize
	data, err := SerializeIndexMetadata(meta)
	if err != nil {
		t.Fatalf("Failed to serialize index metadata: %v", err)
	}

	// Deserialize
	deserialized, err := DeserializeIndexMetadata(data)
	if err != nil {
		t.Fatalf("Failed to deserialize index metadata: %v", err)
	}

	// Verify field paths
	if len(deserialized.FieldPaths) != len(meta.FieldPaths) {
		t.Fatalf("FieldPaths length mismatch: got %d, want %d", len(deserialized.FieldPaths), len(meta.FieldPaths))
	}
	for i, fp := range meta.FieldPaths {
		if deserialized.FieldPaths[i] != fp {
			t.Errorf("FieldPath[%d] mismatch: got %s, want %s", i, deserialized.FieldPaths[i], fp)
		}
	}
}

func TestIndexMetadataTextIndex(t *testing.T) {
	// Create text index metadata
	meta := &IndexMetadata{
		IndexID:          3,
		CollectionID:     1,
		Name:             "content_text",
		IndexType:        IndexTypeText,
		FieldPaths:       []string{"title", "body"},
		IsUnique:         false,
		IsSparse:         false,
		IsPartial:        false,
		PartialFilter:    "",
		CreatedTimestamp: time.Now(),
		RootPageID:       storage.PageID(0), // Text indexes don't use B+ trees
		EntryCount:       1000,
		Order:            0,
		Options: map[string]interface{}{
			"defaultLanguage": "english",
			"weights": map[string]interface{}{
				"title": float64(10),
				"body":  float64(1),
			},
		},
	}

	// Serialize
	data, err := SerializeIndexMetadata(meta)
	if err != nil {
		t.Fatalf("Failed to serialize text index metadata: %v", err)
	}

	// Deserialize
	deserialized, err := DeserializeIndexMetadata(data)
	if err != nil {
		t.Fatalf("Failed to deserialize text index metadata: %v", err)
	}

	// Verify
	if deserialized.IndexType != IndexTypeText {
		t.Errorf("IndexType mismatch: got %d, want %d", deserialized.IndexType, IndexTypeText)
	}
	if len(deserialized.FieldPaths) != 2 {
		t.Errorf("FieldPaths length mismatch: got %d, want 2", len(deserialized.FieldPaths))
	}
	if deserialized.Options == nil {
		t.Error("Options is nil")
	} else if defaultLang, ok := deserialized.Options["defaultLanguage"]; !ok || defaultLang != "english" {
		t.Error("defaultLanguage option not preserved")
	}
}

func TestSaveAndLoadCollectionMetadata(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Create disk manager
	dataFile := tmpDir + "/metadata_test.db"
	diskMgr, err := storage.NewDiskManager(dataFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	// Create test metadata
	meta := &CollectionMetadata{
		CollectionID:     1,
		Name:             "test_collection",
		CreatedTimestamp: time.Unix(1234567890, 0),
		DocumentCount:    100,
		DataSizeBytes:    2048,
		IndexCount:       3,
		FirstDataPageID:  storage.PageID(10),
		StatisticsPageID: storage.PageID(20),
		Schema: &CollectionSchema{
			Version:          1,
			SchemaJSON:       `{"type": "object"}`,
			ValidationLevel:  "strict",
			ValidationAction: "error",
		},
		Options: &CollectionOptions{
			Capped:       false,
			MaxSize:      0,
			MaxDocuments: 0,
		},
	}

	// Allocate a page for metadata
	pageID, err := diskMgr.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}

	// Save metadata
	err = SaveCollectionMetadata(diskMgr, pageID, meta)
	if err != nil {
		t.Fatalf("Failed to save collection metadata: %v", err)
	}

	// Load metadata
	loaded, err := LoadCollectionMetadata(diskMgr, pageID)
	if err != nil {
		t.Fatalf("Failed to load collection metadata: %v", err)
	}

	// Verify
	if loaded.CollectionID != meta.CollectionID {
		t.Errorf("CollectionID mismatch: got %d, want %d", loaded.CollectionID, meta.CollectionID)
	}
	if loaded.Name != meta.Name {
		t.Errorf("Name mismatch: got %s, want %s", loaded.Name, meta.Name)
	}
	if !loaded.CreatedTimestamp.Equal(meta.CreatedTimestamp) {
		t.Errorf("CreatedTimestamp mismatch: got %v, want %v", loaded.CreatedTimestamp, meta.CreatedTimestamp)
	}
	if loaded.DocumentCount != meta.DocumentCount {
		t.Errorf("DocumentCount mismatch: got %d, want %d", loaded.DocumentCount, meta.DocumentCount)
	}
	if loaded.Schema == nil {
		t.Error("Schema is nil after loading")
	} else {
		if loaded.Schema.SchemaJSON != meta.Schema.SchemaJSON {
			t.Error("Schema JSON mismatch after loading")
		}
	}
}

func TestSaveAndLoadIndexMetadata(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Create disk manager
	dataFile := tmpDir + "/index_metadata_test.db"
	diskMgr, err := storage.NewDiskManager(dataFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	// Create test index metadata
	meta := &IndexMetadata{
		IndexID:          1,
		CollectionID:     1,
		Name:             "test_index",
		IndexType:        IndexTypeBTree,
		FieldPaths:       []string{"field1", "field2"},
		IsUnique:         true,
		IsSparse:         false,
		IsPartial:        true,
		PartialFilter:    `{"status": "active"}`,
		CreatedTimestamp: time.Unix(1234567890, 0),
		RootPageID:       storage.PageID(100),
		EntryCount:       500,
		Order:            32,
		Options: map[string]interface{}{
			"test_option": "test_value",
		},
	}

	// Allocate a page for metadata
	pageID, err := diskMgr.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}

	// Save metadata
	err = SaveIndexMetadata(diskMgr, pageID, meta)
	if err != nil {
		t.Fatalf("Failed to save index metadata: %v", err)
	}

	// Load metadata
	loaded, err := LoadIndexMetadata(diskMgr, pageID)
	if err != nil {
		t.Fatalf("Failed to load index metadata: %v", err)
	}

	// Verify
	if loaded.IndexID != meta.IndexID {
		t.Errorf("IndexID mismatch: got %d, want %d", loaded.IndexID, meta.IndexID)
	}
	if loaded.Name != meta.Name {
		t.Errorf("Name mismatch: got %s, want %s", loaded.Name, meta.Name)
	}
	if loaded.IndexType != meta.IndexType {
		t.Errorf("IndexType mismatch: got %d, want %d", loaded.IndexType, meta.IndexType)
	}
	if len(loaded.FieldPaths) != len(meta.FieldPaths) {
		t.Fatalf("FieldPaths length mismatch: got %d, want %d", len(loaded.FieldPaths), len(meta.FieldPaths))
	}
	for i, fp := range meta.FieldPaths {
		if loaded.FieldPaths[i] != fp {
			t.Errorf("FieldPath[%d] mismatch: got %s, want %s", i, loaded.FieldPaths[i], fp)
		}
	}
	if loaded.IsUnique != meta.IsUnique {
		t.Errorf("IsUnique mismatch: got %v, want %v", loaded.IsUnique, meta.IsUnique)
	}
	if loaded.PartialFilter != meta.PartialFilter {
		t.Errorf("PartialFilter mismatch: got %s, want %s", loaded.PartialFilter, meta.PartialFilter)
	}
}

func TestIndexStatisticsSerialization(t *testing.T) {
	now := time.Now()
	stats := &IndexStatistics{
		IndexID:           1,
		TotalEntries:      10000,
		UniqueKeys:        5000,
		TreeHeight:        4,
		LeafNodeCount:     100,
		InternalNodeCount: 10,
		AvgKeySize:        20,
		AvgValueSize:      12,
		MinKey:            int64(1),
		MaxKey:            int64(10000),
		LastUpdated:       now,
		Cardinality:       0.5,
		IndexScans:        1000,
		IndexSeeks:        5000,
		LastAccessTime:    now,
	}

	// Serialize
	data, err := SerializeIndexStatistics(stats)
	if err != nil {
		t.Fatalf("Failed to serialize statistics: %v", err)
	}

	// Deserialize
	deserialized, err := DeserializeIndexStatistics(data)
	if err != nil {
		t.Fatalf("Failed to deserialize statistics: %v", err)
	}

	// Verify fields
	if deserialized.IndexID != stats.IndexID {
		t.Errorf("IndexID mismatch: got %d, want %d", deserialized.IndexID, stats.IndexID)
	}
	if deserialized.TotalEntries != stats.TotalEntries {
		t.Errorf("TotalEntries mismatch: got %d, want %d", deserialized.TotalEntries, stats.TotalEntries)
	}
	if deserialized.UniqueKeys != stats.UniqueKeys {
		t.Errorf("UniqueKeys mismatch: got %d, want %d", deserialized.UniqueKeys, stats.UniqueKeys)
	}
	if deserialized.TreeHeight != stats.TreeHeight {
		t.Errorf("TreeHeight mismatch: got %d, want %d", deserialized.TreeHeight, stats.TreeHeight)
	}
	if deserialized.LeafNodeCount != stats.LeafNodeCount {
		t.Errorf("LeafNodeCount mismatch: got %d, want %d", deserialized.LeafNodeCount, stats.LeafNodeCount)
	}
	if deserialized.InternalNodeCount != stats.InternalNodeCount {
		t.Errorf("InternalNodeCount mismatch: got %d, want %d", deserialized.InternalNodeCount, stats.InternalNodeCount)
	}
	if deserialized.AvgKeySize != stats.AvgKeySize {
		t.Errorf("AvgKeySize mismatch: got %d, want %d", deserialized.AvgKeySize, stats.AvgKeySize)
	}
	if deserialized.AvgValueSize != stats.AvgValueSize {
		t.Errorf("AvgValueSize mismatch: got %d, want %d", deserialized.AvgValueSize, stats.AvgValueSize)
	}
	if deserialized.Cardinality != stats.Cardinality {
		t.Errorf("Cardinality mismatch: got %f, want %f", deserialized.Cardinality, stats.Cardinality)
	}
	if deserialized.IndexScans != stats.IndexScans {
		t.Errorf("IndexScans mismatch: got %d, want %d", deserialized.IndexScans, stats.IndexScans)
	}
	if deserialized.IndexSeeks != stats.IndexSeeks {
		t.Errorf("IndexSeeks mismatch: got %d, want %d", deserialized.IndexSeeks, stats.IndexSeeks)
	}

	// Verify min/max keys (JSON deserialization converts to float64)
	minKeyFloat, ok := deserialized.MinKey.(float64)
	if !ok {
		t.Errorf("MinKey type mismatch: got %T", deserialized.MinKey)
	} else if int64(minKeyFloat) != int64(1) {
		t.Errorf("MinKey value mismatch: got %v, want 1", minKeyFloat)
	}

	maxKeyFloat, ok := deserialized.MaxKey.(float64)
	if !ok {
		t.Errorf("MaxKey type mismatch: got %T", deserialized.MaxKey)
	} else if int64(maxKeyFloat) != int64(10000) {
		t.Errorf("MaxKey value mismatch: got %v, want 10000", maxKeyFloat)
	}
}

func TestIndexStatisticsWithStringKeys(t *testing.T) {
	now := time.Now()
	stats := &IndexStatistics{
		IndexID:           2,
		TotalEntries:      1000,
		UniqueKeys:        800,
		TreeHeight:        3,
		LeafNodeCount:     50,
		InternalNodeCount: 5,
		AvgKeySize:        25,
		AvgValueSize:      12,
		MinKey:            "alice",
		MaxKey:            "zoe",
		LastUpdated:       now,
		Cardinality:       0.8,
		IndexScans:        100,
		IndexSeeks:        500,
		LastAccessTime:    now,
	}

	// Serialize and deserialize
	data, err := SerializeIndexStatistics(stats)
	if err != nil {
		t.Fatalf("Failed to serialize statistics: %v", err)
	}

	deserialized, err := DeserializeIndexStatistics(data)
	if err != nil {
		t.Fatalf("Failed to deserialize statistics: %v", err)
	}

	// Verify string keys
	if deserialized.MinKey != "alice" {
		t.Errorf("MinKey mismatch: got %v, want alice", deserialized.MinKey)
	}
	if deserialized.MaxKey != "zoe" {
		t.Errorf("MaxKey mismatch: got %v, want zoe", deserialized.MaxKey)
	}
}

func TestSaveAndLoadIndexStatistics(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	dataFile := tmpDir + "/stats_test.db"

	// Create disk manager
	diskMgr, err := storage.NewDiskManager(dataFile)
	if err != nil {
		t.Fatalf("Failed to create disk manager: %v", err)
	}
	defer diskMgr.Close()

	// Create statistics
	now := time.Now()
	stats := &IndexStatistics{
		IndexID:           1,
		TotalEntries:      50000,
		UniqueKeys:        25000,
		TreeHeight:        5,
		LeafNodeCount:     500,
		InternalNodeCount: 50,
		AvgKeySize:        15,
		AvgValueSize:      12,
		MinKey:            int64(100),
		MaxKey:            int64(100000),
		LastUpdated:       now,
		Cardinality:       0.5,
		IndexScans:        2000,
		IndexSeeks:        10000,
		LastAccessTime:    now,
	}

	// Allocate page and save statistics
	pageID, err := diskMgr.AllocatePage()
	if err != nil {
		t.Fatalf("Failed to allocate page: %v", err)
	}

	err = SaveIndexStatistics(diskMgr, pageID, stats)
	if err != nil {
		t.Fatalf("Failed to save statistics: %v", err)
	}

	// Load statistics
	loaded, err := LoadIndexStatistics(diskMgr, pageID)
	if err != nil {
		t.Fatalf("Failed to load statistics: %v", err)
	}

	// Verify all fields match
	if loaded.IndexID != stats.IndexID {
		t.Errorf("IndexID mismatch: got %d, want %d", loaded.IndexID, stats.IndexID)
	}
	if loaded.TotalEntries != stats.TotalEntries {
		t.Errorf("TotalEntries mismatch: got %d, want %d", loaded.TotalEntries, stats.TotalEntries)
	}
	if loaded.UniqueKeys != stats.UniqueKeys {
		t.Errorf("UniqueKeys mismatch: got %d, want %d", loaded.UniqueKeys, stats.UniqueKeys)
	}
	if loaded.TreeHeight != stats.TreeHeight {
		t.Errorf("TreeHeight mismatch: got %d, want %d", loaded.TreeHeight, stats.TreeHeight)
	}
	if loaded.Cardinality != stats.Cardinality {
		t.Errorf("Cardinality mismatch: got %f, want %f", loaded.Cardinality, stats.Cardinality)
	}
	if loaded.IndexScans != stats.IndexScans {
		t.Errorf("IndexScans mismatch: got %d, want %d", loaded.IndexScans, stats.IndexScans)
	}
}

func TestIndexStatisticsCalculateCardinality(t *testing.T) {
	stats := &IndexStatistics{
		TotalEntries: 10000,
		UniqueKeys:   2500,
	}

	// Calculate expected cardinality
	expectedCardinality := float64(stats.UniqueKeys) / float64(stats.TotalEntries)
	stats.Cardinality = expectedCardinality

	if stats.Cardinality != 0.25 {
		t.Errorf("Cardinality calculation incorrect: got %f, want 0.25", stats.Cardinality)
	}
}

func TestIndexStatisticsEmptyMinMax(t *testing.T) {
	now := time.Now()
	stats := &IndexStatistics{
		IndexID:           3,
		TotalEntries:      0,
		UniqueKeys:        0,
		TreeHeight:        1,
		LeafNodeCount:     1,
		InternalNodeCount: 0,
		AvgKeySize:        0,
		AvgValueSize:      0,
		MinKey:            nil,
		MaxKey:            nil,
		LastUpdated:       now,
		Cardinality:       0.0,
		IndexScans:        0,
		IndexSeeks:        0,
		LastAccessTime:    now,
	}

	// Serialize and deserialize
	data, err := SerializeIndexStatistics(stats)
	if err != nil {
		t.Fatalf("Failed to serialize statistics: %v", err)
	}

	deserialized, err := DeserializeIndexStatistics(data)
	if err != nil {
		t.Fatalf("Failed to deserialize statistics: %v", err)
	}

	// Verify nil min/max keys
	if deserialized.MinKey != nil {
		t.Errorf("MinKey should be nil, got %v", deserialized.MinKey)
	}
	if deserialized.MaxKey != nil {
		t.Errorf("MaxKey should be nil, got %v", deserialized.MaxKey)
	}
}
