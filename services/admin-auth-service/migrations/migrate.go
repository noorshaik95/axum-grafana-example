package migrations

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrations applies pending *.sql migration files in lexical order.
// Each file is applied inside a transaction and recorded in schema_migrations.
func RunMigrations(db *sql.DB, migrationsPath string) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	files, err := os.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("read migrations dir %s: %w", migrationsPath, err)
	}

	var toApply []string
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".sql") || strings.HasSuffix(f.Name(), ".down.sql") {
			continue
		}
		toApply = append(toApply, f.Name())
	}
	sort.Strings(toApply)

	for _, name := range toApply {
		version := strings.TrimSuffix(name, ".sql")

		var applied bool
		if err := db.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)",
			version,
		).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if applied {
			continue
		}
		if err := apply(db, migrationsPath, name, version); err != nil {
			return err
		}
		log.Printf("admin-auth-service: applied migration %s", version)
	}
	return nil
}

func apply(db *sql.DB, dir, name, version string) error {
	content, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return fmt.Errorf("read %s: %w", name, err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx %s: %w", version, err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(string(content)); err != nil {
		return fmt.Errorf("exec %s: %w", name, err)
	}
	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (version) VALUES ($1)", version,
	); err != nil {
		return fmt.Errorf("record %s: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit %s: %w", version, err)
	}
	return nil
}
