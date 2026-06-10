package domain

// Profile is the runtime representation of a keyword profile used for domain scoring.
// Profiles are loaded from built-in definitions or user-defined TOML config sections.
type Profile struct {
	// Name is the unique identifier for this profile (e.g., "crypto", "phishing", "all").
	Name string `toml:"name"`

	// Keywords is the generic-tier list of terms to search for in domain names.
	// Matching is case-insensitive substring matching. Each match contributes
	// 1 point. These are broad, noise-prone terms (e.g. "login", "swap", "token")
	// that gain signal only in combination with other heuristics.
	Keywords []string `toml:"keywords"`

	// BrandKeywords is the high-precision tier: brand names and impersonation
	// targets (e.g. "metamask", "paypal", "openai"). Each match contributes 3
	// points and participates in the typosquat and short-lived-cert heuristics.
	// This field is additive and optional — custom TOML profiles that predate it
	// continue to work, with all of their `keywords` treated as the generic tier.
	BrandKeywords []string `toml:"brand_keywords"`

	// SuspiciousTLDs is the list of top-level domains that receive a +1 score bonus.
	// Include the leading dot (e.g., ".xyz", ".top").
	SuspiciousTLDs []string `toml:"suspicious_tlds"`

	// SkipSuffixes is the list of domain suffixes to exclude from scoring entirely.
	// Domains matching any of these suffixes are returned with a score of zero
	// regardless of keyword content. Used to filter infrastructure noise.
	SkipSuffixes []string `toml:"skip_suffixes"`

	// Description is a human-readable summary of the profile's purpose.
	Description string `toml:"description"`
}
