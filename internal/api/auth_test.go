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
	"github.com/google/uuid"
	"github.com/youruser/fortbyte/internal/models"
)

func TestHashToken_Deterministic(t *testing.T) {
	t.Parallel()
	a := hashToken("sometoken")
	b := hashToken("sometoken")
	if a != b {
		t.Errorf("hashToken not deterministic: %q != %q", a, b)
	}
}

func TestHashToken_DifferentInputs(t *testing.T) {
	t.Parallel()
	a := hashToken("token-a")
	b := hashToken("token-b")
	if a == b {
		t.Error("hashToken returned same hash for different inputs")
	}
}

func TestHashToken_IsSHA256Hex(t *testing.T) {
	t.Parallel()
	h := hashToken("test")
	// SHA-256 hex is 64 characters.
	if len(h) != 64 {
		t.Errorf("hashToken output length = %d, want 64", len(h))
	}
}

func TestGenerateAPIKey_Format(t *testing.T) {
	t.Parallel()
	key, err := generateAPIKey()
	if err != nil {
		t.Fatalf("generateAPIKey() error: %v", err)
	}

	if !strings.HasPrefix(key, "fb_") {
		t.Errorf("key missing fb_ prefix: %q", key)
	}
	// fb_ (3) + 64 hex chars = 67 total.
	if len(key) != 67 {
		t.Errorf("key length = %d, want 67", len(key))
	}

	hexPart := key[3:]
	for _, c := range hexPart {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Errorf("non-hex character in key: %c", c)
			break
		}
	}
}

func TestGenerateAPIKey_Unique(t *testing.T) {
	t.Parallel()
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		key, err := generateAPIKey()
		if err != nil {
			t.Fatalf("generateAPIKey() error on iteration %d: %v", i, err)
		}
		if seen[key] {
			t.Fatalf("duplicate key generated on iteration %d", i)
		}
		seen[key] = true
	}
}

func TestGenerateRefreshToken_Length(t *testing.T) {
	t.Parallel()
	token, err := generateRefreshToken()
	if err != nil {
		t.Fatalf("generateRefreshToken() error: %v", err)
	}
	// 32 bytes hex-encoded = 64 chars.
	if len(token) != 64 {
		t.Errorf("refresh token length = %d, want 64", len(token))
	}
}

func TestGenerateRefreshToken_Unique(t *testing.T) {
	t.Parallel()
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		token, err := generateRefreshToken()
		if err != nil {
			t.Fatalf("generateRefreshToken() error on iteration %d: %v", i, err)
		}
		if seen[token] {
			t.Fatalf("duplicate refresh token on iteration %d", i)
		}
		seen[token] = true
	}
}

func TestGenerateAccessToken_ValidJWT(t *testing.T) {
	t.Parallel()
	secret := []byte("test-secret-key-for-jwt-signing-1234")
	userID := uuid.New()

	tokenStr, err := generateAccessToken(userID, secret)
	if err != nil {
		t.Fatalf("generateAccessToken() error: %v", err)
	}

	// Parse and verify.
	claims := &jwt.RegisteredClaims{}
	parsed, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		return secret, nil
	})
	if err != nil {
		t.Fatalf("ParseWithClaims error: %v", err)
	}
	if !parsed.Valid {
		t.Fatal("token is not valid")
	}

	if claims.Subject != userID.String() {
		t.Errorf("Subject = %q, want %q", claims.Subject, userID.String())
	}
	if claims.IssuedAt == nil {
		t.Error("IssuedAt is nil")
	}
	if claims.ExpiresAt == nil {
		t.Error("ExpiresAt is nil")
	}
	if claims.ExpiresAt == nil || claims.IssuedAt == nil {
		t.Error("IssuedAt or ExpiresAt is nil")
	} else {
		issued := claims.IssuedAt.Time
		expires := claims.ExpiresAt.Time
		ttl := expires.Sub(issued)
		if ttl != accessTokenTTL {
			t.Errorf("token TTL = %v, want %v", ttl, accessTokenTTL)
		}
	}
}

func TestGenerateAccessToken_WrongSecret(t *testing.T) {
	t.Parallel()
	secret := []byte("correct-secret-key-for-jwt-12345678")
	wrongSecret := []byte("wrong-secret-key-for-jwt-12345678")
	userID := uuid.New()

	tokenStr, err := generateAccessToken(userID, secret)
	if err != nil {
		t.Fatalf("generateAccessToken() error: %v", err)
	}

	claims := &jwt.RegisteredClaims{}
	_, err = jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		return wrongSecret, nil
	})
	if err == nil {
		t.Fatal("token should be invalid with wrong secret")
	}
}

