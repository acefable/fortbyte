package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// TokenResponse is the auth API token pair response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// apiError is the standard error envelope from the server.
type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Client is a minimal HTTP client for the fortbyte auth API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New creates a client for the given server URL (e.g., "http://localhost:8080").
// /api/v1 is appended automatically if not already present.
func New(baseURL string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	if !strings.HasSuffix(baseURL, "/api/v1") {
		baseURL += "/api/v1"
	}
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Register creates a new user account and returns tokens.
func (c *Client) Register(email, password string) (*TokenResponse, error) {
	return c.post("/auth/register", map[string]string{"email": email, "password": password}, "")
}

// Login authenticates an existing user and returns tokens.
func (c *Client) Login(email, password string) (*TokenResponse, error) {
	return c.post("/auth/login", map[string]string{"email": email, "password": password}, "")
}

// Refresh exchanges a refresh token for a new token pair.
func (c *Client) Refresh(refreshToken string) (*TokenResponse, error) {
	return c.post("/auth/refresh", map[string]string{"refresh_token": refreshToken}, "")
}

// Logout revokes all refresh tokens for the authenticated user.
func (c *Client) Logout(accessToken string) error {
	_, err := c.post("/auth/logout", nil, accessToken)
	return err
}

func (c *Client) post(path string, body any, bearerToken string) (*TokenResponse, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr apiError
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("%s", apiErr.Error.Message)
		}
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var tok TokenResponse
	if err := json.Unmarshal(respBody, &tok); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &tok, nil
}
