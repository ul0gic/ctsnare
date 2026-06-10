// Package scoring implements domain scoring heuristics.
package scoring

import (
	"strings"

	"github.com/ul0gic/ctsnare/internal/config"
	"github.com/ul0gic/ctsnare/internal/domain"
)

// Config carries the data-driven inputs the engine needs that live outside the
// per-profile keyword lists: the two TLD tiers and the watched free-hosting
// platforms. A zero Config is valid — the engine falls back to built-in tier
// defaults and an empty platform list — so tests can call NewEngine() with no
// arguments.
type Config struct {
	// BurnerTLDs is the resolved burner tier (+6). Empty falls back to defaults.
	BurnerTLDs []string
	// CheapTLDs is the resolved cheap tier (+1). Empty falls back to defaults.
	CheapTLDs []string
	// WatchPlatforms are free-hosting suffixes whose tenants are scored rather
	// than skipped (Package B). Empty disables platform scoring.
	WatchPlatforms []string
	// CategoryKeywords maps a lowercased keyword to its profile category
	// ("crypto", "phishing", "ai"). Used to attribute a hit's strongest match.
	// Nil disables categorization.
	CategoryKeywords map[string]string
}

// Engine scores domains against keyword profiles, satisfying domain.Scorer.
type Engine struct {
	burnerTLDs       []string
	cheapTLDs        []string
	watchPlatforms   []string
	categoryKeywords map[string]string
}

// NewEngine creates a scoring engine. Pass zero or one Config: omitting it
// (NewEngine()) yields an engine with built-in TLD tier defaults, no watched
// platforms, and no category attribution — adequate for unit tests. Production
// callers pass a populated Config built from the loaded configuration.
func NewEngine(cfg ...Config) *Engine {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	burner, cheap := config.ResolveTLDTiers(config.TLDTiers{Burner: c.BurnerTLDs, Cheap: c.CheapTLDs})
	return &Engine{
		burnerTLDs:       burner,
		cheapTLDs:        cheap,
		watchPlatforms:   c.WatchPlatforms,
		categoryKeywords: c.CategoryKeywords,
	}
}

// Score runs all heuristics against the domain using the given profile and
// returns a ScoredDomain. Domains matching a skip suffix are immediately
// returned with a zero score. Certificate-level heuristics are skipped; callers
// with certificate context should use ScoreWithCert.
func (e *Engine) Score(domainName string, profile *domain.Profile) domain.ScoredDomain {
	return e.ScoreWithCert(domainName, profile, domain.CertMeta{})
}

// ScoreWithCert runs all of Score's heuristics plus the certificate-level
// heuristics driven by cert. A zero CertMeta disables the cert heuristics, so
// the result matches Score exactly.
func (e *Engine) ScoreWithCert(domainName string, profile *domain.Profile, cert domain.CertMeta) domain.ScoredDomain {
	lower := strings.ToLower(domainName)

	// Watched free-hosting platforms get special treatment (Package B): the
	// tenant labels are scored in isolation and the domain is stored only if a
	// brand/typosquat/homoglyph signal fires on the tenant. Non-platform
	// domains fall through to the normal scoring path.
	if platform := e.matchWatchPlatform(lower); platform != "" {
		return e.scorePlatformTenant(domainName, lower, platform, profile, cert)
	}

	// Check skip suffixes first -- infrastructure domains generate noise.
	for _, suffix := range profile.SkipSuffixes {
		if strings.HasSuffix(lower, strings.ToLower(suffix)) {
			return domain.ScoredDomain{Domain: domainName}
		}
	}

	acc := e.scoreCore(domainName, profile, cert)
	return acc.result(domainName)
}

// accumulator collects the running score, matched keywords, fired signals, and
// the strongest category as heuristics run. It centralizes the bookkeeping so
// each heuristic stays a pure function returning (points, signal).
type accumulator struct {
	engine    *Engine
	score     int
	matched   []string
	signals   []string
	category  string
	catWeight int // points of the strongest categorized match so far
}

func (e *Engine) newAccumulator() *accumulator {
	return &accumulator{engine: e}
}

// addSignal records a signal key once (deduplicated) and adds its points.
func (a *accumulator) addSignal(points int, signal string) {
	if points == 0 {
		return
	}
	a.score += points
	if signal != "" && !contains(a.signals, signal) {
		a.signals = append(a.signals, signal)
	}
}

// addKeyword records a matched keyword, its points, the signal key, and updates
// the strongest-category attribution from the keyword's category (if any).
func (a *accumulator) addKeyword(kw string, points int, signal string) {
	a.score += points
	a.matched = append(a.matched, kw)
	if signal != "" && !contains(a.signals, signal) {
		a.signals = append(a.signals, signal)
	}
	a.attribute(kw, points)
}

