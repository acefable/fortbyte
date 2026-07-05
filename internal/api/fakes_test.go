package api

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/youruser/fortbyte/internal/models"
	"github.com/youruser/fortbyte/internal/repository"
)

type fakeUserRepo struct {
	users map[string]*models.User // keyed by email
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: make(map[string]*models.User)}
}

func (f *fakeUserRepo) Create(ctx context.Context, user *models.User) (*models.User, error) {
	if _, ok := f.users[user.Email]; ok {
		return nil, fmt.Errorf("insert user: %w", repository.ErrEmailExists)
	}
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	f.users[user.Email] = user
	return user, nil
}

func (f *fakeUserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	u, ok := f.users[email]
	if !ok {
		return nil, fmt.Errorf("get user by email: %w", repository.ErrNotFound)
	}
	return u, nil
}

type fakeAPIKeyRepo struct {
	keys map[string]*models.APIKey // keyed by key_hash
}

func newFakeAPIKeyRepo() *fakeAPIKeyRepo {
	return &fakeAPIKeyRepo{keys: make(map[string]*models.APIKey)}
}

func (f *fakeAPIKeyRepo) Create(ctx context.Context, userID uuid.UUID, name, keyHash string, expiresAt *time.Time) (*models.APIKey, error) {
	key := &models.APIKey{
		ID: uuid.New(), UserID: userID, Name: name,
		KeyHash: keyHash, ExpiresAt: expiresAt, CreatedAt: time.Now(),
	}
	f.keys[keyHash] = key
	return key, nil
}

func (f *fakeAPIKeyRepo) GetByKeyHash(ctx context.Context, keyHash string) (*models.APIKey, error) {
	k, ok := f.keys[keyHash]
	if !ok {
		return nil, fmt.Errorf("get api key: %w", repository.ErrKeyNotFound)
	}
	return k, nil
}

func (f *fakeAPIKeyRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	for hash, k := range f.keys {
		if k.ID == id && k.UserID == userID {
			delete(f.keys, hash)
			return nil
		}
	}
	return fmt.Errorf("delete api key: %w", repository.ErrKeyNotFound)
}

type fakeRefreshTokenRepo struct {
	tokens map[string]*models.RefreshToken // keyed by token_hash
}

func newFakeRefreshTokenRepo() *fakeRefreshTokenRepo {
	return &fakeRefreshTokenRepo{tokens: make(map[string]*models.RefreshToken)}
}

func (f *fakeRefreshTokenRepo) Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*models.RefreshToken, error) {
	t := &models.RefreshToken{
		ID: uuid.New(), UserID: userID, TokenHash: tokenHash,
		ExpiresAt: expiresAt, CreatedAt: time.Now(),
	}
	f.tokens[tokenHash] = t
	return t, nil
}

func (f *fakeRefreshTokenRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	t, ok := f.tokens[tokenHash]
	if !ok {
		return nil, fmt.Errorf("get refresh token: %w", repository.ErrNotFound)
	}
	return t, nil
}

func (f *fakeRefreshTokenRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	for _, t := range f.tokens {
		if t.ID == id {
			now := time.Now()
			t.RevokedAt = &now
			return nil
		}
	}
	return nil
}

func (f *fakeRefreshTokenRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	for _, t := range f.tokens {
		if t.UserID == userID && t.RevokedAt == nil {
			t.RevokedAt = &now
		}
	}
	return nil
}
