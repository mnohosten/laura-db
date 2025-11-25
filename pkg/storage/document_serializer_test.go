package storage

import (
	"testing"

	"github.com/mnohosten/laura-db/pkg/document"
)

func TestDocumentSerializer_SerializeDeserialize(t *testing.T) {
	serializer := NewDocumentSerializer()

	// Create test document
	doc := document.NewDocument()
	doc.Set("_id", document.NewObjectID())
	doc.Set("name", "Alice")
	doc.Set("age", int64(30))
	doc.Set("active", true)
	doc.Set("score", 95.5)

	// Serialize
	data, err := serializer.SerializeDocument(doc)
	if err != nil {
		t.Fatalf("SerializeDocument failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Serialized data is empty")
	}

	// Deserialize
	deserializedDoc, err := serializer.DeserializeDocument(data)
	if err != nil {
		t.Fatalf("DeserializeDocument failed: %v", err)
	}

	// Verify fields
	name, exists := deserializedDoc.Get("name")
	if !exists || name.(string) != "Alice" {
		t.Error("name field not correctly deserialized")
	}

	age, exists := deserializedDoc.Get("age")
	if !exists || age.(int64) != 30 {
		t.Error("age field not correctly deserialized")
	}

	active, exists := deserializedDoc.Get("active")
	if !exists || active.(bool) != true {
		t.Error("active field not correctly deserialized")
	}

	score, exists := deserializedDoc.Get("score")
	if !exists || score.(float64) != 95.5 {
		t.Error("score field not correctly deserialized")
	}
}

func TestDocumentSerializer_SerializeNilDocument(t *testing.T) {
	serializer := NewDocumentSerializer()

	_, err := serializer.SerializeDocument(nil)
	if err == nil {
		t.Error("Expected error when serializing nil document")
	}
}

func TestDocumentSerializer_DeserializeEmptyData(t *testing.T) {
	serializer := NewDocumentSerializer()

	_, err := serializer.DeserializeDocument([]byte{})
	if err == nil {
		t.Error("Expected error when deserializing empty data")
	}
}

func TestDocumentSerializer_SerializeFromMap(t *testing.T) {
	serializer := NewDocumentSerializer()

	m := map[string]interface{}{
		"name":   "Bob",
		"age":    int64(25),
		"active": true,
	}

	data, err := serializer.SerializeDocumentFromMap(m)
	if err != nil {
		t.Fatalf("SerializeDocumentFromMap failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Serialized data is empty")
	}

	// Deserialize and verify
	resultMap, err := serializer.DeserializeDocumentToMap(data)
	if err != nil {
		t.Fatalf("DeserializeDocumentToMap failed: %v", err)
	}

	if resultMap["name"].(string) != "Bob" {
		t.Error("name not correctly serialized/deserialized")
	}

	if resultMap["age"].(int64) != 25 {
		t.Error("age not correctly serialized/deserialized")
	}
}

func TestDocumentSerializer_SerializeNilMap(t *testing.T) {
	serializer := NewDocumentSerializer()

	_, err := serializer.SerializeDocumentFromMap(nil)
	if err == nil {
		t.Error("Expected error when serializing nil map")
	}
}

func TestDocumentSerializer_SerializeNestedDocument(t *testing.T) {
	serializer := NewDocumentSerializer()

	// Create nested document
	doc := document.NewDocument()
	doc.Set("name", "Alice")

	address := document.NewDocument()
	address.Set("city", "New York")
	address.Set("zip", "10001")
	doc.Set("address", address)

	// Serialize and deserialize
	data, err := serializer.SerializeDocument(doc)
	if err != nil {
		t.Fatalf("SerializeDocument failed: %v", err)
	}

	deserializedDoc, err := serializer.DeserializeDocument(data)
	if err != nil {
		t.Fatalf("DeserializeDocument failed: %v", err)
	}

	// Verify nested document
	addr, exists := deserializedDoc.Get("address")
	if !exists {
		t.Fatal("address field not found")
	}

	addrDoc := addr.(*document.Document)
	city, exists := addrDoc.Get("city")
	if !exists || city.(string) != "New York" {
		t.Error("Nested document not correctly serialized/deserialized")
	}
}

