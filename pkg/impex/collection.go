package impex

import (
	"fmt"
	"io"

	"github.com/mnohosten/laura-db/pkg/document"
)

// CollectionExporter provides high-level export functionality for collections
type CollectionExporter struct{}

// NewCollectionExporter creates a new collection exporter
func NewCollectionExporter() *CollectionExporter {
	return &CollectionExporter{}
}

// ExportJSON exports documents to JSON format
func (e *CollectionExporter) ExportJSON(writer io.Writer, docs []*document.Document, pretty bool) error {
	exporter := NewJSONExporter(pretty)
	return exporter.Export(writer, docs)
}

// ExportCSV exports documents to CSV format with optional field selection
func (e *CollectionExporter) ExportCSV(writer io.Writer, docs []*document.Document, fields []string) error {
	exporter := NewCSVExporter(fields)
	return exporter.Export(writer, docs)
}

// CollectionImporter provides high-level import functionality for collections
type CollectionImporter struct{}

// NewCollectionImporter creates a new collection importer
func NewCollectionImporter() *CollectionImporter {
	return &CollectionImporter{}
}

// ImportJSON imports documents from JSON format
func (i *CollectionImporter) ImportJSON(reader io.Reader) ([]*document.Document, error) {
	importer := NewJSONImporter()
	return importer.Import(reader)
}

// ImportCSV imports documents from CSV format with optional headers
func (i *CollectionImporter) ImportCSV(reader io.Reader, headers []string) ([]*document.Document, error) {
	importer := NewCSVImporter(headers)
	return importer.Import(reader)
}

// Format represents the export/import format
type Format string

const (
	// FormatJSON represents JSON format
	FormatJSON Format = "json"
	// FormatCSV represents CSV format
	FormatCSV Format = "csv"
)

// Export is a convenience function that exports documents in the specified format
func Export(writer io.Writer, docs []*document.Document, format Format, options map[string]interface{}) error {
	switch format {
	case FormatJSON:
		pretty := false
		if val, ok := options["pretty"]; ok {
			if p, ok := val.(bool); ok {
				pretty = p
			}
		}
		exporter := NewJSONExporter(pretty)
		return exporter.Export(writer, docs)

	case FormatCSV:
		var fields []string
		if val, ok := options["fields"]; ok {
			if f, ok := val.([]string); ok {
				fields = f
			}
		}
		exporter := NewCSVExporter(fields)
		return exporter.Export(writer, docs)

	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}
}

// Import is a convenience function that imports documents in the specified format
func Import(reader io.Reader, format Format, options map[string]interface{}) ([]*document.Document, error) {
	switch format {
	case FormatJSON:
		importer := NewJSONImporter()
		return importer.Import(reader)

	case FormatCSV:
		var headers []string
		if val, ok := options["headers"]; ok {
			if h, ok := val.([]string); ok {
				headers = h
			}
		}
		importer := NewCSVImporter(headers)
		return importer.Import(reader)

	default:
		return nil, fmt.Errorf("unsupported import format: %s", format)
	}
}
