package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/youruser/fortbyte/internal/crypto"
	"github.com/youruser/fortbyte/internal/vault"
)

func TestImportJSON(t *testing.T) {
	resetCmdFlags(t)
	dir := setupTestVault(t)

	importData := `[{"name":"NEW_SECRET","project":"myapp","env":"prod","value":"imported","url":"https://example.com","notes":"from json"}]`
	importFile := filepath.Join(t.TempDir(), "import.json")
	if err := os.WriteFile(importFile, []byte(importData), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"import", importFile})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Added:   1 secrets") {
		t.Errorf("expected 1 added secret, got:\n%s", output)
	}

	salt, _ := vault.GetSalt(dir)
	key := crypto.DeriveKey("password1234", salt)
	v, _ := vault.Open(dir, key)
	p, _, pf := findProjectByName(v, "myapp")
	if !pf {
		t.Fatal("project 'myapp' not found after import")
	}
	e, _, ef := findEnvironmentByName(v, "prod", p.UID)
	if !ef {
		t.Fatal("environment 'prod' not found after import")
	}
	_, _, found := v.FindSecretByName("NEW_SECRET", p.UID, e.UID)
	if !found {
		t.Error("NEW_SECRET not found after import")
	}
}

func TestImportJSONCreatesProjectEnv(t *testing.T) {
	resetCmdFlags(t)
	dir := setupTestVault(t)

	importData := `[{"name":"CREATED_SECRET","project":"brand-new","env":"staging","value":"test123"}]`
	importFile := filepath.Join(t.TempDir(), "create.json")
	if err := os.WriteFile(importFile, []byte(importData), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"import", importFile})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Added:   1 secrets") {
		t.Errorf("expected 1 added, got:\n%s", output)
	}

	salt, _ := vault.GetSalt(dir)
	key := crypto.DeriveKey("password1234", salt)
	v, _ := vault.Open(dir, key)
	p, _, found := findProjectByName(v, "brand-new")
	if !found {
		t.Fatal("project 'brand-new' not created")
	}
	_, _, found = findEnvironmentByName(v, "staging", p.UID)
	if !found {
		t.Fatal("environment 'staging' not created in project 'brand-new'")
	}
}

func TestImportEnv(t *testing.T) {
	resetCmdFlags(t)
	dir := setupTestVault(t)
	salt, _ := vault.GetSalt(dir)
	key := crypto.DeriveKey("password1234", salt)
	v, _ := vault.Open(dir, key)
	pUID, _ := v.AddProject(vault.Project{Name: "myapp"})
	v.AddEnvironment(vault.Environment{Name: "dev", ProjectUID: pUID})
	v.Save(dir, key)

	importFile := filepath.Join(t.TempDir(), "import.env")
	if err := os.WriteFile(importFile, []byte("NEW_VAR=hello\nOTHER_VAR=world\n"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"import", importFile, "--project", "myapp", "--env", "dev"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Added:   2 secrets") {
		t.Errorf("expected 2 added, got:\n%s", output)
	}
}

func TestImportEnvRequiresProject(t *testing.T) {
	resetCmdFlags(t)
	importFile := filepath.Join(t.TempDir(), "import.env")
	if err := os.WriteFile(importFile, []byte("KEY=val\n"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"import", importFile})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for .env import without --project")
	}
	if !strings.Contains(err.Error(), "--project is required") {
		t.Errorf("error should mention '--project is required', got: %v", err)
	}
}

func TestImportSkipsDuplicates(t *testing.T) {
	resetCmdFlags(t)
	dir := setupTestVault(t)
	seedVault(t, dir)

	importData := `[{"name":"DB_PASS","value":"new_value","project":"myapp","env":"prod"}]`
	importFile := filepath.Join(t.TempDir(), "dup.json")
	if err := os.WriteFile(importFile, []byte(importData), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"import", importFile})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Added:   0 secrets") {
		t.Errorf("expected 0 added, got:\n%s", output)
	}
	if !strings.Contains(output, "Skipped: 1 secrets") {
		t.Errorf("expected 1 skipped, got:\n%s", output)
	}
	if !strings.Contains(output, "DB_PASS") {
		t.Errorf("skipped list should mention DB_PASS, got:\n%s", output)
	}
}

