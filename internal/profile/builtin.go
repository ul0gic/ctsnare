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
	// NOTE: free-tenant PaaS suffixes (pages.dev, workers.dev, netlify.app,
	// vercel.app, firebaseapp.com, github.io) are deliberately NOT skipped here.
	// They are governed by WatchPlatformSuffixes instead, so brand-on-platform
	// phishing (paypal-login.pages.dev) is still caught via tenant-only scoring.
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

// WatchPlatformSuffixes are free-hosting / PaaS suffixes that are NOT skipped
// outright. Instead, only the tenant labels (the part left of the suffix) are
// scored: a brand/typosquat/homoglyph hit on the tenant marks the domain as
// hosted abuse and stores it, while anything else is discarded silently.
//
// Rationale: legitimate brands never host on free PaaS, so paypal-login.pages.dev
// is near-certain phishing — but pages.dev is also home to millions of benign
// tenants, so a blanket skip blinds us to real abuse and a blanket score floods
// the feed. Tenant-only scoring threads that needle. These suffixes are removed
// from the plain skip list so the watch path, not the skip path, governs them.
//
// Like the skip list this is config-overridable; see config.WatchPlatforms.
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
	"appdomain.cloud",
	"therapymatch.info",
}

// CryptoProfile targets cryptocurrency scams, underground casinos, and financial fraud.
//
// BrandKeywords (+3) are exact wallet/exchange/chain brand names — a hit here is
// near-certain impersonation. Keywords (+1) are broad crypto/DeFi/gambling terms
// that are common in legitimate fintech and gain signal only in combination.
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
//
// BrandKeywords (+3) are exact brand and bank names. Keywords (+1) are action
// words (login, verify) that occur on countless legitimate sites and are only
// meaningful alongside a brand hit, a suspicious TLD, or other heuristics.
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

// AIProfile targets impersonation of AI/LLM products and vendors — a fast-growing
// lure for fake "free credits", account-verification, and token-grant scams.
//
// BrandKeywords (+3) are exact product/vendor names. Keywords (+1) are generic
// AI terms that appear on many legitimate sites.
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
