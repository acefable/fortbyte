package repository

import (
	"errors"
	"fmt"
	"testing"

	"github.com/youruser/fortbyte/internal/models"
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

func TestScanUserSignature(t *testing.T) {
	// Verify scanUser compiles with the expected signature.
	// We can't call it without a real pgx.Row, but we confirm it exists.
	_ = scanUser      // compile-time reference
	_ = models.User{} // ensure model compiles
}
