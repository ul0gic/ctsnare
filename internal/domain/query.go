package domain

import "time"

// QueryFilter defines hit query criteria. Zero values mean no filter on that
// field; fields combine with AND.
type QueryFilter struct {
	// Matched unescaped against the keywords JSON text, so LIKE wildcards (% _)
	// and JSON punctuation in the value are treated literally.
	Keyword string

	ScoreMin int

	// "HIGH", "MED", or "LOW".
	Severity string

	// Hits created within this duration before now.
	Since time.Duration

	// Leading dot optional — both ".xyz" and "xyz" are accepted.
	TLD string

	Session string

	Limit int

	Offset int

	// "domain", "score", "severity", "session", "created_at", "updated_at",
	// "ct_log", "profile"; unrecognized values fall back to "created_at".
	SortBy string

	// "ASC" or "DESC" (case-insensitive); any other value defaults to "DESC".
	SortDir string

	// Tri-state: nil no filter, true only bookmarked, false only non-bookmarked.
	Bookmarked *bool

	LiveOnly bool

	// Exact match on base_domain; used for subdomain drill-down.
	BaseDomain string

	// AND semantics; each key matched as a quoted JSON element so it can't
	// substring-match another key.
	Signals []string

	// Exact match on category (e.g. "phishing", "hosted-abuse").
	Category string

	// Case-insensitive substring of issuer or issuer_cn.
	Issuer string

	// Case-insensitive substring of hosting_provider.
	Provider string

	// Matches brand in any form: exact ("name"), typosquat ("~name"), homoglyph ("*name").
	Brand string

	// Apex-plus-subdomain match (domain == Domain OR subdomain of Domain), value
	// normalized first. Mirrors domainutil.MatchesTrackTarget; a parity test guards drift.
	Domain string

	// Exact element match on resolved_ips via json_each, so a partial IP can't
	// match a longer address; pivots a cluster to its co-hosted domains.
	SharedIP string
}

// DBStats contains aggregate statistics about stored hits.
type DBStats struct {
	TotalHits int

	BySeverity map[Severity]int

	// Sorted by count descending.
	TopKeywords []KeywordCount

	// Zero if no hits are stored.
	FirstHit time.Time

	// Zero if no hits are stored.
	LastHit time.Time
}

// KeywordCount tracks how many times a keyword has been matched across stored hits.
type KeywordCount struct {
	Keyword string

	Count int
}
