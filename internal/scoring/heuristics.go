package scoring

import (
	"strings"
	"unicode"
)

// shortKeywordMaxLen is the length at or below which a keyword must match on a
// label boundary rather than as a bare substring. Short brand tokens like "dhl"
// otherwise match random gibberish (e.g. "...edhljmrsqpqx...") and flood the
// feed with false positives. Longer keywords keep substring semantics, where an
// embedded match is far more likely to be intentional.
const shortKeywordMaxLen = 4

// matchKeywords returns the list of keywords found in the domain string.
// Matching is case-insensitive. Keywords of length <= shortKeywordMaxLen must
// match on a token boundary (delimited by '.', '-', start, or end); longer
// keywords match as plain substrings. The caller assigns per-tier points.
func matchKeywords(domain string, keywords []string, _ int) []string {
	lower := strings.ToLower(domain)
	var matched []string
	for _, kw := range keywords {
		if keywordMatches(lower, strings.ToLower(kw)) {
			matched = append(matched, kw)
		}
	}
	return matched
}

// keywordMatches reports whether kw matches haystack under the tier rules:
// short keywords (<= shortKeywordMaxLen) require a token boundary on both
// sides of the match; longer keywords match anywhere.
func keywordMatches(haystack, kw string) bool {
	if kw == "" {
		return false
	}
	if len(kw) > shortKeywordMaxLen {
		return strings.Contains(haystack, kw)
	}
	return matchesAtBoundary(haystack, kw)
}

// matchesAtBoundary reports whether kw appears in haystack delimited on both
// sides by a token boundary: the start/end of the string or a '.'/'-'
// separator. This treats domain labels and hyphen-separated segments as tokens,
// so "dhl" matches "dhl-tracking.icu" and "track.dhl.evil.com" but not the
// substring inside "...sadbirds..." or a random high-entropy label.
func matchesAtBoundary(haystack, kw string) bool {
	from := 0
	for {
		idx := strings.Index(haystack[from:], kw)
		if idx < 0 {
			return false
		}
		start := from + idx
		end := start + len(kw)
		if isBoundary(haystack, start-1) && isBoundary(haystack, end) {
			return true
		}
		from = start + 1
		if from >= len(haystack) {
			return false
		}
	}
}

// isBoundary reports whether position i in s is a token boundary: out of range
// (string edge) or a '.'/'-' delimiter.
func isBoundary(s string, i int) bool {
	if i < 0 || i >= len(s) {
		return true
	}
	return s[i] == '.' || s[i] == '-'
}

// scoreTLDTier classifies the domain's TLD into the burner (+6) or cheap (+1)
// tier and returns the points and signal key. Profile-level suspiciousTLDs are
// treated as additional cheap-tier entries for backward compatibility with
// custom profiles. Burner takes precedence over cheap when a TLD appears in
// both. Returns (0, "") when no tier matches.
func (e *Engine) scoreTLDTier(domain string, profileSuspicious []string) (int, string) {
	lower := strings.ToLower(domain)
	if hasAnyTLDSuffix(lower, e.burnerTLDs) {
		return tldBurnerPoints, SignalBurnerTLD
	}
	if hasAnyTLDSuffix(lower, e.cheapTLDs) || hasAnyTLDSuffix(lower, profileSuspicious) {
		return tldCheapPoints, SignalSuspiciousTLD
	}
	return 0, ""
}

// TLD tier weights mirror config.BurnerTLDPoints / config.CheapTLDPoints. They
// are duplicated as local constants to keep the scoring package free of a
// config import in the hot path; the values are asserted equal by a test.
const (
	tldBurnerPoints = 6
	tldCheapPoints  = 1
)

// hasAnyTLDSuffix reports whether lower ends with any of the given TLD suffixes.
func hasAnyTLDSuffix(lower string, tlds []string) bool {
	for _, tld := range tlds {
		if strings.HasSuffix(lower, strings.ToLower(tld)) {
			return true
		}
	}
	return false
}

// numericSLDPoints is the bonus for an all-digit registered SLD on a tiered TLD.
const numericSLDPoints = 3

