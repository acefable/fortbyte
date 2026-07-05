package api

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/youruser/fortbyte/internal/models"
)

// UserRepository defines the database operations needed by auth handlers.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
}

// APIKeyRepository defines the database operations needed by API key handlers.
type APIKeyRepository interface {
	Create(ctx context.Context, userID uuid.UUID, name, keyHash string, expiresAt *time.Time) (*models.APIKey, error)
	GetByKeyHash(ctx context.Context, keyHash string) (*models.APIKey, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

// RefreshTokenRepository defines the database operations needed by token handlers.
type RefreshTokenRepository interface {
	Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*models.RefreshToken, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

// Handlers holds repository dependencies for all API endpoint handlers.
type Handlers struct {
	Users     UserRepository
	APIKeys   APIKeyRepository
	Refresh   RefreshTokenRepository
	JWTSecret []byte
}
