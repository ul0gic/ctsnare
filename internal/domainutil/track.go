package domainutil

import "strings"

// NormalizeTrackTarget canonicalizes a user-supplied domain-tracking target so
// that matching is exact and case-insensitive. It lowercases the value, strips a
// leading "*." wildcard or bare leading ".", and strips a trailing "." (FQDN
// notation). An empty or whitespace-only target normalizes to "".
//
// Examples:
//
//	"OpenAI.com"   -> "openai.com"
//	"*.openai.com" -> "openai.com"
//	".openai.com"  -> "openai.com"
//	"openai.com."  -> "openai.com"
func NormalizeTrackTarget(target string) string {
	t := strings.ToLower(strings.TrimSpace(target))
	t = strings.TrimPrefix(t, "*.")
	t = strings.TrimPrefix(t, ".")
	t = strings.TrimSuffix(t, ".")
	return t
}

// NormalizeTrackTargets normalizes each target and drops any that normalize to
// an empty string. The returned slice preserves input order. A nil or empty
// input yields nil.
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

// MatchesTrackTarget reports whether the extracted/stored domain d matches the
// tracking target according to apex-plus-subdomain semantics: d matches iff
// d == target OR d is a subdomain of target (d ends with "." + target).
//
// The target MUST already be normalized (lowercased, no wildcard/leading/trailing
// dot) — callers should pass values produced by NormalizeTrackTarget. The domain
// d is lowercased here so the comparison is case-insensitive.
//
// This is the authoritative Go matcher. The SQL predicate in the storage layer
// (LOWER(domain) = ? OR LOWER(domain) LIKE ?) must agree with it; a parity test
// enforces this.
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

// MatchesAnyTrackTarget reports whether d matches any of the given normalized
// targets. Returns false for an empty target list.
func MatchesAnyTrackTarget(d string, targets []string) bool {
	for _, t := range targets {
		if MatchesTrackTarget(d, t) {
			return true
		}
	}
	return false
}
