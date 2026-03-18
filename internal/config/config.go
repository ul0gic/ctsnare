// Package config handles configuration loading and defaults for ctsnare.
package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ul0gic/ctsnare/internal/domain"
)

// CTLogConfig defines the URL and human-readable name for a single CT log.
type CTLogConfig struct {
	// URL is the base URL of the CT log, without a trailing slash.
	// Example: "https://ct.googleapis.com/logs/us1/argon2025h1"
	URL string `toml:"url"`

	// Name is the human-readable label shown in log output and the TUI stats bar.
	Name string `toml:"name"`
}

// SkipOverrides holds user customizations to the skip suffix list.
// These are persisted to the TOML config file under [skip_overrides] and
// managed via `ctsnare skip add/remove/reset`.
//
// The effective skip list is computed as:
//
//	effective = GlobalSkipSuffixes + Additions - Removals
type SkipOverrides struct {
	// Additions are domain suffixes the user has added to the skip list,
	// on top of the hardcoded GlobalSkipSuffixes.
	Additions []string `toml:"additions"`

	// Removals are GlobalSkipSuffixes entries the user has un-skipped,
	// allowing those domains to be scored again.
	Removals []string `toml:"removals"`
}

// Config holds all configurable values for ctsnare.
// All fields have sensible defaults -- use DefaultConfig to get a ready-to-use Config
// without a config file. Fields from the TOML file override defaults; CLI flags
// override both.
type Config struct {
	// CTLogs is the list of Certificate Transparency logs to poll.
	// Defaults to Google Argon 2026h1, Argon 2026h2, and Xenon 2026h1.
	CTLogs []CTLogConfig `toml:"ct_logs"`

	// DefaultProfile is the keyword profile to use when --profile is not specified.
	// Defaults to "all" (combined crypto + phishing keywords).
	DefaultProfile string `toml:"default_profile"`

	// BatchSize is the number of CT log entries to fetch per poll request per log.
	// Larger values increase throughput at the cost of memory. Default: 256.
	BatchSize int `toml:"batch_size"`

	// PollInterval is how long to wait between consecutive polls of each log.
	// Default: 5 seconds. Set lower for near-real-time monitoring.
	PollInterval time.Duration `toml:"poll_interval"`

	// DBPath is the filesystem path to the SQLite database file.
	// Parent directories are created automatically. Defaults to the XDG-compliant path:
	// ~/.local/share/ctsnare/ctsnare.db (or $XDG_DATA_HOME/ctsnare/ctsnare.db).
	DBPath string `toml:"db_path"`

	// CustomProfiles is a map of user-defined profiles loaded from the TOML config.
	// Keys are profile names; values are Profile definitions.
	// A profile can extend a built-in by setting Description to "extends:<name>".
	CustomProfiles map[string]domain.Profile `toml:"custom_profiles"`

	// SkipOverrides holds user additions and removals to the skip suffix list.
	// Managed via `ctsnare skip add/remove/reset` and persisted under [skip_overrides].
	SkipOverrides SkipOverrides `toml:"skip_overrides"`

	// Backtrack is the number of CT log entries behind the current tip to start at.
	// When > 0, the poller begins at (tree_size - Backtrack), giving immediate
	// results on launch. Default: 0 (start at the tip, wait for new entries).
	Backtrack int64 `toml:"backtrack"`

	// MinScore is the minimum score threshold for storing hits in the database.
	// Domains scoring below this are discarded. Default: 0 (store all scored hits).
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

// Load reads a TOML config file and returns a Config with defaults applied
// for any missing values. If the file does not exist, it returns the default
// config without error. An empty path also returns defaults.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
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

// LoadSkipOverrides reads the TOML config file and returns only the
// SkipOverrides section. If the file does not exist, returns empty
// overrides without error. This is a lightweight read used by
// `ctsnare skip list` without loading the full config.
func LoadSkipOverrides(path string) (SkipOverrides, error) {
	if path == "" {
		return SkipOverrides{}, nil
	}

	data, err := os.ReadFile(path)
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

// SaveSkipOverrides reads the existing TOML config file (or creates it if it
// does not exist), updates only the [skip_overrides] section, and writes it
// back atomically (temp file + rename). Parent directories are created if needed.
func SaveSkipOverrides(path string, overrides SkipOverrides) error {
	if path == "" {
		return fmt.Errorf("config path is empty")
	}

	// Ensure parent directory exists.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory %s: %w", dir, err)
	}

	// Read existing config or start with an empty one.
	var rawConfig map[string]any
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("reading config file %s: %w", path, err)
		}
		// File does not exist -- start from scratch.
		rawConfig = make(map[string]any)
	} else {
		if err := toml.Unmarshal(data, &rawConfig); err != nil {
			return fmt.Errorf("parsing config file %s: %w", path, err)
		}
	}

	// Prepare the overrides for encoding. Use empty slices instead of nil
	// to produce `additions = []` rather than omitting the key.
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

	// Encode to buffer.
	var buf bytes.Buffer
	buf.WriteString("# ctsnare configuration\n")
	buf.WriteString("# Manage skip suffixes with: ctsnare skip add/remove/list/reset\n")
	buf.WriteString("#\n")
	buf.WriteString("# The old 'skip_suffixes' key is deprecated and ignored.\n")
	buf.WriteString("# Use [skip_overrides] instead.\n\n")

	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(rawConfig); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	// Atomic write: write to temp file in same directory, then rename.
	tmpFile, err := os.CreateTemp(dir, "config-*.toml.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(buf.Bytes()); err != nil {
		tmpFile.Close()    //nolint:errcheck
		os.Remove(tmpPath) //nolint:errcheck
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath) //nolint:errcheck
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) //nolint:errcheck
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
