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
