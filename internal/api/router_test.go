package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRouterRoutesHealth(t *testing.T) {
	r := NewRouter(nil) // nil pool → Recoverer catches panic → 500

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	// Recoverer catches the nil-pool panic and returns 500.
	// Chi's Recoverer writes no body — only the status code is reliable here.
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("health with nil pool: got status %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestNewRouterReturns404ForUnknown(t *testing.T) {
	r := NewRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("unknown route: got status %d, want %d", rr.Code, http.StatusNotFound)
	}
}
