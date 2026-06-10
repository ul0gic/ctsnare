package scoring

import "github.com/ul0gic/ctsnare/internal/domain"

// Certificate-level heuristic tuning.
const (
	// largeSANCount is the SAN-count threshold above which a cert earns +1. Bulk
	// SAN sets are common in disposable/wildcard phishing infrastructure.
	largeSANCount = 20
	// shortValidityDays is the validity-period threshold (in days) at or below
	// which a brand-bait cert earns +1. Free CAs issue short-lived certs, and a
	// short-lived cert covering a brand impersonation is a strong combined signal.
	shortValidityDays = 90
	certSANPoints     = 1
	certShortLived    = 1
)

// scoreCert applies certificate-level heuristics:
//   - +1 when the cert carries a large SAN set (>= largeSANCount).
//   - +1 when the cert is short-lived (<= shortValidityDays) AND a brand keyword
//     matched, since a short-lived cert on brand bait is a free-CA abuse pattern.
//
// A zero CertMeta (no certificate context) contributes nothing.
func scoreCert(cert domain.CertMeta, brandHit bool) int {
	score := 0
	if cert.SANCount >= largeSANCount {
		score += certSANPoints
	}
	if brandHit && cert.ValidityDays > 0 && cert.ValidityDays <= shortValidityDays {
		score += certShortLived
	}
	return score
}
