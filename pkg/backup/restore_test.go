package backup

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/mnohosten/laura-db/pkg/document"
)

func TestRestorer_RestoreFromReader(t *testing.T) {
	// Create a valid backup JSON
	backup := NewBackupFormat("test_db")
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":  "doc1",
			"name": "Alice",
		}),
	}
	backup.AddCollection("users", docs, []IndexBackup{})

	// Encode to JSON
	var buf bytes.Buffer
	backuper := NewBackuper(false)
	err := backuper.BackupToWriter(&buf, backup)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Restore from JSON
	restorer := NewRestorer()
	restored, err := restorer.RestoreFromReader(&buf)
	if err != nil {
		t.Fatalf("RestoreFromReader failed: %v", err)
	}

	if restored.DatabaseName != "test_db" {
		t.Errorf("Expected database name test_db, got %s", restored.DatabaseName)
	}

	if len(restored.Collections) != 1 {
		t.Errorf("Expected 1 collection, got %d", len(restored.Collections))
	}
}

func TestRestorer_ValidateBackup(t *testing.T) {
	restorer := NewRestorer()

	tests := []struct {
		name        string
		backup      *BackupFormat
		expectError bool
	}{
		{
			name: "Valid backup",
			backup: &BackupFormat{
				Version:      "1.0",
				Timestamp:    time.Now(),
				DatabaseName: "test_db",
				Collections:  []CollectionBackup{},
			},
			expectError: false,
		},
		{
			name: "Missing version",
			backup: &BackupFormat{
				Timestamp:    time.Now(),
				DatabaseName: "test_db",
				Collections:  []CollectionBackup{},
			},
			expectError: true,
		},
		{
			name: "Unsupported version",
			backup: &BackupFormat{
				Version:      "2.0",
				Timestamp:    time.Now(),
				DatabaseName: "test_db",
				Collections:  []CollectionBackup{},
			},
			expectError: true,
		},
		{
			name: "Missing database name",
			backup: &BackupFormat{
				Version:     "1.0",
				Timestamp:   time.Now(),
				Collections: []CollectionBackup{},
			},
			expectError: true,
		},
		{
			name: "Missing collections",
			backup: &BackupFormat{
				Version:      "1.0",
				Timestamp:    time.Now(),
				DatabaseName: "test_db",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := restorer.validateBackup(tt.backup)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestConvertDocumentFromBackup(t *testing.T) {
	tests := []struct {
		name        string
		docBackup   DocumentBackup
		expectError bool
	}{
		{
			name: "Simple document",
			docBackup: DocumentBackup{
				ID: "doc1",
				Fields: map[string]interface{}{
					"name": "Alice",
					"age":  float64(30), // JSON numbers are float64
				},
			},
			expectError: false,
		},
		{
			name: "Document with ObjectID",
			docBackup: DocumentBackup{
				ID: document.NewObjectID().Hex(),
				Fields: map[string]interface{}{
					"name": "Bob",
				},
			},
			expectError: false,
		},
		{
			name: "Document with timestamp",
			docBackup: DocumentBackup{
				ID: "doc3",
				Fields: map[string]interface{}{
					"created": time.Now().Format(time.RFC3339),
				},
			},
			expectError: false,
		},
		{
			name: "Document with nested fields",
			docBackup: DocumentBackup{
				ID: "doc4",
				Fields: map[string]interface{}{
					"metadata": map[string]interface{}{
						"created": time.Now().Format(time.RFC3339),
						"tags":    []interface{}{"tag1", "tag2"},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docMap, err := ConvertDocumentFromBackup(tt.docBackup)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				// Verify _id is present
				if _, exists := docMap["_id"]; !exists {
					t.Errorf("Expected _id field in converted document")
				}
			}
		})
	}
}

func TestConvertValueFromBackup_Numbers(t *testing.T) {
	// Test integer conversion
	result, err := convertValueFromBackup(float64(42))
	if err != nil {
		t.Fatalf("Failed to convert number: %v", err)
	}
	if v, ok := result.(int64); !ok || v != 42 {
		t.Errorf("Expected int64(42), got %T(%v)", result, result)
	}

	// Test float conversion
	result, err = convertValueFromBackup(3.14)
	if err != nil {
		t.Fatalf("Failed to convert float: %v", err)
	}
	if v, ok := result.(float64); !ok || v != 3.14 {
		t.Errorf("Expected float64(3.14), got %T(%v)", result, result)
	}
}

func TestConvertValueFromBackup_ObjectID(t *testing.T) {
	oid := document.NewObjectID()
	hexStr := oid.Hex()

	result, err := convertValueFromBackup(hexStr)
	if err != nil {
		t.Fatalf("Failed to convert ObjectID: %v", err)
	}

	if convertedOID, ok := result.(document.ObjectID); !ok {
		t.Errorf("Expected ObjectID, got %T", result)
	} else if convertedOID.Hex() != hexStr {
		t.Errorf("ObjectID mismatch: expected %s, got %s", hexStr, convertedOID.Hex())
	}
}

func TestConvertValueFromBackup_Time(t *testing.T) {
	now := time.Now()
	timeStr := now.Format(time.RFC3339)

	result, err := convertValueFromBackup(timeStr)
	if err != nil {
		t.Fatalf("Failed to convert time: %v", err)
	}

	if convertedTime, ok := result.(time.Time); !ok {
		t.Errorf("Expected time.Time, got %T", result)
	} else if convertedTime.Unix() != now.Unix() {
		t.Errorf("Time mismatch")
	}
}

func TestConvertValueFromBackup_Array(t *testing.T) {
	arr := []interface{}{
		"string",
		float64(42),
		true,
	}

	result, err := convertValueFromBackup(arr)
	if err != nil {
		t.Fatalf("Failed to convert array: %v", err)
	}

	if convertedArr, ok := result.([]interface{}); !ok {
		t.Errorf("Expected []interface{}, got %T", result)
	} else {
		if len(convertedArr) != 3 {
			t.Errorf("Expected 3 elements, got %d", len(convertedArr))
		}

		// First element should remain string
		if _, ok := convertedArr[0].(string); !ok {
			t.Errorf("Expected first element to be string")
		}

		// Second element should be converted to int64
		if _, ok := convertedArr[1].(int64); !ok {
			t.Errorf("Expected second element to be int64")
		}

		// Third element should remain bool
		if _, ok := convertedArr[2].(bool); !ok {
			t.Errorf("Expected third element to be bool")
		}
	}
}

func TestConvertValueFromBackup_NestedDocument(t *testing.T) {
	nested := map[string]interface{}{
		"name": "Alice",
		"age":  float64(30),
		"metadata": map[string]interface{}{
			"created": time.Now().Format(time.RFC3339),
		},
	}

	result, err := convertValueFromBackup(nested)
	if err != nil {
		t.Fatalf("Failed to convert nested document: %v", err)
	}

	if convertedMap, ok := result.(map[string]interface{}); !ok {
		t.Errorf("Expected map[string]interface{}, got %T", result)
	} else {
		// Check age is converted to int64
		if age, ok := convertedMap["age"].(int64); !ok {
			t.Errorf("Expected age to be int64, got %T", convertedMap["age"])
		} else if age != 30 {
			t.Errorf("Expected age 30, got %d", age)
		}

		// Check nested metadata
		if metadata, ok := convertedMap["metadata"].(map[string]interface{}); !ok {
			t.Errorf("Expected metadata to be map")
		} else {
			// Check nested timestamp is converted to time.Time
			if _, ok := metadata["created"].(time.Time); !ok {
				t.Errorf("Expected created to be time.Time, got %T", metadata["created"])
			}
		}
	}
}

func TestConvertValueFromBackup_GeoJSON(t *testing.T) {
	geoJSON := map[string]interface{}{
		"type":        "Point",
		"coordinates": []interface{}{float64(-122.4), float64(37.8)},
	}

	result, err := convertValueFromBackup(geoJSON)
	if err != nil {
		t.Fatalf("Failed to convert GeoJSON: %v", err)
	}

	// GeoJSON should be preserved as-is
	if resultMap, ok := result.(map[string]interface{}); !ok {
		t.Errorf("Expected map[string]interface{}, got %T", result)
	} else {
		if resultMap["type"] != "Point" {
			t.Errorf("Expected type Point, got %v", resultMap["type"])
		}
	}
}

func TestDefaultRestoreOptions(t *testing.T) {
	opts := DefaultRestoreOptions()

	if opts.DropExisting {
		t.Errorf("Expected DropExisting to be false by default")
	}

	if opts.SkipIndexes {
		t.Errorf("Expected SkipIndexes to be false by default")
	}

	if opts.TargetDatabase != "" {
		t.Errorf("Expected TargetDatabase to be empty by default")
	}
}

func TestValidateBackupFile(t *testing.T) {
	// This test requires a temporary file, so we'll test the validation logic
	restorer := NewRestorer()

	// Test with invalid JSON
	invalidJSON := strings.NewReader("{invalid json")
	_, err := restorer.RestoreFromReader(invalidJSON)
	if err == nil {
		t.Errorf("Expected error for invalid JSON")
	}

	// Test with valid JSON but invalid backup format
	invalidBackup := strings.NewReader(`{"version": "1.0"}`)
	_, err = restorer.RestoreFromReader(invalidBackup)
	if err == nil {
		t.Errorf("Expected error for invalid backup format")
	}
}

func TestRoundTripBackupRestore(t *testing.T) {
	// Create a backup with various data types
	backup := NewBackupFormat("test_db")

	oid := document.NewObjectID()
	now := time.Now()

	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"_id":       oid,
			"name":      "Alice",
			"age":       int64(30),
			"active":    true,
			"score":     95.5,
			"timestamp": now,
			"tags":      []interface{}{"tag1", "tag2"},
			"metadata": map[string]interface{}{
				"created": now,
			},
		}),
	}

	indexes := []IndexBackup{
		{
			Name:       "name_idx",
			Type:       "btree",
			FieldPaths: []string{"name"},
			Unique:     true,
		},
	}

	backup.AddCollection("users", docs, indexes)

	// Backup to JSON
	var buf bytes.Buffer
	backuper := NewBackuper(false)
	err := backuper.BackupToWriter(&buf, backup)
	if err != nil {
		t.Fatalf("Failed to backup: %v", err)
	}

	// Restore from JSON
	restorer := NewRestorer()
	restored, err := restorer.RestoreFromReader(&buf)
	if err != nil {
		t.Fatalf("Failed to restore: %v", err)
	}

	// Verify restored data
	if restored.DatabaseName != "test_db" {
		t.Errorf("Database name mismatch")
	}

	if len(restored.Collections) != 1 {
		t.Fatalf("Expected 1 collection, got %d", len(restored.Collections))
	}

	coll := restored.Collections[0]
	if coll.Name != "users" {
		t.Errorf("Collection name mismatch")
	}

	if len(coll.Documents) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(coll.Documents))
	}

	// Convert restored document back
	docMap, err := ConvertDocumentFromBackup(coll.Documents[0])
	if err != nil {
		t.Fatalf("Failed to convert document: %v", err)
	}

	// Verify fields
	if docMap["name"] != "Alice" {
		t.Errorf("Name mismatch")
	}

	if age, ok := docMap["age"].(int64); !ok || age != 30 {
		t.Errorf("Age mismatch")
	}

	if active, ok := docMap["active"].(bool); !ok || !active {
		t.Errorf("Active mismatch")
	}

	// Verify ObjectID restoration
	if restoredOID, ok := docMap["_id"].(document.ObjectID); !ok {
		t.Errorf("Expected _id to be ObjectID, got %T", docMap["_id"])
	} else if restoredOID.Hex() != oid.Hex() {
		t.Errorf("ObjectID mismatch")
	}

	// Verify indexes
	if len(coll.Indexes) != 1 {
		t.Fatalf("Expected 1 index, got %d", len(coll.Indexes))
	}

	if coll.Indexes[0].Name != "name_idx" {
		t.Errorf("Index name mismatch")
	}
}
