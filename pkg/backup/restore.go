package backup

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/mnohosten/laura-db/pkg/document"
)

// Restorer restores database from backups
type Restorer struct {
	// Future options: Restore to specific point in time, selective restore, etc.
}

// NewRestorer creates a new restorer
func NewRestorer() *Restorer {
	return &Restorer{}
}

// RestoreFromFile restores a database from a backup file
func (r *Restorer) RestoreFromFile(path string) (*BackupFormat, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	return r.RestoreFromReader(file)
}

// RestoreFromReader restores a database from an io.Reader
func (r *Restorer) RestoreFromReader(reader io.Reader) (*BackupFormat, error) {
	var backup BackupFormat

	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&backup); err != nil {
		return nil, fmt.Errorf("failed to decode backup: %w", err)
	}

	// Validate backup format
	if err := r.validateBackup(&backup); err != nil {
		return nil, fmt.Errorf("invalid backup format: %w", err)
	}

	return &backup, nil
}

// validateBackup validates the backup format
func (r *Restorer) validateBackup(backup *BackupFormat) error {
	if backup.Version == "" {
		return fmt.Errorf("missing version field")
	}

	if backup.Version != "1.0" {
		return fmt.Errorf("unsupported backup version: %s", backup.Version)
	}

	if backup.DatabaseName == "" {
		return fmt.Errorf("missing database name")
	}

	if backup.Collections == nil {
		return fmt.Errorf("missing collections field")
	}

	return nil
}

// ConvertDocumentFromBackup converts a DocumentBackup to a map suitable for insertion
func ConvertDocumentFromBackup(docBackup DocumentBackup) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Parse _id
	if docBackup.ID != "" {
		// Try to parse as ObjectID
		if oid, err := document.ObjectIDFromHex(docBackup.ID); err == nil {
			result["_id"] = oid
		} else {
			// Use as string if not a valid ObjectID
			result["_id"] = docBackup.ID
		}
	}

	// Convert all field values
	for key, value := range docBackup.Fields {
		convertedValue, err := convertValueFromBackup(value)
		if err != nil {
			return nil, fmt.Errorf("failed to convert field %s: %w", key, err)
		}
		result[key] = convertedValue
	}

	return result, nil
}

// convertValueFromBackup converts backup values back to document values
func convertValueFromBackup(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		// Try to parse as ObjectID
		if oid, err := document.ObjectIDFromHex(v); err == nil {
			return oid, nil
		}
		// Try to parse as time
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
		// Recursively convert array elements
		result := make([]interface{}, len(v))
		for i, elem := range v {
			converted, err := convertValueFromBackup(elem)
			if err != nil {
				return nil, err
			}
			result[i] = converted
		}
		return result, nil

	case map[string]interface{}:
		// Check if it's a GeoJSON point
		if typeVal, hasType := v["type"]; hasType {
			if typeStr, ok := typeVal.(string); ok && typeStr == "Point" {
				// Return as-is for geo points
				return v, nil
			}
		}

		// Recursively convert nested documents
		result := make(map[string]interface{})
		for key, val := range v {
			converted, err := convertValueFromBackup(val)
			if err != nil {
				return nil, err
			}
			result[key] = converted
		}
		return result, nil

	default:
		// Return as-is for bool, nil, etc.
		return v, nil
	}
}

// RestoreOptions configures restore behavior
type RestoreOptions struct {
	DropExisting   bool   // Drop existing collections before restore
	SkipIndexes    bool   // Don't restore indexes
	TargetDatabase string // Restore to a different database name
}

// DefaultRestoreOptions returns default restore options
func DefaultRestoreOptions() *RestoreOptions {
	return &RestoreOptions{
		DropExisting:   false,
		SkipIndexes:    false,
		TargetDatabase: "",
	}
}

// ValidateBackupFile validates a backup file without restoring it
func ValidateBackupFile(path string) (*BackupFormat, error) {
	restorer := NewRestorer()
	return restorer.RestoreFromFile(path)
}

// ValidateBackup validates a backup format
func ValidateBackup(backup *BackupFormat) error {
	restorer := NewRestorer()
	return restorer.validateBackup(backup)
}
