package main

import (
	"testing"
)

func TestRootCommandStructure(t *testing.T) {
	subs := rootCmd.Commands()
	names := make(map[string]bool)
	for _, c := range subs {
		names[c.Name()] = true
	}
	expected := []string{"init", "lock", "reset", "list", "project", "env", "secret", "status", "find", "export", "import", "config", "generate"}
	for _, n := range expected {
		if !names[n] {
			t.Errorf("root missing subcommand: %s", n)
		}
	}
}

func TestProjectSubcommands(t *testing.T) {
	subs := projectCmd.Commands()
	names := make(map[string]bool)
	for _, c := range subs {
		names[c.Name()] = true
	}
	expected := []string{"add", "edit", "remove", "list", "show"}
	for _, n := range expected {
		if !names[n] {
			t.Errorf("project missing subcommand: %s", n)
		}
	}
	if len(subs) != len(expected) {
		t.Errorf("project has %d subcommands, want %d", len(subs), len(expected))
	}
}

func TestEnvSubcommands(t *testing.T) {
	subs := envCmd.Commands()
	names := make(map[string]bool)
	for _, c := range subs {
		names[c.Name()] = true
	}
	expected := []string{"add", "edit", "remove", "list", "show"}
	for _, n := range expected {
		if !names[n] {
			t.Errorf("env missing subcommand: %s", n)
		}
	}
	if len(subs) != len(expected) {
		t.Errorf("env has %d subcommands, want %d", len(subs), len(expected))
	}

	flag := envCmd.PersistentFlags().Lookup("project")
	if flag == nil {
		t.Error("envCmd missing persistent --project flag")
	}
}

func TestSecretSubcommands(t *testing.T) {
	subs := secretCmd.Commands()
	names := make(map[string]bool)
	for _, c := range subs {
		names[c.Name()] = true
	}
	expected := []string{"add", "edit", "remove", "list", "reveal", "show", "move", "copy"}
	for _, n := range expected {
		if !names[n] {
			t.Errorf("secret missing subcommand: %s", n)
		}
	}
	if len(subs) != len(expected) {
		t.Errorf("secret has %d subcommands, want %d", len(subs), len(expected))
	}

	flag := secretCmd.PersistentFlags().Lookup("project")
	if flag == nil {
		t.Error("secretCmd missing persistent --project flag")
	}
}

func TestStatusSubcommand(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "status" {
			found = true
			break
		}
	}
	if !found {
		t.Error("status subcommand not registered with rootCmd")
	}
}
