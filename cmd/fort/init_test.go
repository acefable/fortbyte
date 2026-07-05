package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/youruser/fortbyte/internal/vault"
)

func TestInitRejectsExistingVault(t *testing.T) {
	stubSession(t)
	warnOut = io.Discard
	t.Cleanup(func() { warnOut = os.Stderr })
	dir := t.TempDir()
	vaultPath := filepath.Join(dir, vault.FileName)
	if err := os.WriteFile(vaultPath, []byte("fake"), 0600); err != nil {
		t.Fatalf("create fake vault: %v", err)
	}
	err := runInit(dir, "password1234", "password1234")
	if err == nil {
		t.Fatal("expected error for existing vault")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should say 'already exists', got: %v", err)
	}
}

func TestInitValidatesPasswordLength(t *testing.T) {
	stubSession(t)
	warnOut = io.Discard
	t.Cleanup(func() { warnOut = os.Stderr })
	dir := t.TempDir()
	tests := []struct {
		name     string
		password string
		confirm  string
		wantErr  bool
	}{
		{"too short", "short", "short", true},
		{"min length", "12345678", "12345678", false}, // 8 chars = OK
		{"too long", strings.Repeat("a", 1025), strings.Repeat("a", 1025), true},
		{"max length", strings.Repeat("a", 1024), strings.Repeat("a", 1024), false}, // 1024 = OK
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := filepath.Join(dir, tt.name)
			if err := os.MkdirAll(d, 0700); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			err := runInit(d, tt.password, tt.confirm)
			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestInitRejectsMismatchedConfirm(t *testing.T) {
	stubSession(t)
	warnOut = io.Discard
	t.Cleanup(func() { warnOut = os.Stderr })
	dir := t.TempDir()
	err := runInit(dir, "password1234", "different1234")
	if err == nil {
		t.Fatal("expected error for mismatched confirm")
	}
	if !strings.Contains(err.Error(), "do not match") {
		t.Errorf("error should say 'do not match', got: %v", err)
	}
}

func TestInitCreatesVault(t *testing.T) {
	stubSession(t)
	warnOut = io.Discard
	t.Cleanup(func() { warnOut = os.Stderr })
	dir := t.TempDir()
	if err := runInit(dir, "password1234", "password1234"); err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	vaultPath := filepath.Join(dir, vault.FileName)
	if _, err := os.Stat(vaultPath); os.IsNotExist(err) {
		t.Fatal("vault file was not created")
	}
}
