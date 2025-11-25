package handlers

import (
	"net/http"
	"time"
)

// Health returns a health check handler
func (h *Handlers) Health(startTime time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uptime := time.Since(startTime)
		result := map[string]interface{}{
			"status": "healthy",
			"uptime": uptime.String(),
			"time":   time.Now().Format(time.RFC3339),
		}
		writeSuccess(w, result)
	}
}

// GetDatabaseStats returns comprehensive database statistics
func (h *Handlers) GetDatabaseStats(w http.ResponseWriter, r *http.Request) {
	stats := h.db.Stats()
	writeSuccess(w, stats)
}

// ListCollections returns a list of all collections
func (h *Handlers) ListCollections(w http.ResponseWriter, r *http.Request) {
	collections := h.db.ListCollections()
	result := map[string]interface{}{
		"collections": collections,
	}
	writeSuccess(w, result)
}
