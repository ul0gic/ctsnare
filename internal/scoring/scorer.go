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
// returned with a zero score.
func (e *Engine) Score(domainName string, profile *domain.Profile) domain.ScoredDomain {
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

	totalScore += scoreTLD(domainName, profile.SuspiciousTLDs)
	totalScore += scoreDomainLength(domainName)
	totalScore += scoreHyphenDensity(domainName)
	totalScore += scoreNumberSequences(domainName)
	totalScore += scoreMultiKeywordBonus(len(matched))

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
