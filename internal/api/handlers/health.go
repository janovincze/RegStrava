package handlers

import (
	"net/http"
	"time"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

// Health handles GET /health
func Health(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   "1.0.0",
	}

	respondJSON(w, http.StatusOK, response)
}

// Ready handles GET /ready (for kubernetes readiness probe)
func Ready(w http.ResponseWriter, r *http.Request) {
	// TODO: Add actual readiness checks (DB connection, Redis connection, etc.)
	respondJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
