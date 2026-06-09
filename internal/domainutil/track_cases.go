package domainutil

// TrackMatchCase is one row in the shared domain-matcher case table. Target is
// already normalized (lowercased, no wildcard/leading/trailing dot).
type TrackMatchCase struct {
	Name   string
	Domain string
	Target string
	Want   bool
}

// TrackMatchCases is the authoritative case table for domain-tracking match
// semantics. It is shared (not test-local) so the storage-layer SQL parity test
// can reuse the exact same cases as the Go matcher test, guaranteeing the Go
// matcher (MatchesTrackTarget) and the SQL predicate
// (LOWER(domain) = ? OR LOWER(domain) LIKE ?) never drift.
var TrackMatchCases = []TrackMatchCase{
	{"apex exact", "openai.com", "openai.com", true},
	{"subdomain", "api.openai.com", "openai.com", true},
	{"deep subdomain", "a.b.c.openai.com", "openai.com", true},
	{"case-insensitive domain", "API.OpenAI.Com", "openai.com", true},
	{"sibling non-match", "notopenai.com", "openai.com", false},
	{"suffix trap non-match", "openai.com.evil.com", "openai.com", false},
	{"unrelated", "anthropic.com", "openai.com", false},
	{"prefix-of-target non-match", "openai.co", "openai.com", false},
	{"empty target", "openai.com", "", false},
	{"empty domain", "", "openai.com", false},
}
