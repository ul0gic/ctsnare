package domain

// Profile is a keyword profile for domain scoring, loaded from built-in
// definitions or user-defined TOML config sections.
type Profile struct {
	Name string `toml:"name"`

	// Generic tier: case-insensitive substring match, +1 point each.
	Keywords []string `toml:"keywords"`

	// High-precision brand tier: +3 points each, feeds typosquat/short-lived-cert
	// heuristics. Optional and additive — pre-existing custom profiles still work.
	BrandKeywords []string `toml:"brand_keywords"`

	// Include the leading dot (e.g. ".xyz"); each gives a +1 bonus.
	SuspiciousTLDs []string `toml:"suspicious_tlds"`

	// Matching domains score zero regardless of keywords; filters infrastructure noise.
	SkipSuffixes []string `toml:"skip_suffixes"`

	Description string `toml:"description"`
}
