// Package storage implements SQLite-backed persistence for ctsnare.
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	// Pure Go SQLite driver -- no CGo, compiles into the binary.
	_ "modernc.org/sqlite"
)

// DB wraps a SQLite database connection and provides persistence operations
// for ctsnare hits.
type DB struct {
	db *sql.DB
}

// NewDB opens (or creates) a SQLite database at the given path, enables WAL
// mode for concurrent access, and runs the schema migration. Parent
// directories are created if they do not exist.
func NewDB(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("creating database directory %s: %w", dir, err)
	}

	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database %s: %w", dbPath, err)
	}

	// Enable WAL mode for crash safety and concurrent read/write.
	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("enabling WAL mode: %w", err)
	}

	// Set busy timeout so concurrent writers wait for locks instead of
	// immediately returning SQLITE_BUSY. Without this, poller goroutines
	// silently drop hits when write contention occurs.
	if _, err := sqlDB.Exec("PRAGMA busy_timeout=5000"); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("setting busy timeout: %w", err)
	}

	// Enable foreign key enforcement.
	if _, err := sqlDB.Exec("PRAGMA foreign_keys=ON"); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	// Run schema creation.
	if _, err := sqlDB.Exec(schemaSQL); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("creating schema: %w", err)
	}

	// Run V2 migration (enrichment + bookmark columns). Each ALTER TABLE
	// statement is executed individually so that already-existing columns
	// are silently skipped (idempotent).
	if err := runMigrationV2(sqlDB); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("running V2 migration: %w", err)
	}

	return &DB{db: sqlDB}, nil
}

// runMigrationV2 adds enrichment and bookmark columns to the hits table.
// Each ALTER TABLE is run individually; "duplicate column name" errors
// are silently ignored so the migration is idempotent.
func runMigrationV2(sqlDB *sql.DB) error {
	// Execute each ALTER TABLE statement individually.
	stmts := strings.Split(migrationV2SQL, ";")
	for _, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := sqlDB.Exec(stmt); err != nil {
			// SQLite returns "duplicate column name: X" when the column already exists.
			// This is expected on subsequent runs -- skip silently.
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return fmt.Errorf("executing migration statement: %w", err)
		}
	}

	// Create indexes (IF NOT EXISTS makes these naturally idempotent).
	if _, err := sqlDB.Exec(migrationV2IndexSQL); err != nil {
		return fmt.Errorf("creating V2 indexes: %w", err)
	}
	return nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	return d.db.Close()
}
