package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/youruser/fortbyte/internal/repository"
)

// createAPIKeyRequest is the request body for creating an API key.
type createAPIKeyRequest struct {
	Name          string `json:"name"`
	ExpiresInDays *int   `json:"expires_in_days,omitempty"`
}

// createAPIKeyResponse is the response after creating an API key.
type createAPIKeyResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Key       string    `json:"key"`
	CreatedAt time.Time `json:"created_at"`
}

// generateAPIKey creates a random API key with fb_ prefix: fb_ + 64 hex chars = 66 total.
func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate api key: %w", err)
	}
	return "fb_" + hex.EncodeToString(b), nil
}

// createAPIKeyHandler handles POST /api/v1/auth/api-keys. Requires auth middleware.
func createAPIKeyHandler(db *pgxpool.Pool) http.HandlerFunc {
	apiKeyRepo := repository.NewAPIKeyRepository(db)

	return func(w http.ResponseWriter, r *http.Request) {
		userID := UserIDFromContext(r)
		if userID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "auth_error", "authentication required")
			return
		}

		var req createAPIKeyRequest
		if err := decodeJSONBody(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
			return
		}
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "name is required")
			return
		}

		rawKey, err := generateAPIKey()
		if err != nil {
			slog.Error("generate api key failed", "error", err)
			writeError(w, http.StatusInternalServerError, "server_error", "internal server error")
			return
		}

		// ponytail: SHA-256 instead of bcrypt for API key hashing — needed for
		// deterministic lookup in auth middleware (bcrypt salts differ per hash).
		// The raw key has 264 bits of entropy; SHA-256 is sufficient.
		hash := sha256.Sum256([]byte(rawKey))
		keyHash := hex.EncodeToString(hash[:])

		var expiresAt *time.Time
		if req.ExpiresInDays != nil && *req.ExpiresInDays > 0 {
			t := time.Now().AddDate(0, 0, *req.ExpiresInDays)
			expiresAt = &t
		}

		key, err := apiKeyRepo.Create(r.Context(), userID, req.Name, keyHash, expiresAt)
		if err != nil {
			slog.Error("create api key failed", "error", err)
			writeError(w, http.StatusInternalServerError, "server_error", "internal server error")
			return
		}

		writeJSON(w, http.StatusCreated, createAPIKeyResponse{
			ID:        key.ID,
			Name:      key.Name,
			Key:       rawKey,
			CreatedAt: key.CreatedAt,
		})
	}
}

// deleteAPIKeyHandler handles DELETE /api/v1/auth/api-keys/{keyID}. Requires auth middleware.
func deleteAPIKeyHandler(db *pgxpool.Pool) http.HandlerFunc {
	apiKeyRepo := repository.NewAPIKeyRepository(db)

	return func(w http.ResponseWriter, r *http.Request) {
		userID := UserIDFromContext(r)
		if userID == uuid.Nil {
			writeError(w, http.StatusUnauthorized, "auth_error", "authentication required")
			return
		}

		keyID, err := uuid.Parse(chi.URLParam(r, "keyID"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid key id")
			return
		}

		if err := apiKeyRepo.Delete(r.Context(), keyID, userID); err != nil {
			if errors.Is(err, repository.ErrKeyNotFound) {
				writeError(w, http.StatusNotFound, "not_found", "api key not found")
				return
			}
			slog.Error("delete api key failed", "error", err)
			writeError(w, http.StatusInternalServerError, "server_error", "internal server error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}
