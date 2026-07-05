package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigSubcommands(t *testing.T) {
	subs := configCmd.Commands()
	names := make(map[string]bool)
	for _, c := range subs {
		names[c.Name()] = true
	}
	if !names["set"] {
		t.Error("config missing 'set' subcommand")
	}
	if !names["get"] {
		t.Error("config missing 'get' subcommand")
	}
}

func TestConfigGetUnknownKey(t *testing.T) {
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"config", "get", "unknown-key"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown config key")
	}
	if !strings.Contains(err.Error(), "unknown config key") {
		t.Errorf("error should mention 'unknown config key', got: %v", err)
	}
}

func TestConfigSetUnknownKey(t *testing.T) {
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"config", "set", "unknown-key", "value"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown config key")
	}
	if !strings.Contains(err.Error(), "unknown config key") {
		t.Errorf("error should mention 'unknown config key', got: %v", err)
	}
}

func TestConfigRequiresArgs(t *testing.T) {
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	rootCmd.SetArgs([]string{"config", "set"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for config set with no args")
	}

	rootCmd.SetArgs([]string{"config", "get"})
	err = rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for config get with no args")
	}
}

func TestLoadSaveConfigRoundtrip(t *testing.T) {
	origHome, _ := os.UserHomeDir()
	tmpDir := t.TempDir()
	if err := os.Setenv("HOME", tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	cfg := config{VaultDir: "/custom/vault/path"}
	if err := saveConfig(cfg); err != nil {
		t.Fatalf("saveConfig: %v", err)
	}

	got, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if got.VaultDir != "/custom/vault/path" {
		t.Errorf("VaultDir = %q, want %q", got.VaultDir, "/custom/vault/path")
	}
}

func TestLoadConfigMissing(t *testing.T) {
	origHome, _ := os.UserHomeDir()
	tmpDir := t.TempDir()
	if err := os.Setenv("HOME", tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig on missing file: %v", err)
	}
	if cfg.VaultDir != "" {
		t.Errorf("expected empty VaultDir, got %q", cfg.VaultDir)
	}
}

func TestLoadConfigCorrupt(t *testing.T) {
	origHome, _ := os.UserHomeDir()
	tmpDir := t.TempDir()
	if err := os.Setenv("HOME", tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	configDir := filepath.Join(tmpDir, fortDir)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected error for corrupt config")
	}
	if !strings.Contains(err.Error(), "invalid config file") {
		t.Errorf("error should mention 'invalid config file', got: %v", err)
	}
}
