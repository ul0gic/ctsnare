// Package config handles configuration loading and defaults for ctsnare.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ul0gic/ctsnare/internal/domain"
)

// CTLogConfig defines the URL and human-readable name for a single CT log.
type CTLogConfig struct {
	// URL is the CT log base URL, without a trailing slash.
	URL string `toml:"url"`

	Name string `toml:"name"`
}

// SkipOverrides holds user edits to the skip suffix list under [skip_overrides];
// the effective list is GlobalSkipSuffixes + Additions - Removals.
type SkipOverrides struct {
	Additions []string `toml:"additions"`
	Removals  []string `toml:"removals"`
}

// Config holds all configurable values for ctsnare. TOML overrides DefaultConfig;
// CLI flags override both. Use DefaultConfig for a ready-to-use value.
type Config struct {
	CTLogs []CTLogConfig `toml:"ct_logs"`

	// DefaultProfile is used when --profile is not specified. Default: "all".
	DefaultProfile string `toml:"default_profile"`

	// BatchSize is entries fetched per poll request per log. Default: 256.
	BatchSize int `toml:"batch_size"`

	// PollInterval is the wait between consecutive polls of each log. Default: 5s.
	PollInterval time.Duration `toml:"poll_interval"`

	// DBPath defaults to the XDG path ~/.local/share/ctsnare/ctsnare.db; parents are created.
	DBPath string `toml:"db_path"`

	// CustomProfiles can extend a built-in via Description "extends:<name>".
	CustomProfiles map[string]domain.Profile `toml:"custom_profiles"`

	SkipOverrides SkipOverrides `toml:"skip_overrides"`

	// TLDTiers is the burner/cheap suspicious-TLD system; empty tiers fall back to defaults.
	TLDTiers TLDTiers `toml:"tld_tiers"`

	// Backtrack starts the poller at (tree_size - Backtrack) when > 0. Default: 0 (tip).
	Backtrack int64 `toml:"backtrack"`

	// MinScore is the threshold below which hits are discarded. Default: 0 (store all scored).
	MinScore int `toml:"min_score"`
}

// DefaultConfig returns a Config with sensible production defaults.
// The returned config is ready to use without a config file.
func DefaultConfig() *Config {
	return &Config{
		CTLogs: []CTLogConfig{
			{
				URL:  "https://ct.googleapis.com/logs/us1/argon2026h1",
				Name: "Google Argon 2026h1",
			},
			{
				URL:  "https://ct.googleapis.com/logs/us1/argon2026h2",
				Name: "Google Argon 2026h2",
			},
			{
				URL:  "https://ct.googleapis.com/logs/eu1/xenon2026h1",
				Name: "Google Xenon 2026h1",
			},
		},
		DefaultProfile: "all",
		BatchSize:      256,
		PollInterval:   5 * time.Second,
		DBPath:         defaultDBPath(),
		CustomProfiles: make(map[string]domain.Profile),
		SkipOverrides:  SkipOverrides{},
	}
}

// Load reads a TOML config file with defaults applied for missing values.
// A missing file or empty path returns the default config without error.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path) //nolint:gosec // path is the user-supplied config file location, not untrusted input
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	applyDefaults(cfg)

	return cfg, nil
}

// MergeFlags applies CLI flag overrides to the config. Zero values are
// treated as "not set" and do not override.
func MergeFlags(cfg *Config, dbPath string, batchSize int, pollInterval time.Duration, backtrack int64, minScore int) {
	if dbPath != "" {
		cfg.DBPath = dbPath
	}
	if batchSize > 0 {
		cfg.BatchSize = batchSize
	}
	if pollInterval > 0 {
		cfg.PollInterval = pollInterval
	}
	if backtrack > 0 {
		cfg.Backtrack = backtrack
	}
	if minScore > 0 {
		cfg.MinScore = minScore
	}
}

// DefaultConfigPath returns the XDG-compliant config file path:
// $XDG_CONFIG_HOME/ctsnare/config.toml or ~/.config/ctsnare/config.toml.
func DefaultConfigPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(".", "ctsnare", "config.toml")
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "ctsnare", "config.toml")
}

