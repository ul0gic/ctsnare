package domain

import "time"

// QueryFilter defines the filtering criteria for querying hits from storage.
// All fields are optional — zero values mean "no filter on this field".
// Multiple fields are combined with AND logic.
type QueryFilter struct {
	// Keyword filters hits where the keywords JSON column contains this substring.
	// Case-sensitive substring match against the stored JSON.
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

	// Bookmarked filters to only bookmarked hits when true.
	// False (default) means no bookmark filter.
	Bookmarked bool

	// LiveOnly filters to only live domains (those that responded to HTTP probe) when true.
	// False (default) means no liveness filter.
	LiveOnly bool
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
