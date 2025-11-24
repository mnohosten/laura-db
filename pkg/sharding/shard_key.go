package sharding

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"sort"

	"github.com/mnohosten/laura-db/pkg/document"
)

// ShardKeyType represents the type of sharding strategy
type ShardKeyType int

const (
	// ShardKeyTypeRange uses range-based sharding
	ShardKeyTypeRange ShardKeyType = iota
	// ShardKeyTypeHash uses hash-based sharding
	ShardKeyTypeHash
)

func (t ShardKeyType) String() string {
	switch t {
	case ShardKeyTypeRange:
		return "range"
	case ShardKeyTypeHash:
		return "hash"
	default:
		return "unknown"
	}
}

// ShardKey represents the configuration for sharding
type ShardKey struct {
	// Fields to shard on (e.g., ["user_id"], ["country", "user_id"])
	Fields []string
	// Type of sharding (range or hash)
	Type ShardKeyType
	// Unique indicates if shard key guarantees uniqueness
	Unique bool
}

// NewShardKey creates a new shard key configuration
func NewShardKey(fields []string, shardType ShardKeyType) *ShardKey {
	return &ShardKey{
		Fields: fields,
		Type:   shardType,
		Unique: false,
	}
}

// NewRangeShardKey creates a range-based shard key
func NewRangeShardKey(fields ...string) *ShardKey {
	return &ShardKey{
		Fields: fields,
		Type:   ShardKeyTypeRange,
		Unique: false,
	}
}

// NewHashShardKey creates a hash-based shard key
func NewHashShardKey(fields ...string) *ShardKey {
	return &ShardKey{
		Fields: fields,
		Type:   ShardKeyTypeHash,
		Unique: false,
	}
}

// ExtractShardKeyValue extracts the shard key value from a document
func (sk *ShardKey) ExtractShardKeyValue(doc map[string]interface{}) (interface{}, error) {
	if len(sk.Fields) == 0 {
		return nil, fmt.Errorf("shard key has no fields")
	}

	// Single field shard key
	if len(sk.Fields) == 1 {
		val, ok := doc[sk.Fields[0]]
		if !ok {
			return nil, fmt.Errorf("document missing shard key field: %s", sk.Fields[0])
		}
		return val, nil
	}

	// Compound shard key - create composite key
	compositeKey := make(map[string]interface{})
	for _, field := range sk.Fields {
		val, ok := doc[field]
		if !ok {
			return nil, fmt.Errorf("document missing shard key field: %s", field)
		}
		compositeKey[field] = val
	}

	return compositeKey, nil
}

// HashValue computes a hash value for a given shard key value
// Used for hash-based sharding
func (sk *ShardKey) HashValue(value interface{}) uint64 {
	// Convert value to bytes for hashing
	var data []byte

	switch v := value.(type) {
	case string:
		data = []byte(v)
	case int:
		data = make([]byte, 8)
		binary.BigEndian.PutUint64(data, uint64(v))
	case int64:
		data = make([]byte, 8)
		binary.BigEndian.PutUint64(data, uint64(v))
	case float64:
		data = make([]byte, 8)
		binary.BigEndian.PutUint64(data, uint64(v))
	case document.ObjectID:
		data = v[:]
	case map[string]interface{}:
		// Compound key - concatenate field values
		var keys []string
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys) // Ensure consistent ordering

		for _, k := range keys {
			fieldData := fmt.Sprintf("%v", v[k])
			data = append(data, []byte(fieldData)...)
		}
	default:
		// Fallback to string representation
		data = []byte(fmt.Sprintf("%v", v))
	}

	// Use FNV-1a hash for speed and good distribution
	h := fnv.New64a()
	h.Write(data)
	return h.Sum64()
}

// HashValueMD5 computes an MD5 hash value (alternative hash function)
// Some systems prefer MD5 for better distribution characteristics
func (sk *ShardKey) HashValueMD5(value interface{}) uint64 {
	var data []byte

	switch v := value.(type) {
	case string:
		data = []byte(v)
	case int:
		data = make([]byte, 8)
		binary.BigEndian.PutUint64(data, uint64(v))
	case int64:
		data = make([]byte, 8)
		binary.BigEndian.PutUint64(data, uint64(v))
	case float64:
		data = make([]byte, 8)
		binary.BigEndian.PutUint64(data, uint64(v))
	case document.ObjectID:
		data = v[:]
	case map[string]interface{}:
		// Compound key - concatenate field values
		var keys []string
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			fieldData := fmt.Sprintf("%v", v[k])
			data = append(data, []byte(fieldData)...)
		}
	default:
		data = []byte(fmt.Sprintf("%v", v))
	}

	// Compute MD5 hash
	hash := md5.Sum(data)
	// Convert first 8 bytes to uint64
	return binary.BigEndian.Uint64(hash[:8])
}

// CompareValues compares two shard key values for range-based sharding
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func (sk *ShardKey) CompareValues(a, b interface{}) int {
	// Handle nil values
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	// Type-specific comparison
	switch va := a.(type) {
	case string:
		vb, ok := b.(string)
		if !ok {
			// Type mismatch - compare as strings
			return compareStrings(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b))
		}
		return compareStrings(va, vb)

	case int:
		vb, ok := b.(int)
		if !ok {
			return compareStrings(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b))
		}
		return compareInts(int64(va), int64(vb))

	case int64:
		vb, ok := b.(int64)
		if !ok {
			return compareStrings(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b))
		}
		return compareInts(va, vb)

	case float64:
		vb, ok := b.(float64)
		if !ok {
			return compareStrings(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b))
		}
		return compareFloats(va, vb)

	case document.ObjectID:
		vb, ok := b.(document.ObjectID)
		if !ok {
			return compareStrings(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b))
		}
		return compareObjectIDs(va, vb)

	case map[string]interface{}:
		// Compound key comparison
		vb, ok := b.(map[string]interface{})
		if !ok {
			return compareStrings(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b))
		}
		return sk.compareCompoundKeys(va, vb)

	default:
		// Fallback to string comparison
		return compareStrings(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b))
	}
}

// compareCompoundKeys compares two compound keys field by field
func (sk *ShardKey) compareCompoundKeys(a, b map[string]interface{}) int {
	// Compare fields in order
	for _, field := range sk.Fields {
		va, aok := a[field]
		vb, bok := b[field]

		if !aok && !bok {
			continue
		}
		if !aok {
			return -1
		}
		if !bok {
			return 1
		}

		// Compare field values
		cmp := sk.CompareValues(va, vb)
		if cmp != 0 {
			return cmp
		}
	}

	return 0
}

// Helper comparison functions
func compareStrings(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func compareInts(a, b int64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func compareFloats(a, b float64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func compareObjectIDs(a, b document.ObjectID) int {
	for i := 0; i < len(a); i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

// Validate validates the shard key configuration
func (sk *ShardKey) Validate() error {
	if len(sk.Fields) == 0 {
		return fmt.Errorf("shard key must have at least one field")
	}

	// Check for duplicate fields
	seen := make(map[string]bool)
	for _, field := range sk.Fields {
		if seen[field] {
			return fmt.Errorf("duplicate field in shard key: %s", field)
		}
		seen[field] = true
	}

	return nil
}

// String returns a string representation of the shard key
func (sk *ShardKey) String() string {
	if len(sk.Fields) == 1 {
		return fmt.Sprintf("{%s: %s}", sk.Fields[0], sk.Type)
	}
	return fmt.Sprintf("{%v: %s}", sk.Fields, sk.Type)
}
