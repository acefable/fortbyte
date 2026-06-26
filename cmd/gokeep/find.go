package main

import (
	"github.com/youruser/gokeep/internal/vault"
)

// findProjectByName returns (project, uid, found).
func findProjectByName(v *vault.Vault, name string) (vault.Project, string, bool) {
	for uid, p := range v.ListProjects() {
		if p.Name == name {
			return p, uid, true
		}
	}
	return vault.Project{}, "", false
}

// findEnvironmentByName returns (env, uid, found) scoped to projectUID.
func findEnvironmentByName(v *vault.Vault, name, projectUID string) (vault.Environment, string, bool) {
	for uid, e := range v.ListEnvironments() {
		if e.Name == name && e.ProjectUID == projectUID {
			return e, uid, true
		}
	}
	return vault.Environment{}, "", false
}
