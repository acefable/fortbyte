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

func TestEnvAddRequiresProject(t *testing.T) {
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	rootCmd.SetArgs([]string{"env", "add", "foo"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error running env add without --project")
	}
	if !strings.Contains(err.Error(), "project") {
		t.Errorf("error should mention 'project', got: %v", err)
	}
}

func TestEnvListAllowsNoArgs(t *testing.T) {
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	mockReadPassword(t, "test-password-1234", nil)
	rootCmd.SetArgs([]string{"env", "list"})
	err := rootCmd.Execute()
	if err == nil {
		t.Skip("vault exists in test environment — ok")
	}
	if strings.Contains(err.Error(), "accepts") {
		t.Errorf("env list should not require args, got: %v", err)
	}
}

func TestEnvListJSONEmpty(t *testing.T) {
	resetCmdFlags(t)
	setupTestVault(t)

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"env", "list", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("env list: %v", err)
	}
	var items []any
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if len(items) != 0 {
		t.Errorf("expected empty array, got %d items", len(items))
	}
}

func TestEnvListJSONPopulated(t *testing.T) {
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
	p1UID, _ := v.AddProject(vault.Project{Name: "myapp"})
	p2UID, _ := v.AddProject(vault.Project{Name: "other"})
	v.AddEnvironment(vault.Environment{Name: "prod", ProjectUID: p1UID})
	v.AddEnvironment(vault.Environment{Name: "dev", ProjectUID: p1UID})
	v.AddEnvironment(vault.Environment{Name: "staging", ProjectUID: p2UID})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"env", "list", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("env list: %v", err)
	}
	var items []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 envs, got %d", len(items))
	}
	for _, item := range items {
		if item["name"] == nil || item["name"] == "" {
			t.Errorf("env missing name: %v", item)
		}
		if item["uid"] == nil || item["uid"] == "" {
			t.Errorf("env missing uid: %v", item)
		}
		if item["project"] == nil || item["project"] == "" {
			t.Errorf("env missing project: %v", item)
		}
	}
}

func TestEnvListJSONScoped(t *testing.T) {
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
	p1UID, _ := v.AddProject(vault.Project{Name: "myapp"})
	p2UID, _ := v.AddProject(vault.Project{Name: "other"})
	v.AddEnvironment(vault.Environment{Name: "prod", ProjectUID: p1UID})
	v.AddEnvironment(vault.Environment{Name: "dev", ProjectUID: p1UID})
	v.AddEnvironment(vault.Environment{Name: "staging", ProjectUID: p2UID})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"env", "list", "--project", "myapp", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("env list: %v", err)
	}
	var items []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 envs in myapp, got %d", len(items))
	}
	for _, item := range items {
		if item["project"] != "myapp" {
			t.Errorf("env project = %v, want myapp", item["project"])
		}
	}
}

func TestEnvListJSONProjectNotFound(t *testing.T) {
	resetCmdFlags(t)
	setupTestVault(t)

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"env", "list", "--project", "nonexistent", "--format", "json"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestEnvShowJSON(t *testing.T) {
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
	pUID, _ := v.AddProject(vault.Project{Name: "myapp"})
	eUID, _ := v.AddEnvironment(vault.Environment{Name: "prod", ProjectUID: pUID, Description: "production", Notes: "be careful"})
	v.AddSecret(vault.Secret{Name: "DB_PASS", Value: "s3cret", ProjectUID: pUID, EnvironmentUID: eUID})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"env", "show", "prod", "--project", "myapp", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("env show: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if got["name"] != "prod" {
		t.Errorf("name = %v, want prod", got["name"])
	}
	if got["project"] != "myapp" {
		t.Errorf("project = %v, want myapp", got["project"])
	}
	if got["description"] != "production" {
		t.Errorf("description = %v, want production", got["description"])
	}
	if got["notes"] != "be careful" {
		t.Errorf("notes = %v, want be careful", got["notes"])
	}
	if got["created"] == nil || got["created"] == "" {
		t.Error("created should not be empty")
	}
	secrets, ok := got["secrets"].([]any)
	if !ok {
		t.Fatalf("secrets missing or wrong type: %v", got["secrets"])
	}
	if len(secrets) != 1 {
		t.Errorf("expected 1 secret, got %d", len(secrets))
	}
}

func TestEnvShowJSONEmptySecrets(t *testing.T) {
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
	pUID, _ := v.AddProject(vault.Project{Name: "myapp"})
	v.AddEnvironment(vault.Environment{Name: "empty-env", ProjectUID: pUID})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"env", "show", "empty-env", "--project", "myapp", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("env show: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	secrets, ok := got["secrets"].([]any)
	if !ok {
		t.Fatalf("secrets missing or wrong type (nil = null): %v", got["secrets"])
	}
	if len(secrets) != 0 {
		t.Errorf("expected 0 secrets, got %d", len(secrets))
	}
}

func TestEnvListWithoutProjectFlag(t *testing.T) {
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
	p1UID, _ := v.AddProject(vault.Project{Name: "app1"})
	p2UID, _ := v.AddProject(vault.Project{Name: "app2"})
	v.AddEnvironment(vault.Environment{Name: "dev", ProjectUID: p1UID})
	v.AddEnvironment(vault.Environment{Name: "prod", ProjectUID: p2UID})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"env", "list", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("env list: %v", err)
	}
	var items []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 envs across all projects, got %d", len(items))
	}
}
