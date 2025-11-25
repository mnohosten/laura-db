package index

import (
	"testing"

	"github.com/mnohosten/laura-db/pkg/document"
	"github.com/mnohosten/laura-db/pkg/storage"
)

func TestSerializeDeserializeLeafNode(t *testing.T) {
	// Create a leaf node with various key types
	node := &BTreeNode{
		isLeaf: true,
		keys: []interface{}{
			int64(100),
			int64(200),
			int64(300),
		},
		values: []interface{}{
			make([]byte, 12), // Placeholder DocumentID
			make([]byte, 12),
			make([]byte, 12),
		},
	}

	// Create a page
	page := storage.NewPage(storage.PageID(1), storage.PageTypeData)

	// Serialize node
	err := SerializeBTreeNode(node, page, 1, 1)
	if err != nil {
		t.Fatalf("Failed to serialize node: %v", err)
	}

	// Deserialize node
	deserializedNode, err := DeserializeBTreeNode(page)
	if err != nil {
		t.Fatalf("Failed to deserialize node: %v", err)
	}

	// Verify node properties
	if deserializedNode.isLeaf != node.isLeaf {
		t.Errorf("isLeaf mismatch: got %v, want %v", deserializedNode.isLeaf, node.isLeaf)
	}

	if len(deserializedNode.keys) != len(node.keys) {
		t.Errorf("key count mismatch: got %d, want %d", len(deserializedNode.keys), len(node.keys))
	}

	// Verify keys
	for i, key := range node.keys {
		if deserializedNode.keys[i] != key {
			t.Errorf("key %d mismatch: got %v, want %v", i, deserializedNode.keys[i], key)
		}
	}

	if len(deserializedNode.values) != len(node.values) {
		t.Errorf("value count mismatch: got %d, want %d", len(deserializedNode.values), len(node.values))
	}
}

func TestSerializeDeserializeInternalNode(t *testing.T) {
	// Create an internal node
	node := &BTreeNode{
		isLeaf: false,
		keys: []interface{}{
			int64(50),
			int64(150),
		},
		children: []*BTreeNode{
			{}, // Placeholder children
			{},
			{},
		},
	}

	// Create a page
	page := storage.NewPage(storage.PageID(1), storage.PageTypeData)

	// Serialize node
	err := SerializeBTreeNode(node, page, 1, 1)
	if err != nil {
		t.Fatalf("Failed to serialize node: %v", err)
	}

	// Deserialize node
	deserializedNode, err := DeserializeBTreeNode(page)
	if err != nil {
		t.Fatalf("Failed to deserialize node: %v", err)
	}

	// Verify node properties
	if deserializedNode.isLeaf != node.isLeaf {
		t.Errorf("isLeaf mismatch: got %v, want %v", deserializedNode.isLeaf, node.isLeaf)
	}

	if len(deserializedNode.keys) != len(node.keys) {
		t.Errorf("key count mismatch: got %d, want %d", len(deserializedNode.keys), len(node.keys))
	}

	// Verify keys
	for i, key := range node.keys {
		if deserializedNode.keys[i] != key {
			t.Errorf("key %d mismatch: got %v, want %v", i, deserializedNode.keys[i], key)
		}
	}
}

func TestSerializeDeserializeStringKeys(t *testing.T) {
	// Create a leaf node with string keys
	node := &BTreeNode{
		isLeaf: true,
		keys: []interface{}{
			"alice",
			"bob",
			"charlie",
		},
		values: []interface{}{
			make([]byte, 12),
			make([]byte, 12),
			make([]byte, 12),
		},
	}

	// Create a page
	page := storage.NewPage(storage.PageID(1), storage.PageTypeData)

	// Serialize node
	err := SerializeBTreeNode(node, page, 1, 1)
	if err != nil {
		t.Fatalf("Failed to serialize node: %v", err)
	}

	// Deserialize node
	deserializedNode, err := DeserializeBTreeNode(page)
	if err != nil {
		t.Fatalf("Failed to deserialize node: %v", err)
	}

	// Verify keys
	if len(deserializedNode.keys) != len(node.keys) {
		t.Fatalf("key count mismatch: got %d, want %d", len(deserializedNode.keys), len(node.keys))
	}

	for i, key := range node.keys {
		if deserializedNode.keys[i] != key {
			t.Errorf("key %d mismatch: got %v, want %v", i, deserializedNode.keys[i], key)
		}
	}
}

