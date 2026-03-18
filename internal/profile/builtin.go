package profile

import "github.com/ul0gic/ctsnare/internal/domain"

// GlobalSkipSuffixes are infrastructure domain suffixes that universally
// generate noise and should be skipped during scoring regardless of profile.
// These are cloud providers, CDNs, PaaS platforms, and big tech infrastructure
// that will never be phishing targets worth scoring.
//
// This list is the "hardcoded base layer" of the skip suffix system.
// Users can add to or remove from this list via the [skip_overrides] section
// in their TOML config, managed by `ctsnare skip add/remove`.
var GlobalSkipSuffixes = []string{
	// Cloud providers
	"cloudflaressl.com",
	"amazonaws.com",
	"amazonaws.com.cn",
	"azurewebsites.net",
	"azurefd.net",
	"azure-api.net",
	"windows.net",
	"microsoftonline.com",
	"googleusercontent.com",
	"google.com",
	"googleapis.com",
	"gstatic.com",
	"1e100.net",
	// CDN / edge
	"fastly.net",
	"akamaiedge.net",
	"akamai.net",
	"edgekey.net",
	"cloudfront.net",
	"trafficmanager.net",
	// PaaS / hosting
	"herokuapp.com",
	"herokuspace.com",
	"netlify.app",
	"vercel.app",
	"firebaseapp.com",
	"appspot.com",
	"github.io",
	"gitlab.io",
	"pages.dev",
	"workers.dev",
	"fly.dev",
	"render.com",
	"railway.app",
	"onrender.com",
	// IP / dynamic DNS services
	"sslip.io",
	"nip.io",
	"xip.io",
	// Big tech infra
	"apple.com",
	"icloud.com",
	"amazon.com",
	"facebook.com",
	"meta.com",
	"instagram.com",
	"whatsapp.net",
	"linkedin.com",
	"microsoft.com",
	"office.com",
	"office365.com",
	"outlook.com",
	"live.com",
	"netflix.com",
	"paypal.com",
	"google.co",
}

// DefaultUserAdditions are enterprise/SaaS domain suffixes that match
// keywords but are legitimate infrastructure. They are separated from
// GlobalSkipSuffixes so users who specifically monitor enterprise
// infrastructure can easily remove them via `ctsnare skip remove`.
//
// These become the default entries in the [skip_overrides] additions
// array when a fresh config file is generated.
var DefaultUserAdditions = []string{
	"jpmchase.net",
	"sailpoint.com",
	"identitynow-demo.com",
	"aws.dev",
	"appdomain.cloud",
	"therapymatch.info",
}

// CryptoProfile targets cryptocurrency scams, underground casinos, and financial fraud.
var CryptoProfile = domain.Profile{
	Name: "crypto",
	Keywords: []string{
		// Crypto brand impersonation
		"bitcoin", "ethereum", "binance", "coinbase", "metamask",
		"trustwallet", "ledger", "trezor", "opensea", "uniswap",
		"pancakeswap", "solana", "cardano", "blockchain",
		// Crypto scam tactics
		"airdrop", "presale", "giveaway", "rugpull",
		"moonshot", "pump-and", "freemint",
		// DeFi / exchange scams
		"defi", "swap", "staking", "yield-farm", "liquidity",
		"flashloan", "smartcontract",
		// Casino / gambling
		"casino", "jackpot", "sportsbet", "1xbet", "bet365",
		"betway", "slots", "poker", "roulette", "blackjack",
		"lottery", "gambling",
		// Financial fraud
		"wallet", "token", "mining", "crypto", "nft",
	},
	SuspiciousTLDs: []string{
		".xyz", ".top", ".vip", ".win", ".bet",
		".casino", ".click", ".buzz", ".icu", ".monster",
		".quest", ".sbs", ".cfd", ".rest",
	},
	SkipSuffixes: GlobalSkipSuffixes,
	Description:  "Cryptocurrency scams, underground casinos, and financial fraud",
}

// PhishingProfile targets credential phishing and brand impersonation domains.
var PhishingProfile = domain.Profile{
	Name: "phishing",
	Keywords: []string{
		// Brand impersonation — high-value targets
		"paypal", "netflix", "microsoft", "instagram", "facebook",
		"whatsapp", "telegram", "dropbox", "docusign", "linkedin",
		"snapchat", "tiktok", "twitter", "discord", "spotify",
		// Banking / financial brand targets
		"chase", "wellsfargo", "bankofamerica", "citibank", "hsbc",
		"barclays", "santander", "capitalone",
		// Shipping / delivery phishing
		"dhl", "fedex", "usps", "ups-delivery", "royalmail",
		// Action words — only the strong phishing signals
		"signin", "login", "verify", "password", "credential",
		"banking", "webscr", "authenticate", "suspended", "unauthorized",
		"security-alert", "helpdesk", "verification",
	},
	SuspiciousTLDs: []string{
		".xyz", ".top", ".info", ".click", ".buzz",
		".icu", ".monster", ".tk", ".ml", ".ga",
		".cf", ".quest", ".sbs", ".cfd", ".rest",
	},
	SkipSuffixes: GlobalSkipSuffixes,
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
		SkipSuffixes:   GlobalSkipSuffixes,
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
