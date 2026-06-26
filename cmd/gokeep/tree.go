package main

import (
	"sort"

	"github.com/youruser/gokeep/internal/vault"
)

// shortUID returns the first 8 chars of a UID.
func shortUID(uid string) string {
	if len(uid) > 8 {
		return uid[:8]
	}
	return uid
}

// sortedProjectKeys returns project UIDs sorted by project name.
func sortedProjectKeys(projects map[string]vault.Project) []string {
	type kv struct {
		uid  string
		name string
	}
	items := make([]kv, 0, len(projects))
	for uid, p := range projects {
		items = append(items, kv{uid: uid, name: p.Name})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].name < items[j].name })
	keys := make([]string, len(items))
	for i, item := range items {
		keys[i] = item.uid
	}
	return keys
}

// sortedEnvKeys returns env UIDs sorted by env name.
func sortedEnvKeys(envs map[string]vault.Environment) []string {
	type kv struct {
		uid  string
		name string
	}
	items := make([]kv, 0, len(envs))
	for uid, e := range envs {
		items = append(items, kv{uid: uid, name: e.Name})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].name < items[j].name })
	keys := make([]string, len(items))
	for i, item := range items {
		keys[i] = item.uid
	}
	return keys
}

// sortedSecretKeys returns secret UIDs sorted by secret name.
func sortedSecretKeys(secrets map[string]vault.Secret) []string {
	type kv struct {
		uid  string
		name string
	}
	items := make([]kv, 0, len(secrets))
	for uid, s := range secrets {
		items = append(items, kv{uid: uid, name: s.Name})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].name < items[j].name })
	keys := make([]string, len(items))
	for i, item := range items {
		keys[i] = item.uid
	}
	return keys
}
