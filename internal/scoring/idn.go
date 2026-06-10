package scoring

import (
	"strings"

	"golang.org/x/net/idna"
)

// punycodePoints is the bonus for any domain carrying an "xn--" (A-label) part.
// Internationalized domains are legitimate, but in a phishing-hunting context an
// IDN is a strong homograph-attack signal worth surfacing on its own.
const punycodePoints = 2

// scorePunycode returns punycodePoints if any dot-separated label of the domain
// is an "xn--" A-label (an IDN / punycode label). Matching is case-insensitive.
func scorePunycode(domainName string) int {
	lower := strings.ToLower(domainName)
	for _, label := range strings.Split(lower, ".") {
		if strings.HasPrefix(label, "xn--") {
			return punycodePoints
		}
	}
	return 0
}

// confusableForm decodes punycode A-labels to their Unicode form and folds common
// homoglyphs to their ASCII look-alikes, so keyword matching can catch homograph
// attacks such as xn--pypal-4ve.com (pаypal with a Cyrillic 'а') matching "paypal".
//
// It returns the folded ASCII-ish form, or the original (lower-cased) string when
// the domain contains no punycode label — callers should skip re-matching in that
// case to avoid double-counting plain-ASCII keyword hits.
func confusableForm(domainName string) (folded string, hasPuny bool) {
	lower := strings.ToLower(domainName)
	if !strings.Contains(lower, "xn--") {
		return lower, false
	}

	// idna.ToUnicode decodes every A-label in the name; on error fall back to the
	// raw lower-cased form so a malformed label cannot crash the hot path.
	decoded, err := idna.ToUnicode(lower)
	if err != nil {
		decoded = lower
	}

	return foldConfusables(decoded), true
}

// scoreConfusableKeywords decodes a punycode domain and runs keyword matching
// against the confusable-folded Unicode form. It returns the score and any
// newly matched keywords that were NOT already present in the raw ASCII form,
// so a homograph hit is counted once. Matches are reported with a "*" prefix to
// flag that they came from the decoded/folded form rather than literal text.
func scoreConfusableKeywords(domainName string, brand, generic []string) (score int, matched []string) {
	folded, hasPuny := confusableForm(domainName)
	if !hasPuny {
		return 0, nil
	}

	rawLower := strings.ToLower(domainName)

	for _, kw := range brand {
		score, matched = addConfusableMatch(folded, rawLower, kw, brandKeywordPoints, score, matched)
	}
	for _, kw := range generic {
		score, matched = addConfusableMatch(folded, rawLower, kw, genericKeywordPoints, score, matched)
	}
	return score, matched
}

// addConfusableMatch credits a keyword only when it appears in the folded form
// but not in the raw form (i.e. it was hidden behind a homoglyph or A-label).
func addConfusableMatch(folded, raw, kw string, points, score int, matched []string) (int, []string) {
	lkw := strings.ToLower(kw)
	if strings.Contains(folded, lkw) && !strings.Contains(raw, lkw) {
		score += points
		matched = append(matched, "*"+kw)
	}
	return score, matched
}
