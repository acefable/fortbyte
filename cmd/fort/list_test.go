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

func TestListJSONEmpty(t *testing.T) {
	resetCmdFlags(t)
	setupTestVault(t)

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"list", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("list: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	projects, ok := got["projects"].([]any)
	if !ok {
		t.Fatalf("projects field missing or wrong type: %v", got["projects"])
	}
	if len(projects) != 0 {
		t.Errorf("expected empty projects array, got %d", len(projects))
	}
}

func TestListJSONPopulated(t *testing.T) {
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
	eUID1, _ := v.AddEnvironment(vault.Environment{Name: "prod", ProjectUID: pUID})
	eUID2, _ := v.AddEnvironment(vault.Environment{Name: "dev", ProjectUID: pUID})
	v.AddSecret(vault.Secret{Name: "SECRET_A", Value: "a", ProjectUID: pUID, EnvironmentUID: eUID1})
	v.AddSecret(vault.Secret{Name: "SECRET_B", Value: "b", ProjectUID: pUID, EnvironmentUID: eUID2})
	v.AddSecret(vault.Secret{Name: "STANDALONE_SECRET", Value: "c"})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"list", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("list: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	projects, ok := got["projects"].([]any)
	if !ok {
		t.Fatalf("projects field missing or wrong type: %v", got["projects"])
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	p := projects[0].(map[string]any)
	if p["name"] != "myapp" {
		t.Errorf("project name = %v, want myapp", p["name"])
	}
	envs, ok := p["envs"].([]any)
	if !ok {
		t.Fatalf("envs field missing or wrong type: %v", p["envs"])
	}
	if len(envs) != 2 {
		t.Fatalf("expected 2 envs, got %d", len(envs))
	}
	for _, e := range envs {
		env := e.(map[string]any)
		secrets, ok := env["secrets"].([]any)
		if !ok {
			t.Fatalf("secrets missing for env %v", env["name"])
		}
		if len(secrets) != 1 {
			t.Errorf("expected 1 secret in env %v, got %d", env["name"], len(secrets))
		}
	}
	standalone, ok := got["standalone"].([]any)
	if !ok {
		t.Fatalf("standalone field missing or wrong type: %v", got["standalone"])
	}
	if len(standalone) != 1 {
		t.Fatalf("expected 1 standalone secret, got %d", len(standalone))
	}
	s := standalone[0].(map[string]any)
	if s["name"] != "STANDALONE_SECRET" {
		t.Errorf("standalone name = %v, want STANDALONE_SECRET", s["name"])
	}
}

func TestListJSONEmptyEnvSecrets(t *testing.T) {
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
	v.AddEnvironment(vault.Environment{Name: "prod", ProjectUID: pUID})
	v.AddEnvironment(vault.Environment{Name: "empty-env", ProjectUID: pUID})
	v.AddSecret(vault.Secret{Name: "SECRET_A", Value: "a", ProjectUID: pUID})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"list", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("list: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	projects, ok := got["projects"].([]any)
	if !ok {
		t.Fatalf("projects missing: %v", got)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	p := projects[0].(map[string]any)
	envs, ok := p["envs"].([]any)
	if !ok {
		t.Fatalf("envs missing: %v", p)
	}
	for _, e := range envs {
		env := e.(map[string]any)
		secrets, ok := env["secrets"].([]any)
		if !ok {
			t.Fatalf("env %v: secrets missing or wrong type (nil = null): %v", env["name"], env["secrets"])
		}
		if env["name"] == "empty-env" && len(secrets) != 0 {
			t.Errorf("empty-env should have 0 secrets, got %d", len(secrets))
		}
	}
}

func TestListJSONScopedByProject(t *testing.T) {
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
	eUID1, _ := v.AddEnvironment(vault.Environment{Name: "prod", ProjectUID: p1UID})
	v.AddEnvironment(vault.Environment{Name: "staging", ProjectUID: p2UID})
	v.AddSecret(vault.Secret{Name: "SECRET_A", Value: "a", ProjectUID: p1UID, EnvironmentUID: eUID1})
	v.AddSecret(vault.Secret{Name: "SECRET_B", Value: "b", ProjectUID: p2UID})
	v.AddSecret(vault.Secret{Name: "STANDALONE", Value: "s"})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"list", "--project", "myapp", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("list: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	projects, ok := got["projects"].([]any)
	if !ok {
		t.Fatalf("projects missing: %v", got)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	p := projects[0].(map[string]any)
	if p["name"] != "myapp" {
		t.Errorf("project name = %v, want myapp", p["name"])
	}
	if _, ok := got["standalone"]; ok {
		t.Error("standalone should not appear when scoped to a project")
	}
}

func TestListJSONScopedByProjectAndEnv(t *testing.T) {
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
	prodUID, _ := v.AddEnvironment(vault.Environment{Name: "prod", ProjectUID: pUID})
	devUID, _ := v.AddEnvironment(vault.Environment{Name: "dev", ProjectUID: pUID})
	v.AddSecret(vault.Secret{Name: "PROD_SECRET", Value: "p", ProjectUID: pUID, EnvironmentUID: prodUID})
	v.AddSecret(vault.Secret{Name: "DEV_SECRET", Value: "d", ProjectUID: pUID, EnvironmentUID: devUID})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"list", "--project", "myapp", "--env", "prod", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("list: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	projects := got["projects"].([]any)
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	p := projects[0].(map[string]any)
	envs, ok := p["envs"].([]any)
	if !ok {
		t.Fatalf("envs missing: %v", p)
	}
	if len(envs) != 1 {
		t.Fatalf("expected 1 env, got %d", len(envs))
	}
	env := envs[0].(map[string]any)
	if env["name"] != "prod" {
		t.Errorf("env name = %v, want prod", env["name"])
	}
	secrets := env["secrets"].([]any)
	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret in prod, got %d", len(secrets))
	}
	sec := secrets[0].(map[string]any)
	if sec["name"] != "PROD_SECRET" {
		t.Errorf("secret name = %v, want PROD_SECRET", sec["name"])
	}
}

func TestListEnvRequiresProject(t *testing.T) {
	resetCmdFlags(t)
	setupTestVault(t)

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	rootCmd.SetArgs([]string{"list", "--env", "prod"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --env without --project")
	}
	if !strings.Contains(err.Error(), "--env requires --project") {
		t.Errorf("error should mention '--env requires --project', got: %v", err)
	}
}

func TestListJSONPopulatedProjectScopes(t *testing.T) {
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
	eUID, _ := v.AddEnvironment(vault.Environment{Name: "prod", ProjectUID: pUID})
	v.AddSecret(vault.Secret{Name: "SECRET_A", Value: "a", ProjectUID: pUID, EnvironmentUID: eUID})
	v.AddSecret(vault.Secret{Name: "STANDALONE", Value: "s"})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"list", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("list: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	standalone, ok := got["standalone"].([]any)
	if !ok {
		t.Fatalf("standalone missing: %v", got)
	}
	if len(standalone) != 1 {
		t.Errorf("expected 1 standalone, got %d", len(standalone))
	}

	buf.Reset()
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"list", "--project", "myapp", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("list scoped: %v", err)
	}
	var got2 map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got2); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if _, ok := got2["standalone"]; ok {
		t.Error("standalone should not appear when scoped to a project")
	}
}
