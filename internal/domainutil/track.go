package domainutil

import "strings"

// NormalizeTrackTarget canonicalizes a tracking target for exact,
// case-insensitive matching; whitespace-only input normalizes to "".
//
//	"OpenAI.com"   -> "openai.com"
//	"*.openai.com" -> "openai.com"
//	".openai.com"  -> "openai.com"
func NormalizeTrackTarget(target string) string {
	t := strings.ToLower(strings.TrimSpace(target))
	t = strings.TrimPrefix(t, "*.")
	t = strings.TrimPrefix(t, ".")
	t = strings.TrimSuffix(t, ".")
	return t
}

// NormalizeTrackTargets normalizes each target in order, dropping empties; nil
// or empty input yields nil.
func NormalizeTrackTargets(targets []string) []string {
	if len(targets) == 0 {
		return nil
	}
	out := make([]string, 0, len(targets))
	for _, t := range targets {
		if n := NormalizeTrackTarget(t); n != "" {
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// MatchesTrackTarget reports apex-plus-subdomain match (d == target OR subdomain
// of target). target MUST be pre-normalized; the storage SQL predicate must agree (parity test).
func MatchesTrackTarget(d, target string) bool {
	if target == "" {
		return false
	}
	dl := strings.ToLower(d)
	if dl == target {
		return true
	}
	return strings.HasSuffix(dl, "."+target)
}

// MatchesAnyTrackTarget reports whether d matches any normalized target.
func MatchesAnyTrackTarget(d string, targets []string) bool {
	for _, t := range targets {
		if MatchesTrackTarget(d, t) {
			return true
		}
	}
	return false
}
