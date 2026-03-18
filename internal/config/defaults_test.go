package config

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig_CTLogs_NonEmpty(t *testing.T) {
	cfg := DefaultConfig()
	require.NotEmpty(t, cfg.CTLogs, "default config must have at least one CT log")
}

func TestDefaultConfig_CTLogs_ValidURLs(t *testing.T) {
	cfg := DefaultConfig()
	for _, log := range cfg.CTLogs {
		assert.NotEmpty(t, log.URL, "CT log URL must not be empty")
		assert.NotEmpty(t, log.Name, "CT log name must not be empty")

		parsed, err := url.Parse(log.URL)
		require.NoError(t, err, "CT log URL must be parseable: %s", log.URL)
		assert.Equal(t, "https", parsed.Scheme, "CT log URL must use HTTPS: %s", log.URL)
		assert.NotEmpty(t, parsed.Host, "CT log URL must have a host: %s", log.URL)
	}
}

func TestDefaultConfig_BatchSize_Positive(t *testing.T) {
	cfg := DefaultConfig()
	assert.Greater(t, cfg.BatchSize, 0, "batch size must be positive")
}

func TestDefaultConfig_PollInterval_Positive(t *testing.T) {
	cfg := DefaultConfig()
	assert.Greater(t, cfg.PollInterval, time.Duration(0), "poll interval must be positive")
}

func TestDefaultConfig_DBPath_NonEmpty(t *testing.T) {
	cfg := DefaultConfig()
	assert.NotEmpty(t, cfg.DBPath, "database path must not be empty")
}

func TestDefaultConfig_DBPath_ContainsCtsnare(t *testing.T) {
	cfg := DefaultConfig()
	assert.Contains(t, cfg.DBPath, "ctsnare", "database path must include ctsnare")
}

func TestDefaultConfig_DefaultProfile_NonEmpty(t *testing.T) {
	cfg := DefaultConfig()
	assert.NotEmpty(t, cfg.DefaultProfile, "default profile must not be empty")
	assert.Equal(t, "all", cfg.DefaultProfile, "default profile should be 'all'")
}

func TestDefaultConfig_SkipOverrides_Empty(t *testing.T) {
	cfg := DefaultConfig()
	assert.Empty(t, cfg.SkipOverrides.Additions, "default config should have no skip override additions")
	assert.Empty(t, cfg.SkipOverrides.Removals, "default config should have no skip override removals")
}

func TestDefaultConfig_CustomProfiles_Initialized(t *testing.T) {
	cfg := DefaultConfig()
	assert.NotNil(t, cfg.CustomProfiles, "custom profiles map must be initialized (not nil)")
}

func TestDefaultDBPath_XDGDataHome_Set(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)

	path := defaultDBPath()
	assert.True(t, strings.HasPrefix(path, dir),
		"path %q should start with XDG_DATA_HOME %q", path, dir)
	assert.Contains(t, path, "ctsnare")
	assert.True(t, filepath.IsAbs(path), "path should be absolute")
}

func TestDefaultDBPath_XDGDataHome_Unset(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")

	path := defaultDBPath()
	home, err := os.UserHomeDir()
	if err != nil {
		// If we can't get home dir, the function falls back to "ctsnare.db".
		assert.Equal(t, "ctsnare.db", path)
		return
	}

	expected := filepath.Join(home, ".local", "share")
	assert.True(t, strings.HasPrefix(path, expected),
		"path %q should start with %q when XDG_DATA_HOME is unset", path, expected)
	assert.Contains(t, path, "ctsnare")
}

func TestApplyDefaults_FillsZeroValues(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)

	assert.Greater(t, cfg.BatchSize, 0, "batch size should be filled")
	assert.Greater(t, cfg.PollInterval, time.Duration(0), "poll interval should be filled")
	assert.NotEmpty(t, cfg.DBPath, "db path should be filled")
	assert.NotEmpty(t, cfg.DefaultProfile, "default profile should be filled")
	assert.NotEmpty(t, cfg.CTLogs, "CT logs should be filled")
	assert.NotNil(t, cfg.CustomProfiles, "custom profiles should be initialized")
	assert.NotNil(t, cfg.SkipOverrides.Additions, "skip overrides additions should be initialized")
	assert.NotNil(t, cfg.SkipOverrides.Removals, "skip overrides removals should be initialized")
}

func TestApplyDefaults_PreservesExistingValues(t *testing.T) {
	cfg := &Config{
		BatchSize:      1024,
		PollInterval:   30 * time.Second,
		DBPath:         "/custom/path.db",
		DefaultProfile: "crypto",
		CTLogs: []CTLogConfig{
			{URL: "https://custom.log/ct", Name: "Custom"},
		},
		SkipOverrides: SkipOverrides{
			Additions: []string{"custom.com"},
			Removals:  []string{"google.com"},
		},
	}
	applyDefaults(cfg)

	assert.Equal(t, 1024, cfg.BatchSize, "existing batch size should be preserved")
	assert.Equal(t, 30*time.Second, cfg.PollInterval, "existing poll interval should be preserved")
	assert.Equal(t, "/custom/path.db", cfg.DBPath, "existing db path should be preserved")
	assert.Equal(t, "crypto", cfg.DefaultProfile, "existing default profile should be preserved")
	assert.Len(t, cfg.CTLogs, 1, "existing CT logs should be preserved")
	assert.Equal(t, []string{"custom.com"}, cfg.SkipOverrides.Additions, "existing additions should be preserved")
	assert.Equal(t, []string{"google.com"}, cfg.SkipOverrides.Removals, "existing removals should be preserved")
}

func TestDefaultConfigPath_NonEmpty(t *testing.T) {
	path := DefaultConfigPath()
	assert.NotEmpty(t, path, "default config path must not be empty")
	assert.Contains(t, path, "ctsnare", "default config path must contain ctsnare")
	assert.True(t, filepath.IsAbs(path) || strings.Contains(path, "ctsnare"),
		"config path should be absolute or contain ctsnare")
}

func TestDefaultConfigPath_XDGConfigHome_Set(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	path := DefaultConfigPath()
	assert.True(t, strings.HasPrefix(path, dir),
		"path %q should start with XDG_CONFIG_HOME %q", path, dir)
	assert.True(t, strings.HasSuffix(path, "config.toml"),
		"path %q should end with config.toml", path)
}
