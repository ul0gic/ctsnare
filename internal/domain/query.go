package domain

import "time"

// QueryFilter defines the filtering criteria for querying hits from storage.
type QueryFilter struct {
	Keyword  string
	ScoreMin int
	Severity string
	Since    time.Duration
	TLD      string
	Session  string
	Limit    int
	Offset   int
	SortBy   string
	SortDir  string
}

// DBStats contains aggregate statistics about stored hits.
type DBStats struct {
	TotalHits   int
	BySeverity  map[Severity]int
	TopKeywords []KeywordCount
	FirstHit    time.Time
	LastHit     time.Time
}

// KeywordCount tracks how many times a keyword has been matched.
type KeywordCount struct {
	Keyword string
	Count   int
}
