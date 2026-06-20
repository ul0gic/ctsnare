package storage

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

const migrationV2SQL = `
ALTER TABLE hits ADD COLUMN is_live INTEGER DEFAULT 0;
ALTER TABLE hits ADD COLUMN resolved_ips TEXT DEFAULT '[]';
ALTER TABLE hits ADD COLUMN hosting_provider TEXT DEFAULT '';
ALTER TABLE hits ADD COLUMN http_status INTEGER DEFAULT 0;
ALTER TABLE hits ADD COLUMN live_checked_at DATETIME;
ALTER TABLE hits ADD COLUMN bookmarked INTEGER DEFAULT 0;
`

const migrationV2IndexSQL = `
CREATE INDEX IF NOT EXISTS idx_hits_bookmarked ON hits (bookmarked) WHERE bookmarked = 1;
CREATE INDEX IF NOT EXISTS idx_hits_is_live    ON hits (is_live);
`

const migrationV3SQL = `
ALTER TABLE hits ADD COLUMN base_domain TEXT DEFAULT '';
`

const migrationV3IndexSQL = `
CREATE INDEX IF NOT EXISTS idx_hits_base_domain ON hits (base_domain);
`

const migrationV4SQL = `
ALTER TABLE hits ADD COLUMN signals TEXT DEFAULT '[]';
ALTER TABLE hits ADD COLUMN category TEXT DEFAULT '';
`

const migrationV4IndexSQL = `
CREATE INDEX IF NOT EXISTS idx_hits_category ON hits (category);
`
