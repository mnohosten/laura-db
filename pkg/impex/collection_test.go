package impex

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mnohosten/laura-db/pkg/document"
)

func TestCollectionExporter(t *testing.T) {
	// Create test documents
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"name": "Alice",
			"age":  int64(30),
		}),
		document.NewDocumentFromMap(map[string]interface{}{
			"name": "Bob",
			"age":  int64(25),
		}),
	}

	t.Run("NewCollectionExporter", func(t *testing.T) {
		exporter := NewCollectionExporter()
		if exporter == nil {
			t.Fatal("Expected non-nil exporter")
		}
	})

	t.Run("ExportJSON", func(t *testing.T) {
		exporter := NewCollectionExporter()
		var buf bytes.Buffer
		err := exporter.ExportJSON(&buf, docs, false)
		if err != nil {
			t.Fatalf("ExportJSON failed: %v", err)
		}
		if buf.Len() == 0 {
			t.Error("Expected non-empty output")
		}
	})

	t.Run("ExportJSONPretty", func(t *testing.T) {
		exporter := NewCollectionExporter()
		var buf bytes.Buffer
		err := exporter.ExportJSON(&buf, docs, true)
		if err != nil {
			t.Fatalf("ExportJSON with pretty failed: %v", err)
		}
		if buf.Len() == 0 {
			t.Error("Expected non-empty output")
		}
	})

	t.Run("ExportCSV", func(t *testing.T) {
		exporter := NewCollectionExporter()
		var buf bytes.Buffer
		err := exporter.ExportCSV(&buf, docs, []string{"name", "age"})
		if err != nil {
			t.Fatalf("ExportCSV failed: %v", err)
		}
		if buf.Len() == 0 {
			t.Error("Expected non-empty output")
		}
	})

	t.Run("ExportCSVAutoDetect", func(t *testing.T) {
		exporter := NewCollectionExporter()
		var buf bytes.Buffer
		err := exporter.ExportCSV(&buf, docs, nil) // Auto-detect fields
		if err != nil {
			t.Fatalf("ExportCSV with auto-detect failed: %v", err)
		}
		if buf.Len() == 0 {
			t.Error("Expected non-empty output")
		}
	})
}

func TestCollectionImporter(t *testing.T) {
	t.Run("NewCollectionImporter", func(t *testing.T) {
		importer := NewCollectionImporter()
		if importer == nil {
			t.Fatal("Expected non-nil importer")
		}
	})

	t.Run("ImportJSON", func(t *testing.T) {
		jsonData := `[{"name":"Alice","age":30},{"name":"Bob","age":25}]`
		importer := NewCollectionImporter()
		docs, err := importer.ImportJSON(strings.NewReader(jsonData))
		if err != nil {
			t.Fatalf("ImportJSON failed: %v", err)
		}
		if len(docs) != 2 {
			t.Errorf("Expected 2 documents, got %d", len(docs))
		}
	})

	t.Run("ImportCSV", func(t *testing.T) {
		csvData := "name,age\nAlice,30\nBob,25"
		importer := NewCollectionImporter()
		docs, err := importer.ImportCSV(strings.NewReader(csvData), nil)
		if err != nil {
			t.Fatalf("ImportCSV failed: %v", err)
		}
		if len(docs) != 2 {
			t.Errorf("Expected 2 documents, got %d", len(docs))
		}
	})

	t.Run("ImportCSVWithHeaders", func(t *testing.T) {
		csvData := "Alice,30\nBob,25"
		importer := NewCollectionImporter()
		docs, err := importer.ImportCSV(strings.NewReader(csvData), []string{"name", "age"})
		if err != nil {
			t.Fatalf("ImportCSV with headers failed: %v", err)
		}
		if len(docs) != 2 {
			t.Errorf("Expected 2 documents, got %d", len(docs))
		}
	})
}

