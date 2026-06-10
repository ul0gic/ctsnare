// Package scoring implements domain scoring heuristics.
package scoring

import (
	"strings"

	"github.com/ul0gic/ctsnare/internal/domain"
)

// Engine scores domains against keyword profiles, satisfying domain.Scorer.
type Engine struct{}

// NewEngine creates a new scoring engine.
func NewEngine() *Engine {
	return &Engine{}
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
	// Check skip suffixes first -- infrastructure domains generate noise.
	for _, suffix := range profile.SkipSuffixes {
		if strings.HasSuffix(strings.ToLower(domainName), strings.ToLower(suffix)) {
			return domain.ScoredDomain{
				Domain:          domainName,
				Score:           0,
				Severity:        "",
				MatchedKeywords: nil,
			}
		}
	}

	totalScore := 0

	// Brand tier (+3 each) is the high-precision signal; generic tier (+1 each)
	// is broad and noise-prone. Both feed the multi-keyword bonus.
	brandScore, brandMatched := matchKeywords(domainName, profile.BrandKeywords, brandKeywordPoints)
	genScore, genMatched := matchKeywords(domainName, profile.Keywords, genericKeywordPoints)
	totalScore += brandScore + genScore

	matched := make([]string, 0, len(brandMatched)+len(genMatched))
	matched = append(matched, brandMatched...)
	matched = append(matched, genMatched...)

	// Punycode / homograph: flag IDN labels, then re-match keywords against the
	// confusable-folded Unicode form to catch look-alike impersonation that the
	// literal ASCII scan above misses.
	totalScore += scorePunycode(domainName)
	confScore, confMatched := scoreConfusableKeywords(domainName, profile.BrandKeywords, profile.Keywords)
	totalScore += confScore
	matched = append(matched, confMatched...)

	// Typosquat: brand near-misses one or two edits away (e.g. "paypol",
	// "metmask") that the literal substring scan cannot catch.
	typoScore, typoMatched := scoreTyposquat(domainName, profile.BrandKeywords)
	totalScore += typoScore
	matched = append(matched, typoMatched...)

	totalScore += scoreTLD(domainName, profile.SuspiciousTLDs)
	totalScore += scoreDomainLength(domainName)
	totalScore += scoreHyphenDensity(domainName)
	totalScore += scoreNumberSequences(domainName)
	totalScore += scoreDGA(domainName)
	totalScore += scoreMultiKeywordBonus(len(matched))

	// Certificate-level heuristics. A brand hit here means any literal, confusable,
	// or typosquat brand match — all carry a brand-impersonation signal.
	brandHit := len(brandMatched) > 0 || len(confMatched) > 0 || len(typoMatched) > 0
	totalScore += scoreCert(cert, brandHit)

	severity := classifySeverity(totalScore)

	if len(matched) == 0 {
		matched = nil
	}

	return domain.ScoredDomain{
		Domain:          domainName,
		Score:           totalScore,
		Severity:        severity,
		MatchedKeywords: matched,
	}
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
