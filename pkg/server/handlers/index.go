package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// CreateIndexRequest represents an index creation request
type CreateIndexRequest struct {
	Field  string `json:"field"`
	Unique bool   `json:"unique"`
}

// CreateIndex creates an index on a field
func (h *Handlers) CreateIndex(w http.ResponseWriter, r *http.Request) {
	collectionName := chi.URLParam(r, "collection")
	if collectionName == "" {
		writeError(w, &BadRequestError{Message: "collection name is required"})
		return
	}

	coll, err := h.getCollection(collectionName)
	if err != nil {
		writeError(w, err)
		return
	}

	var req CreateIndexRequest
	if err := parseJSONBody(r, &req); err != nil {
		writeError(w, err)
		return
	}

	if req.Field == "" {
		writeError(w, &BadRequestError{Message: "field is required"})
		return
	}

	if err := coll.CreateIndex(req.Field, req.Unique); err != nil {
		writeError(w, &InternalError{Message: err.Error()})
		return
	}

	result := map[string]interface{}{
		"collection": collectionName,
		"field":      req.Field,
		"unique":     req.Unique,
	}
	writeSuccess(w, result)
}

// ListIndexes lists all indexes on a collection
func (h *Handlers) ListIndexes(w http.ResponseWriter, r *http.Request) {
	collectionName := chi.URLParam(r, "collection")
	if collectionName == "" {
		writeError(w, &BadRequestError{Message: "collection name is required"})
		return
	}

	coll, err := h.getCollection(collectionName)
	if err != nil {
		writeError(w, err)
		return
	}

	indexes := coll.ListIndexes()

	result := map[string]interface{}{
		"collection": collectionName,
		"indexes":    indexes,
	}
	writeSuccess(w, result)
}

// DropIndex deletes an index by name
func (h *Handlers) DropIndex(w http.ResponseWriter, r *http.Request) {
	collectionName := chi.URLParam(r, "collection")
	indexName := chi.URLParam(r, "name")

	if collectionName == "" {
		writeError(w, &BadRequestError{Message: "collection name is required"})
		return
	}
	if indexName == "" {
		writeError(w, &BadRequestError{Message: "index name is required"})
		return
	}

	coll, err := h.getCollection(collectionName)
	if err != nil {
		writeError(w, err)
		return
	}

	if err := coll.DropIndex(indexName); err != nil {
		writeError(w, &InternalError{Message: err.Error()})
		return
	}

	result := map[string]interface{}{
		"collection": collectionName,
		"index":      indexName,
	}
	writeSuccess(w, result)
}