func TestSerializeDeserializeFloat64Keys(t *testing.T) {
	// Create a leaf node with float64 keys
	node := &BTreeNode{
		isLeaf: true,
		keys: []interface{}{
			float64(1.5),
			float64(2.7),
			float64(3.9),
		},
		values: []interface{}{
			make([]byte, 12),
			make([]byte, 12),
			make([]byte, 12),
		},
	}

	// Create a page
	page := storage.NewPage(storage.PageID(1), storage.PageTypeData)

	// Serialize node
	err := SerializeBTreeNode(node, page, 1, 1)
	if err != nil {
		t.Fatalf("Failed to serialize node: %v", err)
	}

	// Deserialize node
	deserializedNode, err := DeserializeBTreeNode(page)
	if err != nil {
		t.Fatalf("Failed to deserialize node: %v", err)
	}

	// Verify keys
	if len(deserializedNode.keys) != len(node.keys) {
		t.Fatalf("key count mismatch: got %d, want %d", len(deserializedNode.keys), len(node.keys))
	}

	for i, key := range node.keys {
		if deserializedNode.keys[i] != key {
			t.Errorf("key %d mismatch: got %v, want %v", i, deserializedNode.keys[i], key)
		}
	}
}

func TestSerializeDeserializeObjectIDKeys(t *testing.T) {
	// Create ObjectIDs
	oid1 := document.NewObjectID()
	oid2 := document.NewObjectID()
	oid3 := document.NewObjectID()

	// Create a leaf node with ObjectID keys
	node := &BTreeNode{
		isLeaf: true,
		keys: []interface{}{
			oid1,
			oid2,
			oid3,
		},
		values: []interface{}{
			make([]byte, 12),
			make([]byte, 12),
			make([]byte, 12),
		},
	}

	// Create a page
	page := storage.NewPage(storage.PageID(1), storage.PageTypeData)

	// Serialize node
	err := SerializeBTreeNode(node, page, 1, 1)
	if err != nil {
		t.Fatalf("Failed to serialize node: %v", err)
	}

	// Deserialize node
	deserializedNode, err := DeserializeBTreeNode(page)
	if err != nil {
		t.Fatalf("Failed to deserialize node: %v", err)
	}

	// Verify keys
	if len(deserializedNode.keys) != len(node.keys) {
		t.Fatalf("key count mismatch: got %d, want %d", len(deserializedNode.keys), len(node.keys))
	}

	for i, key := range node.keys {
		expectedOID := key.(document.ObjectID)
		actualOID := deserializedNode.keys[i].(document.ObjectID)
		if expectedOID != actualOID {
			t.Errorf("key %d mismatch: got %v, want %v", i, actualOID, expectedOID)
		}
	}
}

func TestSerializeDeserializeMixedKeys(t *testing.T) {
	// Create a leaf node with mixed key types (not recommended in practice, but testing serialization)
	// Note: In real usage, all keys in a node should be the same type
	oid := document.NewObjectID()

	node := &BTreeNode{
		isLeaf: true,
		keys: []interface{}{
			int64(100),
			"test",
			float64(3.14),
			oid,
		},
		values: []interface{}{
			make([]byte, 12),
			make([]byte, 12),
			make([]byte, 12),
			make([]byte, 12),
		},
	}

	// Create a page
	page := storage.NewPage(storage.PageID(1), storage.PageTypeData)

	// Serialize node
	err := SerializeBTreeNode(node, page, 1, 1)
	if err != nil {
		t.Fatalf("Failed to serialize node: %v", err)
	}

	// Deserialize node
	deserializedNode, err := DeserializeBTreeNode(page)
	if err != nil {
		t.Fatalf("Failed to deserialize node: %v", err)
	}

	// Verify keys
	if len(deserializedNode.keys) != len(node.keys) {
		t.Fatalf("key count mismatch: got %d, want %d", len(deserializedNode.keys), len(node.keys))
	}

	// Verify each key individually with type assertion
	if deserializedNode.keys[0] != int64(100) {
		t.Errorf("key 0 mismatch: got %v, want %v", deserializedNode.keys[0], int64(100))
	}
	if deserializedNode.keys[1] != "test" {
		t.Errorf("key 1 mismatch: got %v, want %v", deserializedNode.keys[1], "test")
	}
	if deserializedNode.keys[2] != float64(3.14) {
		t.Errorf("key 2 mismatch: got %v, want %v", deserializedNode.keys[2], float64(3.14))
	}
	actualOID := deserializedNode.keys[3].(document.ObjectID)
	if actualOID != oid {
		t.Errorf("key 3 mismatch: got %v, want %v", actualOID, oid)
	}
}

