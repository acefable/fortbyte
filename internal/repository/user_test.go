package repository

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lowercases", "User@Example.com", "user@example.com"},
		{"trims spaces", "  user@example.com  ", "user@example.com"},
		{"lowercase+trim", "  USER@EXAMPLE.COM  ", "user@example.com"},
		{"already clean", "user@example.com", "user@example.com"},
		{"empty string", "", ""},
		{"tabs and newlines", "\tuser@example.com\n", "user@example.com"},
		{"mixed case with dots", "First.Last@Gmail.COM", "first.last@gmail.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeEmail(tt.input)
			if got != tt.want {
				t.Errorf("normalizeEmail(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSentinelErrors(t *testing.T) {
	t.Run("wrapped ErrNotFound is unwrappable", func(t *testing.T) {
		wrapped := fmt.Errorf("get user by id: %w", ErrNotFound)
		if !errors.Is(wrapped, ErrNotFound) {
			t.Error("wrapped ErrNotFound should be unwrappable with errors.Is")
		}
	})

	t.Run("wrapped ErrEmailExists is unwrappable", func(t *testing.T) {
		wrapped := fmt.Errorf("insert user: %w", ErrEmailExists)
		if !errors.Is(wrapped, ErrEmailExists) {
			t.Error("wrapped ErrEmailExists should be unwrappable with errors.Is")
		}
	})

	t.Run("different sentinel does not match", func(t *testing.T) {
		wrapped := fmt.Errorf("get user by id: %w", ErrNotFound)
		if errors.Is(wrapped, ErrEmailExists) {
			t.Error("ErrNotFound should not match ErrEmailExists")
		}
	})
}

// fakeRow implements pgx.Row for testing scanUser without a real DB.
type fakeRow struct {
	values []any
	err    error
}

func (r *fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i, d := range dest {
		if i >= len(r.values) {
			break
		}
		switch dst := d.(type) {
		case *uuid.UUID:
			*dst = r.values[i].(uuid.UUID)
		case *string:
			*dst = r.values[i].(string)
		case *time.Time:
			*dst = r.values[i].(time.Time)
		}
	}
	return nil
}

func TestScanUserFields(t *testing.T) {
	wantID := uuid.New()
	wantEmail := "test@example.com"
	wantHash := "argon2id$hash$here"
	wantCreated := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	wantUpdated := time.Date(2025, 6, 20, 14, 0, 0, 0, time.UTC)

	row := &fakeRow{
		values: []any{wantID, wantEmail, wantHash, wantCreated, wantUpdated},
	}

	got, err := scanUser(row)
	if err != nil {
		t.Fatalf("scanUser returned unexpected error: %v", err)
	}

	if got.ID != wantID {
		t.Errorf("ID = %v, want %v", got.ID, wantID)
	}
	if got.Email != wantEmail {
		t.Errorf("Email = %q, want %q", got.Email, wantEmail)
	}
	if got.PasswordHash != wantHash {
		t.Errorf("PasswordHash = %q, want %q", got.PasswordHash, wantHash)
	}
	if !got.CreatedAt.Equal(wantCreated) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, wantCreated)
	}
	if !got.UpdatedAt.Equal(wantUpdated) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, wantUpdated)
	}
}

func TestScanUserError(t *testing.T) {
	row := &fakeRow{err: fmt.Errorf("connection lost")}

	_, err := scanUser(row)
	if err == nil {
		t.Fatal("scanUser should return error from row.Scan")
	}
	if err.Error() != "connection lost" {
		t.Errorf("got error %q, want %q", err.Error(), "connection lost")
	}
}
