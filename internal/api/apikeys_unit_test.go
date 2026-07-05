package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestUnitCreateAPIKey_Success(t *testing.T) {
	h := newTestHandlers()
	user := setupTestUser(t, h, "apikey@example.com", "password123")

	body := `{"name":"ci-key"}`
	req := httptest.NewRequest(http.MethodPost, "/api-keys", strings.NewReader(body))
	ctx := context.WithValue(req.Context(), userIDKey, user.ID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.CreateAPIKey(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rr.Code, rr.Body.String())
	}

	var resp createAPIKeyResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !strings.HasPrefix(resp.Key, "fb_") {
		t.Errorf("key missing fb_ prefix: %q", resp.Key)
	}
	if resp.Name != "ci-key" {
		t.Errorf("Name = %q, want %q", resp.Name, "ci-key")
	}
	if resp.ID == uuid.Nil {
		t.Error("ID is nil")
	}
}

func TestUnitCreateAPIKey_MissingName(t *testing.T) {
	h := newTestHandlers()
	user := setupTestUser(t, h, "apikey@example.com", "password123")

	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPost, "/api-keys", strings.NewReader(body))
	ctx := context.WithValue(req.Context(), userIDKey, user.ID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.CreateAPIKey(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestUnitDeleteAPIKey_Success(t *testing.T) {
	h := newTestHandlers()
	user := setupTestUser(t, h, "apikey@example.com", "password123")

	// Create a key.
	rawKey, err := generateAPIKey()
	if err != nil {
		t.Fatalf("generate api key: %v", err)
	}
	keyHash := hashToken(rawKey)
	key, err := h.APIKeys.Create(context.Background(), user.ID, "test-key", keyHash, nil)
	if err != nil {
		t.Fatalf("create fake key: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api-keys/"+key.ID.String(), nil)
	ctx := context.WithValue(req.Context(), userIDKey, user.ID)
	req = req.WithContext(ctx)

	// chi URL param needs to be set on the request context for chi.URLParam to work.
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("keyID", key.ID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rr := httptest.NewRecorder()
	h.DeleteAPIKey(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	// Key should be gone.
	if err := h.APIKeys.Delete(context.Background(), key.ID, user.ID); err == nil {
		t.Error("key should be deleted already")
	}
}

func TestUnitDeleteAPIKey_NotFound(t *testing.T) {
	h := newTestHandlers()
	user := setupTestUser(t, h, "apikey@example.com", "password123")

	nonExistentID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api-keys/"+nonExistentID.String(), nil)
	ctx := context.WithValue(req.Context(), userIDKey, user.ID)
	req = req.WithContext(ctx)

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("keyID", nonExistentID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rr := httptest.NewRecorder()
	h.DeleteAPIKey(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestUnitDeleteAPIKey_CrossUser(t *testing.T) {
	h := newTestHandlers()
	owner := setupTestUser(t, h, "owner@example.com", "password123")
	other := setupTestUser(t, h, "other@example.com", "password123")

	// Create a key belonging to owner.
	rawKey, err := generateAPIKey()
	if err != nil {
		t.Fatalf("generate api key: %v", err)
	}
	keyHash := hashToken(rawKey)
	key, err := h.APIKeys.Create(context.Background(), owner.ID, "owner-key", keyHash, nil)
	if err != nil {
		t.Fatalf("create fake key: %v", err)
	}

	// Attempt to delete as other user.
	req := httptest.NewRequest(http.MethodDelete, "/api-keys/"+key.ID.String(), nil)
	ctx := context.WithValue(req.Context(), userIDKey, other.ID)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("keyID", key.ID.String())
	ctx = context.WithValue(ctx, chi.RouteCtxKey, routeCtx)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.DeleteAPIKey(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}
