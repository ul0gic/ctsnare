package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWatchDomainFlag_Repeatable verifies that --domain on the watch command is
// a repeatable string slice: multiple --domain flags accumulate.
func TestWatchDomainFlag_Repeatable(t *testing.T) {
	f := watchCmd.Flags().Lookup("domain")
	require.NotNil(t, f, "watch must define a --domain flag")
	assert.Equal(t, "stringArray", f.Value.Type(), "--domain must be a repeatable string slice")

	got, err := watchCmd.Flags().GetStringArray("domain")
	require.NoError(t, err)
	// Default is no targets.
	assert.Empty(t, got)

	require.NoError(t, watchCmd.Flags().Set("domain", "openai.com"))
	require.NoError(t, watchCmd.Flags().Set("domain", "anthropic.com"))

	got, err = watchCmd.Flags().GetStringArray("domain")
	require.NoError(t, err)
	assert.Equal(t, []string{"openai.com", "anthropic.com"}, got)

	// Reset so other tests / runs see a clean slice (flags are package-global).
	t.Cleanup(func() { watchDomains = nil })
}

// TestQueryDomainFlag_Registered verifies the query command exposes a --domain
// string filter flag.
func TestQueryDomainFlag_Registered(t *testing.T) {
	f := queryCmd.Flags().Lookup("domain")
	require.NotNil(t, f, "query must define a --domain flag")
	assert.Equal(t, "string", f.Value.Type())
}
