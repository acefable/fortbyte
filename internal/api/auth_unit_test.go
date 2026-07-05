package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/youruser/fortbyte/internal/models"
)

func newTestHandlers() *Handlers {
	return &Handlers{
		Users:     newFakeUserRepo(),
		APIKeys:   newFakeAPIKeyRepo(),
		Refresh:   newFakeRefreshTokenRepo(),
		JWTSecret: []byte("unit-test-jwt-secret-32-bytes!!!1"),
	}
}

func setupTestUser(t *testing.T, h *Handlers, email, password string) *models.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		t.Fatalf("bcrypt hash: %v", err)
	}
	u := &models.User{Email: email, PasswordHash: string(hash)}
	created, err := h.Users.Create(context.Background(), u)
	if err != nil {
		t.Fatalf("create fake user: %v", err)
	}
	return created
}

func decodeResponse[T any](t *testing.T, rr *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(rr.Body).Decode(&v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return v
}

func TestUnitRegister_Success(t *testing.T) {
	h := newTestHandlers()
	body := `{"email":"test@example.com","password":"secure123"}`
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rr.Code, rr.Body.String())
	}

	resp := decodeResponse[tokenResponse](t, rr)
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("missing tokens in response")
	}
	if resp.ExpiresIn != int(accessTokenTTL.Seconds()) {
		t.Errorf("ExpiresIn = %d, want %d", resp.ExpiresIn, int(accessTokenTTL.Seconds()))
	}
}

func TestUnitRegister_InvalidEmail(t *testing.T) {
	h := newTestHandlers()
	body := `{"email":"not-an-email","password":"secure123"}`
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestUnitRegister_ShortPassword(t *testing.T) {
	h := newTestHandlers()
	body := `{"email":"test@example.com","password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestUnitRegister_DuplicateEmail(t *testing.T) {
	h := newTestHandlers()
	setupTestUser(t, h, "dupe@example.com", "password123")

	body := `{"email":"dupe@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409; body: %s", rr.Code, rr.Body.String())
	}
}

func TestUnitLogin_Success(t *testing.T) {
	h := newTestHandlers()
	setupTestUser(t, h, "login@example.com", "password123")

	body := `{"email":"login@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	resp := decodeResponse[tokenResponse](t, rr)
	if resp.AccessToken == "" {
		t.Fatal("missing access token")
	}
}

func TestUnitLogin_WrongPassword(t *testing.T) {
	h := newTestHandlers()
	setupTestUser(t, h, "login@example.com", "password123")

	body := `{"email":"login@example.com","password":"wrongpassword"}`
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestUnitLogin_UserNotFound(t *testing.T) {
	h := newTestHandlers()

	body := `{"email":"nobody@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestUnitRefreshTokens_Success(t *testing.T) {
	h := newTestHandlers()
	user := setupTestUser(t, h, "refresh@example.com", "password123")

	// Create a refresh token directly via the fake repo.
	rawRefresh, err := generateRefreshToken()
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}
	refreshHash := hashToken(rawRefresh)
	if _, err := h.Refresh.Create(context.Background(), user.ID, refreshHash, time.Now().Add(refreshTokenTTL)); err != nil {
		t.Fatalf("store refresh token: %v", err)
	}

	body := `{"refresh_token":"` + rawRefresh + `"}`
	req := httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.RefreshTokens(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	resp := decodeResponse[tokenResponse](t, rr)
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("missing tokens in refresh response")
	}
	// Old token should be revoked.
	if tok, _ := h.Refresh.GetByTokenHash(context.Background(), refreshHash); tok == nil || tok.RevokedAt == nil {
		t.Error("old refresh token should be revoked after rotation")
	}
}

func TestUnitRefreshTokens_ExpiredToken(t *testing.T) {
	h := newTestHandlers()
	user := setupTestUser(t, h, "expired@example.com", "password123")

	rawRefresh, err := generateRefreshToken()
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}
	refreshHash := hashToken(rawRefresh)
	if _, err := h.Refresh.Create(context.Background(), user.ID, refreshHash, time.Now().Add(-1*time.Hour)); err != nil {
		t.Fatalf("store expired refresh token: %v", err)
	}

	body := `{"refresh_token":"` + rawRefresh + `"}`
	req := httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.RefreshTokens(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestUnitRefreshTokens_RevokedToken(t *testing.T) {
	h := newTestHandlers()
	user := setupTestUser(t, h, "revoked@example.com", "password123")

	rawRefresh, err := generateRefreshToken()
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}
	refreshHash := hashToken(rawRefresh)
	tok, err := h.Refresh.Create(context.Background(), user.ID, refreshHash, time.Now().Add(refreshTokenTTL))
	if err != nil {
		t.Fatalf("store refresh token: %v", err)
	}
	if err := h.Refresh.Revoke(context.Background(), tok.ID); err != nil {
		t.Fatalf("revoke token: %v", err)
	}

	body := `{"refresh_token":"` + rawRefresh + `"}`
	req := httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.RefreshTokens(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestUnitLogout_Success(t *testing.T) {
	h := newTestHandlers()
	user := setupTestUser(t, h, "logout@example.com", "password123")

	// Store a refresh token for the user.
	if _, err := h.Refresh.Create(context.Background(), user.ID, hashToken("token1"), time.Now().Add(refreshTokenTTL)); err != nil {
		t.Fatalf("store token1: %v", err)
	}
	if _, err := h.Refresh.Create(context.Background(), user.ID, hashToken("token2"), time.Now().Add(refreshTokenTTL)); err != nil {
		t.Fatalf("store token2: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	ctx := context.WithValue(req.Context(), userIDKey, user.ID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Logout(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	// All tokens should be revoked.
	for _, tok := range h.Refresh.(*fakeRefreshTokenRepo).tokens {
		if tok.RevokedAt == nil {
			t.Errorf("token %s not revoked", tok.TokenHash)
		}
	}
}

func TestUnitAuthMiddleware_ValidJWT(t *testing.T) {
	h := newTestHandlers()
	user := setupTestUser(t, h, "middleware@example.com", "password123")

	accessToken, err := generateAccessToken(user.ID, h.JWTSecret)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	mw := h.authMiddleware()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := UserIDFromContext(r)
		if id != user.ID {
			t.Errorf("user ID in context = %v, want %v", id, user.ID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestUnitAuthMiddleware_ExpiredJWT(t *testing.T) {
	h := newTestHandlers()
	user := setupTestUser(t, h, "expiredjwt@example.com", "password123")

	// Create an expired JWT manually.
	claims := jwt.RegisteredClaims{
		Subject:   user.ID.String(),
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredToken, err := token.SignedString(h.JWTSecret)
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	mw := h.authMiddleware()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for expired token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestUnitAuthMiddleware_MissingHeader(t *testing.T) {
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

func TestUnitAuthMiddleware_ValidAPIKey(t *testing.T) {
	h := newTestHandlers()
	user := setupTestUser(t, h, "apikeyuser@example.com", "password123")

	// Generate and store an API key.
	rawKey, err := generateAPIKey()
	if err != nil {
		t.Fatalf("generate api key: %v", err)
	}
	keyHash := hashToken(rawKey)

	if _, err := h.APIKeys.Create(context.Background(), user.ID, "test-key", keyHash, nil); err != nil {
		t.Fatalf("create fake api key: %v", err)
	}

	mw := h.authMiddleware()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := UserIDFromContext(r)
		if id != user.ID {
			t.Errorf("user ID in context = %v, want %v", id, user.ID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
}