// attribute updates the strongest category if kw maps to a category and its
// weight exceeds the best seen so far. The "*"/"~" fuzzy-match prefixes are
// stripped before lookup so homoglyph/typosquat hits still attribute.
func (a *accumulator) attribute(kw string, points int) {
	if a.engine.categoryKeywords == nil {
		return
	}
	key := strings.ToLower(strings.TrimLeft(kw, "*~"))
	cat, ok := a.engine.categoryKeywords[key]
	if !ok {
		return
	}
	if points > a.catWeight {
		a.catWeight = points
		a.category = cat
	}
}

// result finalizes the accumulator into a ScoredDomain.
func (a *accumulator) result(domainName string) domain.ScoredDomain {
	matched := a.matched
	if len(matched) == 0 {
		matched = nil
	}
	signals := a.signals
	if len(signals) == 0 {
		signals = nil
	}
	return domain.ScoredDomain{
		Domain:          domainName,
		Score:           a.score,
		Severity:        classifySeverity(a.score),
		MatchedKeywords: matched,
		Signals:         signals,
		Category:        a.category,
	}
}

// scoreCore runs the full heuristic suite against domainName and returns the
// populated accumulator. Shared by the normal path and (indirectly) reused
// logic for platform tenants.
func (e *Engine) scoreCore(domainName string, profile *domain.Profile, cert domain.CertMeta) *accumulator {
	acc := e.newAccumulator()
	e.scoreKeywordTiers(acc, domainName, profile)
	e.scoreStructural(acc, domainName, profile)

	// Certificate-level heuristics. A brand hit means any literal, confusable,
	// or typosquat brand match — all carry a brand-impersonation signal.
	brandHit := contains(acc.signals, SignalBrandKeyword) ||
		contains(acc.signals, SignalHomoglyph) ||
		contains(acc.signals, SignalTyposquat)
	e.scoreCertSignals(acc, cert, brandHit)

	acc.addSignal(scoreMultiKeywordBonus(len(acc.matched)), SignalMultiKeyword)
	return acc
}

// scoreKeywordTiers runs literal, confusable, and typosquat keyword matching.
func (e *Engine) scoreKeywordTiers(acc *accumulator, domainName string, profile *domain.Profile) {
	// Brand tier (+3) is high-precision; generic tier (+1) is broad.
	for _, kw := range matchKeywords(domainName, profile.BrandKeywords, brandKeywordPoints) {
		acc.addKeyword(kw, brandKeywordPoints, SignalBrandKeyword)
	}
	for _, kw := range matchKeywords(domainName, profile.Keywords, genericKeywordPoints) {
		acc.addKeyword(kw, genericKeywordPoints, SignalGenericKeyword)
	}

	// Punycode / homograph: flag IDN labels, then re-match keywords against the
	// confusable-folded Unicode form to catch look-alike impersonation.
	acc.addSignal(scorePunycode(domainName), SignalPunycode)
	for _, m := range scoreConfusableKeywords(domainName, profile.BrandKeywords, profile.Keywords) {
		acc.addKeyword(m.keyword, m.points, SignalHomoglyph)
	}

	// Typosquat: brand near-misses one or two edits away.
	for _, kw := range scoreTyposquat(domainName, profile.BrandKeywords) {
		acc.addKeyword(kw, typosquatPoints, SignalTyposquat)
	}
}

// scoreStructural runs the TLD-tier and domain-shape heuristics.
func (e *Engine) scoreStructural(acc *accumulator, domainName string, profile *domain.Profile) {
	points, signal := e.scoreTLDTier(domainName, profile.SuspiciousTLDs)
	acc.addSignal(points, signal)

	tier := signal // "" | suspicious-tld | burner-tld
	acc.addSignal(scoreNumericSLD(domainName, tier), SignalNumericSLD)

	brandHit := contains(acc.signals, SignalBrandKeyword) ||
		contains(acc.signals, SignalHomoglyph) ||
		contains(acc.signals, SignalTyposquat)
	keywordHit := len(acc.matched) > 0
	acc.addSignal(scoreDeceptivePrefix(domainName, keywordHit, tier != "", brandHit), SignalDeceptivePrefix)

	acc.addSignal(scoreDomainLength(domainName), SignalLongDomain)
	acc.addSignal(scoreHyphenDensity(domainName), SignalHyphens)
	acc.addSignal(scoreNumberSequences(domainName), SignalDigitSeq)
	acc.addSignal(scoreDGA(domainName), SignalEntropy)
}

// Keyword tier weights. Brand keywords are high-precision impersonation
// targets; generic keywords are broad terms that gain signal in combination.
const (
	brandKeywordPoints   = 3
	genericKeywordPoints = 1
)

// classifySeverity maps a numeric score to a severity level.
// HIGH >= 8, MED 5-7, LOW 1-4, empty string for 0.
func classifySeverity(score int) domain.Severity {
	switch {
	case score >= 8:
		return domain.SeverityHigh
	case score >= 5:
		return domain.SeverityMed
	case score >= 1:
		return domain.SeverityLow
	default:
		return ""
	}
}

// contains reports whether s is present in slice.
func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
