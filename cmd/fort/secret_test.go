package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/youruser/fortbyte/internal/crypto"
	"github.com/youruser/fortbyte/internal/vault"
)

func TestSecretAddFlags(t *testing.T) {
	if secretCmd.PersistentFlags().Lookup("project") == nil {
		t.Error("secret missing persistent --project flag")
	}
	localFlags := []string{"env", "value", "url", "notes", "generate"}
	for _, name := range localFlags {
		if secretAddCmd.Flags().Lookup(name) == nil {
			t.Errorf("secret add missing flag: %s", name)
		}
	}
}

func TestSecretMoveFlags(t *testing.T) {
	if secretMoveCmd.Flags().Lookup("dest-project") == nil {
		t.Error("secret move missing --dest-project flag")
	}
	if secretMoveCmd.Flags().Lookup("dest-env") == nil {
		t.Error("secret move missing --dest-env flag")
	}
	if secretMoveCmd.Flags().Lookup("env") == nil {
		t.Error("secret move missing --env flag")
	}
	if secretMoveCmd.Flags().Lookup("dest-project").Annotations[cobra.BashCompOneRequiredFlag] == nil {
		t.Error("--dest-project should be required")
	}
}

func TestSecretCopyFlags(t *testing.T) {
	if secretCopyCmd.Flags().Lookup("dest-project") == nil {
		t.Error("secret copy missing --dest-project flag")
	}
	if secretCopyCmd.Flags().Lookup("dest-env") == nil {
		t.Error("secret copy missing --dest-env flag")
	}
	if secretCopyCmd.Flags().Lookup("env") == nil {
		t.Error("secret copy missing --env flag")
	}
	if secretCopyCmd.Flags().Lookup("name") == nil {
		t.Error("secret copy missing --name flag")
	}
	if secretCopyCmd.Flags().Lookup("dest-project").Annotations[cobra.BashCompOneRequiredFlag] == nil {
		t.Error("--dest-project should be required")
	}
}

func TestSecretMove(t *testing.T) {
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
	pSrcUID, _ := v.AddProject(vault.Project{Name: "src"})
	pDstUID, _ := v.AddProject(vault.Project{Name: "dst"})
	v.AddSecret(vault.Secret{Name: "MY_SECRET", Value: "s3cret", ProjectUID: pSrcUID, URL: "https://example.com", Notes: "test"})
	v.AddSecret(vault.Secret{Name: "DUP_SECRET", Value: "dup", ProjectUID: pSrcUID})
	v.AddSecret(vault.Secret{Name: "DUP_SECRET", Value: "dup", ProjectUID: pDstUID})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}
	vaultFile := filepath.Join(dir, vault.FileName)
	origVaultBytes, err := os.ReadFile(vaultFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		resetMoveCopyFlags(t)
	})

	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr string
	}{
		{
			"dest project required",
			[]string{"secret", "move", "DUP_SECRET", "--project", "dst"},
			"",
			"required flag(s) \"dest-project\" not set",
		},
		{
			"dest duplicate",
			[]string{"secret", "move", "DUP_SECRET", "--project", "dst", "--dest-project", "src"},
			"",
			"secret 'DUP_SECRET' already exists at destination",
		},
		{
			"move to different project",
			[]string{"secret", "move", "MY_SECRET", "--project", "src", "--dest-project", "dst"},
			"Secret 'MY_SECRET' moved to project 'dst'",
			"",
		},
		{
			"source not found",
			[]string{"secret", "move", "NONEXISTENT", "--project", "src", "--dest-project", "dst"},
			"",
			"secret 'NONEXISTENT' not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(vaultFile, origVaultBytes, 0600); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}
			resetMoveCopyFlags(t)

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
				t.Fatalf("execute: %v", err)
			}
			output := buf.String()
			if !strings.Contains(output, tt.want) {
				t.Errorf("output should contain %q, got:\n%s", tt.want, output)
			}
		})
	}
}

