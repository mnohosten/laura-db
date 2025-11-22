package index

import (
	"bytes"
	"fmt"

	"github.com/mnohosten/laura-db/pkg/document"
)

// CompositeKey represents a key composed of multiple field values
// Used for compound indexes on multiple fields
type CompositeKey struct {
	Values []interface{} // Field values in order
}

// NewCompositeKey creates a new composite key from multiple values
func NewCompositeKey(values ...interface{}) *CompositeKey {
	return &CompositeKey{
		Values: values,
	}
}

// Compare compares two composite keys field by field
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func (ck *CompositeKey) Compare(other *CompositeKey) int {
	// Compare each field in order
	minLen := len(ck.Values)
	if len(other.Values) < minLen {
		minLen = len(other.Values)
	}

	for i := 0; i < minLen; i++ {
		cmp := compareValues(ck.Values[i], other.Values[i])
		if cmp != 0 {
			return cmp // First different field determines order
		}
	}

	// All compared fields are equal, check length
	if len(ck.Values) < len(other.Values) {
		return -1 // Shorter key is "less than"
	} else if len(ck.Values) > len(other.Values) {
		return 1 // Longer key is "greater than"
	}

	return 0 // Completely equal
}

// MatchesPrefix checks if this composite key matches a prefix
// Used for queries that only specify some fields of a compound index
// Example: index on [city, age], query on city only
func (ck *CompositeKey) MatchesPrefix(prefix *CompositeKey) bool {
	if len(prefix.Values) > len(ck.Values) {
		return false
	}

	for i := 0; i < len(prefix.Values); i++ {
		if compareValues(ck.Values[i], prefix.Values[i]) != 0 {
			return false
		}
	}

	return true
}

// String returns a string representation of the composite key
func (ck *CompositeKey) String() string {
	return fmt.Sprintf("%v", ck.Values)
}

// compareValues compares two values of potentially different types
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func compareValues(a, b interface{}) int {
	// Handle nil values
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1 // nil is less than any value
	}
	if b == nil {
		return 1 // any value is greater than nil
	}

	// Type-specific comparison
	switch va := a.(type) {
	case int:
		if vb, ok := b.(int); ok {
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}
	case int64:
		if vb, ok := b.(int64); ok {
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}
		// Try to compare with int
		if vb, ok := b.(int); ok {
			vb64 := int64(vb)
			if va < vb64 {
				return -1
			} else if va > vb64 {
				return 1
			}
			return 0
		}
	case int32:
		if vb, ok := b.(int32); ok {
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}
	case float64:
		if vb, ok := b.(float64); ok {
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}
	case string:
		if vb, ok := b.(string); ok {
			return bytes.Compare([]byte(va), []byte(vb))
		}
	case []byte:
		if vb, ok := b.([]byte); ok {
			return bytes.Compare(va, vb)
		}
	case document.ObjectID:
		if vb, ok := b.(document.ObjectID); ok {
			return bytes.Compare(va[:], vb[:])
		}
	case bool:
		if vb, ok := b.(bool); ok {
			if va == vb {
				return 0
			}
			if !va && vb {
				return -1 // false < true
			}
			return 1
		}
	}

	// If types don't match or are unsupported, treat as equal
	return 0
}
