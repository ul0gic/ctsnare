package scoring

import (
	"math"
	"strings"
)

const dgaPoints = 1

// Entropy alone is a weak discriminator at label lengths ("stackoverflow" 3.55
// vs random base32 3.46 overlap), so a high consonant-run-plus-digit shape is OR'd in.
const (
	entropyMinLen      = 10
	entropyThreshold   = 3.6
	dgaConsonantRunLen = 4
)

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
