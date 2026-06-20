package scoring

import (
	"strings"

	"github.com/ul0gic/ctsnare/internal/domain"
)

const (
	largeSANCount     = 20
	shortValidityDays = 90
	certSANPoints     = 1
	certShortLived    = 1
	certFreeCAPoints  = 1
	// Short-lived and free-CA both describe free-CA abuse; cap the pair to avoid double-counting.
	certIssuerCap = 2
)

var freeCASubstrings = []string{
	"let's encrypt",
	"lets encrypt",
	"letsencrypt",
	"zerossl",
	"google trust services",
}

// scoreCertSignals adds SAN-count, short-lived-brand, and free-CA-brand signals.
// The latter two are capped jointly at certIssuerCap; a zero CertMeta contributes nothing.
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

	// Points go to short-lived first, then free-CA, but both signals are recorded if they fired.
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
