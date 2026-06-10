package scoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchKeywords(t *testing.T) {
	tests := []struct {
		name        string
		domain      string
		keywords    []string
		wantMatched []string
	}{
		{
			name:        "single brand keyword match",
			domain:      "bitcoin-shop.com",
			keywords:    []string{"bitcoin"},
			wantMatched: []string{"bitcoin"},
		},
		{
			name:        "single generic keyword match",
			domain:      "login-shop.com",
			keywords:    []string{"login"},
			wantMatched: []string{"login"},
		},
		{
			name:        "multiple generic keyword matches",
			domain:      "bitcoin-wallet-login.com",
			keywords:    []string{"bitcoin", "wallet", "login"},
			wantMatched: []string{"bitcoin", "wallet", "login"},
		},
		{
			name:        "case insensitive matching",
			domain:      "BITCOIN-WALLET.com",
			keywords:    []string{"bitcoin", "wallet"},
			wantMatched: []string{"bitcoin", "wallet"},
		},
		{
			name:        "mixed case keywords and domain with short keyword on boundary",
			domain:      "Bitcoin-Shop.COM",
			keywords:    []string{"BITCOIN", "Shop"}, // "shop" is len 4 -> boundary match
			wantMatched: []string{"BITCOIN", "Shop"},
		},
		{
			name:        "long keyword partial match within label",
			domain:      "mybitcoindex.com",
			keywords:    []string{"bitcoin"}, // len 7 -> substring semantics
			wantMatched: []string{"bitcoin"},
		},
		{
			name:        "no match",
			domain:      "example.com",
			keywords:    []string{"bitcoin", "wallet"},
			wantMatched: nil,
		},
		{
			name:        "empty domain",
			domain:      "",
			keywords:    []string{"bitcoin"},
			wantMatched: nil,
		},
		{
			name:        "empty keywords list",
			domain:      "bitcoin.com",
			keywords:    []string{},
			wantMatched: nil,
		},
		{
			name:        "nil keywords list",
			domain:      "bitcoin.com",
			keywords:    nil,
			wantMatched: nil,
		},
		{
			name:        "keyword is entire domain",
			domain:      "bitcoin",
			keywords:    []string{"bitcoin"},
			wantMatched: []string{"bitcoin"},
		},
		{
			name:        "overlapping keyword substrings",
			domain:      "wallet-walletconnect.com",
			keywords:    []string{"wallet"},
			wantMatched: []string{"wallet"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched := matchKeywords(tt.domain, tt.keywords, 0)
			assert.Equal(t, tt.wantMatched, matched)
		})
	}
}

// TestMatchKeywords_ShortKeywordBoundary proves short keywords (<= 4 chars)
// match only on label/token boundaries, killing the substring false positives
// observed on the live CT firehose while keeping legitimate matches.
func TestMatchKeywords_ShortKeywordBoundary(t *testing.T) {
	brand := []string{"dhl"}

	// Real false positive observed live: "dhl" buried inside a random label.
	gibberish := "mvjdffvpgfedhljmrsqpqx.fis.cmh.globaltest.prod.sadbirds.aws.dev"
	assert.Nil(t, matchKeywords(gibberish, brand, 0),
		"short keyword must not substring-match gibberish")

	// True positives: token-boundary hits must still match.
	assert.Equal(t, []string{"dhl"}, matchKeywords("dhl-tracking.icu", brand, 0),
		"hyphen boundary should match")
	assert.Equal(t, []string{"dhl"}, matchKeywords("track.dhl.evil.com", brand, 0),
		"dot boundaries should match")
	assert.Equal(t, []string{"dhl"}, matchKeywords("dhl.com", brand, 0),
		"label-as-whole should match")

	// Longer keyword keeps substring semantics.
	assert.Equal(t, []string{"paypal"}, matchKeywords("mypaypaltest.com", []string{"paypal"}, 0),
		"long keyword keeps substring match")
}

func TestScoreTLDTier(t *testing.T) {
	engine := NewEngine() // built-in tier defaults

	tests := []struct {
		name        string
		domain      string
		profileTLDs []string
		wantPoints  int
		wantSignal  string
	}{
		{"burner tld scores 6", "evil.tk", nil, tldBurnerPoints, SignalBurnerTLD},
		{"burner takes precedence over cheap", "evil.top", nil, tldBurnerPoints, SignalBurnerTLD},
		{"cheap tld scores 1", "evil.xyz", nil, tldCheapPoints, SignalSuspiciousTLD},
		{"neutral tld scores 0", "example.com", nil, 0, ""},
		{"profile suspicious tld treated as cheap", "evil.example", []string{".example"}, tldCheapPoints, SignalSuspiciousTLD},
		{"case insensitive burner", "evil.TK", nil, tldBurnerPoints, SignalBurnerTLD},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			points, signal := engine.scoreTLDTier(tt.domain, tt.profileTLDs)
			assert.Equal(t, tt.wantPoints, points)
			assert.Equal(t, tt.wantSignal, signal)
		})
	}
}

