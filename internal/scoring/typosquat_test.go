package scoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDamerauLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		max  int
		want int
	}{
		{"paypal", "paypal", 2, 0},
		{"paypol", "paypal", 2, 1},  // substitution
		{"paypall", "paypal", 2, 1}, // insertion
		{"papal", "paypal", 2, 1},   // deletion
		{"paypla", "paypal", 2, 1},  // transposition
		{"coinbase", "coinbsae", 2, 1},
		{"example", "paypal", 2, 3},   // far apart -> early-bail returns max+1
		{"", "paypal", 6, 6},          // empty a
		{"paypal", "", 6, 6},          // empty b
		{"metamask", "metmask", 2, 1}, // deletion in 8-char brand
	}
	for _, tt := range tests {
		got := damerauLevenshtein(tt.a, tt.b, tt.max)
		assert.Equal(t, tt.want, got, "damerauLevenshtein(%q,%q,%d)", tt.a, tt.b, tt.max)
	}
}

func TestAllowedDistance(t *testing.T) {
	assert.Equal(t, 0, allowedDistance(4))
	assert.Equal(t, 1, allowedDistance(5))
	assert.Equal(t, 1, allowedDistance(7))
	assert.Equal(t, 2, allowedDistance(8))
	assert.Equal(t, 2, allowedDistance(12))
}

func TestScoreTyposquat(t *testing.T) {
	brand := []string{"paypal", "coinbase", "metamask", "hsbc"}

	tests := []struct {
		name        string
		domain      string
		wantScore   int
		wantMatched []string
	}{
		{
			name:        "exact substring is not a typosquat",
			domain:      "paypal-login.com",
			wantScore:   0,
			wantMatched: nil,
		},
		{
			name:        "one-edit substitution scores",
			domain:      "paypol.com",
			wantScore:   3,
			wantMatched: []string{"~paypal"},
		},
		{
			name:        "transposition scores",
			domain:      "paypla.com",
			wantScore:   3,
			wantMatched: []string{"~paypal"},
		},
		{
			name:        "two edits on long brand scores",
			domain:      "metmsk.com", // metamask -> metmsk = distance 2, len 8 accepts <= 2
			wantScore:   3,
			wantMatched: []string{"~metamask"},
		},
		{
			name:        "three edits on long brand rejected",
			domain:      "mtmsk.com", // metamask -> mtmsk = distance 3 -> reject
			wantScore:   0,
			wantMatched: nil,
		},
		{
			name:        "deletion on long brand scores",
			domain:      "metmask.com", // metamask -> metmask = distance 1
			wantScore:   3,
			wantMatched: []string{"~metamask"},
		},
		{
			name:        "short brand below min length not fuzzy matched",
			domain:      "hsbd.com", // hsbc is len 4 -> never fuzzy matched
			wantScore:   0,
			wantMatched: nil,
		},
		{
			name:        "unrelated domain no score",
			domain:      "example.com",
			wantScore:   0,
			wantMatched: nil,
		},
		{
			name:        "subdomain label matched",
			domain:      "paypol.evil.xyz",
			wantScore:   3,
			wantMatched: []string{"~paypal"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, matched := scoreTyposquat(tt.domain, brand)
			assert.Equal(t, tt.wantScore, score)
			assert.Equal(t, tt.wantMatched, matched)
		})
	}
}

// TestScoreTyposquat_FalsePositiveGuard ensures common legitimate brand-adjacent
// words do not trip the typosquat heuristic against the live brand list.
func TestScoreTyposquat_FalsePositiveGuard(t *testing.T) {
	brand := []string{"paypal", "coinbase", "microsoft", "anthropic", "openai"}
	legit := []string{
		"example.com",
		"github.com",
		"wikipedia.org",
		"stackoverflow.com",
		"cloudflare.com",
		"developer.mozilla.org",
	}
	for _, d := range legit {
		score, matched := scoreTyposquat(d, brand)
		assert.Equal(t, 0, score, "legit domain %q should not typosquat-match (matched=%v)", d, matched)
	}
}

func BenchmarkScoreTyposquat(b *testing.B) {
	brand := []string{
		"paypal", "coinbase", "metamask", "microsoft", "anthropic",
		"openai", "wellsfargo", "bankofamerica", "binance", "docusign",
	}
	domains := []string{
		"paypol-secure-login.com", "example.com", "metmask.io",
		"a-very-long-unrelated-domain-name.net", "github.com",
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scoreTyposquat(domains[i%len(domains)], brand)
	}
}
