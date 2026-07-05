// Package api provides the HTTP router and handlers for the fortbyte hosted API.
package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/youruser/fortbyte/internal/repository"
)

// NewRouter creates a chi Mux with all API routes mounted under /api/v1/.
func NewRouter(db *pgxpool.Pool, jwtSecret []byte) *chi.Mux {
	h := &Handlers{
		Users:     repository.NewUserRepository(db),
		APIKeys:   repository.NewAPIKeyRepository(db),
		Refresh:   repository.NewRefreshTokenRepository(db),
		JWTSecret: jwtSecret,
	}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	// ponytail: add logging, request ID, CORS middleware here later

	r.Route("/api/v1", func(r chi.Router) {
		// Public
		r.Get("/health", healthHandler(db))
		r.Get("/ready", readyHandler())

		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", h.Register)
			r.Post("/login", h.Login)
			r.Post("/refresh", h.RefreshTokens)

			// Protected
			r.Group(func(r chi.Router) {
				r.Use(h.authMiddleware())
				r.Post("/logout", h.Logout)
				r.Post("/api-keys", h.CreateAPIKey)
				r.Delete("/api-keys/{keyID}", h.DeleteAPIKey)
			})
		})
	})

	return r
}
