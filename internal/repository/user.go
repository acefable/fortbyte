package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/youruser/fortbyte/internal/models"
)

// Sentinel errors for user repository operations.
var (
	ErrNotFound    = errors.New("user not found")
	ErrEmailExists = errors.New("email already exists")
)

// UserRepository provides database operations for users.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a UserRepository backed by the given pool.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// normalizeEmail lowercases and trims the email for consistent storage and lookup.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// scanUser extracts a User from a pgx Row. Shared by GetByID and GetByEmail.
func scanUser(row pgx.Row) (*models.User, error) {
	user := &models.User{}
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

// Create inserts a new user and returns it with the generated ID and timestamps.
func (r *UserRepository) Create(ctx context.Context, user *models.User) (*models.User, error) {
	user.ID = uuid.New()
	user.Email = normalizeEmail(user.Email)

	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)
		 RETURNING created_at, updated_at`,
		user.ID, user.Email, user.PasswordHash,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		// ponytail: pg unique violation code is "23505" — check for email constraint
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, fmt.Errorf("insert user: %w", ErrEmailExists)
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return user, nil
}

// GetByID retrieves a user by their UUID.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, err := scanUser(r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at, updated_at FROM users WHERE id = $1`,
		id,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get user by id: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
}

// GetByEmail retrieves a user by their email address.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := scanUser(r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at, updated_at FROM users WHERE email = $1`,
		normalizeEmail(email),
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get user by email: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return user, nil
}
