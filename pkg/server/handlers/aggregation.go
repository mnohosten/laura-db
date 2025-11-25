package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// AggregateRequest represents an aggregation request
type AggregateRequest struct {
	Pipeline []map[string]interface{} `json:"pipeline"`
}

// Aggregate executes an aggregation pipeline
func (h *Handlers) Aggregate(w http.ResponseWriter, r *http.Request) {
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

	var req AggregateRequest
	if err := parseJSONBody(r, &req); err != nil {
		writeError(w, err)
		return
	}

	if len(req.Pipeline) == 0 {
		writeError(w, &BadRequestError{Message: "pipeline is required"})
		return
	}

	docs, err := coll.Aggregate(req.Pipeline)
	if err != nil {
		writeError(w, &InternalError{Message: err.Error()})
		return
	}

	// Convert documents to maps for JSON serialization
	results := make([]map[string]interface{}, len(docs))
	for i, doc := range docs {
		results[i] = doc.ToMap()
	}

	writeSuccessWithCount(w, results, len(results))
}
