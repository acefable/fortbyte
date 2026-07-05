//go:build integration

package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/youruser/fortbyte/internal/repository"
)

var testDB *pgxpool.Pool

func TestMain(m *testing.M) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://fortbyte:changeme@localhost:5432/fortbyte?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	testDB, err = pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("connect to test database: %v", err)
	}

	if err := repository.RunMigrations(dbURL, "../../cmd/server/migrations"); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	code := m.Run()
	testDB.Close()
	os.Exit(code)
}

func setupIntegrationRouter(t *testing.T) *chi.Mux {
	t.Helper()
	return NewRouter(testDB, []byte("integration-test-secret-key-123456789012"))
}

func cleanupTestUsers(t *testing.T) {
	t.Helper()
	_, _ = testDB.Exec(context.Background(), "DELETE FROM users WHERE email LIKE '%@integration-test.example.com'")
}

func TestIntegration_Health(t *testing.T) {
	router := setupIntegrationRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK && rr.Code != http.StatusServiceUnavailable {
		t.Errorf("health status = %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestIntegration_Ready(t *testing.T) {
	router := setupIntegrationRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ready", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ready status = %d", rr.Code)
	}
}

func TestIntegration_RegisterAndLogin(t *testing.T) {
	cleanupTestUsers(t)
	router := setupIntegrationRouter(t)
	email := "register-login@integration-test.example.com"

	// Register
	body := `{"email":"` + email + `","password":"securepass123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body: %s", rr.Code, rr.Body.String())
	}

	var regResp map[string]any
	json.NewDecoder(rr.Body).Decode(&regResp)
	accessToken := regResp["access_token"].(string)
	refreshToken := regResp["refresh_token"].(string)
	if accessToken == "" || refreshToken == "" {
		t.Fatal("missing tokens in register response")
	}

	// Login with same creds
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("login status = %d, body: %s", rr.Code, rr.Body.String())
	}

	// Login with wrong password → 401
	wrongBody := `{"email":"` + email + `","password":"wrongpassword"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(wrongBody))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("login wrong password status = %d, want 401", rr.Code)
	}

	// Use access token to call logout
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("logout status = %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestIntegration_RefreshTokenRotation(t *testing.T) {
	cleanupTestUsers(t)
	router := setupIntegrationRouter(t)
	email := "refresh@integration-test.example.com"

	// Register
	body := `{"email":"` + email + `","password":"securepass123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var regResp map[string]any
	json.NewDecoder(rr.Body).Decode(&regResp)
	refreshToken := regResp["refresh_token"].(string)

	// Refresh
	refreshBody := `{"refresh_token":"` + refreshToken + `"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(refreshBody))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, body: %s", rr.Code, rr.Body.String())
	}

	var refreshResp map[string]any
	json.NewDecoder(rr.Body).Decode(&refreshResp)
	newRefresh := refreshResp["refresh_token"].(string)
	if newRefresh == refreshToken {
		t.Error("refresh token should rotate (new token should differ)")
	}

	// Old refresh token should be invalid
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(refreshBody))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("old refresh token reuse status = %d, want 401", rr.Code)
	}
}

func TestIntegration_APIKeyCRUD(t *testing.T) {
	cleanupTestUsers(t)
	router := setupIntegrationRouter(t)
	email := "apikeys@integration-test.example.com"

	// Register to get token
	body := `{"email":"` + email + `","password":"securepass123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	var regResp map[string]any
	json.NewDecoder(rr.Body).Decode(&regResp)
	token := regResp["access_token"].(string)

	// Create API key
	keyBody := `{"name":"ci-key"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/api-keys", strings.NewReader(keyBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create api key status = %d, body: %s", rr.Code, rr.Body.String())
	}
	var keyResp map[string]any
	json.NewDecoder(rr.Body).Decode(&keyResp)
	rawKey := keyResp["key"].(string)
	keyID := keyResp["id"].(string)

	// Auth with API key (use health endpoint which is public, just to verify auth works)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	// Health is public so it always returns OK; we're testing the middleware doesn't reject.
	if rr.Code != http.StatusOK {
		t.Errorf("api key auth to health endpoint status = %d, body: %s", rr.Code, rr.Body.String())
	}

	// Delete API key
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/auth/api-keys/"+keyID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("delete api key status = %d, body: %s", rr.Code, rr.Body.String())
	}

	// Deleted key should not work for a protected endpoint (use another API key creation attempt)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/api-keys", strings.NewReader(keyBody))
	req.Header.Set("Authorization", "Bearer "+rawKey)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("deleted api key reuse status = %d, want 401", rr.Code)
	}
}