func TestExportConvenienceFunction(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentFromMap(map[string]interface{}{
			"name": "Alice",
			"age":  int64(30),
		}),
	}

	t.Run("ExportJSON", func(t *testing.T) {
		var buf bytes.Buffer
		err := Export(&buf, docs, FormatJSON, nil)
		if err != nil {
			t.Fatalf("Export JSON failed: %v", err)
		}
		if buf.Len() == 0 {
			t.Error("Expected non-empty output")
		}
	})

	t.Run("ExportJSONPretty", func(t *testing.T) {
		var buf bytes.Buffer
		err := Export(&buf, docs, FormatJSON, map[string]interface{}{"pretty": true})
		if err != nil {
			t.Fatalf("Export JSON pretty failed: %v", err)
		}
		if buf.Len() == 0 {
			t.Error("Expected non-empty output")
		}
	})

	t.Run("ExportJSONPrettyInvalidType", func(t *testing.T) {
		var buf bytes.Buffer
		// Pass invalid type for pretty option
		err := Export(&buf, docs, FormatJSON, map[string]interface{}{"pretty": "invalid"})
		if err != nil {
			t.Fatalf("Export JSON should ignore invalid pretty type: %v", err)
		}
	})

	t.Run("ExportCSV", func(t *testing.T) {
		var buf bytes.Buffer
		err := Export(&buf, docs, FormatCSV, nil)
		if err != nil {
			t.Fatalf("Export CSV failed: %v", err)
		}
		if buf.Len() == 0 {
			t.Error("Expected non-empty output")
		}
	})

	t.Run("ExportCSVWithFields", func(t *testing.T) {
		var buf bytes.Buffer
		err := Export(&buf, docs, FormatCSV, map[string]interface{}{"fields": []string{"name"}})
		if err != nil {
			t.Fatalf("Export CSV with fields failed: %v", err)
		}
		if buf.Len() == 0 {
			t.Error("Expected non-empty output")
		}
	})

	t.Run("ExportCSVFieldsInvalidType", func(t *testing.T) {
		var buf bytes.Buffer
		// Pass invalid type for fields option
		err := Export(&buf, docs, FormatCSV, map[string]interface{}{"fields": "invalid"})
		if err != nil {
			t.Fatalf("Export CSV should ignore invalid fields type: %v", err)
		}
	})

	t.Run("ExportUnsupportedFormat", func(t *testing.T) {
		var buf bytes.Buffer
		err := Export(&buf, docs, Format("xml"), nil)
		if err == nil {
			t.Error("Expected error for unsupported format")
		}
	})
}

func TestImportConvenienceFunction(t *testing.T) {
	t.Run("ImportJSON", func(t *testing.T) {
		jsonData := `[{"name":"Alice","age":30}]`
		docs, err := Import(strings.NewReader(jsonData), FormatJSON, nil)
		if err != nil {
			t.Fatalf("Import JSON failed: %v", err)
		}
		if len(docs) != 1 {
			t.Errorf("Expected 1 document, got %d", len(docs))
		}
	})

	t.Run("ImportCSV", func(t *testing.T) {
		csvData := "name,age\nAlice,30"
		docs, err := Import(strings.NewReader(csvData), FormatCSV, nil)
		if err != nil {
			t.Fatalf("Import CSV failed: %v", err)
		}
		if len(docs) != 1 {
			t.Errorf("Expected 1 document, got %d", len(docs))
		}
	})

	t.Run("ImportCSVWithHeaders", func(t *testing.T) {
		csvData := "Alice,30"
		docs, err := Import(strings.NewReader(csvData), FormatCSV, map[string]interface{}{"headers": []string{"name", "age"}})
		if err != nil {
			t.Fatalf("Import CSV with headers failed: %v", err)
		}
		if len(docs) != 1 {
			t.Errorf("Expected 1 document, got %d", len(docs))
		}
	})

	t.Run("ImportCSVHeadersInvalidType", func(t *testing.T) {
		csvData := "name,age\nAlice,30"
		// Pass invalid type for headers option
		docs, err := Import(strings.NewReader(csvData), FormatCSV, map[string]interface{}{"headers": "invalid"})
		if err != nil {
			t.Fatalf("Import CSV should ignore invalid headers type: %v", err)
		}
		if len(docs) != 1 {
			t.Errorf("Expected 1 document, got %d", len(docs))
		}
	})

	t.Run("ImportUnsupportedFormat", func(t *testing.T) {
		_, err := Import(strings.NewReader("data"), Format("xml"), nil)
		if err == nil {
			t.Error("Expected error for unsupported format")
		}
	})
}

func TestFormatConstants(t *testing.T) {
	t.Run("FormatJSON", func(t *testing.T) {
		if FormatJSON != "json" {
			t.Errorf("Expected FormatJSON to be 'json', got %s", FormatJSON)
		}
	})

	t.Run("FormatCSV", func(t *testing.T) {
		if FormatCSV != "csv" {
			t.Errorf("Expected FormatCSV to be 'csv', got %s", FormatCSV)
		}
	})
}
