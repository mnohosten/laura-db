package impex

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/mnohosten/laura-db/pkg/document"
)

// JSONExporter exports documents to JSON format
type JSONExporter struct {
	Pretty bool // Enable pretty-printing (indentation)
}

// NewJSONExporter creates a new JSON exporter
func NewJSONExporter(pretty bool) *JSONExporter {
	return &JSONExporter{Pretty: pretty}
}

// Export writes documents to the writer in JSON format
func (e *JSONExporter) Export(writer io.Writer, docs []*document.Document) error {
	// Convert documents to exportable format
	exportDocs := make([]map[string]interface{}, 0, len(docs))
	for _, doc := range docs {
		exportDoc := e.prepareDocument(doc)
		exportDocs = append(exportDocs, exportDoc)
	}

	// Encode to JSON
	encoder := json.NewEncoder(writer)
	if e.Pretty {
		encoder.SetIndent("", "  ")
	}

	if err := encoder.Encode(exportDocs); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// prepareDocument converts a Document to a map suitable for JSON export
func (e *JSONExporter) prepareDocument(doc *document.Document) map[string]interface{} {
	result := make(map[string]interface{})

	// Get all fields from document
	docMap := doc.ToMap()
	for key, value := range docMap {
		result[key] = e.convertValue(value)
	}

	return result
}

// convertValue converts document values to JSON-compatible types
func (e *JSONExporter) convertValue(value interface{}) interface{} {
	switch v := value.(type) {
	case document.ObjectID:
		// Convert ObjectID to hex string
		return v.Hex()
	case time.Time:
		// Convert time to RFC3339 string
		return v.Format(time.RFC3339)
	case []interface{}:
		// Recursively convert array elements
		result := make([]interface{}, len(v))
		for i, elem := range v {
			result[i] = e.convertValue(elem)
		}
		return result
	case map[string]interface{}:
		// Recursively convert nested documents
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = e.convertValue(val)
		}
		return result
	default:
		// Return as-is for primitive types (string, int64, float64, bool, nil)
		return v
	}
}

// JSONImporter imports documents from JSON format
type JSONImporter struct{}

// NewJSONImporter creates a new JSON importer
func NewJSONImporter() *JSONImporter {
	return &JSONImporter{}
}

// Import reads documents from the reader in JSON format
func (i *JSONImporter) Import(reader io.Reader) ([]*document.Document, error) {
	var rawDocs []map[string]interface{}

	// Decode JSON
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&rawDocs); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	// Convert to Document objects
	docs := make([]*document.Document, 0, len(rawDocs))
	for idx, rawDoc := range rawDocs {
		doc, err := i.parseDocument(rawDoc)
		if err != nil {
			return nil, fmt.Errorf("failed to parse document at index %d: %w", idx, err)
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

// parseDocument converts a raw map to a Document
func (i *JSONImporter) parseDocument(rawDoc map[string]interface{}) (*document.Document, error) {
	// Parse and convert values
	parsedDoc := make(map[string]interface{})
	for key, value := range rawDoc {
		parsedValue, err := i.parseValue(value)
		if err != nil {
			return nil, fmt.Errorf("failed to parse field %s: %w", key, err)
		}
		parsedDoc[key] = parsedValue
	}

	// Create Document
	doc := document.NewDocumentFromMap(parsedDoc)
	return doc, nil
}

// parseValue converts JSON values to document-compatible types
func (i *JSONImporter) parseValue(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		// Try to parse as ObjectID (24 hex characters)
		if len(v) == 24 {
			if oid, err := document.ObjectIDFromHex(v); err == nil {
				return oid, nil
			}
		}
		// Try to parse as RFC3339 time
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t, nil
		}
		// Return as string
		return v, nil
	case float64:
		// JSON numbers are float64, convert to int64 if it's a whole number
		if v == float64(int64(v)) {
			return int64(v), nil
		}
		return v, nil
	case []interface{}:
		// Recursively parse array elements
		result := make([]interface{}, len(v))
		for idx, elem := range v {
			parsedElem, err := i.parseValue(elem)
			if err != nil {
				return nil, err
			}
			result[idx] = parsedElem
		}
		return result, nil
	case map[string]interface{}:
		// Recursively parse nested documents
		result := make(map[string]interface{})
		for key, val := range v {
			parsedVal, err := i.parseValue(val)
			if err != nil {
				return nil, err
			}
			result[key] = parsedVal
		}
		return result, nil
	default:
		// Return as-is (bool, nil)
		return v, nil
	}
}
