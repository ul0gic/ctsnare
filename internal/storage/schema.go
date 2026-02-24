package storage

// schemaSQL defines the SQLite table structure and indexes for ctsnare.
const schemaSQL = `
CREATE TABLE IF NOT EXISTS hits (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    domain         TEXT    NOT NULL UNIQUE,
    score          INTEGER NOT NULL,
    severity       TEXT    NOT NULL,
    keywords       TEXT    NOT NULL DEFAULT '[]',
    issuer         TEXT    DEFAULT '',
    issuer_cn      TEXT    DEFAULT '',
    san_domains    TEXT    DEFAULT '[]',
    cert_not_before DATETIME,
    ct_log         TEXT    DEFAULT '',
    profile        TEXT    DEFAULT '',
    session        TEXT    DEFAULT '',
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_hits_score      ON hits (score DESC);
CREATE INDEX IF NOT EXISTS idx_hits_domain     ON hits (domain);
CREATE INDEX IF NOT EXISTS idx_hits_session    ON hits (session);
CREATE INDEX IF NOT EXISTS idx_hits_created_at ON hits (created_at);
CREATE INDEX IF NOT EXISTS idx_hits_severity   ON hits (severity);
`

// migrationV2SQL adds enrichment and bookmark columns to the hits table.
// Each ALTER TABLE is wrapped in a check so it is idempotent -- re-running
// against an already-migrated database is a no-op. SQLite does not support
// ALTER TABLE ADD COLUMN IF NOT EXISTS, so we check the table_info pragma.
//
// New columns:
//   - is_live: whether the domain responded to an HTTP probe (0/1)
//   - resolved_ips: JSON array of DNS A/AAAA records
//   - hosting_provider: detected CDN/host from reverse DNS or IP range
//   - http_status: status code from the liveness probe
//   - live_checked_at: timestamp of the last enrichment probe
//   - bookmarked: user-flagged as interesting (0/1)
//
// New indexes:
//   - idx_hits_bookmarked: partial index on bookmarked=1 for fast bookmark queries
//   - idx_hits_is_live: index on is_live for liveness filtering
const migrationV2SQL = `
ALTER TABLE hits ADD COLUMN is_live INTEGER DEFAULT 0;
ALTER TABLE hits ADD COLUMN resolved_ips TEXT DEFAULT '[]';
ALTER TABLE hits ADD COLUMN hosting_provider TEXT DEFAULT '';
ALTER TABLE hits ADD COLUMN http_status INTEGER DEFAULT 0;
ALTER TABLE hits ADD COLUMN live_checked_at DATETIME;
ALTER TABLE hits ADD COLUMN bookmarked INTEGER DEFAULT 0;
`

// migrationV2IndexSQL creates indexes for the new columns.
// Separated from column additions so index creation can be idempotent via IF NOT EXISTS.
const migrationV2IndexSQL = `
CREATE INDEX IF NOT EXISTS idx_hits_bookmarked ON hits (bookmarked) WHERE bookmarked = 1;
CREATE INDEX IF NOT EXISTS idx_hits_is_live    ON hits (is_live);
`
