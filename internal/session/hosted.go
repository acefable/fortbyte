package session

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

// ponytail: same keyring service as master-password, separate accounts for hosted tokens.
const (
	keyringAccessToken  = "access-token"
	keyringRefreshToken = "refresh-token"
)

// StoreTokens saves the access and refresh tokens to the OS keyring.
func StoreTokens(accessToken, refreshToken string) error {
	if err := keyring.Set(keyringService, keyringAccessToken, accessToken); err != nil {
		return fmt.Errorf("store access token: %w", err)
	}
	if err := keyring.Set(keyringService, keyringRefreshToken, refreshToken); err != nil {
		// Best-effort rollback: clear the access token we just stored.
		keyring.Delete(keyringService, keyringAccessToken)
		return fmt.Errorf("store refresh token: %w", err)
	}
	return nil
}

// LoadAccessToken reads the access token from the OS keyring.
func LoadAccessToken() (string, error) {
	token, err := keyring.Get(keyringService, keyringAccessToken)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrNoSession
		}
		return "", fmt.Errorf("load access token: %w", err)
	}
	return token, nil
}

// LoadRefreshToken reads the refresh token from the OS keyring.
func LoadRefreshToken() (string, error) {
	token, err := keyring.Get(keyringService, keyringRefreshToken)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrNoSession
		}
		return "", fmt.Errorf("load refresh token: %w", err)
	}
	return token, nil
}

// ClearHostedSession removes both access and refresh tokens from the OS keyring.
func ClearHostedSession() error {
	var firstErr error
	for _, acct := range []string{keyringAccessToken, keyringRefreshToken} {
		if err := keyring.Delete(keyringService, acct); err != nil && !errors.Is(err, keyring.ErrNotFound) {
			if firstErr == nil {
				firstErr = fmt.Errorf("delete %s: %w", acct, err)
			}
		}
	}
	return firstErr
}
