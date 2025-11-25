package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/mnohosten/laura-db/pkg/database"
)

// Handlers holds the database instance and provides HTTP handlers
type Handlers struct {
	db *database.Database
}

// New creates a new Handlers instance
func New(db *database.Database) *Handlers {
	return &Handlers{db: db}
}

// getCollection retrieves a collection by name or returns error
func (h *Handlers) getCollection(name string) (*database.Collection, error) {
	coll := h.db.Collection(name)
	if coll == nil {
		return nil, &CollectionNotFoundError{Collection: name}
	}
	return coll, nil
}

// parseJSONBody parses JSON request body into target interface
func parseJSONBody(r *http.Request, target interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return &BadRequestError{Message: "failed to read request body"}
	}
	defer r.Body.Close()

	if len(body) == 0 {
		return &BadRequestError{Message: "request body is empty"}
	}

	if err := json.Unmarshal(body, target); err != nil {
		return &BadRequestError{Message: "invalid JSON: " + err.Error()}
	}

	return nil
}

// Error types for consistent error handling

type BadRequestError struct {
	Message string
}

func (e *BadRequestError) Error() string {
	return e.Message
}

type DocumentNotFoundError struct {
	ID string
}

func (e *DocumentNotFoundError) Error() string {
	return "document not found: " + e.ID
}

type CollectionNotFoundError struct {
	Collection string
}

func (e *CollectionNotFoundError) Error() string {
	return "collection not found: " + e.Collection
}

type DuplicateKeyError struct {
	Field string
	Value interface{}
}

func (e *DuplicateKeyError) Error() string {
	return "duplicate key error"
}

type InternalError struct {
	Message string
}

func (e *InternalError) Error() string {
	return e.Message
}

// writeError writes an error response with appropriate HTTP status code
func writeError(w http.ResponseWriter, err error) {
	var statusCode int
	var errorType string
	var message string

	switch e := err.(type) {
	case *BadRequestError:
		statusCode = http.StatusBadRequest
		errorType = "BadRequest"
		message = e.Message
	case *DocumentNotFoundError:
		statusCode = http.StatusNotFound
		errorType = "DocumentNotFound"
		message = e.Error()
	case *CollectionNotFoundError:
		statusCode = http.StatusNotFound
		errorType = "CollectionNotFound"
		message = e.Error()
	case *DuplicateKeyError:
		statusCode = http.StatusConflict
		errorType = "DuplicateKey"
		message = e.Error()
	case *InternalError:
		statusCode = http.StatusInternalServerError
		errorType = "InternalError"
		message = e.Message
	default:
		statusCode = http.StatusInternalServerError
		errorType = "InternalError"
		message = err.Error()
	}

	response := map[string]interface{}{
		"ok":      false,
		"error":   errorType,
		"message": message,
		"code":    statusCode,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// writeSuccess writes a success response
func writeSuccess(w http.ResponseWriter, result interface{}) {
	response := map[string]interface{}{
		"ok":     true,
		"result": result,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// writeSuccessWithCount writes a success response with count
func writeSuccessWithCount(w http.ResponseWriter, result interface{}, count int) {
	response := map[string]interface{}{
		"ok":     true,
		"result": result,
		"count":  count,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
