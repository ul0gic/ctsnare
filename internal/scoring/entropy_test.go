package scoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShannonEntropy(t *testing.T) {
	assert.Equal(t, 0.0, shannonEntropy(""))
	assert.Equal(t, 0.0, shannonEntropy("aaaa")) // single symbol -> zero entropy
	// "ab" alternating -> 1 bit/char.
	assert.InDelta(t, 1.0, shannonEntropy("abab"), 0.001)
}

func TestScoreDGA(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		want   int
	}{
		// DGA-style labels trip the heuristic.
		{"high entropy base32 label", "a8f3kd91mznx.com", 1},
		{"consonant run plus digit", "x7g2k9qphdz.com", 1},
		{"random looking long label", "qz7kvbtxhwprmn.xyz", 1},
		// Legitimate labels stay clear (false-positive guards).
		{"microsoft", "microsoft.com", 0},
		{"anthropic", "anthropic.com", 0},
		{"wellsfargo", "wellsfargo.com", 0},
		{"bankofamerica", "bankofamerica.com", 0},
		{"cloudflare", "cloudflare.com", 0},
		{"developer mozilla subdomain", "developer.mozilla.org", 0},
		{"stackoverflow", "stackoverflow.com", 0},
		{"google short label below min len", "google.com", 0},
		{"openai short label", "openai.com", 0},
		// Short labels are never evaluated even if random.
		{"short random label skipped", "xk7q.com", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, scoreDGA(tt.domain), "scoreDGA(%q)", tt.domain)
		})
	}
}

func TestHasConsonantRunWithDigit(t *testing.T) {
	assert.True(t, hasConsonantRunWithDigit("x7g2k9qphdz")) // run qphdz + digits
	assert.False(t, hasConsonantRunWithDigit("microsoft1")) // no 4-consonant run
	assert.False(t, hasConsonantRunWithDigit("qphdzxmnbv")) // long run but no digit
	assert.False(t, hasConsonantRunWithDigit("wellsfargo")) // dictionary, no digit
}
