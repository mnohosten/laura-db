package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// CreateCollection creates a new collection
func (h *Handlers) CreateCollection(w http.ResponseWriter, r *http.Request) {
	collectionName := chi.URLParam(r, "collection")
	if collectionName == "" {
		writeError(w, &BadRequestError{Message: "collection name is required"})
		return
	}

	// Create collection (Collection method creates it if it doesn't exist)
	h.db.Collection(collectionName)

	result := map[string]interface{}{
		"collection": collectionName,
	}
	writeSuccess(w, result)
}

// DropCollection deletes a collection and all its documents
func (h *Handlers) DropCollection(w http.ResponseWriter, r *http.Request) {
	collectionName := chi.URLParam(r, "collection")
	if collectionName == "" {
		writeError(w, &BadRequestError{Message: "collection name is required"})
		return
	}

	if err := h.db.DropCollection(collectionName); err != nil {
		writeError(w, &InternalError{Message: err.Error()})
		return
	}

	result := map[string]interface{}{
		"collection": collectionName,
	}
	writeSuccess(w, result)
}

// GetCollectionStats returns statistics for a specific collection
func (h *Handlers) GetCollectionStats(w http.ResponseWriter, r *http.Request) {
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

	stats := coll.Stats()
	writeSuccess(w, stats)
}
