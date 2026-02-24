// Package config handles configuration loading and defaults for ctsnare.
package config

import (
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

// Config holds all configurable values for ctsnare.
// All fields have sensible defaults — use DefaultConfig to get a ready-to-use Config
// without a config file. Fields from the TOML file override defaults; CLI flags
// override both.
type Config struct {
	// CTLogs is the list of Certificate Transparency logs to poll.
	// Defaults to Google Argon 2025h1, Argon 2025h2, and Xenon 2025h1.
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

	// SkipSuffixes is the list of domain suffixes to exclude from scoring.
	// Domains matching any suffix are skipped before keyword matching, preventing
	// infrastructure platforms (CDNs, cloud hosts) from flooding results.
	SkipSuffixes []string `toml:"skip_suffixes"`

	// Backtrack is the number of CT log entries behind the current tip to start at.
	// When > 0, the poller begins at (tree_size - Backtrack), giving immediate
	// results on launch. Default: 0 (start at the tip, wait for new entries).
	Backtrack int64 `toml:"backtrack"`
}

// DefaultConfig returns a Config with sensible production defaults.
// The returned config is ready to use without a config file.
func DefaultConfig() *Config {
	return &Config{
		CTLogs: []CTLogConfig{
			{
				URL:  "https://ct.googleapis.com/logs/us1/argon2025h1",
				Name: "Google Argon 2025h1",
			},
			{
				URL:  "https://ct.googleapis.com/logs/us1/argon2025h2",
				Name: "Google Argon 2025h2",
			},
			{
				URL:  "https://ct.googleapis.com/logs/eu1/xenon2025h1",
				Name: "Google Xenon 2025h1",
			},
		},
		DefaultProfile: "all",
		BatchSize:      256,
		PollInterval:   5 * time.Second,
		DBPath:         defaultDBPath(),
		CustomProfiles: make(map[string]domain.Profile),
		SkipSuffixes:   defaultSkipSuffixes(),
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
func MergeFlags(cfg *Config, dbPath string, batchSize int, pollInterval time.Duration, backtrack int64) {
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
	if len(cfg.SkipSuffixes) == 0 {
		cfg.SkipSuffixes = defaultSkipSuffixes()
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

// defaultSkipSuffixes returns the default list of domain suffixes to skip
// during scoring. These are infrastructure domains that generate noise.
func defaultSkipSuffixes() []string {
	return []string{
		"cloudflaressl.com",
		"amazonaws.com",
		"herokuapp.com",
		"azurewebsites.net",
		"googleusercontent.com",
		"fastly.net",
		"akamaiedge.net",
		"cloudfront.net",
		"github.io",
		"gitlab.io",
		"netlify.app",
		"vercel.app",
		"firebaseapp.com",
		"appspot.com",
		"trafficmanager.net",
		"azure-api.net",
	}
}
