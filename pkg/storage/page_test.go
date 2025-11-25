package storage

import (
	"testing"
)

func TestPageSerializeDeserialize(t *testing.T) {
	original := NewPage(5, PageTypeData)
	copy(original.Data, []byte("test page data"))
	original.IsDirty = true
	original.LSN = 42

	// Serialize
	data := original.Serialize()

	// Deserialize
	deserialized := NewPage(0, PageTypeData)
	err := deserialized.Deserialize(data)
	if err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	// Verify fields
	if deserialized.ID != original.ID {
		t.Errorf("ID mismatch: expected %d, got %d", original.ID, deserialized.ID)
	}
	if deserialized.Type != original.Type {
		t.Errorf("Type mismatch: expected %d, got %d", original.Type, deserialized.Type)
	}
	if deserialized.LSN != original.LSN {
		t.Errorf("LSN mismatch: expected %d, got %d", original.LSN, deserialized.LSN)
	}

	deserializedData := deserialized.Data[:len("test page data")]
	if string(deserializedData) != "test page data" {
		t.Errorf("Data mismatch: expected 'test page data', got '%s'", string(deserializedData))
	}
}

func TestPageDeserializeError(t *testing.T) {
	page := NewPage(0, PageTypeData)

	// Test with too short data
	shortData := make([]byte, 10)
	err := page.Deserialize(shortData)
	if err == nil {
		t.Error("Expected error when deserializing too short data")
	}
}

func TestPageIsPinned(t *testing.T) {
	page := NewPage(0, PageTypeData)

	// Initially not pinned
	if page.IsPinned() {
		t.Error("Expected page to not be pinned initially")
	}

	// Pin the page
	page.Pin()
	if !page.IsPinned() {
		t.Error("Expected page to be pinned after Pin()")
	}
	if page.PinCount != 1 {
		t.Errorf("Expected pin count 1, got %d", page.PinCount)
	}

	// Pin again
	page.Pin()
	if page.PinCount != 2 {
		t.Errorf("Expected pin count 2, got %d", page.PinCount)
	}

	// Unpin
	page.Unpin()
	if page.PinCount != 1 {
		t.Errorf("Expected pin count 1 after unpin, got %d", page.PinCount)
	}
	if !page.IsPinned() {
		t.Error("Expected page to still be pinned")
	}

	// Unpin again
	page.Unpin()
	if page.IsPinned() {
		t.Error("Expected page to not be pinned")
	}
	if page.PinCount != 0 {
		t.Errorf("Expected pin count 0, got %d", page.PinCount)
	}
}

func TestPageMarkDirty(t *testing.T) {
	page := NewPage(0, PageTypeData)

	// Initially not dirty
	if page.IsDirty {
		t.Error("Expected page to not be dirty initially")
	}

	// Mark as dirty
	page.MarkDirty()
	if !page.IsDirty {
		t.Error("Expected page to be dirty after MarkDirty()")
	}
}

func TestPageTypes(t *testing.T) {
	types := []PageType{
		PageTypeData,
		PageTypeIndex,
		PageTypeFreeList,
		PageTypeOverflow,
	}

	for _, pageType := range types {
		page := NewPage(0, pageType)
		if page.Type != pageType {
			t.Errorf("Expected page type %d, got %d", pageType, page.Type)
		}
	}
}

func TestPageDataSize(t *testing.T) {
	page := NewPage(0, PageTypeData)

	// Page.Data should be sized appropriately (PageSize minus header overhead)
	if len(page.Data) == 0 {
		t.Error("Expected non-zero page data size")
	}
	if len(page.Data) > PageSize {
		t.Errorf("Page data size %d exceeds PageSize %d", len(page.Data), PageSize)
	}
}

func TestPageSerializeIndexType(t *testing.T) {
	page := NewPage(10, PageTypeIndex)
	copy(page.Data, []byte("index data"))
	page.LSN = 100

	data := page.Serialize()

	deserialized := NewPage(0, PageTypeData)
	err := deserialized.Deserialize(data)
	if err != nil {
		t.Fatalf("Failed to deserialize index page: %v", err)
	}

	if deserialized.Type != PageTypeIndex {
		t.Errorf("Expected index type, got %d", deserialized.Type)
	}
}
