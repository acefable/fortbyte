package client

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// testServer starts an httptest.Server and returns the client pointed at it.
func testServer(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return New(srv.URL), srv
}

func TestRegister_Success(t *testing.T) {
	t.Parallel()
	c, _ := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/auth/register" {
			t.Errorf("path = %s, want /auth/register", r.URL.Path)
		}
		var body map[string]string
		json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body)
		if body["email"] != "alice@example.com" {
			t.Errorf("email = %s, want alice@example.com", body["email"])
		}
		if body["password"] != "secret123" {
			t.Errorf("password = %s, want secret123", body["password"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"at_abc","refresh_token":"rt_xyz","expires_in":3600}`))
	}))

	tok, err := c.Register("alice@example.com", "secret123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "at_abc" {
		t.Errorf("AccessToken = %q, want at_abc", tok.AccessToken)
	}
	if tok.RefreshToken != "rt_xyz" {
		t.Errorf("RefreshToken = %q, want rt_xyz", tok.RefreshToken)
	}
	if tok.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn = %d, want 3600", tok.ExpiresIn)
	}
}

func TestLogin_Success(t *testing.T) {
	t.Parallel()
	c, _ := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/login" {
			t.Errorf("path = %s, want /auth/login", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"at_login","refresh_token":"rt_login","expires_in":7200}`))
	}))

	tok, err := c.Login("bob@example.com", "pw123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "at_login" {
		t.Errorf("AccessToken = %q, want at_login", tok.AccessToken)
	}
}

func TestRefresh_Success(t *testing.T) {
	t.Parallel()
	c, _ := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/refresh" {
			t.Errorf("path = %s, want /auth/refresh", r.URL.Path)
		}
		var body map[string]string
		json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body)
		if body["refresh_token"] != "rt_old" {
			t.Errorf("refresh_token = %s, want rt_old", body["refresh_token"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"at_new","refresh_token":"rt_new","expires_in":3600}`))
	}))

	tok, err := c.Refresh("rt_old")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "at_new" {
		t.Errorf("AccessToken = %q, want at_new", tok.AccessToken)
	}
}

func TestLogout_Success(t *testing.T) {
	t.Parallel()
	c, _ := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/logout" {
			t.Errorf("path = %s, want /auth/logout", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer my_token" {
			t.Errorf("Authorization = %q, want 'Bearer my_token'", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"logged_out"}`))
	}))

	err := c.Logout("my_token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLogout_IgnoresExtraResponseFields(t *testing.T) {
	t.Parallel()
	// Server returns extra fields — should not cause an error.
	c, _ := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"logged_out","message":"bye","code":200}`))
	}))

	err := c.Logout("token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPost_ServerError_WithMessage(t *testing.T) {
	t.Parallel()
	c, _ := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error":{"code":"already_exists","message":"email already registered"}}`))
	}))

	_, err := c.Register("dup@example.com", "pw")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, err) { // just checking non-nil
		t.Error("error should not be nil")
	}
	// The message should come from the server.
	if got := err.Error(); got != "email already registered" {
		t.Errorf("error = %q, want 'email already registered'", got)
	}
}

func TestPost_ServerError_NoMessage(t *testing.T) {
	t.Parallel()
	c, _ := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"unexpected":"format"}`))
	}))

	_, err := c.Login("x@y.com", "pw")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Falls back to raw body when no error.message field.
	if got := err.Error(); got != "HTTP 500" {
		t.Errorf("error = %q", got)
	}
}

func TestPost_ServerError_PlainText(t *testing.T) {
	t.Parallel()
	c, _ := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`bad request`))
	}))

	_, err := c.Login("x@y.com", "pw")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "HTTP 400" {
		t.Errorf("error = %q", got)
	}
}

func TestPost_BearerTokenHeader(t *testing.T) {
	t.Parallel()
	c, _ := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer tok_123" {
			t.Errorf("Authorization = %q, want 'Bearer tok_123'", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))

	// Logout passes the access token as a bearer token.
	err := c.Logout("tok_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPost_NoBearerTokenWhenEmpty(t *testing.T) {
	t.Parallel()
	c, _ := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "" {
			t.Errorf("Authorization = %q, want empty", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))

	_, err := c.Register("x@y.com", "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPost_NonJSONResponse(t *testing.T) {
	t.Parallel()
	c, _ := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(`hello`))
	}))

	_, err := c.Login("x@y.com", "pw")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Should get a decode error.
	if got := err.Error(); got != "decode response: invalid character 'h' looking for beginning of value" {
		t.Errorf("error = %q", got)
	}
}

func TestNew_TrimsTrailingSlash(t *testing.T) {
	t.Parallel()
	c := New("http://localhost:8080/api/v1/")
	if c.baseURL != "http://localhost:8080/api/v1" {
		t.Errorf("baseURL = %q, trailing slash not trimmed", c.baseURL)
	}
}

func TestPost_NilBody(t *testing.T) {
	t.Parallel()
	c, _ := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// When body is nil, Content-Length should be absent or 0.
		if r.ContentLength > 0 {
			t.Errorf("ContentLength = %d, want 0", r.ContentLength)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"","refresh_token":"","expires_in":0}`))
	}))

	// Logout sends nil body — exercises the nil-body path in post().
	err := c.Logout("some_token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
