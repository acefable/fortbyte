package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRouterRoutesHealth(t *testing.T) {
	r := NewRouter(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Nil pool panics in repository.Ping; chi's Recoverer returns 500.
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("health with nil pool: got status %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestNewRouterReturns404ForUnknown(t *testing.T) {
	r := NewRouter(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("unknown route: got status %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestNewRouterReturns404ForUnknownPost(t *testing.T) {
	r := NewRouter(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/nonexistent", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("unknown POST route: got status %d, want %d", rr.Code, http.StatusNotFound)
	}
}