// scoreNumericSLD returns numericSLDPoints when the registered SLD label is
// entirely digits (length >= 3) AND the TLD is in the burner or cheap tier.
// Numeric SLDs on neutral TLDs (e.g. 12306.cn) are frequently legitimate, so
// this fires only when corroborated by a suspicious TLD. Per Interisle, 4-6
// digit numeric SLDs on burner registries are near-diagnostic of toll-scam and
// smishing infrastructure. tier is the signal key returned by scoreTLDTier.
func scoreNumericSLD(domain, tier string) int {
	if tier != SignalBurnerTLD && tier != SignalSuspiciousTLD {
		return 0
	}
	sld := registeredSLD(domain)
	if len(sld) < 3 {
		return 0
	}
	for i := 0; i < len(sld); i++ {
		if sld[i] < '0' || sld[i] > '9' {
			return 0
		}
	}
	return numericSLDPoints
}

// deceptivePrefixPoints is the bonus for a com-/www-/-com deceptive label shape.
const deceptivePrefixPoints = 2

// scoreDeceptivePrefix returns deceptivePrefixPoints when the leading label or
// the registered SLD begins with "com-" or "www-", or the SLD ends with "-com"
// (e.g. paypal-com.icu, com-tollpay.xin). These shapes spoof a "...com." path
// inside the label. To stay cheap and false-positive-resistant it fires only
// when paired with at least one keyword match, a tiered TLD, or a brand hit.
func scoreDeceptivePrefix(domain string, keywordHit, tieredTLD, brandHit bool) int {
	if !keywordHit && !tieredTLD && !brandHit {
		return 0
	}
	lower := strings.ToLower(domain)
	labels := strings.Split(lower, ".")
	if len(labels) == 0 {
		return 0
	}
	leading := labels[0]
	sld := registeredSLD(lower)
	if hasDeceptiveShape(leading) || hasDeceptiveShape(sld) {
		return deceptivePrefixPoints
	}
	return 0
}

// hasDeceptiveShape reports whether label carries a com-/www-/-com spoof shape.
func hasDeceptiveShape(label string) bool {
	if label == "" {
		return false
	}
	return strings.HasPrefix(label, "com-") ||
		strings.HasPrefix(label, "www-") ||
		strings.HasSuffix(label, "-com")
}

// scoreDomainLength returns +1 if the registered domain portion (everything
// before the last dot-separated TLD) exceeds 30 characters.
func scoreDomainLength(domain string) int {
	registered := registeredPart(domain)
	if len(registered) > 30 {
		return 1
	}
	return 0
}

// scoreHyphenDensity returns +1 if the registered domain contains 2 or more
// hyphens, a common pattern in phishing and typosquatting domains.
func scoreHyphenDensity(domain string) int {
	registered := registeredPart(domain)
	if strings.Count(registered, "-") >= 2 {
		return 1
	}
	return 0
}

// scoreNumberSequences returns +1 if the domain contains 4 or more
// consecutive digits, common in auto-generated malicious domains.
func scoreNumberSequences(domain string) int {
	consecutive := 0
	for _, r := range domain {
		if unicode.IsDigit(r) {
			consecutive++
			if consecutive >= 4 {
				return 1
			}
		} else {
			consecutive = 0
		}
	}
	return 0
}

// scoreMultiKeywordBonus returns +2 if 3 or more keywords matched,
// indicating a higher likelihood of intentional impersonation.
func scoreMultiKeywordBonus(matchCount int) int {
	if matchCount >= 3 {
		return 2
	}
	return 0
}

// registeredPart extracts the registered domain name excluding the TLD.
// For "evil-bank-login.phishing.xyz" this returns "evil-bank-login.phishing".
// For simple domains like "example.com" this returns "example".
func registeredPart(domain string) string {
	idx := strings.LastIndex(domain, ".")
	if idx < 0 {
		return domain
	}
	return domain[:idx]
}

// registeredSLD returns the second-level domain label (the rightmost label of
// the registered part). For "paypal-com.icu" this is "paypal-com"; for
// "login.paypal.xyz" it is "paypal"; for "example.com" it is "example".
func registeredSLD(domain string) string {
	registered := registeredPart(strings.ToLower(domain))
	if idx := strings.LastIndex(registered, "."); idx >= 0 {
		return registered[idx+1:]
	}
	return registered
}
