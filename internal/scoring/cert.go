package scoring

import (
	"strings"

	"github.com/ul0gic/ctsnare/internal/domain"
)

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
	certFreeCAPoints  = 1
	// certIssuerCap bounds the combined short-lived + free-CA issuer contribution.
	// Both signals describe the same free-CA abuse pattern, so we credit at most
	// +2 to avoid double-counting a single underlying behavior.
	certIssuerCap = 2
)

// freeCASubstrings are case-insensitive substrings that identify the major free
// certificate authorities. A free CA covering a brand impersonation is a weak
// corroborating signal — free certs are cheap and ubiquitous, so this only
// matters alongside a brand match and is capped together with the short-lived
// signal at certIssuerCap.
var freeCASubstrings = []string{
	"let's encrypt",
	"lets encrypt",
	"letsencrypt",
	"zerossl",
	"google trust services",
}

// scoreCertSignals applies certificate-level heuristics to the accumulator:
//   - +1 (san-count) when the cert carries a large SAN set (>= largeSANCount).
//   - +1 (short-lived-brand) when the cert is short-lived AND a brand matched.
//   - +1 (free-ca-brand) when a free CA issued the cert AND a brand matched.
//
// The short-lived and free-CA signals both describe free-CA abuse, so their
// combined contribution is capped at certIssuerCap (+2). A zero CertMeta
// contributes nothing.
func (e *Engine) scoreCertSignals(acc *accumulator, cert domain.CertMeta, brandHit bool) {
	if cert.SANCount >= largeSANCount {
		acc.addSignal(certSANPoints, SignalSANCount)
	}

	if !brandHit {
		return
	}

	issuerContribution := 0
	shortLived := cert.ValidityDays > 0 && cert.ValidityDays <= shortValidityDays
	freeCA := isFreeCA(cert.Issuer)

	if shortLived {
		issuerContribution += certShortLived
	}
	if freeCA {
		issuerContribution += certFreeCAPoints
	}
	if issuerContribution > certIssuerCap {
		issuerContribution = certIssuerCap
	}

	// Record the signal(s) without exceeding the capped point budget. Points are
	// attributed to short-lived first, then free-CA, but both signals are
	// recorded if they fired so the breakdown stays accurate.
	remaining := issuerContribution
	if shortLived {
		give := min2(certShortLived, remaining)
		acc.addSignal(give, SignalShortLivedBrand)
		remaining -= give
	}
	if freeCA {
		acc.addSignal(remaining, SignalFreeCABrand)
	}
}

// isFreeCA reports whether issuer contains a known free-CA marker.
func isFreeCA(issuer string) bool {
	if issuer == "" {
		return false
	}
	lower := strings.ToLower(issuer)
	for _, sub := range freeCASubstrings {
		if strings.Contains(lower, sub) {
			return true
		}
	}
	return false
}

func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}
