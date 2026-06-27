// Package main is the CLI for gokeep, a personal secrets manager.
package main

import (
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	minPasswordLen = 8
	maxPasswordLen = 1024
	gokeepDir      = ".gokeep"
)

var vaultDir string

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
	Use:   "gokeep",
	Short: "Personal secrets manager",
}

func init() {
	// Compute vault directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		vaultDir = filepath.Join(homeDir, gokeepDir)
	}
}