func TestSerializeDeserializeEmptyNode(t *testing.T) {
	// Create an empty leaf node
	node := &BTreeNode{
		isLeaf: true,
		keys:   []interface{}{},
		values: []interface{}{},
	}

	// Create a page
	page := storage.NewPage(storage.PageID(1), storage.PageTypeData)

	// Serialize node
	err := SerializeBTreeNode(node, page, 1, 1)
	if err != nil {
		t.Fatalf("Failed to serialize node: %v", err)
	}

	// Deserialize node
	deserializedNode, err := DeserializeBTreeNode(page)
	if err != nil {
		t.Fatalf("Failed to deserialize node: %v", err)
	}

	// Verify node properties
	if deserializedNode.isLeaf != node.isLeaf {
		t.Errorf("isLeaf mismatch: got %v, want %v", deserializedNode.isLeaf, node.isLeaf)
	}

	if len(deserializedNode.keys) != 0 {
		t.Errorf("expected empty keys, got %d keys", len(deserializedNode.keys))
	}

	if len(deserializedNode.values) != 0 {
		t.Errorf("expected empty values, got %d values", len(deserializedNode.values))
	}
}

func TestSerializeDeserializeLargeNode(t *testing.T) {
	// Create a node with many keys (up to order-1 keys)
	keys := make([]interface{}, 30)
	values := make([]interface{}, 30)
	for i := 0; i < 30; i++ {
		keys[i] = int64(i * 100)
		values[i] = make([]byte, 12)
	}

	node := &BTreeNode{
		isLeaf: true,
		keys:   keys,
		values: values,
	}

	// Create a page
	page := storage.NewPage(storage.PageID(1), storage.PageTypeData)

	// Serialize node
	err := SerializeBTreeNode(node, page, 1, 1)
	if err != nil {
		t.Fatalf("Failed to serialize node: %v", err)
	}

	// Deserialize node
	deserializedNode, err := DeserializeBTreeNode(page)
	if err != nil {
		t.Fatalf("Failed to deserialize node: %v", err)
	}

	// Verify all keys
	if len(deserializedNode.keys) != len(node.keys) {
		t.Fatalf("key count mismatch: got %d, want %d", len(deserializedNode.keys), len(node.keys))
	}

	for i := 0; i < 30; i++ {
		if deserializedNode.keys[i] != keys[i] {
			t.Errorf("key %d mismatch: got %v, want %v", i, deserializedNode.keys[i], keys[i])
		}
	}
}

func TestSerializeDeserializeLongStrings(t *testing.T) {
	// Create a node with long string keys
	node := &BTreeNode{
		isLeaf: true,
		keys: []interface{}{
			"this_is_a_very_long_string_key_that_tests_variable_length_encoding_123456789",
			"another_long_string_with_special_chars_!@#$%^&*()",
			"short",
		},
		values: []interface{}{
			make([]byte, 12),
			make([]byte, 12),
			make([]byte, 12),
		},
	}

	// Create a page
	page := storage.NewPage(storage.PageID(1), storage.PageTypeData)

	// Serialize node
	err := SerializeBTreeNode(node, page, 1, 1)
	if err != nil {
		t.Fatalf("Failed to serialize node: %v", err)
	}

	// Deserialize node
	deserializedNode, err := DeserializeBTreeNode(page)
	if err != nil {
		t.Fatalf("Failed to deserialize node: %v", err)
	}

	// Verify keys
	if len(deserializedNode.keys) != len(node.keys) {
		t.Fatalf("key count mismatch: got %d, want %d", len(deserializedNode.keys), len(node.keys))
	}

	for i, key := range node.keys {
		if deserializedNode.keys[i] != key {
			t.Errorf("key %d mismatch: got %v, want %v", i, deserializedNode.keys[i], key)
		}
	}
}

func TestSerializeNilNode(t *testing.T) {
	// Create a page
	page := storage.NewPage(storage.PageID(1), storage.PageTypeData)

	// Try to serialize nil node
	err := SerializeBTreeNode(nil, page, 1, 1)
	if err == nil {
		t.Error("Expected error when serializing nil node")
	}
}
