package domain

import "time"

// Severity represents the threat level of a scored domain.
// Thresholds: HIGH >= 8, MED 5-7, LOW 1-4.
type Severity string

const (
	SeverityHigh Severity = "HIGH"

	SeverityMed Severity = "MED"

	SeverityLow Severity = "LOW"
)

// Hit is a scored domain persisted to storage. Every field except Session comes
// from the certificate and scoring engine; Session comes from the --session flag.
type Hit struct {
	Domain string

	Score int

	Severity Severity

	Keywords []string

	// Which heuristics fired (e.g. "typosquat"), unlike Keywords' matched terms;
	// makes hits filterable by detection mechanism.
	Signals []string

	// Strongest-match bucket: "crypto", "phishing", "ai", "hosted-abuse",
	// "tracker", or empty.
	Category string

	Issuer string

	IssuerCN string

	SANDomains []string

	CertNotBefore time.Time

	CTLog string

	Profile string

	// Empty means the default (untagged) session.
	Session string

	CreatedAt time.Time

	UpdatedAt time.Time

	// Enrichment-populated; false by default.
	IsLive bool

	// Enrichment-populated; nil by default.
	ResolvedIPs []string

	// Enrichment-populated; empty by default.
	HostingProvider string

	// Zero when no probe has been performed.
	HTTPStatus int

	// Zero when no probe has been performed.
	LiveCheckedAt time.Time

	Bookmarked bool

	// Registrable base domain for grouping subdomains; empty for legacy rows
	// until the V3 migration backfill runs.
	BaseDomain string
}

// CTLogEntry is a raw CT log entry before scoring: output of the CT log HTTP
// client, input to the parser.
type CTLogEntry struct {
	// Base64-decoded MerkleTreeLeaf bytes from the CT log API.
	LeafInput []byte

	// Base64-decoded extra_data bytes from the CT log API.
	ExtraData []byte

	Index int64

	LogURL string
}

// ScoredDomain is the scoring engine's output; the poller builds the full Hit
// from it plus certificate metadata.
type ScoredDomain struct {
	Domain string

	Score int

	Severity Severity

	MatchedKeywords []string

	// Which heuristics fired; see the scoring package for the full key set.
	Signals []string

	// Strongest-match bucket: "crypto", "phishing", "ai", "hosted-abuse", or empty.
	Category string
}
