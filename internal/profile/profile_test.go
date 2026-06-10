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
	assert.Contains(t, p.BrandKeywords, "bitcoin")
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
	assert.Contains(t, p.BrandKeywords, "paypal")
	assert.Contains(t, p.SuspiciousTLDs, ".tk")
	assert.NotContains(t, p.SuspiciousTLDs, ".info", "info removed as too noisy")
	assert.NotEmpty(t, p.SkipSuffixes)
}

func TestLoadProfile_BuiltinAI(t *testing.T) {
	m := NewManager(nil)
	p, err := m.LoadProfile("ai")
	require.NoError(t, err)

	assert.Equal(t, "ai", p.Name)
	assert.Contains(t, p.BrandKeywords, "openai")
	assert.Contains(t, p.BrandKeywords, "anthropic")
	assert.NotEmpty(t, p.SuspiciousTLDs)
	assert.NotEmpty(t, p.Description)
}

func TestLoadProfile_BuiltinAll(t *testing.T) {
	m := NewManager(nil)
	p, err := m.LoadProfile("all")
	require.NoError(t, err)

	assert.Equal(t, "all", p.Name)
	// Should contain brand keywords from crypto, phishing, and ai
	assert.Contains(t, p.BrandKeywords, "bitcoin")
	assert.Contains(t, p.BrandKeywords, "paypal")
	assert.Contains(t, p.BrandKeywords, "openai")
	// Should contain generic keywords from crypto and phishing
	assert.Contains(t, p.Keywords, "login")
	assert.Contains(t, p.Keywords, "wallet")
	// Should contain TLDs from all
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

	assert.Equal(t, []string{"ai", "all", "crypto", "phishing"}, names)
}

func TestCustomProfile_ExtendsBuiltin(t *testing.T) {
	custom := map[string]domain.Profile{
		"my-crypto": {
			Keywords:      []string{"nft", "custom-term"},
			BrandKeywords: []string{"mybrand"},
			Description:   "extends:crypto",
		},
	}
	m := NewManager(custom)
	p, err := m.LoadProfile("my-crypto")
	require.NoError(t, err)

	assert.Equal(t, "my-crypto", p.Name)
	// Should inherit crypto brand keywords plus custom brand keywords
	assert.Contains(t, p.BrandKeywords, "bitcoin")
	assert.Contains(t, p.BrandKeywords, "mybrand")
	// Should inherit crypto generic keywords plus custom ones
	assert.Contains(t, p.Keywords, "nft")
	assert.Contains(t, p.Keywords, "custom-term")
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
	assert.NotContains(t, p.Keywords, "login")
	assert.NotContains(t, p.BrandKeywords, "bitcoin")
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
	assert.Contains(t, names, "ai")
}

func TestNewManager_NilCustomProfiles(t *testing.T) {
	m := NewManager(nil)
	assert.NotNil(t, m)
	assert.Len(t, m.ListProfiles(), 4)
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

	seenBrand := make(map[string]struct{})
	for _, kw := range p.BrandKeywords {
		_, exists := seenBrand[kw]
		assert.False(t, exists, "duplicate brand keyword: %s", kw)
		seenBrand[kw] = struct{}{}
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
