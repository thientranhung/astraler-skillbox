package repositories

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"

	"github.com/astraler/skillbox/core-go/migrations"
)

// OpenDatabase opens (or creates) a SQLite database at path, applies required
// PRAGMAs, and runs pending migrations. Returns a ready-to-use *sql.DB.
func OpenDatabase(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := applyPragmas(db); err != nil {
		db.Close()
		return nil, err
	}

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func applyPragmas(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("pragma %q: %w", p, err)
		}
	}
	return nil
}

func runMigrations(db *sql.DB) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("migrations source: %w", err)
	}

	driver, err := migratesqlite.WithInstance(db, &migratesqlite.Config{})
	if err != nil {
		return fmt.Errorf("migrate driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