func TestDocumentSerializer_SerializeArrays(t *testing.T) {
	serializer := NewDocumentSerializer()

	doc := document.NewDocument()
	doc.Set("tags", []interface{}{"admin", "user", "developer"})
	doc.Set("scores", []interface{}{int64(90), int64(85), int64(95)})

	// Serialize and deserialize
	data, err := serializer.SerializeDocument(doc)
	if err != nil {
		t.Fatalf("SerializeDocument failed: %v", err)
	}

	deserializedDoc, err := serializer.DeserializeDocument(data)
	if err != nil {
		t.Fatalf("DeserializeDocument failed: %v", err)
	}

	// Verify arrays
	tags, exists := deserializedDoc.Get("tags")
	if !exists {
		t.Fatal("tags field not found")
	}

	tagsArr := tags.([]interface{})
	if len(tagsArr) != 3 || tagsArr[0].(string) != "admin" {
		t.Error("tags array not correctly serialized/deserialized")
	}
}

func TestDocumentSerializer_EstimateDocumentSize(t *testing.T) {
	serializer := NewDocumentSerializer()

	doc := document.NewDocument()
	doc.Set("name", "Alice")
	doc.Set("age", int64(30))

	size := serializer.EstimateDocumentSize(doc)
	if size == 0 {
		t.Error("Estimated size should not be zero")
	}

	// Actual serialization should produce same size
	data, err := serializer.SerializeDocument(doc)
	if err != nil {
		t.Fatalf("SerializeDocument failed: %v", err)
	}

	if size != len(data) {
		t.Errorf("Estimated size %d does not match actual size %d", size, len(data))
	}
}

