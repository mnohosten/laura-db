package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mnohosten/laura-db/pkg/database"
)

// InsertDocument inserts a new document with auto-generated ID
func (h *Handlers) InsertDocument(w http.ResponseWriter, r *http.Request) {
	collectionName := chi.URLParam(r, "collection")
	if collectionName == "" {
		writeError(w, &BadRequestError{Message: "collection name is required"})
		return
	}

	coll := h.db.Collection(collectionName)

	var doc map[string]interface{}
	if err := parseJSONBody(r, &doc); err != nil {
		writeError(w, err)
		return
	}

	id, err := coll.InsertOne(doc)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			writeError(w, &DuplicateKeyError{})
		} else {
			writeError(w, &InternalError{Message: err.Error()})
		}
		return
	}

	result := map[string]interface{}{
		"id":         id,
		"collection": collectionName,
	}
	writeSuccess(w, result)
}

// InsertDocumentWithID inserts a document with a specific ID
func (h *Handlers) InsertDocumentWithID(w http.ResponseWriter, r *http.Request) {
	collectionName := chi.URLParam(r, "collection")
	id := chi.URLParam(r, "id")

	if collectionName == "" {
		writeError(w, &BadRequestError{Message: "collection name is required"})
		return
	}
	if id == "" {
		writeError(w, &BadRequestError{Message: "document ID is required"})
		return
	}

	coll := h.db.Collection(collectionName)

	var doc map[string]interface{}
	if err := parseJSONBody(r, &doc); err != nil {
		writeError(w, err)
		return
	}

	// Set the _id field
	doc["_id"] = id

	insertedID, err := coll.InsertOne(doc)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			writeError(w, &DuplicateKeyError{})
		} else {
			writeError(w, &InternalError{Message: err.Error()})
		}
		return
	}

	result := map[string]interface{}{
		"id":         insertedID,
		"collection": collectionName,
	}
	writeSuccess(w, result)
}

// GetDocument retrieves a document by ID
func (h *Handlers) GetDocument(w http.ResponseWriter, r *http.Request) {
	collectionName := chi.URLParam(r, "collection")
	id := chi.URLParam(r, "id")

	if collectionName == "" {
		writeError(w, &BadRequestError{Message: "collection name is required"})
		return
	}
	if id == "" {
		writeError(w, &BadRequestError{Message: "document ID is required"})
		return
	}

	coll, err := h.getCollection(collectionName)
	if err != nil {
		writeError(w, err)
		return
	}

	filter := map[string]interface{}{
		"_id": id,
	}

	doc, err := coll.FindOne(filter)
	if err != nil {
		writeError(w, &InternalError{Message: err.Error()})
		return
	}

	if doc == nil {
		writeError(w, &DocumentNotFoundError{ID: id})
		return
	}

	writeSuccess(w, doc.ToMap())
}

// UpdateDocument updates a document using update operators
func (h *Handlers) UpdateDocument(w http.ResponseWriter, r *http.Request) {
	collectionName := chi.URLParam(r, "collection")
	id := chi.URLParam(r, "id")

	if collectionName == "" {
		writeError(w, &BadRequestError{Message: "collection name is required"})
		return
	}
	if id == "" {
		writeError(w, &BadRequestError{Message: "document ID is required"})
		return
	}

	coll, err := h.getCollection(collectionName)
	if err != nil {
		writeError(w, err)
		return
	}

	var update map[string]interface{}
	if err := parseJSONBody(r, &update); err != nil {
		writeError(w, err)
		return
	}

	filter := map[string]interface{}{
		"_id": id,
	}

	if err := coll.UpdateOne(filter, update); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, &DocumentNotFoundError{ID: id})
		} else {
			writeError(w, &InternalError{Message: err.Error()})
		}
		return
	}

	result := map[string]interface{}{
		"id":         id,
		"collection": collectionName,
	}
	writeSuccess(w, result)
}

