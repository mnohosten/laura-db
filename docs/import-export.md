# Import/Export Utilities

LauraDB provides comprehensive import/export utilities for moving data in and out of the database using standard formats (JSON and CSV).

## Package: `pkg/impex`

The `impex` package provides both low-level and high-level APIs for importing and exporting documents.

## Features

### JSON Format

#### Export
- **Pretty-printing**: Optional indentation for human-readable output
- **Type preservation**: ObjectID, time.Time, nested documents, and arrays
- **Complete data**: All fields and complex structures preserved

#### Import
- **Smart type parsing**: Automatically detects and converts ObjectID (24-char hex), RFC3339 timestamps
- **Nested structures**: Full support for arrays and nested documents
- **Number conversion**: JSON numbers converted to int64 when appropriate

### CSV Format

#### Export
- **Field selection**: Specify which fields to export
- **Auto-detection**: Automatically detect all fields across documents
- **Complex types**: Arrays and objects encoded as JSON within CSV cells
- **Missing fields**: Handle documents with different schemas gracefully

#### Import
- **Type detection**: Automatically parse int64, float64, bool, and string
- **JSON in cells**: Parse JSON-encoded arrays/objects in CSV cells
- **Header row**: Auto-detect headers or provide custom headers
- **Empty values**: Skip empty cells rather than inserting empty strings

## API Reference

### High-Level Functions

```go
// Export documents to a writer in the specified format
func Export(writer io.Writer, docs []*document.Document, format Format, options map[string]interface{}) error

// Import documents from a reader in the specified format
func Import(reader io.Reader, format Format, options map[string]interface{}) ([]*document.Document, error)
```

**Formats:**
- `impex.FormatJSON` - JSON format
- `impex.FormatCSV` - CSV format

**Options for JSON:**
- `"pretty"` (bool): Enable pretty-printing with indentation

**Options for CSV:**
- `"fields"` ([]string): Specific fields to export (export only)
- `"headers"` ([]string): Column headers (import only, if CSV has no header row)

### Specialized Exporters/Importers

#### JSONExporter

```go
type JSONExporter struct {
    Pretty bool // Enable pretty-printing
}

func NewJSONExporter(pretty bool) *JSONExporter
func (e *JSONExporter) Export(writer io.Writer, docs []*document.Document) error
```

#### JSONImporter

```go
type JSONImporter struct{}

func NewJSONImporter() *JSONImporter
func (i *JSONImporter) Import(reader io.Reader) ([]*document.Document, error)
```

#### CSVExporter

```go
type CSVExporter struct {
    Fields []string // Specific fields to export (empty = all fields)
}

func NewCSVExporter(fields []string) *CSVExporter
func (e *CSVExporter) Export(writer io.Writer, docs []*document.Document) error
```

#### CSVImporter

```go
type CSVImporter struct {
    Headers []string // Column headers (if not in first row)
}

func NewCSVImporter(headers []string) *CSVImporter
func (i *CSVImporter) Import(reader io.Reader) ([]*document.Document, error)
```

## Usage Examples

### Basic JSON Export/Import

```go
package main

import (
    "os"
    "github.com/mnohosten/laura-db/pkg/database"
    "github.com/mnohosten/laura-db/pkg/impex"
)

func main() {
    // Open database and get documents
    db, _ := database.Open(database.DefaultConfig("./data"))
    coll := db.Collection("users")
    docs, _ := coll.Find(map[string]interface{}{})

    // Export to JSON with pretty-printing
    file, _ := os.Create("users.json")
    defer file.Close()

    impex.Export(file, docs, impex.FormatJSON, map[string]interface{}{
        "pretty": true,
    })

    // Import from JSON
    file, _ = os.Open("users.json")
    defer file.Close()

    importedDocs, _ := impex.Import(file, impex.FormatJSON, nil)

    // Insert imported documents
    for _, doc := range importedDocs {
        coll.InsertOne(doc.ToMap())
    }
}
```

### CSV Export with Field Selection

```go
// Export only specific fields to CSV
csvFile, _ := os.Create("users.csv")
defer csvFile.Close()

impex.Export(csvFile, docs, impex.FormatCSV, map[string]interface{}{
    "fields": []string{"name", "email", "age", "active"},
})
```

### CSV Export with Auto-Detection

```go
// Auto-detect all fields across documents
csvFile, _ := os.Create("users-full.csv")
defer csvFile.Close()

impex.Export(csvFile, docs, impex.FormatCSV, nil)
```

### CSV Import with Custom Headers

