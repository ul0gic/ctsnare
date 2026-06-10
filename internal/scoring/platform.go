package scoring

import (
	"strings"

	"github.com/ul0gic/ctsnare/internal/domain"
)

// platformBonusPoints is the bonus added when a brand/typosquat/homoglyph
// signal fires on a tenant hosted under a watched free-hosting platform.
// Legitimate brands never host on free PaaS, so brand-on-platform is
// near-certain phishing — but only the tenant labels are scored, since the
// platform suffix itself is shared by countless benign tenants.
const platformBonusPoints = 2

// matchWatchPlatform returns the longest watched-platform suffix that lower
// ends with (matched on a label boundary), or "" if none. Longest-match
// avoids a shorter suffix shadowing a more specific one.
func (e *Engine) matchWatchPlatform(lower string) string {
	best := ""
	for _, p := range e.watchPlatforms {
		suffix := strings.ToLower(p)
		if lower == suffix || strings.HasSuffix(lower, "."+suffix) {
			if len(suffix) > len(best) {
				best = suffix
			}
		}
	}
	return best
}

// scorePlatformTenant scores a domain hosted under a watched free-hosting
// platform. Only the tenant labels (everything to the left of the platform
// suffix) are scored. The domain is stored only when a brand-impersonation
// signal (brand keyword, typosquat, or homoglyph) fires on the tenant; in that
// case it earns a +platformBonusPoints "hosted-abuse" bonus and the
// hosted-abuse category. When nothing fires the domain is discarded like a
// skip (score 0), so benign tenants never spam the feed. The platform apex
// itself (no tenant labels) is always discarded.
func (e *Engine) scorePlatformTenant(domainName, lower, platform string, profile *domain.Profile, cert domain.CertMeta) domain.ScoredDomain {
	tenant := strings.TrimSuffix(lower, "."+platform)
	// The apex itself (lower == platform) has no tenant labels — never store.
	if tenant == lower || tenant == "" {
		return domain.ScoredDomain{Domain: domainName}
	}

	// Score only the tenant portion. We reuse the full core heuristic suite
	// against the tenant string so typosquat/homoglyph/keyword detection all
	// apply, but structural TLD heuristics see the tenant, not the platform TLD.
	acc := e.scoreCore(tenant, profile, cert)

	brandSignal := contains(acc.signals, SignalBrandKeyword) ||
		contains(acc.signals, SignalTyposquat) ||
		contains(acc.signals, SignalHomoglyph)
	if !brandSignal {
		// Nothing impersonation-shaped fired: discard like a skip.
		return domain.ScoredDomain{Domain: domainName}
	}

	acc.addSignal(platformBonusPoints, SignalHostedAbuse)
	acc.category = CategoryHostedAbuse

	result := acc.result(domainName)
	// Report the full domain, not the tenant slice.
	result.Domain = domainName
	return result
}
