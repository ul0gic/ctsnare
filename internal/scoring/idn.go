package scoring

import (
	"strings"

	"golang.org/x/net/idna"
)

// Bonus for any "xn--" A-label: legitimate IDNs, but a strong homograph signal worth surfacing.
const punycodePoints = 2

// scorePunycode returns punycodePoints when any label is an "xn--" A-label.
func scorePunycode(domainName string) int {
	lower := strings.ToLower(domainName)
	for _, label := range strings.Split(lower, ".") {
		if strings.HasPrefix(label, "xn--") {
			return punycodePoints
		}
	}
	return 0
}

// confusableForm decodes punycode A-labels and folds homoglyphs to ASCII so keyword
// matching catches homograph attacks; hasPuny is false (and folded is the raw lower form) when there is no A-label.
func confusableForm(domainName string) (folded string, hasPuny bool) {
	lower := strings.ToLower(domainName)
	if !strings.Contains(lower, "xn--") {
		return lower, false
	}

	// On decode error fall back to the raw lower form so a malformed label cannot crash the hot path.
	decoded, err := idna.ToUnicode(lower)
	if err != nil {
		decoded = lower
	}

	return foldConfusables(decoded), true
}

// confusableMatch is a keyword that matched only after confusable folding.
type confusableMatch struct {
	keyword string
	points  int
}

// scoreConfusableKeywords returns keywords matching the confusable-folded form but
// not the raw ASCII (so a homograph hit counts once), prefixed "*" to flag folding.
func scoreConfusableKeywords(domainName string, brand, generic []string) []confusableMatch {
	folded, hasPuny := confusableForm(domainName)
	if !hasPuny {
		return nil
	}

	rawLower := strings.ToLower(domainName)
	var matches []confusableMatch

	for _, kw := range brand {
		matches = appendConfusableMatch(matches, folded, rawLower, kw, brandKeywordPoints)
	}
	for _, kw := range generic {
		matches = appendConfusableMatch(matches, folded, rawLower, kw, genericKeywordPoints)
	}
	return matches
}

// appendConfusableMatch credits a keyword only when it appears in folded but not raw.
func appendConfusableMatch(matches []confusableMatch, folded, raw, kw string, points int) []confusableMatch {
	lkw := strings.ToLower(kw)
	if strings.Contains(folded, lkw) && !strings.Contains(raw, lkw) {
		matches = append(matches, confusableMatch{keyword: "*" + kw, points: points})
	}
	return matches
}
