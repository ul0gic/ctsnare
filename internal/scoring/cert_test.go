package scoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ul0gic/ctsnare/internal/domain"
)

func TestScoreCert(t *testing.T) {
	tests := []struct {
		name     string
		cert     domain.CertMeta
		brandHit bool
		want     int
	}{
		{"zero meta no score", domain.CertMeta{}, false, 0},
		{"large SAN set scores", domain.CertMeta{SANCount: 20}, false, 1},
		{"SAN set below threshold", domain.CertMeta{SANCount: 19}, false, 0},
		{"short-lived with brand scores", domain.CertMeta{ValidityDays: 90}, true, 1},
		{"short-lived without brand no score", domain.CertMeta{ValidityDays: 90}, false, 0},
		{"long-lived with brand no score", domain.CertMeta{ValidityDays: 365}, true, 0},
		{"zero validity ignored", domain.CertMeta{ValidityDays: 0}, true, 0},
		{
			name:     "both signals stack",
			cert:     domain.CertMeta{SANCount: 50, ValidityDays: 30},
			brandHit: true,
			want:     2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, scoreCert(tt.cert, tt.brandHit))
		})
	}
}

// TestScoreWithCert_MatchesScoreOnZeroMeta proves the additive contract: a zero
// CertMeta yields the same score as the cert-free Score method.
func TestScoreWithCert_MatchesScoreOnZeroMeta(t *testing.T) {
	engine := NewEngine()
	profile := testProfile()
	d := "bitcoin-wallet-login.xyz"

	plain := engine.Score(d, profile)
	withZero := engine.ScoreWithCert(d, profile, domain.CertMeta{})

	assert.Equal(t, plain.Score, withZero.Score)
	assert.Equal(t, plain.Severity, withZero.Severity)
	assert.Equal(t, plain.MatchedKeywords, withZero.MatchedKeywords)
}

// TestScoreWithCert_ShortLivedBrandBait verifies a brand hit on a short-lived,
// large-SAN cert picks up both certificate-level points.
func TestScoreWithCert_ShortLivedBrandBait(t *testing.T) {
	engine := NewEngine()
	profile := testProfile() // brand: bitcoin, paypal

	d := "bitcoin-login.com"
	plain := engine.Score(d, profile)
	withCert := engine.ScoreWithCert(d, profile, domain.CertMeta{SANCount: 30, ValidityDays: 60})

	assert.Equal(t, plain.Score+2, withCert.Score,
		"large SAN (+1) and short-lived brand bait (+1) should add 2 points")
}
