package domain

import "time"

// Severity represents the threat level of a scored domain.
type Severity string

const (
	SeverityHigh Severity = "HIGH"
	SeverityMed  Severity = "MED"
	SeverityLow  Severity = "LOW"
)

// Hit represents a scored domain that matched keyword heuristics, persisted to storage.
type Hit struct {
	Domain        string
	Score         int
	Severity      Severity
	Keywords      []string
	Issuer        string
	IssuerCN      string
	SANDomains    []string
	CertNotBefore time.Time
	CTLog         string
	Profile       string
	Session       string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CTLogEntry represents a raw entry from a Certificate Transparency log before scoring.
type CTLogEntry struct {
	LeafInput []byte
	ExtraData []byte
	Index     int64
	LogURL    string
}

// ScoredDomain is the output of the scoring engine and input to storage.
type ScoredDomain struct {
	Domain          string
	Score           int
	Severity        Severity
	MatchedKeywords []string
}
