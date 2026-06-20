// Package scoring implements domain scoring heuristics.
package scoring

import (
	"strings"

	"github.com/ul0gic/ctsnare/internal/config"
	"github.com/ul0gic/ctsnare/internal/domain"
)

// Config holds the engine's data-driven inputs outside per-profile keywords. A
// zero Config is valid: empty fields fall back to built-in defaults.
type Config struct {
	BurnerTLDs     []string
	CheapTLDs      []string
	WatchPlatforms []string
	// CategoryKeywords maps a lowercased keyword to its profile category; nil disables categorization.
	CategoryKeywords map[string]string
}

// Engine scores domains against keyword profiles, satisfying domain.Scorer.
type Engine struct {
	burnerTLDs       []string
	cheapTLDs        []string
	watchPlatforms   []string
	categoryKeywords map[string]string
}

// NewEngine creates a scoring engine; pass zero or one Config (omitting it yields built-in defaults).
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

// Score runs all non-certificate heuristics; callers with cert context use ScoreWithCert.
func (e *Engine) Score(domainName string, profile *domain.Profile) domain.ScoredDomain {
	return e.ScoreWithCert(domainName, profile, domain.CertMeta{})
}

// ScoreWithCert runs Score's heuristics plus cert-level ones; a zero CertMeta matches Score exactly.
func (e *Engine) ScoreWithCert(domainName string, profile *domain.Profile, cert domain.CertMeta) domain.ScoredDomain {
	lower := strings.ToLower(domainName)

	if platform := e.matchWatchPlatform(lower); platform != "" {
		return e.scorePlatformTenant(domainName, lower, platform, profile, cert)
	}

	// Skip suffixes first: infrastructure domains generate noise.
	for _, suffix := range profile.SkipSuffixes {
		if strings.HasSuffix(lower, strings.ToLower(suffix)) {
			return domain.ScoredDomain{Domain: domainName}
		}
	}

	acc := e.scoreCore(domainName, profile, cert)
	return acc.result(domainName)
}

// accumulator collects score, matched keywords, fired signals, and strongest category as heuristics run.
type accumulator struct {
	engine    *Engine
	score     int
	matched   []string
	signals   []string
	category  string
	catWeight int
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

// addKeyword records a matched keyword, its points and signal, and updates category attribution.
func (a *accumulator) addKeyword(kw string, points int, signal string) {
	a.score += points
	a.matched = append(a.matched, kw)
	if signal != "" && !contains(a.signals, signal) {
		a.signals = append(a.signals, signal)
	}
	a.attribute(kw, points)
}

// attribute updates the strongest category when kw outweighs the best so far;
// "*"/"~" fuzzy prefixes are stripped before lookup so homoglyph/typosquat hits attribute.
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

// scoreCore runs the full heuristic suite against domainName and returns the accumulator.
func (e *Engine) scoreCore(domainName string, profile *domain.Profile, cert domain.CertMeta) *accumulator {
	acc := e.newAccumulator()
	e.scoreKeywordTiers(acc, domainName, profile)
	e.scoreStructural(acc, domainName, profile)

	// Any literal, confusable, or typosquat brand match counts as a brand hit.
	brandHit := contains(acc.signals, SignalBrandKeyword) ||
		contains(acc.signals, SignalHomoglyph) ||
		contains(acc.signals, SignalTyposquat)
	e.scoreCertSignals(acc, cert, brandHit)

	acc.addSignal(scoreMultiKeywordBonus(len(acc.matched)), SignalMultiKeyword)
	return acc
}

// scoreKeywordTiers runs literal, confusable, and typosquat keyword matching.
func (e *Engine) scoreKeywordTiers(acc *accumulator, domainName string, profile *domain.Profile) {
	for _, kw := range matchKeywords(domainName, profile.BrandKeywords, brandKeywordPoints) {
		acc.addKeyword(kw, brandKeywordPoints, SignalBrandKeyword)
	}
	for _, kw := range matchKeywords(domainName, profile.Keywords, genericKeywordPoints) {
		acc.addKeyword(kw, genericKeywordPoints, SignalGenericKeyword)
	}

	acc.addSignal(scorePunycode(domainName), SignalPunycode)
	for _, m := range scoreConfusableKeywords(domainName, profile.BrandKeywords, profile.Keywords) {
		acc.addKeyword(m.keyword, m.points, SignalHomoglyph)
	}

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

// Brand keywords are high-precision; generic keywords gain signal in combination.
const (
	brandKeywordPoints   = 3
	genericKeywordPoints = 1
)

// classifySeverity maps a score to severity: HIGH >= 8, MED 5-7, LOW 1-4, "" for 0.
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

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
