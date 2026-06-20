package scoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ul0gic/ctsnare/internal/config"
)

// TestTLDTierConstantsMatchConfig asserts the local tier weights stay in
// lockstep with the config package so the two definitions cannot silently diverge.
func TestTLDTierConstantsMatchConfig(t *testing.T) {
	assert.Equal(t, config.BurnerTLDPoints, tldBurnerPoints)
	assert.Equal(t, config.CheapTLDPoints, tldCheapPoints)
}

func TestScoreNumericSLD(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		tier   string
		want   int
	}{
		{"all-digit sld on burner tld scores", "12345.tk", SignalBurnerTLD, numericSLDPoints},
		{"all-digit sld on cheap tld scores", "8888.xyz", SignalSuspiciousTLD, numericSLDPoints},
		{"all-digit sld on neutral tld no score", "12306.cn", "", 0},
		{"short numeric sld below min length no score", "12.tk", SignalBurnerTLD, 0},
		{"mixed alphanumeric sld no score", "12abc.tk", SignalBurnerTLD, 0},
		{"sld with subdomain numeric still scores", "login.4006.xin", SignalBurnerTLD, numericSLDPoints},
		{"alpha sld no score", "paypal.tk", SignalBurnerTLD, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, scoreNumericSLD(tt.domain, tt.tier))
		})
	}
}

// TestScoreNumericSLD_FalsePositiveGuard proves a legitimate-shaped numeric SLD
// on a neutral TLD (the canonical 12306.cn rail-ticket site) never fires.
func TestScoreNumericSLD_FalsePositiveGuard(t *testing.T) {
	assert.Equal(t, 0, scoreNumericSLD("12306.cn", ""))
	assert.Equal(t, 0, scoreNumericSLD("163.com", ""))
}

func TestScoreDeceptivePrefix(t *testing.T) {
	tests := []struct {
		name       string
		domain     string
		keywordHit bool
		tieredTLD  bool
		brandHit   bool
		want       int
	}{
		{"com- prefix with tld scores", "com-tollpay.xin", false, true, false, deceptivePrefixPoints},
		{"-com suffix on sld with keyword scores", "paypal-com.icu", true, false, true, deceptivePrefixPoints},
		{"www- prefix with brand scores", "www-paypal.xyz", false, false, true, deceptivePrefixPoints},
		{"deceptive shape but no corroboration no score", "com-something.com", false, false, false, 0},
		{"plain domain no score", "community.example.com", true, false, false, 0},
		{"legitimate www subdomain no score", "www.example.com", true, false, false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scoreDeceptivePrefix(tt.domain, tt.keywordHit, tt.tieredTLD, tt.brandHit)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestScoreDeceptivePrefix_FalsePositiveGuard proves benign "com"-containing
// labels do not trip the heuristic even when corroboration is present.
func TestScoreDeceptivePrefix_FalsePositiveGuard(t *testing.T) {
	// "community" starts with "com" but not "com-"; must not fire.
	assert.Equal(t, 0, scoreDeceptivePrefix("community.example.com", true, true, true))
	// "telecom" ends with "com" but not "-com"; must not fire.
	assert.Equal(t, 0, scoreDeceptivePrefix("telecom.example.org", true, true, true))
}

func TestRegisteredSLD(t *testing.T) {
	tests := []struct {
		domain string
		want   string
	}{
		{"paypal-com.icu", "paypal-com"},
		{"login.paypal.xyz", "paypal"},
		{"example.com", "example"},
		{"localhost", "localhost"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, registeredSLD(tt.domain), tt.domain)
	}
}
