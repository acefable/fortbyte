// Package main is the CLI for fortbyte, a personal secrets manager.
package main

import (
	"io"
	"os"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	minPasswordLen = 8
	maxPasswordLen = 1024
	fortDir        = ".fort"
)

var vaultDir string

// serverURL holds the value of the --server persistent flag.
var serverURL string

// readPasswordFn reads a password from stdin without echoing.
// Override in tests to avoid requiring a real terminal.
var readPasswordFn = func() (string, error) {
	pw, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	return string(pw), nil
}

// warnOut receives non-fatal warning messages (keyring store failures, etc.).
// Override in tests to suppress or capture.
var warnOut io.Writer = os.Stderr

var rootCmd = &cobra.Command{
	Use:   "fort",
	Short: "Personal secrets manager",
}

func init() {
	vaultDir = defaultVaultDir()
	cfg, _ := loadConfig()
	if cfg.VaultDir != "" {
		vaultDir = cfg.VaultDir
	}
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "", "API server URL")
}

// resolveAPIURL returns the API server URL with the following priority:
// 1. --server flag (explicit override)
// 2. FORTBYTE_API_URL env var
// 3. api_url from config file
// Returns empty string if not configured (caller should error).
func resolveAPIURL() string {
	if serverURL != "" {
		return serverURL
	}
	if env := os.Getenv("FORTBYTE_API_URL"); env != "" {
		return env
	}
	cfg, _ := loadConfig()
	return cfg.APIURL
}