```go
// Import CSV without header row
csvFile, _ := os.Open("users-no-header.csv")
defer csvFile.Close()

importedDocs, _ := impex.Import(csvFile, impex.FormatCSV, map[string]interface{}{
    "headers": []string{"name", "email", "age"},
})
```

### Using Specialized Exporters

```go
// Create JSON exporter with pretty-printing
jsonExporter := impex.NewJSONExporter(true)
var buffer bytes.Buffer
jsonExporter.Export(&buffer, docs)

// Create CSV exporter with specific fields
csvExporter := impex.NewCSVExporter([]string{"_id", "name", "email"})
csvExporter.Export(&buffer, docs)
```

## Type Handling

### JSON

| Go Type | JSON Export | JSON Import |
|---------|-------------|-------------|
| `string` | String | String |
| `int64` | Number | int64 or float64 |
| `float64` | Number | float64 |
| `bool` | Boolean | Boolean |
| `document.ObjectID` | Hex string (24 chars) | ObjectID |
| `time.Time` | RFC3339 string | time.Time |
| `[]interface{}` | Array | Array |
| `map[string]interface{}` | Object | Object |
| `nil` | null | nil |

### CSV

| Go Type | CSV Export | CSV Import |
|---------|------------|------------|
| `string` | String value | String |
| `int64` | Number string | int64 |
| `float64` | Number string | float64 |
| `bool` | "true"/"false" | bool |
| `document.ObjectID` | Hex string | ObjectID (if 24 chars) |
| `time.Time` | RFC3339 string | time.Time |
| `[]interface{}` | JSON-encoded | Array (if JSON) |
| `map[string]interface{}` | JSON-encoded | Object (if JSON) |
| Missing field | Empty string | Skipped |

## Best Practices

### Performance

1. **Batch operations**: Import documents in batches rather than one at a time
2. **Buffer I/O**: Use buffered readers/writers for large datasets
3. **Field selection**: Specify only needed fields for CSV export to reduce file size

### Data Integrity

1. **Preserve _id**: Keep `_id` field when exporting for data migration
2. **Remove _id**: Delete `_id` when importing to generate new IDs
3. **Round-trip testing**: Test export→import cycle to verify data integrity

### Format Selection

- **JSON**: Best for complex nested structures, full type preservation
- **CSV**: Best for tabular data, Excel compatibility, smaller file size for flat documents

### Error Handling

```go
// Always check errors
docs, err := impex.Import(reader, impex.FormatJSON, nil)
if err != nil {
    log.Fatalf("Import failed: %v", err)
}

// Validate imported data
for i, doc := range docs {
    if doc.Len() == 0 {
        log.Printf("Warning: Document %d is empty", i)
    }
}
```

## Implementation Details

### JSON Export Process

1. Convert each `Document` to `map[string]interface{}` using `ToMap()`
2. Transform special types (ObjectID → hex, time.Time → RFC3339)
3. Encode to JSON using standard `encoding/json` package
4. Apply pretty-printing if requested

### JSON Import Process

1. Decode JSON array using standard `encoding/json` package
2. Parse each element as `map[string]interface{}`
3. Detect and convert special types:
   - 24-char hex strings → ObjectID
   - RFC3339 strings → time.Time
   - Whole numbers → int64
4. Create `Document` from parsed map

### CSV Export Process

1. Determine fields to export (specified or auto-detected)
2. Write header row with field names
3. For each document:
   - Extract field values
   - Format complex types as JSON
   - Write row to CSV writer

### CSV Import Process

1. Read header row (or use provided headers)
2. For each data row:
   - Parse values with type detection
   - Skip empty values
   - Create document from row
3. Return all documents

## Testing

The package includes 20 comprehensive tests covering:

- JSON export/import with various data types
- CSV export/import with field selection
- Round-trip data integrity
- Complex nested structures
- Edge cases (empty documents, missing fields, special characters)
- Type conversion accuracy

Run tests:

```bash
go test ./pkg/impex -v
```

## Examples

See the complete working example in [examples/import-export/main.go](../examples/import-export/main.go) which demonstrates:

- Inserting sample documents
- Exporting to JSON and CSV
- Importing from both formats
- Re-inserting imported documents
- Auto-detected field export

Run the example:

```bash
make examples
./bin/import-export-demo
```

## Future Enhancements

Potential improvements for future versions:

- **Streaming**: Support streaming large datasets without loading into memory
- **Compression**: Gzip support for compressed export/import
- **Formats**: Additional formats (XML, Parquet, Avro)
- **Validation**: Schema validation during import
- **Transformation**: Field mapping and transformation during import/export
- **Progress**: Progress callbacks for long-running operations
