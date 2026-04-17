package migrations

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
)

// RunMigrations executes all pending SQL migration files
func RunMigrations(db *sql.DB) error {
	log.Info().Msg("Starting database migrations...")

	// Get migrations directory path
	migrationsPath := getMigrationsPath()

	// Read all migration files
	files, err := os.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Filter and sort SQL files
	var sqlFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			sqlFiles = append(sqlFiles, file.Name())
		}
	}
	sort.Strings(sqlFiles)

	// Execute each migration
	for _, filename := range sqlFiles {
		version := strings.TrimSuffix(filename, ".sql")

		// Check if migration already applied
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists)
		if err == nil && exists {
			log.Debug().Str("version", version).Msg("Migration already applied, skipping")
			continue
		}

		// Read migration file
		content, err := os.ReadFile(filepath.Join(migrationsPath, filename))
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		// Execute migration
		log.Info().Str("file", filename).Msg("Applying migration")
		_, err = db.Exec(string(content))
		if err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", filename, err)
		}

		log.Info().Str("file", filename).Msg("Migration applied successfully")
	}

	log.Info().Msg("All migrations completed successfully")
	return nil
}

// getMigrationsPath returns the path to migrations directory
func getMigrationsPath() string {
	// Try relative path first (for local development)
	paths := []string{
		"./migrations",
		"./services/onboarding-service/migrations",
		"../migrations",
		"/app/migrations", // Docker container path
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Default to current directory
	return "./migrations"
}
