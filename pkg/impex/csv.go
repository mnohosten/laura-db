package impex

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mnohosten/laura-db/pkg/document"
)

// CSVExporter exports documents to CSV format
type CSVExporter struct {
	Fields []string // Specific fields to export (empty = all fields)
}

// NewCSVExporter creates a new CSV exporter
func NewCSVExporter(fields []string) *CSVExporter {
	return &CSVExporter{Fields: fields}
}

// Export writes documents to the writer in CSV format
func (e *CSVExporter) Export(writer io.Writer, docs []*document.Document) error {
	if len(docs) == 0 {
		return nil // Nothing to export
	}

	// Determine which fields to export
	fields := e.Fields
	if len(fields) == 0 {
		// Auto-detect fields from first document and all documents
		fieldSet := make(map[string]bool)
		for _, doc := range docs {
			docMap := doc.ToMap()
			for key := range docMap {
				fieldSet[key] = true
			}
		}
		fields = make([]string, 0, len(fieldSet))
		for field := range fieldSet {
			fields = append(fields, field)
		}
		// Sort fields for consistent output (with _id first)
		sort.Slice(fields, func(i, j int) bool {
			if fields[i] == "_id" {
				return true
			}
			if fields[j] == "_id" {
				return false
			}
			return fields[i] < fields[j]
		})
	}

	// Create CSV writer
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Write header row
	if err := csvWriter.Write(fields); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write document rows
	for _, doc := range docs {
		row := make([]string, len(fields))
		for i, field := range fields {
			if value, exists := doc.Get(field); exists {
				row[i] = e.formatValue(value)
			} else {
				row[i] = "" // Empty string for missing fields
			}
		}
		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// formatValue converts a document value to a CSV string
func (e *CSVExporter) formatValue(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case document.ObjectID:
		return v.Hex()
	case time.Time:
		return v.Format(time.RFC3339)
	case []interface{}, map[string]interface{}:
		// For complex types, encode as JSON
		bytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(bytes)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// CSVImporter imports documents from CSV format
type CSVImporter struct {
	Headers []string // Column headers (if not in first row)
}

// NewCSVImporter creates a new CSV importer
func NewCSVImporter(headers []string) *CSVImporter {
	return &CSVImporter{Headers: headers}
}

// Import reads documents from the reader in CSV format
func (i *CSVImporter) Import(reader io.Reader) ([]*document.Document, error) {
	csvReader := csv.NewReader(reader)

	// Read header row or use provided headers
	var headers []string
	if len(i.Headers) > 0 {
		headers = i.Headers
	} else {
		var err error
		headers, err = csvReader.Read()
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV header: %w", err)
		}
	}

	// Read all rows
	docs := make([]*document.Document, 0)
	rowNum := 1 // Start at 1 (after header)

	for {
		row, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row %d: %w", rowNum, err)
		}

		// Parse row into document
		doc, err := i.parseRow(headers, row)
		if err != nil {
			return nil, fmt.Errorf("failed to parse CSV row %d: %w", rowNum, err)
		}

		docs = append(docs, doc)
		rowNum++
	}

	return docs, nil
}

// parseRow converts a CSV row to a Document
func (i *CSVImporter) parseRow(headers []string, row []string) (*document.Document, error) {
	docMap := make(map[string]interface{})

	for idx, header := range headers {
		if idx >= len(row) {
			break // Skip if row has fewer columns than headers
		}

		value := row[idx]
		if value == "" {
			continue // Skip empty values
		}

		// Parse value
		parsedValue := i.parseValue(value)
		docMap[header] = parsedValue
	}

	return document.NewDocumentFromMap(docMap), nil
}

// parseValue parses a CSV string value to appropriate type
func (i *CSVImporter) parseValue(value string) interface{} {
	// Try to parse as different types in order of specificity

	// 1. Try boolean
	if value == "true" || value == "false" {
		return value == "true"
	}

	// 2. Try integer
	if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
		return intVal
	}

	// 3. Try float
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}

	// 4. Try ObjectID (24 hex characters)
	if len(value) == 24 {
		if oid, err := document.ObjectIDFromHex(value); err == nil {
			return oid
		}
	}

	// 5. Try RFC3339 timestamp
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t
	}

	// 6. Try JSON array or object (starts with [ or {)
	if strings.HasPrefix(value, "[") || strings.HasPrefix(value, "{") {
		var jsonValue interface{}
		if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
			return i.convertJSONValue(jsonValue)
		}
	}

	// 7. Default to string
	return value
}

// convertJSONValue converts JSON-decoded values to document-compatible types
func (i *CSVImporter) convertJSONValue(value interface{}) interface{} {
	switch v := value.(type) {
	case float64:
		// Convert to int64 if it's a whole number
		if v == float64(int64(v)) {
			return int64(v)
		}
		return v
	case []interface{}:
		// Recursively convert array elements
		result := make([]interface{}, len(v))
		for idx, elem := range v {
			result[idx] = i.convertJSONValue(elem)
		}
		return result
	case map[string]interface{}:
		// Recursively convert nested objects
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = i.convertJSONValue(val)
		}
		return result
	default:
		return v
	}
}
