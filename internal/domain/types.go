package domain

import "time"

// Severity represents the threat level of a scored domain.
// The three levels map to score thresholds: HIGH >= 6, MED 4-5, LOW 1-3.
type Severity string

const (
	// SeverityHigh indicates a score of 6 or above — near-certain malicious intent.
	// Typically a multi-keyword hit on a suspicious TLD.
	SeverityHigh Severity = "HIGH"

	// SeverityMed indicates a score of 4 or 5 — suspicious, worth investigating.
	SeverityMed Severity = "MED"

	// SeverityLow indicates a score of 1 to 3 — single keyword match, noise-prone.
	SeverityLow Severity = "LOW"
)

// Hit represents a scored domain that matched keyword heuristics, persisted to storage.
// Every field except Session is populated from the certificate and scoring engine.
// Session is set from the --session CLI flag and used to group monitoring runs.
type Hit struct {
	// Domain is the matched domain name extracted from the certificate CN or SAN.
	Domain string

	// Score is the total numeric score assigned by the scoring engine.
	Score int

	// Severity is the threat level classification derived from Score.
	Severity Severity

	// Keywords contains the list of profile keywords found in Domain.
	Keywords []string

	// Issuer is the certificate issuer organization name.
	Issuer string

	// IssuerCN is the certificate issuer Common Name.
	IssuerCN string

	// SANDomains contains all Subject Alternative Name DNS entries from the certificate.
	// This includes the domain itself plus any other domains sharing the certificate.
	SANDomains []string

	// CertNotBefore is the certificate validity start timestamp.
	CertNotBefore time.Time

	// CTLog is the name of the CT log from which this entry was fetched.
	CTLog string

	// Profile is the name of the keyword profile active when this hit was scored.
	Profile string

	// Session is an optional user-defined tag for grouping monitoring runs.
	// Empty string means the default (untagged) session.
	Session string

	// CreatedAt is when this hit was first stored in the database.
	CreatedAt time.Time

	// UpdatedAt is when this hit was last updated (e.g., after a duplicate domain appears in a new cert).
	UpdatedAt time.Time

	// IsLive indicates the domain responded to an HTTP probe (HEAD request).
	// Populated by the enrichment pipeline; false by default.
	IsLive bool

	// ResolvedIPs contains DNS A/AAAA records for the domain.
	// Populated by the enrichment pipeline; nil by default.
	ResolvedIPs []string

	// HostingProvider is the detected CDN or hosting provider from reverse DNS or IP range matching.
	// Populated by the enrichment pipeline; empty string by default.
	HostingProvider string

	// HTTPStatus is the HTTP status code returned by the liveness probe.
	// Zero when no probe has been performed.
	HTTPStatus int

	// LiveCheckedAt is when the enrichment liveness probe was last run.
	// Zero value when no probe has been performed.
	LiveCheckedAt time.Time

	// Bookmarked indicates the user has flagged this hit as interesting.
	// False by default.
	Bookmarked bool
}

// CTLogEntry represents a raw entry from a Certificate Transparency log before scoring.
// It is the output of the CT log HTTP client and the input to the parser.
type CTLogEntry struct {
	// LeafInput is the raw base64-decoded MerkleTreeLeaf bytes from the CT log API.
	LeafInput []byte

	// ExtraData is the raw base64-decoded extra_data bytes from the CT log API.
	ExtraData []byte

	// Index is the zero-based position of this entry in the CT log tree.
	Index int64

	// LogURL is the base URL of the CT log that produced this entry.
	LogURL string
}

// ScoredDomain is the output of the scoring engine and input to storage.
// It carries only the fields needed to persist a hit — the full Hit is
// constructed in the poller with additional certificate metadata.
type ScoredDomain struct {
	// Domain is the domain name that was scored.
	Domain string

	// Score is the total numeric score from all heuristics.
	Score int

	// Severity is the threat level classification.
	Severity Severity

	// MatchedKeywords is the list of profile keywords found in Domain.
	MatchedKeywords []string
}
