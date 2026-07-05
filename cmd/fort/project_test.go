package main

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/youruser/fortbyte/internal/crypto"
	"github.com/youruser/fortbyte/internal/vault"
)

func TestProjectAddRequiresName(t *testing.T) {
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	rootCmd.SetArgs([]string{"project", "add"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing name argument")
	}
}

func TestProjectRemoveRequiresName(t *testing.T) {
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	rootCmd.SetArgs([]string{"project", "remove"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing name argument")
	}
}

func TestProjectListJSONEmpty(t *testing.T) {
	resetCmdFlags(t)
	setupTestVault(t)

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"project", "list", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("project list: %v", err)
	}
	var items []any
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if len(items) != 0 {
		t.Errorf("expected empty array, got %d items", len(items))
	}
}

func TestProjectListJSONPopulated(t *testing.T) {
	resetCmdFlags(t)
	dir := setupTestVault(t)
	salt, err := vault.GetSalt(dir)
	if err != nil {
		t.Fatalf("GetSalt: %v", err)
	}
	key := crypto.DeriveKey("password1234", salt)
	v, err := vault.Open(dir, key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	v.AddProject(vault.Project{Name: "alpha", Description: "first", URL: "https://alpha.example.com"})
	v.AddProject(vault.Project{Name: "beta", Description: "second"})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"project", "list", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("project list: %v", err)
	}
	var items []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(items))
	}
	if items[0]["name"] != "alpha" {
		t.Errorf("first project name = %v, want alpha", items[0]["name"])
	}
	if items[0]["description"] != "first" {
		t.Errorf("alpha description = %v, want first", items[0]["description"])
	}
	if items[0]["url"] != "https://alpha.example.com" {
		t.Errorf("alpha url = %v, want https://alpha.example.com", items[0]["url"])
	}
	if items[1]["name"] != "beta" {
		t.Errorf("second project name = %v, want beta", items[1]["name"])
	}
	uid, ok := items[0]["uid"].(string)
	if !ok || len(uid) < 16 {
		t.Errorf("uid should be a full string, got: %v", items[0]["uid"])
	}
}

func TestProjectShowJSON(t *testing.T) {
	resetCmdFlags(t)
	dir := setupTestVault(t)
	salt, err := vault.GetSalt(dir)
	if err != nil {
		t.Fatalf("GetSalt: %v", err)
	}
	key := crypto.DeriveKey("password1234", salt)
	v, err := vault.Open(dir, key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	pUID, _ := v.AddProject(vault.Project{Name: "myapp", Description: "app", URL: "https://app.example.com", Notes: "notes here"})
	eUID, _ := v.AddEnvironment(vault.Environment{Name: "prod", ProjectUID: pUID})
	v.AddSecret(vault.Secret{Name: "DB_PASS", Value: "s3cret", ProjectUID: pUID, EnvironmentUID: eUID})
	v.AddSecret(vault.Secret{Name: "API_KEY", Value: "key123", ProjectUID: pUID})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"project", "show", "myapp", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("project show: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if got["name"] != "myapp" {
		t.Errorf("name = %v, want myapp", got["name"])
	}
	if got["description"] != "app" {
		t.Errorf("description = %v, want app", got["description"])
	}
	if got["url"] != "https://app.example.com" {
		t.Errorf("url = %v, want https://app.example.com", got["url"])
	}
	if got["notes"] != "notes here" {
		t.Errorf("notes = %v, want notes here", got["notes"])
	}
	if got["created"] == nil || got["created"] == "" {
		t.Error("created should not be empty")
	}
	if got["updated"] == nil || got["updated"] == "" {
		t.Error("updated should not be empty")
	}
	envs, ok := got["environments"].([]any)
	if !ok {
		t.Fatalf("environments missing or wrong type: %v", got["environments"])
	}
	if len(envs) != 1 {
		t.Errorf("expected 1 environment, got %d", len(envs))
	}
	env := envs[0].(map[string]any)
	if env["name"] != "prod" {
		t.Errorf("env name = %v, want prod", env["name"])
	}
	if got["secret_count"] != float64(2) {
		t.Errorf("secret_count = %v, want 2", got["secret_count"])
	}
}

func TestProjectShowJSONNotFound(t *testing.T) {
	resetCmdFlags(t)
	setupTestVault(t)

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	rootCmd.SetArgs([]string{"project", "show", "nonexistent", "--format", "json"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestProjectShowJSONEmptyEnvs(t *testing.T) {
	resetCmdFlags(t)
	dir := setupTestVault(t)
	salt, err := vault.GetSalt(dir)
	if err != nil {
		t.Fatalf("GetSalt: %v", err)
	}
	key := crypto.DeriveKey("password1234", salt)
	v, err := vault.Open(dir, key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	v.AddProject(vault.Project{Name: "empty-project"})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"project", "show", "empty-project", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("project show: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	envs, ok := got["environments"].([]any)
	if !ok {
		t.Fatalf("environments missing or wrong type (nil = null): %v", got["environments"])
	}
	if len(envs) != 0 {
		t.Errorf("expected 0 environments, got %d", len(envs))
	}
}