// LoadSkipOverrides reads only the [skip_overrides] section, avoiding a full
// config load. A missing file or empty path returns empty overrides.
func LoadSkipOverrides(path string) (SkipOverrides, error) {
	if path == "" {
		return SkipOverrides{}, nil
	}

	data, err := os.ReadFile(path) //nolint:gosec // path is the user-supplied config file location, not untrusted input
	if err != nil {
		if os.IsNotExist(err) {
			return SkipOverrides{}, nil
		}
		return SkipOverrides{}, fmt.Errorf("reading config file %s: %w", path, err)
	}

	var partial struct {
		SkipOverrides SkipOverrides `toml:"skip_overrides"`
	}
	if err := toml.Unmarshal(data, &partial); err != nil {
		return SkipOverrides{}, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	return partial.SkipOverrides, nil
}

// SaveSkipOverrides updates only the [skip_overrides] section of the config and
// writes it back atomically (temp file + rename), creating parents as needed.
func SaveSkipOverrides(path string, overrides SkipOverrides) error {
	if path == "" {
		return errors.New("config path is empty")
	}

	// 0700 keeps the config dir user-private.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config directory %s: %w", dir, err)
	}

	rawConfig, err := loadRawConfig(path)
	if err != nil {
		return err
	}

	// Empty slices instead of nil so TOML emits `additions = []` not an omitted key.
	additions := overrides.Additions
	if additions == nil {
		additions = []string{}
	}
	removals := overrides.Removals
	if removals == nil {
		removals = []string{}
	}

	rawConfig["skip_overrides"] = map[string]any{
		"additions": additions,
		"removals":  removals,
	}

	var buf bytes.Buffer
	buf.WriteString("# ctsnare configuration\n")
	buf.WriteString("# Manage skip suffixes with: ctsnare skip add/remove/list/reset\n")
	buf.WriteString("#\n")
	buf.WriteString("# The old 'skip_suffixes' key is deprecated and ignored.\n")
	buf.WriteString("# Use [skip_overrides] instead.\n\n")

	encoder := toml.NewEncoder(&buf)
	if err = encoder.Encode(rawConfig); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	return atomicWrite(dir, path, buf.Bytes())
}

// loadRawConfig reads and parses an existing config file into a raw map. A
// missing file yields an empty map so callers can write a fresh config.
func loadRawConfig(path string) (map[string]any, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is the user-supplied config file location, not untrusted input
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}
	rawConfig := make(map[string]any)
	if err := toml.Unmarshal(data, &rawConfig); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}
	return rawConfig, nil
}

// atomicWrite writes data to path by first writing a temp file in dir and then
// renaming it into place, so a crash mid-write never leaves a partial config.
func atomicWrite(dir, path string, data []byte) error {
	tmpFile, err := os.CreateTemp(dir, "config-*.toml.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup of a temp file on an error path
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup of a temp file on an error path
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup of a temp file on an error path
		return fmt.Errorf("renaming temp file to %s: %w", path, err)
	}
	return nil
}

// applyDefaults fills in zero-valued fields with sensible defaults.
func applyDefaults(cfg *Config) {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 256
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 5 * time.Second
	}
	if cfg.DBPath == "" {
		cfg.DBPath = defaultDBPath()
	}
	if cfg.DefaultProfile == "" {
		cfg.DefaultProfile = "all"
	}
	if len(cfg.CTLogs) == 0 {
		cfg.CTLogs = DefaultConfig().CTLogs
	}
	if cfg.CustomProfiles == nil {
		cfg.CustomProfiles = make(map[string]domain.Profile)
	}
	if cfg.SkipOverrides.Additions == nil {
		cfg.SkipOverrides.Additions = []string{}
	}
	if cfg.SkipOverrides.Removals == nil {
		cfg.SkipOverrides.Removals = []string{}
	}
}

// defaultDBPath returns the XDG-compliant database path.
func defaultDBPath() string {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "ctsnare.db"
		}
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "ctsnare", "ctsnare.db")
}
