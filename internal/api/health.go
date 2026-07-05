package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/youruser/fortbyte/internal/models"
	"github.com/youruser/fortbyte/internal/repository"
)

// writeJSON encodes v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write json failed", "error", err)
	}
}

// writeError writes a standard API error response.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, models.ErrorResponse{Error: models.ErrorDetail{Code: code, Message: message}})
}

// healthHandler returns the server health status including DB connectivity.
func healthHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		dbStatus := "up"
		statusStr := "ok"
		if err := repository.Ping(ctx, db); err != nil {
			dbStatus = "down"
			statusStr = "unhealthy"
		}

		status := http.StatusOK
		if dbStatus == "down" {
			status = http.StatusServiceUnavailable
		}
		writeJSON(w, status, map[string]string{
			"status": statusStr,
			"db":     dbStatus,
		})
	}
}

// readyHandler returns a simple readiness check (no DB dependency).
func readyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}