// DeleteDocument deletes a document by ID
func (h *Handlers) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	collectionName := chi.URLParam(r, "collection")
	id := chi.URLParam(r, "id")

	if collectionName == "" {
		writeError(w, &BadRequestError{Message: "collection name is required"})
		return
	}
	if id == "" {
		writeError(w, &BadRequestError{Message: "document ID is required"})
		return
	}

	coll, err := h.getCollection(collectionName)
	if err != nil {
		writeError(w, err)
		return
	}

	filter := map[string]interface{}{
		"_id": id,
	}

	if err := coll.DeleteOne(filter); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, &DocumentNotFoundError{ID: id})
		} else {
			writeError(w, &InternalError{Message: err.Error()})
		}
		return
	}

	result := map[string]interface{}{
		"id":         id,
		"collection": collectionName,
	}
	writeSuccess(w, result)
}

// BulkInsert inserts multiple documents at once
func (h *Handlers) BulkInsert(w http.ResponseWriter, r *http.Request) {
	collectionName := chi.URLParam(r, "collection")
	if collectionName == "" {
		writeError(w, &BadRequestError{Message: "collection name is required"})
		return
	}

	coll := h.db.Collection(collectionName)

	var docs []map[string]interface{}
	if err := parseJSONBody(r, &docs); err != nil {
		writeError(w, err)
		return
	}

	if len(docs) == 0 {
		writeError(w, &BadRequestError{Message: "no documents provided"})
		return
	}

	ids, err := coll.InsertMany(docs)
	if err != nil {
		writeError(w, &InternalError{Message: err.Error()})
		return
	}

	result := map[string]interface{}{
		"ids":        ids,
		"collection": collectionName,
	}
	writeSuccessWithCount(w, result, len(ids))
}

// BulkWrite performs multiple insert, update, and delete operations in a single request
func (h *Handlers) BulkWrite(w http.ResponseWriter, r *http.Request) {
	collectionName := chi.URLParam(r, "collection")
	if collectionName == "" {
		writeError(w, &BadRequestError{Message: "collection name is required"})
		return
	}

	coll := h.db.Collection(collectionName)

	var request struct {
		Operations []struct {
			Type     string                 `json:"type"`
			Document map[string]interface{} `json:"document,omitempty"`
			Filter   map[string]interface{} `json:"filter,omitempty"`
			Update   map[string]interface{} `json:"update,omitempty"`
		} `json:"operations"`
		Ordered bool `json:"ordered"` // default false, but we'll set to true if not specified
	}

	if err := parseJSONBody(r, &request); err != nil {
		writeError(w, err)
		return
	}

	if len(request.Operations) == 0 {
		writeError(w, &BadRequestError{Message: "no operations provided"})
		return
	}

	// Convert request operations to database.BulkOperation
	operations := make([]database.BulkOperation, len(request.Operations))
	for i, op := range request.Operations {
		operations[i] = database.BulkOperation{
			Type:     op.Type,
			Document: op.Document,
			Filter:   op.Filter,
			Update:   op.Update,
		}
	}

	// Default to ordered=true if not specified (MongoDB behavior)
	ordered := true
	if r.URL.Query().Get("ordered") == "false" {
		ordered = false
	}

	result, err := coll.BulkWrite(operations, ordered)
	if err != nil {
		// If there are partial results, include them in the response
		if result != nil {
			response := map[string]interface{}{
				"ok":            false,
				"error":         "BulkWriteError",
				"message":       err.Error(),
				"code":          http.StatusMultiStatus,
				"insertedCount": result.InsertedCount,
				"modifiedCount": result.ModifiedCount,
				"deletedCount":  result.DeletedCount,
				"insertedIds":   result.InsertedIds,
				"errors":        result.Errors,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMultiStatus)
			json.NewEncoder(w).Encode(response)
			return
		}
		writeError(w, &InternalError{Message: err.Error()})
		return
	}

	writeSuccess(w, result)
}
