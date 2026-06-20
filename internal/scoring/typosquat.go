package scoring

import "strings"

const typosquatPoints = 3

// Short brands tolerate one edit to limit false positives; longer brands absorb two.
const (
	typoMinLenDist1 = 5
	typoMinLenDist2 = 8
)

// scoreTyposquat reports brand keywords within an edit-distance window of a
// registered label (not already an exact substring), prefixed "~" to flag fuzziness.
func scoreTyposquat(domainName string, brand []string) (matched []string) {
	if len(brand) == 0 {
		return nil
	}

	lower := strings.ToLower(domainName)
	registered := registeredPart(lower)

	for _, kw := range brand {
		lkw := strings.ToLower(kw)
		if len(lkw) < typoMinLenDist1 {
			continue
		}
		// Exact substring is a literal keyword hit, already scored elsewhere.
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

// labelNearBrand reports whether any registered label is within maxDist edits of brand.
func labelNearBrand(registered, brand string, maxDist int) bool {
	bl := len(brand)
	for _, label := range strings.Split(registered, ".") {
		if label == "" {
			continue
		}
		if abs(len(label)-bl) > maxDist {
			continue
		}
		if damerauLevenshtein(label, brand, maxDist) <= maxDist {
			return true
		}
	}
	return false
}

// damerauLevenshtein returns the optimal-string-alignment distance between a and b,
// bailing out with maxDist+1 once a row's best exceeds maxDist.
func damerauLevenshtein(a, b string, maxDist int) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// prev2 = row i-2, prev = row i-1, cur = row i: one backing slice, three views.
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

func dlCell(a, b string, i, j int, prev2, prev, cur []int) int {
	cost := 1
	if a[i-1] == b[j-1] {
		cost = 0
	}
	best := min3(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
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