func TestDecodeJSONBody_Success(t *testing.T) {
	t.Parallel()
	body := `{"email":"test@example.com","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	var got authRequest
	if err := decodeJSONBody(req, &got); err != nil {
		t.Fatalf("decodeJSONBody() error: %v", err)
	}
	if got.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "test@example.com")
	}
	if got.Password != "secret123" {
		t.Errorf("Password = %q, want %q", got.Password, "secret123")
	}
}

func TestDecodeJSONBody_InvalidJSON(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{bad json"))
	var got authRequest
	if err := decodeJSONBody(req, &got); err == nil {
		t.Fatal("decodeJSONBody should fail on invalid JSON")
	}
}

func TestDecodeJSONBody_EmptyBody(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	var got authRequest
	if err := decodeJSONBody(req, &got); err == nil {
		t.Fatal("decodeJSONBody should fail on empty body")
	}
}

func TestDecodeJSONBody_SizeLimit(t *testing.T) {
	t.Parallel()
	// Body > 1MB should be rejected.
	bigBody := strings.Repeat("x", 1<<20+1)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(bigBody))
	var got map[string]any
	if err := decodeJSONBody(req, &got); err == nil {
		t.Fatal("decodeJSONBody should fail on body > 1MB")
	}
}

func TestWriteJSON_Success(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var got map[string]string
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if got["status"] != "ok" {
		t.Errorf("status = %q, want %q", got["status"], "ok")
	}
}

func TestWriteError_Success(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	writeError(w, http.StatusUnauthorized, "auth_error", "invalid token")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	var got models.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if got.Error.Code != "auth_error" {
		t.Errorf("code = %q, want %q", got.Error.Code, "auth_error")
	}
	if got.Error.Message != "invalid token" {
		t.Errorf("message = %q, want %q", got.Error.Message, "invalid token")
	}
}

func TestUserIDFromContext_Present(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, userIDKey, id)
	req = req.WithContext(ctx)

	got := UserIDFromContext(req)
	if got != id {
		t.Errorf("UserIDFromContext() = %v, want %v", got, id)
	}
}

func TestUserIDFromContext_Missing(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	got := UserIDFromContext(req)
	if got != uuid.Nil {
		t.Errorf("UserIDFromContext() = %v, want uuid.Nil", got)
	}
}

func TestUserIDFromContext_WrongType(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, userIDKey, "not-a-uuid")
	req = req.WithContext(ctx)

	got := UserIDFromContext(req)
	if got != uuid.Nil {
		t.Errorf("UserIDFromContext() with wrong type = %v, want uuid.Nil", got)
	}
}

func TestAuthRequestJSON_RoundTrip(t *testing.T) {
	t.Parallel()
	original := authRequest{Email: "user@test.com", Password: "pass1234"}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded authRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded != original {
		t.Errorf("round-trip mismatch: %+v != %+v", decoded, original)
	}
}

func TestTokenResponseJSON_RoundTrip(t *testing.T) {
	t.Parallel()
	original := tokenResponse{
		AccessToken:  "eyJhbGc...",
		RefreshToken: "abc123",
		ExpiresIn:    900,
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded tokenResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded != original {
		t.Errorf("round-trip mismatch: %+v != %+v", decoded, original)
	}
}

func TestEmailRegex(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{"valid simple", "user@example.com", true},
		{"valid subdomain", "user@sub.example.com", true},
		{"valid plus", "user+tag@example.com", true},
		{"no at", "userexample.com", false},
		{"no domain", "user@", false},
		{"no local", "@example.com", false},
		{"spaces", "user @example.com", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := emailRegex.MatchString(tt.email)
			if got != tt.want {
				t.Errorf("emailRegex.MatchString(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

func TestAccessTokenTTL(t *testing.T) {
	t.Parallel()
	if accessTokenTTL != 15*time.Minute {
		t.Errorf("accessTokenTTL = %v, want 15m", accessTokenTTL)
	}
}

func TestRefreshTokenTTL(t *testing.T) {
	t.Parallel()
	if refreshTokenTTL != 7*24*time.Hour {
		t.Errorf("refreshTokenTTL = %v, want 168h", refreshTokenTTL)
	}
}
