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

[[ct_logs]]
url = "https://ct.example.com/log1"
name = "Test Log 1"

[[ct_logs]]
url = "https://ct.example.com/log2"
name = "Test Log 2"

[skip_overrides]
additions = ["example.com", "test.org"]
removals = ["google.com"]
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
	assert.Equal(t, []string{"example.com", "test.org"}, cfg.SkipOverrides.Additions)
	assert.Equal(t, []string{"google.com"}, cfg.SkipOverrides.Removals)
}

func TestLoad_SkipOverrides_EmptySection(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	content := `
[skip_overrides]
additions = []
removals = []
`
	err := os.WriteFile(cfgPath, []byte(content), 0o600)
	require.NoError(t, err)

	cfg, err := Load(cfgPath)
	require.NoError(t, err)

	assert.Empty(t, cfg.SkipOverrides.Additions)
	assert.Empty(t, cfg.SkipOverrides.Removals)
}

func TestLoad_SkipOverrides_MissingSection(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	content := `
default_profile = "all"
`
	err := os.WriteFile(cfgPath, []byte(content), 0o600)
	require.NoError(t, err)

	cfg, err := Load(cfgPath)
	require.NoError(t, err)

	// applyDefaults initializes to empty slices.
	assert.NotNil(t, cfg.SkipOverrides.Additions)
	assert.NotNil(t, cfg.SkipOverrides.Removals)
	assert.Empty(t, cfg.SkipOverrides.Additions)
	assert.Empty(t, cfg.SkipOverrides.Removals)
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

	MergeFlags(cfg, "/custom/path.db", 1024, 30*time.Second, 500, 0)

	assert.Equal(t, "/custom/path.db", cfg.DBPath)
	assert.Equal(t, 1024, cfg.BatchSize)
	assert.Equal(t, 30*time.Second, cfg.PollInterval)
	assert.Equal(t, int64(500), cfg.Backtrack)
	// Other fields unchanged
	assert.Equal(t, original.DefaultProfile, cfg.DefaultProfile)
}

func TestMergeFlags_ZeroValuesDoNotOverride(t *testing.T) {
	cfg := DefaultConfig()
	original := *cfg

	MergeFlags(cfg, "", 0, 0, 0, 0)

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

func TestLoad_BackwardCompatibility_OldSkipSuffixesKey(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	// Old config format with deprecated skip_suffixes key.
	// BurntSushi/toml does NOT ignore unknown keys by default when
	// unmarshaling into a struct. But since we use Unmarshal on a
	// pre-populated struct, unknown keys are silently dropped.
	content := `
default_profile = "all"
skip_suffixes = ["example.com"]
`
	err := os.WriteFile(cfgPath, []byte(content), 0o600)
	require.NoError(t, err)

	cfg, err := Load(cfgPath)
	require.NoError(t, err, "old skip_suffixes key should parse without error")
	assert.Equal(t, "all", cfg.DefaultProfile)
}

func TestDefaultConfigPath_ReturnsAbsolutePath(t *testing.T) {
	path := DefaultConfigPath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "ctsnare")
}

func TestSaveSkipOverrides_CreatesFileAndDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "nested", "config.toml")

	overrides := SkipOverrides{
		Additions: []string{"example.com"},
		Removals:  []string{"google.com"},
	}

	err := SaveSkipOverrides(path, overrides)
	require.NoError(t, err)

	// Verify file was created.
	_, statErr := os.Stat(path)
	assert.NoError(t, statErr, "config file should exist after save")
}

func TestSaveSkipOverrides_ThenLoadSkipOverrides_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	original := SkipOverrides{
		Additions: []string{"corp.net", "internal.org"},
		Removals:  []string{"google.com", "apple.com"},
	}

	err := SaveSkipOverrides(path, original)
	require.NoError(t, err)

	loaded, err := LoadSkipOverrides(path)
	require.NoError(t, err)

	assert.Equal(t, original.Additions, loaded.Additions)
	assert.Equal(t, original.Removals, loaded.Removals)
}

