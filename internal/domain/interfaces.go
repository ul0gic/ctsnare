package domain

import "context"

// Scorer scores a domain against a profile's keyword heuristics.
// Implementations apply all configured heuristics and return a ScoredDomain
// with the total score, severity classification, and matched keywords.
// A score of zero means no keywords matched and the domain should be discarded.
type Scorer interface {
	// Score runs all scoring heuristics against domainName using the given profile.
	// Returns a ScoredDomain with Score == 0 when the domain matches a skip suffix
	// or has no keyword matches.
	Score(domain string, profile *Profile) ScoredDomain
}

// Store provides persistence operations for hits.
// All methods accept a context for cancellation and timeout propagation.
// Implementations must be safe for concurrent use.
type Store interface {
	// InsertHit inserts a new hit record. Returns an error if a record with
	// the same domain already exists. Prefer UpsertHit for deduplication.
	InsertHit(ctx context.Context, hit Hit) error

	// QueryHits returns hits matching the given filter criteria. All filter
	// fields are optional — an empty QueryFilter returns all hits.
	QueryHits(ctx context.Context, filter QueryFilter) ([]Hit, error)

	// UpsertHit inserts or updates a hit keyed on domain. If a record for the
	// domain already exists, it is updated with the new score, keywords, and
	// certificate metadata. This is the primary write path.
	UpsertHit(ctx context.Context, hit Hit) error

	// Stats returns aggregate statistics about all stored hits including
	// total count, breakdown by severity, top keywords, and date range.
	Stats(ctx context.Context) (DBStats, error)

	// ClearAll removes all hit records from the database.
	ClearAll(ctx context.Context) error

	// ClearSession removes all hit records tagged with the given session name.
	ClearSession(ctx context.Context, session string) error

	// Close releases the underlying database connection. Must be called when
	// the store is no longer needed.
	Close() error
}

// ProfileLoader loads and lists keyword profiles.
// Implementations provide access to both built-in and user-defined profiles.
type ProfileLoader interface {
	// LoadProfile returns the named profile or an error if it does not exist.
	// Built-in profile names are "crypto", "phishing", and "all".
	LoadProfile(name string) (*Profile, error)

	// ListProfiles returns all available profile names in sorted order,
	// including both built-in and any custom profiles from config.
	ListProfiles() []string
}
