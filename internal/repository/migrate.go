package repository

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations applies pending database migrations using golang-migrate.
func RunMigrations(databaseURL, migrationsPath string) error {
	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		absPath = migrationsPath
	}

	// golang-migrate pgx/v5 driver registers as "pgx5", not "postgres".
	dbURL := strings.Replace(databaseURL, "postgres://", "pgx5://", 1)
	dbURL = strings.Replace(dbURL, "postgresql://", "pgx5://", 1)

	m, err := migrate.New("file://"+absPath, dbURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
