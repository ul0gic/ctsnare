package domain

// Profile is the runtime representation of a keyword profile used for domain scoring.
// Profiles are loaded from built-in definitions or user-defined TOML config sections.
type Profile struct {
	// Name is the unique identifier for this profile (e.g., "crypto", "phishing", "all").
	Name string

	// Keywords is the list of terms to search for in domain names.
	// Matching is case-insensitive substring matching. Each match contributes 2 points.
	Keywords []string

	// SuspiciousTLDs is the list of top-level domains that receive a +1 score bonus.
	// Include the leading dot (e.g., ".xyz", ".top").
	SuspiciousTLDs []string

	// SkipSuffixes is the list of domain suffixes to exclude from scoring entirely.
	// Domains matching any of these suffixes are returned with a score of zero
	// regardless of keyword content. Used to filter infrastructure noise.
	SkipSuffixes []string

	// Description is a human-readable summary of the profile's purpose.
	Description string
}
