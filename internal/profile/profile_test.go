package profile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ul0gic/ctsnare/internal/domain"
)

func TestLoadProfile_BuiltinCrypto(t *testing.T) {
	m := NewManager(nil)
	p, err := m.LoadProfile("crypto")
	require.NoError(t, err)

	assert.Equal(t, "crypto", p.Name)
	assert.Contains(t, p.Keywords, "bitcoin")
	assert.Contains(t, p.Keywords, "wallet")
	assert.Contains(t, p.SuspiciousTLDs, ".xyz")
	assert.NotEmpty(t, p.SkipSuffixes)
	assert.NotEmpty(t, p.Description)
}

func TestLoadProfile_BuiltinPhishing(t *testing.T) {
	m := NewManager(nil)
	p, err := m.LoadProfile("phishing")
	require.NoError(t, err)

	assert.Equal(t, "phishing", p.Name)
	assert.Contains(t, p.Keywords, "login")
	assert.Contains(t, p.Keywords, "paypal")
	assert.Contains(t, p.SuspiciousTLDs, ".tk")
	assert.NotEmpty(t, p.SkipSuffixes)
}

func TestLoadProfile_BuiltinAll(t *testing.T) {
	m := NewManager(nil)
	p, err := m.LoadProfile("all")
	require.NoError(t, err)

	assert.Equal(t, "all", p.Name)
	// Should contain keywords from both crypto and phishing
	assert.Contains(t, p.Keywords, "bitcoin")
	assert.Contains(t, p.Keywords, "login")
	// Should contain TLDs from both
	assert.Contains(t, p.SuspiciousTLDs, ".xyz")
	assert.Contains(t, p.SuspiciousTLDs, ".tk")
}

func TestLoadProfile_UnknownReturnsError(t *testing.T) {
	m := NewManager(nil)
	_, err := m.LoadProfile("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown profile")
}

func TestListProfiles_ReturnsSortedNames(t *testing.T) {
	m := NewManager(nil)
	names := m.ListProfiles()

	assert.Equal(t, []string{"all", "crypto", "phishing"}, names)
}

func TestCustomProfile_ExtendsBuiltin(t *testing.T) {
	custom := map[string]domain.Profile{
		"my-crypto": {
			Keywords:    []string{"nft", "opensea"},
			Description: "extends:crypto",
		},
	}
	m := NewManager(custom)
	p, err := m.LoadProfile("my-crypto")
	require.NoError(t, err)

	assert.Equal(t, "my-crypto", p.Name)
	// Should have crypto keywords plus custom ones
	assert.Contains(t, p.Keywords, "bitcoin")
	assert.Contains(t, p.Keywords, "nft")
	assert.Contains(t, p.Keywords, "opensea")
	// Should have crypto TLDs
	assert.Contains(t, p.SuspiciousTLDs, ".xyz")
}

func TestCustomProfile_WithoutExtends(t *testing.T) {
	custom := map[string]domain.Profile{
		"fresh": {
			Keywords:       []string{"test", "demo"},
			SuspiciousTLDs: []string{".test"},
			Description:    "A fresh custom profile",
		},
	}
	m := NewManager(custom)
	p, err := m.LoadProfile("fresh")
	require.NoError(t, err)

	assert.Equal(t, "fresh", p.Name)
	assert.Equal(t, []string{"test", "demo"}, p.Keywords)
	assert.Equal(t, []string{".test"}, p.SuspiciousTLDs)
	// Should NOT have any built-in keywords
	assert.NotContains(t, p.Keywords, "bitcoin")
	assert.NotContains(t, p.Keywords, "login")
}

func TestCustomProfile_AppearsInList(t *testing.T) {
	custom := map[string]domain.Profile{
		"custom-one": {Keywords: []string{"test"}},
	}
	m := NewManager(custom)
	names := m.ListProfiles()

	assert.Contains(t, names, "custom-one")
	assert.Contains(t, names, "all")
	assert.Contains(t, names, "crypto")
	assert.Contains(t, names, "phishing")
}

func TestNewManager_NilCustomProfiles(t *testing.T) {
	m := NewManager(nil)
	assert.NotNil(t, m)
	assert.Len(t, m.ListProfiles(), 3)
}

func TestAllProfile_NoDuplicateKeywords(t *testing.T) {
	m := NewManager(nil)
	p, err := m.LoadProfile("all")
	require.NoError(t, err)

	seen := make(map[string]struct{})
	for _, kw := range p.Keywords {
		_, exists := seen[kw]
		assert.False(t, exists, "duplicate keyword: %s", kw)
		seen[kw] = struct{}{}
	}
}

func TestAllProfile_NoDuplicateTLDs(t *testing.T) {
	m := NewManager(nil)
	p, err := m.LoadProfile("all")
	require.NoError(t, err)

	seen := make(map[string]struct{})
	for _, tld := range p.SuspiciousTLDs {
		_, exists := seen[tld]
		assert.False(t, exists, "duplicate TLD: %s", tld)
		seen[tld] = struct{}{}
	}
}
