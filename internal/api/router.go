// Package api provides the HTTP router and handlers for the fortbyte hosted API.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/youruser/fortbyte/internal/repository"
)

// NewRouter creates a chi Mux with all API routes mounted under /api/v1/.
func NewRouter(db *pgxpool.Pool) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", healthHandler(db))
	})

	return r
}

func healthHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		dbStatus := "up"
		status := "ok"
		if err := repository.Ping(ctx, db); err != nil {
			dbStatus = "down"
			status = "unhealthy"
		}

		resp := map[string]string{
			"status": status,
			"db":     dbStatus,
		}

		w.Header().Set("Content-Type", "application/json")
		if dbStatus == "down" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			slog.Error("health encode failed", "error", err)
		}
	}
}
