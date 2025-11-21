# Document Format

## Overview

The document format is a BSON-like binary encoding that efficiently stores structured data. It supports various data types and maintains field ordering for consistency.

## Supported Types

| Type | Description | Storage Size |
|------|-------------|--------------|
| Null | Absence of value | 0 bytes |
| Boolean | true/false | 1 byte |
| Int32 | 32-bit integer | 4 bytes |
| Int64 | 64-bit integer | 8 bytes |
| Float64 | 64-bit floating point | 8 bytes |
| String | UTF-8 string | 4 bytes (length) + string + 1 byte (null) |
| Binary | Raw byte array | 4 bytes (length) + 1 byte (subtype) + data |
| ObjectID | 12-byte unique identifier | 12 bytes |
| Array | Ordered list of values | Encoded as document |
| Document | Nested key-value pairs | Variable |
| Timestamp | Unix timestamp (int64) | 8 bytes |

## BSON Encoding Format

### Document Structure

```
[4-byte size][elements...][0x00 terminator]
```

### Element Structure

```
[1-byte type][cstring key][value data]
```

### Example Encoding

Document:
```json
{
  "name": "Alice",
  "age": 30,
  "active": true
}
```

Binary layout:
```
[size: 37 bytes]
  [0x06][name\0][0x06][Alice\0]
  [0x03][age\0][30 00 00 00]
  [0x01][active\0][0x01]
[0x00]
```

## ObjectID

ObjectID provides a unique 12-byte identifier for documents.

### Structure

```
[4 bytes: timestamp][5 bytes: random][3 bytes: counter]
```

- **Timestamp**: Unix timestamp in seconds (allows rough chronological sorting)
- **Random**: Random bytes for uniqueness across machines
- **Counter**: Incrementing counter for uniqueness within the same second

### Properties

- **Unique**: Extremely low collision probability
- **Sortable**: Earlier IDs sort before later ones
- **Embedded timestamp**: Creation time can be extracted
- **Compact**: Only 12 bytes vs UUID's 16 bytes

### Usage

```go
// Generate new ID
id := document.NewObjectID()
fmt.Println(id.Hex()) // "507f1f77bcf86cd799439011"

// Parse from hex
id, err := document.ObjectIDFromHex("507f1f77bcf86cd799439011")

// Extract timestamp
timestamp := id.Timestamp()
```

## Document Operations

### Creating Documents

```go
// Empty document
doc := document.NewDocument()

// From map
doc := document.NewDocumentFromMap(map[string]interface{}{
    "name": "Alice",
    "age": 30,
})

// Setting fields
doc.Set("email", "alice@example.com")
doc.Set("tags", []interface{}{"admin", "user"})
```

### Reading Fields

```go
// Get field value
value, exists := doc.Get("name")
if exists {
    name := value.(string)
}

// Get typed value
val, exists := doc.GetValue("age")
if exists && val.Type == document.TypeInt64 {
    age := val.Data.(int64)
}

// Check field existence
if doc.Has("email") {
    // Field exists
}
```

### Modifying Documents

```go
// Update field
doc.Set("age", 31)

// Delete field
doc.Delete("email")

// Clone document
clone := doc.Clone()
```

### Nested Documents

```go
// Create nested document
doc := document.NewDocument()
doc.Set("user", map[string]interface{}{
    "name": "Alice",
    "address": map[string]interface{}{
        "city": "New York",
        "zip": "10001",
    },
})

// Access nested field (simple implementation)
value, _ := doc.Get("user")
userDoc := value.(*document.Document)
address, _ := userDoc.Get("address")
```

## Encoding/Decoding

### Encoding to BSON

```go
doc := document.NewDocument()
doc.Set("name", "Alice")
doc.Set("age", int64(30))

encoder := document.NewEncoder()
data, err := encoder.Encode(doc)
// data is now a byte slice containing BSON representation
```

### Decoding from BSON

```go
decoder := document.NewDecoder(data)
doc, err := decoder.Decode()

name, _ := doc.Get("name")
fmt.Println(name) // "Alice"
```

## Design Decisions

### Why BSON-like format?

1. **Binary efficiency**: More compact than JSON, faster to parse
2. **Type preservation**: Distinguishes int32 vs int64, maintains type information
3. **Embedded length**: Documents know their size, enabling fast skipping
4. **Null termination**: C-strings are simple and widely compatible

### Field Ordering

We maintain insertion order for:
- **Deterministic serialization**: Same document always encodes identically
- **Debugging**: Easier to read and compare binary dumps
- **Index stability**: Field order doesn't affect comparisons

### Trade-offs

**Advantages:**
- Fast serialization/deserialization
- Type-safe with preserved type information
- Compact binary representation
- Self-describing format

**Limitations:**
- Larger than Protocol Buffers (no schema compression)
- No built-in schema validation
- Fixed types (can't extend without format version bump)

## Performance Characteristics

| Operation | Complexity | Notes |
|-----------|------------|-------|
| Encode | O(n) | n = total bytes in document |
| Decode | O(n) | Single pass through data |
| Get field | O(1) | HashMap lookup |
| Set field | O(1) | HashMap insert |
| Clone | O(n) | Deep copy of all fields |

## Future Enhancements

Potential improvements for educational exploration:

1. **Compression**: LZ4 or Snappy for reduced storage
2. **Schema validation**: JSON Schema-like validation
3. **Decimal128**: Precise decimal numbers for financial data
4. **Regular expressions**: First-class regex type
5. **Code with scope**: Store JavaScript code (MongoDB compatibility)
6. **Min/Max keys**: Special comparison values for range queries
