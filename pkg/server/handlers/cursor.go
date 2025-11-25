package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/query"
)

// CreateCursorRequest represents a cursor creation request
type CreateCursorRequest struct {
	Collection string                 `json:"collection"`
	Filter     map[string]interface{} `json:"filter"`
	Projection map[string]bool        `json:"projection"`
	Sort       []map[string]interface{} `json:"sort"`
	Limit      int                    `json:"limit"`
	Skip       int                    `json:"skip"`
	BatchSize  int                    `json:"batchSize"`
	Timeout    string                 `json:"timeout"` // e.g., "5m", "10m"
}

// CreateCursorResponse represents a cursor creation response
type CreateCursorResponse struct {
	CursorID  string `json:"cursorId"`
	Count     int    `json:"count"`
	BatchSize int    `json:"batchSize"`
}

// FetchBatchResponse represents a batch fetch response
type FetchBatchResponse struct {
	Documents []map[string]interface{} `json:"documents"`
	Position  int                      `json:"position"`
	Remaining int                      `json:"remaining"`
	HasMore   bool                     `json:"hasMore"`
}

// CreateCursor creates a new server-side cursor
func (h *Handlers) CreateCursor(w http.ResponseWriter, r *http.Request) {
	var req CreateCursorRequest
	if err := parseJSONBody(r, &req); err != nil {
		writeError(w, err)
		return
	}

	// Validate collection name
	if req.Collection == "" {
		writeError(w, &BadRequestError{Message: "collection name is required"})
		return
	}

	// Get collection
	coll, err := h.getCollection(req.Collection)
	if err != nil {
		writeError(w, err)
		return
	}

	// Parse filter
	filter := req.Filter
	if filter == nil {
		filter = map[string]interface{}{}
	}

	// Build query using builder pattern
	q := query.NewQuery(filter)

	// Add projection if specified
	if req.Projection != nil {
		q = q.WithProjection(req.Projection)
	}

	// Parse and add sort options
	if len(req.Sort) > 0 {
		sortFields := make([]query.SortField, 0, len(req.Sort))
		for _, s := range req.Sort {
			if field, ok := s["field"].(string); ok {
				ascending := true
				if orderStr, ok := s["order"].(string); ok && orderStr == "desc" {
					ascending = false
				}
				sortFields = append(sortFields, query.SortField{
					Field:     field,
					Ascending: ascending,
				})
			}
		}
		q = q.WithSort(sortFields)
	}

	// Add limit and skip
	if req.Limit > 0 {
		q = q.WithLimit(req.Limit)
	}
	if req.Skip > 0 {
		q = q.WithSkip(req.Skip)
	}

	// Build cursor options
	cursorOpts := database.DefaultCursorOptions()
	if req.BatchSize > 0 {
		cursorOpts.BatchSize = req.BatchSize
	}
	if req.Timeout != "" {
		timeout, err := time.ParseDuration(req.Timeout)
		if err != nil {
			writeError(w, &BadRequestError{Message: "invalid timeout format: " + err.Error()})
			return
		}
		cursorOpts.Timeout = timeout
	}

	// Create cursor through cursor manager
	cursor, err := h.db.CursorManager().CreateCursor(coll, q, cursorOpts)
	if err != nil {
		writeError(w, &InternalError{Message: err.Error()})
		return
	}

	// Return cursor info
	response := CreateCursorResponse{
		CursorID:  cursor.ID(),
		Count:     cursor.Count(),
		BatchSize: cursor.BatchSize(),
	}

	writeSuccess(w, response)
}

// FetchBatch fetches the next batch of documents from a cursor
func (h *Handlers) FetchBatch(w http.ResponseWriter, r *http.Request) {
	cursorID := chi.URLParam(r, "cursorId")
	if cursorID == "" {
		writeError(w, &BadRequestError{Message: "cursor ID is required"})
		return
	}

	// Get cursor from manager
	cursor, err := h.db.CursorManager().GetCursor(cursorID)
	if err != nil {
		writeError(w, &BadRequestError{Message: "cursor not found: " + cursorID})
		return
	}

	// Fetch next batch
	var docs []map[string]interface{}
	batchSize := cursor.BatchSize()
	for i := 0; i < batchSize && cursor.HasNext(); i++ {
		doc, err := cursor.Next()
		if err != nil {
			writeError(w, &InternalError{Message: err.Error()})
			return
		}
		docs = append(docs, doc.ToMap())
	}

	// Build response
	response := FetchBatchResponse{
		Documents: docs,
		Position:  cursor.Position(),
		Remaining: cursor.Remaining(),
		HasMore:   cursor.HasNext(),
	}

	writeSuccess(w, response)
}

// CloseCursor closes and removes a cursor
func (h *Handlers) CloseCursor(w http.ResponseWriter, r *http.Request) {
	cursorID := chi.URLParam(r, "cursorId")
	if cursorID == "" {
		writeError(w, &BadRequestError{Message: "cursor ID is required"})
		return
	}

	// Close cursor through manager
	err := h.db.CursorManager().CloseCursor(cursorID)
	if err != nil {
		writeError(w, &BadRequestError{Message: "cursor not found: " + cursorID})
		return
	}

	writeSuccess(w, map[string]bool{"ok": true})
}
