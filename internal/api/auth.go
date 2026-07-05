package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/youruser/fortbyte/internal/models"
	"github.com/youruser/fortbyte/internal/repository"
)

// ponytail: access token TTL, refresh token TTL — make configurable when needed.
const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

// authRequest is the shared login/register request body.
type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// tokenResponse is the shared JWT + refresh token response body.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// refreshRequest is the body for token refresh.
type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// generateRefreshToken creates a cryptographically random refresh token (hex-encoded).
func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// hashToken returns the SHA-256 hex hash of a token string for database storage/lookup.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// generateAccessToken creates a signed JWT access token for the given user.
func generateAccessToken(userID uuid.UUID, jwtSecret []byte) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

// issueTokenPair creates an access + refresh token pair for a user and stores the refresh token.
func (h *Handlers) issueTokenPair(ctx context.Context, userID uuid.UUID) (*tokenResponse, error) {
	accessToken, err := generateAccessToken(userID, h.JWTSecret)
	if err != nil {
		return nil, err
	}

	rawRefresh, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}
	refreshHash := hashToken(rawRefresh)
	if _, err := h.Refresh.Create(ctx, userID, refreshHash, time.Now().Add(refreshTokenTTL)); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    int(accessTokenTTL.Seconds()),
	}, nil
}

// decodeJSONBody reads and decodes a JSON request body (max 1 MB).
func decodeJSONBody(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}

// Register handles POST /api/v1/auth/register.
func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	if !emailRegex.MatchString(req.Email) {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid email format")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "bad_request", "password must be at least 8 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		slog.Error("bcrypt hash failed", "error", err)
		writeError(w, http.StatusInternalServerError, "server_error", "internal server error")
		return
	}

	user, err := h.Users.Create(r.Context(), &models.User{
		Email:        req.Email,
		PasswordHash: string(hash),
	})
	if err != nil {
		if errors.Is(err, repository.ErrEmailExists) {
			writeError(w, http.StatusConflict, "conflict", "email already registered")
			return
		}
		slog.Error("create user failed", "error", err)
		writeError(w, http.StatusInternalServerError, "server_error", "internal server error")
		return
	}

	resp, err := h.issueTokenPair(r.Context(), user.ID)
	if err != nil {
		slog.Error("issue token pair failed", "error", err)
		writeError(w, http.StatusInternalServerError, "server_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// Login handles POST /api/v1/auth/login.
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	user, err := h.Users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		// Uniform error: don't distinguish wrong email from wrong password.
		writeError(w, http.StatusUnauthorized, "auth_error", "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "auth_error", "invalid credentials")
		return
	}

	resp, err := h.issueTokenPair(r.Context(), user.ID)
	if err != nil {
		slog.Error("issue token pair failed", "error", err)
		writeError(w, http.StatusInternalServerError, "server_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// RefreshTokens handles POST /api/v1/auth/refresh.
func (h *Handlers) RefreshTokens(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "refresh_token is required")
		return
	}

	tokenHash := hashToken(req.RefreshToken)
	existing, err := h.Refresh.GetByTokenHash(r.Context(), tokenHash)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "auth_error", "invalid refresh token")
		return
	}

	if existing.RevokedAt != nil {
		// ponytail: revoked token reuse detection — log and reject. In production, revoke all tokens for the user.
		slog.Warn("attempted reuse of revoked refresh token", "user_id", existing.UserID)
		writeError(w, http.StatusUnauthorized, "auth_error", "invalid refresh token")
		return
	}

	if time.Now().After(existing.ExpiresAt) {
		writeError(w, http.StatusUnauthorized, "auth_error", "refresh token expired")
		return
	}

	// Rotate: revoke old token, issue new pair.
	if err := h.Refresh.Revoke(r.Context(), existing.ID); err != nil {
		slog.Error("revoke refresh token failed", "error", err)
		writeError(w, http.StatusInternalServerError, "server_error", "internal server error")
		return
	}

	accessToken, err := generateAccessToken(existing.UserID, h.JWTSecret)
	if err != nil {
		slog.Error("generate access token failed", "error", err)
		writeError(w, http.StatusInternalServerError, "server_error", "internal server error")
		return
	}

	rawRefresh, err := generateRefreshToken()
	if err != nil {
		slog.Error("generate refresh token failed", "error", err)
		writeError(w, http.StatusInternalServerError, "server_error", "internal server error")
		return
	}
	newHash := hashToken(rawRefresh)
	if _, err := h.Refresh.Create(r.Context(), existing.UserID, newHash, time.Now().Add(refreshTokenTTL)); err != nil {
		slog.Error("store new refresh token failed", "error", err)
		writeError(w, http.StatusInternalServerError, "server_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, &tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    int(accessTokenTTL.Seconds()),
	})
}

// Logout handles POST /api/v1/auth/logout. Requires auth middleware.
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r)
	if userID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "auth_error", "authentication required")
		return
	}

	if err := h.Refresh.RevokeAllForUser(r.Context(), userID); err != nil {
		slog.Error("revoke all refresh tokens failed", "error", err)
		writeError(w, http.StatusInternalServerError, "server_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}
