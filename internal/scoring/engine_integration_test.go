package scoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ul0gic/ctsnare/internal/domain"
)

// integrationProfile mirrors a realistic phishing/crypto profile for end-to-end
// scoring assertions.
func integrationProfile() *domain.Profile {
	return &domain.Profile{
		Name:          "test",
		BrandKeywords: []string{"paypal", "coinbase", "bitcoin"},
		Keywords:      []string{"login", "wallet", "verify"},
		SkipSuffixes:  []string{"amazonaws.com"},
	}
}

// integrationEngine builds an engine wired with the production category map and
// the watch platforms, using default TLD tiers.
func integrationEngine() *Engine {
	return NewEngine(Config{
		WatchPlatforms: []string{"pages.dev", "netlify.app"},
		CategoryKeywords: map[string]string{
			"paypal": CategoryPhishing, "login": CategoryPhishing,
			"coinbase": CategoryCrypto, "bitcoin": CategoryCrypto, "wallet": CategoryCrypto,
		},
	})
}

func TestEngine_SignalsAndCategory(t *testing.T) {
	engine := integrationEngine()
	profile := integrationProfile()

	t.Run("brand keyword emits signal and category", func(t *testing.T) {
		r := engine.Score("paypal-login.com", profile)
		assert.Contains(t, r.Signals, SignalBrandKeyword)
		assert.Contains(t, r.Signals, SignalGenericKeyword)
		assert.Equal(t, CategoryPhishing, r.Category, "brand (+3) outweighs generic (+1)")
	})

	t.Run("burner tld alone reaches MED and is auto-stored", func(t *testing.T) {
		r := engine.Score("randomthing.tk", profile)
		assert.Equal(t, 6, r.Score)
		assert.Equal(t, domain.SeverityMed, r.Severity)
		assert.Contains(t, r.Signals, SignalBurnerTLD)
	})

	t.Run("burner tld plus numeric sld is HIGH on its own", func(t *testing.T) {
		r := engine.Score("4006.xin", profile)
		// burner(+6) + numeric-sld(+3) + 4-digit digit-seq(+1) = 10 (>= 8 = HIGH).
		assert.Equal(t, 10, r.Score)
		assert.Equal(t, domain.SeverityHigh, r.Severity)
		assert.Contains(t, r.Signals, SignalBurnerTLD)
		assert.Contains(t, r.Signals, SignalNumericSLD)
	})

	t.Run("3-digit numeric sld on burner is HIGH without digit-seq", func(t *testing.T) {
		r := engine.Score("400.xin", profile)
		// burner(+6) + numeric-sld(+3) = 9; only 3 digits so no digit-seq.
		assert.Equal(t, 9, r.Score)
		assert.Equal(t, domain.SeverityHigh, r.Severity)
		assert.NotContains(t, r.Signals, SignalDigitSeq)
	})

	t.Run("deceptive prefix fires with tiered tld", func(t *testing.T) {
		r := engine.Score("com-tollpay.xin", profile)
		assert.Contains(t, r.Signals, SignalDeceptivePrefix)
		assert.Contains(t, r.Signals, SignalBurnerTLD)
	})

	t.Run("free CA brand cert signal", func(t *testing.T) {
		r := engine.ScoreWithCert("paypal-login.com", profile,
			domain.CertMeta{Issuer: "Let's Encrypt", ValidityDays: 60})
		// short-lived(+1) + free-ca(+1) capped at +2, both recorded.
		assert.Contains(t, r.Signals, SignalShortLivedBrand)
		assert.Contains(t, r.Signals, SignalFreeCABrand)
	})

	t.Run("numeric sld on neutral tld does not fire", func(t *testing.T) {
		r := engine.Score("12306.cn", profile)
		assert.NotContains(t, r.Signals, SignalNumericSLD)
	})
}

func TestEngine_WatchPlatform(t *testing.T) {
	engine := integrationEngine()
	profile := integrationProfile()

	t.Run("brand on platform stores with hosted-abuse", func(t *testing.T) {
		r := engine.Score("paypal-login.pages.dev", profile)
		// tenant "paypal-login": paypal(+3) + login(+1) + hosted-abuse(+2) = 6.
		assert.GreaterOrEqual(t, r.Score, 5)
		assert.Equal(t, domain.SeverityMed, r.Severity)
		assert.Contains(t, r.Signals, SignalHostedAbuse)
		assert.Contains(t, r.Signals, SignalBrandKeyword)
		assert.Equal(t, CategoryHostedAbuse, r.Category)
		assert.Equal(t, "paypal-login.pages.dev", r.Domain, "full domain reported, not tenant")
	})

	t.Run("typosquat on platform fires", func(t *testing.T) {
		r := engine.Score("coinbasse.netlify.app", profile)
		assert.Contains(t, r.Signals, SignalTyposquat)
		assert.Contains(t, r.Signals, SignalHostedAbuse)
		assert.Equal(t, CategoryHostedAbuse, r.Category)
	})

	t.Run("benign tenant discarded", func(t *testing.T) {
		r := engine.Score("my-cool-blog.pages.dev", profile)
		assert.Equal(t, 0, r.Score)
		assert.Empty(t, r.Signals)
		assert.Empty(t, r.Severity)
	})

	t.Run("platform apex never stored", func(t *testing.T) {
		r := engine.Score("pages.dev", profile)
		assert.Equal(t, 0, r.Score)
		assert.Empty(t, r.Signals)
	})
}