func TestDocumentSerializer_CanFitInSinglePage(t *testing.T) {
	serializer := NewDocumentSerializer()

	// Small document should fit
	smallDoc := document.NewDocument()
	smallDoc.Set("name", "Alice")
	smallDoc.Set("age", int64(30))

	canFit, err := serializer.CanFitInSinglePage(smallDoc)
	if err != nil {
		t.Fatalf("CanFitInSinglePage failed: %v", err)
	}

	if !canFit {
		t.Error("Small document should fit in a single page")
	}

	// Large document should not fit
	largeDoc := document.NewDocument()
	largeData := make([]byte, MaxSinglePageDocumentSize+1000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	largeDoc.Set("data", largeData)

	canFit, err = serializer.CanFitInSinglePage(largeDoc)
	if err != nil {
		t.Fatalf("CanFitInSinglePage failed: %v", err)
	}

	if canFit {
		t.Error("Large document should not fit in a single page")
	}
}

// TestDocumentPageManager_InsertAndGet tests inserting and retrieving documents from a page
func TestDocumentPageManager_InsertAndGet(t *testing.T) {
	// Create a new page
	page := NewPage(1, PageTypeData)

	// Create slotted page
	slottedPage, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	// Create document page manager
	dpm := NewDocumentPageManager()

	// Create test document
	doc := document.NewDocument()
	doc.Set("_id", document.NewObjectID())
	doc.Set("name", "Alice")
	doc.Set("age", int64(30))
	doc.Set("email", "alice@example.com")

	// Insert document
	slotID, err := dpm.InsertDocument(slottedPage, doc)
	if err != nil {
		t.Fatalf("InsertDocument failed: %v", err)
	}

	if slotID != 0 {
		t.Errorf("Expected first slot to have ID 0, got %d", slotID)
	}

	// Retrieve document
	retrievedDoc, err := dpm.GetDocument(slottedPage, slotID)
	if err != nil {
		t.Fatalf("GetDocument failed: %v", err)
	}

	// Verify document fields
	name, exists := retrievedDoc.Get("name")
	if !exists || name.(string) != "Alice" {
		t.Error("name field not correctly retrieved")
	}

	age, exists := retrievedDoc.Get("age")
	if !exists || age.(int64) != 30 {
		t.Error("age field not correctly retrieved")
	}

	email, exists := retrievedDoc.Get("email")
	if !exists || email.(string) != "alice@example.com" {
		t.Error("email field not correctly retrieved")
	}
}

// TestDocumentPageManager_InsertMultipleDocuments tests inserting multiple documents
func TestDocumentPageManager_InsertMultipleDocuments(t *testing.T) {
	page := NewPage(1, PageTypeData)

	slottedPage, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	dpm := NewDocumentPageManager()

	// Insert multiple documents
	numDocs := 10
	slotIDs := make([]uint16, numDocs)

	for i := 0; i < numDocs; i++ {
		doc := document.NewDocument()
		doc.Set("_id", document.NewObjectID())
		doc.Set("index", int64(i))
		doc.Set("name", "User"+string(rune('A'+i)))

		slotID, err := dpm.InsertDocument(slottedPage, doc)
		if err != nil {
			t.Fatalf("Failed to insert document %d: %v", i, err)
		}
		slotIDs[i] = slotID
	}

	// Verify all documents
	for i := 0; i < numDocs; i++ {
		doc, err := dpm.GetDocument(slottedPage, slotIDs[i])
		if err != nil {
			t.Fatalf("Failed to get document %d: %v", i, err)
		}

		index, exists := doc.Get("index")
		if !exists || index.(int64) != int64(i) {
			t.Errorf("Document %d has incorrect index", i)
		}
	}
}

// TestDocumentPageManager_UpdateDocument tests updating a document
func TestDocumentPageManager_UpdateDocument(t *testing.T) {
	page := NewPage(1, PageTypeData)

	slottedPage, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	dpm := NewDocumentPageManager()

	// Insert document
	doc := document.NewDocument()
	doc.Set("_id", document.NewObjectID())
	doc.Set("name", "Alice")
	doc.Set("age", int64(30))

	slotID, err := dpm.InsertDocument(slottedPage, doc)
	if err != nil {
		t.Fatalf("InsertDocument failed: %v", err)
	}

	// Update document (smaller size)
	updatedDoc := document.NewDocument()
	updatedDoc.Set("_id", document.NewObjectID())
	updatedDoc.Set("name", "Bob")

	err = dpm.UpdateDocument(slottedPage, slotID, updatedDoc)
	if err != nil {
		t.Fatalf("UpdateDocument failed: %v", err)
	}

	// Verify update
	retrievedDoc, err := dpm.GetDocument(slottedPage, slotID)
	if err != nil {
		t.Fatalf("GetDocument failed: %v", err)
	}

	name, exists := retrievedDoc.Get("name")
	if !exists || name.(string) != "Bob" {
		t.Error("Document not correctly updated")
	}

	// Age should not exist in updated document
	_, exists = retrievedDoc.Get("age")
	if exists {
		t.Error("Old field still exists after update")
	}
}

// TestDocumentPageManager_DeleteDocument tests deleting a document
func TestDocumentPageManager_DeleteDocument(t *testing.T) {
	page := NewPage(1, PageTypeData)

	slottedPage, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	dpm := NewDocumentPageManager()

	// Insert document
	doc := document.NewDocument()
	doc.Set("name", "Alice")
	doc.Set("age", int64(30))

	slotID, err := dpm.InsertDocument(slottedPage, doc)
	if err != nil {
		t.Fatalf("InsertDocument failed: %v", err)
	}

	// Delete document
	err = dpm.DeleteDocument(slottedPage, slotID)
	if err != nil {
		t.Fatalf("DeleteDocument failed: %v", err)
	}

	// Try to retrieve deleted document
	_, err = dpm.GetDocument(slottedPage, slotID)
	if err == nil {
		t.Error("Expected error when getting deleted document")
	}
}

// TestDocumentPageManager_InsertFromMap tests inserting from map
func TestDocumentPageManager_InsertFromMap(t *testing.T) {
	page := NewPage(1, PageTypeData)

	slottedPage, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	dpm := NewDocumentPageManager()

	// Insert from map
	m := map[string]interface{}{
		"name":  "Charlie",
		"age":   int64(35),
		"email": "charlie@example.com",
	}

	slotID, err := dpm.InsertDocumentFromMap(slottedPage, m)
	if err != nil {
		t.Fatalf("InsertDocumentFromMap failed: %v", err)
	}

	// Retrieve as map
	resultMap, err := dpm.GetDocumentAsMap(slottedPage, slotID)
	if err != nil {
		t.Fatalf("GetDocumentAsMap failed: %v", err)
	}

	if resultMap["name"].(string) != "Charlie" {
		t.Error("name not correctly stored")
	}

	if resultMap["age"].(int64) != 35 {
		t.Error("age not correctly stored")
	}
}

// TestDocumentPageManager_GetPageCapacity tests page capacity analysis
func TestDocumentPageManager_GetPageCapacity(t *testing.T) {
	page := NewPage(1, PageTypeData)

	slottedPage, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	dpm := NewDocumentPageManager()

	// Check initial capacity
	capacity := dpm.GetPageCapacity(slottedPage)
	if capacity.TotalSpace == 0 {
		t.Error("Initial capacity should not be zero")
	}

	if capacity.ActiveSlotCount != 0 {
		t.Error("Initial active slot count should be zero")
	}

	// Insert some documents
	for i := 0; i < 5; i++ {
		doc := document.NewDocument()
		doc.Set("index", int64(i))
		doc.Set("data", "Some data here")

		_, err := dpm.InsertDocument(slottedPage, doc)
		if err != nil {
			t.Fatalf("Failed to insert document: %v", err)
		}
	}

	// Check capacity after inserts
	capacity = dpm.GetPageCapacity(slottedPage)
	if capacity.ActiveSlotCount != 5 {
		t.Errorf("Expected 5 active slots, got %d", capacity.ActiveSlotCount)
	}

	if capacity.TotalSpace >= int(SlottedPageAvailableSpace) {
		t.Error("Total space should decrease after inserts")
	}
}

// TestDocumentPageManager_NilPage tests error handling for nil page
func TestDocumentPageManager_NilPage(t *testing.T) {
	dpm := NewDocumentPageManager()
	doc := document.NewDocument()
	doc.Set("name", "Alice")

	// Insert with nil page
	_, err := dpm.InsertDocument(nil, doc)
	if err == nil {
		t.Error("Expected error when inserting into nil page")
	}

	// Get with nil page
	_, err = dpm.GetDocument(nil, 0)
	if err == nil {
		t.Error("Expected error when getting from nil page")
	}

	// Update with nil page
	err = dpm.UpdateDocument(nil, 0, doc)
	if err == nil {
		t.Error("Expected error when updating in nil page")
	}

	// Delete with nil page
	err = dpm.DeleteDocument(nil, 0)
	if err == nil {
		t.Error("Expected error when deleting from nil page")
	}
}

// TestDocumentPageManager_NilDocument tests error handling for nil document
func TestDocumentPageManager_NilDocument(t *testing.T) {
	page := NewPage(1, PageTypeData)

	slottedPage, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	dpm := NewDocumentPageManager()

	// Insert nil document
	_, err = dpm.InsertDocument(slottedPage, nil)
	if err == nil {
		t.Error("Expected error when inserting nil document")
	}

	// Update with nil document
	err = dpm.UpdateDocument(slottedPage, 0, nil)
	if err == nil {
		t.Error("Expected error when updating with nil document")
	}
}

// TestDocumentSerializer_EstimateDocumentSizeEdgeCases tests edge cases for size estimation
func TestDocumentSerializer_EstimateDocumentSizeEdgeCases(t *testing.T) {
	ser := NewDocumentSerializer()

	// Empty document
	emptyDoc := document.NewDocument()
	size := ser.EstimateDocumentSize(emptyDoc)
	if size <= 0 {
		t.Error("Expected positive size for empty document")
	}

	// Document with various field types
	doc := document.NewDocument()
	doc.Set("string", "test")
	doc.Set("number", int64(42))
	doc.Set("bool", true)
	doc.Set("null", nil)

	size = ser.EstimateDocumentSize(doc)
	if size <= 0 {
		t.Error("Expected positive size for document with various types")
	}

	// Document with array
	doc2 := document.NewDocument()
	doc2.Set("array", []interface{}{int64(1), int64(2), int64(3)})
	size2 := ser.EstimateDocumentSize(doc2)
	if size2 <= 0 {
		t.Error("Expected positive size for document with array")
	}
}

// TestDocumentSerializer_GetDocumentAsMapError tests error handling
func TestDocumentSerializer_GetDocumentAsMapError(t *testing.T) {
	page := NewPage(1, PageTypeData)
	slottedPage, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	dpm := NewDocumentPageManager()

	// Try to get document with invalid slot ID
	_, err = dpm.GetDocumentAsMap(slottedPage, 999)
	if err == nil {
		t.Error("Expected error when getting document with invalid slot ID")
	}
}

// TestDocumentPageManager_UpdateDocumentSizeChange tests updating with different sizes
func TestDocumentPageManager_UpdateDocumentSizeChange(t *testing.T) {
	page := NewPage(1, PageTypeData)
	slottedPage, err := NewSlottedPage(page)
	if err != nil {
		t.Fatalf("Failed to create slotted page: %v", err)
	}

	dpm := NewDocumentPageManager()

	// Insert initial document
	doc := document.NewDocument()
	doc.Set("field", "small")
	slotID, err := dpm.InsertDocument(slottedPage, doc)
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Update with larger document
	doc2 := document.NewDocument()
	doc2.Set("field", "much larger content with more data")
	err = dpm.UpdateDocument(slottedPage, slotID, doc2)
	if err != nil {
		t.Fatalf("Failed to update document: %v", err)
	}

	// Verify updated content
	retrieved, err := dpm.GetDocument(slottedPage, slotID)
	if err != nil {
		t.Fatalf("Failed to get updated document: %v", err)
	}

	if val, _ := retrieved.Get("field"); val != "much larger content with more data" {
		t.Error("Document was not updated correctly")
	}
}
