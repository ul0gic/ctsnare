// Package storage implements SQLite-backed persistence for ctsnare.
package storage

import (
	"context"
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
func NewDB(dbPath string) (_ *DB, err error) {
	// One-shot startup setup runs against a background context; the caller's
	// request context is not yet available at construction time.
	ctx := context.Background()

	dir := filepath.Dir(dbPath)
	if err = os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("creating database directory %s: %w", dir, err)
	}

	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database %s: %w", dbPath, err)
	}
	// Close the connection if any setup step below fails. The close error is
	// intentionally subordinate to the primary setup error.
	defer func() {
		if err != nil {
			_ = sqlDB.Close()
		}
	}()

	if err = applyPragmas(ctx, sqlDB); err != nil {
		return nil, err
	}

	// Run schema creation.
	if _, err = sqlDB.ExecContext(ctx, schemaSQL); err != nil {
		return nil, fmt.Errorf("creating schema: %w", err)
	}

	if err = runMigrations(ctx, sqlDB); err != nil {
		return nil, err
	}

	return &DB{db: sqlDB}, nil
}

// applyPragmas configures the per-connection SQLite settings: WAL mode for
// concurrent access, a busy timeout so writers wait for locks instead of
// failing with SQLITE_BUSY, and foreign-key enforcement.
func applyPragmas(ctx context.Context, sqlDB *sql.DB) error {
	pragmas := []struct {
		stmt string
		desc string
	}{
		{"PRAGMA journal_mode=WAL", "enabling WAL mode"},
		{"PRAGMA busy_timeout=5000", "setting busy timeout"},
		{"PRAGMA foreign_keys=ON", "enabling foreign keys"},
	}
	for _, p := range pragmas {
		if _, err := sqlDB.ExecContext(ctx, p.stmt); err != nil {
			return fmt.Errorf("%s: %w", p.desc, err)
		}
	}
	return nil
}

// runMigrations applies the V2 and V3 schema migrations and backfills the
// base_domain column. All steps are idempotent and safe to re-run.
func runMigrations(ctx context.Context, sqlDB *sql.DB) error {
	// V2 migration (enrichment + bookmark columns).
	if err := runMigrationV2(ctx, sqlDB); err != nil {
		return fmt.Errorf("running V2 migration: %w", err)
	}
	// V3 migration (base_domain column for subdomain grouping).
	if err := runMigrationV3(ctx, sqlDB); err != nil {
		return fmt.Errorf("running V3 migration: %w", err)
	}
	// V4 migration (signals + category columns).
	if err := runMigrationV4(ctx, sqlDB); err != nil {
		return fmt.Errorf("running V4 migration: %w", err)
	}
	// Backfill base_domain for any rows where it is still empty.
	if err := backfillBaseDomain(ctx, sqlDB); err != nil {
		return fmt.Errorf("backfilling base_domain: %w", err)
	}
	return nil
}

// runMigrationV2 adds enrichment and bookmark columns to the hits table.
// Each ALTER TABLE is run individually; "duplicate column name" errors
// are silently ignored so the migration is idempotent.
func runMigrationV2(ctx context.Context, sqlDB *sql.DB) error {
	// Execute each ALTER TABLE statement individually.
	stmts := strings.Split(migrationV2SQL, ";")
	for _, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := sqlDB.ExecContext(ctx, stmt); err != nil {
			// SQLite returns "duplicate column name: X" when the column already exists.
			// This is expected on subsequent runs -- skip silently.
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return fmt.Errorf("executing migration statement: %w", err)
		}
	}

	// Create indexes (IF NOT EXISTS makes these naturally idempotent).
	if _, err := sqlDB.ExecContext(ctx, migrationV2IndexSQL); err != nil {
		return fmt.Errorf("creating V2 indexes: %w", err)
	}
	return nil
}

// runMigrationV3 adds the base_domain column for subdomain grouping.
func runMigrationV3(ctx context.Context, sqlDB *sql.DB) error {
	return runColumnMigration(ctx, sqlDB, "V3", migrationV3SQL, migrationV3IndexSQL)
}

// runMigrationV4 adds the signals + category columns.
func runMigrationV4(ctx context.Context, sqlDB *sql.DB) error {
	return runColumnMigration(ctx, sqlDB, "V4", migrationV4SQL, migrationV4IndexSQL)
}

// runColumnMigration applies a set of ALTER TABLE ADD COLUMN statements
// followed by index creation. Each ALTER is run individually; "duplicate column
// name" errors are silently ignored so the migration is idempotent and safe to
// re-run. SQLite lacks ALTER TABLE ADD COLUMN IF NOT EXISTS, so this duplicate
// check is the idempotency mechanism. label identifies the migration in errors.
func runColumnMigration(ctx context.Context, sqlDB *sql.DB, label, columnSQL, indexSQL string) error {
	for _, stmt := range strings.Split(columnSQL, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := sqlDB.ExecContext(ctx, stmt); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return fmt.Errorf("executing %s migration statement: %w", label, err)
		}
	}
	if _, err := sqlDB.ExecContext(ctx, indexSQL); err != nil {
		return fmt.Errorf("creating %s indexes: %w", label, err)
	}
	return nil
}

// baseDomainUpdate pairs a stored domain with its computed base domain.
type baseDomainUpdate struct {
	domain     string
	baseDomain string
}

// backfillBaseDomain reads all rows with an empty base_domain and computes
// the base domain from the domain column. This runs on first startup after
// upgrading to the V3 schema and is a no-op on subsequent startups.
func backfillBaseDomain(ctx context.Context, sqlDB *sql.DB) error {
	updates, err := collectBackfillUpdates(ctx, sqlDB)
	if err != nil {
		return err
	}
	if len(updates) == 0 {
		return nil
	}
	return applyBackfillUpdates(ctx, sqlDB, updates)
}

// collectBackfillUpdates loads every row missing a base_domain and computes the
// base domain for each.
func collectBackfillUpdates(ctx context.Context, sqlDB *sql.DB) ([]baseDomainUpdate, error) {
	rows, err := sqlDB.QueryContext(ctx, "SELECT domain FROM hits WHERE base_domain = '' OR base_domain IS NULL")
	if err != nil {
		return nil, fmt.Errorf("querying rows for backfill: %w", err)
	}
	defer rows.Close()

	var updates []baseDomainUpdate
	for rows.Next() {
		var d string
		if err = rows.Scan(&d); err != nil {
			return nil, fmt.Errorf("scanning domain for backfill: %w", err)
		}
		updates = append(updates, baseDomainUpdate{domain: d, baseDomain: domainutil.BaseDomain(d)})
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating backfill rows: %w", err)
	}
	return updates, nil
}

// applyBackfillUpdates writes the computed base domains in a single transaction.
func applyBackfillUpdates(ctx context.Context, sqlDB *sql.DB, updates []baseDomainUpdate) error {
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning backfill transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	stmt, err := tx.PrepareContext(ctx, "UPDATE hits SET base_domain = ? WHERE domain = ?")
	if err != nil {
		return fmt.Errorf("preparing backfill statement: %w", err)
	}
	defer stmt.Close() //nolint:errcheck // closing prepared statement in deferred cleanup

	for _, u := range updates {
		if _, err := stmt.ExecContext(ctx, u.baseDomain, u.domain); err != nil {
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
