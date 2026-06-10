package config

// TLDTiers configures the two-tier suspicious-TLD scoring system. It lives in
// the TOML config under [tld_tiers] rather than in code because the underlying
// abuse data churns quarterly: registries are added to and removed from abuse
// blocklists (Interisle Cybercrime Supply Chain reports, Spamhaus's most-abused
// TLD tables) every quarter. Hard-coding the lists would mean a release cycle to
// react to a newly weaponized registry, so the lists are data, not logic.
//
// When either Burner or Cheap is set in the config file, it REPLACES the
// corresponding built-in default wholesale (it is not merged). This lets an
// operator track the current quarter's abuse landscape without recompiling.
// Per-profile suspicious_tlds remain additive cheap-tier entries on top of
// whatever this tier system resolves to, preserving backward compatibility.
type TLDTiers struct {
	// Burner is the +6 tier: registries dominated by disposable, bulk-registered
	// abuse domains. A burner-tier match alone reaches MED and is auto-stored;
	// any corroborating signal tips it to HIGH (>= 8).
	Burner []string `toml:"burner"`

	// Cheap is the +1 tier: low-cost registries with elevated but not dominant
	// abuse rates. A cheap-tier match needs corroboration to matter.
	Cheap []string `toml:"cheap"`
}

// TLD tier scoring weights.
const (
	// BurnerTLDPoints is the score contribution for a burner-tier TLD match.
	BurnerTLDPoints = 6
	// CheapTLDPoints is the score contribution for a cheap-tier TLD match.
	CheapTLDPoints = 1
)

// DefaultBurnerTLDs is the built-in burner tier (+6). These are registries
// observed to be dominated by disposable abuse infrastructure (toll-scam,
// smishing, throwaway phishing). Override via [tld_tiers] burner = [...].
var DefaultBurnerTLDs = []string{
	".su", ".ru", ".tk", ".ml", ".ga", ".cf", ".gq",
	".cfd", ".sbs", ".icu", ".bond", ".xin", ".top", ".cc",
}

// DefaultCheapTLDs is the built-in cheap tier (+1). Low-cost registries with
// elevated abuse rates. Override via [tld_tiers] cheap = [...].
var DefaultCheapTLDs = []string{
	".xyz", ".click", ".buzz", ".monster", ".quest", ".rest",
	".lol", ".win", ".bet", ".vip", ".shop", ".casino",
	".cyou", ".boats", ".mom",
}

// ResolveTLDTiers returns the effective burner and cheap tiers. A non-empty
// configured tier replaces the corresponding default; an empty (unset) tier
// falls back to the built-in default.
func ResolveTLDTiers(t TLDTiers) (burner, cheap []string) {
	burner = t.Burner
	if len(burner) == 0 {
		burner = DefaultBurnerTLDs
	}
	cheap = t.Cheap
	if len(cheap) == 0 {
		cheap = DefaultCheapTLDs
	}
	return burner, cheap
}