func TestSaveSkipOverrides_PreservesOtherSections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Write a config with CT logs first.
	initial := `
default_profile = "crypto"
batch_size = 1024

[[ct_logs]]
url = "https://ct.example.com/log1"
name = "Test Log"
`
	err := os.WriteFile(path, []byte(initial), 0o600)
	require.NoError(t, err)

	// Save skip overrides -- should preserve the rest.
	overrides := SkipOverrides{
		Additions: []string{"test.com"},
		Removals:  []string{},
	}
	err = SaveSkipOverrides(path, overrides)
	require.NoError(t, err)

	// Load the full config and verify other sections are intact.
	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, "crypto", cfg.DefaultProfile)
	assert.Equal(t, 1024, cfg.BatchSize)
	assert.Len(t, cfg.CTLogs, 1)
	assert.Equal(t, "https://ct.example.com/log1", cfg.CTLogs[0].URL)
	assert.Equal(t, []string{"test.com"}, cfg.SkipOverrides.Additions)
}

func TestSaveSkipOverrides_EmptyOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	overrides := SkipOverrides{
		Additions: []string{},
		Removals:  []string{},
	}

	err := SaveSkipOverrides(path, overrides)
	require.NoError(t, err)

	loaded, err := LoadSkipOverrides(path)
	require.NoError(t, err)

	assert.Empty(t, loaded.Additions)
	assert.Empty(t, loaded.Removals)
}

func TestSaveSkipOverrides_EmptyPath_ReturnsError(t *testing.T) {
	err := SaveSkipOverrides("", SkipOverrides{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config path is empty")
}

func TestLoadSkipOverrides_NonExistentFile_ReturnsEmpty(t *testing.T) {
	overrides, err := LoadSkipOverrides("/nonexistent/path/config.toml")
	require.NoError(t, err)

	assert.Empty(t, overrides.Additions)
	assert.Empty(t, overrides.Removals)
}

func TestLoadSkipOverrides_EmptyPath_ReturnsEmpty(t *testing.T) {
	overrides, err := LoadSkipOverrides("")
	require.NoError(t, err)

	assert.Empty(t, overrides.Additions)
	assert.Empty(t, overrides.Removals)
}

func TestLoadSkipOverrides_InvalidTOML_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	err := os.WriteFile(path, []byte("not valid [[[toml"), 0o600)
	require.NoError(t, err)

	_, err = LoadSkipOverrides(path)
	assert.Error(t, err)
}

func TestSaveSkipOverrides_NilSlicesBecomesEmptyArrays(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Nil slices should be saved as empty arrays, not omitted.
	overrides := SkipOverrides{
		Additions: nil,
		Removals:  nil,
	}

	err := SaveSkipOverrides(path, overrides)
	require.NoError(t, err)

	loaded, err := LoadSkipOverrides(path)
	require.NoError(t, err)
	assert.NotNil(t, loaded.Additions)
	assert.NotNil(t, loaded.Removals)
}

func TestSaveSkipOverrides_OverwritesExistingOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Save initial overrides.
	initial := SkipOverrides{
		Additions: []string{"first.com"},
		Removals:  []string{},
	}
	require.NoError(t, SaveSkipOverrides(path, initial))

	// Overwrite with different overrides.
	updated := SkipOverrides{
		Additions: []string{"second.com", "third.com"},
		Removals:  []string{"google.com"},
	}
	require.NoError(t, SaveSkipOverrides(path, updated))

	// Load and verify the updated values.
	loaded, err := LoadSkipOverrides(path)
	require.NoError(t, err)
	assert.Equal(t, updated.Additions, loaded.Additions)
	assert.Equal(t, updated.Removals, loaded.Removals)
}

func TestSaveSkipOverrides_InvalidExistingTOML_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Write invalid TOML first.
	err := os.WriteFile(path, []byte("not valid [[[toml"), 0o600)
	require.NoError(t, err)

	// Trying to save should fail because it reads the existing file first.
	err = SaveSkipOverrides(path, SkipOverrides{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing config file")
}

func TestDefaultConfigPath_XDGConfigHome_Unset(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")

	path := DefaultConfigPath()
	home, err := os.UserHomeDir()
	if err != nil {
		// Falls back to relative path.
		assert.Contains(t, path, "ctsnare")
		return
	}

	assert.Contains(t, path, filepath.Join(home, ".config"))
	assert.Contains(t, path, "config.toml")
}
