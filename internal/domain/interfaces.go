package domain

import "context"

// Scorer scores a domain against a profile's keyword heuristics.
type Scorer interface {
	Score(domain string, profile *Profile) ScoredDomain
}

// Store provides persistence operations for hits.
type Store interface {
	InsertHit(ctx context.Context, hit Hit) error
	QueryHits(ctx context.Context, filter QueryFilter) ([]Hit, error)
	UpsertHit(ctx context.Context, hit Hit) error
	Stats(ctx context.Context) (DBStats, error)
	ClearAll(ctx context.Context) error
	ClearSession(ctx context.Context, session string) error
	Close() error
}

// ProfileLoader loads and lists keyword profiles.
type ProfileLoader interface {
	LoadProfile(name string) (*Profile, error)
	ListProfiles() []string
}
