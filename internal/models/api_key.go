package models

import (
	"time"

	"github.com/google/uuid"
)

// APIKey represents a long-lived API key for programmatic access.
type APIKey struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Name      string
	KeyHash   string
	ExpiresAt *time.Time
	CreatedAt time.Time
}

// RefreshToken represents a short-lived refresh token issued alongside a JWT access token.
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}
