// Package dbmigrate runs schema migrations against the application database.
//
// Migrations are SQL files embedded into the binary so that a deployed
// release contains exactly the migrations that match its code. The
// directory layout follows golang-migrate conventions:
//
//	migrations/
//	  000001_baseline.up.sql
//	  000001_baseline.down.sql
//	  000002_perf_indexes.up.sql
//	  000002_perf_indexes.down.sql
//
// Adding a new migration:
//  1. Pick the next number: NNNNNN_short_description.{up,down}.sql
//  2. Write idempotent SQL where reasonable (IF NOT EXISTS / IF EXISTS)
//  3. Test the round trip locally:
//     migrate -path internal/dbmigrate/migrations -database "$DATABASE_URL" up
//     migrate -path internal/dbmigrate/migrations -database "$DATABASE_URL" down -all
//
// The runtime never invokes the CLI — it calls Up() at server startup.
package dbmigrate

import (
	"embed"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Up applies all pending migrations against the database held by the given
// GORM DB. It does not call gorm.AutoMigrate — migrations are the only
// source of truth for schema changes. Returns nil when the DB is already
// at the latest version.
//
// Dirty state aborts startup: if a previous migration crashed partway, an
// operator must inspect and either roll forward manually or `force` the
// version. Silently retrying would risk inconsistent schema.
func Up(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}

	srcDriver, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("init embedded source: %w", err)
	}

	dbDriver, err := migratepg.WithInstance(sqlDB, &migratepg.Config{})
	if err != nil {
		return fmt.Errorf("init postgres driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", srcDriver, "postgres", dbDriver)
	if err != nil {
		return fmt.Errorf("new migrate instance: %w", err)
	}

	// Don't call m.Close() — that would close the shared sql.DB underneath
	// gorm. Instead release just the source driver. The database driver is
	// the same *sql.DB GORM keeps using.
	defer func() {
		if err := srcDriver.Close(); err != nil {
			slog.Warn("dbmigrate: source close failed", "error", err)
		}
	}()

	ver, dirty, err := m.Version()
	switch {
	case err == nil && dirty:
		return fmt.Errorf("database is in dirty state at version %d; manual intervention required", ver)
	case err != nil && !errors.Is(err, migrate.ErrNilVersion):
		return fmt.Errorf("read version: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	newVer, _, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("read post-apply version: %w", err)
	}
	slog.Info("dbmigrate: schema up to date", "version", newVer)
	return nil
}
