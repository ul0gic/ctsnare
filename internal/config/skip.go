package config

// MergeSkipSuffixes returns the deduplicated effective skip list:
// globals + overrides.Additions - overrides.Removals.
func MergeSkipSuffixes(globals []string, overrides SkipOverrides) []string {
	removals := make(map[string]struct{}, len(overrides.Removals))
	for _, r := range overrides.Removals {
		removals[r] = struct{}{}
	}

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

	for _, a := range overrides.Additions {
		if _, dup := seen[a]; dup {
			continue
		}
		seen[a] = struct{}{}
		result = append(result, a)
	}

	return result
}
