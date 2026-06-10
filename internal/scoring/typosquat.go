package scoring

import "strings"

// typosquatPoints is the bonus for a domain label that is a near-miss of a brand
// keyword (one or two edits away) without being an exact substring match.
const typosquatPoints = 3

// Edit-distance acceptance windows. Short brands tolerate only a single edit to
// keep false positives down ("paypol" -> paypal, but not "papal"); longer brands
// can absorb two edits and still be an obvious squat ("anthropics" patterns).
const (
	typoMinLenDist1 = 5 // brand length >= 5 accepts distance 1
	typoMinLenDist2 = 8 // brand length >= 8 accepts distance <= 2
)

// scoreTyposquat checks each registered-domain label against the brand keywords
// for a Damerau-Levenshtein near match. A label that is distance 1 from a brand
// (brand length >= 5) or distance <= 2 (brand length >= 8) — and is not already
// an exact substring of the domain — scores typosquatPoints and is reported as a
// matched keyword with a "~" prefix so the user sees it was fuzzy.
//
// This runs on every domain from the CT firehose, so it is allocation-light: it
// scans only the dot-separated labels of the registered part, applies a cheap
// length-window pre-filter before computing any distance, and short-circuits as
// soon as a brand produces a hit.
func scoreTyposquat(domainName string, brand []string) (matched []string) {
	if len(brand) == 0 {
		return nil
	}

	lower := strings.ToLower(domainName)
	registered := registeredPart(lower)

	for _, kw := range brand {
		lkw := strings.ToLower(kw)
		// Skip brands too short to fuzzy-match safely.
		if len(lkw) < typoMinLenDist1 {
			continue
		}
		// An exact substring is a literal keyword hit, already scored elsewhere.
		if strings.Contains(lower, lkw) {
			continue
		}

		maxDist := allowedDistance(len(lkw))
		if maxDist == 0 {
			continue
		}

		if labelNearBrand(registered, lkw, maxDist) {
			matched = append(matched, "~"+kw)
		}
	}
	return matched
}

// allowedDistance returns the maximum edit distance accepted for a brand of the
// given length, or 0 if the brand is too short to fuzzy-match.
func allowedDistance(brandLen int) int {
	switch {
	case brandLen >= typoMinLenDist2:
		return 2
	case brandLen >= typoMinLenDist1:
		return 1
	default:
		return 0
	}
}

// labelNearBrand reports whether any dot-separated label of registered is within
// maxDist Damerau-Levenshtein edits of brand. A length-window pre-filter rejects
// labels that cannot possibly be within maxDist before the O(n*m) computation.
func labelNearBrand(registered, brand string, maxDist int) bool {
	bl := len(brand)
	for _, label := range strings.Split(registered, ".") {
		if label == "" {
			continue
		}
		if abs(len(label)-bl) > maxDist {
			continue // length window: too different to be within maxDist
		}
		if damerauLevenshtein(label, brand, maxDist) <= maxDist {
			return true
		}
	}
	return false
}

// damerauLevenshtein computes the optimal-string-alignment (restricted
// Damerau-Levenshtein) distance between a and b, counting insertions, deletions,
// substitutions, and adjacent transpositions. It bails out early, returning
// maxDist+1, once the best achievable distance on a row exceeds maxDist. Two
// rolling rows keep allocation to a single slice.
func damerauLevenshtein(a, b string, maxDist int) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// prev2 = row i-2, prev = row i-1, cur = row i. One backing slice, three views.
	buf := make([]int, 3*(lb+1))
	prev2 := buf[:lb+1]
	prev := buf[lb+1 : 2*(lb+1)]
	cur := buf[2*(lb+1):]

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		cur[0] = i
		rowMin := cur[0]
		for j := 1; j <= lb; j++ {
			best := dlCell(a, b, i, j, prev2, prev, cur)
			cur[j] = best
			if best < rowMin {
				rowMin = best
			}
		}
		if rowMin > maxDist {
			return maxDist + 1
		}
		prev2, prev, cur = prev, cur, prev2
	}
	return prev[lb]
}

// dlCell computes one cell of the Damerau-Levenshtein matrix given the two
// previous rows. Split out of damerauLevenshtein to keep that function's
// cognitive complexity in check.
func dlCell(a, b string, i, j int, prev2, prev, cur []int) int {
	cost := 1
	if a[i-1] == b[j-1] {
		cost = 0
	}
	best := min3(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
	// Adjacent transposition.
	if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
		if t := prev2[j-2] + 1; t < best {
			best = t
		}
	}
	return best
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