func TestSecretCopy(t *testing.T) {
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
	pSrcUID, _ := v.AddProject(vault.Project{Name: "src"})
	_, _ = v.AddProject(vault.Project{Name: "dst"})
	v.AddSecret(vault.Secret{Name: "MY_SECRET", Value: "s3cret", ProjectUID: pSrcUID, URL: "https://example.com", Notes: "test"})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}
	vaultFile := filepath.Join(dir, vault.FileName)
	origVaultBytes, err := os.ReadFile(vaultFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		resetMoveCopyFlags(t)
	})

	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr string
	}{
		{
			"dest project required",
			[]string{"secret", "copy", "MY_SECRET"},
			"",
			"required flag(s) \"dest-project\" not set",
		},
		{
			"dest duplicate",
			[]string{"secret", "copy", "MY_SECRET", "--project", "src", "--dest-project", "src"},
			"",
			"secret 'MY_SECRET' already exists at destination",
		},
		{
			"copy to different project",
			[]string{"secret", "copy", "MY_SECRET", "--project", "src", "--dest-project", "dst"},
			"Secret 'MY_SECRET' copied as 'MY_SECRET' to project 'dst'",
			"",
		},
		{
			"copy with rename",
			[]string{"secret", "copy", "MY_SECRET", "--project", "src", "--dest-project", "dst", "--name", "RENAMED"},
			"Secret 'MY_SECRET' copied as 'RENAMED' to project 'dst'",
			"",
		},
		{
			"source not found",
			[]string{"secret", "copy", "NONEXISTENT", "--project", "src", "--dest-project", "dst"},
			"",
			"secret 'NONEXISTENT' not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(vaultFile, origVaultBytes, 0600); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}
			resetMoveCopyFlags(t)

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
				t.Fatalf("execute: %v", err)
			}
			output := buf.String()
			if !strings.Contains(output, tt.want) {
				t.Errorf("output should contain %q, got:\n%s", tt.want, output)
			}
		})
	}
}

func TestSecretMoveWithEnv(t *testing.T) {
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
	devUID, _ := v.AddEnvironment(vault.Environment{Name: "dev", ProjectUID: pUID})
	_, _ = v.AddEnvironment(vault.Environment{Name: "prod", ProjectUID: pUID})
	v.AddSecret(vault.Secret{Name: "MY_SECRET", Value: "s3cret", ProjectUID: pUID, EnvironmentUID: devUID})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		resetMoveCopyFlags(t)
	})

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"secret", "move", "MY_SECRET", "--project", "myapp", "--env", "dev", "--dest-project", "myapp", "--dest-env", "prod"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "moved to project 'myapp' (env: 'prod')") {
		t.Errorf("expected env context in output, got:\n%s", output)
	}
}

