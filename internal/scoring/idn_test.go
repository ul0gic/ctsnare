package scoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScorePunycode(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		want   int
	}{
		{"plain ascii no score", "paypal.com", 0},
		{"punycode label scores", "xn--pypal-4ve.com", 2},
		{"punycode in subdomain scores", "login.xn--80ak6aa92e.com", 2},
		{"uppercase XN-- still matches", "XN--PYPAL-4VE.com", 2},
		{"ascii with xn substring but not label", "foxn-news.com", 0},
		{"empty domain", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, scorePunycode(tt.domain))
		})
	}
}

func TestFoldConfusables(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"leet digits", "p4yp0l", "paypol"},
		{"cyrillic a", "pаypal", "paypal"},
		{"greek o", "cοinbase", "coinbase"},
		{"accented", "páypal", "paypal"},
		{"plain ascii unchanged", "paypal", "paypal"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, foldConfusables(tt.in))
		})
	}
}

func TestScoreConfusableKeywords(t *testing.T) {
	brand := []string{"paypal", "coinbase"}
	generic := []string{"login"}

	tests := []struct {
		name        string
		domain      string
		wantScore   int
		wantMatched []string
	}{
		{
			name:        "non-punycode domain skipped",
			domain:      "paypal-login.com",
			wantScore:   0,
			wantMatched: nil,
		},
		{
			name:        "cyrillic homograph paypal matches brand",
			domain:      "xn--pypal-4ve.com", // pаypal with Cyrillic а
			wantScore:   3,
			wantMatched: []string{"*paypal"},
		},
		{
			name:        "punycode label that decodes to nothing relevant",
			domain:      "xn--80ak6aa92e.com", // apple in Cyrillic homoglyphs (not in keyword set)
			wantScore:   0,
			wantMatched: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, matched := scoreConfusableKeywords(tt.domain, brand, generic)
			assert.Equal(t, tt.wantScore, score)
			assert.Equal(t, tt.wantMatched, matched)
		})
	}
}

// TestConfusable_NoDoubleCount guards that a keyword already present in the raw
// ASCII form is NOT re-credited by the confusable pass.
func TestConfusable_NoDoubleCount(t *testing.T) {
	// Domain has a real punycode label but "paypal" appears literally in ASCII.
	score, matched := scoreConfusableKeywords("paypal.xn--80ak6aa92e.com",
		[]string{"paypal"}, nil)
	assert.Equal(t, 0, score)
	assert.Nil(t, matched)
}
