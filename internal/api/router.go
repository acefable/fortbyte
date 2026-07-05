// Package api provides the HTTP router and handlers for the fortbyte hosted API.
package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewRouter creates a chi Mux with all API routes mounted under /api/v1/.
func NewRouter(db *pgxpool.Pool, jwtSecret []byte) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	// ponytail: add logging, request ID, CORS middleware here later

	r.Route("/api/v1", func(r chi.Router) {
		// Public
		r.Get("/health", healthHandler(db))
		r.Get("/ready", readyHandler())

		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", registerHandler(db, jwtSecret))
			r.Post("/login", loginHandler(db, jwtSecret))
			r.Post("/refresh", refreshHandler(db, jwtSecret))

			// Protected
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware(db, jwtSecret))
				r.Post("/logout", logoutHandler(db))
				r.Post("/api-keys", createAPIKeyHandler(db))
				r.Delete("/api-keys/{keyID}", deleteAPIKeyHandler(db))
			})
		})
	})

	return r
}
