package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTLDTiers_TOMLReplacesDefaults(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.toml")
	require.NoError(t, os.WriteFile(p, []byte("[tld_tiers]\nburner=[\".zz\"]\ncheap=[\".pp\"]\n"), 0o600))
	c, err := Load(p)
	require.NoError(t, err)
	b, ch := ResolveTLDTiers(c.TLDTiers)
	assert.Equal(t, []string{".zz"}, b)
	assert.Equal(t, []string{".pp"}, ch)
}

func TestTLDTiers_UnsetUsesDefaults(t *testing.T) {
	b, ch := ResolveTLDTiers(TLDTiers{})
	assert.Equal(t, DefaultBurnerTLDs, b)
	assert.Equal(t, DefaultCheapTLDs, ch)
}
