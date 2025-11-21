package document

import (
	"fmt"
)

// Document represents a BSON-like document (key-value pairs)
type Document struct {
	fields map[string]*Value
	order  []string // Maintain insertion order
}

// NewDocument creates a new empty document
func NewDocument() *Document {
	return &Document{
		fields: make(map[string]*Value),
		order:  make([]string, 0),
	}
}

// NewDocumentFromMap creates a document from a map
func NewDocumentFromMap(m map[string]interface{}) *Document {
	doc := NewDocument()
	for k, v := range m {
		doc.Set(k, v)
	}
	return doc
}

// Set sets a field value in the document
func (d *Document) Set(key string, value interface{}) {
	if _, exists := d.fields[key]; !exists {
		d.order = append(d.order, key)
	}
	d.fields[key] = NewValue(value)
}

// Get retrieves a field value from the document
func (d *Document) Get(key string) (interface{}, bool) {
	if v, ok := d.fields[key]; ok {
		return v.Data, true
	}
	return nil, false
}

// GetValue retrieves a typed value from the document
func (d *Document) GetValue(key string) (*Value, bool) {
	v, ok := d.fields[key]
	return v, ok
}

// Has checks if a field exists in the document
func (d *Document) Has(key string) bool {
	_, ok := d.fields[key]
	return ok
}

// Delete removes a field from the document
func (d *Document) Delete(key string) {
	if _, ok := d.fields[key]; !ok {
		return
	}

	delete(d.fields, key)

	// Remove from order
	for i, k := range d.order {
		if k == key {
			d.order = append(d.order[:i], d.order[i+1:]...)
			break
		}
	}
}

// Keys returns all field names in insertion order
func (d *Document) Keys() []string {
	return d.order
}

// Len returns the number of fields in the document
func (d *Document) Len() int {
	return len(d.fields)
}

// ToMap converts the document to a map[string]interface{}
func (d *Document) ToMap() map[string]interface{} {
	m := make(map[string]interface{}, len(d.fields))
	for k, v := range d.fields {
		m[k] = d.valueToInterface(v)
	}
	return m
}

// valueToInterface converts a Value to interface{} recursively
func (d *Document) valueToInterface(v *Value) interface{} {
	switch v.Type {
	case TypeDocument:
		if doc, ok := v.Data.(*Document); ok {
			return doc.ToMap()
		}
		if m, ok := v.Data.(map[string]interface{}); ok {
			return m
		}
	case TypeArray:
		if arr, ok := v.Data.([]interface{}); ok {
			result := make([]interface{}, len(arr))
			for i, item := range arr {
				if val, ok := item.(*Value); ok {
					result[i] = d.valueToInterface(val)
				} else {
					result[i] = item
				}
			}
			return result
		}
	}
	return v.Data
}

// Clone creates a deep copy of the document
func (d *Document) Clone() *Document {
	clone := NewDocument()
	for _, key := range d.order {
		if v, ok := d.fields[key]; ok {
			clone.Set(key, d.cloneValue(v))
		}
	}
	return clone
}

// cloneValue creates a deep copy of a value
func (d *Document) cloneValue(v *Value) interface{} {
	switch v.Type {
	case TypeDocument:
		if doc, ok := v.Data.(*Document); ok {
			return doc.Clone()
		}
		if m, ok := v.Data.(map[string]interface{}); ok {
			clone := make(map[string]interface{})
			for k, val := range m {
				clone[k] = val
			}
			return clone
		}
	case TypeArray:
		if arr, ok := v.Data.([]interface{}); ok {
			clone := make([]interface{}, len(arr))
			copy(clone, arr)
			return clone
		}
	case TypeBinary:
		if b, ok := v.Data.([]byte); ok {
			clone := make([]byte, len(b))
			copy(clone, b)
			return clone
		}
	}
	return v.Data
}

// GetNested retrieves a value using dot notation (e.g., "user.address.city")
func (d *Document) GetNested(path string) (interface{}, bool) {
	// Simple implementation without parsing
	// In production, this would handle arrays and complex paths
	return d.Get(path)
}

// String returns a string representation of the document
func (d *Document) String() string {
	return fmt.Sprintf("%v", d.ToMap())
}
