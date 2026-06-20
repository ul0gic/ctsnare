package domain

import "context"

// CertMeta carries certificate metadata for cert-level heuristics.
// The zero value is valid and disables the cert-based heuristics.
type CertMeta struct {
	SANCount int

	// Zero means unknown (heuristic skipped).
	ValidityDays int

	// Issuer CN and organization joined for free-CA substring matching; empty disables it.
	Issuer string
}

// Scorer scores a domain against a profile's keyword heuristics.
// Score == 0 means no match and the domain should be discarded.
type Scorer interface {
	Score(domain string, profile *Profile) ScoredDomain

	// A zero CertMeta yields the same result as Score.
	ScoreWithCert(domain string, profile *Profile, cert CertMeta) ScoredDomain
}

// Store persists hits. Implementations must be safe for concurrent use.
type Store interface {
	// Errors if the domain already exists; prefer UpsertHit for deduplication.
	InsertHit(ctx context.Context, hit Hit) error

	// An empty QueryFilter returns all hits.
	QueryHits(ctx context.Context, filter QueryFilter) ([]Hit, error)

	// Inserts or updates keyed on domain; the primary write path.
	UpsertHit(ctx context.Context, hit Hit) error

	Stats(ctx context.Context) (DBStats, error)

	ClearAll(ctx context.Context) error

	ClearSession(ctx context.Context, session string) error

	SetBookmark(ctx context.Context, domain string, bookmarked bool) error

	DeleteHit(ctx context.Context, domain string) error

	DeleteHits(ctx context.Context, domains []string) error

	UpdateEnrichment(ctx context.Context, domain string, isLive bool, resolvedIPs []string, hostingProvider string, httpStatus int) error

	CountByBaseDomain(ctx context.Context, baseDomain string) (int, error)

	QueryHitsByBaseDomain(ctx context.Context, baseDomain string) ([]Hit, error)

	// Aggregates each IP hosting two or more domains; CDN-edge providers are excluded.
	NetworkClusters(ctx context.Context) ([]NetworkCluster, error)

	Close() error
}

// ProfileLoader loads built-in and user-defined keyword profiles.
type ProfileLoader interface {
	// Built-in names are "crypto", "phishing", "ai", and "all".
	LoadProfile(name string) (*Profile, error)

	// Sorted, including built-in and custom profiles from config.
	ListProfiles() []string
}
