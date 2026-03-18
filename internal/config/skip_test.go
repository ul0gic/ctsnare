package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeSkipSuffixes_GlobalsOnly(t *testing.T) {
	globals := []string{"a.com", "b.com", "c.com"}
	overrides := SkipOverrides{}

	result := MergeSkipSuffixes(globals, overrides)
	assert.Equal(t, globals, result)
}

func TestMergeSkipSuffixes_GlobalsPlusAdditions(t *testing.T) {
	globals := []string{"a.com", "b.com"}
	overrides := SkipOverrides{
		Additions: []string{"x.com", "y.com"},
	}

	result := MergeSkipSuffixes(globals, overrides)
	assert.Equal(t, []string{"a.com", "b.com", "x.com", "y.com"}, result)
}

func TestMergeSkipSuffixes_GlobalsPlusRemovals(t *testing.T) {
	globals := []string{"a.com", "b.com", "c.com"}
	overrides := SkipOverrides{
		Removals: []string{"b.com"},
	}

	result := MergeSkipSuffixes(globals, overrides)
	assert.Equal(t, []string{"a.com", "c.com"}, result)
}

func TestMergeSkipSuffixes_FullMerge(t *testing.T) {
	globals := []string{"a.com", "b.com", "c.com"}
	overrides := SkipOverrides{
		Additions: []string{"x.com"},
		Removals:  []string{"b.com"},
	}

	result := MergeSkipSuffixes(globals, overrides)
	assert.Equal(t, []string{"a.com", "c.com", "x.com"}, result)
}

func TestMergeSkipSuffixes_AdditionDuplicatesGlobal(t *testing.T) {
	globals := []string{"a.com", "b.com"}
	overrides := SkipOverrides{
		Additions: []string{"a.com", "c.com"},
	}

	result := MergeSkipSuffixes(globals, overrides)
	// a.com should appear only once (from globals), c.com appended.
	assert.Equal(t, []string{"a.com", "b.com", "c.com"}, result)
}

func TestMergeSkipSuffixes_RemovalOfNonExistent(t *testing.T) {
	globals := []string{"a.com", "b.com"}
	overrides := SkipOverrides{
		Removals: []string{"nonexistent.com"},
	}

	// Removing a non-existent domain has no effect.
	result := MergeSkipSuffixes(globals, overrides)
	assert.Equal(t, []string{"a.com", "b.com"}, result)
}

func TestMergeSkipSuffixes_EmptyGlobalsWithAdditions(t *testing.T) {
	overrides := SkipOverrides{
		Additions: []string{"x.com", "y.com"},
	}

	result := MergeSkipSuffixes(nil, overrides)
	assert.Equal(t, []string{"x.com", "y.com"}, result)
}

func TestMergeSkipSuffixes_EmptyEverything(t *testing.T) {
	result := MergeSkipSuffixes(nil, SkipOverrides{})
	assert.Empty(t, result)
}

func TestMergeSkipSuffixes_DeduplicatesWithinGlobals(t *testing.T) {
	// Edge case: if globals have duplicates, they are deduplicated.
	globals := []string{"a.com", "b.com", "a.com"}
	overrides := SkipOverrides{}

	result := MergeSkipSuffixes(globals, overrides)
	assert.Equal(t, []string{"a.com", "b.com"}, result)
}

func TestMergeSkipSuffixes_DeduplicatesWithinAdditions(t *testing.T) {
	globals := []string{"a.com"}
	overrides := SkipOverrides{
		Additions: []string{"x.com", "x.com"},
	}

	result := MergeSkipSuffixes(globals, overrides)
	assert.Equal(t, []string{"a.com", "x.com"}, result)
}

func TestMergeSkipSuffixes_RemoveAllGlobals(t *testing.T) {
	globals := []string{"a.com", "b.com"}
	overrides := SkipOverrides{
		Removals: []string{"a.com", "b.com"},
	}

	result := MergeSkipSuffixes(globals, overrides)
	assert.Empty(t, result)
}

func TestMergeSkipSuffixes_RemoveGlobalThenAddItBack(t *testing.T) {
	// If a domain is in both removals and additions, the removal wins
	// for globals, but the addition is also present -- so it ends up
	// in the result via additions.
	globals := []string{"a.com", "b.com"}
	overrides := SkipOverrides{
		Additions: []string{"a.com"},
		Removals:  []string{"a.com"},
	}

	result := MergeSkipSuffixes(globals, overrides)
	// a.com removed from globals, but added back via additions.
	assert.Equal(t, []string{"b.com", "a.com"}, result)
}
