package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/youruser/fortbyte/internal/models"
)

// Sentinel errors for auth repository operations.
var (
	ErrKeyNotFound          = errors.New("api key not found")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)

// APIKeyRepository provides database operations for API keys.
type APIKeyRepository struct {
	pool *pgxpool.Pool
}

// NewAPIKeyRepository creates an APIKeyRepository backed by the given pool.
func NewAPIKeyRepository(pool *pgxpool.Pool) *APIKeyRepository {
	return &APIKeyRepository{pool: pool}
}

// Create inserts a new API key and returns it with the generated ID and timestamp.
func (r *APIKeyRepository) Create(ctx context.Context, userID uuid.UUID, name, keyHash string, expiresAt *time.Time) (*models.APIKey, error) {
	key := &models.APIKey{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      name,
		KeyHash:   keyHash,
		ExpiresAt: expiresAt,
	}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO api_keys (id, user_id, name, key_hash, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())
		 RETURNING created_at`,
		key.ID, key.UserID, key.Name, key.KeyHash, key.ExpiresAt,
	).Scan(&key.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert api key: %w", err)
	}
	return key, nil
}

// GetByKeyHash retrieves an API key by its hash for auth middleware lookup.
func (r *APIKeyRepository) GetByKeyHash(ctx context.Context, keyHash string) (*models.APIKey, error) {
	key := &models.APIKey{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, key_hash, expires_at, created_at
		 FROM api_keys WHERE key_hash = $1`,
		keyHash,
	).Scan(&key.ID, &key.UserID, &key.Name, &key.KeyHash, &key.ExpiresAt, &key.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get api key by hash: %w", ErrKeyNotFound)
		}
		return nil, fmt.Errorf("get api key by hash: %w", err)
	}
	// ponytail: expiry check deferred to middleware
	return key, nil
}

// Delete removes an API key, ensuring it belongs to the specified user.
func (r *APIKeyRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM api_keys WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete api key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete api key: %w", ErrKeyNotFound)
	}
	return nil
}

// DeleteAllForUser removes all API keys belonging to a user.
func (r *APIKeyRepository) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM api_keys WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("delete all api keys for user: %w", err)
	}
	return nil
}

// RefreshTokenRepository provides database operations for refresh tokens.
type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

// NewRefreshTokenRepository creates a RefreshTokenRepository backed by the given pool.
func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

// Create inserts a new refresh token and returns it with generated ID and timestamp.
func (r *RefreshTokenRepository) Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*models.RefreshToken, error) {
	token := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 RETURNING created_at`,
		token.ID, token.UserID, token.TokenHash, token.ExpiresAt,
	).Scan(&token.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert refresh token: %w", err)
	}
	return token, nil
}

// GetByTokenHash retrieves a refresh token by its SHA-256 hash.
func (r *RefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	token := &models.RefreshToken{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		 FROM refresh_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.RevokedAt, &token.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get refresh token by hash: %w", ErrRefreshTokenNotFound)
		}
		return nil, fmt.Errorf("get refresh token by hash: %w", err)
	}
	return token, nil
}

// Revoke sets the revoked_at timestamp for a refresh token.
func (r *RefreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

// RevokeAllForUser revokes all refresh tokens belonging to a user.
func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("revoke all refresh tokens for user: %w", err)
	}
	return nil
}
