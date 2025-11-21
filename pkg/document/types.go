package document

import "time"

// Type represents the BSON data type of a value
type Type byte

const (
	TypeFloat64   Type = 0x01
	TypeString    Type = 0x02
	TypeDocument  Type = 0x03
	TypeArray     Type = 0x04
	TypeBinary    Type = 0x05
	TypeObjectID  Type = 0x07
	TypeBoolean   Type = 0x08
	TypeNull      Type = 0x0A
	TypeInt32     Type = 0x10
	TypeTimestamp Type = 0x11
	TypeInt64     Type = 0x12
)

// String returns the string representation of the type
func (t Type) String() string {
	switch t {
	case TypeNull:
		return "null"
	case TypeBoolean:
		return "boolean"
	case TypeInt32:
		return "int32"
	case TypeInt64:
		return "int64"
	case TypeFloat64:
		return "float64"
	case TypeString:
		return "string"
	case TypeBinary:
		return "binary"
	case TypeObjectID:
		return "objectid"
	case TypeArray:
		return "array"
	case TypeDocument:
		return "document"
	case TypeTimestamp:
		return "timestamp"
	default:
		return "unknown"
	}
}

// Value represents a typed value in a document
type Value struct {
	Type Type
	Data interface{}
}

// NewValue creates a new typed value
func NewValue(data interface{}) *Value {
	v := &Value{Data: data}

	switch data.(type) {
	case nil:
		v.Type = TypeNull
	case bool:
		v.Type = TypeBoolean
	case int32:
		v.Type = TypeInt32
	case int64:
		v.Type = TypeInt64
	case int:
		// Convert int to int64
		v.Type = TypeInt64
		v.Data = int64(data.(int))
	case float64:
		v.Type = TypeFloat64
	case string:
		v.Type = TypeString
	case []byte:
		v.Type = TypeBinary
	case ObjectID:
		v.Type = TypeObjectID
	case time.Time:
		v.Type = TypeTimestamp
	case []interface{}:
		v.Type = TypeArray
	case map[string]interface{}:
		v.Type = TypeDocument
	case *Document:
		v.Type = TypeDocument
	case Document:
		v.Type = TypeDocument
	default:
		v.Type = TypeNull
		v.Data = nil
	}

	return v
}
