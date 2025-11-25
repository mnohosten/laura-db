package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/query"
)

// SearchRequest represents a search request
type SearchRequest struct {
	Filter     map[string]interface{}   `json:"filter"`
	Projection map[string]bool          `json:"projection"`
	Sort       []map[string]interface{} `json:"sort"`
	Limit      int                      `json:"limit"`
	Skip       int                      `json:"skip"`
}

// CountRequest represents a count request
type CountRequest struct {
	Filter map[string]interface{} `json:"filter"`
}

// SearchDocuments searches documents with filters, projection, sorting, and pagination
func (h *Handlers) SearchDocuments(w http.ResponseWriter, r *http.Request) {
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

	var req SearchRequest
	if err := parseJSONBody(r, &req); err != nil {
		writeError(w, err)
		return
	}

	// Build query options
	opts := &database.QueryOptions{
		Projection: req.Projection,
		Limit:      req.Limit,
		Skip:       req.Skip,
	}

	// Parse sort options
	if len(req.Sort) > 0 {
		opts.Sort = make([]query.SortField, 0, len(req.Sort))
		for _, s := range req.Sort {
			if field, ok := s["field"].(string); ok {
				ascending := true
				if orderStr, ok := s["order"].(string); ok && orderStr == "desc" {
					ascending = false
				}
				opts.Sort = append(opts.Sort, query.SortField{
					Field:     field,
					Ascending: ascending,
				})
			}
		}
	}

	// Execute query
	filter := req.Filter
	if filter == nil {
		filter = map[string]interface{}{}
	}

	docs, err := coll.FindWithOptions(filter, opts)
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

// CountDocuments counts all documents in a collection
func (h *Handlers) CountDocuments(w http.ResponseWriter, r *http.Request) {
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

	count, err := coll.Count(map[string]interface{}{})
	if err != nil {
		writeError(w, &InternalError{Message: err.Error()})
		return
	}

	result := map[string]interface{}{
		"collection": collectionName,
		"count":      count,
	}
	writeSuccess(w, result)
}

// CountDocumentsWithFilter counts documents matching a filter
func (h *Handlers) CountDocumentsWithFilter(w http.ResponseWriter, r *http.Request) {
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

	var req CountRequest
	if err := parseJSONBody(r, &req); err != nil {
		writeError(w, err)
		return
	}

	filter := req.Filter
	if filter == nil {
		filter = map[string]interface{}{}
	}

	count, err := coll.Count(filter)
	if err != nil {
		writeError(w, &InternalError{Message: err.Error()})
		return
	}

	result := map[string]interface{}{
		"collection": collectionName,
		"count":      count,
	}
	writeSuccess(w, result)
}
