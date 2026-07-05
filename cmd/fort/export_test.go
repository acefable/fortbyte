package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportJSON(t *testing.T) {
	resetCmdFlags(t)
	dir := setupTestVault(t)
	seedVault(t, dir)

	outFile := filepath.Join(t.TempDir(), "export.json")
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"export", outFile})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("export: %v", err)
	}
	if !strings.Contains(buf.String(), "Exported 3 secrets") {
		t.Errorf("expected 3 secrets exported, got: %s", buf.String())
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var entries []exportEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	for _, e := range entries {
		switch e.Name {
		case "DB_PASS":
			if e.Value != "s3cret" {
				t.Errorf("DB_PASS value = %q, want %q", e.Value, "s3cret")
			}
			if e.Project != "myapp" {
				t.Errorf("DB_PASS project = %q, want %q", e.Project, "myapp")
			}
			if e.URL != "https://db.example.com" {
				t.Errorf("DB_PASS URL = %q", e.URL)
			}
		case "API_KEY", "STANDALONE":
			// ok
		default:
			t.Errorf("unexpected entry: %s", e.Name)
		}
	}
}

func TestExportJSONScoped(t *testing.T) {
	resetCmdFlags(t)
	dir := setupTestVault(t)
	seedVault(t, dir)

	outFile := filepath.Join(t.TempDir(), "scoped.json")
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"export", outFile, "--project", "myapp"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("export: %v", err)
	}
	if !strings.Contains(buf.String(), "Exported 2 secrets") {
		t.Errorf("expected 2 secrets, got: %s", buf.String())
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var entries []exportEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Project != "myapp" {
			t.Errorf("entry %q has project %q, want %q", e.Name, e.Project, "myapp")
		}
	}
}

func TestExportEnv(t *testing.T) {
	resetCmdFlags(t)
	dir := setupTestVault(t)
	seedVault(t, dir)

	outFile := filepath.Join(t.TempDir(), "export.env")
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"export", outFile})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("export: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "DB_PASS=s3cret") {
		t.Errorf("missing DB_PASS=s3cret in .env output:\n%s", content)
	}
	if !strings.Contains(content, "API_KEY=key123") {
		t.Errorf("missing API_KEY=key123 in .env output:\n%s", content)
	}
	if !strings.Contains(content, "# Project: myapp | Env: prod") {
		t.Errorf("missing scope comment in .env output:\n%s", content)
	}
}

func TestExportAutoDetect(t *testing.T) {
	resetCmdFlags(t)
	dir := setupTestVault(t)
	seedVault(t, dir)

	outJSON := filepath.Join(t.TempDir(), "auto.json")
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"export", outJSON})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("export json: %v", err)
	}
	data, _ := os.ReadFile(outJSON)
	var entries []exportEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Errorf("expected JSON output, got parse error: %v", err)
	}

	outEnv := filepath.Join(t.TempDir(), "auto.env")
	buf.Reset()
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"export", outEnv})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("export env: %v", err)
	}
	envData, _ := os.ReadFile(outEnv)
	if !strings.Contains(string(envData), "=") {
		t.Errorf("expected .env output with KEY=value, got:\n%s", envData)
	}
}

func TestExportUnknownFormat(t *testing.T) {
	resetCmdFlags(t)
	dir := setupTestVault(t)
	seedVault(t, dir)

	outFile := filepath.Join(t.TempDir(), "export.txt")
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"export", outFile})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for .txt extension")
	}
	if !strings.Contains(err.Error(), "cannot detect format") {
		t.Errorf("error should mention 'cannot detect format', got: %v", err)
	}
}

func TestExportEnvRequiresProject(t *testing.T) {
	resetCmdFlags(t)
	dir := setupTestVault(t)
	seedVault(t, dir)

	outFile := filepath.Join(t.TempDir(), "export.env")
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"export", outFile, "--env", "prod"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --env without --project")
	}
	if !strings.Contains(err.Error(), "--env requires --project") {
		t.Errorf("error should mention '--env requires --project', got: %v", err)
	}
}
