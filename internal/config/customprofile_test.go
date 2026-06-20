package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoad_CustomProfile_SnakeCaseKeysBind is the BUG-008 regression: snake_case
// profile keys must bind; without explicit toml tags they were silently dropped.
func TestLoad_CustomProfile_SnakeCaseKeysBind(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	content := `
[custom_profiles.openai]
name = "openai"
description = "Track OpenAI brand abuse"
keywords = ["openai", "chatgpt"]
suspicious_tlds = [".xyz", ".top", ".click"]
skip_suffixes = ["openai.com", "cloudflare.net"]
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o600))

	cfg, err := Load(cfgPath)
	require.NoError(t, err)

	prof, ok := cfg.CustomProfiles["openai"]
	require.True(t, ok, "custom profile 'openai' must load")

	assert.Equal(t, "openai", prof.Name)
	assert.Equal(t, "Track OpenAI brand abuse", prof.Description)
	assert.Equal(t, []string{"openai", "chatgpt"}, prof.Keywords)

	// The crux of BUG-008: these were silently empty before the toml tags.
	require.NotEmpty(t, prof.SuspiciousTLDs, "suspicious_tlds must bind (BUG-008)")
	assert.Equal(t, []string{".xyz", ".top", ".click"}, prof.SuspiciousTLDs)
	require.NotEmpty(t, prof.SkipSuffixes, "skip_suffixes must bind (BUG-008)")
	assert.Equal(t, []string{"openai.com", "cloudflare.net"}, prof.SkipSuffixes)
}
