package scoring

import (
	"strings"

	"github.com/ul0gic/ctsnare/internal/domain"
)

// Bonus when brand/typosquat/homoglyph fires on a tenant under a watched free-hosting platform.
const platformBonusPoints = 2

// matchWatchPlatform returns the longest watched-platform suffix lower ends with
// (on a label boundary), or "" — longest-match avoids a shorter suffix shadowing a specific one.
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

// scorePlatformTenant scores only the tenant labels under a watched platform,
// storing the domain (with a hosted-abuse bonus) only if a brand signal fires there.
func (e *Engine) scorePlatformTenant(domainName, lower, platform string, profile *domain.Profile, cert domain.CertMeta) domain.ScoredDomain {
	tenant := strings.TrimSuffix(lower, "."+platform)
	if tenant == lower || tenant == "" {
		return domain.ScoredDomain{Domain: domainName}
	}

	// Score the tenant string so structural TLD heuristics see the tenant, not the platform TLD.
	acc := e.scoreCore(tenant, profile, cert)

	brandSignal := contains(acc.signals, SignalBrandKeyword) ||
		contains(acc.signals, SignalTyposquat) ||
		contains(acc.signals, SignalHomoglyph)
	if !brandSignal {
		return domain.ScoredDomain{Domain: domainName}
	}

	acc.addSignal(platformBonusPoints, SignalHostedAbuse)
	acc.category = CategoryHostedAbuse

	result := acc.result(domainName)
	result.Domain = domainName
	return result
}