func TestSecretEnvRequiresProject(t *testing.T) {
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	rootCmd.SetArgs([]string{"secret", "list", "--env", "prod"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error running secret list --env without --project")
	}
	if !strings.Contains(err.Error(), "requires") {
		t.Errorf("error should mention 'requires', got: %v", err)
	}
}

func TestSecretListFilter(t *testing.T) {
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
	v.AddSecret(vault.Secret{Name: "GitHub Token", Value: "x", URL: "https://github.com", Notes: "PAT"})
	v.AddSecret(vault.Secret{Name: "AWS Key", Value: "x", URL: "https://aws.amazon.com", Notes: "IAM"})
	v.AddSecret(vault.Secret{Name: "Database", Value: "x", Notes: "postgres"})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	tests := []struct {
		name    string
		filter  string
		want    string
		wantNot string
	}{
		{"filter by name", "github", "GitHub Token", "AWS Key"},
		{"filter by URL", "aws", "AWS Key", "GitHub Token"},
		{"filter by notes", "postgres", "Database", "AWS Key"},
		{"no match", "nonexistent", "No secrets.", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(io.Discard)
			rootCmd.SetArgs([]string{"secret", "list", "--filter", tt.filter})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("secret list: %v", err)
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

func TestSecretRevealJSON(t *testing.T) {
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
	v.AddSecret(vault.Secret{
		Name: "DB_PASS", Value: "s3cret", ProjectUID: pUID, EnvironmentUID: eUID,
		URL: "https://db.example.com", Notes: "database",
	})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"secret", "reveal", "DB_PASS", "--project", "myapp", "--env", "prod", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("reveal: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, buf.String())
	}
	if got["name"] != "DB_PASS" {
		t.Errorf("name = %v, want DB_PASS", got["name"])
	}
	if got["value"] != "s3cret" {
		t.Errorf("value = %v, want s3cret", got["value"])
	}
	if got["project"] != "myapp" {
		t.Errorf("project = %v, want myapp", got["project"])
	}
	if got["env"] != "prod" {
		t.Errorf("env = %v, want prod", got["env"])
	}
	if got["url"] != "https://db.example.com" {
		t.Errorf("url = %v", got["url"])
	}
	if got["notes"] != "database" {
		t.Errorf("notes = %v", got["notes"])
	}
}

func TestSecretRevealHumanStillWorks(t *testing.T) {
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
	v.AddSecret(vault.Secret{Name: "MY_SECRET", Value: "val", ProjectUID: pUID})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"secret", "reveal", "MY_SECRET", "--project", "myapp"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("reveal: %v", err)
	}
	output := stripANSI(buf.String())
	if !strings.Contains(output, "Name:") || !strings.Contains(output, "MY_SECRET") {
		t.Errorf("human output missing name, got:\n%s", output)
	}
	if !strings.Contains(output, "Value:") || !strings.Contains(output, "val") {
		t.Errorf("human output missing value, got:\n%s", output)
	}
	if strings.Contains(output, "{") {
		t.Errorf("human output contains JSON: %s", output)
	}
}

func TestSecretListJSONEmpty(t *testing.T) {
	resetCmdFlags(t)
	setupTestVault(t)

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"secret", "list", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("secret list: %v", err)
	}
	var items []any
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if len(items) != 0 {
		t.Errorf("expected empty array, got %d items", len(items))
	}
}

func TestSecretShowJSON(t *testing.T) {
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
	secretUID, _ := v.AddSecret(vault.Secret{
		Name: "DB_PASS", Value: "s3cret", ProjectUID: pUID, EnvironmentUID: eUID,
		URL: "https://db.example.com", Notes: "database",
	})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"secret", "show", "DB_PASS", "--project", "myapp", "--env", "prod", "--format", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("secret show: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if got["name"] != "DB_PASS" {
		t.Errorf("name = %v, want DB_PASS", got["name"])
	}
	if got["project"] != "myapp" {
		t.Errorf("project = %v, want myapp", got["project"])
	}
	if got["env"] != "prod" {
		t.Errorf("env = %v, want prod", got["env"])
	}
	if got["url"] != "https://db.example.com" {
		t.Errorf("url = %v", got["url"])
	}
	if got["notes"] != "database" {
		t.Errorf("notes = %v", got["notes"])
	}
	uid, ok := got["uid"].(string)
	if !ok {
		t.Errorf("uid missing or not a string: %v", got["uid"])
	} else if uid != secretUID {
		t.Errorf("uid = %q, want full uid %q", uid, secretUID)
	}
}

func TestSecretAddGenerate(t *testing.T) {
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
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"secret", "add", "AUTO_SECRET", "--generate", "--url", "", "--notes", ""})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("secret add --generate: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "added") {
		t.Errorf("expected success message, got: %s", output)
	}

	v2, err := vault.Open(dir, crypto.DeriveKey("password1234", salt))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	s, _, found := v2.FindSecretByName("AUTO_SECRET", "", "")
	if !found {
		t.Fatal("AUTO_SECRET not found after add --generate")
	}
	if len(s.Value) != 24 {
		t.Errorf("generated value length = %d, want 24", len(s.Value))
	}
}

func TestSecretRevealClipFlagExists(t *testing.T) {
	if secretRevealCmd.Flags().Lookup("clip") == nil {
		t.Error("secret reveal missing --clip flag")
	}
}

func TestSecretRevealClip(t *testing.T) {
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
	v.AddSecret(vault.Secret{Name: "CLIP_SECRET", Value: "clipboard-value", ProjectUID: pUID})
	if err := v.Save(dir, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"secret", "reveal", "CLIP_SECRET", "--project", "myapp", "--clip"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("reveal --clip: %v", err)
	}
	output := outBuf.String()
	if !strings.Contains(output, "Value:") {
		t.Errorf("expected value in output, got: %s", output)
	}
}
