// Package storage implements SQLite-backed persistence for ctsnare.
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ul0gic/ctsnare/internal/domainutil"

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

	// Run V3 migration (base_domain column for subdomain grouping).
	if err := runMigrationV3(sqlDB); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("running V3 migration: %w", err)
	}

	// Backfill base_domain for any rows where it is still empty.
	if err := backfillBaseDomain(sqlDB); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("backfilling base_domain: %w", err)
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

// runMigrationV3 adds the base_domain column for subdomain grouping.
// The ALTER TABLE is run individually; "duplicate column name" errors
// are silently ignored so the migration is idempotent.
func runMigrationV3(sqlDB *sql.DB) error {
	stmts := strings.Split(migrationV3SQL, ";")
	for _, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := sqlDB.Exec(stmt); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return fmt.Errorf("executing V3 migration statement: %w", err)
		}
	}

	if _, err := sqlDB.Exec(migrationV3IndexSQL); err != nil {
		return fmt.Errorf("creating V3 indexes: %w", err)
	}
	return nil
}

// backfillBaseDomain reads all rows with an empty base_domain and computes
// the base domain from the domain column. This runs on first startup after
// upgrading to the V3 schema and is a no-op on subsequent startups.
func backfillBaseDomain(sqlDB *sql.DB) error {
	rows, err := sqlDB.Query("SELECT domain FROM hits WHERE base_domain = '' OR base_domain IS NULL")
	if err != nil {
		return fmt.Errorf("querying rows for backfill: %w", err)
	}
	defer rows.Close()

	type update struct {
		domain     string
		baseDomain string
	}
	var updates []update

	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return fmt.Errorf("scanning domain for backfill: %w", err)
		}
		bd := domainutil.BaseDomain(d)
		updates = append(updates, update{domain: d, baseDomain: bd})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating backfill rows: %w", err)
	}

	if len(updates) == 0 {
		return nil
	}

	tx, err := sqlDB.Begin()
	if err != nil {
		return fmt.Errorf("beginning backfill transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	stmt, err := tx.Prepare("UPDATE hits SET base_domain = ? WHERE domain = ?")
	if err != nil {
		return fmt.Errorf("preparing backfill statement: %w", err)
	}
	defer stmt.Close() //nolint:errcheck // closing prepared statement in deferred cleanup

	for _, u := range updates {
		if _, err := stmt.Exec(u.baseDomain, u.domain); err != nil {
			return fmt.Errorf("backfilling base_domain for %s: %w", u.domain, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing backfill transaction: %w", err)
	}
	return nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	return d.db.Close()
}
