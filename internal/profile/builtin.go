package profile

import "github.com/ul0gic/ctsnare/internal/domain"

// GlobalSkipSuffixes are infrastructure suffixes skipped during scoring on every
// profile; config-overridable via [skip_overrides] (ctsnare skip add/remove).
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
	// Free-tenant PaaS (pages.dev, github.io, ...) live in WatchPlatformSuffixes,
	// not here, so brand-on-platform phishing is still caught by tenant scoring.
	"herokuapp.com",
	"herokuspace.com",
	"appspot.com",
	"gitlab.io",
	"fly.dev",
	"render.com",
	"railway.app",
	"onrender.com",
	// Cloud test-cert churn observed live on the CT firehose.
	"aws.dev",
	"a2z.eu",
	"crm.dev",
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

// WatchPlatformSuffixes are free PaaS suffixes scored on the tenant label only:
// a brand hit there flags hosted abuse, everything else is dropped. Config-overridable.
var WatchPlatformSuffixes = []string{
	"pages.dev",
	"workers.dev",
	"r2.dev",
	"web.app",
	"firebaseapp.com",
	"netlify.app",
	"vercel.app",
	"glitch.me",
	"repl.co",
	"github.io",
	"weebly.com",
	"wixsite.com",
}

// DefaultUserAdditions are legitimate enterprise/SaaS suffixes seeded into a
// fresh config's [skip_overrides]; kept separate so users can remove them easily.
var DefaultUserAdditions = []string{
	"jpmchase.net",
	"sailpoint.com",
	"identitynow-demo.com",
	"appdomain.cloud",
	"therapymatch.info",
}

// CryptoProfile targets crypto scams, underground casinos, and financial fraud.
// BrandKeywords (+3) are exact impersonation targets; Keywords (+1) gain signal only in combination.
var CryptoProfile = domain.Profile{
	Name: "crypto",
	BrandKeywords: []string{
		// Wallets / exchanges / chains — high-precision impersonation targets
		"bitcoin", "ethereum", "binance", "coinbase", "metamask",
		"trustwallet", "ledger", "trezor", "opensea", "uniswap",
		"pancakeswap", "solana", "cardano",
		// Gambling brands
		"1xbet", "bet365", "betway",
	},
	Keywords: []string{
		// Crypto scam tactics
		"airdrop", "presale", "giveaway", "rugpull",
		"moonshot", "pump-and", "freemint",
		// DeFi / exchange terms
		"defi", "swap", "staking", "yield-farm", "liquidity",
		"flashloan", "smartcontract", "blockchain",
		// Casino / gambling terms
		"casino", "jackpot", "sportsbet",
		"slots", "poker", "roulette", "blackjack",
		"lottery", "gambling",
		// Generic financial terms
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
// BrandKeywords (+3) are exact brands; Keywords (+1) are action words meaningful only alongside another signal.
var PhishingProfile = domain.Profile{
	Name: "phishing",
	BrandKeywords: []string{
		// Brand impersonation — high-value targets
		"paypal", "netflix", "microsoft", "instagram", "facebook",
		"whatsapp", "telegram", "dropbox", "docusign", "linkedin",
		"snapchat", "tiktok", "twitter", "discord", "spotify",
		// Banking / financial brand targets
		"chase", "wellsfargo", "bankofamerica", "citibank", "hsbc",
		"barclays", "santander", "capitalone",
		// Shipping / delivery brands
		"dhl", "fedex", "usps", "royalmail",
	},
	Keywords: []string{
		// Action words — only the strong phishing signals
		"signin", "login", "verify", "password", "credential",
		"banking", "webscr", "authenticate", "suspended", "unauthorized",
		"security-alert", "helpdesk", "verification",
	},
	SuspiciousTLDs: []string{
		".xyz", ".top", ".click", ".buzz",
		".icu", ".monster", ".tk", ".ml", ".ga",
		".cf", ".quest", ".sbs", ".cfd", ".rest",
	},
	SkipSuffixes: GlobalSkipSuffixes,
	Description:  "Credential phishing and brand impersonation domains",
}

// AIProfile targets AI/LLM product and vendor impersonation (fake free-credit
// and token-grant scams). BrandKeywords (+3) are exact vendors; Keywords (+1) are generic.
var AIProfile = domain.Profile{
	Name: "ai",
	BrandKeywords: []string{
		// AI vendors / products — high-precision impersonation targets
		"openai", "chatgpt", "anthropic", "claude", "gemini",
		"copilot", "midjourney", "deepseek", "perplexity",
		"huggingface", "stability-ai", "grok-ai",
	},
	Keywords: []string{
		// Generic AI lure terms
		"airdrop", "free-credits", "gpt", "llm", "prompt",
	},
	SuspiciousTLDs: []string{
		".xyz", ".top", ".click", ".buzz", ".icu",
		".monster", ".quest", ".sbs", ".cfd", ".rest", ".app",
	},
	SkipSuffixes: GlobalSkipSuffixes,
	Description:  "AI/LLM product and vendor impersonation scams",
}

// AllProfile combines keywords and TLDs from all built-in profiles.
var AllProfile = buildAllProfile()

// buildAllProfile merges all built-in profiles into a single combined profile.
func buildAllProfile() domain.Profile {
	brand := mergeUnique(CryptoProfile.BrandKeywords, PhishingProfile.BrandKeywords)
	brand = mergeUnique(brand, AIProfile.BrandKeywords)
	keywords := mergeUnique(CryptoProfile.Keywords, PhishingProfile.Keywords)
	keywords = mergeUnique(keywords, AIProfile.Keywords)
	tlds := mergeUnique(CryptoProfile.SuspiciousTLDs, PhishingProfile.SuspiciousTLDs)
	tlds = mergeUnique(tlds, AIProfile.SuspiciousTLDs)

	return domain.Profile{
		Name:           "all",
		BrandKeywords:  brand,
		Keywords:       keywords,
		SuspiciousTLDs: tlds,
		SkipSuffixes:   GlobalSkipSuffixes,
		Description:    "Combined profile — all keywords and TLDs from crypto + phishing + ai",
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
