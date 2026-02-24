package scoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ul0gic/ctsnare/internal/domain"
)

func testProfile() *domain.Profile {
	return &domain.Profile{
		Name:     "test",
		Keywords: []string{"bitcoin", "login", "wallet", "exchange", "verify"},
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
			name:         "single keyword match scores LOW",
			domain:       "bitcoin-news.com",
			wantMinScore: 2,
			wantMaxScore: 3,
			wantSeverity: domain.SeverityLow,
			wantKeywords: []string{"bitcoin"},
		},
		{
			name:         "two keyword matches scores MED",
			domain:       "bitcoin-wallet.com",
			wantMinScore: 4,
			wantMaxScore: 5,
			wantSeverity: domain.SeverityMed,
			wantKeywords: []string{"bitcoin", "wallet"},
		},
		{
			name:         "three keywords with bonus scores HIGH",
			domain:       "bitcoin-wallet-login.xyz",
			wantMinScore: 6,
			wantMaxScore: 20,
			wantSeverity: domain.SeverityHigh,
			wantKeywords: []string{"bitcoin", "login", "wallet"},
		},
		{
			name:         "suspicious TLD adds point",
			domain:       "bitcoin-shop.xyz",
			wantMinScore: 3,
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
			wantMaxScore: 5,
			wantSeverity: domain.SeverityMed,
			wantKeywords: []string{"bitcoin", "wallet"},
		},
		{
			name:         "long domain adds point",
			domain:       "this-is-a-very-long-bitcoin-domain-name.com",
			wantMinScore: 4,
			wantMaxScore: 6,
			wantSeverity: domain.SeverityMed,
		},
		{
			name:         "hyphen-heavy domain adds point",
			domain:       "bitcoin-secure-login-verify.com",
			wantMinScore: 6,
			wantMaxScore: 20,
			wantSeverity: domain.SeverityHigh,
		},
		{
			name:         "number sequences add point",
			domain:       "bitcoin1234.com",
			wantMinScore: 3,
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

	// With empty profile, no keywords match but structural heuristics
	// (hyphen density) still apply: "bitcoin-wallet-login" has 2 hyphens.
	result := engine.Score("bitcoin-wallet-login.xyz", profile)
	assert.Equal(t, 1, result.Score)
	assert.Empty(t, result.MatchedKeywords)
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
		{4, domain.SeverityMed},
		{5, domain.SeverityMed},
		{6, domain.SeverityHigh},
		{10, domain.SeverityHigh},
		{100, domain.SeverityHigh},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.severity, classifySeverity(tt.score),
			"classifySeverity(%d)", tt.score)
	}
}
