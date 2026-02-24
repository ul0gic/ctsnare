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
	kwScore, matched := matchKeywords(domainName, profile.Keywords)
	totalScore += kwScore
	totalScore += scoreTLD(domainName, profile.SuspiciousTLDs)
	totalScore += scoreDomainLength(domainName)
	totalScore += scoreHyphenDensity(domainName)
	totalScore += scoreNumberSequences(domainName)
	totalScore += scoreMultiKeywordBonus(len(matched))

	severity := classifySeverity(totalScore)

	return domain.ScoredDomain{
		Domain:          domainName,
		Score:           totalScore,
		Severity:        severity,
		MatchedKeywords: matched,
	}
}

// classifySeverity maps a numeric score to a severity level.
// HIGH >= 6, MED 4-5, LOW 1-3, empty string for 0.
func classifySeverity(score int) domain.Severity {
	switch {
	case score >= 6:
		return domain.SeverityHigh
	case score >= 4:
		return domain.SeverityMed
	case score >= 1:
		return domain.SeverityLow
	default:
		return ""
	}
}
