package scoring

import (
	"math"
	"strings"
)

// dgaPoints is the bonus for a domain label that looks algorithmically generated.
const dgaPoints = 1

// Entropy heuristic tuning. We use Shannon entropy (bits per character) rather
// than an n-gram or dictionary model because it is O(n), allocation-free, and
// needs no embedded corpus — it runs on every CT firehose domain. Labels shorter
// than entropyMinLen are skipped: short strings have too few samples for entropy
// to be meaningful and produce noise.
//
// Entropy alone is a weak discriminator at DNS-label lengths — "stackoverflow"
// (3.55) and a random base32 label (3.46) overlap. So we combine two cheap
// signals and trip on either: (1) entropy above entropyThreshold, which catches
// high-symbol-variety labels, or (2) a long consonant run together with a digit,
// the classic shape of a DGA label like "x7g2k9qphdz". Real dictionary/brand
// labels ("microsoft", "wellsfargo", "developermozilla") have neither a digit
// nor entropy that high, so they stay clear.
const (
	entropyMinLen      = 10
	entropyThreshold   = 3.6 // bits/char
	dgaConsonantRunLen = 4
)

// scoreDGA returns dgaPoints if any registered-domain label of length >=
// entropyMinLen looks algorithmically generated, by either the entropy or the
// consonant-run-plus-digit signal. The TLD is ignored.
func scoreDGA(domainName string) int {
	registered := registeredPart(strings.ToLower(domainName))
	for _, label := range strings.Split(registered, ".") {
		if len(label) < entropyMinLen {
			continue
		}
		if shannonEntropy(label) > entropyThreshold || hasConsonantRunWithDigit(label) {
			return dgaPoints
		}
	}
	return 0
}

// hasConsonantRunWithDigit reports whether label contains a run of at least
// dgaConsonantRunLen consecutive consonants AND at least one digit — the typical
// unpronounceable, mixed shape of an algorithmically generated label. Pronounceable
// brand words rarely string four consonants together, and legitimate labels of
// this length seldom embed digits.
func hasConsonantRunWithDigit(label string) bool {
	run := 0
	hasDigit := false
	longRun := false
	for i := 0; i < len(label); i++ {
		c := label[i]
		switch {
		case c >= '0' && c <= '9':
			hasDigit = true
			run = 0
		case isConsonant(c):
			run++
			if run >= dgaConsonantRunLen {
				longRun = true
			}
		default:
			run = 0
		}
	}
	return longRun && hasDigit
}

// isConsonant reports whether c is an ASCII consonant letter.
func isConsonant(c byte) bool {
	if c < 'a' || c > 'z' {
		return false
	}
	switch c {
	case 'a', 'e', 'i', 'o', 'u':
		return false
	default:
		return true
	}
}

// shannonEntropy returns the Shannon entropy of s in bits per character. An empty
// string has zero entropy. Bytes are used as symbols, which is adequate for the
// ASCII-dominant DNS label space.
func shannonEntropy(s string) float64 {
	if s == "" {
		return 0
	}
	var freq [256]int
	for i := 0; i < len(s); i++ {
		freq[s[i]]++
	}
	n := float64(len(s))
	var entropy float64
	for _, c := range freq {
		if c == 0 {
			continue
		}
		p := float64(c) / n
		entropy -= p * math.Log2(p)
	}
	return entropy
}
