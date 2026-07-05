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

func TestFindProjectByName(t *testing.T) {
	v := &vault.Vault{
		Projects:     make(map[string]vault.Project),
		Environments: make(map[string]vault.Environment),
		Secrets:      make(map[string]vault.Secret),
	}
	if _, err := v.AddProject(vault.Project{Name: "myproject"}); err != nil {
		t.Fatalf("AddProject: %v", err)
	}
	if _, err := v.AddProject(vault.Project{Name: "other"}); err != nil {
		t.Fatalf("AddProject: %v", err)
	}

	tests := []struct {
		name      string
		projName  string
		wantFound bool
	}{
		{"present", "myproject", true},
		{"absent", "nonexistent", false},
		{"case sensitive", "MyProject", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _, found := findProjectByName(v, tt.projName)
			if found != tt.wantFound {
				t.Errorf("findProjectByName(%q) found=%v, want %v", tt.projName, found, tt.wantFound)
			}
			if found && p.Name != tt.projName {
				t.Errorf("expected name %q, got %q", tt.projName, p.Name)
			}
		})
	}
}

func TestFindEnvironmentByName(t *testing.T) {
	v := &vault.Vault{
		Projects:     make(map[string]vault.Project),
		Environments: make(map[string]vault.Environment),
		Secrets:      make(map[string]vault.Secret),
	}
	if _, err := v.AddProject(vault.Project{Name: "p1"}); err != nil {
		t.Fatalf("AddProject: %v", err)
	}
	if _, err := v.AddProject(vault.Project{Name: "p2"}); err != nil {
		t.Fatalf("AddProject: %v", err)
	}
	projUID1 := ""
	projUID2 := ""
	for uid, p := range v.ListProjects() {
		if p.Name == "p1" {
			projUID1 = uid
		} else {
			projUID2 = uid
		}
	}
	if _, err := v.AddEnvironment(vault.Environment{Name: "dev", ProjectUID: projUID1}); err != nil {
		t.Fatalf("AddEnvironment: %v", err)
	}
	if _, err := v.AddEnvironment(vault.Environment{Name: "prod", ProjectUID: projUID1}); err != nil {
		t.Fatalf("AddEnvironment: %v", err)
	}

	tests := []struct {
		name       string
		envName    string
		projectUID string
		wantFound  bool
	}{
		{"present in project", "dev", projUID1, true},
		{"present but different project", "dev", projUID2, false},
		{"absent", "staging", projUID1, false},
		{"scoping works", "prod", projUID1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, found := findEnvironmentByName(v, tt.envName, tt.projectUID)
			if found != tt.wantFound {
				t.Errorf("findEnvironmentByName(%q, %q) found=%v, want %v", tt.envName, tt.projectUID, found, tt.wantFound)
			}
		})
	}
}

func TestFindCmdRequiresPattern(t *testing.T) {
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	rootCmd.SetArgs([]string{"find"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing pattern argument")
	}
}

func TestFindCmdMatchesSecrets(t *testing.T) {
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
	v.AddSecret(vault.Secret{Name: "GitHub Token", ProjectUID: pUID, EnvironmentUID: eUID, Value: "x", URL: "https://github.com", Notes: "PAT"})
	v.AddSecret(vault.Secret{Name: "AWS Key", Value: "x", URL: "https://aws.amazon.com", Notes: "IAM"})
	v.AddSecret(vault.Secret{Name: "Database", Value: "x", Notes: "postgres connection"})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	tests := []struct {
		name    string
		args    []string
		want    string
		wantNot string
		wantErr string
	}{
		{"match by name", []string{"find", "GitHub"}, "GitHub Token", "", ""},
		{"match by URL", []string{"find", "aws.amazon"}, "AWS Key", "", ""},
		{"match by notes", []string{"find", "postgres"}, "Database", "", ""},
		{"case insensitive", []string{"find", "github token"}, "GitHub Token", "", ""},
		{"no match", []string{"find", "nonexistent"}, "No secrets matching 'nonexistent'", "", ""},
		{"match by project name", []string{"find", "GitHub", "--project", "myapp"}, "GitHub Token", "AWS Key", ""},
		{"match by project and env", []string{"find", "GitHub", "--project", "myapp", "--env", "prod"}, "GitHub Token", "AWS Key", ""},
		{"no match by wrong project", []string{"find", "GitHub", "--project", "nonexistent"}, "", "", "project 'nonexistent' not found"},
		{"no match by wrong env", []string{"find", "GitHub", "--project", "myapp", "--env", "nonexistent"}, "", "", "environment 'nonexistent' not found"},
		{"env without project (expect error)", []string{"find", "any_pattern", "--env", "prod"}, "", "", "--env requires --project"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetExportImportFlags()
			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(io.Discard)
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error should contain %q, got: %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("find: %v", err)
			}
			output := buf.String()
			if tt.want != "" && !strings.Contains(output, tt.want) {
				t.Errorf("output should contain %q, got:\n%s", tt.want, output)
			}
			if tt.wantNot != "" && strings.Contains(output, tt.wantNot) {
				t.Errorf("output should NOT contain %q, got:\n%s", tt.wantNot, output)
			}
		})
	}
}

func TestFindJSON(t *testing.T) {
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
	v.AddSecret(vault.Secret{Name: "GitHub API Key", ProjectUID: pUID, EnvironmentUID: eUID, Value: "x", URL: "https://github.com"})
	v.AddSecret(vault.Secret{Name: "AWS Key", Value: "x", URL: "https://aws.amazon.com"})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	tests := []struct {
		name    string
		args    []string
		wantLen int
		wantKey string
	}{
		{"find all", []string{"find", "key", "--format", "json"}, 2, ""},
		{"find scoped", []string{"find", "GitHub", "--project", "myapp", "--format", "json"}, 1, "GitHub API Key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetExportImportFlags()
			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(io.Discard)
			rootCmd.SetArgs(tt.args)
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("find: %v", err)
			}
			var items []map[string]any
			if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
				t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
			}
			if len(items) != tt.wantLen {
				t.Errorf("got %d items, want %d", len(items), tt.wantLen)
			}
			if tt.wantKey != "" && len(items) > 0 {
				if items[0]["name"] != tt.wantKey {
					t.Errorf("first item name = %v, want %s", items[0]["name"], tt.wantKey)
				}
			}
		})
	}
}

func TestFindEmptyJSON(t *testing.T) {
	resetCmdFlags(t)
	setupTestVault(t)

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"find", "nonexistent", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("find: %v", err)
	}
	var items []any
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if len(items) != 0 {
		t.Errorf("expected empty array, got %d items", len(items))
	}
}
