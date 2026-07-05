package main

import (
	"io"
	"regexp"
	"testing"

	"github.com/spf13/pflag"

	"github.com/youruser/fortbyte/internal/crypto"
	"github.com/youruser/fortbyte/internal/session"
	"github.com/youruser/fortbyte/internal/vault"
)

// ansiRe matches ANSI escape sequences.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes ANSI escape sequences from s.
func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// mockReadPassword overrides readPasswordFn for tests that call rootCmd.Execute()
// on paths that hit getKey (which reads a password from the terminal).
func mockReadPassword(t *testing.T, password string, err error) {
	t.Helper()
	orig := readPasswordFn
	readPasswordFn = func() (string, error) { return password, err }
	t.Cleanup(func() { readPasswordFn = orig })
}

// stubSession overrides session.StorePassword/LoadPassword/Clear for tests
// that would otherwise touch the host OS keyring.
func stubSession(t *testing.T) {
	t.Helper()
	origStore, origLoad, origClear := session.StorePassword, session.LoadPassword, session.Clear
	session.StorePassword = func(string, string) error { return nil }
	session.LoadPassword = func() (string, error) { return "stub-password-1234", nil }
	session.Clear = func(string) error { return nil }
	t.Cleanup(func() {
		session.StorePassword, session.LoadPassword, session.Clear = origStore, origLoad, origClear
	})
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no ansi", "hello world", "hello world"},
		{"empty", "", ""},
		{"single seq", "\x1b[31mred\x1b[0m", "red"},
		{"nested", "\x1b[1m\x1b[32mgreen bold\x1b[0m", "green bold"},
		{"mixed", "before \x1b[34mblue\x1b[0m after", "before blue after"},
		{"multiple", "\x1b[31ma\x1b[0m \x1b[32mb\x1b[0m", "a b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(tt.input)
			if got != tt.want {
				t.Errorf("stripANSI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// resetMoveCopyFlags resets cobra flag state for secret move/copy commands
// and the parent secretCmd persistent flags. pflag.Set() marks Changed=true
// permanently; this resets both the value and Changed so required flag checks
// and scoped lookups work correctly across table-driven subtests.
func resetMoveCopyFlags(t *testing.T) {
	t.Helper()
	resetFlag(t, secretCmd.PersistentFlags(), "project")
	for _, name := range []string{"dest-project", "dest-env", "env"} {
		resetFlag(t, secretMoveCmd.Flags(), name)
	}
	for _, name := range []string{"dest-project", "dest-env", "env", "name"} {
		resetFlag(t, secretCopyCmd.Flags(), name)
	}
}

func resetFlag(t *testing.T, fs *pflag.FlagSet, name string) {
	t.Helper()
	f := fs.Lookup(name)
	if f == nil {
		return
	}
	_ = f.Value.Set("")
	f.Changed = false
}

// setupTestVault creates a fresh vault in a temp dir, overrides vaultDir,
// and stubs session + password reading. Returns the temp dir.
func setupTestVault(t *testing.T) string {
	t.Helper()
	stubSession(t)
	mockReadPassword(t, "password1234", nil)
	origWarn := warnOut
	warnOut = io.Discard
	t.Cleanup(func() { warnOut = origWarn })
	dir := t.TempDir()
	origDir := vaultDir
	vaultDir = dir
	t.Cleanup(func() { vaultDir = origDir })
	if err := runInit(dir, "password1234", "password1234"); err != nil {
		t.Fatalf("runInit: %v", err)
	}
	return dir
}

// seedVault adds test data to the vault and saves it.
func seedVault(t *testing.T, dir string) {
	t.Helper()
	salt, err := vault.GetSalt(dir)
	if err != nil {
		t.Fatalf("GetSalt: %v", err)
	}
	key := crypto.DeriveKey("password1234", salt)
	v, err := vault.Open(dir, key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	pUID, _ := v.AddProject(vault.Project{Name: "myapp"})
	eUID, _ := v.AddEnvironment(vault.Environment{Name: "prod", ProjectUID: pUID})
	v.AddSecret(vault.Secret{Name: "DB_PASS", Value: "s3cret", ProjectUID: pUID, EnvironmentUID: eUID, URL: "https://db.example.com", Notes: "database"})
	v.AddSecret(vault.Secret{Name: "API_KEY", Value: "key123", ProjectUID: pUID, EnvironmentUID: eUID})
	v.AddSecret(vault.Secret{Name: "STANDALONE", Value: "solo"})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}
}

// resetCmdFlags resets cobra flag values that leak between Execute() calls.
// pflag doesn't reset unset flags between parses, so we must do it explicitly.
func resetCmdFlags(t *testing.T) {
	t.Helper()
	resetExportImportFlags()
	t.Cleanup(resetExportImportFlags)
}

func resetExportImportFlags() {
	rootCmd.SetArgs(nil)
	_ = secretCmd.PersistentFlags().Set("project", "")
	for _, name := range []string{"project", "env", "format"} {
		_ = exportCmd.Flags().Set(name, "")
		_ = importCmd.Flags().Set(name, "")
	}
	for _, name := range []string{"project", "env", "format"} {
		_ = findCmd.Flags().Set(name, "")
	}
	for _, name := range []string{"env", "format"} {
		_ = secretRevealCmd.Flags().Set(name, "")
		_ = secretShowCmd.Flags().Set(name, "")
	}
	for _, name := range []string{"env", "format", "filter"} {
		_ = secretListCmd.Flags().Set(name, "")
	}
	for _, name := range []string{"project", "env", "format"} {
		_ = listCmd.Flags().Set(name, "")
	}
	_ = envCmd.PersistentFlags().Set("project", "")
}
