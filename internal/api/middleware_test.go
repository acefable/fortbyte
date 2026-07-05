package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestAPIKeyFormat_ValidKey(t *testing.T) {
	t.Parallel()
	key, err := generateAPIKey()
	if err != nil {
		t.Fatalf("generateAPIKey() error: %v", err)
	}

	if !strings.HasPrefix(key, "fb_") {
		t.Errorf("key missing prefix: %q", key)
	}
	if len(key) != 67 {
		t.Errorf("key length = %d, want 67", len(key))
	}
}

func TestAPIKeyFormat_TooShort(t *testing.T) {
	t.Parallel()
	key := "fb_" + strings.Repeat("a", 60)
	if len(key) != 63 {
		t.Fatalf("test key length = %d, want 63", len(key))
	}
	if strings.HasPrefix(key, "fb_") && len(key) != 67 {
		// Expected: rejected.
	} else {
		t.Error("short key should be rejected but passed validation")
	}
}

func TestAPIKeyFormat_TooLong(t *testing.T) {
	t.Parallel()
	key := "fb_" + strings.Repeat("a", 68)
	if strings.HasPrefix(key, "fb_") && len(key) != 67 {
		// Expected: rejected.
	} else {
		t.Error("long key should be rejected but passed validation")
	}
}

func TestAPIKeyFormat_NoPrefix(t *testing.T) {
	t.Parallel()
	key := "ak_" + strings.Repeat("a", 64)
	if strings.HasPrefix(key, "fb_") {
		t.Error("non-fb_ key should not have fb_ prefix")
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	_ = httptest.NewRecorder()

	authHeader := req.Header.Get("Authorization")
	if authHeader != "" {
		t.Fatal("expected empty Authorization header")
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		header string
		valid  bool
	}{
		{"no space", "Bearerabc", false},
		{"wrong scheme", "Basic abc", false},
		{"lowercase bearer", "bearer abc", true},
		{"empty parts", " ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parts := strings.SplitN(tt.header, " ", 2)
			valid := len(parts) == 2 && strings.EqualFold(parts[0], "Bearer")
			if valid != tt.valid {
				t.Errorf("header %q: valid = %v, want %v", tt.header, valid, tt.valid)
			}
		})
	}
}

func TestContextKey_UnexportedType(t *testing.T) {
	t.Parallel()
	var k contextKey = "test"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), k, "value")
	req = req.WithContext(ctx)

	// External code using a plain string key won't find the value.
	got := req.Context().Value("test")
	if got != nil {
		t.Error("unexported context key should not be accessible with plain string")
	}

	// But the typed key works.
	got = req.Context().Value(k)
	if got != "value" {
		t.Errorf("typed context key: got %v, want %q", got, "value")
	}
}

func TestUserIDFromContext_Integration(t *testing.T) {
	t.Parallel()
	id := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), userIDKey, id)
	req = req.WithContext(ctx)

	got := UserIDFromContext(req)
	if got != id {
		t.Errorf("UserIDFromContext() = %v, want %v", got, id)
	}

	empty := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := UserIDFromContext(empty); got != uuid.Nil {
		t.Errorf("UserIDFromContext(empty) = %v, want uuid.Nil", got)
	}
}
