package scoring

import (
	"strings"
	"unicode"
)

// matchKeywords returns the score and list of matched keywords found in the
// domain string. Each keyword match contributes 2 points. Matching is
// case-insensitive substring matching.
func matchKeywords(domain string, keywords []string) (score int, matched []string) {
	lower := strings.ToLower(domain)
	for _, kw := range keywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			score += 2
			matched = append(matched, kw)
		}
	}
	return score, matched
}

// scoreTLD returns +1 if the domain ends with any of the suspicious TLDs.
func scoreTLD(domain string, suspiciousTLDs []string) int {
	lower := strings.ToLower(domain)
	for _, tld := range suspiciousTLDs {
		if strings.HasSuffix(lower, strings.ToLower(tld)) {
			return 1
		}
	}
	return 0
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
	count := strings.Count(registered, "-")
	if count >= 2 {
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
