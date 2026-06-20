package config

// TLDTiers is the two-tier suspicious-TLD system under [tld_tiers]; lists live in
// config because abuse data churns quarterly. A set tier replaces its default wholesale.
type TLDTiers struct {
	// Burner is the +6 tier: a match alone reaches MED; any corroboration tips it to HIGH.
	Burner []string `toml:"burner"`

	// Cheap is the +1 tier: needs corroboration to matter.
	Cheap []string `toml:"cheap"`
}

// TLD tier scoring weights.
const (
	BurnerTLDPoints = 6
	CheapTLDPoints  = 1
)

// DefaultBurnerTLDs is the built-in +6 tier; override via [tld_tiers] burner.
var DefaultBurnerTLDs = []string{
	".su", ".ru", ".tk", ".ml", ".ga", ".cf", ".gq",
	".cfd", ".sbs", ".icu", ".bond", ".xin", ".top", ".cc",
}

// DefaultCheapTLDs is the built-in +1 tier; override via [tld_tiers] cheap.
var DefaultCheapTLDs = []string{
	".xyz", ".click", ".buzz", ".monster", ".quest", ".rest",
	".lol", ".win", ".bet", ".vip", ".shop", ".casino",
	".cyou", ".boats", ".mom",
}

// ResolveTLDTiers returns the effective tiers: a non-empty configured tier
// replaces its default, an unset tier falls back to the built-in.
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
