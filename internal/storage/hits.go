package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ul0gic/ctsnare/internal/domain"
	"github.com/ul0gic/ctsnare/internal/domainutil"
)

// timestampFormat is the ISO 8601 format used for storing timestamps in SQLite.
const timestampFormat = "2006-01-02T15:04:05Z"

// UpsertHit inserts or replaces a hit keyed on domain (deduplication).
// Keywords, SANDomains, and ResolvedIPs are stored as JSON arrays.
func (d *DB) UpsertHit(ctx context.Context, hit domain.Hit) error {
	cols, err := marshalHitColumns(hit)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(timestampFormat)

	const query = `
		INSERT INTO hits (domain, score, severity, keywords, signals, category, issuer, issuer_cn, san_domains,
			cert_not_before, ct_log, profile, session, created_at, updated_at,
			is_live, resolved_ips, hosting_provider, http_status, live_checked_at, bookmarked,
			base_domain)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(domain) DO UPDATE SET
			score = excluded.score,
			severity = excluded.severity,
			keywords = excluded.keywords,
			signals = excluded.signals,
			category = excluded.category,
			issuer = excluded.issuer,
			issuer_cn = excluded.issuer_cn,
			san_domains = excluded.san_domains,
			cert_not_before = excluded.cert_not_before,
			ct_log = excluded.ct_log,
			profile = excluded.profile,
			session = excluded.session,
			updated_at = excluded.updated_at,
			base_domain = excluded.base_domain
	`

	_, err = d.db.ExecContext(ctx, query,
		hit.Domain,
		hit.Score,
		string(hit.Severity),
		cols.keywords,
		cols.signals,
		hit.Category,
		hit.Issuer,
		hit.IssuerCN,
		cols.sanDomains,
		hit.CertNotBefore.UTC().Format(timestampFormat),
		hit.CTLog,
		hit.Profile,
		hit.Session,
		now,
		now,
		cols.isLive,
		cols.resolvedIPs,
		hit.HostingProvider,
		hit.HTTPStatus,
		cols.liveCheckedAt,
		cols.bookmarked,
		domainutil.BaseDomain(hit.Domain),
	)
	if err != nil {
		return fmt.Errorf("upserting hit for %s: %w", hit.Domain, err)
	}
	return nil
}

// hitColumns holds the serialized scalar forms of a Hit's compound fields,
// shared by InsertHit and UpsertHit to avoid duplicating the marshaling logic.
type hitColumns struct {
	keywords      string
	signals       string
	sanDomains    string
	resolvedIPs   string
	isLive        int
	bookmarked    int
	liveCheckedAt interface{}
}

// marshalHitColumns serializes a Hit's JSON-array and boolean fields into the
// scalar forms SQLite stores.
func marshalHitColumns(hit domain.Hit) (hitColumns, error) {
	keywords, err := json.Marshal(hit.Keywords)
	if err != nil {
		return hitColumns{}, fmt.Errorf("marshaling keywords: %w", err)
	}
	signals, err := json.Marshal(hit.Signals)
	if err != nil {
		return hitColumns{}, fmt.Errorf("marshaling signals: %w", err)
	}
	sanDomains, err := json.Marshal(hit.SANDomains)
	if err != nil {
		return hitColumns{}, fmt.Errorf("marshaling SAN domains: %w", err)
	}
	resolvedIPs, err := json.Marshal(hit.ResolvedIPs)
	if err != nil {
		return hitColumns{}, fmt.Errorf("marshaling resolved IPs: %w", err)
	}

	c := hitColumns{
		keywords:    string(keywords),
		signals:     string(signals),
		sanDomains:  string(sanDomains),
		resolvedIPs: string(resolvedIPs),
	}
	if hit.IsLive {
		c.isLive = 1
	}
	if hit.Bookmarked {
		c.bookmarked = 1
	}
	if !hit.LiveCheckedAt.IsZero() {
		c.liveCheckedAt = hit.LiveCheckedAt.UTC().Format(timestampFormat)
	}
	return c, nil
}

