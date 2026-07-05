package main

import (
	"testing"

	"github.com/youruser/fortbyte/internal/vault"
)

func TestShortUID(t *testing.T) {
	tests := []struct {
		name     string
		uid      string
		expected string
	}{
		{"full uid", "abcdef1234567890abcdef1234567890", "abcdef123456"},
		{"short uid", "abc", "abc"},
		{"empty", "", ""},
		{"exactly 12", "123456789012", "123456789012"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shortUID(tt.uid)
			if got != tt.expected {
				t.Errorf("shortUID(%q) = %q, want %q", tt.uid, got, tt.expected)
			}
		})
	}
}

func TestSortedProjectKeys(t *testing.T) {
	tests := []struct {
		name     string
		projects map[string]vault.Project
		wantLen  int
	}{
		{"empty", map[string]vault.Project{}, 0},
		{
			"single",
			map[string]vault.Project{"uid1": {Name: "foo"}},
			1,
		},
		{
			"multiple sorted",
			map[string]vault.Project{
				"uid1": {Name: "c"},
				"uid2": {Name: "a"},
				"uid3": {Name: "b"},
			},
			3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortedKeysByName(tt.projects, func(p vault.Project) string { return p.Name })
			if len(got) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(got), tt.wantLen)
			}
			for i := 1; i < len(got); i++ {
				if tt.projects[got[i-1]].Name > tt.projects[got[i]].Name {
					t.Errorf("not sorted: %q > %q", tt.projects[got[i-1]].Name, tt.projects[got[i]].Name)
				}
			}
		})
	}
}

//nolint:dupl // same pattern for different types (project/env/secret)
func TestSortedEnvKeys(t *testing.T) {
	tests := []struct {
		name    string
		envs    map[string]vault.Environment
		wantLen int
	}{
		{"empty", map[string]vault.Environment{}, 0},
		{
			"multiple sorted",
			map[string]vault.Environment{
				"uid1": {Name: "prod"},
				"uid2": {Name: "dev"},
				"uid3": {Name: "staging"},
			},
			3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortedKeysByName(tt.envs, func(e vault.Environment) string { return e.Name })
			if len(got) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(got), tt.wantLen)
			}
			for i := 1; i < len(got); i++ {
				if tt.envs[got[i-1]].Name > tt.envs[got[i]].Name {
					t.Errorf("not sorted: %q > %q", tt.envs[got[i-1]].Name, tt.envs[got[i]].Name)
				}
			}
		})
	}
}

//nolint:dupl // same pattern for different types (project/env/secret)
func TestSortedSecretKeys(t *testing.T) {
	tests := []struct {
		name    string
		secrets map[string]vault.Secret
		wantLen int
	}{
		{"empty", map[string]vault.Secret{}, 0},
		{
			"multiple sorted",
			map[string]vault.Secret{
				"uid1": {Name: "api_key"},
				"uid2": {Name: "db_pass"},
				"uid3": {Name: "aws_key"},
			},
			3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortedKeysByName(tt.secrets, func(s vault.Secret) string { return s.Name })
			if len(got) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(got), tt.wantLen)
			}
			for i := 1; i < len(got); i++ {
				if tt.secrets[got[i-1]].Name > tt.secrets[got[i]].Name {
					t.Errorf("not sorted: %q > %q", tt.secrets[got[i-1]].Name, tt.secrets[got[i]].Name)
				}
			}
		})
	}
}
