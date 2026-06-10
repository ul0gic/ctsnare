package scoring

// Signal keys are the stable, machine-readable identifiers for each scoring
// heuristic. They are persisted on every hit (the `signals` column) and exposed
// to the user via `query --signal <key>`, so they form a public contract:
// renaming one is a breaking change for stored data and saved queries. Add new
// keys here rather than coining ad-hoc strings at the call site.
const (
	// SignalBrandKeyword fires when a brand-tier keyword matched literally.
	SignalBrandKeyword = "brand-keyword"
	// SignalGenericKeyword fires when a generic-tier keyword matched literally.
	SignalGenericKeyword = "generic-keyword"
	// SignalTyposquat fires when a label is an edit-distance near-miss of a brand.
	SignalTyposquat = "typosquat"
	// SignalHomoglyph fires when a keyword matched only after confusable folding.
	SignalHomoglyph = "homoglyph"
	// SignalPunycode fires when any label is an xn-- A-label.
	SignalPunycode = "punycode"
	// SignalSuspiciousTLD fires on a cheap-tier TLD (+1).
	SignalSuspiciousTLD = "suspicious-tld"
	// SignalBurnerTLD fires on a burner-tier TLD (+6).
	SignalBurnerTLD = "burner-tld"
	// SignalNumericSLD fires when the SLD is all digits on a tiered TLD.
	SignalNumericSLD = "numeric-sld"
	// SignalDeceptivePrefix fires on com-/www-/-com deceptive label shapes.
	SignalDeceptivePrefix = "deceptive-prefix"
	// SignalEntropy fires on a high-entropy / DGA-shaped label.
	SignalEntropy = "entropy"
	// SignalLongDomain fires when the registered domain exceeds the length cap.
	SignalLongDomain = "long-domain"
	// SignalHyphens fires on hyphen-dense registered domains.
	SignalHyphens = "hyphens"
	// SignalDigitSeq fires on a long consecutive-digit run.
	SignalDigitSeq = "digit-seq"
	// SignalSANCount fires on a large certificate SAN set.
	SignalSANCount = "san-count"
	// SignalShortLivedBrand fires on a short-lived cert covering a brand match.
	SignalShortLivedBrand = "short-lived-brand"
	// SignalFreeCABrand fires on a free-CA issuer covering a brand match.
	SignalFreeCABrand = "free-ca-brand"
	// SignalHostedAbuse fires when a brand/typosquat/homoglyph signal hits a
	// tenant label on a watched free-hosting platform.
	SignalHostedAbuse = "hosted-abuse"
	// SignalMultiKeyword fires when three or more keywords matched.
	SignalMultiKeyword = "multi-keyword"
)

// Category values classify a hit by the profile bucket that produced its
// strongest match. They are persisted on the hit and filterable via
// `query --category`.
const (
	CategoryCrypto      = "crypto"
	CategoryPhishing    = "phishing"
	CategoryAI          = "ai"
	CategoryHostedAbuse = "hosted-abuse"
	CategoryTracker     = "tracker"
)
