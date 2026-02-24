package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_ValidTOML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	content := `
default_profile = "crypto"
batch_size = 512
poll_interval = "10s"
db_path = "/tmp/test.db"
skip_suffixes = ["example.com"]

[[ct_logs]]
url = "https://ct.example.com/log1"
name = "Test Log 1"

[[ct_logs]]
url = "https://ct.example.com/log2"
name = "Test Log 2"
`
	err := os.WriteFile(cfgPath, []byte(content), 0o600)
	require.NoError(t, err)

	cfg, err := Load(cfgPath)
	require.NoError(t, err)

	assert.Equal(t, "crypto", cfg.DefaultProfile)
	assert.Equal(t, 512, cfg.BatchSize)
	assert.Equal(t, 10*time.Second, cfg.PollInterval)
	assert.Equal(t, "/tmp/test.db", cfg.DBPath)
	assert.Len(t, cfg.CTLogs, 2)
	assert.Equal(t, "https://ct.example.com/log1", cfg.CTLogs[0].URL)
	assert.Equal(t, "Test Log 1", cfg.CTLogs[0].Name)
	assert.Equal(t, []string{"example.com"}, cfg.SkipSuffixes)
}

func TestLoad_EmptyFile_UsesDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	err := os.WriteFile(cfgPath, []byte(""), 0o600)
	require.NoError(t, err)

	cfg, err := Load(cfgPath)
	require.NoError(t, err)

	defaults := DefaultConfig()
	assert.Equal(t, defaults.BatchSize, cfg.BatchSize)
	assert.Equal(t, defaults.PollInterval, cfg.PollInterval)
	assert.Equal(t, defaults.DefaultProfile, cfg.DefaultProfile)
	assert.NotEmpty(t, cfg.CTLogs)
	assert.NotEmpty(t, cfg.SkipSuffixes)
}

func TestLoad_NonExistentFile_ReturnsDefaults(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.toml")
	require.NoError(t, err)

	defaults := DefaultConfig()
	assert.Equal(t, defaults.BatchSize, cfg.BatchSize)
	assert.Equal(t, defaults.DefaultProfile, cfg.DefaultProfile)
}

func TestLoad_EmptyPath_ReturnsDefaults(t *testing.T) {
	cfg, err := Load("")
	require.NoError(t, err)

	defaults := DefaultConfig()
	assert.Equal(t, defaults.BatchSize, cfg.BatchSize)
}

func TestLoad_InvalidTOML_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	err := os.WriteFile(cfgPath, []byte("not valid [[[toml"), 0o600)
	require.NoError(t, err)

	_, err = Load(cfgPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing config file")
}

func TestMergeFlags_OverridesNonZeroValues(t *testing.T) {
	cfg := DefaultConfig()
	original := *cfg

	MergeFlags(cfg, "/custom/path.db", 1024, 30*time.Second)

	assert.Equal(t, "/custom/path.db", cfg.DBPath)
	assert.Equal(t, 1024, cfg.BatchSize)
	assert.Equal(t, 30*time.Second, cfg.PollInterval)
	// Other fields unchanged
	assert.Equal(t, original.DefaultProfile, cfg.DefaultProfile)
}

func TestMergeFlags_ZeroValuesDoNotOverride(t *testing.T) {
	cfg := DefaultConfig()
	original := *cfg

	MergeFlags(cfg, "", 0, 0)

	assert.Equal(t, original.DBPath, cfg.DBPath)
	assert.Equal(t, original.BatchSize, cfg.BatchSize)
	assert.Equal(t, original.PollInterval, cfg.PollInterval)
}

func TestDefaultConfig_HasSensibleDefaults(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "all", cfg.DefaultProfile)
	assert.Equal(t, 256, cfg.BatchSize)
	assert.Equal(t, 5*time.Second, cfg.PollInterval)
	assert.NotEmpty(t, cfg.DBPath)
	assert.Len(t, cfg.CTLogs, 3)
	assert.NotNil(t, cfg.CustomProfiles)
	assert.NotEmpty(t, cfg.SkipSuffixes)

	for _, log := range cfg.CTLogs {
		assert.NotEmpty(t, log.URL)
		assert.NotEmpty(t, log.Name)
	}
}

func TestDefaultDBPath_XDGCompliant(t *testing.T) {
	path := defaultDBPath()
	assert.Contains(t, path, "ctsnare")
	assert.True(t, filepath.IsAbs(path) || path == "ctsnare.db",
		"DB path should be absolute or fallback")
}

func TestLoad_PartialConfig_MergesWithDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	content := `
default_profile = "phishing"
`
	err := os.WriteFile(cfgPath, []byte(content), 0o600)
	require.NoError(t, err)

	cfg, err := Load(cfgPath)
	require.NoError(t, err)

	assert.Equal(t, "phishing", cfg.DefaultProfile)
	// Unset fields get defaults
	assert.Equal(t, 256, cfg.BatchSize)
	assert.Equal(t, 5*time.Second, cfg.PollInterval)
	assert.NotEmpty(t, cfg.CTLogs)
}
