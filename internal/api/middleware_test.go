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
	h := newTestHandlers()
	mw := h.authMiddleware()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with invalid key format")
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer fb_"+strings.Repeat("a", 60))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestAPIKeyFormat_TooLong(t *testing.T) {
	t.Parallel()
	h := newTestHandlers()
	mw := h.authMiddleware()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with invalid key format")
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer fb_"+strings.Repeat("a", 68))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestAPIKeyFormat_NoPrefix(t *testing.T) {
	t.Parallel()
	h := newTestHandlers()
	mw := h.authMiddleware()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with invalid key format")
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ak_"+strings.Repeat("a", 64))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	t.Parallel()
	h := newTestHandlers()
	mw := h.authMiddleware()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called without auth header")
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	t.Parallel()
	h := newTestHandlers()
	mw := h.authMiddleware()

	tests := []struct {
		name   string
		header string
	}{
		{"no space", "Bearerabc"},
		{"wrong scheme", "Basic abc"},
		{"empty parts", " "},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("handler should not be called with invalid header")
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tt.header)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", rr.Code)
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
