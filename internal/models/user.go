package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a tenant in the hosted secrets manager.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string `json:"-"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
