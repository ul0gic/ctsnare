package domain

import "time"

// QueryFilter defines the filtering criteria for querying hits from storage.
// All fields are optional — zero values mean "no filter on this field".
// Multiple fields are combined with AND logic.
type QueryFilter struct {
	// Keyword filters hits where the keywords JSON column contains this substring.
	// Case-insensitive (ASCII) substring match via SQL LIKE against the stored
	// keywords JSON array. Because the match runs against the JSON-serialized
	// text, the value is not escaped — JSON punctuation and LIKE wildcards
	// (% and _) in the keyword are treated literally as part of the pattern.
	Keyword string

	// ScoreMin filters hits with a score at or above this value.
	// Zero means no minimum score filter.
	ScoreMin int

	// Severity filters hits matching this severity level: "HIGH", "MED", or "LOW".
	// Empty string means no severity filter.
	Severity string

	// Since filters hits created within this duration before now.
	// For example, 24*time.Hour shows only hits from the last 24 hours.
	// Zero means no time filter.
	Since time.Duration

	// TLD filters hits where the domain ends with this suffix.
	// A leading dot is optional — both ".xyz" and "xyz" are accepted.
	TLD string

	// Session filters hits tagged with this session name.
	// Empty string means no session filter.
	Session string

	// Limit caps the number of results returned. Zero means no limit.
	Limit int

	// Offset skips this many results before returning, for pagination.
	Offset int

	// SortBy is the column to sort by. Accepted values: "domain", "score",
	// "severity", "session", "created_at", "updated_at", "ct_log", "profile".
	// Unrecognized values fall back to "created_at".
	SortBy string

	// SortDir is the sort direction: "ASC" or "DESC" (case-insensitive).
	// Any other value defaults to "DESC".
	SortDir string

	// Bookmarked is a tri-state bookmark filter. Nil (default) means no bookmark
	// filter; a non-nil true means only bookmarked hits; a non-nil false means
	// only non-bookmarked hits.
	Bookmarked *bool

	// LiveOnly filters to only live domains (those that responded to HTTP probe) when true.
	// False (default) means no liveness filter.
	LiveOnly bool

	// BaseDomain filters hits whose base_domain column matches this exact value.
	// Used for subdomain drill-down from the detail view.
	// Empty string means no base domain filter.
	BaseDomain string

	// Signals filters hits whose signals JSON array contains every listed
	// signal key (AND semantics — repeatable on the CLI). Each key is matched
	// as a quoted JSON element ("key") against the stored array text, so a key
	// cannot spuriously match a substring of another key. Empty means no filter.
	Signals []string

	// Category filters hits whose category column equals this value exactly
	// (e.g. "phishing", "hosted-abuse"). Empty means no category filter.
	Category string

	// Issuer filters hits whose issuer or issuer_cn contains this substring,
	// case-insensitive. Empty means no issuer filter.
	Issuer string

	// Provider filters hits whose hosting_provider column contains this
	// substring, case-insensitive. Empty means no provider filter.
	Provider string

	// Brand filters hits whose keywords array contains the given brand name in
	// any of its match forms: exact ("name"), typosquat ("~name"), or homoglyph
	// ("*name"). Empty means no brand filter.
	Brand string

	// Domain filters hits to a tracked target using apex-plus-subdomain
	// semantics: the row matches when its domain equals Domain OR is a subdomain
	// of Domain (ends with "." + Domain). The value is normalized (lowercased,
	// leading "*."/"." and trailing "." stripped) before matching. This mirrors
	// domainutil.MatchesTrackTarget; a parity test enforces agreement between the
	// Go matcher and the SQL predicate. Empty string means no domain filter.
	Domain string
}

// DBStats contains aggregate statistics about stored hits.
// Returned by Store.Stats.
type DBStats struct {
	// TotalHits is the total number of hits in the database.
	TotalHits int

	// BySeverity maps each severity level to its hit count.
	BySeverity map[Severity]int

	// TopKeywords lists the most frequently matched keywords, sorted by count descending.
	TopKeywords []KeywordCount

	// FirstHit is the timestamp of the earliest stored hit.
	// Zero if no hits are stored.
	FirstHit time.Time

	// LastHit is the timestamp of the most recently stored hit.
	// Zero if no hits are stored.
	LastHit time.Time
}

// KeywordCount tracks how many times a keyword has been matched across stored hits.
type KeywordCount struct {
	// Keyword is the matched keyword string.
	Keyword string

	// Count is the number of hits where this keyword was matched.
	Count int
}
