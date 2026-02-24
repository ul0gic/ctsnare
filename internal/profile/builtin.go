package profile

import "github.com/ul0gic/ctsnare/internal/domain"

// commonSkipSuffixes are infrastructure domain suffixes that generate
// noise and should be skipped during scoring regardless of profile.
var commonSkipSuffixes = []string{
	"cloudflaressl.com",
	"amazonaws.com",
	"herokuapp.com",
	"azurewebsites.net",
	"googleusercontent.com",
	"fastly.net",
	"akamaiedge.net",
	"cloudfront.net",
	"github.io",
	"gitlab.io",
	"netlify.app",
	"vercel.app",
	"firebaseapp.com",
	"appspot.com",
	"trafficmanager.net",
	"azure-api.net",
}

// CryptoProfile targets cryptocurrency and financial scam domains.
var CryptoProfile = domain.Profile{
	Name: "crypto",
	Keywords: []string{
		"casino", "swap", "exchange", "airdrop", "token",
		"wallet", "invest", "mining", "defi", "stake",
		"yield", "claim", "reward", "bonus", "crypto",
		"bitcoin", "ethereum", "binance", "coinbase", "metamask",
	},
	SuspiciousTLDs: []string{
		".xyz", ".top", ".vip", ".win", ".bet",
		".casino", ".click", ".buzz", ".icu", ".monster",
	},
	SkipSuffixes: commonSkipSuffixes,
	Description:  "Cryptocurrency, casino, and financial scam domains",
}

// PhishingProfile targets credential phishing and brand impersonation domains.
var PhishingProfile = domain.Profile{
	Name: "phishing",
	Keywords: []string{
		"login", "signin", "verify", "secure", "account",
		"update", "confirm", "banking", "paypal", "microsoft",
		"apple", "google", "amazon", "netflix", "support",
		"helpdesk", "password", "credential",
	},
	SuspiciousTLDs: []string{
		".xyz", ".top", ".info", ".click", ".buzz",
		".icu", ".monster", ".tk", ".ml", ".ga",
	},
	SkipSuffixes: commonSkipSuffixes,
	Description:  "Credential phishing and brand impersonation domains",
}

// AllProfile combines keywords and TLDs from all built-in profiles.
var AllProfile = buildAllProfile()

// buildAllProfile merges all built-in profiles into a single combined profile.
func buildAllProfile() domain.Profile {
	keywords := mergeUnique(CryptoProfile.Keywords, PhishingProfile.Keywords)
	tlds := mergeUnique(CryptoProfile.SuspiciousTLDs, PhishingProfile.SuspiciousTLDs)

	return domain.Profile{
		Name:           "all",
		Keywords:       keywords,
		SuspiciousTLDs: tlds,
		SkipSuffixes:   commonSkipSuffixes,
		Description:    "Combined profile — all keywords and TLDs from crypto + phishing",
	}
}

// mergeUnique combines two string slices, deduplicating entries.
func mergeUnique(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	result := make([]string, 0, len(a)+len(b))

	for _, s := range a {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			result = append(result, s)
		}
	}
	for _, s := range b {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			result = append(result, s)
		}
	}

	return result
}