func TestImportSummaryOutput(t *testing.T) {
	resetCmdFlags(t)
	setupTestVault(t)

	importData := `[{"name":"A","value":"1","project":"p1"},{"name":"B","value":"2","project":"p1"}]`
	importFile := filepath.Join(t.TempDir(), "summary.json")
	if err := os.WriteFile(importFile, []byte(importData), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"import", importFile})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Import complete:") {
		t.Errorf("missing header:\n%s", output)
	}
	if !strings.Contains(output, "Added:   2 secrets") {
		t.Errorf("expected 2 added, got:\n%s", output)
	}

	buf.Reset()
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"import", importFile})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("import (re-import): %v", err)
	}
	output = buf.String()
	if !strings.Contains(output, "Added:   0 secrets") {
		t.Errorf("re-import: expected 0 added, got:\n%s", output)
	}
	if !strings.Contains(output, "Skipped: 2 secrets") {
		t.Errorf("re-import: expected 2 skipped, got:\n%s", output)
	}
}

func TestImportEnvParsesComments(t *testing.T) {
	resetCmdFlags(t)
	dir := setupTestVault(t)
	salt, _ := vault.GetSalt(dir)
	key := crypto.DeriveKey("password1234", salt)
	v, _ := vault.Open(dir, key)
	pUID, _ := v.AddProject(vault.Project{Name: "myapp"})
	v.Save(dir, key)

	envContent := "# This is a comment\nREAL_VALUE=hello\n# Another comment\n\n\nANOTHER=world\n"
	importFile := filepath.Join(t.TempDir(), "comments.env")
	if err := os.WriteFile(importFile, []byte(envContent), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"import", importFile, "--project", "myapp"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Added:   2 secrets") {
		t.Errorf("expected 2 added (comments/blanks skipped), got:\n%s", output)
	}

	v2, _ := vault.Open(dir, key)
	_, _, f1 := v2.FindSecretByName("REAL_VALUE", pUID, "")
	_, _, f2 := v2.FindSecretByName("ANOTHER", pUID, "")
	if !f1 || !f2 {
		t.Error("secrets not found after import with comments")
	}
}

func TestImportEnvParsesQuotes(t *testing.T) {
	resetCmdFlags(t)
	dir := setupTestVault(t)
	salt, _ := vault.GetSalt(dir)
	key := crypto.DeriveKey("password1234", salt)
	v, _ := vault.Open(dir, key)
	pUID, _ := v.AddProject(vault.Project{Name: "myapp"})
	v.Save(dir, key)

	envContent := "DB_HOST=\"localhost\"\nDB_PORT='5432'\nDB_CONN=\"host=localhost port=5432\"\n"
	importFile := filepath.Join(t.TempDir(), "quoted.env")
	if err := os.WriteFile(importFile, []byte(envContent), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"import", importFile, "--project", "myapp"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}

	v2, _ := vault.Open(dir, key)
	s1, _, f1 := v2.FindSecretByName("DB_HOST", pUID, "")
	s2, _, f2 := v2.FindSecretByName("DB_PORT", pUID, "")
	s3, _, f3 := v2.FindSecretByName("DB_CONN", pUID, "")
	if !f1 || !f2 || !f3 {
		t.Fatal("secrets not found after quoted import")
	}
	if s1.Value != "localhost" {
		t.Errorf("DB_HOST value = %q, want %q", s1.Value, "localhost")
	}
	if s2.Value != "5432" {
		t.Errorf("DB_PORT value = %q, want %q", s2.Value, "5432")
	}
	if s3.Value != "host=localhost port=5432" {
		t.Errorf("DB_CONN value = %q, want %q", s3.Value, "host=localhost port=5432")
	}
}

func TestExportImportRoundtrip(t *testing.T) {
	resetCmdFlags(t)
	dir := setupTestVault(t)
	seedVault(t, dir)

	outFile := filepath.Join(t.TempDir(), "roundtrip.json")
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"export", outFile})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("export: %v", err)
	}

	dir2 := setupTestVault(t)
	salt2, _ := vault.GetSalt(dir2)
	key2 := crypto.DeriveKey("password1234", salt2)
	v2, _ := vault.Open(dir2, key2)
	v2.AddProject(vault.Project{Name: "myapp"})
	v2.Save(dir2, key2)

	buf.Reset()
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"import", outFile})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}
	if !strings.Contains(buf.String(), "Added:   3 secrets") {
		t.Errorf("roundtrip: expected 3 added, got:\n%s", buf.String())
	}

	v3, _ := vault.Open(dir2, key2)
	secrets := v3.ListSecrets()
	if len(secrets) != 3 {
		t.Errorf("expected 3 secrets after roundtrip, got %d", len(secrets))
	}
	names := make(map[string]bool)
	for _, s := range secrets {
		names[s.Name] = true
	}
	for _, name := range []string{"DB_PASS", "API_KEY", "STANDALONE"} {
		if !names[name] {
			t.Errorf("secret %q not found after roundtrip", name)
		}
	}
}