// InsertHit inserts a new hit. Returns an error if the domain already exists.
func (d *DB) InsertHit(ctx context.Context, hit domain.Hit) error {
	cols, err := marshalHitColumns(hit)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(timestampFormat)

	const query = `
		INSERT INTO hits (domain, score, severity, keywords, signals, category, issuer, issuer_cn, san_domains,
			cert_not_before, ct_log, profile, session, created_at, updated_at,
			is_live, resolved_ips, hosting_provider, http_status, live_checked_at, bookmarked,
			base_domain)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = d.db.ExecContext(ctx, query,
		hit.Domain,
		hit.Score,
		string(hit.Severity),
		cols.keywords,
		cols.signals,
		hit.Category,
		hit.Issuer,
		hit.IssuerCN,
		cols.sanDomains,
		hit.CertNotBefore.UTC().Format(timestampFormat),
		hit.CTLog,
		hit.Profile,
		hit.Session,
		now,
		now,
		cols.isLive,
		cols.resolvedIPs,
		hit.HostingProvider,
		hit.HTTPStatus,
		cols.liveCheckedAt,
		cols.bookmarked,
		domainutil.BaseDomain(hit.Domain),
	)
	if err != nil {
		return fmt.Errorf("inserting hit for %s: %w", hit.Domain, err)
	}
	return nil
}

// buildWhereClause translates the filter fields into parameterized SQL
// predicates and their bound arguments. All values are bound via placeholders
// to prevent SQL injection.
func buildWhereClause(filter domain.QueryFilter) (where []string, args []interface{}) {
	// Simple equality filters keyed on a non-empty string value.
	for _, f := range []struct {
		predicate string
		value     string
	}{
		{"severity = ?", filter.Severity},
		{"session = ?", filter.Session},
		{"base_domain = ?", filter.BaseDomain},
		{"category = ?", filter.Category},
	} {
		if f.value != "" {
			where = append(where, f.predicate)
			args = append(args, f.value)
		}
	}

	where, args = appendMatchPredicates(filter, where, args)
	where, args = appendSignalPredicates(filter, where, args)

	if pred := bookmarkedPredicate(filter.Bookmarked); pred != "" {
		where = append(where, pred)
	}
	if filter.LiveOnly {
		where = append(where, "is_live = 1")
	}
	return where, args
}

// appendMatchPredicates appends the substring/range/domain-shape predicates
// (keyword, score floor, time window, TLD suffix, domain-tracking) to the given
// where/args slices and returns the extended slices. Splitting these out of
// buildWhereClause keeps each function within the cyclomatic-complexity budget.
func appendMatchPredicates(filter domain.QueryFilter, where []string, args []interface{}) ([]string, []interface{}) {
	if filter.Keyword != "" {
		where = append(where, "keywords LIKE ?")
		args = append(args, "%"+filter.Keyword+"%")
	}
	if filter.ScoreMin > 0 {
		where = append(where, "score >= ?")
		args = append(args, filter.ScoreMin)
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
	if filter.Issuer != "" {
		// Match either the issuer org or the issuer CN, case-insensitive.
		where = append(where, "(issuer LIKE ? OR issuer_cn LIKE ?)")
		pat := "%" + filter.Issuer + "%"
		args = append(args, pat, pat)
	}
	if filter.Provider != "" {
		where = append(where, "hosting_provider LIKE ?")
		args = append(args, "%"+filter.Provider+"%")
	}
	if filter.Brand != "" {
		// A brand matches as an exact ("name"), typosquat ("~name"), or
		// homoglyph ("*name") entry in the keywords JSON array. Each variant is
		// matched as a quoted JSON element so it cannot partially match a longer
		// keyword. The patterns are bound as parameters, never interpolated.
		where = append(where, `(keywords LIKE ? OR keywords LIKE ? OR keywords LIKE ?)`)
		args = append(args,
			`%"`+filter.Brand+`"%`,
			`%"~`+filter.Brand+`"%`,
			`%"*`+filter.Brand+`"%`,
		)
	}
	if pred, target := domainTrackPredicate(filter.Domain); pred != "" {
		where = append(where, pred)
		args = append(args, target, "%."+target)
	}
	where, args = appendSharedIPPredicate(filter, where, args)
	return where, args
}

// appendSharedIPPredicate appends a json_each EXISTS predicate that matches rows
// whose resolved_ips JSON array contains the exact IP. json_each expands the
// array into one row per element; the predicate is true when any element equals
// the bound value. Exact comparison (not LIKE) means a partial address cannot
// match a longer one. The value is bound as a parameter, never interpolated.
func appendSharedIPPredicate(filter domain.QueryFilter, where []string, args []interface{}) ([]string, []interface{}) {
	if filter.SharedIP == "" {
		return where, args
	}
	where = append(where, "EXISTS (SELECT 1 FROM json_each(hits.resolved_ips) WHERE json_each.value = ?)")
	args = append(args, filter.SharedIP)
	return where, args
}

// appendSignalPredicates appends one predicate per requested signal key (AND
// semantics): a row matches only when its signals JSON array contains every
// listed key. Each key is matched as a quoted JSON element ("key") so it cannot
// partially match another key. Values are bound as parameters.
func appendSignalPredicates(filter domain.QueryFilter, where []string, args []interface{}) ([]string, []interface{}) {
	for _, sig := range filter.Signals {
		if sig == "" {
			continue
		}
		where = append(where, "signals LIKE ?")
		args = append(args, `%"`+sig+`"%`)
	}
	return where, args
}

// domainTrackPredicate builds the apex-plus-subdomain SQL predicate for a
// domain-tracking filter, mirroring domainutil.MatchesTrackTarget: a row matches
// when its domain equals the normalized target OR is a subdomain of it. It
// returns an empty predicate when the target normalizes to "" (no filter).
//
// The "." in the LIKE pattern is literal; domain names contain no LIKE wildcards
// (% or _), and both operands are bound as parameters by the caller — never
// concatenated into the SQL string.
func domainTrackPredicate(domainFilter string) (predicate, target string) {
	target = domainutil.NormalizeTrackTarget(domainFilter)
	if target == "" {
		return "", ""
	}
	return "(LOWER(domain) = ? OR LOWER(domain) LIKE ?)", target
}

// bookmarkedPredicate maps the tri-state bookmark filter to a SQL predicate.
// A nil filter yields no predicate; true matches bookmarked rows, false matches
// non-bookmarked rows.
func bookmarkedPredicate(bookmarked *bool) string {
	if bookmarked == nil {
		return ""
	}
	if *bookmarked {
		return "bookmarked = 1"
	}
	return "bookmarked = 0"
}

// severityRankExpr ranks the severity TEXT column by threat level rather than
// lexically. HIGH > MED > LOW, so a DESC sort surfaces the highest-threat hits
// first. The expression is a constant — no user data is interpolated — so it
// preserves the injection-safe posture of orderClause.
const severityRankExpr = "CASE severity WHEN 'HIGH' THEN 3 WHEN 'MED' THEN 2 WHEN 'LOW' THEN 1 ELSE 0 END"

// orderClause builds the ORDER BY clause from the filter's sort settings.
func orderClause(filter domain.QueryFilter) string {
	sortBy := "created_at"
	if filter.SortBy != "" {
		sortBy = sanitizeSortColumn(filter.SortBy)
	}
	sortDir := "DESC"
	if strings.EqualFold(filter.SortDir, "ASC") {
		sortDir = "ASC"
	}
	// Severity is a TEXT column whose values (HIGH/MED/LOW) have no natural
	// lexical threat order; rank them numerically instead.
	sortExpr := sortBy
	if sortBy == "severity" {
		sortExpr = severityRankExpr
	}
	// SECURITY: sortBy is sanitized through sanitizeSortColumn() allowlist and
	// the severity case maps to a constant expression; sortDir is limited to
	// "ASC"/"DESC" by the check above. Both are safe for direct interpolation.
	// ORDER BY does not support parameterized placeholders.
	return fmt.Sprintf(" ORDER BY %s %s", sortExpr, sortDir)
}

// QueryHits builds and executes a dynamic SQL query from the filter fields.
// All filter criteria use parameterized queries to prevent SQL injection.
func (d *DB) QueryHits(ctx context.Context, filter domain.QueryFilter) ([]domain.Hit, error) {
	where, args := buildWhereClause(filter)

	query := "SELECT domain, score, severity, keywords, signals, category, issuer, issuer_cn, san_domains, cert_not_before, ct_log, profile, session, created_at, updated_at, is_live, resolved_ips, hosting_provider, http_status, live_checked_at, bookmarked, base_domain FROM hits"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	// orderClause interpolates only an allowlisted sort column and a fixed
	// ASC/DESC direction; no user data reaches the query string here.
	query += orderClause(filter) //nolint:gosec // ORDER BY built from allowlisted column + fixed direction

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
// SQLite returns timestamps as strings and booleans as integers,
// so we scan and convert them manually.
func scanHit(rows interface {
	Scan(dest ...interface{}) error
},
) (domain.Hit, error) {
	var hit domain.Hit
	var severity string
	var keywordsJSON string
	var signalsJSON string
	var sanDomainsJSON string
	var certNotBeforeStr string
	var createdAtStr string
	var updatedAtStr string
	var isLive int
	var resolvedIPsJSON string
	var liveCheckedAtStr *string
	var bookmarked int

	err := rows.Scan(
		&hit.Domain,
		&hit.Score,
		&severity,
		&keywordsJSON,
		&signalsJSON,
		&hit.Category,
		&hit.Issuer,
		&hit.IssuerCN,
		&sanDomainsJSON,
		&certNotBeforeStr,
		&hit.CTLog,
		&hit.Profile,
		&hit.Session,
		&createdAtStr,
		&updatedAtStr,
		&isLive,
		&resolvedIPsJSON,
		&hit.HostingProvider,
		&hit.HTTPStatus,
		&liveCheckedAtStr,
		&bookmarked,
		&hit.BaseDomain,
	)
	if err != nil {
		return domain.Hit{}, fmt.Errorf("scanning hit row: %w", err)
	}

	hit.Severity = domain.Severity(severity)
	hit.CertNotBefore = parseTimestamp(certNotBeforeStr)
	hit.CreatedAt = parseTimestamp(createdAtStr)
	hit.UpdatedAt = parseTimestamp(updatedAtStr)
	hit.IsLive = isLive != 0
	hit.Bookmarked = bookmarked != 0

	if liveCheckedAtStr != nil {
		hit.LiveCheckedAt = parseTimestamp(*liveCheckedAtStr)
	}

	if err := json.Unmarshal([]byte(keywordsJSON), &hit.Keywords); err != nil {
		return domain.Hit{}, fmt.Errorf("unmarshaling keywords: %w", err)
	}
	if signalsJSON != "" {
		if err := json.Unmarshal([]byte(signalsJSON), &hit.Signals); err != nil {
			return domain.Hit{}, fmt.Errorf("unmarshaling signals: %w", err)
		}
	}
	if err := json.Unmarshal([]byte(sanDomainsJSON), &hit.SANDomains); err != nil {
		return domain.Hit{}, fmt.Errorf("unmarshaling SAN domains: %w", err)
	}
	if resolvedIPsJSON != "" {
		if err := json.Unmarshal([]byte(resolvedIPsJSON), &hit.ResolvedIPs); err != nil {
			return domain.Hit{}, fmt.Errorf("unmarshaling resolved IPs: %w", err)
		}
	}

	return hit, nil
}

// sanitizeSortColumn maps user-provided sort column names to safe SQL column
// names. Returns "created_at" for unrecognized inputs to prevent injection.
func sanitizeSortColumn(col string) string {
	allowed := map[string]string{
		"domain":          "domain",
		"score":           "score",
		"severity":        "severity",
		"session":         "session",
		"created_at":      "created_at",
		"updated_at":      "updated_at",
		"ct_log":          "ct_log",
		"profile":         "profile",
		"category":        "category",
		"is_live":         "is_live",
		"bookmarked":      "bookmarked",
		"http_status":     "http_status",
		"live_checked_at": "live_checked_at",
		"base_domain":     "base_domain",
		"resolved_ips":    "resolved_ips",
	}
	if safe, ok := allowed[strings.ToLower(col)]; ok {
		return safe
	}
	return "created_at"
}

// SetBookmark sets or clears the bookmark flag on a hit identified by domain.
func (d *DB) SetBookmark(ctx context.Context, domain string, bookmarked bool) error {
	val := 0
	if bookmarked {
		val = 1
	}
	_, err := d.db.ExecContext(ctx, "UPDATE hits SET bookmarked = ? WHERE domain = ?", val, domain)
	if err != nil {
		return fmt.Errorf("setting bookmark for %s: %w", domain, err)
	}
	return nil
}

// DeleteHit removes a single hit identified by domain.
func (d *DB) DeleteHit(ctx context.Context, domain string) error {
	_, err := d.db.ExecContext(ctx, "DELETE FROM hits WHERE domain = ?", domain)
	if err != nil {
		return fmt.Errorf("deleting hit for %s: %w", domain, err)
	}
	return nil
}

// DeleteHits removes multiple hits identified by their domains.
// Uses a transaction with batched parameter binding for atomicity.
func (d *DB) DeleteHits(ctx context.Context, domains []string) error {
	if len(domains) == 0 {
		return nil
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning delete transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	placeholders := make([]string, len(domains))
	args := make([]interface{}, len(domains))
	for i, dom := range domains {
		placeholders[i] = "?"
		args[i] = dom
	}

	// Only "?" placeholders are concatenated; the domain values are bound
	// via args and never interpolated into the query string.
	query := "DELETE FROM hits WHERE domain IN (" + strings.Join(placeholders, ",") + ")" //nolint:gosec // placeholders are literal "?", values are parameterized
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("deleting %d hits: %w", len(domains), err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing delete transaction: %w", err)
	}
	return nil
}

// CountByBaseDomain returns the number of hits sharing the given base domain.
func (d *DB) CountByBaseDomain(ctx context.Context, baseDomain string) (int, error) {
	var count int
	err := d.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM hits WHERE base_domain = ?", baseDomain,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting hits for base domain %s: %w", baseDomain, err)
	}
	return count, nil
}

// QueryHitsByBaseDomain returns all hits whose base_domain matches the given value.
func (d *DB) QueryHitsByBaseDomain(ctx context.Context, baseDomain string) ([]domain.Hit, error) {
	return d.QueryHits(ctx, domain.QueryFilter{BaseDomain: baseDomain})
}

// UpdateEnrichment updates the enrichment fields on a hit identified by domain.
// Serializes resolvedIPs as a JSON array.
func (d *DB) UpdateEnrichment(ctx context.Context, domain string, isLive bool, resolvedIPs []string, hostingProvider string, httpStatus int) error {
	ipsJSON, err := json.Marshal(resolvedIPs)
	if err != nil {
		return fmt.Errorf("marshaling resolved IPs: %w", err)
	}

	isLiveInt := 0
	if isLive {
		isLiveInt = 1
	}

	now := time.Now().UTC().Format(timestampFormat)

	const query = `UPDATE hits SET is_live = ?, resolved_ips = ?, hosting_provider = ?, http_status = ?, live_checked_at = ? WHERE domain = ?`
	_, err = d.db.ExecContext(ctx, query, isLiveInt, string(ipsJSON), hostingProvider, httpStatus, now, domain)
	if err != nil {
		return fmt.Errorf("updating enrichment for %s: %w", domain, err)
	}
	return nil
}
