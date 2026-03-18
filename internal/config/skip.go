package config

// MergeSkipSuffixes computes the effective skip suffix list by merging
// hardcoded globals with user overrides. The merge logic is:
//
//	effective = globals + overrides.Additions - overrides.Removals
//
// The result is deduplicated. This function is the single source of truth
// for the effective skip list -- it is called once at startup and the result
// injected into the profile before scoring begins.
func MergeSkipSuffixes(globals []string, overrides SkipOverrides) []string {
	// Build removals set for O(1) lookup.
	removals := make(map[string]struct{}, len(overrides.Removals))
	for _, r := range overrides.Removals {
		removals[r] = struct{}{}
	}

	// Start with globals minus removals.
	seen := make(map[string]struct{}, len(globals)+len(overrides.Additions))
	result := make([]string, 0, len(globals)+len(overrides.Additions))

	for _, g := range globals {
		if _, removed := removals[g]; removed {
			continue
		}
		if _, dup := seen[g]; dup {
			continue
		}
		seen[g] = struct{}{}
		result = append(result, g)
	}

	// Append user additions, deduplicating against globals.
	for _, a := range overrides.Additions {
		if _, dup := seen[a]; dup {
			continue
		}
		seen[a] = struct{}{}
		result = append(result, a)
	}

	return result
}
