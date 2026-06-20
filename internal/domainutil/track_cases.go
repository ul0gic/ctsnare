package domainutil

// TrackMatchCase is one row in the shared matcher case table; Target is pre-normalized.
type TrackMatchCase struct {
	Name   string
	Domain string
	Target string
	Want   bool
}

// TrackMatchCases is shared (not test-local) so the storage SQL parity test and
// the Go matcher test reuse identical cases, guaranteeing the two never drift.
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
