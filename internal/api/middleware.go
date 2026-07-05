package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const userIDKey contextKey = "user_id"

// UserIDFromContext extracts the authenticated user ID from the request context.
// Returns uuid.Nil if not present.
func UserIDFromContext(r *http.Request) uuid.UUID {
	id, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}

// authMiddleware validates JWT or API key Bearer tokens and injects the user ID into the context.
func (h *Handlers) authMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, http.StatusUnauthorized, "auth_error", "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeError(w, http.StatusUnauthorized, "auth_error", "invalid authorization format")
				return
			}
			token := parts[1]

			if strings.HasPrefix(token, "fb_") {
				h.authenticateAPIKey(w, r, token)
			} else {
				h.authenticateJWT(w, r, token)
			}

			if UserIDFromContext(r) == uuid.Nil {
				return // authenticate* already wrote the error response
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (h *Handlers) authenticateJWT(w http.ResponseWriter, r *http.Request, tokenString string) {
	claims := &jwt.RegisteredClaims{}
	parsed, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return h.JWTSecret, nil
	})
	if err != nil || !parsed.Valid {
		writeError(w, http.StatusUnauthorized, "auth_error", "invalid or expired token")
		return
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "auth_error", "invalid token subject")
		return
	}

	ctx := context.WithValue(r.Context(), userIDKey, userID)
	*r = *r.WithContext(ctx)
}

func (h *Handlers) authenticateAPIKey(w http.ResponseWriter, r *http.Request, rawKey string) {
	// ponytail: SHA-256 hash for deterministic lookup (bcrypt salts differ per hash).
	// API key format: fb_<64-char-hex>
	// fb_ prefix (3) + 64 hex chars = 67 total.
	if !strings.HasPrefix(rawKey, "fb_") || len(rawKey) != 67 {
		writeError(w, http.StatusUnauthorized, "auth_error", "invalid api key format")
		return
	}

	// Hash the raw key with SHA-256 for lookup (bcrypt would be too slow).
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	key, err := h.APIKeys.GetByKeyHash(r.Context(), keyHash)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "auth_error", "invalid api key")
		return
	}

	if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
		writeError(w, http.StatusUnauthorized, "auth_error", "api key expired")
		return
	}

	ctx := context.WithValue(r.Context(), userIDKey, key.UserID)
	*r = *r.WithContext(ctx)
}
