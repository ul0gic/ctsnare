package scoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ul0gic/ctsnare/internal/domain"
)

func testProfile() *domain.Profile {
	return &domain.Profile{
		Name:          "test",
		BrandKeywords: []string{"bitcoin", "paypal"},
		Keywords:      []string{"login", "wallet", "exchange", "verify"},
		SuspiciousTLDs: []string{
			".xyz", ".top", ".icu",
		},
		SkipSuffixes: []string{
			"cloudflaressl.com",
			"amazonaws.com",
		},
	}
}

func TestEngine_Score(t *testing.T) {
	engine := NewEngine()
	profile := testProfile()

	tests := []struct {
		name           string
		domain         string
		wantMinScore   int
		wantMaxScore   int
		wantSeverity   domain.Severity
		wantKeywords   []string
		wantNoKeywords bool
	}{
		{
			name:         "single brand keyword match scores LOW",
			domain:       "bitcoin-news.com",
			wantMinScore: 3,
			wantMaxScore: 4,
			wantSeverity: domain.SeverityLow,
			wantKeywords: []string{"bitcoin"},
		},
		{
			name:         "brand plus generic keyword scores LOW",
			domain:       "bitcoin-wallet.com",
			wantMinScore: 4,
			wantMaxScore: 4,
			wantSeverity: domain.SeverityLow,
			wantKeywords: []string{"bitcoin", "wallet"},
		},
		{
			name:         "three keywords with bonus scores HIGH",
			domain:       "bitcoin-wallet-login.xyz",
			wantMinScore: 8,
			wantMaxScore: 20,
			wantSeverity: domain.SeverityHigh,
			wantKeywords: []string{"bitcoin", "login", "wallet"},
		},
		{
			name:         "brand keyword plus suspicious TLD scores LOW",
			domain:       "bitcoin-shop.xyz",
			wantMinScore: 4,
			wantMaxScore: 5,
			wantSeverity: domain.SeverityLow,
			wantKeywords: []string{"bitcoin"},
		},
		{
			name:           "skip suffix returns zero score",
			domain:         "bitcoin-something.cloudflaressl.com",
			wantMinScore:   0,
			wantMaxScore:   0,
			wantSeverity:   "",
			wantNoKeywords: true,
		},
		{
			name:           "no matching keywords returns zero",
			domain:         "example.com",
			wantMinScore:   0,
			wantMaxScore:   0,
			wantSeverity:   "",
			wantNoKeywords: true,
		},
		{
			name:         "case-insensitive matching",
			domain:       "BITCOIN-WALLET.com",
			wantMinScore: 4,
			wantMaxScore: 4,
			wantSeverity: domain.SeverityLow,
			wantKeywords: []string{"bitcoin", "wallet"},
		},
		{
			name:         "long domain adds point",
			domain:       "this-is-a-very-long-bitcoin-domain-name.com",
			wantMinScore: 5,
			wantMaxScore: 6,
			wantSeverity: domain.SeverityMed,
		},
		{
			name:         "hyphen-heavy domain scores HIGH with brand and multi-bonus",
			domain:       "bitcoin-secure-login-verify.com",
			wantMinScore: 8,
			wantMaxScore: 20,
			wantSeverity: domain.SeverityHigh,
		},
		{
			name:         "number sequences add point",
			domain:       "bitcoin1234.com",
			wantMinScore: 4,
			wantMaxScore: 5,
			wantSeverity: domain.SeverityLow,
		},
		{
			name:         "domain with all heuristics triggered",
			domain:       "bitcoin-wallet-login-verify-exchange1234.xyz",
			wantMinScore: 10,
			wantMaxScore: 20,
			wantSeverity: domain.SeverityHigh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.Score(tt.domain, profile)

			assert.Equal(t, tt.domain, result.Domain)
			assert.GreaterOrEqual(t, result.Score, tt.wantMinScore,
				"score %d should be >= %d", result.Score, tt.wantMinScore)
			assert.LessOrEqual(t, result.Score, tt.wantMaxScore,
				"score %d should be <= %d", result.Score, tt.wantMaxScore)
			assert.Equal(t, tt.wantSeverity, result.Severity)

			if tt.wantNoKeywords {
				assert.Empty(t, result.MatchedKeywords)
			}
			if tt.wantKeywords != nil {
				for _, kw := range tt.wantKeywords {
					assert.Contains(t, result.MatchedKeywords, kw)
				}
			}
		})
	}
}

func TestEngine_Score_EmptyProfile(t *testing.T) {
	engine := NewEngine()
	profile := &domain.Profile{}

	// Structural heuristics are global, not per-profile: 2 hyphens (+1) + cheap-tier .xyz (+1).
	result := engine.Score("bitcoin-wallet-login.xyz", profile)
	assert.Equal(t, 2, result.Score)
	assert.Empty(t, result.MatchedKeywords)
	assert.Contains(t, result.Signals, SignalSuspiciousTLD)
	assert.Contains(t, result.Signals, SignalHyphens)
}

func TestEngine_Score_EmptyProfile_SimpleDomain(t *testing.T) {
	engine := NewEngine()
	profile := &domain.Profile{}

	result := engine.Score("example.com", profile)
	assert.Equal(t, 0, result.Score)
	assert.Empty(t, result.MatchedKeywords)
}

func TestEngine_Score_SkipSuffix_CaseInsensitive(t *testing.T) {
	engine := NewEngine()
	profile := testProfile()

	result := engine.Score("something.CLOUDFLARESSL.COM", profile)
	assert.Equal(t, 0, result.Score)
}

func TestClassifySeverity(t *testing.T) {
	tests := []struct {
		score    int
		severity domain.Severity
	}{
		{0, ""},
		{1, domain.SeverityLow},
		{2, domain.SeverityLow},
		{3, domain.SeverityLow},
		{4, domain.SeverityLow},
		{5, domain.SeverityMed},
		{6, domain.SeverityMed},
		{7, domain.SeverityMed},
		{8, domain.SeverityHigh},
		{10, domain.SeverityHigh},
		{100, domain.SeverityHigh},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.severity, classifySeverity(tt.score),
			"classifySeverity(%d)", tt.score)
	}
}
