package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ul0gic/ctsnare/internal/domain"
)

// timestampFormat is the ISO 8601 format used for storing timestamps in SQLite.
const timestampFormat = "2006-01-02T15:04:05Z"

// UpsertHit inserts or replaces a hit keyed on domain (deduplication).
// Keywords and SANDomains are stored as JSON arrays.
func (d *DB) UpsertHit(ctx context.Context, hit domain.Hit) error {
	keywords, err := json.Marshal(hit.Keywords)
	if err != nil {
		return fmt.Errorf("marshaling keywords: %w", err)
	}
	sanDomains, err := json.Marshal(hit.SANDomains)
	if err != nil {
		return fmt.Errorf("marshaling SAN domains: %w", err)
	}

	now := time.Now().UTC().Format(timestampFormat)

	const query = `
		INSERT INTO hits (domain, score, severity, keywords, issuer, issuer_cn, san_domains,
			cert_not_before, ct_log, profile, session, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(domain) DO UPDATE SET
			score = excluded.score,
			severity = excluded.severity,
			keywords = excluded.keywords,
			issuer = excluded.issuer,
			issuer_cn = excluded.issuer_cn,
			san_domains = excluded.san_domains,
			cert_not_before = excluded.cert_not_before,
			ct_log = excluded.ct_log,
			profile = excluded.profile,
			session = excluded.session,
			updated_at = excluded.updated_at
	`

	_, err = d.db.ExecContext(ctx, query,
		hit.Domain,
		hit.Score,
		string(hit.Severity),
		string(keywords),
		hit.Issuer,
		hit.IssuerCN,
		string(sanDomains),
		hit.CertNotBefore.UTC().Format(timestampFormat),
		hit.CTLog,
		hit.Profile,
		hit.Session,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("upserting hit for %s: %w", hit.Domain, err)
	}
	return nil
}

// InsertHit inserts a new hit. Returns an error if the domain already exists.
func (d *DB) InsertHit(ctx context.Context, hit domain.Hit) error {
	keywords, err := json.Marshal(hit.Keywords)
	if err != nil {
		return fmt.Errorf("marshaling keywords: %w", err)
	}
	sanDomains, err := json.Marshal(hit.SANDomains)
	if err != nil {
		return fmt.Errorf("marshaling SAN domains: %w", err)
	}

	now := time.Now().UTC().Format(timestampFormat)

	const query = `
		INSERT INTO hits (domain, score, severity, keywords, issuer, issuer_cn, san_domains,
			cert_not_before, ct_log, profile, session, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = d.db.ExecContext(ctx, query,
		hit.Domain,
		hit.Score,
		string(hit.Severity),
		string(keywords),
		hit.Issuer,
		hit.IssuerCN,
		string(sanDomains),
		hit.CertNotBefore.UTC().Format(timestampFormat),
		hit.CTLog,
		hit.Profile,
		hit.Session,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("inserting hit for %s: %w", hit.Domain, err)
	}
	return nil
}

// QueryHits builds and executes a dynamic SQL query from the filter fields.
// All filter criteria use parameterized queries to prevent SQL injection.
func (d *DB) QueryHits(ctx context.Context, filter domain.QueryFilter) ([]domain.Hit, error) {
	var where []string
	var args []interface{}

	if filter.Keyword != "" {
		where = append(where, "keywords LIKE ?")
		args = append(args, "%"+filter.Keyword+"%")
	}
	if filter.ScoreMin > 0 {
		where = append(where, "score >= ?")
		args = append(args, filter.ScoreMin)
	}
	if filter.Severity != "" {
		where = append(where, "severity = ?")
		args = append(args, filter.Severity)
	}
	if filter.Session != "" {
		where = append(where, "session = ?")
		args = append(args, filter.Session)
	}
	if filter.Since > 0 {
		since := time.Now().Add(-filter.Since).UTC().Format(timestampFormat)
		where = append(where, "created_at >= ?")
		args = append(args, since)
	}
	if filter.TLD != "" {
		tld := filter.TLD
		if !strings.HasPrefix(tld, ".") {
			tld = "." + tld
		}
		where = append(where, "domain LIKE ?")
		args = append(args, "%"+tld)
	}

	query := "SELECT domain, score, severity, keywords, issuer, issuer_cn, san_domains, cert_not_before, ct_log, profile, session, created_at, updated_at FROM hits"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	// Sort clause.
	sortBy := "created_at"
	if filter.SortBy != "" {
		sortBy = sanitizeSortColumn(filter.SortBy)
	}
	sortDir := "DESC"
	if strings.EqualFold(filter.SortDir, "ASC") {
		sortDir = "ASC"
	}
	// SECURITY: sortBy is sanitized through sanitizeSortColumn() allowlist;
	// sortDir is limited to "ASC"/"DESC" by the check above. Both are safe
	// for direct interpolation. ORDER BY does not support parameterized placeholders.
	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortDir)

	// Pagination.
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying hits: %w", err)
	}
	defer rows.Close()

	var hits []domain.Hit
	for rows.Next() {
		hit, err := scanHit(rows)
		if err != nil {
			return nil, err
		}
		hits = append(hits, hit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating hit rows: %w", err)
	}

	return hits, nil
}

// scanHit reads a single row from a rows cursor into a domain.Hit.
// SQLite returns timestamps as strings, so we scan and parse them manually.
func scanHit(rows interface {
	Scan(dest ...interface{}) error
}) (domain.Hit, error) {
	var hit domain.Hit
	var severity string
	var keywordsJSON string
	var sanDomainsJSON string
	var certNotBeforeStr string
	var createdAtStr string
	var updatedAtStr string

	err := rows.Scan(
		&hit.Domain,
		&hit.Score,
		&severity,
		&keywordsJSON,
		&hit.Issuer,
		&hit.IssuerCN,
		&sanDomainsJSON,
		&certNotBeforeStr,
		&hit.CTLog,
		&hit.Profile,
		&hit.Session,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		return domain.Hit{}, fmt.Errorf("scanning hit row: %w", err)
	}

	hit.Severity = domain.Severity(severity)
	hit.CertNotBefore = parseTimestamp(certNotBeforeStr)
	hit.CreatedAt = parseTimestamp(createdAtStr)
	hit.UpdatedAt = parseTimestamp(updatedAtStr)

	if err := json.Unmarshal([]byte(keywordsJSON), &hit.Keywords); err != nil {
		return domain.Hit{}, fmt.Errorf("unmarshaling keywords: %w", err)
	}
	if err := json.Unmarshal([]byte(sanDomainsJSON), &hit.SANDomains); err != nil {
		return domain.Hit{}, fmt.Errorf("unmarshaling SAN domains: %w", err)
	}

	return hit, nil
}

// sanitizeSortColumn maps user-provided sort column names to safe SQL column
// names. Returns "created_at" for unrecognized inputs to prevent injection.
func sanitizeSortColumn(col string) string {
	allowed := map[string]string{
		"domain":     "domain",
		"score":      "score",
		"severity":   "severity",
		"session":    "session",
		"created_at": "created_at",
		"updated_at": "updated_at",
		"ct_log":     "ct_log",
		"profile":    "profile",
	}
	if safe, ok := allowed[strings.ToLower(col)]; ok {
		return safe
	}
	return "created_at"
}
