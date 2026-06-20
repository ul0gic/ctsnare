package scoring

import (
	"strings"
	"unicode"
)

// At/below this length a keyword must match on a token boundary, else short tokens
// like "dhl" match random gibberish ("...edhljmrsqpqx...") and flood the feed.
const shortKeywordMaxLen = 4

// matchKeywords returns the keywords found in domain (case-insensitive); the caller assigns points.
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

func keywordMatches(haystack, kw string) bool {
	if kw == "" {
		return false
	}
	if len(kw) > shortKeywordMaxLen {
		return strings.Contains(haystack, kw)
	}
	return matchesAtBoundary(haystack, kw)
}

// matchesAtBoundary reports whether kw appears in haystack delimited on both sides
// by a token boundary (string edge or '.'/'-'), treating labels and hyphen segments as tokens.
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

func isBoundary(s string, i int) bool {
	if i < 0 || i >= len(s) {
		return true
	}
	return s[i] == '.' || s[i] == '-'
}

// scoreTLDTier returns the points and signal key for the domain's TLD tier (burner
// over cheap), with profile suspiciousTLDs as extra cheap-tier entries; (0, "") if none.
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

// Duplicated from config to keep a config import out of the hot path; a test asserts equality.
const (
	tldBurnerPoints = 6
	tldCheapPoints  = 1
)

func hasAnyTLDSuffix(lower string, tlds []string) bool {
	for _, tld := range tlds {
		if strings.HasSuffix(lower, strings.ToLower(tld)) {
			return true
		}
	}
	return false
}

const numericSLDPoints = 3

// scoreNumericSLD fires on an all-digit SLD (len >= 3) only on a tiered TLD: numeric
// SLDs on neutral TLDs (12306.cn) are often legitimate, but per Interisle near-diagnostic on burner registries.
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

const deceptivePrefixPoints = 2

// scoreDeceptivePrefix fires on a com-/www-/-com label shape (paypal-com.icu) that spoofs
// a "...com." path, but only with corroboration: a keyword match, tiered TLD, or brand hit.
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

func hasDeceptiveShape(label string) bool {
	if label == "" {
		return false
	}
	return strings.HasPrefix(label, "com-") ||
		strings.HasPrefix(label, "www-") ||
		strings.HasSuffix(label, "-com")
}

// scoreDomainLength returns +1 when the registered domain portion exceeds 30 characters.
func scoreDomainLength(domain string) int {
	registered := registeredPart(domain)
	if len(registered) > 30 {
		return 1
	}
	return 0
}

// scoreHyphenDensity returns +1 when the registered domain contains 2 or more hyphens.
func scoreHyphenDensity(domain string) int {
	registered := registeredPart(domain)
	if strings.Count(registered, "-") >= 2 {
		return 1
	}
	return 0
}

// scoreNumberSequences returns +1 when the domain contains 4 or more consecutive digits.
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

// scoreMultiKeywordBonus returns +2 when 3 or more keywords matched.
func scoreMultiKeywordBonus(matchCount int) int {
	if matchCount >= 3 {
		return 2
	}
	return 0
}

// registeredPart drops the TLD: "evil.phishing.xyz" -> "evil.phishing", "example.com" -> "example".
func registeredPart(domain string) string {
	idx := strings.LastIndex(domain, ".")
	if idx < 0 {
		return domain
	}
	return domain[:idx]
}

// registeredSLD returns the rightmost label of the registered part: "login.paypal.xyz" -> "paypal".
func registeredSLD(domain string) string {
	registered := registeredPart(strings.ToLower(domain))
	if idx := strings.LastIndex(registered, "."); idx >= 0 {
		return registered[idx+1:]
	}
	return registered
}