func TestScoreDomainLength(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		want   int
	}{
		{
			name:   "short domain no score",
			domain: "example.com",
			want:   0,
		},
		{
			name:   "exactly 30 chars registered part no score",
			domain: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.com", // 30 chars before .com
			want:   0,
		},
		{
			name:   "31 chars registered part scores",
			domain: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.com", // 31 chars before .com
			want:   1,
		},
		{
			name:   "very long domain scores",
			domain: "this-is-a-really-long-domain-name-that-should-definitely-score.com",
			want:   1,
		},
		{
			name:   "single char domain no score",
			domain: "a.com",
			want:   0,
		},
		{
			name:   "domain without TLD uses full string as registered part",
			domain: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", // 31 chars, no dot
			want:   1,
		},
		{
			name:   "empty domain no score",
			domain: "",
			want:   0,
		},
		{
			name:   "subdomain counts in registered part",
			domain: "sub.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.com",
			want:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scoreDomainLength(tt.domain)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestScoreHyphenDensity(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		want   int
	}{
		{
			name:   "zero hyphens no score",
			domain: "example.com",
			want:   0,
		},
		{
			name:   "one hyphen no score",
			domain: "my-site.com",
			want:   0,
		},
		{
			name:   "two hyphens scores",
			domain: "my-evil-site.com",
			want:   1,
		},
		{
			name:   "three hyphens scores",
			domain: "my-very-evil-site.com",
			want:   1,
		},
		{
			name:   "many hyphens scores",
			domain: "a-b-c-d-e-f.com",
			want:   1,
		},
		{
			name:   "hyphens in TLD not counted",
			domain: "example.co-uk",
			want:   0,
		},
		{
			name:   "empty domain no score",
			domain: "",
			want:   0,
		},
		{
			name:   "consecutive hyphens counted",
			domain: "evil--site.com",
			want:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scoreHyphenDensity(tt.domain)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestScoreNumberSequences(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		want   int
	}{
		{
			name:   "no digits no score",
			domain: "example.com",
			want:   0,
		},
		{
			name:   "three consecutive digits no score",
			domain: "evil123.com",
			want:   0,
		},
		{
			name:   "exactly four consecutive digits scores",
			domain: "evil1234.com",
			want:   1,
		},
		{
			name:   "ten consecutive digits scores",
			domain: "evil1234567890.com",
			want:   1,
		},
		{
			name:   "scattered digits not consecutive no score",
			domain: "e1v2i3l.com",
			want:   0,
		},
		{
			name:   "digits separated by letters no score",
			domain: "abc12def34.com",
			want:   0,
		},
		{
			name:   "digits at end of domain",
			domain: "evil.com1234",
			want:   1,
		},
		{
			name:   "digits at start of domain",
			domain: "1234evil.com",
			want:   1,
		},
		{
			name:   "empty domain no score",
			domain: "",
			want:   0,
		},
		{
			name:   "only digits domain",
			domain: "12345.com",
			want:   1,
		},
		{
			name:   "unicode digits count",
			domain: "evil\u0661\u0662\u0663\u0664.com", // Arabic-Indic digits
			want:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scoreNumberSequences(tt.domain)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestScoreMultiKeywordBonus(t *testing.T) {
	tests := []struct {
		name       string
		matchCount int
		want       int
	}{
		{
			name:       "zero matches no bonus",
			matchCount: 0,
			want:       0,
		},
		{
			name:       "one match no bonus",
			matchCount: 1,
			want:       0,
		},
		{
			name:       "two matches no bonus",
			matchCount: 2,
			want:       0,
		},
		{
			name:       "three matches scores bonus",
			matchCount: 3,
			want:       2,
		},
		{
			name:       "five matches scores bonus",
			matchCount: 5,
			want:       2,
		},
		{
			name:       "large match count scores bonus",
			matchCount: 100,
			want:       2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scoreMultiKeywordBonus(tt.matchCount)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRegisteredPart(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		want   string
	}{
		{
			name:   "simple domain",
			domain: "example.com",
			want:   "example",
		},
		{
			name:   "subdomain",
			domain: "sub.example.com",
			want:   "sub.example",
		},
		{
			name:   "no dot returns full string",
			domain: "localhost",
			want:   "localhost",
		},
		{
			name:   "empty string",
			domain: "",
			want:   "",
		},
		{
			name:   "trailing dot",
			domain: "example.",
			want:   "example",
		},
		{
			name:   "multiple subdomains",
			domain: "a.b.c.d.com",
			want:   "a.b.c.d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := registeredPart(tt.domain)
			assert.Equal(t, tt.want, got)
		})
	}
}
