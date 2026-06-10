package scoring

import "strings"

// confusableMap folds common homoglyphs to their ASCII look-alike. It covers the
// digit/letter swaps used in leetspeak typosquats and the Cyrillic/Greek
// look-alikes used in IDN homograph attacks. It is deliberately small and
// curated — a full Unicode confusables table is overkill for keyword folding and
// would risk over-folding legitimate text.
var confusableMap = map[rune]rune{
	// Leetspeak digit -> letter
	'0': 'o',
	'1': 'l',
	'3': 'e',
	'4': 'a',
	'5': 's',
	'7': 't',
	// Cyrillic look-alikes -> Latin
	'а': 'a', // U+0430
	'е': 'e', // U+0435
	'о': 'o', // U+043E
	'р': 'p', // U+0440
	'с': 'c', // U+0441
	'х': 'x', // U+0445
	'у': 'y', // U+0443
	'ѕ': 's', // U+0455
	'і': 'i', // U+0456
	'ј': 'j', // U+0458
	'ԁ': 'd', // U+0501
	// Greek look-alikes -> Latin
	'ο': 'o', // U+03BF
	'α': 'a', // U+03B1
	'ρ': 'p', // U+03C1
	'ν': 'v', // U+03BD
	'τ': 't', // U+03C4
	// Latin-1 / accented look-alikes -> base letter
	'à': 'a', 'á': 'a', 'â': 'a', 'ä': 'a', 'å': 'a',
	'è': 'e', 'é': 'e', 'ê': 'e', 'ë': 'e',
	'ì': 'i', 'í': 'i', 'î': 'i', 'ï': 'i',
	'ò': 'o', 'ó': 'o', 'ô': 'o', 'ö': 'o', 'ø': 'o',
	'ù': 'u', 'ú': 'u', 'û': 'u', 'ü': 'u',
	'ñ': 'n', 'ç': 'c',
}

// foldConfusables maps each rune of s through confusableMap, leaving runes with
// no entry unchanged. Used to normalize a decoded IDN before keyword matching so
// that homoglyph substitutions collapse onto their ASCII targets.
func foldConfusables(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if folded, ok := confusableMap[r]; ok {
			b.WriteRune(folded)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
