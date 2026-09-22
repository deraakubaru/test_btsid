package database

import (
	"errors"
	"fmt"
	"io/fs"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// RunMigrations applies pending SQL schema migrations from an embedded file system.
// Fails fast if any migration error occurs (excluding ErrNoChange).
func RunMigrations(migrationFS fs.FS, dbURL string) error {
	driver, err := iofs.New(migrationFS, ".")
	if err != nil {
		return fmt.Errorf("failed to create iofs migration driver: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", driver, dbURL)
	if err != nil {
		return fmt.Errorf("failed to initialize migration instance: %w", err)
	}
	defer m.Close()

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migration execution failed: %w", err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		log.Println("Database schema is up to date (no pending migrations)")
	} else {
		log.Println("Database migrations applied successfully")
	}

	return nil
}
